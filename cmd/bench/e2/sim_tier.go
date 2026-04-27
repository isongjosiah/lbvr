// In-process tiers.Client stand-in for the E2 retrieval-latency-CDF bench.
// Calibration is BIT-IDENTICAL to cmd/bench/e9/sim_tier.go (CLAUDE.md §4.5);
// see that file for the literature pointers behind each (mu, sigma). E2
// has no failure injection — Drop()/Reset() exist so the type satisfies
// the same shape as e9's simTier and so the gateway-fast-path sanity
// assertion ("≥99% RecoveryFastPath when no failures injected") in
// cmd/bench/e2/main.go has the same machinery available; this experiment
// never calls Drop. Failure modes belong to E6/E9.

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

// Per-tier lognormal calibration — copied verbatim from
// cmd/bench/e9/sim_tier.go to keep the LBVR fast-path curve in this
// figure directly comparable with E9's "baseline" CDF. Conversion:
// mu = ln(P50_ns); sigma = (ln(P99_ns) - mu) / 2.326. Hardcoded so the
// bench is deterministic and `go vet`-clean (no init math).

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
// Mirrors CLAUDE.md §8 environment-fingerprint discipline: figures must
// not be mistaken for live measurements.
type distSpec struct {
	Family      string  `json:"family"`
	Mu          float64 `json:"mu"`
	Sigma       float64 `json:"sigma"`
	P50ms       int     `json:"p50_ms"`
	P99ms       int     `json:"p99_ms"`
	Calibration string  `json:"calibration"`
}

// simTier is a tiers.Client backed by an in-memory map and a calibrated
// lognormal latency model. Independent rand.Source per tier so streams
// stay reproducible (seed = top-level seed + tier idx).
type simTier struct {
	storeMu sync.RWMutex
	store   map[string][]byte

	name      string
	tierClass uint8
	muDist    float64
	sigma     float64

	// rngMu guards rng — rand.Rand is not safe for concurrent use, and
	// the bench fans out simultaneous Get calls (one per tier shard).
	rngMu sync.Mutex
	rng   *rand.Rand

	// dropped exists for interface parity with cmd/bench/e9/simTier;
	// E2 never sets it. Kept so the gateway recovery code paths see an
	// identical client shape to E9's, which simplifies cross-experiment
	// debugging if the LBVR P50 ever drifts between the two figures.
	dropped atomic.Bool

	// cancelObserved counts Get invocations that returned ctx.Err()
	// before their latency expired. Used in tests if needed.
	cancelObserved atomic.Int32
}

// newSimTier wires the tier with its class-specific lognormal params.
// seed must be unique per tier to keep the stream independent.
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

// Put stores synchronously. We measure Get latency, so injecting latency
// on Put would inflate the bench setup phase without changing the figure
// under measurement.
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

// Get blocks for a lognormal-distributed duration, then either errors
// (if Drop()) or returns the stored bytes. Cancellation during the wait
// returns ctx.Err() immediately — no shard data, no fake error,
// matching the gateway's expectation.
func (s *simTier) Get(ctx context.Context, cid string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

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

// Drop / Reset / WasCancelled — interface-parity stubs with e9.simTier;
// E2 does not invoke Drop().
func (s *simTier) Drop()              { s.dropped.Store(true) }
func (s *simTier) Reset()             { s.dropped.Store(false) }
func (s *simTier) WasCancelled() bool { return s.cancelObserved.Load() > 0 }

// sampleLatency draws one lognormal sample and converts to a Duration.
// rand.NormFloat64 → exp(mu+sigma*z); strictly positive by construction.
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
