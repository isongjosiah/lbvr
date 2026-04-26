package provenance

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// KeyResolver maps a gateway DID to its BLS12-381 G1 public key. The
// production resolver is chain-backed (AuditorLog.gatewayKeys); the
// offline CLI ships a StaticKeyResolver populated from a JSON file.
type KeyResolver interface {
	Resolve(did string) ([PublicKeySize]byte, error)
}

// StaticKeyResolver is a trivial in-memory implementation. The verifier
// CLI uses this when loaded from --keys-file; tests use it for the
// happy-path round trip.
type StaticKeyResolver map[string][PublicKeySize]byte

// Resolve returns the registered public key for did, or an error if
// the DID is unknown. The error message includes the DID so a verifier
// log can identify the missing signer.
func (r StaticKeyResolver) Resolve(did string) ([PublicKeySize]byte, error) {
	k, ok := r[did]
	if !ok {
		return [PublicKeySize]byte{}, fmt.Errorf("provenance: unknown signer %q", did)
	}
	return k, nil
}

// AnchorResolver returns the on-chain provenance anchor for a given
// (bundleID, retrievalID) pair. The chain client implementation calls
// AuditorLog.getProvenanceAnchor; tests use a static map.
type AnchorResolver interface {
	Resolve(bundleID, retrievalID [32]byte) (provHash [32]byte, blockNumber uint64, err error)
}

// StaticAnchorResolver is the in-memory test implementation. Key is
// hex(bundleID) + ":" + hex(retrievalID).
type StaticAnchorResolver map[string]struct {
	ProvHash    [32]byte
	BlockNumber uint64
}

// Resolve looks up the anchor; returns ErrNoAnchor when missing so the
// Verifier can report a stable failure reason.
func (r StaticAnchorResolver) Resolve(bundleID, retrievalID [32]byte) ([32]byte, uint64, error) {
	key := hex.EncodeToString(bundleID[:]) + ":" + hex.EncodeToString(retrievalID[:])
	v, ok := r[key]
	if !ok {
		return [32]byte{}, 0, ErrNoAnchor
	}
	return v.ProvHash, v.BlockNumber, nil
}

// SetAnchor is a convenience helper for tests.
func (r StaticAnchorResolver) SetAnchor(bundleID, retrievalID [32]byte, provHash [32]byte, blockNumber uint64) {
	key := hex.EncodeToString(bundleID[:]) + ":" + hex.EncodeToString(retrievalID[:])
	r[key] = struct {
		ProvHash    [32]byte
		BlockNumber uint64
	}{ProvHash: provHash, BlockNumber: blockNumber}
}

// Sentinel errors. Callers branch on these via errors.Is.
var (
	ErrNoAnchor          = errors.New("provenance: no on-chain anchor")
	ErrHashMismatch      = errors.New("provenance: hash mismatch")
	ErrUnknownSigner     = errors.New("provenance: unknown signer")
	ErrInsufficientQuora = errors.New("provenance: insufficient signers for quorum")
	ErrAlgorithmMismatch = errors.New("provenance: signature algorithm not supported")
	ErrMissingSig        = errors.New("provenance: signed-required node has no sig")
)

// Verifier is the entry point for verifying a finalised PROV document.
// Both Keys and Anchors are required; AllowOffline drops the anchor
// check (useful for testing the cryptographic chain in isolation, NOT
// recommended for production verifiers).
type Verifier struct {
	Keys         KeyResolver
	Anchors      AnchorResolver
	AllowOffline bool
}

// SignatureCheck is per-signed-node verification status. The verifier
// runs every signed node and includes results for all of them so a
// caller can report partial failures.
type SignatureCheck struct {
	NodeID          string
	NodeKind        string // "entity" | "activity" | "agent"
	AttestationType AttestationType
	Valid           bool
	Reason          string
}

// Result is the verifier's structured output. Valid is false on any
// failure; FailureReason captures the first hard failure (hash mismatch,
// missing anchor, signature failure, …) so a caller can dispatch on it.
type Result struct {
	Valid           bool
	BundleID        [32]byte
	RetrievalID     [32]byte
	ComputedHash    [32]byte
	AnchoredBlock   uint64
	FailureReason   string
	SignatureChecks []SignatureCheck
}

// Verify parses the PROV document, checks the on-chain anchor, then
// verifies every signed node. Returns a non-nil Result even on failure
// so the caller can report which step broke.
//
// Procedure (matches docs/provenance-spec.md §5.5 + §7.3):
//
//  1. Parse PROV-JSON.
//  2. Recompute the canonical hash of the document with
//     lbvr:provenanceRoot removed.
//  3. Look up on-chain anchor; compare hash. Mismatch → invalid.
//  4. For each signed node (bundle entity + retrieval activity):
//     a. Strip its sig field → JCS-canonicalise → message bytes.
//     b. Look up public keys for sigBlock.Signers via KeyResolver.
//     c. Aggregate the public keys.
//     d. Decode signature bytes from the hex string.
//     e. BLS-verify against the aggregated pubkey + message.
//     f. Check len(Signers) >= QuorumThreshold.
func (v *Verifier) Verify(provDoc []byte) (*Result, error) {
	if v == nil || v.Keys == nil {
		return nil, errors.New("provenance: verify: nil verifier or KeyResolver")
	}

	res := &Result{}

	// 1. Parse.
	var doc Document
	if err := json.Unmarshal(provDoc, &doc); err != nil {
		res.FailureReason = "parse_failed"
		return res, fmt.Errorf("provenance: parse: %w", err)
	}

	// Extract bundle + retrieval IDs from the node IDs (lbvr:bundle/<short>
	// / lbvr:retrieval/<short>). The full 32-byte ID would normally come
	// from the URL the verifier is given, but for offline mode we cannot
	// recover it from the short form alone — so the caller passes the IDs
	// out-of-band (Anchors.Resolve takes them). The Document doesn't carry
	// the full bundle/retrieval IDs in any field; the on-chain anchor key
	// is supplied by the caller via the AnchorResolver.
	//
	// To keep the offline test mode usable, we read the full IDs from
	// the lbvr:provenanceRoot.AnchorContract is NOT a place to put them;
	// we instead require the caller to pass them in via the static
	// anchor resolver keyed by hex(bundleID):hex(retrievalID). The
	// verifier surfaces only short IDs in Result; tests recover the full
	// IDs from their own state.
	bundleID, retrievalID, err := extractIDs(&doc)
	if err != nil {
		res.FailureReason = "id_extraction_failed"
		return res, err
	}
	copy(res.BundleID[:], bundleID[:])
	copy(res.RetrievalID[:], retrievalID[:])

	// 2 + 3. Recompute canonical hash with provenanceRoot removed,
	// compare against on-chain anchor.
	stripped := doc
	stripped.ProvenanceRoot = nil
	strippedJSON, err := json.Marshal(stripped)
	if err != nil {
		res.FailureReason = "remarshal_failed"
		return res, fmt.Errorf("provenance: re-marshal: %w", err)
	}
	hash, err := CanonicalHash(strippedJSON)
	if err != nil {
		res.FailureReason = "canonicalize_failed"
		return res, fmt.Errorf("provenance: canonicalize: %w", err)
	}
	res.ComputedHash = hash

	if !v.AllowOffline {
		if v.Anchors == nil {
			return res, errors.New("provenance: verify: nil AnchorResolver (set AllowOffline to bypass)")
		}
		anchorHash, blockNum, err := v.Anchors.Resolve(bundleID, retrievalID)
		if err != nil {
			res.FailureReason = "no_anchor"
			return res, fmt.Errorf("provenance: anchor lookup: %w", err)
		}
		if anchorHash != hash {
			res.FailureReason = "hash_mismatch"
			return res, ErrHashMismatch
		}
		res.AnchoredBlock = blockNum
	}

	// 4. Verify each signed node. For now: bundle entity (entity_integrity)
	// + retrieval activity (retrieval_quorum). Future versions may extend
	// to agent attestations; the verifier walks all signed nodes
	// uniformly so adding a new attestation is purely a generator-side
	// change.
	if err := v.verifySignedNodes(&doc, res); err != nil {
		// Failure already recorded in res. Return nil err for
		// "verifier completed; document invalid" — surface real
		// errors only for parse / IO problems above.
		if res.FailureReason == "" {
			res.FailureReason = err.Error()
		}
		return res, nil
	}

	res.Valid = true
	return res, nil
}

// verifySignedNodes iterates over every node that should carry a sig
// and validates it. Failures append to res.SignatureChecks; the first
// failure also sets res.FailureReason.
func (v *Verifier) verifySignedNodes(doc *Document, res *Result) error {
	bundleID, err := doc.findBundleEntityID()
	if err != nil {
		res.FailureReason = "no_bundle_entity"
		return err
	}
	retrievalID, err := doc.findRetrievalActivityID()
	if err != nil {
		res.FailureReason = "no_retrieval_activity"
		return err
	}

	var firstFail string

	check := v.verifyNode(doc.Entity, bundleID, "entity", AttestationEntityIntegrity)
	res.SignatureChecks = append(res.SignatureChecks, check)
	if !check.Valid && firstFail == "" {
		firstFail = check.Reason
	}

	check = v.verifyNode(doc.Activity, retrievalID, "activity", AttestationRetrievalQuorum)
	res.SignatureChecks = append(res.SignatureChecks, check)
	if !check.Valid && firstFail == "" {
		firstFail = check.Reason
	}

	if firstFail != "" {
		res.FailureReason = firstFail
		return errors.New(firstFail)
	}
	return nil
}

// verifyNode runs the cryptographic checks on a single signed node and
// returns a SignatureCheck with the outcome. Never panics on malformed
// input — every error path resolves to (Valid: false, Reason: ...).
func (v *Verifier) verifyNode(
	store map[string]json.RawMessage,
	nodeID string,
	nodeKind string,
	expectedAttest AttestationType,
) SignatureCheck {
	check := SignatureCheck{NodeID: nodeID, NodeKind: nodeKind, AttestationType: expectedAttest}

	raw, ok := store[nodeID]
	if !ok {
		check.Reason = "node_not_found"
		return check
	}

	// Extract the sig block. A signed-required node without a sig is a
	// hard failure (ErrMissingSig); the verifier doesn't silently accept
	// "no sig means no claim."
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		check.Reason = "node_parse_failed"
		return check
	}
	sigRaw, ok := probe["sig"]
	if !ok {
		check.Reason = "missing_sig"
		return check
	}
	var sigBlock SignatureBlock
	if err := json.Unmarshal(sigRaw, &sigBlock); err != nil {
		check.Reason = "sig_parse_failed"
		return check
	}
	check.AttestationType = sigBlock.AttestationType

	if sigBlock.Algorithm != AlgorithmBLS12381G2 {
		check.Reason = "algorithm_mismatch"
		return check
	}
	if len(sigBlock.Signers) < sigBlock.QuorumThreshold {
		check.Reason = "insufficient_quorum"
		return check
	}

	// Compute the message: strip sig field, canonicalise, sign that.
	stripped, err := StripSigField(raw)
	if err != nil {
		check.Reason = "strip_failed"
		return check
	}
	canonical, err := Canonicalize(stripped)
	if err != nil {
		check.Reason = "canonicalize_failed"
		return check
	}

	// Resolve and aggregate public keys.
	pubKeys := make([][PublicKeySize]byte, 0, len(sigBlock.Signers))
	for _, did := range sigBlock.Signers {
		pk, err := v.Keys.Resolve(did)
		if err != nil {
			check.Reason = "unknown_signer"
			return check
		}
		pubKeys = append(pubKeys, pk)
	}
	aggPub, err := AggregatePublicKeys(pubKeys)
	if err != nil {
		check.Reason = "aggregate_pub_failed"
		return check
	}

	// Decode signature.
	sigBytes, err := decodeHexPrefixed(sigBlock.Signature)
	if err != nil {
		check.Reason = "signature_invalid"
		return check
	}
	if len(sigBytes) != SignatureSize {
		check.Reason = "signature_invalid"
		return check
	}
	var sig [SignatureSize]byte
	copy(sig[:], sigBytes)

	// BLS-verify.
	if err := Verify(aggPub, canonical, sig); err != nil {
		check.Reason = "signature_invalid"
		return check
	}

	check.Valid = true
	return check
}

// extractIDs recovers the full 32-byte bundle and retrieval IDs from a
// document. The PROV node IDs only encode the first 4 bytes, so strict
// recovery is impossible. We therefore expect the caller to pass the
// full IDs out-of-band via AnchorResolver / Verifier state. This function
// returns the first 4 bytes (left-padded with zeros to 32) so the
// AnchorResolver key is a deterministic function of the document.
//
// For offline tests using StaticAnchorResolver this is sufficient because
// the test populates the resolver with the same short-form keys. For
// chain-backed verifiers, the calling CLI must derive the full IDs from
// the URL it was invoked with and pass them via AnchorResolver.
func extractIDs(doc *Document) ([32]byte, [32]byte, error) {
	var bundleID, retrievalID [32]byte

	for id := range doc.Entity {
		const prefix = "lbvr:bundle/"
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		short := id[len(prefix):]
		if len(short) < 8 {
			return bundleID, retrievalID, fmt.Errorf("bundle id %q too short", id)
		}
		raw, err := hex.DecodeString(short[:8])
		if err != nil {
			return bundleID, retrievalID, fmt.Errorf("bundle id %q not hex: %w", id, err)
		}
		copy(bundleID[:], raw)
		break
	}
	for id := range doc.Activity {
		const prefix = "lbvr:retrieval/"
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		short := id[len(prefix):]
		if len(short) < 8 {
			return bundleID, retrievalID, fmt.Errorf("retrieval id %q too short", id)
		}
		raw, err := hex.DecodeString(short[:8])
		if err != nil {
			return bundleID, retrievalID, fmt.Errorf("retrieval id %q not hex: %w", id, err)
		}
		copy(retrievalID[:], raw)
		break
	}
	return bundleID, retrievalID, nil
}

// decodeHexPrefixed parses "0x..." hex strings. Strict: requires the
// 0x prefix and an even number of hex digits.
func decodeHexPrefixed(s string) ([]byte, error) {
	if !strings.HasPrefix(s, "0x") {
		return nil, fmt.Errorf("hex: missing 0x prefix")
	}
	return hex.DecodeString(s[2:])
}
