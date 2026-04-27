// Local mock anchor for the E-PROV bench.
//
// We deliberately do NOT depend on cmd/gateway/anchor.go: that one returns
// instantly (acceptable for unit tests, unrealistic for a wall-clock
// measurement that wants to model the on-chain anchor as one stage). For
// E-PROV we want a calibrated submit-receipt round-trip latency on
// Anchor() so the per-stage bar chart is honest about what the gateway
// would experience in production.
//
// Calibration (CLAUDE.md §4.6, docs/provenance-spec.md §6.2):
// Polygon zkEVM Cardona block time is ~2 s, but anchor latency for a
// small SSTORE-only tx without a finality wait is dominated by RPC
// submit + receipt round-trip — typically ~30 ms median, ~200 ms P99 on
// a healthy testnet. We use a lognormal distribution because RPC
// latency is heavy-tailed (occasional retries, mempool congestion).
//
//	mu_ns    = ln(30_000_000)
//	sigma    = (ln(200_000_000) - mu_ns) / 2.326   // 2.326 = z(P99)
//
// Hardcoded so the bench is deterministic and `go vet`-clean.

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	mrand "math/rand"
	"sync"
	"time"
)

// Anchor latency calibration. See file-level comment for rationale.
const (
	anchorMu    = 17.2166744 // ln(30e6 ns) — 30 ms median
	anchorSigma = 0.8157     // → P99 ≈ 200 ms
	anchorP50ms = 30
	anchorP99ms = 200
)

// AnchorRecord mirrors the gateway's analogous struct. We intentionally
// avoid importing cmd/gateway (different package; would also pull HTTP
// deps the bench has no need for).
type AnchorRecord struct {
	ProvHash    [32]byte
	BlockNumber uint64
	TxHash      string
}

// AnchorClient is the small interface the bench depends on.
type AnchorClient interface {
	Anchor(ctx context.Context, bundleID, retrievalID, provHash [32]byte) (AnchorRecord, error)
}

// mockAnchor is a calibrated-latency in-memory anchor. Concurrent-safe
// (the bench runs single-threaded but tests may exercise it under -race).
type mockAnchor struct {
	mu        sync.Mutex
	anchors   map[[64]byte]AnchorRecord
	nextBlock uint64

	// rng is bench-private so a -seed flag in main.go controls its
	// stream. rand.Rand is not safe for concurrent use; we serialise
	// via mu (the same mutex protecting the anchor map).
	rng *mrand.Rand
}

// newMockAnchor seeds the latency stream from seed; map starts empty;
// block numbers are 1-indexed so a verifier never reads "block 0" for a
// successfully-anchored doc.
func newMockAnchor(seed int64) *mockAnchor {
	return &mockAnchor{
		anchors:   make(map[[64]byte]AnchorRecord),
		nextBlock: 1,
		rng:       mrand.New(mrand.NewSource(seed)),
	}
}

// Anchor blocks for a lognormal-distributed duration, then writes the
// record. Re-anchoring the same (bundleID, retrievalID) errors — matches
// AuditorLog.AlreadyAnchored.
func (m *mockAnchor) Anchor(ctx context.Context, bundleID, retrievalID, provHash [32]byte) (AnchorRecord, error) {
	if provHash == ([32]byte{}) {
		return AnchorRecord{}, errors.New("anchor: provHash is zero")
	}

	wait := m.sampleLatency()
	timer := time.NewTimer(wait)
	select {
	case <-timer.C:
		// proceed
	case <-ctx.Done():
		timer.Stop()
		return AnchorRecord{}, ctx.Err()
	}

	key := concatID(bundleID, retrievalID)
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.anchors[key]; ok {
		return AnchorRecord{}, fmt.Errorf("anchor: already anchored at block %d", existing.BlockNumber)
	}

	rec := AnchorRecord{
		ProvHash:    provHash,
		BlockNumber: m.nextBlock,
		TxHash:      "0xmock" + hex.EncodeToString(randBytes(28)),
	}
	m.anchors[key] = rec
	m.nextBlock++
	return rec, nil
}

// sampleLatency draws one lognormal sample; strictly positive by
// construction. Sub-nanosecond floor mirrors sim_tier.go's defensive
// clamp.
func (m *mockAnchor) sampleLatency() time.Duration {
	m.mu.Lock()
	z := m.rng.NormFloat64()
	m.mu.Unlock()
	ns := math.Exp(anchorMu + anchorSigma*z)
	if ns < 1 {
		ns = 1
	}
	if ns > float64(math.MaxInt64) {
		ns = float64(math.MaxInt64)
	}
	return time.Duration(ns)
}

func concatID(bundleID, retrievalID [32]byte) (out [64]byte) {
	copy(out[:32], bundleID[:])
	copy(out[32:], retrievalID[:])
	return
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}
