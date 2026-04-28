package main

import (
	"errors"

	"github.com/isongjosiah/lbvr-med/internal/erasure"
	"github.com/isongjosiah/lbvr-med/internal/ingest"
)

// Compile-time guarantees that the CLI-side encoders satisfy the
// orchestration package's Encoder contract. If the contract drifts, the
// build breaks here rather than at the constructor call site.
var (
	_ ingest.Encoder = erasureEncoder{}
	_ ingest.Encoder = replicaEncoder{}
)

// erasureEncoder is the production default — wraps internal/erasure.Encode
// (klauspost/reedsolomon under the hood). RS(2,3) gives single-tier-failure
// tolerance at 1.5× storage overhead.
type erasureEncoder struct{}

// Encode implements ingest.Encoder via the RS(2,3) encoder.
func (erasureEncoder) Encode(data []byte) (shards [3][]byte, paddedLen int, err error) {
	return erasure.Encode(data)
}

// replicaEncoder produces three identical copies of data. Kept for tests
// where the assertion is "every tier got the same bytes" — useful for
// isolating tier-client behaviour from erasure correctness. Production
// callers must use erasureEncoder.
type replicaEncoder struct{}

// Encode implements ingest.Encoder by replicating data three times.
func (replicaEncoder) Encode(data []byte) (shards [3][]byte, paddedLen int, err error) {
	if len(data) == 0 {
		return shards, 0, errors.New("encoder: empty payload")
	}
	for i := 0; i < 3; i++ {
		shards[i] = append([]byte(nil), data...) // independent copies
	}
	return shards, len(data), nil
}
