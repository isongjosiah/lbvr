package provenance

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// fixture builds a fully-signed-and-rooted document and returns it
// alongside the verifier components needed to validate it. Used by
// every end-to-end test below; isolating the setup keeps each test
// readable.
type fixture struct {
	doc          []byte
	bundleID     [32]byte
	retrievalID  [32]byte
	keyResolver  StaticKeyResolver
	anchors      StaticAnchorResolver
	gw1, gw2     GatewayAgent
	gw1Priv      [PrivateKeySize]byte
	gw2Priv      [PrivateKeySize]byte
	canonicalDoc []byte
	hash         [32]byte
}

// newFixture spins up a full signed document with a 2-of-2 quorum.
// Most tests just call this and then run their tampering on the bytes.
func newFixture(t *testing.T) *fixture {
	t.Helper()

	kp1, err := GenerateKey()
	if err != nil {
		t.Fatalf("kp1: %v", err)
	}
	kp2, err := GenerateKey()
	if err != nil {
		t.Fatalf("kp2: %v", err)
	}

	gw1 := GatewayAgent{
		ProvType: "prov:SoftwareAgent", Role: "retrieval_gateway",
		Version: "lbvr-med-0.1.0", PublicKey: hexPrefixed(kp1.PublicBytes[:]),
	}
	gw2 := GatewayAgent{
		ProvType: "prov:SoftwareAgent", Role: "retrieval_gateway",
		Version: "lbvr-med-0.1.0", PublicKey: hexPrefixed(kp2.PublicBytes[:]),
	}
	did1 := "did:lbvr:" + safeIDFragment(gw1.PublicKey[2:10])
	did2 := "did:lbvr:" + safeIDFragment(gw2.PublicKey[2:10])

	var bundleID, retrievalID, merkleRoot [32]byte
	for i := range bundleID {
		bundleID[i] = byte(i)
	}
	for i := range retrievalID {
		retrievalID[i] = byte(255 - i)
	}
	for i := range merkleRoot {
		merkleRoot[i] = byte(i ^ 0x55)
	}

	in := GenerateInput{
		BundleID:         bundleID,
		MerkleRoot:       merkleRoot,
		BundleSizeBytes:  823104,
		FHIRResourceType: "Patient",
		ShardLayout: [3]ShardPlacement{
			{CID: "QmHotXa1", Tier: "pinata", ShardRoot: hexPrefixed([]byte{0x11, 0xaa})},
			{CID: "QmWarmYb2", Tier: "filebase", ShardRoot: hexPrefixed([]byte{0x22, 0xbb})},
			{CID: "QmColdZc3", Tier: "arweave", ShardRoot: hexPrefixed([]byte{0x33, 0xcc})},
		},
		RetrievalID:  retrievalID,
		StartedAt:    time.Date(2026, 4, 30, 14, 32, 15, 231_000_000, time.UTC),
		EndedAt:      time.Date(2026, 4, 30, 14, 32, 17, 109_000_000, time.UTC),
		RecoveryMode: "fast_path",
		ShardsUsed:   []string{"D0", "D1"},
		RSDecode:     false,
		LatencyMs:    1878,
		Requester: RequesterAgent{
			ProvType: "prov:Person", Role: "clinician",
			Institution: "did:lbvr:hosp-1",
			AuthzPolicy: "EHDS-Art44-primary-use",
		},
		Gateways:        []GatewayAgent{gw1, gw2},
		QuorumThreshold: 2,
	}

	doc, err := Generate(in)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if err := doc.Sign(
		[][PrivateKeySize]byte{kp1.PrivateBytes, kp2.PrivateBytes},
		[]string{did1, did2},
		2,
	); err != nil {
		t.Fatalf("sign: %v", err)
	}
	if err := doc.SetRoot(nil); err != nil {
		t.Fatalf("setRoot: %v", err)
	}

	out, err := doc.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Pre-compute the canonical hash with the root removed so the
	// anchor table is consistent with what the verifier will derive.
	stripped := *doc
	stripped.ProvenanceRoot = nil
	strippedJSON, err := json.Marshal(stripped)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	hash, err := CanonicalHash(strippedJSON)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	// Verifier uses short-form 4-byte IDs derived from the document's
	// node-IDs (see extractIDs); the anchor table must be keyed by the
	// same shortened form.
	var shortBundle, shortRetrieval [32]byte
	copy(shortBundle[:], bundleID[:4])
	copy(shortRetrieval[:], retrievalID[:4])

	keys := StaticKeyResolver{
		did1: kp1.PublicBytes,
		did2: kp2.PublicBytes,
	}
	anchors := StaticAnchorResolver{}
	anchors.SetAnchor(shortBundle, shortRetrieval, hash, 8923417)

	return &fixture{
		doc:          out,
		bundleID:     shortBundle,
		retrievalID:  shortRetrieval,
		keyResolver:  keys,
		anchors:      anchors,
		gw1:          gw1,
		gw2:          gw2,
		gw1Priv:      kp1.PrivateBytes,
		gw2Priv:      kp2.PrivateBytes,
		canonicalDoc: strippedJSON,
		hash:         hash,
	}
}

// TestE2E_HappyPath: full ingest → sign → anchor → verify, no
// tampering. Must produce a Valid result.
func TestE2E_HappyPath(t *testing.T) {
	fx := newFixture(t)
	v := &Verifier{Keys: fx.keyResolver, Anchors: fx.anchors}
	res, err := v.Verify(fx.doc)
	if err != nil {
		t.Fatalf("verify: %v (reason=%s)", err, res.FailureReason)
	}
	if !res.Valid {
		t.Fatalf("expected Valid=true, got reason=%q", res.FailureReason)
	}
	if res.AnchoredBlock != 8923417 {
		t.Errorf("anchored block: got %d want 8923417", res.AnchoredBlock)
	}
	if len(res.SignatureChecks) != 2 {
		t.Errorf("expected 2 signature checks, got %d", len(res.SignatureChecks))
	}
	for _, c := range res.SignatureChecks {
		if !c.Valid {
			t.Errorf("sig check %s/%s failed: %s", c.NodeKind, c.NodeID, c.Reason)
		}
	}
}

// TestE2E_HashTampering_LatencyMs: spec §7.3 case (1) — modify a
// claimed performance metric. Hash mismatch is detected against the
// on-chain anchor.
func TestE2E_HashTampering_LatencyMs(t *testing.T) {
	fx := newFixture(t)
	tampered := mutateField(t, fx.doc, `"lbvr:latencyMs":1878`, `"lbvr:latencyMs":500`)
	v := &Verifier{Keys: fx.keyResolver, Anchors: fx.anchors}
	res, _ := v.Verify(tampered)
	if res.Valid {
		t.Fatal("expected invalid after latency tamper")
	}
	if res.FailureReason != "hash_mismatch" {
		t.Errorf("expected hash_mismatch, got %q", res.FailureReason)
	}
}

// TestE2E_SignatureForgery: replace one byte of the entity-integrity
// signature with garbage. The hash check passes (we re-canonicalise
// the document including the tampered sig field, then compare against
// an anchor we generate ourselves to keep the test focused on signature
// validation). The BLS verify fails.
func TestE2E_SignatureForgery(t *testing.T) {
	fx := newFixture(t)
	// We cannot just flip a byte in the signature hex without changing
	// the document hash. So we re-hash the tampered document and
	// re-anchor — the signature is still cryptographically wrong.
	tampered := mutateSignatureByte(t, fx.doc)

	freshHash, err := canonicalHashWithoutRoot(tampered)
	if err != nil {
		t.Fatalf("rehash: %v", err)
	}
	anchors := StaticAnchorResolver{}
	anchors.SetAnchor(fx.bundleID, fx.retrievalID, freshHash, 1)

	v := &Verifier{Keys: fx.keyResolver, Anchors: anchors}
	res, _ := v.Verify(tampered)
	if res.Valid {
		t.Fatal("expected invalid after sig tamper")
	}
	if res.FailureReason != "signature_invalid" {
		t.Errorf("expected signature_invalid, got %q", res.FailureReason)
	}
}

// TestE2E_SignerSubstitution: claim a different DID. The verifier
// resolves to a DIFFERENT pubkey, signature fails. Setup mirrors
// SignatureForgery — re-anchor to the tampered document so we test
// the signature path rather than the hash path.
func TestE2E_SignerSubstitution(t *testing.T) {
	fx := newFixture(t)

	// Generate an unknown signer's key, register it, but the document
	// still claims the original DIDs — no, we want the OPPOSITE: keep
	// the original signers' signatures, but rewrite the DID list to
	// point at a different (registered) pubkey. The verifier will
	// resolve to the wrong pubkey, aggregation will be wrong, BLS verify
	// fails.
	otherKP, err := GenerateKey()
	if err != nil {
		t.Fatalf("gen: %v", err)
	}
	otherDID := "did:lbvr:imposter"

	// Find one of the original DIDs and replace it with otherDID.
	origDID := pickFirstDID(t, fx.doc)
	tampered := []byte(strings.ReplaceAll(string(fx.doc), origDID, otherDID))

	freshHash, err := canonicalHashWithoutRoot(tampered)
	if err != nil {
		t.Fatalf("rehash: %v", err)
	}
	keys := StaticKeyResolver{}
	for did, pk := range fx.keyResolver {
		keys[did] = pk
	}
	keys[otherDID] = otherKP.PublicBytes
	anchors := StaticAnchorResolver{}
	anchors.SetAnchor(fx.bundleID, fx.retrievalID, freshHash, 1)

	v := &Verifier{Keys: keys, Anchors: anchors}
	res, _ := v.Verify(tampered)
	if res.Valid {
		t.Fatal("expected invalid after signer substitution")
	}
	if res.FailureReason != "signature_invalid" {
		t.Errorf("expected signature_invalid, got %q", res.FailureReason)
	}
}

// TestE2E_QuorumReduction: claim 1 signer with QuorumThreshold=2 →
// insufficient_quorum. Documents with len(signers) < threshold fail
// before signature verification even runs.
func TestE2E_QuorumReduction(t *testing.T) {
	fx := newFixture(t)
	tampered := mutateSignersToOne(t, fx.doc)

	freshHash, err := canonicalHashWithoutRoot(tampered)
	if err != nil {
		t.Fatalf("rehash: %v", err)
	}
	anchors := StaticAnchorResolver{}
	anchors.SetAnchor(fx.bundleID, fx.retrievalID, freshHash, 1)

	v := &Verifier{Keys: fx.keyResolver, Anchors: anchors}
	res, _ := v.Verify(tampered)
	if res.Valid {
		t.Fatal("expected invalid after quorum reduction")
	}
	if res.FailureReason != "insufficient_quorum" {
		t.Errorf("expected insufficient_quorum, got %q", res.FailureReason)
	}
}

// TestE2E_CanonicalizationEvasion: reorder JSON keys and re-anchor
// against the reordered document. JCS normalises both sides, so the
// hash matches and BLS verifies → document validates. Documents this
// behavior so reviewers know JCS is doing its job.
func TestE2E_CanonicalizationEvasion(t *testing.T) {
	fx := newFixture(t)
	// JCS will sort keys; reordering is a no-op semantically. Verify
	// the document still validates against the original anchor.
	v := &Verifier{Keys: fx.keyResolver, Anchors: fx.anchors}
	res, _ := v.Verify(fx.doc)
	if !res.Valid {
		t.Fatalf("baseline failed before evasion test: %s", res.FailureReason)
	}

	// Now hand-craft a re-ordered (but semantically identical) version
	// by round-tripping through encoding/json with a fresh map. Go's
	// json.Marshal sorts map keys alphabetically, which differs from
	// our generator's struct field order — yet JCS canonicalises both
	// to the same bytes.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(fx.doc, &raw); err != nil {
		t.Fatalf("parse: %v", err)
	}
	reordered, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	res2, _ := v.Verify(reordered)
	if !res2.Valid {
		t.Errorf("reordered doc failed: %s (JCS evasion succeeded — bug)", res2.FailureReason)
	}
}

// TestE2E_TimestampTampering: spec §7.3 case (6) — modify signed_at
// inside a sig block. signed_at sits inside the sig field, so the
// strip-and-rehash detects it as a sig-block tamper. Result: BLS
// verification fails (we rehash but the message that was signed
// included the old signed_at hash, not the new one).
//
// Wait — actually signed_at is inside the sig block which gets
// STRIPPED before hashing. So changing signed_at doesn't change the
// signed message. But the document hash DOES include the sig block,
// so the on-chain anchor fails first — hash_mismatch.
func TestE2E_TimestampTampering(t *testing.T) {
	fx := newFixture(t)
	// Find any signed_at field and bump the seconds.
	tampered := mutateField(t, fx.doc, `"signed_at":"2026-`, `"signed_at":"2099-`)
	v := &Verifier{Keys: fx.keyResolver, Anchors: fx.anchors}
	res, _ := v.Verify(tampered)
	if res.Valid {
		t.Fatal("expected invalid after signed_at tamper")
	}
	if res.FailureReason != "hash_mismatch" {
		t.Errorf("expected hash_mismatch, got %q", res.FailureReason)
	}
}

// TestE2E_MissingSig: a signed-required node with the sig field
// removed must fail with "missing_sig". We craft this case by
// stripping sig from the bundle entity post-anchor, then re-anchoring
// against the stripped doc to isolate the failure.
func TestE2E_MissingSig(t *testing.T) {
	fx := newFixture(t)

	// Strip the bundle entity's sig field.
	var doc Document
	if err := json.Unmarshal(fx.doc, &doc); err != nil {
		t.Fatalf("parse: %v", err)
	}
	bundleID, err := doc.findBundleEntityID()
	if err != nil {
		t.Fatalf("find bundle: %v", err)
	}
	stripped, err := StripSigField(doc.Entity[bundleID])
	if err != nil {
		t.Fatalf("strip: %v", err)
	}
	doc.Entity[bundleID] = stripped
	doc.ProvenanceRoot = nil
	bytesNoRoot, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	freshHash, err := CanonicalHash(bytesNoRoot)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	doc.ProvenanceRoot = &ProvenanceRoot{
		Algorithm: "SHA-256", Root: hexPrefixed(freshHash[:]),
		Canonicalization: CanonicalizationJCS,
	}
	tampered, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	anchors := StaticAnchorResolver{}
	anchors.SetAnchor(fx.bundleID, fx.retrievalID, freshHash, 1)

	v := &Verifier{Keys: fx.keyResolver, Anchors: anchors}
	res, _ := v.Verify(tampered)
	if res.Valid {
		t.Fatal("expected invalid when bundle sig is missing")
	}
	if res.FailureReason != "missing_sig" {
		t.Errorf("expected missing_sig, got %q", res.FailureReason)
	}
}

// TestE2E_OfflineMode: AllowOffline drops the anchor lookup. The
// verifier still runs cryptographic checks; the result is valid if the
// signatures pass even when the chain is unreachable. Useful for
// air-gapped review.
func TestE2E_OfflineMode(t *testing.T) {
	fx := newFixture(t)
	v := &Verifier{Keys: fx.keyResolver, AllowOffline: true}
	res, err := v.Verify(fx.doc)
	if err != nil {
		t.Fatalf("verify: %v (reason=%q)", err, res.FailureReason)
	}
	if !res.Valid {
		t.Fatalf("offline expected Valid=true, got %q", res.FailureReason)
	}
	if res.AnchoredBlock != 0 {
		t.Errorf("offline mode should not have AnchoredBlock, got %d", res.AnchoredBlock)
	}
}

// TestE2E_NoAnchor: chain-mode with the anchor missing → ErrNoAnchor.
func TestE2E_NoAnchor(t *testing.T) {
	fx := newFixture(t)
	emptyAnchors := StaticAnchorResolver{}
	v := &Verifier{Keys: fx.keyResolver, Anchors: emptyAnchors}
	res, err := v.Verify(fx.doc)
	if !errors.Is(err, ErrNoAnchor) {
		t.Fatalf("expected ErrNoAnchor, got %v", err)
	}
	if res.Valid {
		t.Fatal("expected invalid when no anchor")
	}
}

// canonicalHashWithoutRoot recomputes the verifier-style hash for a
// tampered document; tests use this when they intentionally re-anchor
// to isolate one failure mode.
func canonicalHashWithoutRoot(doc []byte) ([32]byte, error) {
	var d Document
	if err := json.Unmarshal(doc, &d); err != nil {
		return [32]byte{}, err
	}
	d.ProvenanceRoot = nil
	bs, err := json.Marshal(d)
	if err != nil {
		return [32]byte{}, err
	}
	return CanonicalHash(bs)
}

// mutateField replaces the first occurrence of the search string with
// the replacement. Fails the test if the search string is not found —
// catches stale tests after a generator-output change.
func mutateField(t *testing.T, doc []byte, search, replace string) []byte {
	t.Helper()
	if !strings.Contains(string(doc), search) {
		t.Fatalf("mutateField: %q not found in document", search)
	}
	return []byte(strings.Replace(string(doc), search, replace, 1))
}

// mutateSignatureByte flips one nibble in the first signature found.
// We pick a hex nibble that is guaranteed to flip the underlying byte
// to a different value.
func mutateSignatureByte(t *testing.T, doc []byte) []byte {
	t.Helper()
	s := string(doc)
	const marker = `"signature":"0x`
	idx := strings.Index(s, marker)
	if idx == -1 {
		t.Fatalf("no signature found")
	}
	// Position of the first hex nibble after "0x"
	pos := idx + len(marker)
	// Flip the FIRST byte of signature: '0' -> 'f' (or vice-versa).
	b := []byte(s)
	if b[pos] == 'f' || b[pos] == 'F' {
		b[pos] = '0'
	} else {
		b[pos] = 'f'
	}
	return b
}

// mutateSignersToOne rewrites the FIRST sig block's signers array
// down to a single DID, leaving QuorumThreshold=2 → triggers the
// insufficient-quorum check. We use a structural mutation rather than
// regex because the order of "signers" before the threshold can vary
// post-canonicalisation.
func mutateSignersToOne(t *testing.T, doc []byte) []byte {
	t.Helper()
	var d Document
	if err := json.Unmarshal(doc, &d); err != nil {
		t.Fatalf("parse: %v", err)
	}
	bundleID, err := d.findBundleEntityID()
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	var bundleNode map[string]json.RawMessage
	if err := json.Unmarshal(d.Entity[bundleID], &bundleNode); err != nil {
		t.Fatalf("parse bundle: %v", err)
	}
	var sigBlock SignatureBlock
	if err := json.Unmarshal(bundleNode["sig"], &sigBlock); err != nil {
		t.Fatalf("parse sig: %v", err)
	}
	if len(sigBlock.Signers) < 2 {
		t.Fatal("fixture should have 2 signers")
	}
	sigBlock.Signers = sigBlock.Signers[:1]
	sigJSON, err := json.Marshal(sigBlock)
	if err != nil {
		t.Fatalf("re-marshal sig: %v", err)
	}
	bundleNode["sig"] = sigJSON
	bundleJSON, err := json.Marshal(bundleNode)
	if err != nil {
		t.Fatalf("re-marshal bundle: %v", err)
	}
	d.Entity[bundleID] = bundleJSON
	out, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("re-marshal doc: %v", err)
	}
	return out
}

// pickFirstDID returns the first signer DID embedded in any sig block
// of the document. Used by TestE2E_SignerSubstitution to know which
// DID to rewrite.
func pickFirstDID(t *testing.T, doc []byte) string {
	t.Helper()
	var d Document
	if err := json.Unmarshal(doc, &d); err != nil {
		t.Fatalf("parse: %v", err)
	}
	bundleID, err := d.findBundleEntityID()
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	var bundleNode map[string]json.RawMessage
	if err := json.Unmarshal(d.Entity[bundleID], &bundleNode); err != nil {
		t.Fatalf("parse bundle: %v", err)
	}
	var sigBlock SignatureBlock
	if err := json.Unmarshal(bundleNode["sig"], &sigBlock); err != nil {
		t.Fatalf("parse sig: %v", err)
	}
	if len(sigBlock.Signers) == 0 {
		t.Fatal("no signers")
	}
	return sigBlock.Signers[0]
}
