package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	mrand "math/rand"
	"sort"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/merkle"
	"github.com/isongjosiah/lbvr-med/internal/por"
	"github.com/isongjosiah/lbvr-med/internal/provenance"
)

// proveStage measures Half A — Go-side prove time at one sweep point.
//
// Per rep we pick a random chunkIdx (uniform over [0, numChunks)) and
// time:
//
//	merkleNS  — tree.Proof(idx)              (deterministic per idx)
//	signNS    — provenance.Sign(priv, msg)   (BLS pairing)
//	totalNS   — por.Prove() = sum + struct construction
//
// One tree is built per sweep point and reused across reps; building
// the tree is part of ingest, not prove (see CLAUDE.md §4.4 — the
// responder already holds the tree). Tree-build time is reported
// separately as buildTreeNS in the row.
type proveResult struct {
	BuildTreeNS int64    // wall to merkle.Build the synthetic bundle
	BuildBytes  int      // bundle size fed to Build (= numChunks × ChunkSize)
	MerkleNS    []int64  // per-rep tree.Proof(idx)
	SignNS      []int64  // per-rep provenance.Sign
	TotalNS     []int64  // per-rep por.Prove (≈ MerkleNS + SignNS + struct setup)
}

func runProve(rng *mrand.Rand, point SweepPoint, reps int) (*proveResult, error) {
	if reps < 1 {
		return nil, fmt.Errorf("reps must be ≥ 1, got %d", reps)
	}

	// Synthesise a bundle. Random bytes (crypto/rand for true entropy —
	// the prove path doesn't depend on payload content, only on tree
	// shape, but using deterministic input would bias sha256 cycle counts
	// across reps). Buffer is reused; per-rep we just slice into it.
	bundleSize := point.BundleSizeBytes()
	payload := make([]byte, bundleSize)
	if _, err := rand.Read(payload); err != nil {
		return nil, fmt.Errorf("rand payload: %w", err)
	}

	tBuildStart := time.Now()
	tree, err := merkle.Build(bytes.NewReader(payload))
	tBuildNS := time.Since(tBuildStart).Nanoseconds()
	if err != nil {
		return nil, fmt.Errorf("merkle.Build: %w", err)
	}
	if tree.NumChunks() != point.NumChunks {
		return nil, fmt.Errorf("tree.NumChunks=%d want %d", tree.NumChunks(), point.NumChunks)
	}

	// One responder BLS keypair per sweep point. Reusing across reps
	// matches the deployed-gateway model (see cmd/bench/eprov §realism
	// notes) and keeps keygen out of the per-rep critical path.
	kp, err := provenance.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("GenerateKey: %w", err)
	}

	out := &proveResult{
		BuildTreeNS: tBuildNS,
		BuildBytes:  bundleSize,
		MerkleNS:    make([]int64, reps),
		SignNS:      make([]int64, reps),
		TotalNS:     make([]int64, reps),
	}

	for r := 0; r < reps; r++ {
		chunkIdx := rng.Intn(point.NumChunks)
		nonce := [32]byte{}
		// Mix iter into nonce so signMessage differs per rep — prevents
		// any potential libsodium-style sig caching from biasing the
		// distribution. The first 8 bytes are enough to vary the input.
		for j := 0; j < 8; j++ {
			nonce[j] = byte(rng.Intn(256))
		}
		ch := por.Challenge{
			BundleID: [32]byte{0xab, 0xcd, byte(point.Depth)},
			ShardIdx: 0,
			ChunkIdx: uint32(chunkIdx),
			Nonce:    nonce,
		}

		chunkBytes := payload[chunkIdx*merkle.ChunkSize : (chunkIdx+1)*merkle.ChunkSize]

		// Stage 1: merkle proof construction (isolated).
		t0 := time.Now()
		_, perr := tree.Proof(chunkIdx)
		mNS := time.Since(t0).Nanoseconds()
		if perr != nil {
			return nil, fmt.Errorf("rep %d: tree.Proof: %w", r, perr)
		}
		out.MerkleNS[r] = mNS

		// Stage 2: BLS sign (isolated).
		t0 = time.Now()
		_, serr := provenance.Sign(kp.PrivateBytes, por.ProveMessage(chunkBytes, ch.Nonce))
		sNS := time.Since(t0).Nanoseconds()
		if serr != nil {
			return nil, fmt.Errorf("rep %d: Sign: %w", r, serr)
		}
		out.SignNS[r] = sNS

		// Stage total: por.Prove on the same inputs (independent run, so
		// timer captures the production code path including struct setup
		// + sha256 leaf hash that the isolated stages skip).
		t0 = time.Now()
		_, terr := por.Prove(tree, chunkBytes, chunkIdx, ch, kp.PrivateBytes)
		tNS := time.Since(t0).Nanoseconds()
		if terr != nil {
			return nil, fmt.Errorf("rep %d: por.Prove: %w", r, terr)
		}
		out.TotalNS[r] = tNS
	}

	return out, nil
}

// percentiles returns p50, p95, p99, min, max in nanoseconds. Caller
// supplies an already-populated slice; we sort a copy so the input
// remains untouched (the bench JSON wants the raw per-rep series).
func percentiles(samples []int64) Percentiles {
	if len(samples) == 0 {
		return Percentiles{}
	}
	sorted := make([]int64, len(samples))
	copy(sorted, samples)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	pick := func(p float64) int64 {
		idx := int(float64(len(sorted)-1) * p)
		return sorted[idx]
	}

	return Percentiles{
		P50: pick(0.50),
		P95: pick(0.95),
		P99: pick(0.99),
		Min: sorted[0],
		Max: sorted[len(sorted)-1],
	}
}

// Percentiles is a small struct used for both prove-time and gas
// distributions. Gas has only a single point per (fn, depth) so its
// fields are identical when only one observation exists.
type Percentiles struct {
	P50 int64 `json:"p50"`
	P95 int64 `json:"p95"`
	P99 int64 `json:"p99"`
	Min int64 `json:"min"`
	Max int64 `json:"max"`
}
