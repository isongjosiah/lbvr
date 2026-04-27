// In-process tiers.Client stand-in for the E3 RTT × loss matrix bench.
//
// Adapted from cmd/bench/e9/sim_tier.go with two added knobs:
//
//   extraRTT  — fixed Duration added on every Get before the lognormal
//               sample. Models the wide-area round-trip beyond the
//               tier's own service-time distribution.
//   lossRate  — probability in [0, 1] that a Get returns
//               errors.New("simTier: WAN drop") AFTER waiting extraRTT
//               (no lognormal added on dropped requests; the timeout is
//               the sentinel cost). Mirrors Toxiproxy's `limit_data` +
//               `latency` toxics conceptually.
//
// The base lognormal calibration (per-tier service time) is identical to
// E9: hot 80/400, warm 120/600, cold 500/8000. Drop()/Reset() semantics
// from E9 are preserved but unused by E3 — WAN conditions degrade everyone
// uniformly via extraRTT + lossRate, not via per-tier hard failures.
//
// IMPORTANT: every JSON artefact this bench produces names the
// distribution family + parameters (and the WAN cell) in env.json so the
// resulting heatmap is not misread as a live-tier measurement.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

// Per-tier lognormal calibration — verbatim from cmd/bench/e9/sim_tier.go
// so the two benches share a single source of truth for the underlying
// service-time distribution. Conversion: mu = ln(P50_ns); sigma =
// (ln(P99_ns) - mu) / 2.326. Hardcoded so the bench is deterministic.

// Hot tier — 80ms / 400ms.
const (
	hotMu    = 18.1974681
	hotSigma = 0.6919175
	hotP50ms = 80
	hotP99ms = 400
)

// Warm tier — 120ms / 600ms.
const (
	warmMu    = 18.6030495
	warmSigma = 0.6919175
	warmP50ms = 120
	warmP99ms = 600
)

// Cold tier — 500ms / 8000ms.
const (
	coldMu    = 20.0300718
	coldSigma = 1.1919608
	coldP50ms = 500
	coldP99ms = 8000
)

// distSpec is the env.json record describing one tier's latency model.
type distSpec struct {
	Family      string  `json:"family"`
	Mu          float64 `json:"mu"`
	Sigma       float64 `json:"sigma"`
	P50ms       int     `json:"p50_ms"`
	P99ms       int     `json:"p99_ms"`
	Calibration string  `json:"calibration"`
}

// simTier is a tiers.Client backed by an in-memory map and a calibrated
// lognormal latency model, plus per-cell extraRTT and lossRate.
// Independent rand.Source per tier so failure injection is reproducible.
type simTier struct {
	storeMu sync.RWMutex
	store   map[string][]byte

	name      string
	tierClass uint8
	muDist    float64
	sigma     float64

	// rngMu guards rng — rand.Rand is not safe for concurrent use, and
	// the gateway fans out simultaneous Get calls (one per tier shard).
	// The same mutex protects the loss-coin draw.
	rngMu sync.Mutex
	rng   *rand.Rand

	// extraRTT is added on every Get before the lognormal wait. Mirrors
	// Toxiproxy's `latency` toxic.
	extraRTT time.Duration
	// lossRate ∈ [0, 1] is the per-Get probability of returning
	// "WAN drop" after the extraRTT wait. Mirrors Toxiproxy's
	// `limit_data` toxic at the level of forced request failures.
	lossRate float64

	// dropped, when set, makes Get return an error after the latency
	// wait. Inherited from e9; unused by E3 but kept so the simTier
	// type can be reused if a future cell needs hard-down behaviour.
	dropped atomic.Bool

	// cancelObserved counts Get invocations that returned ctx.Err()
	// before their latency expired. Useful for diagnostics.
	cancelObserved atomic.Int32
}

// newSimTier wires the tier with its class-specific lognormal params.
// extraRTT and lossRate default to zero (the (10ms-baseline) cell uses
// SetWAN to apply 10ms / 0% explicitly).
func newSimTier(name string, class uint8, seed int64) *simTier {
	var mu, sig float64
	switch class {
	case tiers.TierHot:
		mu, sig = hotMu, hotSigma
	case tiers.TierWarm:
		mu, sig = warmMu, warmSigma
	case tiers.TierCold:
		mu, sig = coldMu, coldSigma
	default:
		mu, sig = warmMu, warmSigma
	}
	return &simTier{
		store:     make(map[string][]byte),
		name:      name,
		tierClass: class,
		muDist:    mu,
		sigma:     sig,
		rng:       rand.New(rand.NewSource(seed)),
	}
}

// Name reports the tier's display name.
func (s *simTier) Name() string { return s.name }

// TierClass reports the on-chain enum class (Hot/Warm/Cold).
func (s *simTier) TierClass() uint8 { return s.tierClass }

// SetWAN configures the per-cell impairment knobs. Safe to call between
// matrix cells but not concurrent with in-flight Gets — the bench harness
// quiesces between cells by construction.
func (s *simTier) SetWAN(extraRTT time.Duration, lossRate float64) {
	if lossRate < 0 {
		lossRate = 0
	}
	if lossRate > 1 {
		lossRate = 1
	}
	s.extraRTT = extraRTT
	s.lossRate = lossRate
}

// Put stores synchronously. We're measuring Get latency, so injecting
// latency on Put would inflate the bench setup phase without changing
// the figure under measurement.
func (s *simTier) Put(ctx context.Context, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", errors.New("simTier: empty data")
	}
	h := sha256.Sum256(data)
	cid := s.name + "-" + hex.EncodeToString(h[:])
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	s.store[cid] = cp
	return cid, nil
}

// Get blocks for extraRTT + lognormal sample, then either errors (if
// either Drop() or a loss-coin came up) or returns the stored bytes.
//
// Loss-coin semantics: after waiting extraRTT only, the sim flips a
// per-tier RNG-backed coin against lossRate. If it lands "drop", we
// return a sentinel error WITHOUT adding the lognormal sample — i.e.
// the WAN drop cuts the request short at the round-trip boundary, which
// is what Toxiproxy's `limit_data` simulates. If the coin lands "keep",
// we proceed with the lognormal wait and return the stored bytes (or
// dropped/not-found errors as in E9).
//
// Cancellation during ANY of the waits (extraRTT or lognormal) returns
// ctx.Err() immediately — no shard data, no fake error, matching the
// gateway's expectation.
func (s *simTier) Get(ctx context.Context, cid string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// 1. extraRTT wait — interruptible by ctx cancel.
	if s.extraRTT > 0 {
		t := time.NewTimer(s.extraRTT)
		select {
		case <-t.C:
			// proceed
		case <-ctx.Done():
			t.Stop()
			s.cancelObserved.Add(1)
			return nil, ctx.Err()
		}
	}

	// 2. Loss-coin draw. Atomic with the rng mutex so we never
	// interleave the coin with the lognormal sample.
	if s.lossRate > 0 {
		s.rngMu.Lock()
		coin := s.rng.Float64()
		s.rngMu.Unlock()
		if coin < s.lossRate {
			return nil, errors.New("simTier: WAN drop")
		}
	}

	// 3. Lognormal service-time wait — also interruptible.
	wait := s.sampleLatency()
	timer := time.NewTimer(wait)
	select {
	case <-timer.C:
		// proceed
	case <-ctx.Done():
		timer.Stop()
		s.cancelObserved.Add(1)
		return nil, ctx.Err()
	}

	if s.dropped.Load() {
		return nil, errors.New("simTier: dropped")
	}

	s.storeMu.RLock()
	defer s.storeMu.RUnlock()
	v, ok := s.store[cid]
	if !ok {
		return nil, errors.New("simTier: not found")
	}
	out := make([]byte, len(v))
	copy(out, v)
	return out, nil
}

// Stat returns minimal metadata. The bench never calls it, but the
// tiers.Client interface requires it.
func (s *simTier) Stat(ctx context.Context, cid string) (*tiers.Stat, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.storeMu.RLock()
	defer s.storeMu.RUnlock()
	v, ok := s.store[cid]
	if !ok {
		return nil, errors.New("simTier: not found")
	}
	return &tiers.Stat{CID: cid, SizeBytes: int64(len(v)), StoredAt: time.Now()}, nil
}

// Delete removes a CID. The bench never calls it, but the tiers.Client
// interface requires it.
func (s *simTier) Delete(ctx context.Context, cid string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	delete(s.store, cid)
	return nil
}

// Drop / Reset preserved for parity with e9 simTier. Unused by E3.
func (s *simTier) Drop()              { s.dropped.Store(true) }
func (s *simTier) Reset()             { s.dropped.Store(false) }
func (s *simTier) WasCancelled() bool { return s.cancelObserved.Load() > 0 }

// sampleLatency draws one lognormal sample and converts to a Duration.
func (s *simTier) sampleLatency() time.Duration {
	s.rngMu.Lock()
	z := s.rng.NormFloat64()
	s.rngMu.Unlock()
	ns := math.Exp(s.muDist + s.sigma*z)
	if ns < 1 {
		ns = 1
	}
	if ns > float64(math.MaxInt64) {
		ns = float64(math.MaxInt64)
	}
	return time.Duration(ns)
}

// distSpecFor returns the env.json record for a tier class.
func distSpecFor(class uint8) distSpec {
	switch class {
	case tiers.TierHot:
		return distSpec{
			Family: "lognormal", Mu: hotMu, Sigma: hotSigma,
			P50ms: hotP50ms, P99ms: hotP99ms,
			Calibration: "Pinata dedicated-gateway stand-in; Trautwein INFOCOM 2024 + Pinata SLA range",
		}
	case tiers.TierWarm:
		return distSpec{
			Family: "lognormal", Mu: warmMu, Sigma: warmSigma,
			P50ms: warmP50ms, P99ms: warmP99ms,
			Calibration: "Filebase S3 stand-in; AWS region-local typical",
		}
	case tiers.TierCold:
		return distSpec{
			Family: "lognormal", Mu: coldMu, Sigma: coldSigma,
			P50ms: coldP50ms, P99ms: coldP99ms,
			Calibration: "Irys/Arweave stand-in; chain-settled reads spike into the tail",
		}
	}
	return distSpec{}
}
