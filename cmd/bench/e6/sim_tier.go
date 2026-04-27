// Byzantine-aware in-process tiers.Client stand-in for the E6 / E6b
// withstand benches. Adapted from cmd/bench/e9/sim_tier.go: same calibrated
// lognormal latency model (CLAUDE.md §4.5; Trautwein INFOCOM 2024 + Pinata
// SLA range), but with two adversarial behaviours layered on top:
//
//	byzantineUniform        - corrupt bytes returned on every Get.
//	byzantineTierSelective  - corrupt bytes returned UNLESS the calling ctx
//	                          carries lbvrCallPurposeKey == "por_challenge",
//	                          in which case honest bytes are served. This
//	                          mirrors §5's metadata-correlated adversary that
//	                          passes PoR cadences but degrades real
//	                          retrievals.
//
// The corruption preserves length so the recovery state machine in
// internal/gateway/recovery.go cannot detect it directly — the breach is
// caught downstream by AES-GCM auth (decrypt) or by the Merkle re-verify
// in cmd/gateway/handler.go. We mirror that contract here: the bench's
// post-Recover verifier checks the padded-bytes SHA-256 against the honest
// reference recorded at Put time, which is exactly what the gateway's
// Merkle path would observe (the gateway hashes the decrypted plaintext;
// here we hash the encrypted payload because the bench skips the decrypt
// step — both are equivalent integrity gates against byte corruption).
//
// Latency injection STAYS calibrated lognormal: byzantine != slow. The
// threat model in §5 is correctness, not availability — a slow-down test
// is what E3 / Toxiproxy covers.

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

// byzantineMode toggles the corruption strategy. Modes are mutually
// exclusive; per-tier byzantine flags then opt individual tiers into the
// chosen mode (an honest tier ignores the mode entirely).
type byzantineMode uint8

const (
	byzantineNone          byzantineMode = iota
	byzantineUniform                     // every Get on a byzantine tier returns corrupt bytes.
	byzantineTierSelective               // honest bytes when ctx is "por_challenge"; corrupt otherwise.
)

// callPurposeKey is the context.Value key the bench uses to tag a call as
// either a PoR challenge or a real retrieval. Defining a private type
// avoids accidental collision with anything the gateway might add later.
type callPurposeKey struct{}

const (
	callPurposePoRChallenge = "por_challenge"
	callPurposeRetrieval    = "retrieval"
)

// withCallPurpose attaches the call-purpose tag to a context. The bench
// uses this to differentiate "I'm asking honestly to confirm storage"
// (PoR challenge) from "I'm a clinician asking for the real bytes"
// (retrieval). A tier-selective adversary serves the former honestly.
func withCallPurpose(ctx context.Context, purpose string) context.Context {
	return context.WithValue(ctx, callPurposeKey{}, purpose)
}

// callPurposeFrom extracts the purpose tag, defaulting to "retrieval"
// (the most adversarially-stressed path) if no tag is present. Defaulting
// to retrieval — not PoR — means a forgetful caller still triggers
// byzantine behaviour, which is the safer default for testing.
func callPurposeFrom(ctx context.Context) string {
	v := ctx.Value(callPurposeKey{})
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return callPurposeRetrieval
}

// Per-tier lognormal calibration — mirrors cmd/bench/e9/sim_tier.go so
// the two benches share a comparable latency baseline.
const (
	hotMu    = 18.1974681
	hotSigma = 0.6919175
	hotP50ms = 80
	hotP99ms = 400

	warmMu    = 18.6030495
	warmSigma = 0.6919175
	warmP50ms = 120
	warmP99ms = 600

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
// lognormal latency model. Per-tier RNG so failure injection is
// reproducible (seed = top-level seed + tier idx). Adds a byzantine bool
// + shared-mode pointer to the e9 baseline.
type simTier struct {
	storeMu sync.RWMutex
	store   map[string][]byte

	name      string
	tierClass uint8
	muDist    float64
	sigma     float64

	rngMu sync.Mutex
	rng   *rand.Rand

	// byzantine, when true, makes this tier serve corrupt bytes per the
	// run's byzantineMode setting. Atomic so the bench can flip
	// individual tiers between bundles without taking the storeMu.
	byzantine atomic.Bool

	// mode is shared across all simTiers in a single bench run — it
	// names which adversarial behaviour applies when byzantine=true.
	mode byzantineMode
}

// newSimTier wires the tier with its class-specific lognormal params.
// seed must be unique per tier to keep the stream independent.
func newSimTier(name string, class uint8, seed int64, mode byzantineMode) *simTier {
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
		mode:      mode,
	}
}

// Name reports the tier's display name.
func (s *simTier) Name() string { return s.name }

// TierClass reports the on-chain enum class (Hot/Warm/Cold).
func (s *simTier) TierClass() uint8 { return s.tierClass }

// SetByzantine flips this tier into adversarial mode (true) or back to
// honest (false). The active mode (uniform vs tier-selective) is fixed
// at construction; this just toggles whether THIS tier participates.
func (s *simTier) SetByzantine(b bool) { s.byzantine.Store(b) }

// IsByzantine reports the current adversarial flag.
func (s *simTier) IsByzantine() bool { return s.byzantine.Load() }

// Put stores synchronously. Latency injection on Put would inflate the
// bench setup phase without changing the figure under measurement.
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

// Get blocks for a lognormal-distributed duration, then either returns
// honest bytes or corrupted bytes per the byzantine flag + mode.
//
// Corruption preserves length (XOR of every byte with 0xFF) so the
// recovery state machine treats this as a successful fetch — the breach
// is caught downstream by Merkle re-verify, mirroring the production
// gateway's contract.
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
		return nil, ctx.Err()
	}

	s.storeMu.RLock()
	v, ok := s.store[cid]
	s.storeMu.RUnlock()
	if !ok {
		return nil, errors.New("simTier: not found")
	}

	out := make([]byte, len(v))
	copy(out, v)

	if s.byzantine.Load() && s.shouldCorrupt(ctx) {
		// Length-preserving corruption: XOR every byte with 0xFF.
		// Pattern is unimportant — the only requirement is that the
		// returned bytes != stored bytes. Stable corruption (vs random
		// per-call) keeps the bench deterministic given a fixed seed.
		for i := range out {
			out[i] ^= 0xFF
		}
	}
	return out, nil
}

// shouldCorrupt encodes the adversary's decision rule.
//
//	byzantineUniform        - always corrupt.
//	byzantineTierSelective  - honest if and only if the ctx purpose is
//	                          "por_challenge"; corrupt otherwise. This is
//	                          the §5 metadata-correlated adversary: it
//	                          looks clean during sampled PoR but degrades
//	                          real reads.
//	byzantineNone           - never corrupt (defensive; SetByzantine
//	                          should never be true under None).
func (s *simTier) shouldCorrupt(ctx context.Context) bool {
	switch s.mode {
	case byzantineUniform:
		return true
	case byzantineTierSelective:
		return callPurposeFrom(ctx) != callPurposePoRChallenge
	default:
		return false
	}
}

// Stat returns minimal metadata. The bench never calls it but the
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

// Delete removes a CID. The bench never calls it but the tiers.Client
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
