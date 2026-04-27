// Provenance integration: builds, signs, anchors, and caches the
// PROV-JSON document for every successful retrieval (CLAUDE.md §4.6;
// docs/provenance-spec.md). The hot-path cost is measured in E-PROV;
// fail-closed semantics mean a provenance failure aborts the response
// (a retrieval that cannot be accounted for is not allowed to leak).
//
// Cache + side-endpoint design: the PROV doc is stored in an in-memory
// LRU keyed by retrievalID; the bundle response carries
// X-LBVR-Retrieval-ID so the verifier (or oncall, or auditor) can fetch
// the doc via GET /prov/{retrievalID}. Production should swap the LRU
// for a Pinata upload that returns a CID — TODO(D12+).

package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/gateway"
	"github.com/isongjosiah/lbvr-med/internal/provenance"
	"github.com/isongjosiah/lbvr-med/internal/registry"
)

// provCache is the in-memory store of recently-emitted PROV docs.
// 1024 entries × ~5 KiB each ≈ 5 MiB ceiling; eviction is FIFO not LRU
// because the access pattern is "fetch once shortly after the bundle"
// and the gain from LRU is not worth the bookkeeping for D11 scope.
type provCache struct {
	mu    sync.RWMutex
	store map[[32]byte][]byte
	order [][32]byte
	cap   int
}

func newProvCache(capacity int) *provCache {
	if capacity <= 0 {
		capacity = 1024
	}
	return &provCache{store: make(map[[32]byte][]byte), cap: capacity}
}

func (c *provCache) put(retrievalID [32]byte, doc []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.store[retrievalID]; ok {
		c.store[retrievalID] = doc
		return
	}
	if len(c.store) >= c.cap {
		evict := c.order[0]
		c.order = c.order[1:]
		delete(c.store, evict)
	}
	c.store[retrievalID] = doc
	c.order = append(c.order, retrievalID)
}

func (c *provCache) get(retrievalID [32]byte) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	doc, ok := c.store[retrievalID]
	return doc, ok
}

// emitProvenanceInput is the per-request data the gateway already has
// at retrieval-completion time.
type emitProvenanceInput struct {
	BundleID        [32]byte
	Record          *registry.BundleRecord
	BundleSizeBytes int64
	Stats           gateway.RecoveryStats
	StartedAt       time.Time
	EndedAt         time.Time
}

// emitProvenance builds, signs, anchors, and finalises the PROV doc.
// Returns the marshalled JCS-canonical bytes and the freshly-minted
// retrievalID. Fail-closed: any error here means the retrieval is
// abandoned by the caller (no body returned to the client).
func (g *Gateway) emitProvenance(ctx context.Context, in emitProvenanceInput) ([]byte, [32]byte, error) {
	if g.anchor == nil || len(g.signerKeys) == 0 {
		return nil, [32]byte{}, errors.New("provenance: gateway not configured for provenance")
	}

	var retrievalID [32]byte
	if _, err := rand.Read(retrievalID[:]); err != nil {
		return nil, retrievalID, fmt.Errorf("provenance: retrievalID: %w", err)
	}

	genIn := provenance.GenerateInput{
		BundleID:         in.BundleID,
		MerkleRoot:       in.Record.MerkleRoot,
		ShardLayout:      mapShardLayout(in.Record.Shards),
		BundleSizeBytes:  in.BundleSizeBytes,
		FHIRResourceType: "Bundle",

		RetrievalID:  retrievalID,
		StartedAt:    in.StartedAt,
		EndedAt:      in.EndedAt,
		RecoveryMode: recoveryModeString(in.Stats.Mode),
		ShardsUsed:   shardsUsed(in.Stats),
		RSDecode:     in.Stats.DecodeNanos > 0,
		LatencyMs:    in.EndedAt.Sub(in.StartedAt).Milliseconds(),

		Requester:       g.requester,
		Gateways:        g.gatewayAgents,
		QuorumThreshold: g.quorumThreshold,
	}

	doc, err := provenance.Generate(genIn)
	if err != nil {
		return nil, retrievalID, fmt.Errorf("provenance: generate: %w", err)
	}
	if err := doc.Sign(g.signerKeys, g.signerDIDs, g.quorumThreshold); err != nil {
		return nil, retrievalID, fmt.Errorf("provenance: sign: %w", err)
	}

	// Compute the JCS canonical hash *with provenanceRoot stripped*.
	// SetRoot will then attach the root payload (hash + on-chain anchor
	// info) without polluting the hash that gets anchored.
	pre, err := doc.Marshal()
	if err != nil {
		return nil, retrievalID, fmt.Errorf("provenance: marshal pre-anchor: %w", err)
	}
	hash := sha256.Sum256(pre)

	anchor, err := g.anchor.Anchor(ctx, in.BundleID, retrievalID, hash)
	if err != nil {
		return nil, retrievalID, fmt.Errorf("provenance: anchor: %w", err)
	}

	root := &provenance.ProvenanceRoot{
		Algorithm:        "SHA-256",
		Root:             "0x" + hex.EncodeToString(hash[:]),
		Canonicalization: "JCS-RFC8785",
		AnchoredOnChain:  true,
		ChainTxHash:      anchor.TxHash,
		BlockNumber:      anchor.BlockNumber,
		AnchorContract:   g.anchorContract,
		AnchoredAt:       time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	if err := doc.SetRoot(root); err != nil {
		return nil, retrievalID, fmt.Errorf("provenance: setRoot: %w", err)
	}

	final, err := doc.Marshal()
	if err != nil {
		return nil, retrievalID, fmt.Errorf("provenance: marshal final: %w", err)
	}
	g.provCache.put(retrievalID, final)
	return final, retrievalID, nil
}

// mapShardLayout converts the registry's tier-class enum into the spec's
// string tier names ("pinata"/"filebase"/"arweave") and computes a
// per-shard SHA-256 placeholder. The real per-shard Merkle root would
// be carried in the registry; the schema bump for that is journal-scope
// (CLAUDE.md §3.2). For now we synthesise from the CID hash so the
// field is non-empty and verifiable as deterministic.
func mapShardLayout(shards [3]registry.ShardPlacement) [3]provenance.ShardPlacement {
	out := [3]provenance.ShardPlacement{}
	tierNames := [3]string{"pinata", "filebase", "arweave"}
	for i, s := range shards {
		h := sha256.Sum256([]byte(s.CID))
		out[i] = provenance.ShardPlacement{
			CID:       s.CID,
			Tier:      tierNames[i],
			ShardRoot: "0x" + hex.EncodeToString(h[:]),
		}
	}
	return out
}

// recoveryModeString maps the gateway recovery enum to spec strings.
func recoveryModeString(m gateway.RecoveryMode) string {
	switch m {
	case gateway.RecoveryFastPath:
		return "fast_path"
	case gateway.RecoverySlowPath:
		return "slow_path"
	default:
		return "failure"
	}
}

// shardsUsed reports which shard slots actually contributed to the
// reconstructed bundle. Fast path always credits D0+D1 (P0 is unused
// by design even when it happened to return before cancellation).
// Slow path credits whichever 2 of 3 returned without error.
func shardsUsed(stats gateway.RecoveryStats) []string {
	names := []string{"D0", "D1", "P0"}
	if stats.Mode == gateway.RecoveryFastPath {
		return []string{"D0", "D1"}
	}
	out := make([]string, 0, 2)
	for i, e := range stats.ShardErrors {
		if e == nil {
			out = append(out, names[i])
		}
	}
	return out
}

// parseRetrievalID parses 64 hex chars (with or without 0x prefix) into
// a [32]byte.
func parseRetrievalID(s string) ([32]byte, error) {
	var out [32]byte
	if len(s) >= 2 && (s[:2] == "0x" || s[:2] == "0X") {
		s = s[2:]
	}
	if len(s) != 64 {
		return out, fmt.Errorf("retrievalID must be 32 bytes (64 hex chars), got %d", len(s))
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return out, fmt.Errorf("retrievalID hex: %w", err)
	}
	copy(out[:], b)
	return out, nil
}
