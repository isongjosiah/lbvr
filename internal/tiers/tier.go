// Package tiers defines the common Client interface implemented by every
// storage backend in LBVR-Med (see CLAUDE.md §4.1 and §4.5). The ingest
// client (L1), placement orchestrator (L2), and retrieval gateway (L3)
// address backends through this interface only.
//
// The cid string is the backend's native content identifier: an IPFS CID
// for the Pinata and Filebase clients, an Arweave/Irys tx id for the
// Arweave client. The on-chain CIDRegistry stores it as []byte so the
// contract does not need to understand the namespace.
package tiers

import (
	"context"
	"time"
)

// TierClass mirrors the on-chain enum (CIDRegistry.sol).
const (
	TierHot  uint8 = 0
	TierWarm uint8 = 1
	TierCold uint8 = 2
)

// Stat is the minimal metadata the retrieval gateway needs for PoR
// scheduling and circuit-breaker decisions. It does not include per-backend
// fields (e.g. Pinata pin regions, Filebase S3 ETag) — those live in
// backend-specific helpers if ever needed.
type Stat struct {
	CID       string
	SizeBytes int64
	StoredAt  time.Time
}

type Putter interface {
	Put(ctx context.Context, data []byte) (cid string, err error)
}

type Getter interface {
	Get(ctx context.Context, cid string) ([]byte, error)
}

type Statter interface {
	Stat(ctx context.Context, cid string) (*Stat, error)
}

type Deleter interface {
	Delete(ctx context.Context, cid string) error
}

// Client is the union interface every tier backend must implement. Name
// and TierClass are used by the placement orchestrator (§4.5) and appear
// verbatim in the on-chain shardLayout struct.
type Client interface {
	Putter
	Getter
	Statter
	Deleter
	Name() string
	TierClass() uint8
}
