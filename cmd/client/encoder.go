package main

import (
	"errors"

	"github.com/isongjosiah/lbvr-med/internal/erasure"
)

// Encoder is the abstraction the ingest pipeline uses to fan an encrypted
// bundle out into the three on-tier shards (CLAUDE.md §4.2 step 4 / §4.5).
//
// Contract:
//   - Encode receives the AES-GCM-sealed concatenated chunks.
//   - It returns exactly 3 shards (RS(2,3) invariant — see CIDRegistry.sol's
//     _SHARD_COUNT). paddedLen is the original input length the gateway
//     needs to trim trailing padding off after Decode.
//   - For deterministic content addressing, Encode must be a pure function
//     of `data` — same input → same shards.
type Encoder interface {
	Encode(data []byte) (shards [3][]byte, paddedLen int, err error)
}

// erasureEncoder is the production default — wraps internal/erasure.Encode
// (klauspost/reedsolomon under the hood). RS(2,3) gives single-tier-failure
// tolerance at 1.5× storage overhead.
type erasureEncoder struct{}

// Encode implements Encoder via the RS(2,3) encoder.
func (erasureEncoder) Encode(data []byte) (shards [3][]byte, paddedLen int, err error) {
	return erasure.Encode(data)
}

// replicaEncoder produces three identical copies of data. Kept for tests
// where the assertion is "every tier got the same bytes" — useful for
// isolating tier-client behaviour from erasure correctness. Production
// callers must use erasureEncoder.
type replicaEncoder struct{}

// Encode implements Encoder by replicating data three times.
func (replicaEncoder) Encode(data []byte) (shards [3][]byte, paddedLen int, err error) {
	if len(data) == 0 {
		return shards, 0, errors.New("encoder: empty payload")
	}
	for i := 0; i < 3; i++ {
		shards[i] = append([]byte(nil), data...) // independent copies
	}
	return shards, len(data), nil
}
