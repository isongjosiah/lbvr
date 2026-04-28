// In-process tiers.Client stand-in for the E1 ingest-throughput bench.
// Copied verbatim from cmd/bench/e9 — same calibration so the §V
// narrative can talk about ingest and recovery latency in the same
// breath without explaining a second sim distribution.
//
// Real Pinata/Filebase/Irys integration waits on .env keys (see
// docs/eval-protocol.md §3 cold-tier decision); this sim is calibrated
// from CLAUDE.md §4.5 expected ranges and Trautwein INFOCOM 2024.
//
// IMPORTANT: every JSON artefact this bench produces names the
// distribution family + parameters in env.json so the resulting
// throughput number is not misread as a live-tier measurement.

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

// Per-tier lognormal calibration (CLAUDE.md §4.5; identical to E9).
//
// hotDistP50  =  80ms / hotDistP99  =  400ms — Pinata dedicated-gateway stand-in.
// warmDistP50 = 120ms / warmDistP99 =  600ms — Filebase S3 stand-in.
// coldDistP50 = 500ms / coldDistP99 = 8000ms — Irys/Arweave stand-in.

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

// simTier is a tiers.Client with a calibrated lognormal Put-latency
// model. Independent rand.Source per tier so failure injection is
// reproducible (seed = top-level seed + tier idx).
//
// E1 measures ingest throughput. The latency budget is applied to Put
// (not Get as in E9). The payload is NOT retained — see Put's docstring.
type simTier struct {
	name      string
	tierClass uint8
	muDist    float64
	sigma     float64

	// rngMu guards rng — rand.Rand is not safe for concurrent use, and
	// the Ingester fans out simultaneous Put calls (one per tier shard).
	rngMu sync.Mutex
	rng   *rand.Rand

	// dropped, when set, makes Put/Get return an error after the latency
	// wait. Not used by E1 itself but kept for parity with the e9 sim.
	dropped atomic.Bool

	// cancelObserved counts Get/Put invocations that returned ctx.Err()
	// before their latency expired.
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

// Put blocks for a lognormal-distributed duration, then records the CID
// (sha256 hash) WITHOUT retaining the payload. E1 inverts the e9 convention
// in two ways:
//
//  1. Latency is injected on Put (e9 injects on Get) — ingest throughput
//     is dominated by upload latency, not retrieval.
//  2. The payload is intentionally NOT retained. At the 10K scale we
//     would otherwise pin ~52 GiB of shard bytes across the three sim
//     tiers (10K bundles × 3 shards × 1.5× RS(2,3) overhead × ~3.5 MB
//     bundle median), which OOMs the 16 GiB host. We need only the CID
//     for the registry write; nothing in the bench reads the payload
//     back. Get returns "not found" by design — see the docstring.
//
// Cancellation during the wait returns ctx.Err() immediately.
func (s *simTier) Put(ctx context.Context, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", errors.New("simTier: empty data")
	}

	wait := s.sampleLatency()
	timer := time.NewTimer(wait)
	select {
	case <-timer.C:
		// proceed
	case <-ctx.Done():
		timer.Stop()
		s.cancelObserved.Add(1)
		return "", ctx.Err()
	}

	if s.dropped.Load() {
		return "", errors.New("simTier: dropped")
	}

	// Hash for content-addressing parity with e9; no retention.
	h := sha256.Sum256(data)
	return s.name + "-" + hex.EncodeToString(h[:]), nil
}

// Get always returns "not retained": E1 never calls Get and Put discards
// the payload to keep the 10K scale within the host's RAM budget. The
// tiers.Client interface still requires the method; if a future caller
// wires Get into E1, the loud error names the constraint.
func (s *simTier) Get(ctx context.Context, cid string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, errors.New("simTier: payload not retained (E1 measures Put-latency only)")
}

// Stat — same retention story as Get. Reports zero size.
func (s *simTier) Stat(ctx context.Context, cid string) (*tiers.Stat, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &tiers.Stat{CID: cid, SizeBytes: 0, StoredAt: time.Now()}, nil
}

// Delete is a no-op; nothing is retained to delete.
func (s *simTier) Delete(ctx context.Context, cid string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

// Drop / Reset / WasCancelled — kept for parity with cmd/bench/e9 even
// though E1 itself never injects failures (no failure modes in the §8 row
// for ingest throughput).
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
			Calibration: "Irys/Arweave stand-in; chain-settled writes spike into the tail",
		}
	}
	return distSpec{}
}
