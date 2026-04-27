// Single-tier baseline simulators for E2. Each baseline mimics what a
// naive operator would observe if they retrieved a 1-of-1 replica from a
// single backend — no quorum, no Merkle verify, no erasure. The bench
// records pure latency-stack samples; no bytes are actually moved (the
// bench is comparing latency distributions, not throughput).
//
// Calibration rationale per backend (CLAUDE.md §3.1 row "Baselines"):
//
//   pinata_only — P50 80ms / P99 400ms. Identical to LBVR's hot tier
//     calibration (sim_tier.go); a "Pinata-alone" baseline is exactly
//     what a single-tier hot read looks like minus the gateway's quorum
//     wait. Source: Pinata published SLA range + Trautwein INFOCOM 2024
//     "IPFS in the Fast Lane" measurement of dedicated-gateway P95.
//
//   s3 — P50 60ms / P99 300ms. AWS S3 region-local typical observed
//     latencies (well-published in AWS performance docs and customer
//     case studies; e.g. AWS S3 Standard GET latency is "tens of ms"
//     for objects <128KB). Sub-Pinata because S3's edge presence is
//     denser than any IPFS pinning service.
//
//   storj — P50 200ms / P99 1000ms. Storj's distributed-storage P50 is
//     reported in their "Hotrodding Decentralized Storage" technical
//     report and confirmed by their public dashboard at
//     status.storj.io. The P99 captures the multi-node assembly tail —
//     Storj reads must reconstitute from 29 of 80 erasure pieces, so
//     the slowest piece dominates.
//
//   ipfsio — P50 2000ms / P99 10000ms. Trautwein et al. INFOCOM 2024
//     directly measure the public IPFS gateway and report P95 between
//     2 and 10 seconds; we calibrate P99 to the upper bound of that
//     window. This baseline is the canonical "naive decentralized
//     storage" stalking horse.
//
// All four use lognormal distributions with sigma chosen so that
// P99 = exp(mu + 2.326 * sigma); see eval/scripts/e2_latency_cdf.py
// for the inverse mapping. Mu values are ln(P50_ns).

package main

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"
)

// baseline is the uniform interface every single-tier simulator
// implements. Fetch samples a latency, sleeps for it, returns. ctx
// cancellation interrupts the wait and surfaces ctx.Err().
type baseline interface {
	Name() string
	Fetch(ctx context.Context) (latency time.Duration, err error)
}

// baselineSpec is the env.json record for one baseline curve.
type baselineSpec struct {
	Name        string  `json:"name"`
	Family      string  `json:"family"`
	Mu          float64 `json:"mu"`
	Sigma       float64 `json:"sigma"`
	P50ms       int     `json:"p50_ms"`
	P99ms       int     `json:"p99_ms"`
	Calibration string  `json:"calibration"`
}

// Calibration constants — derived from (P50_ms, P99_ms) via
// mu = ln(P50_ns), sigma = (ln(P99_ns) - mu) / 2.326. Hardcoded so the
// bench is deterministic and `go vet`-clean (no init math).

// pinata_only — 80ms / 400ms (matches sim_tier.go hot tier).
const (
	pinataOnlyMu    = 18.1974681
	pinataOnlySigma = 0.6919175
	pinataOnlyP50ms = 80
	pinataOnlyP99ms = 400
)

// s3 — 60ms / 300ms.
//
// mu = ln(60e6)   = 17.9098552
// sigma = (ln(300e6) - mu) / 2.326
//
//	= (19.5192931 - 17.9098552) / 2.326
//	= 0.6919165
const (
	s3Mu    = 17.9098552
	s3Sigma = 0.6919165
	s3P50ms = 60
	s3P99ms = 300
)

// storj — 200ms / 1000ms.
//
// mu = ln(2e8)     = 19.1138280
// sigma = (ln(1e9) - mu) / 2.326
//
//	= (20.7232658 - 19.1138280) / 2.326
//	= 0.6919165
const (
	storjMu    = 19.1138280
	storjSigma = 0.6919165
	storjP50ms = 200
	storjP99ms = 1000
)

// ipfsio — 2000ms / 10000ms (Trautwein INFOCOM 2024 public-gateway).
//
// mu = ln(2e9)     = 21.4164129
// sigma = (ln(1e10) - mu) / 2.326
//
//	= (23.0258509 - 21.4164129) / 2.326
//	= 0.6919165
const (
	ipfsioMu    = 21.4164129
	ipfsioSigma = 0.6919165
	ipfsioP50ms = 2000
	ipfsioP99ms = 10000
)

// baselineModes is the canonical baseline ordering used in the JSON
// schema and on the figure legend. Ordered cheapest → slowest at P50.
var baselineModes = []string{"s3", "pinata_only", "storj", "ipfsio"}

// allModes lists every per-bundle measurement key in the run record:
// LBVR fast path first, then the four single-tier baselines.
var allModes = append([]string{"lbvr"}, baselineModes...)

// lognormalBaseline is the shared implementation behind every per-tier
// baseline. Constructed once per run; safe for concurrent Fetch via the
// rngMu lock (rand.Rand is not safe for concurrent use).
type lognormalBaseline struct {
	name  string
	mu    float64
	sigma float64

	rngMu sync.Mutex
	rng   *rand.Rand
}

// newLognormalBaseline wires a baseline from its calibrated parameters.
// seed must be unique per baseline so streams stay independent across
// the four curves.
func newLognormalBaseline(name string, mu, sigma float64, seed int64) *lognormalBaseline {
	return &lognormalBaseline{
		name:  name,
		mu:    mu,
		sigma: sigma,
		rng:   rand.New(rand.NewSource(seed)),
	}
}

// Name reports the baseline's display name (used as a JSON key and as
// the figure-legend label).
func (b *lognormalBaseline) Name() string { return b.name }

// Fetch samples a lognormal latency and sleeps for it. The returned
// latency is exactly the sleep duration unless ctx cancels first, in
// which case both the elapsed time AND ctx.Err() are returned (mirrors
// the simTier.Get contract so the driver treats success and cancel
// uniformly).
func (b *lognormalBaseline) Fetch(ctx context.Context) (time.Duration, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	wait := b.sample()
	start := time.Now()
	timer := time.NewTimer(wait)
	select {
	case <-timer.C:
		return time.Since(start), nil
	case <-ctx.Done():
		timer.Stop()
		return time.Since(start), ctx.Err()
	}
}

// sample draws one lognormal sample. Strictly positive by construction
// (rand.NormFloat64 → exp(mu+sigma*z)).
func (b *lognormalBaseline) sample() time.Duration {
	b.rngMu.Lock()
	z := b.rng.NormFloat64()
	b.rngMu.Unlock()
	ns := math.Exp(b.mu + b.sigma*z)
	if ns < 1 {
		ns = 1
	}
	if ns > float64(math.MaxInt64) {
		ns = float64(math.MaxInt64)
	}
	return time.Duration(ns)
}

// newBaselines builds the four baseline simulators with disjoint seeds.
// Order matches baselineModes so callers can iterate by index. Seeds
// (10..13) are offset from any tier-sim seeds (1..3) the driver also
// uses, keeping all six RNG streams independent.
func newBaselines(seed int64) []baseline {
	return []baseline{
		newLognormalBaseline("s3", s3Mu, s3Sigma, seed+10),
		newLognormalBaseline("pinata_only", pinataOnlyMu, pinataOnlySigma, seed+11),
		newLognormalBaseline("storj", storjMu, storjSigma, seed+12),
		newLognormalBaseline("ipfsio", ipfsioMu, ipfsioSigma, seed+13),
	}
}

// baselineSpecs returns the env.json calibration block for every
// baseline. Keyed by name so the post-processor can join cleanly.
func baselineSpecs() map[string]baselineSpec {
	return map[string]baselineSpec{
		"pinata_only": {
			Name:        "pinata_only",
			Family:      "lognormal",
			Mu:          pinataOnlyMu,
			Sigma:       pinataOnlySigma,
			P50ms:       pinataOnlyP50ms,
			P99ms:       pinataOnlyP99ms,
			Calibration: "single-tier Pinata replica; matches LBVR hot-tier calibration; Trautwein INFOCOM 2024 + Pinata SLA",
		},
		"s3": {
			Name:        "s3",
			Family:      "lognormal",
			Mu:          s3Mu,
			Sigma:       s3Sigma,
			P50ms:       s3P50ms,
			P99ms:       s3P99ms,
			Calibration: "AWS S3 Standard GET, region-local; AWS performance docs",
		},
		"storj": {
			Name:        "storj",
			Family:      "lognormal",
			Mu:          storjMu,
			Sigma:       storjSigma,
			P50ms:       storjP50ms,
			P99ms:       storjP99ms,
			Calibration: "Storj distributed-storage; status.storj.io public dashboard + Storj 'Hotrodding' technical report",
		},
		"ipfsio": {
			Name:        "ipfsio",
			Family:      "lognormal",
			Mu:          ipfsioMu,
			Sigma:       ipfsioSigma,
			P50ms:       ipfsioP50ms,
			P99ms:       ipfsioP99ms,
			Calibration: "public ipfs.io gateway; Trautwein INFOCOM 2024 'IPFS in the Fast Lane' P95 = 2-10s",
		},
	}
}
