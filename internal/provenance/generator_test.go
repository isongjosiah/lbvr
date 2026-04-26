package provenance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// minimalInput returns a GenerateInput sufficient to pass validation.
// Tests start from this and override individual fields to probe edge
// cases.
func minimalInput() GenerateInput {
	gw := GatewayAgent{
		ProvType: "prov:SoftwareAgent", Role: "retrieval_gateway",
		Version: "lbvr-med-0.1.0",
		// 96 hex chars after 0x — gatewayNodeID slices [2:10].
		PublicKey: "0xaabbccddeeff00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff001122334455667788",
	}
	in := GenerateInput{
		BundleSizeBytes:  1024,
		FHIRResourceType: "Bundle",
		ShardLayout: [3]ShardPlacement{
			{CID: "Qm0", Tier: "pinata", ShardRoot: "0x00"},
			{CID: "Qm1", Tier: "filebase", ShardRoot: "0x01"},
			{CID: "Qm2", Tier: "arweave", ShardRoot: "0x02"},
		},
		StartedAt:    time.Date(2026, 4, 30, 14, 32, 15, 0, time.UTC),
		EndedAt:      time.Date(2026, 4, 30, 14, 32, 17, 0, time.UTC),
		RecoveryMode: "fast_path",
		ShardsUsed:   []string{"D0", "D1"},
		LatencyMs:    1500,
		Requester: RequesterAgent{
			ProvType: "prov:Person", Role: "clinician",
			Institution: "did:lbvr:hosp-1",
			AuthzPolicy: "EHDS-Art44",
		},
		Gateways:        []GatewayAgent{gw},
		QuorumThreshold: 1,
	}
	return in
}

// TestGenerate_FastPath_OnlyEmitsUsedShards: a fast-path retrieval
// uses D0+D1 and skips P0. The resulting document should contain
// shard entities for D0 and D1 only — emitting P0 would imply
// erroneously that the parity shard contributed to the reconstruction.
func TestGenerate_FastPath_OnlyEmitsUsedShards(t *testing.T) {
	in := minimalInput()
	in.ShardsUsed = []string{"D0", "D1"}

	doc, err := Generate(in)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for id := range doc.Entity {
		if strings.Contains(id, "shard/P0") {
			t.Errorf("fast path should not emit P0 shard entity, got %q", id)
		}
	}
	wantShards := 0
	for id := range doc.Entity {
		if strings.HasPrefix(id, "lbvr:shard/") {
			wantShards++
		}
	}
	if wantShards != 2 {
		t.Errorf("expected 2 shard entities, got %d", wantShards)
	}
}

// TestGenerate_SlowPath_RSDecode_CoversWasDerivedFrom: when the slow
// path used D1+P0 (D0 was missing), wasDerivedFrom should reference
// D1 and P0 — not D0. The PROV graph encodes which stored shards the
// reconstructed bundle was actually derived from.
func TestGenerate_SlowPath_RSDecode_CoversWasDerivedFrom(t *testing.T) {
	in := minimalInput()
	in.ShardsUsed = []string{"D1", "P0"}
	in.RSDecode = true
	in.RecoveryMode = "slow_path"

	doc, err := Generate(in)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(doc.WasDerivedFrom) != 2 {
		t.Fatalf("expected 2 wasDerivedFrom edges, got %d", len(doc.WasDerivedFrom))
	}
	used := map[string]bool{}
	for _, edge := range doc.WasDerivedFrom {
		used[edge["prov:usedEntity"]] = true
	}
	for k := range used {
		if strings.Contains(k, "shard/D0") {
			t.Errorf("D0 should not appear in wasDerivedFrom for D1+P0 reconstruction, got %q", k)
		}
	}
}

// TestGenerate_RejectsBackwardsTime: endedAt before startedAt is a
// gateway bug; we fail loud rather than emit a nonsensical activity.
func TestGenerate_RejectsBackwardsTime(t *testing.T) {
	in := minimalInput()
	in.EndedAt = in.StartedAt.Add(-1 * time.Second)
	if _, err := Generate(in); err == nil {
		t.Fatal("expected error for endedAt before startedAt")
	}
}

// TestGenerate_RejectsBadQuorumThreshold: threshold > number of
// gateways is a misconfiguration. Caught early so the gateway never
// produces a document that cannot satisfy its own quorum claim.
func TestGenerate_RejectsBadQuorumThreshold(t *testing.T) {
	in := minimalInput()
	in.QuorumThreshold = 99
	if _, err := Generate(in); err == nil {
		t.Fatal("expected error for QuorumThreshold > len(Gateways)")
	}
	in.QuorumThreshold = 0
	if _, err := Generate(in); err == nil {
		t.Fatal("expected error for QuorumThreshold == 0")
	}
}

// TestGenerate_RejectsTooFewShards: <2 ShardsUsed cannot reconstruct
// the bundle (RS(2,3)).
func TestGenerate_RejectsTooFewShards(t *testing.T) {
	in := minimalInput()
	in.ShardsUsed = []string{"D0"}
	if _, err := Generate(in); err == nil {
		t.Fatal("expected error for <2 ShardsUsed")
	}
}

// TestGenerate_RejectsUnknownShardRole: anything outside {D0,D1,P0}
// is rejected. Defends against typos like "D2" leaking into the doc.
func TestGenerate_RejectsUnknownShardRole(t *testing.T) {
	in := minimalInput()
	in.ShardsUsed = []string{"D0", "D2"}
	if _, err := Generate(in); err == nil {
		t.Fatal("expected error for unknown shard role")
	}
}

// TestSetRoot_ProducesStableHash: SetRoot computes the canonical
// hash of the doc-without-root and embeds it. Calling Marshal
// (which canonicalises) and re-hashing the output should yield the
// same hash if we strip the root again — i.e. SetRoot is fixed-point.
func TestSetRoot_ProducesStableHash(t *testing.T) {
	in := minimalInput()
	doc, err := Generate(in)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if err := doc.SetRoot(nil); err != nil {
		t.Fatalf("setRoot: %v", err)
	}
	root := doc.ProvenanceRoot
	if root == nil || root.Root == "" {
		t.Fatal("SetRoot did not populate ProvenanceRoot")
	}
	if root.Canonicalization != CanonicalizationJCS {
		t.Errorf("canonicalization = %q want %q", root.Canonicalization, CanonicalizationJCS)
	}

	// Round-trip through Marshal + parse + strip + rehash.
	marshalled, err := doc.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var d2 Document
	if err := json.Unmarshal(marshalled, &d2); err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	d2.ProvenanceRoot = nil
	bs, err := json.Marshal(d2)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	got, err := CanonicalHash(bs)
	if err != nil {
		t.Fatalf("rehash: %v", err)
	}
	want, _ := decodeHexPrefixed(root.Root)
	if string(got[:]) != string(want) {
		t.Errorf("hash drift: %x vs %x", got, want)
	}
}

// TestSetRoot_PreservesAnchorMetadata: passing a non-nil anchor
// copies the chain fields over. Used by the gateway after the on-chain
// anchor txn confirms.
func TestSetRoot_PreservesAnchorMetadata(t *testing.T) {
	in := minimalInput()
	doc, _ := Generate(in)
	anchor := &ProvenanceRoot{
		AnchoredOnChain: true,
		ChainTxHash:     "0xabc",
		BlockNumber:     123,
		AnchorContract:  "0xdef",
		AnchoredAt:      "2026-04-30T15:00:00.000Z",
	}
	if err := doc.SetRoot(anchor); err != nil {
		t.Fatalf("setRoot: %v", err)
	}
	if !doc.ProvenanceRoot.AnchoredOnChain {
		t.Error("AnchoredOnChain not preserved")
	}
	if doc.ProvenanceRoot.ChainTxHash != "0xabc" {
		t.Errorf("ChainTxHash: %q", doc.ProvenanceRoot.ChainTxHash)
	}
	if doc.ProvenanceRoot.BlockNumber != 123 {
		t.Errorf("BlockNumber: %d", doc.ProvenanceRoot.BlockNumber)
	}
}

// TestGenerate_AllThreeShardsUsed: a verifier could run with all 3
// shards (e.g. cross-checking parity). Document should emit all three
// entity nodes and three wasDerivedFrom edges.
func TestGenerate_AllThreeShardsUsed(t *testing.T) {
	in := minimalInput()
	in.ShardsUsed = []string{"D0", "D1", "P0"}

	doc, err := Generate(in)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	shards := 0
	for id := range doc.Entity {
		if strings.HasPrefix(id, "lbvr:shard/") {
			shards++
		}
	}
	if shards != 3 {
		t.Errorf("expected 3 shard entities, got %d", shards)
	}
	if len(doc.WasDerivedFrom) != 3 {
		t.Errorf("expected 3 wasDerivedFrom edges, got %d", len(doc.WasDerivedFrom))
	}
}

// TestDedupShardRoles: a defensive dedup runs inside Generate so that
// a caller passing duplicates doesn't create duplicate shard entities.
func TestDedupShardRoles(t *testing.T) {
	got := dedupShardRoles([]string{"D0", "D0", "D1"})
	if len(got) != 2 {
		t.Errorf("expected 2 unique roles, got %v", got)
	}
}
