# Cryptographic Provenance Specification — LBVR-Med

Companion document to CLAUDE.md §4.6. Full specification for the W3C PROV-JSON extension with BLS-quorum signing and on-chain anchoring.

---

## 1. Goal and positioning

Every retrieval from the LBVR-Med gateway produces a **tamper-evident, machine-verifiable provenance document** recording what was accessed, by whom, from which shards, validated by which quorum, and when. The document is compatible with W3C PROV-JSON (so it interoperates with yProv/interTwin) but extends it with cryptographic signatures.

**Key differentiators from existing provenance systems:**

| System | Signed? | Canonical? | On-chain anchored? | Standard? |
|---|---|---|---|---|
| yProv (interTwin 2025) | ❌ | ❌ | ❌ | W3C PROV |
| PA-XDT (2025) | hashed only | ❌ | ❌ | custom |
| C2PA (Adobe/Microsoft) | ✅ | ✅ | ❌ | C2PA |
| **LBVR-Med (this work)** | **✅ BLS quorum** | **✅ JCS (RFC 8785)** | **✅ Polygon zkEVM** | **W3C PROV-JSON** |

This is the first system to combine all four properties on the W3C PROV standard.

---

## 2. Standards baseline

**W3C PROV-DM (Provenance Data Model)** defines three primary concepts:

- **Entity**: a thing in the world (a FHIR bundle, a retrieval result, a reconstructed shard)
- **Activity**: something that happens over a period of time (a retrieval operation, a verification check)
- **Agent**: a party bearing responsibility (a clinician, a gateway node, an institution)

And seven core relations, of which LBVR-Med uses four:
- `wasGeneratedBy` (Entity → Activity): the retrieval generated the reconstructed bundle
- `wasAssociatedWith` (Activity → Agent): the clinician is associated with the retrieval
- `used` (Activity → Entity): the retrieval used the stored shards
- `wasDerivedFrom` (Entity → Entity): the reconstructed bundle was derived from the shards

**PROV-JSON** (W3C Note 2014) is the JSON serialization of PROV-DM. LBVR-Med produces valid PROV-JSON that any PROV-consuming tool can parse. The signatures are *additive extensions* — verifiers that don't check signatures still see a valid provenance graph.

**JCS (RFC 8785)** defines a canonical JSON serialization: deterministic key ordering, normalized number representation, UTF-8 NFC string normalization. Canonicalization is essential because cryptographic hashes are byte-sensitive; `{"a":1,"b":2}` and `{"b":2,"a":1}` have different hashes but identical semantics. JCS picks one representative per equivalence class.

---

## 3. Extension schema

### 3.1 Namespace

All LBVR-Med extensions use the prefix `lbvr:` bound to `https://lbvr-med.org/ns/v1#`. This is semantically distinct from `prov:` (core W3C PROV) and allows tools to process our extensions without breaking PROV semantics.

### 3.2 Signature block

The core extension is a `sig` field attached to any Entity, Activity, or Agent node:

```json
{
  "sig": {
    "algorithm": "BLS12-381-G2",
    "signers": ["did:lbvr:gw-1", "did:lbvr:gw-2"],
    "quorum_threshold": 2,
    "signature": "0x8f3a...c7d2",
    "signed_at": "2026-04-30T14:32:17.109Z",
    "attestation_type": "retrieval_quorum"
  }
}
```

**Fields:**
- `algorithm`: always `"BLS12-381-G2"` for this version. Future versions may add post-quantum alternatives.
- `signers`: array of DIDs for the gateway nodes that signed. Must have length ≥ `quorum_threshold`.
- `quorum_threshold`: minimum signers required for this signature to be valid. Set to 2 for conference scope.
- `signature`: hex-encoded aggregated BLS signature (96 bytes in G2).
- `signed_at`: ISO 8601 timestamp with millisecond precision.
- `attestation_type`: enum indicating what the signature attests to. See §3.3.

### 3.3 Attestation types

| Type | Attaches to | Semantics |
|---|---|---|
| `entity_integrity` | Entity | The entity's content matches the declared Merkle root and shard layout at retrieval time |
| `retrieval_quorum` | Activity | The retrieval was performed correctly; the claimed shards were fetched; the reconstruction (if any) succeeded; the timing is accurate |
| `agent_identity` | Agent | The agent's identity was verified via the AAI at retrieval time |
| `derivation_chain` | wasDerivedFrom | The declared derivation (e.g., reconstructed bundle was derived from specific shards) is accurate |

For conference scope, every Entity and Activity gets a signature; Agents are signed only if the AAI provides a verifiable credential.

### 3.4 Provenance root anchor

At the top level of every PROV document, LBVR-Med adds:

```json
{
  "lbvr:provenanceRoot": {
    "algorithm": "SHA-256",
    "root": "0xc4e1...8f9a",
    "canonicalization": "JCS-RFC8785",
    "anchored_on_chain": true,
    "chain_txhash": "0x5a7d...bc11",
    "block_number": 8923417,
    "anchor_contract": "0xAbCd...1234"
  }
}
```

This field allows a verifier to:
1. Compute the JCS-canonical hash of the document (excluding the `lbvr:provenanceRoot` field itself)
2. Look up the anchor on-chain via `AuditorLog.getProvenanceAnchor(bundleId, retrievalId)`
3. Verify the computed hash matches the on-chain anchor

If the document has been tampered with, the hashes won't match.

---

## 4. Complete example document

A full PROV-JSON document for one retrieval, with all extensions:

```json
{
  "@context": "https://www.w3.org/ns/prov",
  "prefix": {
    "lbvr": "https://lbvr-med.org/ns/v1#",
    "fhir": "http://hl7.org/fhir/",
    "prov": "http://www.w3.org/ns/prov#",
    "did": "https://www.w3.org/ns/did#"
  },
  "entity": {
    "lbvr:bundle/abc123": {
      "prov:type": "lbvr:FHIRBundle",
      "fhir:resourceType": "Patient",
      "lbvr:merkleRoot": "0x7a3f...e921",
      "lbvr:shardLayout": {
        "D0": {"cid": "QmXa1...", "tier": "pinata",   "shardRoot": "0x11aa..."},
        "D1": {"cid": "QmYb2...", "tier": "filebase", "shardRoot": "0x22bb..."},
        "P0": {"cid": "QmZc3...", "tier": "arweave",  "shardRoot": "0x33cc..."}
      },
      "lbvr:sizeBytes": 823104,
      "sig": {
        "algorithm": "BLS12-381-G2",
        "signers": ["did:lbvr:gw-1", "did:lbvr:gw-2"],
        "quorum_threshold": 2,
        "signature": "0x8f3a2b...c7d2e1",
        "signed_at": "2026-04-30T14:32:17.109Z",
        "attestation_type": "entity_integrity"
      }
    },
    "lbvr:shard/D0-abc123": {
      "prov:type": "lbvr:ErasureShard",
      "lbvr:shardIndex": 0,
      "lbvr:role": "data",
      "lbvr:tier": "pinata",
      "lbvr:cid": "QmXa1...",
      "lbvr:sizeBytes": 411648
    },
    "lbvr:shard/D1-abc123": {
      "prov:type": "lbvr:ErasureShard",
      "lbvr:shardIndex": 1,
      "lbvr:role": "data",
      "lbvr:tier": "filebase",
      "lbvr:cid": "QmYb2...",
      "lbvr:sizeBytes": 411648
    }
  },
  "activity": {
    "lbvr:retrieval/xyz789": {
      "prov:type": "lbvr:VerifiedRetrieval",
      "prov:startedAtTime": "2026-04-30T14:32:15.231Z",
      "prov:endedAtTime":   "2026-04-30T14:32:17.109Z",
      "lbvr:recoveryMode": "fast_path",
      "lbvr:shardsUsed": ["D0", "D1"],
      "lbvr:rsDecode": false,
      "lbvr:latencyMs": 1878,
      "lbvr:quorumSize": 2,
      "lbvr:quorumThreshold": 2,
      "sig": {
        "algorithm": "BLS12-381-G2",
        "signers": ["did:lbvr:gw-1", "did:lbvr:gw-2"],
        "quorum_threshold": 2,
        "signature": "0x9b2c4a...a481f3",
        "signed_at": "2026-04-30T14:32:17.109Z",
        "attestation_type": "retrieval_quorum"
      }
    }
  },
  "agent": {
    "lbvr:requester/clinician-42": {
      "prov:type": "prov:Person",
      "lbvr:role": "clinician",
      "lbvr:institution": "did:lbvr:hosp-1",
      "lbvr:authzPolicy": "EHDS-Art44-primary-use"
    },
    "lbvr:gateway/gw-1": {
      "prov:type": "prov:SoftwareAgent",
      "lbvr:role": "retrieval_gateway",
      "lbvr:version": "lbvr-med-0.1.0",
      "lbvr:publicKey": "0xaabbcc..."
    }
  },
  "wasGeneratedBy": {
    "_:gen1": {
      "prov:entity": "lbvr:bundle/abc123",
      "prov:activity": "lbvr:retrieval/xyz789"
    }
  },
  "wasAssociatedWith": {
    "_:assoc1": {
      "prov:activity": "lbvr:retrieval/xyz789",
      "prov:agent": "lbvr:requester/clinician-42"
    },
    "_:assoc2": {
      "prov:activity": "lbvr:retrieval/xyz789",
      "prov:agent": "lbvr:gateway/gw-1"
    }
  },
  "used": {
    "_:use1": {
      "prov:activity": "lbvr:retrieval/xyz789",
      "prov:entity": "lbvr:shard/D0-abc123"
    },
    "_:use2": {
      "prov:activity": "lbvr:retrieval/xyz789",
      "prov:entity": "lbvr:shard/D1-abc123"
    }
  },
  "wasDerivedFrom": {
    "_:deriv1": {
      "prov:generatedEntity": "lbvr:bundle/abc123",
      "prov:usedEntity": "lbvr:shard/D0-abc123"
    },
    "_:deriv2": {
      "prov:generatedEntity": "lbvr:bundle/abc123",
      "prov:usedEntity": "lbvr:shard/D1-abc123"
    }
  },
  "lbvr:provenanceRoot": {
    "algorithm": "SHA-256",
    "root": "0xc4e1...8f9a",
    "canonicalization": "JCS-RFC8785",
    "anchored_on_chain": true,
    "chain_txhash": "0x5a7d...bc11",
    "block_number": 8923417,
    "anchor_contract": "0xAbCd...1234",
    "anchored_at": "2026-04-30T14:32:20.453Z"
  }
}
```

---

## 5. BLS signing protocol

### 5.1 Key setup

Each gateway node has a BLS12-381 keypair:
- Private key: 32-byte scalar
- Public key: point on G1 (48 bytes compressed)
- Signature: point on G2 (96 bytes compressed)

Keys are generated at gateway startup and registered in `AuditorLog.registerGatewayKey(didString, publicKey)`.

### 5.2 Message to sign

For each signed node in the PROV document, compute:

```
message = SHA-256(JCS_canonicalize(node_without_sig_field))
```

The `node_without_sig_field` is the target node (Entity, Activity, or Agent) with its `sig` field removed. This avoids circular dependencies.

### 5.3 Signing

Each gateway node in the quorum independently:
1. Verifies the retrieval was correct (shards fetched, M_root matches, timing accurate)
2. Computes the message as above
3. Signs: `sig_i = BLS.sign(sk_i, message)`
4. Sends `sig_i` to the signing coordinator

### 5.4 Aggregation

The coordinator aggregates signatures:

```go
aggregatedSig, err := bls.AggregateSignatures(sig_1, sig_2, ..., sig_k)
```

Result is a single 96-byte signature that can be verified against the aggregated public key of the signers.

### 5.5 Verification

```go
func VerifySignature(provNode []byte, sigBlock SignatureBlock, knownKeys map[string]bls.PublicKey) error {
    // 1. Strip sig field from provNode
    nodeWithoutSig, err := stripSigField(provNode)
    if err != nil {
        return fmt.Errorf("strip sig: %w", err)
    }

    // 2. Canonicalize
    canonical, err := jcs.Canonicalize(nodeWithoutSig)
    if err != nil {
        return fmt.Errorf("canonicalize: %w", err)
    }

    // 3. Hash
    message := sha256.Sum256(canonical)

    // 4. Check quorum threshold met
    if len(sigBlock.Signers) < sigBlock.QuorumThreshold {
        return fmt.Errorf("insufficient signers: %d < %d", len(sigBlock.Signers), sigBlock.QuorumThreshold)
    }

    // 5. Look up public keys for signers
    pubKeys := make([]bls.PublicKey, 0, len(sigBlock.Signers))
    for _, signerDID := range sigBlock.Signers {
        pk, ok := knownKeys[signerDID]
        if !ok {
            return fmt.Errorf("unknown signer: %s", signerDID)
        }
        pubKeys = append(pubKeys, pk)
    }

    // 6. Aggregate public keys
    aggPubKey, err := bls.AggregatePublicKeys(pubKeys)
    if err != nil {
        return fmt.Errorf("aggregate pubkeys: %w", err)
    }

    // 7. Decode signature
    sig, err := bls.SignatureFromBytes(sigBlock.Signature)
    if err != nil {
        return fmt.Errorf("decode sig: %w", err)
    }

    // 8. Verify
    if !bls.Verify(aggPubKey, message[:], sig) {
        return fmt.Errorf("signature verification failed")
    }

    return nil
}
```

---

## 6. On-chain anchoring

### 6.1 Smart contract interface

```solidity
// contracts/src/AuditorLog.sol

contract AuditorLog is Ownable {
    struct ProvenanceAnchor {
        bytes32 provHash;
        uint256 blockNumber;
        uint256 timestamp;
        address anchoredBy;
    }

    // bundleId => retrievalId => anchor
    mapping(bytes32 => mapping(bytes32 => ProvenanceAnchor)) public provenanceAnchors;

    // gateway DID hash => BLS public key
    mapping(bytes32 => bytes) public gatewayKeys;

    event ProvenanceAnchored(
        bytes32 indexed bundleId,
        bytes32 indexed retrievalId,
        bytes32 provHash,
        address indexed anchoredBy
    );

    event GatewayKeyRegistered(bytes32 indexed didHash, bytes publicKey);

    function anchorProvenance(
        bytes32 bundleId,
        bytes32 retrievalId,
        bytes32 provHash
    ) external onlyAuthorizedGateway {
        require(provenanceAnchors[bundleId][retrievalId].provHash == bytes32(0), "already anchored");

        provenanceAnchors[bundleId][retrievalId] = ProvenanceAnchor({
            provHash: provHash,
            blockNumber: block.number,
            timestamp: block.timestamp,
            anchoredBy: msg.sender
        });

        emit ProvenanceAnchored(bundleId, retrievalId, provHash, msg.sender);
    }

    function getProvenanceAnchor(
        bytes32 bundleId,
        bytes32 retrievalId
    ) external view returns (ProvenanceAnchor memory) {
        return provenanceAnchors[bundleId][retrievalId];
    }

    function registerGatewayKey(
        string calldata did,
        bytes calldata publicKey
    ) external onlyOwner {
        bytes32 didHash = keccak256(bytes(did));
        gatewayKeys[didHash] = publicKey;
        emit GatewayKeyRegistered(didHash, publicKey);
    }

    modifier onlyAuthorizedGateway() {
        // for conference: any registered gateway can anchor
        // for journal: add threshold multisig
        require(gatewayKeys[keccak256(abi.encodePacked("did:lbvr:", toHex(msg.sender)))].length > 0, "not a registered gateway");
        _;
    }
}
```

### 6.2 Gas costs

On Polygon zkEVM Cardona:
- `anchorProvenance()`: ~50k gas (single SSTORE for struct fields + event emission)
- `getProvenanceAnchor()`: view, free
- `registerGatewayKey()`: ~60k gas, one-time per gateway

At ~20 gwei gas price and 2500 USD/MATIC:
- Per-retrieval anchoring cost: ~$0.001
- At 100K retrievals: ~$100 total

Affordable even for high-volume deployments. For ultra-high-volume scenarios, the journal extension considers batched anchoring (Merkle tree of anchors, one SSTORE per N retrievals).

---

## 7. Verification service design

### 7.1 Verifier as standalone CLI

```bash
$ lbvr-verify --prov-doc ./retrieval-xyz789.prov.json --rpc https://rpc.cardona.zkevm-rpc.com

Verifying provenance document...
  Bundle ID:     abc123
  Retrieval ID:  xyz789
  Recovery mode: fast_path
  Shards used:   D0, D1
  Latency:       1878 ms
  Signed by:     did:lbvr:gw-1, did:lbvr:gw-2 (quorum 2/2)

Canonicalization: JCS-RFC8785 ✅
Document hash:    0xc4e1...8f9a
On-chain anchor:  0xc4e1...8f9a ✅ MATCH

Signature verification:
  Entity signature (entity_integrity): ✅ VALID
  Activity signature (retrieval_quorum): ✅ VALID

Result: VALID ✅
Anchored at block 8923417 (2026-04-30T14:32:20Z)
```

### 7.2 Verifier as library API

```go
// internal/provenance/verifier.go

type VerificationResult struct {
    Valid           bool
    BundleID        string
    RetrievalID     string
    AnchoredBlock   uint64
    FailureReason   string  // populated if Valid == false
    SignatureChecks []SignatureCheck
}

type SignatureCheck struct {
    NodeType        string  // "entity" | "activity" | "agent"
    NodeID          string
    AttestationType string
    Valid           bool
    Reason          string
}

func (v *ProvenanceVerifier) Verify(ctx context.Context, provDoc []byte) (*VerificationResult, error) {
    // See CLAUDE.md §4.6 for full implementation
}
```

### 7.3 Tampering detection test cases

The verifier must detect:

1. **Hash tampering**: modify any field, hash no longer matches on-chain anchor
2. **Signature forgery**: replace signature with garbage, BLS verification fails
3. **Signer substitution**: claim different signers, aggregated pubkey doesn't match signature
4. **Quorum reduction**: claim fewer signers than threshold, verifier rejects
5. **Canonicalization evasion**: reorder keys to try to pass verification, JCS catches it
6. **Timestamp tampering**: modify `signed_at`, hash changes, anchor mismatch

All six must be caught by the verifier. Unit tests for each in `internal/provenance/verifier_test.go`.

---

## 8. Experiment E-PROV

### 8.1 Measurement targets

| Metric | Target | Where measured |
|---|---|---|
| PROV-JSON generation time | <50 ms per retrieval | `generator.go` unit test |
| JCS canonicalization time | <20 ms for 5KB document | `canonicalize.go` unit test |
| BLS signing (single node) | <10 ms | `signer.go` unit test |
| BLS aggregation (2 sigs) | <5 ms | `signer.go` unit test |
| On-chain anchor gas | <60k gas | Foundry gas snapshot |
| Verification (end-to-end) | <200 ms | `verifier.go` integration test |
| Tamper detection rate | 100% on 6 test cases | verifier test suite |

### 8.2 Experiment design

**Sample size:** 1000 retrievals, varying bundle sizes (10KB to 5MB).

**Measurements per retrieval:**
- `t_gen_ms`: time to build PROV-JSON document from retrieval events
- `t_canon_ms`: time to canonicalize
- `t_sign_ms`: time to collect quorum signatures and aggregate
- `t_anchor_ms`: time to submit on-chain transaction (includes block confirmation wait)
- `anchor_gas_used`: actual gas consumed
- `t_verify_ms`: time for verifier to validate (parse + canonicalize + hash lookup + signature verify)

**Tampering test subset:** 100 retrievals, each tested against all 6 tampering modes. Pass rate must be 100% on detection.

### 8.3 Output figure (Fig. 12)

Stacked bar chart showing mean generation/signing/anchoring/verification times across four bundle size buckets (small/medium/large/xlarge). With annotations for anchor gas cost.

**Expected result:** Anchoring dominates total time (~2-5s for block confirmation), but this is asynchronous and doesn't block the retrieval response. Everything else is <300ms total, meeting clinical SLO comfortably.

---

## 9. Implementation checklist

- [ ] `internal/provenance/schema.go` — types for PROV entities + LBVR extensions
- [ ] `internal/provenance/generator.go` — builds PROV docs from retrieval events
- [ ] `internal/provenance/canonicalize.go` — JCS wrapper around `cyberphone/json-canonicalization`
- [ ] `internal/provenance/signer.go` — BLS signing + aggregation
- [ ] `internal/provenance/verifier.go` — verification logic
- [ ] `internal/provenance/anchor.go` — on-chain anchor submission via Go bindings
- [ ] Unit tests: round-trip (generate → sign → anchor → verify → pass)
- [ ] Unit tests: 6 tampering modes (each must be caught)
- [ ] Unit tests: canonicalization idempotency (canon(canon(x)) == canon(x))
- [ ] `cmd/verifier/main.go` — standalone CLI for external verifiers
- [ ] `contracts/src/AuditorLog.sol` — update with anchorProvenance + gateway key registry
- [ ] Foundry tests for AuditorLog (including gas snapshots)
- [ ] Integration test: ingest → retrieve → generate PROV → anchor → verify from CLI
- [ ] Write `bench-E-PROV.go` harness
- [ ] Run E-PROV at scale (1000 retrievals)
- [ ] Generate Fig. 12 + Table 2 for paper
