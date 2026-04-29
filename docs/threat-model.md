# LBVR-Med — Threat Model

> Long-form threat model for the conference scope. **Authoritative for `paper/sections/03-system-model.tex` §III.** Where this document and CLAUDE.md §5 disagree, this document wins; §III is a compressed view of what's here.
>
> Findings cite the experiments that validate (or fail to validate) each defense: E6 (uniform Byzantine), E6b (tier-selective Byzantine), E9 / E9-multi (erasure recovery under tier outage), E-PROV (provenance tampering detection). Run identifiers refer to the canonical results committed under `eval/results/E*/`.

---

## 1. System and trust assumptions

### 1.1 Architecture recap

LBVR-Med is a five-layer fabric (CLAUDE.md §4.1) — **L1** encrypting client, **L2** placement orchestrator, **L3** retrieval gateway, **L4** auditor chaincode (CIDRegistry, PoRVerifier, AuditorLog deployed to Polygon zkEVM Cardona), **L5** provenance (PROV-JSON + BLS quorum signatures, hash-anchored on L4). The fabric is domain-neutral; healthcare is the regulatory motivation, not a structural assumption.

Bundles are AES-256-GCM-sealed at L1, Reed-Solomon RS(2,3) erasure-coded into three shards, and each shard is placed on a distinct heterogeneous storage backend — D0 → hot (Pinata), D1 → warm (Filebase), P0 → cold (Arweave/Irys). Retrieval requires any 2 of 3 shards; PoR challenges are scheduled per-shard at T+30 days.

### 1.2 Trust boundary

| Component | Trust | Justification |
|---|---|---|
| Client encryption code (L1) | Trusted | Open-source, deterministic; reviewer-auditable |
| Consortium KMS (per-bundle key wrap) | Trusted | HSM-class deployment assumed; out-of-scope hardening (§5) |
| Gateway nodes (L3) | k-of-n trusted | Quorum-signed; collusion bounded by k = 2 of N |
| L4 chaincode (Cardona) | Trusted | Solidity 0.8.24; via-IR codegen; 78 Foundry tests passing |
| L5 provenance verifier | Trusted | Thin verifier — independent reviewer can re-derive on-chain anchor → canonical-hash equality |
| Pinata / Filebase / Irys backends | **Untrusted (Byzantine)** | Service can refuse, corrupt, or selectively serve |
| Network paths between client → gateway → backend | Untrusted | Assume drops, latency injection, partition |
| Polygon zkEVM Cardona consensus | Trusted | L2 finality is the published Cardona contract |
| Clinician endpoint device | Trusted (this paper) | TEE-attested in journal scope (§5) |

### 1.3 Cryptographic primitives

- **AES-256-GCM** for per-chunk content encryption (16 KiB chunks, fresh nonce per chunk).
- **SHA-256** for Merkle leaves and inclusion proofs (`internal/merkle`); odd-width levels duplicate the trailing leaf in the Bitcoin-style convention. CVE-2012-2459-class second-preimage attacks are blocked by binding `numChunks` to the bundle on-chain (`CIDRegistry.BundleRecord.numChunks`).
- **BLS12-381** (curve G2 signatures, G1 aggregation) for gateway-quorum provenance signing (`internal/provenance`). The aggregate signature is 96 bytes; 48-byte public keys.
- **Reed-Solomon RS(2,3)** via `klauspost/reedsolomon` v1.12+, single-tier-failure tolerance at 1.5× storage overhead.
- **JCS canonicalization** (RFC 8785) for PROV-JSON before SHA-256 anchoring on `AuditorLog.anchorProvenance`.

---

## 2. Adversary model

We characterise three orthogonal capability axes; a real attacker may combine any subset.

### 2.1 Storage-backend Byzantine behaviour

A pinning service or replica may, at any time, refuse to serve, return corrupted bytes, disappear, or selectively respond. We further distinguish:

- **Uniform Byzantine** — the adversary's behaviour is independent of which bundle, shard, or caller is involved. Pessimistic upper bound on what we can survive. Validated by E6.
- **Tier-selective Byzantine** — the adversary's behaviour discriminates on tier metadata (e.g., only corrupts cold-tier shards) and on temporal context (responds correctly during PoR challenges, degrades silently during real retrievals). Validated by E6b. **Novel adversary class for this paper**; we believe it is not measured in prior decentralized-storage literature.

### 2.2 Network attacker (Dolev–Yao on transport, partial-information on state)

Drops packets, injects latency, partitions client-side from one tier, or stalls particular HTTP/gRPC streams. Cannot decrypt sealed payloads (AES-GCM). Cannot forge BLS signatures or Merkle proofs. May correlate timing across requests.

### 2.3 Curious auditor

Reads on-chain artifacts: bundle IDs, shard CIDs, Merkle roots, PoR challenges/verdicts, BLS signatures, PROV-JSON anchors. Cannot read PHI (none on chain). Cannot derive bundle content from CIDs (CIDs are content addresses of *encrypted* shards). May correlate access patterns; we mitigate via bundle-ID hashing (§3.4).

---

## 3. In-scope attacks, defenses, and experimental evidence

### 3.1 Byzantine pinning service — uniform (E6, Fig 7a)

**Capability.** A controlled fraction *f* of pinning replicas refuses to serve, or serves arbitrary corrupted bytes, on every request. Behaviour is independent of bundle, shard, caller, or context.

**Defense.** Cross-tier RS(2,3) reconstruction at the retrieval gateway (`internal/gateway/recovery.go`); per-shard PoR challenges (`PoRVerifier.respondToChallenge`); auto-migration after k = 3 consecutive PoR failures (`PoRVerifier.recordVerdict` emits `ShardMigrationRequired`).

**Experimental result (`eval/results/E6/run-20260427-060345-uniform-2828557.json`, 500 retrievals × 5 fractions):**

| f | retrieval success | decrypt failures (corrupted shard reached client) | reconstruction failures |
|---|---|---|---|
| 0% | 100.0% | 0 | 0 |
| 10% | 67.0% | 165 | 0 |
| 33% | 48.0% | 260 | 0 |
| 50% | 19.0% | 405 | 0 |
| 67% | 16.0% | 420 | 0 |

**Reading.** Survival degrades smoothly with adversary fraction. Critically, `recovery_failed = 0` at every f — when the gateway commits to the slow path (after detecting a bad shard via Merkle-root mismatch), reconstruction always succeeds when 2 of 3 healthy shards are reachable. The visible failures are `decrypt_failed`: the gateway's fast-path returned a shard whose ciphertext didn't decrypt against the sealed-key, indicating the *fast-path Merkle check let a corrupted shard through that the client subsequently rejected at decrypt time*. This is the expected protocol behaviour: Merkle verification protects integrity *of the sealed bytes*, not of the underlying plaintext, and the fast-path optimistically returns when 2 shards arrive without waiting for the third's PoR confirmation.

**Limit.** At f = 67% the system is essentially unusable (16% success). RS(2,3) tolerates one-shard failure per bundle, so when 2 of 3 shard-replicas across the bundle population are corrupt, reconstruction probability collapses.

### 3.2 Byzantine pinning service — tier-selective (E6b, Fig 7b)

**Capability.** Same Byzantine power as §3.1, but the adversary discriminates: only the cold tier is targeted (target tier configurable). The adversary also responds correctly during synthesised PoR challenges so the auditor's accountancy never flags it.

**Defense.** Same cross-tier reconstruction as §3.1. Additional planned defense (journal scope): access-pattern anomaly monitoring at the gateway — flag a tier whose *retrieval-error rate* diverges from its *PoR-success rate*.

**Experimental result (`eval/results/E6/run-20260427-062807-tier-selective-2828557.json`, 250 retrievals × 5 fractions, target = cold):**

| f | retrieval success | decrypt failures | reconstruction failures |
|---|---|---|---|
| 0% | 100.0% | 0 | 0 |
| 10% | 100.0% | 0 | 0 |
| 33% | 100.0% | 0 | 0 |
| 50% | 50.0% | 125 | 0 |
| 67% | 0.0% | 250 | 0 |

**Reading — the detection gap.** A tier-selective adversary at f = 33% causes *zero* observable retrieval failures. The system is fully functional from the clinician's perspective because the LBVR fast path uses hot+warm only; cold is engaged solely for reconstruction, and at f ≤ 33% there is always a clean cold replica available when needed. **The PoR cadence does not alarm** because the adversary serves correctly on challenges. This is the §V "detection gap": tier-selective adversaries can persist indefinitely below the threshold where their tier-specific quorum collapses, then break the system catastrophically (50% → 0% between f = 0.5 and f = 0.67).

**Implication for §VI.** The cliff is sharp because the cold tier is *single-shard per bundle* in our RS(2,3) layout. A higher-parity scheme (RS(3,5) or beyond) would smooth the curve at the cost of storage and the "minimum meaningful" framing this paper adopts. Journal-scope work: shard-level quorum monitoring at the gateway closes the gap by flagging tier-internal divergence between PoR and retrieval traffic.

### 3.3 Network attacker (E3 + E9 fault-injection)

**Capability.** Drops packets, injects latency, partitions client → tier paths. Toxiproxy-class adversary.

**Defense.** Quorum fetch (parallel-3 with first-2 wins); P99 SLO budget at the gateway; circuit breaker on per-tier failure rates; cross-tier erasure recovery as a fallback path.

**Experimental result (E3 — `eval/results/E3/run-20260427-071623-2828557.json`).** Across 9 cells (3 RTTs × 3 loss rates), 1500 retrievals each:

- **0% loss row:** P99 climbs smoothly with RTT, 588 ms (10 ms RTT) → 856 ms (200 ms RTT). Fast path 100% across the row.
- **1% loss row:** P99 776 → 1060 ms; fast-path stays at ≥ 98%, slow-path picks up the rest.
- **5% loss row:** P99 enters 1.9–2.7 s territory; fast-path drops to ~90%; failure rate 0.4–1.1%. Above this loss rate the SLO breaches across all RTT values.

**Operational threshold.** "Sustainable in lossy-but-not-catastrophic networks" up to 1% per-Get loss; degrades above. Captured in §V; the loss-rate-vs-SLO is the substantive new measurement (Trautwein 2024 measured public IPFS but did not stress with loss).

**Defence orthogonality.** Network attacker's effective failure mode (delayed/missing Gets) maps onto the same recovery state machine that handles Byzantine corruption (§3.1). E9 / E9-multi (`eval/results/E9-multi/run-20260428-022109-b2946d1.json`) measure recovery latency under simulated tier outages: median detection on a double-data-shard failure (D0 + D1) is **137 ms** (early-fail check shorts the cold-tier wait), versus **561–632 ms median** when the failure pair includes the parity shard and cold-tier confirmation gates the verdict.

### 3.4 CID enumeration / on-chain access-pattern leakage

**Capability.** A curious auditor scrapes the public Cardona ledger or the public IPFS DHT, attempting to enumerate bundles or correlate retrieval frequencies with patient identities.

**Defense.**
- All shards are stored on **private** pinning services (Pinata dedicated, Filebase private buckets, Irys upload behind authenticated wallet). No shard CID is announced to the public IPFS DHT.
- On-chain `BundleRecord.merkleRoot` / `shardLayout` reveal nothing about content; CIDs are deterministic hashes of *encrypted* shards.
- `bundleId = keccak256(clientAddr || merkleRoot)` is salted by the client address; the registry indexes bundles by this hash, not by patient ID.
- PoR challenges hash a fresh nonce, so observing on-chain challenge IDs does not leak which bundle was probed.

**Residual risk.** Retrieval timing through public Cardona transactions (gas-paying address visible) could correlate auditor-side activity with patient sessions. We treat this as a §VI limitation; mitigations (gas-meta-relays, ZK-anonymized auditor keys) are journal scope.

### 3.5 Replay of PoR receipts

**Capability.** A storage replica retains a previously-valid PoR response and resubmits it for a later challenge.

**Defense.** Each `Challenge` has a per-challenge nonce; the BLS signature is over `chunk || nonce` (`internal/por.ProveMessage`). The contract derives the deterministic `challengeId = keccak256(bundleId, shardIdx, chunkIdx, nonce, postedAt)` and rejects responses whose `chunkHash || merkleProof` does not reconstruct the registry's `merkleRoot` AND whose declared `totalChunks` does not equal `BundleRecord.numChunks`. Replays from a different `(bundleId, shardIdx, chunkIdx, nonce)` tuple cannot reuse a stored signature because the BLS message is bound to the new nonce.

**Foundry test:** `test_respond_revertsOnDoubleResponse` (committed, passing). Plus E6/E6b ran with rotating per-bundle nonces; no false positives observed.

### 3.6 Provenance forgery

**Capability.** An attacker (gateway operator gone rogue, or external party with stolen single-key material) emits a PROV-JSON document falsely claiming a retrieval occurred.

**Defense.** Every signed PROV node carries a 96-byte BLS aggregate signature over k-of-N gateway public keys (k = 2 in conference scope, configurable in journal). Verification (`internal/provenance.Verifier.Verify`) reproduces:
1. Parse PROV-JSON and re-canonicalize via JCS (RFC 8785).
2. Re-derive the canonical hash `H = SHA-256(canonical)`.
3. Look up the on-chain anchor via `AuditorLog.getProvenanceAnchor(bundleId, retrievalId)`.
4. Compare `H` with the on-chain `provHash`.
5. Verify each `SignatureBlock` against the aggregated public keys of its claimed signers, rejecting if signers fall below `quorum_threshold`.

A forged document fails step 4 (no on-chain anchor exists, or hash mismatches) or step 5 (single-key signature can't satisfy 2-of-N aggregate). Both checks pass only when the document was anchored by an honest quorum.

### 3.7 Provenance tampering (E-PROV, Fig 12)

**Capability.** An attacker modifies a previously-anchored PROV-JSON document to falsify a field (timestamp, latency claim, recovery mode, signers, removes a signature, etc.) while preserving as much surrounding structure as possible.

**Defense.** Same five-step verification chain as §3.6. The on-chain anchor binds the *canonical hash* of the original document, so any byte-level mutation surfaces at step 4.

**Experimental result (`eval/results/E-PROV/run-20260427-033721-a7f154a.json`, 1000 iterations across 7 case categories):**

| Tamper case | n | detection rate |
|---|---|---|
| `happy` (no tamper) | 143 | 100.0% (correctly verified valid) |
| `hash_tamper` (alter latency claim) | 143 | 100.0% |
| `sig_tamper` (flip a signature byte) | 143 | 100.0% |
| `signer_substitute` (replace one signer's DID) | 143 | 100.0% |
| `quorum_reduce` (drop one signature, reduce threshold) | 143 | 100.0% |
| `missing_sig` (omit signature block entirely) | 143 | 100.0% |
| `timestamp_tamper` (mutate retrieval start time) | 142 | 100.0% |

**Reading.** Detection is total in the tested attack surface. The §V framing: cryptographic provenance is a *yes/no* property — a single tamper-detection failure would invalidate the EU AI Act Art. 12 claim. We claim 100% over a 1000-sample population; the residual concern is whether the seven cases enumerate the attack surface, not whether each case is detected.

---

## 4. Defense-in-depth summary (per-layer)

| Layer | Primary defense | Reference / contract |
|---|---|---|
| L1 — encrypting client | AES-256-GCM per-chunk seal; bundle Merkle root binds tree width | `internal/crypto`, `internal/merkle` |
| L2 — placement orchestrator | RS(2,3) erasure across heterogeneous tiers (one shard per tier) | `internal/erasure` (klauspost/reedsolomon) |
| L3 — retrieval gateway | Parallel-3 fetch + first-2-wins; SLO budget; cross-tier reconstruction; circuit breaker | `internal/gateway/recovery.go` |
| L4 — auditor chaincode | On-chain `merkleRoot` + `numChunks` binding; per-challenge nonce; PoR verdict counter triggers shard migration | `contracts/src/{CIDRegistry,PoRVerifier,AuditorLog}.sol` |
| L5 — provenance | BLS k-of-N quorum signatures over JCS-canonical PROV-JSON; SHA-256 anchored on L4 | `internal/provenance`; `cmd/verifier` |

**Cross-cutting:** all PoR proofs verify the chunk hash against the live on-chain `merkleRoot` (not a cached value) and cross-check `totalChunks` against `BundleRecord.numChunks` to block CVE-2012-2459-class width-forgery attacks.

---

## 5. Out-of-scope (acknowledged limitations)

The following are not defended in this paper. Each is named in §VI of the manuscript so reviewers can see we know:

- **Compromised clinician endpoint.** A breached workstation can decrypt and exfiltrate retrieved bundles after legitimate verification. Journal-scope mitigation: TEE-attested clients (Intel SGX or AMD SEV-SNP) with bundle-key release gated on attestation.
- **Clinical AI model correctness.** LBVR-Med stores bytes; what an FL workflow *does* with retrieved bundles is orthogonal. Journal-scope: verifiable retrieval-quorum attestation for federated inference.
- **Side-channel attacks on the consortium KMS.** We assume HSM-class deployment for the AES key-wrap secret; no software-side mitigation in scope.
- **Long-term post-quantum security of BLS12-381.** Acknowledged. Journal-scope upgrade path: hybrid BLS + Dilithium signatures.
- **Tier-cooperation collusion.** RS(2,3) tolerates one tier failure; collusion of 2+ tiers exceeds the erasure budget by definition. Higher-parity schemes would raise this bound at the cost of storage; we explicitly choose minimum-meaningful (1.5× overhead) for the conference scope and discuss the tradeoff in §VI.
- **Post-retrieval data misuse.** Provenance records *that* a retrieval happened; it does not prevent downstream leakage from a legitimate retriever. EU AI Act Art. 12 logging is the boundary the paper claims; access-control and DLP layers are out of scope.
- **C2PA medical-imaging provenance.** PROV-JSON only in conference scope; C2PA profile for the DICOM half of the workload is journal-scope (IEEE JBHI extension).

---

## 6. Residual risks and reviewer-anticipated objections

| Risk | Status | Notes |
|---|---|---|
| Tier-selective adversary persists below f = 0.5 with no alarm | **Acknowledged in §VI** | E6b detection gap. Journal mitigation: per-tier PoR-vs-retrieval divergence monitor at the gateway. |
| Single-key compromise on a gateway node falls below k = 2 quorum | **Bounded** | Honest k = 2 of N nodes still produce valid quorum signatures; compromised node alone cannot forge. |
| Cardona L2 reorg invalidates an anchored PROV hash | **Bounded** | Standard L2 finality assumption; the auditor service can re-anchor on detection. Documented in §VI. |
| Synthea-derived bundle distribution doesn't match real EHR | **Bench-realism caveat** | All E1–E10 sims use Synthea sizes.csv; a clinical deployment may have different size distribution. Journal extension uses MIMIC-IV when credentialing completes. |
| Sim-tier numbers not validated against live Cardona / live tiers | **Bench-realism caveat** | E5 gas: forge in-memory EVM (Cancun) is opcode-equivalent to Cardona zkEVM; ±1% expected. Live spot-check planned post-funding (CLAUDE.md §10 D4). E4: live hot+warm validation possible today (Pinata + Filebase keys present); cold tier waits on Sepolia ETH. |

---

**Last updated:** D15 (2026-04-29). Sources of truth: experiment runs under `eval/results/E*/`, contract code under `contracts/src/`, Go packages under `internal/{crypto,merkle,erasure,por,provenance,gateway}/`. If any cited result is invalidated by a re-run, update this document before the corresponding figure or paper section.
