package provenance

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

// shardRoles maps the spec's "D0"/"D1"/"P0" labels to their fixed
// position in the on-chain ShardPlacement[3] array. Index 0 = hot,
// 1 = warm, 2 = cold (CLAUDE.md §4.5 placement policy).
var shardRoles = [3]string{"D0", "D1", "P0"}

// shardRoleToIndex inverts shardRoles for lookups.
func shardRoleToIndex(role string) (int, bool) {
	for i, r := range shardRoles {
		if r == role {
			return i, true
		}
	}
	return -1, false
}

// GenerateInput is everything the gateway has at retrieval-completion
// time. The split between "what the bundle is" and "what this retrieval
// did" mirrors the PROV Entity / Activity distinction.
type GenerateInput struct {
	BundleID         [32]byte
	MerkleRoot       [32]byte
	ShardLayout      [3]ShardPlacement
	BundleSizeBytes  int64
	FHIRResourceType string

	RetrievalID  [32]byte
	StartedAt    time.Time
	EndedAt      time.Time
	RecoveryMode string
	ShardsUsed   []string
	RSDecode     bool
	LatencyMs    int64

	Requester       RequesterAgent
	Gateways        []GatewayAgent
	QuorumThreshold int
}

// Generate builds an unsigned PROV-JSON document from a retrieval event.
// The split from Sign / SetRoot lets the gateway:
//
//  1. Build the doc (synchronous, fast)
//  2. Run quorum signing (per-node BLS calls; possibly remote)
//  3. Anchor on-chain (txn submit, async)
//
// Each step is independently testable. After Generate the document has
// no Sig blocks and no ProvenanceRoot — the verifier rejects such a
// document, so Generate's output is intermediate, not shippable.
//
// Generate emits exactly the shard entities listed in ShardsUsed (not
// all 3). The provenance graph reflects what the activity *consumed*,
// not the entire shard inventory — a verifier reading "used: D0, D1"
// learns the fast path was taken; "used: D1, P0" learns D0 was missing
// and reconstruction occurred.
func Generate(in GenerateInput) (*Document, error) {
	if err := validateGenerateInput(in); err != nil {
		return nil, err
	}

	bundleNodeID := bundleNodeID(in.BundleID)
	retrievalNodeID := retrievalNodeID(in.RetrievalID)
	requesterNodeID := requesterNodeID(in.Requester)

	doc := &Document{
		Context:  "https://www.w3.org/ns/prov",
		Prefix:   DefaultPrefixes(),
		Entity:   map[string]json.RawMessage{},
		Activity: map[string]json.RawMessage{},
		Agent:    map[string]json.RawMessage{},
	}

	// Bundle entity. ShardLayout maps role → placement; only the three
	// canonical roles ever appear. Map keys aren't ordered in Go, but
	// JCS sorts them on canonicalisation.
	resourceType := in.FHIRResourceType
	if resourceType == "" {
		resourceType = "Bundle"
	}
	bundle := BundleEntity{
		ProvType:     "lbvr:FHIRBundle",
		ResourceType: resourceType,
		MerkleRoot:   hexPrefixed(in.MerkleRoot[:]),
		SizeBytes:    in.BundleSizeBytes,
		ShardLayout: map[string]ShardPlacement{
			"D0": in.ShardLayout[0],
			"D1": in.ShardLayout[1],
			"P0": in.ShardLayout[2],
		},
	}
	bundleJSON, err := json.Marshal(bundle)
	if err != nil {
		return nil, fmt.Errorf("provenance: marshal bundle: %w", err)
	}
	doc.Entity[bundleNodeID] = bundleJSON

	// Shard entities — only those actually used by this retrieval. A
	// fast-path retrieval emits two shards (D0+D1); a slow-path may
	// emit any 2 of {D0,D1,P0}; an all-three (e.g. cross-check) would
	// emit all three. Order is preserved for the wasDerivedFrom edges.
	usedShards := dedupShardRoles(in.ShardsUsed)
	for _, role := range usedShards {
		idx, ok := shardRoleToIndex(role)
		if !ok {
			return nil, fmt.Errorf("provenance: unknown shard role %q", role)
		}
		placement := in.ShardLayout[idx]
		shardEntity := ShardEntity{
			ProvType:   "lbvr:ErasureShard",
			ShardIndex: idx,
			Role:       shardRoleType(role),
			Tier:       placement.Tier,
			CID:        placement.CID,
			SizeBytes:  shardSizeBytes(in.BundleSizeBytes),
		}
		raw, err := json.Marshal(shardEntity)
		if err != nil {
			return nil, fmt.Errorf("provenance: marshal shard %s: %w", role, err)
		}
		doc.Entity[shardNodeID(in.BundleID, role)] = raw
	}

	// Retrieval activity.
	activity := RetrievalActivity{
		ProvType:        "lbvr:VerifiedRetrieval",
		StartedAtTime:   formatISO8601(in.StartedAt),
		EndedAtTime:     formatISO8601(in.EndedAt),
		RecoveryMode:    in.RecoveryMode,
		ShardsUsed:      usedShards,
		RSDecode:        in.RSDecode,
		LatencyMs:       in.LatencyMs,
		QuorumSize:      len(in.Gateways),
		QuorumThreshold: in.QuorumThreshold,
	}
	activityJSON, err := json.Marshal(activity)
	if err != nil {
		return nil, fmt.Errorf("provenance: marshal activity: %w", err)
	}
	doc.Activity[retrievalNodeID] = activityJSON

	// Agents: requester + each gateway node.
	requesterJSON, err := json.Marshal(in.Requester)
	if err != nil {
		return nil, fmt.Errorf("provenance: marshal requester: %w", err)
	}
	doc.Agent[requesterNodeID] = requesterJSON

	for _, gw := range in.Gateways {
		gwJSON, err := json.Marshal(gw)
		if err != nil {
			return nil, fmt.Errorf("provenance: marshal gateway: %w", err)
		}
		doc.Agent[gatewayNodeID(gw)] = gwJSON
	}

	// Relations.
	doc.WasGeneratedBy = map[string]map[string]string{
		"_:gen1": {
			"prov:entity":   bundleNodeID,
			"prov:activity": retrievalNodeID,
		},
	}

	// wasAssociatedWith: requester is the principal initiator; each
	// gateway gets its own association so a downstream tool can list
	// all parties bound to the retrieval.
	doc.WasAssociatedWith = map[string]map[string]string{
		"_:assoc-requester": {
			"prov:activity": retrievalNodeID,
			"prov:agent":    requesterNodeID,
		},
	}
	for i, gw := range in.Gateways {
		key := fmt.Sprintf("_:assoc-gw-%d", i)
		doc.WasAssociatedWith[key] = map[string]string{
			"prov:activity": retrievalNodeID,
			"prov:agent":    gatewayNodeID(gw),
		}
	}

	// used + wasDerivedFrom mirror each consumed shard. Both sets must
	// be present: `used` is Activity → Entity (the retrieval used the
	// shard); `wasDerivedFrom` is Entity → Entity (the bundle was
	// derived from the shard). This dual encoding is required by W3C
	// PROV-DM §5.5 — `used` alone leaves the derivation chain
	// unstated, and `wasDerivedFrom` alone hides which activity did it.
	doc.Used = map[string]map[string]string{}
	doc.WasDerivedFrom = map[string]map[string]string{}
	for i, role := range usedShards {
		shardID := shardNodeID(in.BundleID, role)
		doc.Used[fmt.Sprintf("_:use-%d", i)] = map[string]string{
			"prov:activity": retrievalNodeID,
			"prov:entity":   shardID,
		}
		doc.WasDerivedFrom[fmt.Sprintf("_:deriv-%d", i)] = map[string]string{
			"prov:generatedEntity": bundleNodeID,
			"prov:usedEntity":      shardID,
		}
	}

	return doc, nil
}

// Sign attaches signature blocks to the bundle entity (entity_integrity)
// and the retrieval activity (retrieval_quorum). Each signature is the
// aggregate of len(privateKeys) BLS signatures over the JCS-canonical
// hash of the node-without-its-sig-field.
//
// privateKeys and signerDIDs must be parallel slices of equal length;
// the i-th key signs as the i-th DID. quorumThreshold is recorded inside
// the SignatureBlock so the verifier can enforce "at least k signed"
// without a separate config lookup.
//
// Sign mutates the document in place. It is NOT safe to call concurrently
// on the same document; the gateway's signing coordinator is the sole
// invoker.
func (d *Document) Sign(privateKeys [][PrivateKeySize]byte, signerDIDs []string, quorumThreshold int) error {
	if d == nil {
		return errors.New("provenance: sign: nil document")
	}
	if len(privateKeys) != len(signerDIDs) {
		return fmt.Errorf("provenance: sign: %d keys vs %d DIDs", len(privateKeys), len(signerDIDs))
	}
	if len(privateKeys) == 0 {
		return errors.New("provenance: sign: no signers")
	}
	if quorumThreshold < 1 || quorumThreshold > len(privateKeys) {
		return fmt.Errorf("provenance: sign: quorum threshold %d invalid for %d signers", quorumThreshold, len(privateKeys))
	}

	now := formatISO8601(time.Now().UTC())

	// Sign every entity that needs entity_integrity. For the conference
	// scope this is the bundle entity only; shard entities carry no
	// signature because their integrity is implied by the bundle's
	// merkleRoot field (each shard root is bound there).
	bundleID, err := d.findBundleEntityID()
	if err != nil {
		return err
	}
	if err := d.signNode(d.Entity, bundleID, privateKeys, signerDIDs, quorumThreshold, AttestationEntityIntegrity, now); err != nil {
		return fmt.Errorf("provenance: sign bundle: %w", err)
	}

	// Sign the retrieval activity (retrieval_quorum).
	retrievalID, err := d.findRetrievalActivityID()
	if err != nil {
		return err
	}
	if err := d.signNode(d.Activity, retrievalID, privateKeys, signerDIDs, quorumThreshold, AttestationRetrievalQuorum, now); err != nil {
		return fmt.Errorf("provenance: sign retrieval: %w", err)
	}

	return nil
}

// signNode strips the existing sig field (if any), canonicalises the
// node, hashes it, signs with each private key, aggregates, and writes
// the resulting SignatureBlock back into the node. The intermediate
// hash is over the JCS-canonical bytes — which is what the verifier
// will reproduce.
func (d *Document) signNode(
	store map[string]json.RawMessage,
	nodeID string,
	privateKeys [][PrivateKeySize]byte,
	signerDIDs []string,
	quorumThreshold int,
	attest AttestationType,
	signedAt string,
) error {
	raw, ok := store[nodeID]
	if !ok {
		return fmt.Errorf("node %q not found", nodeID)
	}

	stripped, err := StripSigField(raw)
	if err != nil {
		return err
	}
	canonical, err := Canonicalize(stripped)
	if err != nil {
		return err
	}
	msg := canonical // BLS hash-to-curve consumes the message bytes directly

	individualSigs := make([][SignatureSize]byte, len(privateKeys))
	for i, sk := range privateKeys {
		sig, err := Sign(sk, msg)
		if err != nil {
			return fmt.Errorf("sign by %d: %w", i, err)
		}
		individualSigs[i] = sig
	}
	aggSig, err := AggregateSignatures(individualSigs)
	if err != nil {
		return err
	}

	sigBlock := SignatureBlock{
		Algorithm:       AlgorithmBLS12381G2,
		Signers:         append([]string(nil), signerDIDs...),
		QuorumThreshold: quorumThreshold,
		Signature:       hexPrefixed(aggSig[:]),
		SignedAt:        signedAt,
		AttestationType: attest,
	}

	// Re-merge: parse the stripped node, attach sig, re-marshal. We
	// cannot string-splice because the original ordering is non-canonical
	// and we'd risk producing invalid JSON.
	var nodeMap map[string]json.RawMessage
	if err := json.Unmarshal(stripped, &nodeMap); err != nil {
		return fmt.Errorf("re-parse stripped node: %w", err)
	}
	sigJSON, err := json.Marshal(sigBlock)
	if err != nil {
		return fmt.Errorf("marshal sig: %w", err)
	}
	nodeMap["sig"] = sigJSON
	merged, err := json.Marshal(nodeMap)
	if err != nil {
		return fmt.Errorf("merge sig: %w", err)
	}
	store[nodeID] = merged
	return nil
}

// SetRoot finalises the document with the JCS canonical-hash root and
// optional on-chain anchor info. After this call the document is
// verifier-ready (Marshal will produce the bytes a verifier expects).
//
// Anchor may be nil if the gateway has not yet submitted the on-chain
// transaction; the verifier will then accept the document only in offline
// mode and report AnchoredOnChain=false. The caller can call SetRoot
// again once the txn is mined to backfill the chain fields.
func (d *Document) SetRoot(anchor *ProvenanceRoot) error {
	if d == nil {
		return errors.New("provenance: set root: nil document")
	}

	// Compute the canonical hash with the existing ProvenanceRoot
	// removed (the root field is metadata about the hash; including
	// it in the hash would be circular).
	d.ProvenanceRoot = nil
	docBytes, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("provenance: set root: marshal: %w", err)
	}
	hash, err := CanonicalHash(docBytes)
	if err != nil {
		return fmt.Errorf("provenance: set root: hash: %w", err)
	}

	root := &ProvenanceRoot{
		Algorithm:        "SHA-256",
		Root:             hexPrefixed(hash[:]),
		Canonicalization: CanonicalizationJCS,
		AnchoredOnChain:  false,
	}
	if anchor != nil {
		root.AnchoredOnChain = anchor.AnchoredOnChain
		root.ChainTxHash = anchor.ChainTxHash
		root.BlockNumber = anchor.BlockNumber
		root.AnchorContract = anchor.AnchorContract
		root.AnchoredAt = anchor.AnchoredAt
	}
	d.ProvenanceRoot = root
	return nil
}

// Marshal returns the JCS-canonical JSON bytes of the finalised document.
// This is what the gateway pins to IPFS and what the verifier consumes.
// Calling Marshal on a document that has not been finalised by SetRoot
// is allowed (useful for tests) but the resulting bytes will not have a
// valid on-chain anchor.
func (d *Document) Marshal() ([]byte, error) {
	raw, err := json.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("provenance: marshal: %w", err)
	}
	return Canonicalize(raw)
}

// findBundleEntityID returns the single Entity ID whose prov:type ==
// "lbvr:FHIRBundle". Returns an error if there is zero or more than one
// such entity (a well-formed document has exactly one).
func (d *Document) findBundleEntityID() (string, error) {
	return d.findNodeByProvType(d.Entity, "lbvr:FHIRBundle")
}

// findRetrievalActivityID returns the single Activity ID whose prov:type
// == "lbvr:VerifiedRetrieval".
func (d *Document) findRetrievalActivityID() (string, error) {
	return d.findNodeByProvType(d.Activity, "lbvr:VerifiedRetrieval")
}

func (d *Document) findNodeByProvType(store map[string]json.RawMessage, wantType string) (string, error) {
	var match string
	for id, raw := range store {
		var probe struct {
			Type string `json:"prov:type"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			continue
		}
		if probe.Type == wantType {
			if match != "" {
				return "", fmt.Errorf("multiple %s nodes", wantType)
			}
			match = id
		}
	}
	if match == "" {
		return "", fmt.Errorf("no %s node", wantType)
	}
	return match, nil
}

// validateGenerateInput catches the easily-detected misuses before the
// caller pays the marshal cost.
func validateGenerateInput(in GenerateInput) error {
	if in.EndedAt.Before(in.StartedAt) {
		return fmt.Errorf("provenance: endedAt before startedAt")
	}
	if in.LatencyMs < 0 {
		return fmt.Errorf("provenance: negative latency")
	}
	if len(in.Gateways) == 0 {
		return fmt.Errorf("provenance: no gateways supplied")
	}
	if in.QuorumThreshold < 1 || in.QuorumThreshold > len(in.Gateways) {
		return fmt.Errorf("provenance: quorum threshold %d invalid for %d gateways", in.QuorumThreshold, len(in.Gateways))
	}
	if len(in.ShardsUsed) < 2 {
		// A retrieval that consumed fewer than 2 shards cannot have
		// reconstructed the bundle; the gateway should have errored.
		return fmt.Errorf("provenance: at least 2 shards must be used (got %d)", len(in.ShardsUsed))
	}
	for _, role := range in.ShardsUsed {
		if _, ok := shardRoleToIndex(role); !ok {
			return fmt.Errorf("provenance: unknown shard role %q (want D0|D1|P0)", role)
		}
	}
	return nil
}

// dedupShardRoles preserves first-occurrence order while removing
// duplicates. Defensive against a caller that passes ["D0","D0","D1"].
func dedupShardRoles(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, r := range in {
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}
	// Stable order so the resulting JSON is byte-identical across
	// runs with the same logical input. JCS would re-sort anyway,
	// but consistent pre-canonicalisation order makes diffs readable.
	sort.Strings(out)
	return out
}

// shardRoleType returns "data" for D0/D1 and "parity" for P0.
func shardRoleType(role string) string {
	if role == "P0" {
		return "parity"
	}
	return "data"
}

// shardSizeBytes is the size of one RS(2,3) shard for a bundle of
// size n. RS(2,3) splits the bundle into 2 data shards (so each is
// half the bundle size, rounded up). We do not need byte-exact
// fidelity here — the value lives in PROV for human inspection only;
// the on-chain registry stores the authoritative shard layout.
func shardSizeBytes(bundleSize int64) int64 {
	if bundleSize <= 0 {
		return 0
	}
	return (bundleSize + 1) / 2
}

// hexPrefixed renders b as "0x" + lowercase hex. The 0x prefix is the
// canonical form across CLAUDE.md (e.g. §4.6 example shows
// "0x7a3f...e921") and matches what Solidity bytes32 fields render as.
func hexPrefixed(b []byte) string {
	out := make([]byte, 2+hex.EncodedLen(len(b)))
	out[0] = '0'
	out[1] = 'x'
	hex.Encode(out[2:], b)
	return string(out)
}

// formatISO8601 returns t in millisecond-precision ISO 8601 with a
// trailing 'Z'. Matches the spec's prov:startedAtTime examples like
// "2026-04-30T14:32:15.231Z".
func formatISO8601(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// bundleNodeID derives the entity ID from the bundleID. Uses the first
// 8 hex chars of the bundleID as a short identifier — enough to
// uniquely identify within a single PROV document, not globally unique.
func bundleNodeID(bundleID [32]byte) string {
	return "lbvr:bundle/" + shortID(bundleID)
}

// retrievalNodeID derives the activity ID similarly.
func retrievalNodeID(retrievalID [32]byte) string {
	return "lbvr:retrieval/" + shortID(retrievalID)
}

// shardNodeID encodes (bundle, role) so multiple bundles in one
// document (future use) wouldn't collide on shard names.
func shardNodeID(bundleID [32]byte, role string) string {
	return "lbvr:shard/" + role + "-" + shortID(bundleID)
}

// requesterNodeID derives a stable agent ID from the requester role +
// institution. We do not include personally identifiable information
// (e.g. clinician name) in the URI; downstream PROV consumers should
// resolve identity through the institution's AAI.
func requesterNodeID(r RequesterAgent) string {
	return "lbvr:requester/" + safeIDFragment(r.Role)
}

// gatewayNodeID derives the gateway agent ID from its public key —
// stable across restarts because the public key is registered on-chain.
func gatewayNodeID(gw GatewayAgent) string {
	id := gw.PublicKey
	if len(id) >= 10 {
		id = id[2:10] // strip "0x" + take next 8 hex chars
	}
	return "lbvr:gateway/gw-" + id
}

// shortID returns the first 8 hex chars of a 32-byte ID.
func shortID(id [32]byte) string {
	const n = 4
	out := make([]byte, hex.EncodedLen(n))
	hex.Encode(out, id[:n])
	return string(out)
}

// safeIDFragment trims to a URI-safe slug. Keeps alphanumerics, dashes,
// underscores; collapses everything else to '-'. Defensive against role
// strings like "clinician #42" leaking control bytes into a URI.
func safeIDFragment(s string) string {
	if s == "" {
		return "anon"
	}
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z',
			c >= 'A' && c <= 'Z',
			c >= '0' && c <= '9',
			c == '-', c == '_':
			out = append(out, c)
		default:
			out = append(out, '-')
		}
	}
	return string(out)
}
