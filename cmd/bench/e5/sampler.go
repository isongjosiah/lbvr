package main

import (
	"fmt"

	"github.com/isongjosiah/lbvr-med/internal/merkle"
)

// sweepPoints are the canonical Merkle proof depths E5 measures. Picked
// to cover the empirical Synthea bundle distribution end-to-end (CLAUDE.md
// §4.5):
//
//	depth=3  → 8 chunks  → 128 KiB bundle (≈ measured P0.24)
//	depth=5  → 32 chunks → 512 KiB
//	depth=7  → 128 chunks → 2 MiB    (≈ P50)
//	depth=9  → 512 chunks → 8 MiB
//	depth=11 → 2048 chunks → 32 MiB  (≈ P95)
//	depth=13 → 8192 chunks → 128 MiB (above measured max — keeps trend visible)
//
// The depth → numChunks mapping is exactly 2^depth: every sample sits at
// a clean tree height boundary so per-level effects are unambiguous in
// the figure.
var sweepPoints = []SweepPoint{
	{Depth: 3, NumChunks: 1 << 3},
	{Depth: 5, NumChunks: 1 << 5},
	{Depth: 7, NumChunks: 1 << 7},
	{Depth: 9, NumChunks: 1 << 9},
	{Depth: 11, NumChunks: 1 << 11},
	{Depth: 13, NumChunks: 1 << 13},
}

// SweepPoint is one row of the bench matrix.
type SweepPoint struct {
	Depth     int
	NumChunks int
}

// BundleSizeBytes returns the on-disk size we'd present to the prove
// path: numChunks × ChunkSize. We pad every bundle to a chunk boundary
// because the bench's prove time should not include partial-chunk
// arithmetic — the contract verifies the leaf hash, not the chunk size.
func (p SweepPoint) BundleSizeBytes() int {
	return p.NumChunks * merkle.ChunkSize
}

func (p SweepPoint) String() string {
	return fmt.Sprintf("depth=%d chunks=%d size=%dB", p.Depth, p.NumChunks, p.BundleSizeBytes())
}
