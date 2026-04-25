// Package erasure implements the Reed-Solomon RS(2,3) cross-tier redundancy
// layer for LBVR-Med (see CLAUDE.md §4.5 and docs/erasure-design.md).
//
// One bundle is split into 2 data shards + 1 parity shard. Any 2 of 3 shards
// reconstruct the original — survives one complete tier outage. Placement
// (which shard goes to which tier) is decided by the orchestrator, not here.
//
// The package is deliberately self-contained: no merkle/crypto/tier deps, so
// it can be unit-tested in isolation. The integrity authority is the
// internal/merkle layer; this package's tamper detection is best-effort and
// only triggers when all 3 shards are present (see decoder.go).
package erasure

import (
	"errors"

	"github.com/klauspost/reedsolomon"
)

// MaxInputBytes caps Encode's input at 1 GiB. Synthea's measured max bundle is
// ~98 MB (CLAUDE.md §3.1, eval/results/synthea-100000/validation.json), so 1
// GiB is ~10x headroom. The cap exists to fail fast on accidental allocator
// blowup (a corrupted size header would otherwise OOM the process).
const MaxInputBytes = 1 << 30

// Shard layout constants — RS(2,3) is locked by CLAUDE.md §4.5; do not
// parameterise. The journal extension upgrades to RS(3,5).
const (
	dataShards   = 2
	parityShards = 1
	totalShards  = dataShards + parityShards
)

// Named errors. Callers compare with errors.Is.
var (
	ErrEmptyInput         = errors.New("erasure: empty input")
	ErrInputTooLarge      = errors.New("erasure: input exceeds MaxInputBytes")
	ErrInsufficientShards = errors.New("erasure: need at least 2 of 3 shards")
	ErrShardSizeMismatch  = errors.New("erasure: shards have inconsistent sizes")
)

// rsEncoder is constructed once at package init and reused. The reedsolomon
// encoder is stateless and safe for concurrent use (see klauspost/reedsolomon
// docs); reusing it avoids per-call SIMD-table setup that would otherwise
// dominate small-blob benchmarks.
var rsEncoder reedsolomon.Encoder

func init() {
	enc, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		// reedsolomon.New only fails on invalid (k,m) pairs; (2,1) is hard-coded
		// and validated by the constants above, so this is unreachable in
		// practice. Panic rather than swallow because a misconfigured encoder
		// would silently produce unrecoverable bundles.
		panic("erasure: failed to construct RS(2,1) encoder: " + err.Error())
	}
	rsEncoder = enc
}
