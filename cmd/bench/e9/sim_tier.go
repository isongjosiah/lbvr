// In-process tiers.Client stand-in for the D9 erasure-recovery bench.
// Real Pinata/Filebase/Irys integration waits on .env keys (see
// docs/eval-protocol.md §3 cold-tier decision); this sim is calibrated
// from CLAUDE.md §4.5 expected ranges and Trautwein INFOCOM 2024.
//
// IMPORTANT: every JSON artefact this bench produces names the
// distribution family + parameters in env.json so the resulting CDF is
// not misread as a live-tier measurement. Toxiproxy + real tiers comes
// after the .env round per docs/eval-protocol.md §3.

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

// Per-tier lognormal calibration (CLAUDE.md §4.5).
//
// hotDistP50  =  80ms / hotDistP99  =  400ms — Pinata dedicated-gateway stand-in.
//   Calibration source: Trautwein INFOCOM 2024 (public-IPFS P95 = 2-10s);
//   dedicated gateways are 10-25x faster per Pinata's published SLA range.
//   Conservative — real measurement will refine post-foundryup.
// warmDistP50 = 120ms / warmDistP99 =  600ms — Filebase S3 stand-in.
//   Calibration: S3-class latency (AWS public docs typical region-local).
// coldDistP50 = 500ms / coldDistP99 = 8000ms — Irys/Arweave stand-in.
//   Calibration: Arweave gateway reads commonly seconds; chain-settled
//   reads can spike to 30s, captured in the lognormal tail.
//
// Conversion: mu = ln(P50_ns); sigma = (ln(P99_ns) - mu) / 2.326. Values
// hardcoded so the bench is deterministic and `go vet`-clean (no init
// math).

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
// lognormal latency model. Independent rand.Source per tier so failure
// injection is reproducible (seed = top-level seed + tier idx).
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

	// dropped, when set, makes Get return an error after the latency
	// wait. Mirrors a hard-down tier without violating the
	// "every fetcher emits one shardResult" contract in
	// internal/gateway/recovery.go.
	dropped atomic.Bool

	// cancelObserved counts Get invocations that returned ctx.Err()
	// before their latency expired. Used to assert cold-tier
	// cancellation in tests.
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

// Get blocks for a lognormal-distributed duration, then either errors
// (if Drop()) or returns the stored bytes. Cancellation during the
// wait returns ctx.Err() immediately — no shard data, no fake error,
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

// Drop flips the tier into hard-failure mode for subsequent Gets.
// Latency is still injected before the error so the gateway's parallel
// fetch sees a realistic failure timing (a real backend that drops
// connections still imposes TCP/TLS round-trips).
func (s *simTier) Drop() { s.dropped.Store(true) }

// Reset clears the dropped flag. Bench harness calls between modes.
func (s *simTier) Reset() { s.dropped.Store(false) }

// WasCancelled reports whether at least one Get observed ctx cancel
// during its latency wait. Mirrors cmd/gateway tests' wasCancelled().
func (s *simTier) WasCancelled() bool { return s.cancelObserved.Load() > 0 }

// sampleLatency draws one lognormal sample and converts to a Duration.
// rand.NormFloat64 → exp(mu+sigma*z); strictly positive by construction.
func (s *simTier) sampleLatency() time.Duration {
	s.rngMu.Lock()
	z := s.rng.NormFloat64()
	s.rngMu.Unlock()
	ns := math.Exp(s.muDist + s.sigma*z)
	if ns < 1 {
		// Sanity floor: a sub-nanosecond Get would suggest the timer
		// trick collapses; never happens with these mu/sigma but
		// defensive against future re-calibration.
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
