// Package registry is the Go-side view of the on-chain CIDRegistry contract
// (contracts/src/CIDRegistry.sol). It mirrors the Solidity types one-for-one
// so callers can build a BundleRecord without ever touching abigen output, and
// it provides two implementations of the Client interface:
//
//   - Mock — an in-memory, concurrency-safe registry for tests and the D6
//     end-to-end ingest pipeline. Returns ErrAlreadyRegistered / ErrNotFound
//     to mirror the contract's revert paths.
//   - Chain (chain.go) — the real eth-client-backed implementation. Stubbed
//     until D12 (foundryup + abigen).
//
// CLAUDE.md §4.2 step 6 binds the Merkle leaf count (NumChunks) on-chain
// alongside the root; without it, a verifier of an off-chain Merkle proof
// could be tricked by a CVE-2012-2459-class second-preimage attack against
// the duplicated-tail Merkle layout used in internal/merkle. This package
// preserves that binding by storing NumChunks as the same uint32 as the
// contract.
package registry

import (
	"context"
	"errors"
	"time"
)

// Tier classes mirror contracts/src/CIDRegistry.sol:CIDRegistry.TierClass.
// Kept as untyped constants on uint8 so callers can compare against
// internal/tiers.TierHot/Warm/Cold without an extra cast.
const (
	TierHot  uint8 = 0
	TierWarm uint8 = 1
	TierCold uint8 = 2
)

// ShardCount is the fixed RS(2,3) shard count enforced by both the
// contract and the Go-side validators. Conference scope per CLAUDE.md §3.1
// and erasure-design.md; relaxed for the journal extension's RS(3,5).
const ShardCount = 3

// ShardPlacement mirrors CIDRegistry.ShardPlacement. CID is stored as a
// string (multibase form) on the Go side because every tier client returns
// the CID as a string; the contract stores the same value as bytes (see
// the rationale comment in CIDRegistry.sol on storing raw multibase).
type ShardPlacement struct {
	CID  string
	Tier uint8
}

// BundleRecord mirrors CIDRegistry.BundleRecord. RegisteredAt /
// LastMigratedAt come from block.timestamp on-chain; on the Mock side we
// fill them with time.Now() at registration / migration so tests can assert
// "non-zero, monotonic" without standing up an Ethereum simulator.
//
// Owner is the 0x-prefixed lowercase hex address of msg.sender at register
// time. Storing it as a string keeps this package free of an
// ethereum/go-ethereum dependency until chain.go gets wired in.
type BundleRecord struct {
	MerkleRoot     [32]byte
	NumChunks      uint32
	Shards         [ShardCount]ShardPlacement
	Owner          string
	PolicyID       [32]byte
	RegisteredAt   time.Time
	LastMigratedAt time.Time
}

// Client is the abstraction every consumer (D6 ingest CLI, D8 retrieval
// gateway, D11 placement orchestrator) talks to. Both Mock and the future
// chain client honour the same contract:
//
//   - RegisterBundle returns ErrAlreadyRegistered if bundleID is taken;
//     ErrEmptyShardCID if any CID is empty; ErrNumChunksZero on numChunks==0.
//   - GetRecord / GetShardLayout return ErrNotFound for unknown IDs.
//   - UpdateShardLayout returns ErrNotFound on unknown IDs and
//     ErrEmptyShardCID on bad inputs.
type Client interface {
	RegisterBundle(ctx context.Context, bundleID [32]byte, rec BundleRecord) error
	GetRecord(ctx context.Context, bundleID [32]byte) (*BundleRecord, error)
	GetShardLayout(ctx context.Context, bundleID [32]byte) ([ShardCount]ShardPlacement, error)
	UpdateShardLayout(ctx context.Context, bundleID [32]byte, shards [ShardCount]ShardPlacement) error
}

// Sentinel errors. Match the revert reasons in CIDRegistry.sol where they
// have a one-to-one mapping; the contract's Solidity custom errors are
// flattened here because the off-chain caller does not need to discriminate
// between the two empty-CID indices.
var (
	ErrAlreadyRegistered = errors.New("registry: bundle already registered")
	ErrNotFound          = errors.New("registry: bundle not found")
	ErrEmptyShardCID     = errors.New("registry: shard CID is empty")
	ErrNumChunksZero     = errors.New("registry: numChunks is zero")
	ErrInvalidShardCount = errors.New("registry: shard count must be 3")
)

// ValidateShards runs the same checks the contract runs in registerBundle /
// updateShardLayout. Exposed so the ingest CLI can fail fast before paying
// the gas of a revert. The fixed-size [ShardCount]ShardPlacement input
// makes the "exactly 3 shards" check unrepresentable as a runtime error;
// only the empty-CID case can fire here.
func ValidateShards(shards [ShardCount]ShardPlacement) error {
	for i, s := range shards {
		if s.CID == "" {
			return errEmptyShardCIDAt(i)
		}
	}
	return nil
}

// errEmptyShardCIDAt wraps ErrEmptyShardCID with the offending index so logs
// can identify which tier had the missing CID without an extra structured
// field. Kept un-exported because callers should branch on errors.Is.
func errEmptyShardCIDAt(i int) error {
	return &emptyShardCIDError{Index: i}
}

type emptyShardCIDError struct {
	Index int
}

func (e *emptyShardCIDError) Error() string {
	return ErrEmptyShardCID.Error() + ": shard index " + intToStr(e.Index)
}

// Is allows errors.Is(err, ErrEmptyShardCID) to match the wrapped form.
func (e *emptyShardCIDError) Is(target error) bool { return target == ErrEmptyShardCID }

// intToStr is the trivial unsigned-int formatter used inside error strings.
// Avoids importing strconv just for one error path; keeps the dependency
// graph of this package equal to {context, errors, sync, time}.
func intToStr(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
