package erasure

import (
	"bytes"
	"fmt"
)

// Decode reconstructs the original data from any 2 of 3 shards. Missing
// shards are passed as nil. paddedLen must equal the value Encode returned
// for this bundle — semantically the *original* input length (see encoder.go
// for the parameter-name caveat). It lives alongside merkleRoot in the
// CIDRegistry, and is what lets Decode trim the trailing zero padding.
//
// Returns:
//   - ErrInsufficientShards if fewer than 2 shards are non-nil
//   - ErrShardSizeMismatch if non-nil shards have different lengths
//   - the underlying reedsolomon error on Reconstruct/Verify failure
//
// Note on integrity: Verify only fires when all 3 shards are present after
// reconstruction; with exactly 2 surviving shards there is no redundancy
// left to detect tampering. Callers MUST verify the reconstructed bytes
// against the bundle's Merkle root (internal/merkle) — that is the only
// authoritative integrity check. Erasure-layer Verify is best-effort.
//
// Implementation note: reedsolomon.Reconstruct fills in missing slots in
// the input slice in place. The fixed-size [3][]byte argument is therefore
// projected to a local [][]byte; callers' input arrays are not mutated, but
// the per-shard byte slices may be — do not reuse a shard slice expecting
// it unchanged after a successful Decode.
func Decode(shards [totalShards][]byte, paddedLen int) ([]byte, error) {
	if paddedLen <= 0 || paddedLen > MaxInputBytes {
		return nil, fmt.Errorf("erasure: invalid paddedLen %d", paddedLen)
	}

	// Count survivors and validate sizes. All non-nil shards must have the
	// same length (RS shards are equal-sized by construction).
	present := 0
	var shardLen int
	for i := 0; i < totalShards; i++ {
		if shards[i] == nil {
			continue
		}
		if shardLen == 0 {
			shardLen = len(shards[i])
		} else if len(shards[i]) != shardLen {
			return nil, ErrShardSizeMismatch
		}
		present++
	}
	if present < dataShards {
		return nil, ErrInsufficientShards
	}
	// paddedLen must be consistent with the shard size: original length lies in
	// ((shardLen-1)*dataShards, shardLen*dataShards]. Outside that range, the
	// registry record is corrupt or shards belong to a different bundle.
	if paddedLen > shardLen*dataShards || paddedLen <= (shardLen-1)*dataShards {
		return nil, fmt.Errorf("erasure: paddedLen %d inconsistent with shard size %d", paddedLen, shardLen)
	}

	// reedsolomon expects a [][]byte; copy from the fixed-size array. Don't
	// alias: Reconstruct allocates new slices for missing entries, so the
	// caller's array stays as-is, but a successful return means decoded
	// shards live in this local slice's backing buffers.
	work := make([][]byte, totalShards)
	for i := 0; i < totalShards; i++ {
		work[i] = shards[i]
	}

	if err := rsEncoder.Reconstruct(work); err != nil {
		return nil, fmt.Errorf("erasure: reconstruct: %w", err)
	}

	// Best-effort integrity check (see method doc): only meaningful with all 3
	// shards present after reconstruction, which is always the case post-
	// Reconstruct. Verify recomputes parity over data shards and compares.
	ok, err := rsEncoder.Verify(work)
	if err != nil {
		return nil, fmt.Errorf("erasure: verify: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("erasure: verify: parity mismatch (corrupted shard)")
	}

	// Reassemble the data shards via Join with outSize=paddedLen so trailing
	// zero pad bytes are dropped (Join writes exactly outSize bytes from the
	// concatenated data shards).
	var out bytes.Buffer
	out.Grow(paddedLen)
	if err := rsEncoder.Join(&out, work, paddedLen); err != nil {
		return nil, fmt.Errorf("erasure: join: %w", err)
	}
	return out.Bytes(), nil
}
