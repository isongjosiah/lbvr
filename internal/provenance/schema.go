// Package provenance implements the LBVR-Med cryptographic PROV-JSON
// extension (CLAUDE.md §4.6 and docs/provenance-spec.md). Every retrieval
// produces a tamper-evident, machine-verifiable PROV-JSON document signed
// by a 2-of-N quorum of gateway nodes; the SHA-256 of its JCS-canonical
// encoding is anchored on-chain.
//
// The package splits into five concerns, each in its own file:
//
//   - schema.go       — Go types matching the JSON schema in §3 of the spec
//   - canonicalize.go — JCS (RFC 8785) wrapper + sig-stripping helper
//   - signer.go       — BLS12-381-G2 sign / aggregate / verify
//   - generator.go    — assembles a Document from a retrieval event,
//     attaches signatures, finalises the root anchor
//   - verifier.go     — parses + canonicalises + verifies a document
//
// The Entity / Activity / Agent maps in Document hold json.RawMessage so
// the verifier can strip a node's `sig` field and re-canonicalise without
// round-tripping through Go structs (which loses field ordering anyway —
// but JCS handles that). It also lets one map hold heterogeneous types
// (BundleEntity vs ShardEntity) without an interface.
package provenance

import "encoding/json"

// AlgorithmBLS12381G2 is the only signing algorithm in this version. The
// journal extension may add a PQ alternative; until then the verifier
// rejects any other value to fail loudly on a future protocol mismatch.
const AlgorithmBLS12381G2 = "BLS12-381-G2"

// CanonicalizationJCS names the canonicalisation scheme used to compute
// the on-chain anchor hash. Stored in lbvr:provenanceRoot so off-chain
// verifiers know which library to invoke.
const CanonicalizationJCS = "JCS-RFC8785"

// AttestationType enumerates what a signature attests to (spec §3.3).
// Stored as a string in the JSON so future additions are forward-compatible
// at the wire level even when older verifiers have not been updated.
type AttestationType string

const (
	AttestationEntityIntegrity    AttestationType = "entity_integrity"
	AttestationRetrievalQuorum    AttestationType = "retrieval_quorum"
	AttestationAgentAuthorization AttestationType = "agent_authorization"
)

// SignatureBlock is the lbvr:sig extension on Entity/Activity/Agent nodes.
// Field names match the canonical JSON keys (lowercase + snake_case) so
// json.Marshal output is JCS-friendly without needing a custom marshaller.
type SignatureBlock struct {
	Algorithm       string          `json:"algorithm"`
	Signers         []string        `json:"signers"`
	QuorumThreshold int             `json:"quorum_threshold"`
	Signature       string          `json:"signature"`
	SignedAt        string          `json:"signed_at"`
	AttestationType AttestationType `json:"attestation_type"`
}

// ShardPlacement mirrors registry.ShardPlacement plus the per-shard
// Merkle root from CLAUDE.md §4.5 (last bullet under "Alignment with
// Merkle tree"). The provenance package keeps its own copy rather than
// importing internal/registry to keep the dependency graph cycle-free.
type ShardPlacement struct {
	CID       string `json:"cid"`
	Tier      string `json:"tier"`
	ShardRoot string `json:"shardRoot"`
}

// BundleEntity is the lbvr:bundle/<id> entity payload. Order of fields
// here is preserved by encoding/json (Go follows struct order), but JCS
// re-sorts on canonicalisation, so the wire form is independent of how
// we declare the struct.
type BundleEntity struct {
	ProvType     string                    `json:"prov:type"`
	ResourceType string                    `json:"fhir:resourceType,omitempty"`
	MerkleRoot   string                    `json:"lbvr:merkleRoot"`
	ShardLayout  map[string]ShardPlacement `json:"lbvr:shardLayout"`
	SizeBytes    int64                     `json:"lbvr:sizeBytes"`
	Sig          *SignatureBlock           `json:"sig,omitempty"`
}

// ShardEntity is one of the lbvr:shard/<role-id> entities included only
// when the activity actually consumed it (see Generate's ShardsUsed).
type ShardEntity struct {
	ProvType   string `json:"prov:type"`
	ShardIndex int    `json:"lbvr:shardIndex"`
	Role       string `json:"lbvr:role"`
	Tier       string `json:"lbvr:tier"`
	CID        string `json:"lbvr:cid"`
	SizeBytes  int64  `json:"lbvr:sizeBytes"`
}

// RetrievalActivity is the lbvr:retrieval/<id> activity payload. Fields
// mirror the spec §4 example. RSDecode is a separate boolean from
// RecoveryMode so a verifier can answer "was the parity shard required?"
// without parsing free-form mode strings.
type RetrievalActivity struct {
	ProvType        string          `json:"prov:type"`
	StartedAtTime   string          `json:"prov:startedAtTime"`
	EndedAtTime     string          `json:"prov:endedAtTime"`
	RecoveryMode    string          `json:"lbvr:recoveryMode"`
	ShardsUsed      []string        `json:"lbvr:shardsUsed"`
	RSDecode        bool            `json:"lbvr:rsDecode"`
	LatencyMs       int64           `json:"lbvr:latencyMs"`
	QuorumSize      int             `json:"lbvr:quorumSize"`
	QuorumThreshold int             `json:"lbvr:quorumThreshold"`
	Sig             *SignatureBlock `json:"sig,omitempty"`
}

// RequesterAgent is the human (or system principal) initiating retrieval.
// AuthzPolicy records which AAI policy permitted the access — relevant
// for EU AI Act Art. 12 audit replay.
type RequesterAgent struct {
	ProvType    string `json:"prov:type"`
	Role        string `json:"lbvr:role"`
	Institution string `json:"lbvr:institution"`
	AuthzPolicy string `json:"lbvr:authzPolicy"`
}

// GatewayAgent is one of the gateway nodes participating in the quorum.
// PublicKey is recorded inline so a one-shot offline verifier can sanity-
// check the DID → key binding without reaching the chain.
type GatewayAgent struct {
	ProvType  string `json:"prov:type"`
	Role      string `json:"lbvr:role"`
	Version   string `json:"lbvr:version"`
	PublicKey string `json:"lbvr:publicKey"`
}

// ProvenanceRoot is the top-level lbvr:provenanceRoot block. Computed by
// SetRoot after Sign — the document must be stable before its own hash
// can be embedded.
type ProvenanceRoot struct {
	Algorithm        string `json:"algorithm"`
	Root             string `json:"root"`
	Canonicalization string `json:"canonicalization"`
	AnchoredOnChain  bool   `json:"anchored_on_chain"`
	ChainTxHash      string `json:"chain_txhash,omitempty"`
	BlockNumber      uint64 `json:"block_number,omitempty"`
	AnchorContract   string `json:"anchor_contract,omitempty"`
	AnchoredAt       string `json:"anchored_at,omitempty"`
}

// Document is the full PROV-JSON document with LBVR extensions.
//
// Entity / Activity / Agent are kept as RawMessage maps so the verifier
// can JCS-canonicalise an individual node, strip its sig, and rehash
// without going through Go's json.Marshal again (which would re-sort
// struct fields and force a second JCS pass to recover stability).
//
// The relation maps (wasGeneratedBy, wasAssociatedWith, used,
// wasDerivedFrom) carry their PROV-JSON form: the outer key is a
// blank node id (e.g. "_:gen1"), the inner map is the relation's own
// key/value pairs (e.g. {"prov:entity": "...", "prov:activity": "..."}).
type Document struct {
	Context           string                       `json:"@context"`
	Prefix            map[string]string            `json:"prefix"`
	Entity            map[string]json.RawMessage   `json:"entity"`
	Activity          map[string]json.RawMessage   `json:"activity"`
	Agent             map[string]json.RawMessage   `json:"agent"`
	WasGeneratedBy    map[string]map[string]string `json:"wasGeneratedBy,omitempty"`
	WasAssociatedWith map[string]map[string]string `json:"wasAssociatedWith,omitempty"`
	Used              map[string]map[string]string `json:"used,omitempty"`
	WasDerivedFrom    map[string]map[string]string `json:"wasDerivedFrom,omitempty"`
	ProvenanceRoot    *ProvenanceRoot              `json:"lbvr:provenanceRoot,omitempty"`
}

// DefaultPrefixes are the four namespace bindings used by every LBVR-Med
// PROV document. Kept as a function (not a var) so callers cannot mutate
// the package-level map and accidentally affect later documents.
func DefaultPrefixes() map[string]string {
	return map[string]string{
		"lbvr": "https://lbvr-med.org/ns/v1#",
		"fhir": "http://hl7.org/fhir/",
		"prov": "http://www.w3.org/ns/prov#",
		"did":  "https://www.w3.org/ns/did#",
	}
}
