// Anchor client for on-chain provenance hashes (CLAUDE.md §4.6;
// docs/provenance-spec.md §6). The gateway anchors the JCS-canonical
// SHA-256 of every retrieval's signed PROV-JSON doc so a verifier can
// later cross-check the document's hash against on-chain bytes.
//
// Two implementations:
//   - mockAnchor: in-memory map. Used by tests and by `lbvr-gateway`
//     until the AuditorLog Cardona deployment + abigen bindings land
//     (D12+). Anchors are accessible to test code via mockAnchor.Get
//     for verifier wiring.
//   - chainAnchor: stub returning ErrChainNotImplemented. Real impl
//     wraps a go-ethereum client + abigen-generated AuditorLog binding.

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
)

// ErrChainNotImplemented is returned by chainAnchor until D12 wires up
// abigen bindings against the deployed AuditorLog contract.
var ErrChainNotImplemented = errors.New("anchor: chain client not implemented; run forge build + abigen first")

// AnchorRecord mirrors AuditorLog.ProvenanceAnchor on-chain.
type AnchorRecord struct {
	ProvHash    [32]byte
	BlockNumber uint64
	TxHash      string
	AnchoredBy  string // 0x-hex address
}

// AnchorClient is the surface the gateway depends on. Both impls satisfy
// it without sharing internals; chainAnchor wraps go-ethereum and abigen,
// mockAnchor wraps a mutex+map.
type AnchorClient interface {
	Anchor(ctx context.Context, bundleID, retrievalID, provHash [32]byte) (AnchorRecord, error)
	Get(ctx context.Context, bundleID, retrievalID [32]byte) (AnchorRecord, error)
}

// mockAnchor is the in-memory implementation. block numbers are a
// monotonic counter starting at 1 so the doc's blockNumber field is
// non-zero (otherwise verifiers might confuse "anchored at block 0"
// with "no anchor"). Concurrent-safe.
type mockAnchor struct {
	mu        sync.RWMutex
	anchors   map[[64]byte]AnchorRecord // key = bundleID || retrievalID
	nextBlock uint64
}

func newMockAnchor() *mockAnchor {
	return &mockAnchor{
		anchors:   make(map[[64]byte]AnchorRecord),
		nextBlock: 1,
	}
}

func (m *mockAnchor) Anchor(_ context.Context, bundleID, retrievalID, provHash [32]byte) (AnchorRecord, error) {
	if provHash == ([32]byte{}) {
		return AnchorRecord{}, errors.New("anchor: provHash is zero")
	}
	key := concatID(bundleID, retrievalID)

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.anchors[key]; ok {
		// Match contract semantics: re-anchoring the same key reverts.
		return AnchorRecord{}, fmt.Errorf("anchor: already anchored at block %d", existing.BlockNumber)
	}

	rec := AnchorRecord{
		ProvHash:    provHash,
		BlockNumber: m.nextBlock,
		TxHash:      "0xmock" + hex.EncodeToString(randBytes(28)),
		AnchoredBy:  "0xmockGateway0000000000000000000000000000",
	}
	m.anchors[key] = rec
	m.nextBlock++
	return rec, nil
}

func (m *mockAnchor) Get(_ context.Context, bundleID, retrievalID [32]byte) (AnchorRecord, error) {
	key := concatID(bundleID, retrievalID)
	m.mu.RLock()
	defer m.mu.RUnlock()
	rec, ok := m.anchors[key]
	if !ok {
		return AnchorRecord{}, errors.New("anchor: not found")
	}
	return rec, nil
}

func concatID(bundleID, retrievalID [32]byte) (out [64]byte) {
	copy(out[:32], bundleID[:])
	copy(out[32:], retrievalID[:])
	return
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

// chainAnchor is the stubbed real implementation. D12 swaps the body
// for go-ethereum + abigen bindings; the surface stays identical.
type chainAnchor struct {
	rpcURL          string
	contractAddress string
}

func newChainAnchor(rpcURL, contractAddress string) (*chainAnchor, error) {
	if rpcURL == "" {
		return nil, errors.New("chainAnchor: rpcURL is empty")
	}
	if contractAddress == "" {
		return nil, errors.New("chainAnchor: contractAddress is empty")
	}
	return &chainAnchor{rpcURL: rpcURL, contractAddress: contractAddress}, nil
}

func (c *chainAnchor) Anchor(_ context.Context, _, _, _ [32]byte) (AnchorRecord, error) {
	return AnchorRecord{}, ErrChainNotImplemented
}
func (c *chainAnchor) Get(_ context.Context, _, _ [32]byte) (AnchorRecord, error) {
	return AnchorRecord{}, ErrChainNotImplemented
}
