package main

import (
	"errors"
)

// Encoder is the abstraction the ingest pipeline uses to fan an encrypted
// bundle out into the three on-tier shards (CLAUDE.md §4.2 step 4 / §4.5).
//
// A parallel agent owns internal/erasure with the real RS(2,3) Encode. To
// avoid coupling D6 to that work we depend only on this minimal interface;
// the wiring is one line in cmd/client/main.go once both land.
//
// Contract:
//   - Encode receives the AES-GCM-sealed concatenated chunks.
//   - It returns exactly 3 shards (RS(2,3) invariant — see CIDRegistry.sol's
//     _SHARD_COUNT). paddedLen is the size to which `data` was padded
//     before encoding, recorded so the retrieval gateway can strip the
//     padding after Decode.
//   - For deterministic content addressing, Encode must be a pure function
//     of `data` — same input → same shards.
type Encoder interface {
	Encode(data []byte) (shards [3][]byte, paddedLen int, err error)
}

// replicaEncoder produces three identical copies of data — the trivial
// "encoder" used as a placeholder until internal/erasure lands. With this
// stub the gateway's "any 2 of 3" decode logic still works (any single
// shard reconstructs the full payload), but the cross-tier storage cost
// is 3× rather than RS(2,3)'s 1.5×, and it does NOT exercise the real
// reconstruction maths. Treat this as compile-time scaffolding only.
//
// TODO(integration): replace at the call site in main.go with the
// internal/erasure encoder. The interface above is shaped so the swap is
// a one-liner — internal/erasure.Encode already has the matching
// `(shards [3][]byte, paddedLen int, err error)` signature, so wrapping
// it is just:
//
//	type erasureEnc struct{}
//	func (erasureEnc) Encode(d []byte) ([3][]byte, int, error) { return erasure.Encode(d) }
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
