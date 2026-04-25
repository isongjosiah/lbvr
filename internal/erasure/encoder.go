package erasure

import "fmt"

// Encode splits data into 2 data shards + 1 parity shard via RS(2,3). All
// three returned shards are the same length: ceil(len(data)/2). The input is
// right-padded with zero bytes internally so Split produces equal-sized
// halves, but the returned paddedLen is the *original* input length — that
// is what Decode needs to trim trailing padding back off and recover the
// caller's exact byte slice. (The parameter name is retained for API
// stability against earlier drafts; semantically it is the original length
// the caller persists.)
//
// Returned slice ordering is stable — [D0, D1, P0] — and is what the
// orchestrator (CLAUDE.md §4.5) maps onto Pinata/Filebase/Arweave.
//
// paddedLen is bounded by MaxInputBytes (1 GiB) and therefore fits in a
// uint32. The on-chain CIDRegistry stores numChunks (chunk count, not byte
// count); converting paddedLen↔numChunks is the caller's responsibility
// because the chunk size lives in internal/merkle, which this package
// deliberately does not import.
func Encode(data []byte) (shards [totalShards][]byte, paddedLen int, err error) {
	if len(data) == 0 {
		return shards, 0, ErrEmptyInput
	}
	if len(data) > MaxInputBytes {
		return shards, 0, ErrInputTooLarge
	}

	origLen := len(data)
	// Pad to a multiple of dataShards so Split produces equal-sized halves.
	// reedsolomon.Split requires this for non-streaming mode.
	padded := origLen
	if padded%dataShards != 0 {
		padded += dataShards - (padded % dataShards)
	}
	buf := make([]byte, padded)
	copy(buf, data) // tail of buf is already zero from make

	split, err := rsEncoder.Split(buf)
	if err != nil {
		return shards, 0, fmt.Errorf("erasure: split: %w", err)
	}
	if len(split) != totalShards {
		// Defensive: Split must return exactly k+m=3 slices for RS(2,1).
		return shards, 0, fmt.Errorf("erasure: split returned %d shards, want %d", len(split), totalShards)
	}
	if err := rsEncoder.Encode(split); err != nil {
		return shards, 0, fmt.Errorf("erasure: encode: %w", err)
	}

	// Copy each shard out of the contiguous backing buffer so callers can
	// upload them to independent tier clients without aliasing — Split's
	// returned slices share the same backing array.
	for i := 0; i < totalShards; i++ {
		shards[i] = append([]byte(nil), split[i]...)
	}
	return shards, origLen, nil
}
