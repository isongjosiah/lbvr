# CLAUDE.md — LBVR-Med

> **Latency-Bounded Verifiable Retrieval fabric for safety-critical federated systems**
> IEEE ICUFN 2026 submission — deadline **May 8, 2026** (6-page full paper)
> Venue: Milan, Italy, July 7–10, 2026. Track 3: Safety-Critical Networks + Secure Storage Networking and Distributed Systems.

This document is the authoritative brief for Claude Code sessions on this project. Read it end-to-end before making any architectural decisions. If you're about to write code that contradicts what's here, stop and flag it instead.

---

## 1. Who is building this and why

**Researcher:** Josiah Ayoola Isong — Lead Backend Engineer (Go, Python, TypeScript, AWS), M.S. candidate in IT Convergence Engineering at Kumoh National Institute of Technology, Graduate Researcher in the Network Systems Laboratory. Domain expertise: federated learning, Byzantine-robust aggregation, privacy-preserving ML, blockchain, ZK-proofs, genomics/healthcare applications.

**Why LBVR-Med and not something else.** Josiah's existing portfolio covers dynamic consent + verifiable unlearning (ConsentChain), federated PRS with Byzantine robustness (XS-FedPRS, RV-FedPRS), federated epidemic intelligence (FedEpi), and privacy-preserving NFT metadata (Plonky3/STARKs). None of this work touches the **storage substrate** layer of safety-critical federated systems. The decentralized-storage literature of 2024–2026 is paradoxically saturated (dozens of near-identical CP-ABE + IPFS + Ethereum EHR papers) yet empirically vacant — no peer-reviewed work measures P99 retrieval latency, PoR-backed durability, cross-tier erasure recovery, or tier-aware Byzantine resilience on realistic workloads. Trautwein et al.'s INFOCOM 2024 paper "IPFS in the Fast Lane" documents PUTs taking "dozens of seconds to minutes," which breaks clinical SLOs. Li et al.'s TRL-based survey (*P2P Netw. Appl.* 2025, PMC12534302) finds the field stuck at TRL-3 proof-of-concept on synthetic data. Manzi et al.'s interTwin paper (FGCS 2025) federates five storage backends across eight countries but publishes no recovery latency numbers and no PoR. This paper fills these gaps.

**Reframing (as of April 2026):** This paper is no longer framed as "yet another medical storage system." It is framed as **a verifiable storage fabric for safety-critical federated systems, evaluated on healthcare as the motivating regulatory case**. The architecture is domain-neutral; healthcare is the instantiation because EHDS/HIPAA provide the clearest retention-SLA motivation. The cross-SLO calibration experiment (§8, E10) empirically demonstrates applicability to scientific-DT and power-grid DT regimes.

---

## 2. The four contributions

> **We make four contributions:**
> 1. **Cross-tier erasure-coded redundancy**: a Reed-Solomon RS(2,3) protocol that spans three heterogeneous decentralized storage backends (hot/warm/cold) with measured recovery latencies under single and multi-tier failures at 100K-patient scale.
> 2. **Cryptographically-attested provenance**: an extension of W3C PROV-JSON with BLS-signed retrieval receipts, providing tamper-evident data-access histories suitable for EU AI Act Art. 12 compliance and interoperable with the yProv/interTwin provenance ecosystem.
> 3. **Cross-SLO calibration**: the first P99 latency measurement for decentralized verifiable storage across clinical (IEC 60601-1-8), scientific-DT (interTwin ~seconds), and power-grid (IEC 61850 GOOSE) SLO regimes, quantifying which decentralized configurations are feasible for which safety-critical domains.
> 4. **Tier-aware Byzantine withstand**: measured resilience under both uniform and tier-selective adversarial pinning behavior, revealing metadata-correlated attacks that evade standard PoR cadences.

Everything in the paper serves these four contributions. If a feature doesn't support at least one, it's out of scope.

---

## 3. Scope decisions (locked)

### 3.1 Conference (May 8 submission)

| Decision | Value | Rationale |
|---|---|---|
| Workload | **Synthea FHIR R4 bundles, 100K patients (~80 GB)** | Write-heavy exposes IPFS PUT weakness; clean narrative; no credentialing; trivial to scale the eval matrix |
| Tiers benchmarked | **3: Pinata (hot) → Filebase (warm) → Arweave/Irys (cold)** | All three available via simple API keys; no 24h Filecoin deal-sealing on critical path |
| Orchestrator language | **Go** | Matches Josiah's backend stack; single-binary deploy; concurrency model fits parallel fetch + quorum PoR |
| Smart contract stack | **Solidity 0.8.24 on Polygon zkEVM Cardona testnet** | 192-byte Groth16 verifier, cheap gas, stable testnet. Foundry for deployment |
| PoR scheme | **Simplified Fang et al. 2024** — per-bundle Merkle root + BLS-signed retrievability receipts, sampled verification on 30-day cadence | Keeps proof generation under 1 s, verification ≤3 ms, gas ≤200k per submission |
| Erasure coding | **Reed-Solomon RS(2,3) across tiers** — 2 data + 1 parity shard, one shard per tier | Survives one complete tier outage; measured recovery is novel contribution |
| Provenance | **W3C PROV-JSON + BLS signatures**, root hash anchored on-chain | Tamper-evident audit trail; interoperable with yProv/interTwin |
| Consensus | **None custom** — Polygon Cardona L2 finality is sufficient | Would explode scope |
| Evaluation dimensions | See §8 experiment table | Matches what Trautwein INFOCOM 2024 measures, plus four novel axes |
| Baselines | Pinata alone, AWS S3, Storj, public IPFS gateway (`ipfs.io`) | Cover centralized, SaaS-decentralized, and naive decentralized baselines |

### 3.2 Journal extension (IEEE JBHI, post-June 2026)

| Decision | Value |
|---|---|
| Workload | **Synthea + re-wrapped NIH ChestX-ray14 as synthetic DICOM** |
| Tiers | **All 5: Pinata, Kubo-local, Filebase, Arweave, Filecoin FVM** |
| Additional layer 1 | Verifiable retrieval-quorum attestation for federated inference — FL clients emit quorum-signed receipts proving they consumed only EHDS-permitted shards |
| Additional layer 2 | Optional C2PA medical-imaging provenance profile for the DICOM half of the workload |
| Target journal | **IEEE Journal of Biomedical and Health Informatics (JBHI)** — Fang et al. 2024 is adjacent published prior art there |
| Alternatives | IEEE Transactions on Network and Service Management; Future Generation Computer Systems; IEEE Internet of Things Journal |

### 3.3 Third paper (cross-domain evaluation, late 2026 / early 2027)

| Decision | Value |
|---|---|
| Scope | Full empirical evaluation across medical, scientific-DT, and power-grid-DT regimes |
| Target venue | IEEE Transactions on Smart Grid OR Future Generation Computer Systems |
| Motivation | Conference paper's E10 cross-SLO calibration identifies which regimes are feasible; third paper validates with real domain datasets |

---

## 4. Architecture (conference scope)

### 4.1 System layers

```
┌──────────────────────────────────────────────────────────────┐
│  L5: Provenance Layer (new)                                  │
│      • W3C PROV-JSON generation per retrieval                │
│      • BLS quorum signatures on PROV assertions              │
│      • Root hash anchored on-chain via AuditorLog            │
└──────────────────────────────────────────────────────────────┘
                              ▲
┌──────────────────────────────────────────────────────────────┐
│  L4: Auditor Chaincode (Solidity)                            │
│      • PoR receipts → EU AI Act Art. 12 log entries          │
│      • Tier-migration events                                 │
│      • Provenance root anchors                               │
└──────────────────────────────────────────────────────────────┘
                              ▲
┌──────────────────────────────────────────────────────────────┐
│  L3: Retrieval Gateway (Go + Gin)                            │
│      • Parallel fetch across replicas, quorum PoR check      │
│      • k-of-n erasure reconstruction (RS(2,3))               │
│      • P99 SLO enforcement + circuit breaker                 │
│      • Provenance emission                                   │
└──────────────────────────────────────────────────────────────┘
                              ▲
┌──────────────────────────────────────────────────────────────┐
│  L2: Placement Orchestrator (Go)                             │
│      • Tier decision: hot / warm / cold based on:            │
│        - file class (FHIR encounter vs historical archive)   │
│        - access-frequency prediction (EWMA over last 30d)    │
│        - age since ingest                                    │
│      • Erasure shard placement (one shard per tier)          │
│      • Emits placement decisions + CIDs to L1 registry       │
└──────────────────────────────────────────────────────────────┘
                              ▲
┌──────────────────────────────────────────────────────────────┐
│  L1: Encrypting Client (Go CLI)                              │
│      • AES-256-GCM per-bundle envelope                       │
│      • Merkle-tree of fixed-size chunks (16 KB)              │
│      • RS(2,3) encoding: 2 data shards + 1 parity shard      │
│      • CID + root hash + shard layout published to registry  │
└──────────────────────────────────────────────────────────────┘
                              ▲
                    Synthea FHIR bundles
```

### 4.2 Data flow — ingest with erasure coding

1. Client reads FHIR bundle (JSON), chunks into 16 KB segments, computes SHA-256 of each.
2. Builds Merkle tree over all chunks; root hash = `M_root`.
3. Encrypts chunks with AES-256-GCM (random nonce per chunk; key wrapped with a consortium-held KMS key — stub for conference).
4. **NEW:** Applies Reed-Solomon RS(2,3) encoding to the encrypted chunk set, producing 2 data shards (D0, D1) and 1 parity shard (P0). Each shard is an independently addressable blob.
5. **NEW:** Placement orchestrator distributes the three shards across the three tiers: D0 → Pinata (hot), D1 → Filebase (warm), P0 → Arweave (cold).
6. Emits on-chain `RegisterBundle(bundleId, M_root, shardLayout, tier_config, owner, policyId)` on the CID Registry contract. The `shardLayout` is a struct mapping shard index → (CID, tier).
7. Placement Orchestrator schedules a PoR challenge for T+30 days (on a sampled shard).

### 4.3 Data flow — retrieve with erasure recovery

1. Client calls `GET /bundle/{bundleId}` on Retrieval Gateway.
2. Gateway queries on-chain registry for `(M_root, shardLayout, tier_history)`.
3. Fires parallel `GET` against all three shard locations (D0, D1, P0).
4. **Recovery logic:**
   - If D0 + D1 both return within SLO budget → no reconstruction needed, direct merge. Fast path.
   - If any one of {D0, D1, P0} is missing or corrupted → RS decode using any 2 of 3 shards. Slow path.
   - If 2+ shards missing → escalate, log breach, return error.
5. Verify reconstructed chunks against M_root; if match, return to client.
6. Emit retrieval receipt (gateway-signed) and PROV-JSON provenance document to auditor.

### 4.4 PoR protocol (conference scope — simplified)

- **Challenge:** Auditor contract picks a random (shard, chunk) pair and posts `Challenge(bundleId, shardIdx, chunkIdx, nonce)` on-chain.
- **Response:** Any storage replica holding that shard proves it holds chunk `i` by submitting `(chunk_i, merkle_proof_i, BLS_sig(chunk_i || nonce))` off-chain to the gateway, which verifies and submits a hash to the chaincode.
- **Verdict:** Contract records success/failure; after `k` consecutive failures on a shard, the shard is auto-migrated to a healthier tier and the erasure layout is updated.

For the journal version, upgrade to full Fang et al. 2024 vector-commitment scheme with polynomial commitments over BLS12-381.

### 4.5 Erasure coding design (NEW — Tier 2 contribution #1)

**Choice of RS(2,3):** The trade-off space is (data shards, parity shards, tier count). RS(2,3) gives single-tier-failure tolerance at 1.5x storage overhead, which matches the three-tier architecture exactly — one shard per tier, survives any one tier outage. RS(3,5) would give stronger guarantees but requires five independent storage providers, which inflates experimental complexity. RS(2,3) is the minimum meaningful erasure configuration for the tiered model.

**Library:** `github.com/klauspost/reedsolomon` v1.12+. Battle-tested, used in production by Minio, Storj, and others. Go-native, no CGO dependencies.

**Shard size considerations:**
- FHIR bundles range from ~200KB (simple encounter) to ~5MB (full patient history with imaging refs). Encode at bundle granularity, not chunk granularity — this matches the Merkle tree boundary.
- For a 1MB bundle: 2 data shards of 512KB each + 1 parity shard of 512KB = 1.5MB total cross-tier, vs 1MB of duplicated storage (3MB naive 3x replication). Erasure wins on storage cost at 2x.
- Minimum viable shard size: 64KB. Bundles smaller than 128KB should be padded or batched — single-shard overhead dominates below this threshold.

**Alignment with Merkle tree:**
- Each shard is itself Merkle-hashed independently → `shard_root_i`.
- Bundle-level `M_root` is computed over the concatenation `H(shard_root_0 || shard_root_1 || parity_root)`.
- This allows PoR challenges to target a specific shard without requiring full bundle reconstruction.

**Placement policy:**
- D0 (first data shard) → hot tier (Pinata): fastest retrieval on the common path.
- D1 (second data shard) → warm tier (Filebase): balances cost and latency.
- P0 (parity shard) → cold tier (Arweave/Irys): cheapest, used only for reconstruction.
- Rationale: fast path retrieves D0+D1 in parallel from hot+warm (both <500ms typically), avoiding the cold-tier latency. Cold tier only engaged for reconstruction.

**Recovery modes and measured dimensions:**
1. **No failure** (baseline): fetch D0+D1 from hot+warm, skip P0. Measure latency.
2. **Single data shard failure**: e.g., D0 missing, fetch D1+P0, reconstruct D0. Measure recovery latency.
3. **Parity shard failure**: P0 missing, D0+D1 sufficient. Fast path. Measure detection latency (time to notice P0 is dead).
4. **Single-tier outage simulation**: Toxiproxy drops all connections to one tier, gateway must recover from remaining two.
5. **Double-tier outage** (journal scope): should fail; measure failure-detection time and error propagation.

**Toxiproxy integration:**
```yaml
# eval/toxiproxy/erasure-failures.yaml
- name: pinata-outage
  listen: 0.0.0.0:8474
  upstream: api.pinata.cloud:443
  toxics:
    - type: timeout
      attributes:
        timeout: 0  # drop all connections

- name: filebase-slow
  listen: 0.0.0.0:8475
  upstream: s3.filebase.com:443
  toxics:
    - type: latency
      attributes:
        latency: 2000  # inject 2s latency
        jitter: 500

- name: arweave-lossy
  listen: 0.0.0.0:8476
  upstream: gateway.irys.xyz:443
  toxics:
    - type: limit_data
      attributes:
        bytes: 1048576  # cap at 1MB, force partial failures
```

**Experiment E9 — erasure recovery (the core new measurement):**

Input: 100 bundles from the 100K corpus, spanning size distribution (10 small <500KB, 60 medium 500KB–2MB, 30 large 2–5MB).

For each bundle, measure:
- `t_baseline`: fetch time with all three shards available (fast path)
- `t_recover_D0`: fetch + reconstruct time with D0 missing (must fetch P0 from cold tier)
- `t_recover_D1`: fetch + reconstruct time with D1 missing
- `t_recover_P0`: fetch time with P0 missing (fast path, no reconstruction)
- `t_detect_P0_dead`: time to confirm P0 is missing in the fast path (upper-bounded by fast-path SLO)

Output: CDF of recovery latencies per failure mode, overlaid. Expected finding: D0-recovery latency is dominated by cold-tier fetch time (Arweave/Irys ~5–30s), not reconstruction time (RS decode of 1MB ~5ms).

**Why this matters:** No published work measures cross-tier erasure recovery latency for decentralized storage. interTwin federates five backends but has no published recovery numbers. Fang 2024 has PoR but no erasure. This is a clean first-in-class measurement.

### 4.6 Cryptographic provenance (NEW — Tier 2 contribution #2)

**Goal:** Every retrieval from the LBVR-Med gateway produces a tamper-evident, machine-verifiable provenance document that records *what was accessed, by whom, from which shards, validated by which quorum, and when*. The document is compatible with W3C PROV-JSON (so it interoperates with yProv/interTwin) but extends it with cryptographic signatures.

**Standards baseline:**
- W3C PROV-DM (Provenance Data Model): Entities, Activities, Agents, and relations (wasGeneratedBy, wasAssociatedWith, used, wasDerivedFrom).
- PROV-JSON: JSON serialization of PROV-DM.
- yProv (interTwin): hash-and-log, no cryptographic signatures. Our extension is strictly additive — a PROV-JSON document produced by LBVR-Med is valid PROV-JSON that any PROV-consuming tool can parse; the signatures are optional extensions that verifiers can choose to check.

**Extension design:**

Add a `sig` block to Activity, Entity, and Agent nodes. Each `sig` block contains:
```json
{
  "@id": "prov:sig",
  "algorithm": "BLS12-381-G2",
  "signers": ["did:lbvr:gateway-node-1", "did:lbvr:gateway-node-2"],
  "quorum_threshold": 2,
  "signature": "0x8f3a...c7d2",
  "signed_at": "2026-04-30T14:32:17Z",
  "attestation_type": "retrieval_quorum"
}
```

**PROV-JSON document structure per retrieval:**

```json
{
  "@context": "https://www.w3.org/ns/prov",
  "prefix": {
    "lbvr": "https://lbvr-med.org/ns/v1#",
    "fhir": "http://hl7.org/fhir/",
    "prov": "http://www.w3.org/ns/prov#"
  },
  "entity": {
    "lbvr:bundle/abc123": {
      "prov:type": "lbvr:FHIRBundle",
      "fhir:resourceType": "Patient",
      "lbvr:merkleRoot": "0x7a3f...e921",
      "lbvr:shardLayout": {
        "D0": {"cid": "QmXa...", "tier": "pinata"},
        "D1": {"cid": "QmYb...", "tier": "filebase"},
        "P0": {"cid": "QmZc...", "tier": "arweave"}
      },
      "sig": {
        "algorithm": "BLS12-381-G2",
        "signers": ["did:lbvr:gw-1", "did:lbvr:gw-2"],
        "quorum_threshold": 2,
        "signature": "0x8f3a...c7d2",
        "attestation_type": "entity_integrity"
      }
    }
  },
  "activity": {
    "lbvr:retrieval/xyz789": {
      "prov:type": "lbvr:VerifiedRetrieval",
      "prov:startedAtTime": "2026-04-30T14:32:15.231Z",
      "prov:endedAtTime": "2026-04-30T14:32:17.109Z",
      "lbvr:recoveryMode": "fast_path",
      "lbvr:shardsUsed": ["D0", "D1"],
      "lbvr:latencyMs": 1878,
      "sig": {
        "algorithm": "BLS12-381-G2",
        "signers": ["did:lbvr:gw-1", "did:lbvr:gw-2"],
        "quorum_threshold": 2,
        "signature": "0x9b2c...a481",
        "attestation_type": "retrieval_quorum"
      }
    }
  },
  "agent": {
    "lbvr:requester/clinician-42": {
      "prov:type": "prov:Person",
      "lbvr:role": "clinician",
      "lbvr:institution": "did:lbvr:hosp-1"
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
    }
  },
  "lbvr:provenanceRoot": {
    "algorithm": "SHA-256",
    "root": "0xc4e1...8f9a",
    "anchored_on_chain": true,
    "chain_txhash": "0x5a7d...bc11",
    "block_number": 8923417
  }
}
```

**On-chain anchoring:**
- The provenance document is stored off-chain (IPFS/Pinata with the bundle itself, or a dedicated Pinata bucket for provenance).
- A SHA-256 hash of the canonicalized PROV-JSON (using JCS canonicalization — RFC 8785) is anchored on-chain via `AuditorLog.anchorProvenance(bundleId, retrievalId, provHash)`.
- Anchoring cost: ~50k gas per retrieval on Polygon zkEVM (~$0.001 at current prices). Affordable even for high-volume retrieval.

**BLS signing design:**
- Each gateway node has a BLS12-381 keypair.
- On retrieval, the quorum (minimum 2 of N gateway nodes) independently verifies Merkle proofs and signs the canonical hash of the relevant PROV node.
- Signatures are aggregated using BLS aggregate signatures → single 96-byte signature represents quorum agreement.
- Verifier checks: (1) signature validates against aggregated public keys of claimed signers, (2) claimed signers meet quorum threshold, (3) PROV-JSON canonical hash matches on-chain anchor.

**Verifier service:**

```go
// internal/provenance/verifier.go
type ProvenanceVerifier struct {
    chainClient   *ethclient.Client
    auditorLog    *contracts.AuditorLog
    knownSigners  map[string]bls.PublicKey  // DID → BLS pubkey
}

func (v *ProvenanceVerifier) Verify(ctx context.Context, provDoc []byte) (*VerificationResult, error) {
    // 1. Parse PROV-JSON
    doc, err := provjson.Parse(provDoc)
    if err != nil {
        return nil, fmt.Errorf("invalid PROV-JSON: %w", err)
    }

    // 2. Canonicalize (JCS - RFC 8785)
    canonical, err := jcs.Canonicalize(doc)
    if err != nil {
        return nil, fmt.Errorf("canonicalization failed: %w", err)
    }

    // 3. Compute hash
    hash := sha256.Sum256(canonical)

    // 4. Look up on-chain anchor
    bundleId := doc.GetBundleId()
    retrievalId := doc.GetRetrievalId()
    anchor, err := v.auditorLog.GetProvenanceAnchor(ctx, bundleId, retrievalId)
    if err != nil {
        return &VerificationResult{Valid: false, Reason: "no on-chain anchor"}, nil
    }

    if !bytes.Equal(hash[:], anchor.ProvHash) {
        return &VerificationResult{Valid: false, Reason: "hash mismatch"}, nil
    }

    // 5. Verify each signature block
    for _, sig := range doc.GetAllSignatures() {
        if err := v.verifyBLSSignature(sig, doc); err != nil {
            return &VerificationResult{Valid: false, Reason: fmt.Sprintf("sig verification failed: %v", err)}, nil
        }
    }

    return &VerificationResult{
        Valid: true,
        BundleId: bundleId,
        RetrievalId: retrievalId,
        AnchoredAt: anchor.BlockNumber,
    }, nil
}
```

**Experiment E-PROV (part of evaluation §8):**
- Generate 1000 retrievals, each with a PROV-JSON document.
- Measure: (a) PROV-JSON generation overhead per retrieval, (b) BLS signature generation overhead, (c) on-chain anchoring gas cost, (d) end-to-end verification time for a random sampled retrieval.
- Expected results: generation <50ms, signing <20ms, anchoring ~50k gas, verification <200ms.

**Tampering detection test:**
- Generate valid PROV document.
- Modify the `lbvr:latencyMs` field from 1878 to 500 (e.g., falsify performance claim).
- Run verifier. Expected: `Valid: false, Reason: "hash mismatch"`.

**Why this matters:** interTwin's yProv is the most cited DT provenance system (2024–2025). Nobody has extended PROV-JSON with cryptographic signatures. Our extension is minimal, standards-compliant, and directly addresses the EU AI Act Art. 12 requirement for tamper-evident audit logs for high-risk AI systems.

---

## 5. Threat model

### In scope

| Adversary | Capability | Defense |
|---|---|---|
| Byzantine pinning service (uniform) | Refuses to serve, serves corrupted content, disappears | Quorum fetch across tiers; PoR challenges; tier migration; erasure recovery |
| Byzantine pinning service (tier-selective) | **NEW**: Serves correctly during PoR challenges but degrades silently during real retrievals; discriminates based on bundle metadata | **NEW**: Measured in E6b; shows detection gap; partial mitigation via access-pattern anomaly monitoring |
| Network attacker | Drops packets, adds latency, partitions client from one tier | Multi-tier redundancy; P99 SLO budget; circuit breaker; erasure recovery across tiers |
| Curious auditor | Reads on-chain metadata | Only CIDs + Merkle roots + BLS sigs + PROV hashes on chain; no PHI; no access patterns (bundle IDs hashed) |
| CID enumeration attack | Scrapes public IPFS DHT | Private pinning services only (Pinata/Filebase); no public gateway announces |
| Replay of PoR receipts | Storage replica replays old receipt | Nonce in challenge; strict one-time verification |
| Provenance forgery | Attacker crafts fake PROV-JSON claiming retrieval happened | BLS quorum signature requirement + on-chain hash anchor |
| Provenance tampering | Attacker modifies existing PROV-JSON to falsify timestamps or shard usage | Canonical hash anchored on-chain; JCS canonicalization prevents semantic-equivalent tampering |

### Out of scope (flag in paper limitations section)

- Compromised clinician endpoints (assume TEE-attested — future work)
- Clinical AI model correctness
- Side-channel attacks on the HSM holding the consortium KMS key
- Long-term PQ-security of BLS12-381 (acknowledged; journal extension path)
- Collusion among 2+ tiers simultaneously (bounded by erasure parameters: RS(2,3) tolerates 1 failure, not 2)
- Post-retrieval data misuse (PROV records access but does not prevent downstream leakage)

---

## 6. Repo layout

```
lbvr-med/
├── CLAUDE.md                   ← this file
├── README.md                   ← lightweight overview, links to CLAUDE.md
├── go.mod
├── go.sum
├── cmd/
│   ├── client/                 ← encrypting ingest CLI
│   ├── gateway/                ← retrieval gateway HTTP API
│   ├── orchestrator/           ← placement decision daemon
│   ├── verifier/               ← NEW: standalone provenance verifier CLI
│   └── bench/                  ← evaluation harness
├── internal/
│   ├── merkle/                 ← chunked Merkle tree, 16 KB chunks, SHA-256
│   ├── crypto/                 ← AES-256-GCM wrapper; BLS12-381 via kilic/bls12-381
│   ├── erasure/                ← NEW: Reed-Solomon RS(2,3) encode/decode
│   │   ├── encoder.go          ← shard generation
│   │   ├── decoder.go          ← reconstruction logic
│   │   ├── layout.go           ← shard placement strategy
│   │   └── *_test.go
│   ├── provenance/             ← NEW: PROV-JSON generation, signing, verification
│   │   ├── schema.go           ← LBVR extension types
│   │   ├── generator.go        ← builds PROV docs from retrieval events
│   │   ├── signer.go           ← BLS aggregation over quorum
│   │   ├── canonicalize.go     ← JCS (RFC 8785) canonicalization
│   │   ├── verifier.go         ← verification logic
│   │   └── *_test.go
│   ├── tiers/
│   │   ├── pinata/             ← Pinata dedicated gateway client
│   │   ├── filebase/           ← Filebase S3-compatible client (aws-sdk-go-v2)
│   │   └── arweave/            ← Irys SDK wrapper (REST)
│   ├── registry/               ← on-chain contract binding (abigen-generated)
│   ├── por/                    ← challenge/response protocol
│   ├── policy/                 ← tier decision rules
│   └── telemetry/              ← Prometheus metrics + structured logging (zap)
├── contracts/
│   ├── src/
│   │   ├── CIDRegistry.sol     ← tracks shard layouts, not just CIDs
│   │   ├── PoRVerifier.sol
│   │   └── AuditorLog.sol      ← emits Art. 12 events + provenance anchors
│   ├── test/                   ← Foundry tests
│   └── script/                 ← deployment scripts
├── eval/
│   ├── synthea/                ← Synthea runner, seed configs for 1K/10K/100K
│   ├── toxiproxy/              ← fault-injection configs
│   │   ├── byzantine-uniform.yaml
│   │   ├── byzantine-tier-selective.yaml    ← NEW
│   │   └── erasure-failures.yaml            ← NEW
│   ├── tc-netem/               ← Linux traffic-control scripts
│   ├── prometheus/             ← scrape configs, Grafana dashboards
│   └── scripts/                ← Python post-processing for latency CDFs + cross-SLO
│       ├── generate_cdf.py
│       ├── slo_calibration.py              ← NEW
│       └── erasure_recovery_cdf.py         ← NEW
├── docs/
│   ├── architecture.md         ← long-form system description
│   ├── threat-model.md
│   ├── eval-protocol.md        ← exact commands to reproduce every figure
│   ├── regulatory-alignment.md ← EHDS / EU AI Act / HIPAA NPRM mapping
│   ├── erasure-design.md       ← NEW: full erasure coding rationale
│   └── provenance-spec.md      ← NEW: PROV-JSON extension spec
├── paper/
│   ├── main.tex                ← IEEEtran, 6 pages, double-column, 10pt
│   ├── sections/
│   │   ├── 00-abstract.tex
│   │   ├── 01-introduction.tex
│   │   ├── 02-related-work.tex
│   │   ├── 03-system-model.tex
│   │   ├── 04-protocol.tex
│   │   ├── 05-evaluation.tex
│   │   ├── 06-discussion.tex
│   │   └── 07-conclusion.tex
│   ├── figures/
│   └── references.bib
└── .github/workflows/          ← CI: go test + foundry test + markdownlint
```

---

## 7. Critical dependencies and versions

### Go
- **Go 1.22+**
- `github.com/ethereum/go-ethereum` (geth bindings, abigen)
- `github.com/kilic/bls12-381` — BLS signatures + aggregation
- `github.com/klauspost/reedsolomon` v1.12+ — erasure coding
- `github.com/aws/aws-sdk-go-v2` — Filebase S3
- `github.com/gin-gonic/gin` — gateway HTTP API
- `github.com/spf13/cobra` — CLI
- `github.com/prometheus/client_golang` — metrics
- `go.uber.org/zap` — logging
- `github.com/cyberphone/json-canonicalization` — JCS (RFC 8785) for PROV-JSON

### Solidity
- **Solidity 0.8.24** — matches Polygon zkEVM support
- **Foundry** for testing/deployment (NOT Hardhat)
- OpenZeppelin Contracts 5.x — Ownable/AccessControl
- Polygon zkEVM Cardona testnet (chain ID 2442)

### Data
- **Synthea** — https://github.com/synthetichealth/synthea (Java 11+ required)
- Run: `./run_synthea -p 100000 --exporter.fhir.export true --exporter.baseDirectory ./output`

### Storage backends (API keys required — document in `.env.example`, never commit real keys)
- **Pinata** — https://pinata.cloud ($20/mo tier for eval)
- **Filebase** — https://filebase.com ($20/mo tier for eval)
- **Irys** (Arweave) — https://irys.xyz (pay-per-upload, budget ~$15 for eval given erasure overhead)

### Benchmark tools
- **hyperfine** — CLI proving-time
- **bombardier** or **wrk2** — gateway throughput
- **toxiproxy** — fault injection
- **Prometheus + Grafana** — telemetry
- **Linux `tc netem`** — WAN condition simulation

---

## 8. Evaluation protocol — exact experiments

Every figure in the paper maps to one experiment. Name experiments `E1`–`E10` plus `E6b` and `E-PROV`.

| ID | Experiment | Output figure | Expected runtime |
|---|---|---|---|
| E1 | Ingest throughput × corpus size (1K / 10K / 100K patients) × tier (with RS(2,3) enabled) | Fig. 2 — ingest throughput bar chart | ~4h |
| E2 | Retrieval latency CDF, 3 tiers × baseline {Pinata-only, S3, Storj, ipfs.io} — fast path (no recovery) | Fig. 3 — latency CDF | ~6h |
| E3 | P50/P95/P99 retrieval × RTT {10, 50, 200 ms} × loss {0, 1, 5%} | Fig. 4 — heatmap | ~8h |
| E4 | Time-to-availability post-PUT (seconds from PUT return to global reachability) | Fig. 5 — availability-vs-time curves | ~4h |
| E5 | PoR proof cost (prove time, verify gas) at varying bundle sizes | Fig. 6 — bar chart with annotations | ~2h |
| E6 | Byzantine withstand — failure rate vs % malicious pinning replicas {0, 10, 33, 50, 67%} — uniform adversary | Fig. 7a — survival curve | ~6h |
| **E6b** | **Byzantine withstand — tier-selective adversary (behaves correctly during PoR, degrades during retrieval)** | **Fig. 7b — detection-gap curve** | **~3h** |
| E7 | Storage cost $/GB over 30-day window, 3 tiers, with RS(2,3) overhead | Fig. 8 — stacked bars | analytical |
| E8 | End-to-end SLO compliance vs IEC 60601 / CDSS targets | Fig. 9 — SLO attainment table | synthesizes E1–E4 |
| **E9** | **Erasure recovery latency — {no failure, D0 missing, D1 missing, P0 missing}** | **Fig. 10 — recovery CDF** | **~3h** |
| **E9-multi** | **Erasure recovery — two-tier failure (should fail, measure detection time)** | **Fig. 10b — failure mode** | **~3h** |
| **E10** | **Cross-SLO calibration — overlay E3 data against clinical, scientific-DT, power-grid SLO thresholds** | **Fig. 11 — SLO attainment across domains** | **post-processing only (~30min)** |
| **E-PROV** | **Provenance generation/signing/verification latency + gas cost + tampering detection** | **Fig. 12 — provenance overhead + Table 2 — detection rates** | **~2h** |

**Experiment reproducibility requirement.** Every experiment must be launched by a single `make bench-E{n}` command that writes raw JSON to `eval/results/E{n}/` with timestamp, commit hash, and environment fingerprint. Post-processing scripts read from there and emit `.pdf` figures into `paper/figures/`.

**Environment fingerprint** (`eval/results/E{n}/env.json`):
```json
{
  "commit_hash": "abc123...",
  "go_version": "1.22.3",
  "os": "Ubuntu 22.04",
  "kernel": "5.15.0-91-generic",
  "cpu": "Intel Xeon E5-2680 v4",
  "network_path": "KIT Gumi → Pinata (via ...)",
  "wall_start": "2026-04-30T07:00:00Z",
  "wall_end": "2026-04-30T11:04:23Z"
}
```

---

## 9. Paper structure (6-page IEEEtran)

| Section | Pages | Must-contain |
|---|---|---|
| I. Introduction | 0.75 | Four contributions framing → reframed thesis (verifiable fabric for safety-critical federated systems) → Trautwein INFOCOM 2024 + interTwin FGCS 2025 + Thwe IEEE Access 2025 as gap citations |
| II. Related Work | 0.5 | Blockchain-EHR saturation → Li et al. 2025 TRL critique → Fang et al. 2024 as nearest prior (PoR but no erasure); interTwin (federated storage but no PoR); yProv (provenance but no crypto); position LBVR-Med as the first to combine all three |
| III. System Model & Threat Model | 0.75 | The 5-layer figure; adversary table from §5 above; emphasize tier-selective adversary as novel |
| IV. Protocol Design | 1.5 | Tier decision rules; **erasure coding design (0.4 pages)**; **PROV-JSON extension with BLS signing (0.4 pages)**; PoR challenge/response; quorum retrieval algorithm; registry schema |
| V. Evaluation | 1.75 | Figs 2–12; discussion of anomalies; **cross-SLO calibration gets its own subsection** |
| VI. Discussion & Regulatory Alignment | 0.5 | How results map to EHDS Art. 44–50, EU AI Act Art. 12 (provenance directly supports this); HIPAA NPRM restore SLAs; cross-domain applicability pointing at Thwe 2025 / interTwin 2025 future work |
| VII. Conclusion + Journal Extension Roadmap | 0.25 | One paragraph pointing to journal (FL-DT) and third paper (cross-domain evaluation) |

**Word budget:** ~6,500 words. Keep prose dense; push detail to figures and tables.

**Figures are the paper.** Reviewers will read figures first. Invest disproportionate time in Figs 3, 4, 10, 11.

---

## 10. Three-week execution plan (revised with Tier 2 front-loading)

### Week 1 — Scaffolding, ingest path, and erasure integration

**D1** — Repo skeleton, Go modules, Foundry project initialized. Synthea running. Generate the 1K sample corpus end-to-end. Pre-commit hooks wired (`go vet`, `golangci-lint`, `forge test`).

**D2** — Generate the full 100K FHIR corpus (wall-clock overnight job). Validate output: schema checks, bundle size distribution plot, sanity checks on FHIR validity.

**D3** — Merkle chunking + AES-GCM encryption path. Property-based tests for chunk boundary cases. Unit tests for Merkle proof verification. **Start `docs/architecture.md`** — stub sections for each subsystem; flesh out the Merkle + crypto subsystems while they're fresh.

**D4** — Three tier clients (Pinata, Filebase, Irys) with integration tests against real accounts. Each client exposes the same minimal interface (`Put`, `Get`, `Stat`, `Delete`). **Also:** 30-minute Irys spike — upload 20 bundles, measure actual time-to-retrieve to confirm async behavior. **Update `docs/architecture.md`** — tier-client subsystem section.

**D5** — `CIDRegistry.sol` with shard-layout schema deployed to Cardona. Go bindings via `abigen`. Foundry tests. **Complete first full pass of `docs/architecture.md`** — all scaffolded subsystems described; mark erasure + provenance sections as "see §4.5/§4.6 of CLAUDE.md and companion specs."

**D6** — End-to-end ingest CLI working for 1K patients on all three tiers. Prometheus metrics emitting for ingest path.

**D7** — **NEW: First pass at erasure coding.** Wire `klauspost/reedsolomon` into the ingest path for RS(2,3). Get it working for one bundle end-to-end on all three tiers (D0 → Pinata, D1 → Filebase, P0 → Arweave). Unit tests for encoder and decoder.

**Week 1 exit criterion:** `lbvr-med ingest --corpus ./synthea/1k --erasure rs-2-3` completes and stores fragments across Pinata(1) + Filebase(1) + Arweave(1). Registry records CIDs for all three fragments plus the reconstruction metadata. Fast-path retrieval (fetch D0+D1, skip P0) works.

### Week 2 — Tier 2 novelty items + retrieval path + evaluation

**D8 — Erasure recovery harness (Tier 2 #1, part 1)**

Build the retrieval gateway with quorum fetch and Merkle verification. Wire in RS(2,3) reconstruction logic. Unit tests first, then integration tests.

Key tasks:
- `internal/erasure/decoder.go` — given any 2 of 3 shards, reconstruct the original.
- Gateway retrieval loop: fetch all 3 shards in parallel, wait for the first 2 to arrive, reconstruct if needed, verify against M_root.
- Handle partial failures: if D0 times out but D1+P0 arrive, reconstruct D0 and continue.

**D9 — Erasure recovery experiment (Tier 2 #1, part 2)**

Write `bench-E9-erasure.go`. Use Toxiproxy (`erasure-failures.yaml`) to simulate each failure mode. Run across 100 bundles. Output raw JSON to `eval/results/E9/`.

**🔴 GO/NO-GO CHECKPOINT at D9 EOD:**
- ✅ Single-tier erasure recovery working, clean latency numbers → proceed to D10
- ⚠️ Working but numbers suspicious → allocate D10 morning to debugging, decide by D10 noon
- ❌ Not working → **DROP Tier 2 #1 contribution**, use D10-D11 for Tier 1 fallback (access-weighted PoR scheduler + expanded E6 scope)

**D10 — Cryptographic provenance layer (Tier 2 #2, part 1)**

Design the PROV-JSON schema extension (`internal/provenance/schema.go`). Implement:
- `generator.go` — builds PROV docs from retrieval events
- `canonicalize.go` — JCS (RFC 8785) canonicalization
- `signer.go` — BLS aggregation over quorum signatures

Key design decisions to finalize today:
- On-chain anchoring: hash only (cheap, ~50k gas) vs full doc (expensive, prohibitive at scale). **Decision: hash only.**
- Quorum size: 2-of-N (minimum meaningful) vs 3-of-N (stronger but slower). **Decision: 2-of-N for conference, note 3-of-N as journal upgrade.**
- Signature aggregation: aggregate all signers into one signature, or individual signatures? **Decision: BLS aggregate for on-wire efficiency, retain individual sigs for auditability.**

**D11 — Cryptographic provenance end-to-end (Tier 2 #2, part 2)**

Wire provenance generation into the retrieval gateway. Every retrieval produces a signed PROV-JSON document. Implement the verifier service (`internal/provenance/verifier.go`). Write the `AuditorLog.anchorProvenance()` Solidity function and deploy.

End-to-end test: ingest → retrieve → verify provenance → detect tampered provenance.

**🔴 GO/NO-GO CHECKPOINT at D11 EOD:**
- ✅ Provenance signing + verification + on-chain anchoring working end-to-end → proceed to D12
- ⚠️ Signing works but verification flaky → keep as narrative contribution in §IV, reduce E-PROV to qualitative demonstration, proceed to D12
- ❌ Not working at all → **DROP Tier 2 #2 contribution**, use D12 morning to expand E6b scope (more adversary configurations) and add E10 depth

**D12 — PoR protocol + full eval harness ready**

`PoRVerifier.sol` with Foundry tests. Integrate PoR into challenge scheduler. Toxiproxy configs finalized (`byzantine-uniform.yaml`, `byzantine-tier-selective.yaml`, `erasure-failures.yaml`). All `make bench-E{1..12}` commands runnable. **Generate `docs/eval-protocol.md`** — exact reproducibility commands for every figure, matched against the `make bench-E{n}` targets that now exist. This doc is what a reviewer or a second researcher would use to replicate the paper's results.

**D13 — E1, E2, E5, E9 on 100K corpus**

Long-running benchmarks (~15h total). Start E1 at 7am. Draft Sections I-II in parallel while experiments run.

**D14 — E3, E6, E6b, E9-multi, E-PROV**

Heavy experiment day (~20h with overlap). E3 (8h) starts at 7am. E6 + E6b (9h combined) run in parallel on separate machines if possible, otherwise sequentially. E9-multi (3h) and E-PROV (2h) at end of day.

**Week 2 exit criterion:** raw results for E1, E2, E3, E5, E6, E6b, E9, E9-multi, E-PROV in `eval/results/`. Fig. 3 (latency CDF) renders correctly.

### Week 3 — Remaining experiments, paper writing, submit

**D15 — E4, E7, E8, E10 + figure polish**

E4 (4h), E7 (30min analytical), E8 (post-processing), E10 (cross-SLO — post-processing of E3 data, ~30min). Total ~5h of experiment time; rest of day is figure polish for all 12 figures.

**D16 — Anomaly chase + final figure polish**

If any experiment produced weird results, today is the day to rerun or debug. Final figure pass — make sure every figure renders cleanly in LaTeX at IEEEtran 6-page width.

**D17 — Draft Sections I–IV**

Introduction (four-contribution claim), Related Work (positioning against Fang 2024, Trautwein 2024, interTwin 2025, Thwe 2025), System Model & Threat Model, Protocol Design. **Before writing §III, generate `docs/threat-model.md`** — the long-form version of CLAUDE.md §5, now informed by E6 and E6b results. The paper's §III is a compressed version of this doc.

**🔴 GO/NO-GO CHECKPOINT at D17 EOD:**
- Is any section going to need more than half a day to finish? If yes, **drop the weakest experiment from the paper** rather than the section. A tight 6-page paper with 3 strong contributions beats a sprawling one with 4 mediocre ones. Priority order for dropping: E9-multi → E6b → E10 (cross-SLO is the cheapest to keep since it's post-processing).

**D18 — Draft Sections V–VII**

Evaluation (longest section; budget disproportionate time), Discussion + Regulatory Alignment, Conclusion. **Before writing §VI, generate `docs/regulatory-alignment.md`** — the long-form mapping of LBVR-Med features to EHDS Art. 44–50, EU AI Act Art. 12, and HIPAA NPRM provisions. The paper's §VI is a compressed summary of this doc, with specific results cited back to figures.

**D19 — Full pass + figure integration + reference check**

Read the full paper end-to-end. Fix awkward transitions. Verify every claim in the evaluation is supported by a figure or table. Check every citation in `references.bib` resolves. Verify regulatory citations against actual regulation text (EU 2024/1689, EU 2025/327, HIPAA NPRM 90 FR 898).

**D20 — Final proofread, format check, internal buffer**

Submit-ready PDF by end of day. This is the only full buffer day.

**D21 (May 8) — EDAS upload**

Target upload by 18:00 UTC. If any remaining polish items, do them in the morning. Do not touch the paper after submission confirmation.

**Hard deadline:** 23:59 UTC May 8, 2026. Target internal completion by end of D20 (May 7).

---

## 11. What to explicitly NOT do

- ❌ Do not invent a new consensus protocol. Polygon Cardona is the ledger. Period.
- ❌ Do not extend to federated learning in the conference paper. That's the journal extension.
- ❌ Do not benchmark Filecoin FVM deals in the conference paper. 24h sealing latency makes the experiment matrix infeasible. Arweave-via-Irys is the cold tier.
- ❌ Do not add C2PA provenance in the conference paper. W3C PROV-JSON is sufficient. C2PA is journal scope (DICOM workload).
- ❌ Do not use the public `ipfs.io` gateway for anything other than as a negative baseline. P95 is 2–10 s.
- ❌ Do not use MIMIC, eICU, UK Biobank, or dbGaP. Credentialing timelines will blow the May 8 deadline.
- ❌ Do not use Hardhat. Foundry is faster and Josiah can pick it up in an hour.
- ❌ Do not write Halo2 or Plonky3 circuits from scratch. Groth16 via Foundry + snarkjs is sufficient for conference-scope PoR.
- ❌ Do not implement RS(3,5) or higher-parity erasure schemes. RS(2,3) is the minimum meaningful configuration for the three-tier model and produces cleaner experimental results.
- ❌ Do not invent a new PROV extension. Stay compatible with W3C PROV-JSON. Signatures are strictly additive extensions.
- ❌ Do not anchor full PROV documents on-chain. Hash-only anchoring. Full docs go to IPFS.
- ❌ Do not skip the D9 or D11 go/no-go checkpoints. The instinct under deadline pressure is to push harder; the correct move is to pivot to fallbacks.

---

## 12. Writing voice and reviewer framing

- Target audience: Track 3 PC — mix of security, networking, and medical-informatics reviewers.
- Tone: systems paper, not cryptographic paper. Measurements first, proofs second.
- Cite Trautwein INFOCOM 2024, Fang IEEE JBHI 2024, Li PMC12534302, Mazzocca IEEE CST 2025, Manzi interTwin FGCS 2025, Thwe IEEE Access 2025 in the first two paragraphs.
- Anchor every claim in the evaluation to a specific clinical SLO (radiology open <2 s; chart pull <500 ms; IEC 60601-1-8 alarm <10 s) AND note the cross-SLO calibration against power-grid / scientific-DT regimes.
- Acknowledge limitations explicitly (Section VI). Reviewers trust papers that name their weaknesses.
- Use "verifiable storage fabric for safety-critical federated systems" as the canonical phrase — not "medical storage system" or "EHR blockchain."

**Anticipated reviewer questions + prepared answers:**

| Q | A |
|---|---|
| "What's the delta over Fang et al. 2024?" | Fang provides the PoR primitive; we provide (1) cross-tier erasure coding with measured recovery, (2) cryptographic provenance, (3) cross-SLO calibration, (4) tier-selective Byzantine evaluation — none of which Fang measures. |
| "How is this different from interTwin?" | interTwin federates five storage backends but publishes no PoR, no recovery latency numbers, no Byzantine-withstand measurements. Our cryptographic PROV-JSON is interoperable with their yProv ecosystem. |
| "Why RS(2,3) and not higher parity?" | Minimum meaningful erasure for a 3-tier model; 1.5x storage overhead vs 3x naive replication; matches our empirical measurement budget. Higher-parity configurations are journal scope. |
| "Is the consortium KMS a weak point?" | Yes, explicitly acknowledged in §VI limitations. HSM deployment is journal-scope work. For the conference, we focus on the storage-fabric properties, not key management. |
| "Does this generalize beyond healthcare?" | Yes — see E10 cross-SLO calibration. Healthcare is the regulatory motivation; the architecture operates on opaque byte streams. Scientific-DT and power-grid-DT applications are third-paper scope. |

---

## 13. How to work with me (Claude Code)

### 13.1 File layout and required reading

**CLAUDE.md lives at the repo root (`./CLAUDE.md`).** Claude Code auto-loads it at session start. Do not move it.

**Companion design documents live in `./docs/` and must be read before touching the corresponding packages:**

| When you are about to work on... | You MUST first read... |
|---|---|
| `internal/erasure/` (RS(2,3) encoding/decoding) | `docs/erasure-design.md` |
| `internal/provenance/` (PROV-JSON, BLS signing, verification) | `docs/provenance-spec.md` |
| Smart contracts (`contracts/src/`) | This file §4, plus relevant companion doc if contract touches erasure or provenance |
| Evaluation harness (`cmd/bench/`, `eval/`) | This file §8, plus both companion docs for E9/E9-multi/E-PROV |
| Paper drafts (`paper/`) | This file §9, §12, §14 |

**The companion docs are authoritative for their domains.** If something in CLAUDE.md §4.5 conflicts with `erasure-design.md`, the companion doc wins — CLAUDE.md is a summary. If something in CLAUDE.md §4.6 conflicts with `provenance-spec.md`, the companion doc wins. If you notice a conflict, flag it and update CLAUDE.md to match the companion doc.

**Other `docs/` files (may or may not exist yet):**
- `docs/architecture.md` — long-form system description (generate on D3-D5 as scaffolding stabilizes)
- `docs/threat-model.md` — expanded threat model (generate before §III paper draft)
- `docs/eval-protocol.md` — exact commands to reproduce every figure (generate on D12 as eval harness completes)
- `docs/regulatory-alignment.md` — EHDS/EU AI Act/HIPAA NPRM mapping (generate before §VI paper draft)

If a needed `docs/*.md` file doesn't exist yet, create it rather than stuffing its content into CLAUDE.md.

### 13.2 Working rhythm

**Session-start pre-flight check (MANDATORY, run at the start of every session):**

Before doing any task work, Claude Code must run this check:

1. Read CLAUDE.md end-to-end (this file).
2. Identify the current calendar day relative to the May 8, 2026 deadline. Map to D1–D21 per §10.
3. Check whether any `docs/*.md` files that should exist by this point are missing or stale:

| By end of... | These docs must exist and be current |
|---|---|
| D5 | `docs/architecture.md` (first full pass) |
| D12 | `docs/eval-protocol.md` |
| D17 | `docs/threat-model.md` |
| D18 | `docs/regulatory-alignment.md` |

4. For each missing or outdated doc: **flag it to Josiah at the start of the session** before proceeding to day-specific task work. Suggested phrasing: *"Before we start today's work: `docs/X.md` is scheduled for D{n} per CLAUDE.md §10 and is currently missing/outdated. Should I generate/update it now, or defer?"*
5. For each relevant companion doc per §13.1: read it if the session's work touches the corresponding package.

Do not skip the pre-flight check even for "quick" edits. The schedule in §10 is what prevents doc drift from compounding.

**Follow the Week 1 → Week 2 → Week 3 plan in §10 strictly.** If you're tempted to skip ahead, the deadline will bite you.

**Respect the go/no-go checkpoints at D9 and D11.** If a Tier 2 contribution isn't working, pivot to the fallback immediately. Do not push through. The checkpoints exist because pushing through is the default failure mode under deadline pressure.

**Prefer small, testable commits.** Every Go package needs unit tests. Every contract needs Foundry tests with ≥80% branch coverage.

**Run `go vet`, `golangci-lint run`, and `forge test` before every commit.** Wire these into pre-commit hooks on D1.

**When in doubt, ask before building.** If you're unsure whether a feature belongs in conference or journal scope, default to conference-out. Less is more for the 6-page version.

**Benchmark integrity.** Never run eval experiments on a laptop with other workloads. Always record: commit hash, Go version, OS + kernel, network path, CPU model, wall-clock start/end. Put this in `eval/results/E{n}/env.json`.

**Regulatory citations.** Use the exact Regulation (EU) 2024/1689 (AI Act), Regulation (EU) 2025/327 (EHDS), HIPAA NPRM 90 FR 898 (6 Jan 2025) — verify with `web_search` if a reviewer might catch a mis-citation.

**Provenance-specific reminders:**
- JCS canonicalization is strict about key ordering and number formatting. Use the `cyberphone/json-canonicalization` library; do not roll your own.
- BLS signatures must verify against the aggregated public key, not individual keys. Get the aggregation right or verification silently fails.
- On-chain anchor should be the hash of the canonical PROV-JSON, not the hash of the original Go struct serialization.

**Erasure-specific reminders:**
- `klauspost/reedsolomon` requires all shards to be the same size. Pad the last chunk of each bundle to a multiple of the shard count.
- Irys uploads settle asynchronously on Arweave. For E9, the erasure test must either (a) pre-warm the cold-tier shard or (b) explicitly measure cold-tier settlement latency as part of the experiment.
- Do not test erasure on bundles smaller than 128KB; single-shard overhead dominates and pollutes the latency CDF.

---

## 14. Key references to cite (maintain in `paper/references.bib`)

| Shortcut | Citation | Role in paper |
|---|---|---|
| `trautwein2024ipfs` | Trautwein et al., "IPFS in the Fast Lane," INFOCOM 2024 | Latency gap motivation |
| `fang2024metaverse` | Fang et al., "Distributed Medical Data Storage Mechanism Based on Proof of Retrievability and Vector Commitment," *IEEE JBHI* 2024 | Nearest technical prior art for PoR |
| `li2025trl` | Li, Lohachab, Dumontier, Urovi, "Privacy preservation in blockchain-based healthcare data sharing: A systematic review," *P2P Netw. Appl.* 2025 | TRL-3 gap citation |
| `mazzocca2025ssi` | Mazzocca, Acar, Uluagac, Montanari, Bellavista, Conti, *IEEE CST* 2025 | DID/VC landscape |
| `manzi2025intertwin` | Manzi et al., "interTwin: Advancing Scientific Digital Twins through AI, Federated Computing and Data," *FGCS* 2025 | Federated storage engine — closest comparable architecture, lacks PoR + recovery measurements |
| `thwe2025dtps` | Thwe, Ştefanov, Rajkumar, Palensky, "Digital Twins for Power Systems: Review of Current Practices, Requirements, Enabling Technologies, Data Federation, and Challenges," *IEEE Access* 2025 | Cross-domain motivation; data-federation framing |
| `savoia2026fcldt` | Savoia, Annunziata, Thakur, Fortino, Piccialli, "Federated continual learning meets digital twins," *Neurocomputing* 2026 | FL+DT gap citation (journal/third-paper motivation) |
| `ehds2025reg` | Regulation (EU) 2025/327 (European Health Data Space) | Regulatory pull |
| `aiact2024` | Regulation (EU) 2024/1689 (AI Act) | Art. 12 logging requirement |
| `hipaanprm2025` | HHS, HIPAA Security Rule NPRM, 90 FR 898, 6 Jan 2025 | Restore SLA citation |
| `iec60601-1-8` | IEC 60601-1-8:2006+A2:2020 | Alarm-system SLO |
| `iec61850goose` | IEC 61850-8-1 (GOOSE) | Power-grid SLO for cross-SLO calibration |
| `ieee11073-10700` | IEEE 11073-10700-2022 | SDC OR/ICU plug-and-play context |
| `w3cprov` | W3C PROV-DM, PROV-JSON Recommendations (2013) | Provenance standard baseline |
| `rfc8785` | RFC 8785 — JSON Canonicalization Scheme (JCS) | Canonicalization for cryptographic signing |
| `plank1997erasure` | Plank, "A Tutorial on Reed-Solomon Coding for Fault-Tolerance in RAID-like Systems," *Software Practice & Experience* 1997 | Erasure coding classical reference |

---

**End of CLAUDE.md.** If anything here becomes outdated during the build, update this file first and the code second.
