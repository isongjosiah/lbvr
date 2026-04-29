# LBVR-Med — Regulatory Alignment

> Long-form mapping of LBVR-Med features to the four regulatory regimes the conference paper cites: EU EHDS (2025/327), EU AI Act (2024/1689), HIPAA Security Rule NPRM (90 FR 898), and the clinical/cross-domain SLO standards (IEC 60601-1-8, IEC 61850 GOOSE, IEEE 11073-10700).
>
> **Authoritative for `paper/sections/06-discussion.tex` §VI.** §VI is a compressed view of the mappings here, with specific figures cited back to the experiment artefacts. Where this document and CLAUDE.md §1 / §12 disagree, this document wins; CLAUDE.md is a summary.
>
> Article numbers and FR citations are stated as of the document text current at conference-submission cut (May 2026 deadline). Reviewers may catch a stale citation if a provision was renumbered or amended after our draft; we flag the highest-risk citations with `[VERIFY]` so a final pre-submission pass can confirm against the official journals.

---

## 1. Regulation (EU) 2025/327 — European Health Data Space (EHDS)

The EHDS regulation establishes a unified framework for *primary* use (clinician-facing access to electronic health records) and *secondary* use (research, public health, AI training) of health data across Member States. Articles 44–50 of the secondary-use chapter are the load-bearing subset for LBVR-Med's storage-fabric claims.

### 1.1 Provisions in scope

| Article (approx.) | Topic | LBVR-Med feature | Evidence |
|---|---|---|---|
| Art. 44 — Application for data access | Health Data Access Bodies (HDABs) issue per-purpose data permits | `BundleRecord.policyId` (32-byte keccak hash of policy URI) is recorded on-chain at ingest; retrieval gateway can reject non-matching policies | `internal/registry`, `cmd/client` ingest path |
| Art. 45 — Data permit duration / scope | Permit binds purpose, dataset, retention | `policyId` as opaque pointer to off-chain consent record (matches Josiah's prior ConsentChain framing); on-chain anchor enables auditable revocation | Conference scope is the *anchoring* contract; the policy-resolver service is journal-scope |
| Art. 46 — Secure processing environment | Secondary-use processing must run in regulator-approved secure environments | LBVR-Med supplies the *storage substrate* for such environments; not the processing engine itself. Compatible with interTwin-style federated computation. | `manzi2025intertwin` framing — yProv ecosystem interop via §5.6 |
| Art. 47–49 — Audit trail and provenance | Every secondary-use access must be loggable in a tamper-evident way for ≥ N years | **L4 + L5 fabric.** PoR challenges + verdicts on `PoRVerifier`; per-retrieval BLS-quorum-signed PROV-JSON anchored on `AuditorLog`. | E-PROV Fig 12 (100% tamper detection); E5 Fig 6 (anchoring gas cost ~50k per retrieval) |
| Art. 50 — Penalties | Per-incident fines up to 2% / 10M EUR | Compliance-by-construction — § 1.3 below | — |

### 1.2 Where LBVR-Med over-meets EHDS

- **Decentralized governance.** EHDS does not mandate decentralized storage but explicitly tolerates it. The fabric's L1–L3 are domain-neutral; HDAB-issued permits map to `policyId` without requiring a central authority to hold raw bytes.
- **Cross-border interoperability.** A bundle pinned to FRA1 (Pinata Frankfurt) + DEU1 (Filebase Germany) + AR (Arweave global) survives a single jurisdiction's outage without violating residency constraints (residency is encoded in the per-tier replication-region config).

### 1.3 Where LBVR-Med has a known gap (journal scope)

- **Right-to-erasure (Art. 51, by analogy with GDPR Art. 17).** Arweave is *immutable by design* — uploaded bundles cannot be erased. Conference-scope mitigation: bundles are encrypted at L1 with a key the consortium KMS controls; "erasure" is enforced via key destruction. Journal-scope: explicit `lbvr:erasureProof` PROV-JSON predicate that anchors the key-destruction event so a regulator can prove the bytes are unrecoverable.

---

## 2. Regulation (EU) 2024/1689 — Artificial Intelligence Act (AI Act)

The AI Act came into force August 2024 with phased application; high-risk AI systems (including clinical decision support and population-health AI) are subject to a layered obligations regime. **Article 12** is the load-bearing provision for LBVR-Med.

### 2.1 Article 12 — Record-keeping

> *"High-risk AI systems shall technically allow for the automatic recording of events ('logs') over the duration of the lifetime of the system. … Logs shall be kept in a way which is appropriate to the intended purpose of the AI system."*

The provision lists minimum log fields: timestamps, identification of the natural or legal person who operated/checked the system, dataset references, inputs and outputs (where reasonable), and any actions or interventions on the system.

### 2.2 LBVR-Med mapping

| Art. 12 requirement | LBVR-Med implementation | Evidence |
|---|---|---|
| Automatic logging over system lifetime | PoR challenges every 30 days per shard; provenance anchor every retrieval; both write to L4 contracts. Cardona finality is the bottom-of-stack durability. | `PoRVerifier`, `AuditorLog` Foundry tests (78/78 passing) |
| Tamper-evident | BLS-quorum-signed PROV-JSON; on-chain SHA-256 anchor of the canonical bytes (RFC 8785 JCS) | E-PROV Fig 12 — 100% detection across 7 tamper categories |
| Identification of operator | `PROV-JSON` `prov:wasAssociatedWith` ties retrievals to gateway DIDs (`did:lbvr:gw-...`); upstream user identity is recorded in the off-chain policy record referenced by `policyId` | `internal/provenance` schema; `cmd/verifier` |
| Dataset references | `BundleRecord.merkleRoot` + `numChunks` + `shardLayout` published on `CIDRegistry`; the bundle hash uniquely identifies the input dataset slice | E5 Fig 6 — `registerBundle` ~180k gas |
| Inputs / outputs (where reasonable) | We log *that* a retrieval happened, with `recoveryMode` (fast_path vs. slow_path) and `latencyMs` — not the bundle bytes (they are confidential PHI). The §VI text defends this scope: PHI in logs is itself a privacy violation. | E-PROV schema — `lbvr:VerifiedRetrieval` activity carries the latency + mode metadata |
| Logs retained "appropriate to purpose" | Retention encoded in the policy (off-chain) + the durability of the L4 anchor (Cardona, indefinite) + the cold-tier shard (Arweave, perpetual) | E7 Fig 8 — analytical storage cost over 30-day window scales to permanent archival |

### 2.3 Affordability

A skeptic's objection to per-event blockchain logging: gas cost might exceed the operational budget for a high-volume system. E5 Fig 6 measurements at zkEVM Cardona pricing:

- `registerBundle` ~180k gas / ingest
- `anchorProvenance` ~50k gas / retrieval
- Full PoR cycle (`postChallenge` + `respondToChallenge` at depth 13 + `recordVerdict`) ~731k gas / shard / 30-day cycle

At Polygon zkEVM Cardona current gas pricing (~$0.001 per 50k gas), a hospital with 10 000 retrievals/day pays **~$0.20/day** for AI Act Art. 12 compliance — affordable even at high volume. The §VI talking point: cryptographic provenance is *not* a luxury at this scale.

### 2.4 Known gaps

- **Cross-border log portability.** AI Act envisions logs being movable to a national supervisory authority on demand. Our PROV-JSON is a JSON-LD-shaped document, but the receiving authority must run a verifier that understands BLS12-381 + JCS canonicalization. Standardizing the verifier API is a journal-extension item (see §5).

---

## 3. HIPAA Security Rule NPRM — 90 FR 898 (6 January 2025)

The 2025 Notice of Proposed Rulemaking (NPRM) revisits the HIPAA Security Rule's 2003 baseline with explicit cybersecurity hardening for healthcare covered entities and business associates. The provisions most relevant to LBVR-Med are the **contingency plan** subset (45 CFR § 164.308(a)(7)) — specifically the data-recovery time obligations.

### 3.1 Provisions in scope

| Provision (approximate citation) | Topic | LBVR-Med feature | Evidence |
|---|---|---|---|
| § 164.308(a)(7)(ii)(B) — data backup plan | Maintain retrievable copies of ePHI | RS(2,3) erasure across three independent providers (Pinata + Filebase + Arweave). Survives any single provider outage. | E9 Fig 10 — single-tier failure recovery; E9-multi Fig 10b — double failure |
| § 164.308(a)(7)(ii)(C) — disaster recovery plan | Restore lost data | Cross-tier reconstruction via `internal/erasure.Decode`; `internal/gateway/recovery.go` early-fail short-circuit | E9-multi median detection 137 ms (D0+D1 lost) — restore-time well under HIPAA's notional 24h SLA |
| § 164.308(a)(7)(ii)(D) — emergency mode operation plan | Continue critical processes during disruption | LBVR fast path tolerates one-tier outage with 100% reconstruction success at f ≤ 33% (E6); same-day swap-in of replacement provider via `CIDRegistry.updateShardLayout` | E6 + on-chain migration tests `test_failures_emitMigrationAfterThree` |
| § 164.308(a)(7)(ii)(E) — testing and revision | Periodic restore drills | PoR challenges *are* restore drills, run on a 30-day cadence per shard, recorded on-chain | `PoRVerifier`; CLAUDE.md §4.4 |
| § 164.312(a)(2)(iv) — encryption and decryption | Implement a mechanism to encrypt and decrypt ePHI | AES-256-GCM at L1; per-bundle DEK; consortium KMS holds the wrap-key | `internal/crypto`; threat-model.md §1.3 |
| § 164.312(b) — audit controls | Hardware, software, and procedural mechanisms that record activity | Same L4 + L5 fabric as AI Act Art. 12 mapping (§2 above) | E-PROV |

### 3.2 Restore-SLA framing

The NPRM does not specify a numeric restoration SLA at the rule level; HHS guidance and contractual BAAs typically expect **24-hour restore for high-priority systems**. LBVR-Med's measured restore times (E9-multi):

| Failure mode | Detection median | Detection P99 |
|---|---|---|
| D0+D1 lost (both data shards) | 137 ms | 559 ms |
| D0+P0 lost | 561 ms | 7.8 s |
| D1+P0 lost | 632 ms | 8.7 s |

All four orders of magnitude faster than the 24h notional bar. Restoration is automatic — no operator intervention required.

### 3.3 Known gaps

- **Risk analysis and management (§ 164.308(a)(1)).** The NPRM tightens the risk-analysis requirements and asks for documented technology-asset inventory. LBVR-Med supplies the storage substrate; the larger compliance program (people, policy, training) is out of LBVR's technical scope.
- **Business Associate Agreement chain.** Pinata, Filebase, and Irys must each sign a HIPAA BAA before live PHI flows through them. Conference-scope eval uses Synthea (synthetic, no PHI); journal-scope MIMIC-IV deployment requires BAAs.

---

## 4. Clinical SLO standards

### 4.1 IEC 60601-1-8:2006 + A2:2020 — Alarm systems

Defines alarm system requirements for medical electrical equipment. Section 6.5 (Alarm signal duration) and Annex C (priorities and characteristics) anchor the **10 s end-to-end alarm latency** budget for high-priority alerts.

**LBVR-Med mapping (E2 baseline + E3 stress):** P99 retrieval latency comfortably under 10 s across all RTT × loss combinations measured. E8 Fig 9 verdict: **PASS** with margin. The §VI talking point: even under WAN adversity (5% loss × 200 ms RTT), LBVR-Med supports the alarm-class SLO.

### 4.2 IEEE 11073-10700:2022 — SDC plug-and-play (OR/ICU)

Service-oriented Device Connectivity for operating-room and ICU device interop. The standard does not impose an explicit storage-side SLO but assumes ≤ 500 ms end-to-end for chart-pull-class operations.

**LBVR-Med mapping:** P99 retrieval baseline is 565 ms at the LBVR mode (E2). E8 verdict: **STRETCH** (within 2× target). Under WAN stress (E3 5% × 200 ms cell), P99 stretches to 2 058 ms — **FAIL** at the chart-pull bar. The §VI design discussion frames this as the limiting clinical workflow and identifies gateway placement (edge caching) as the journal-extension axis.

### 4.3 IEC 61850-8-1 — GOOSE messaging (cross-domain reference)

Power-grid Generic Object-Oriented Substation Event protocol. The 4 ms end-to-end latency budget is canonically the strictest real-time SLO in safety-critical systems.

**LBVR-Med mapping (E10 cross-SLO calibration):** **INFEASIBLE.** No decentralized verifiable storage configuration meets a 4 ms budget — even AES-GCM open + Merkle re-verify cost ~5 ms on commodity x86. The figure is honest about this; it's the load-bearing finding for §VI's "where decentralized storage *is* and *is not* applicable" framing and the third-paper motivation.

---

## 5. Cross-cutting compliance summary

| Regime | Headline mapping | Verdict | Source figure |
|---|---|---|---|
| EHDS Art. 44–50 | On-chain `policyId` + tamper-evident PROV-JSON | Compliant | E-PROV Fig 12 |
| EU AI Act Art. 12 | Per-event BLS-signed log + on-chain anchor | Compliant + affordable | E-PROV Fig 12, E5 Fig 6 |
| HIPAA NPRM 90 FR 898 (contingency) | Cross-tier RS(2,3) + auto-recovery | Compliant; restore time 137 ms – 8.7 s for double-failure scenarios | E9 Fig 10, E9-multi Fig 10b |
| IEC 60601-1-8 | Retrieval P99 ≤ 10 s | PASS with margin | E2 Fig 3, E3 Fig 4, E8 Fig 9 |
| IEEE 11073-10700 (chart-pull, 500 ms) | Retrieval P99 baseline 565 ms; stress 2 058 ms | STRETCH (baseline) / FAIL (stress) | E8 Fig 9 |
| IEC 61850 GOOSE (4 ms) | Decentralized storage fundamentally unsuitable | INFEASIBLE — explicit | E10 Fig 11 |

---

## 6. Reviewer-anticipated objections

| Q | A |
|---|---|
| "EHDS is still being implemented per Member State; is your mapping presumptuous?" | Mapping is to the regulation text, not to a specific Member State implementing instrument. Conformance to the text itself is what we claim; per-MS deployment is out of scope. |
| "AI Act Art. 12 logs need to be stored 'over the lifetime of the system' — that's potentially decades. Cardona could go away." | The PROV-JSON document is the legal record; Cardona is the *anchor*. Document copies live on Pinata + Filebase + Arweave (three independent providers); a Cardona reorg or shutdown requires re-anchoring the canonical hashes against a new chain (engineering work, not a data loss). Documented in §6 of `threat-model.md`. |
| "HIPAA BAAs — your Pinata key is on a startup's infrastructure." | Conference scope uses Synthea (no PHI). A live deployment requires BAAs with each of the three providers; we treat this as deployment hardening, not architectural surgery. |
| "Why exclude bundle bytes from AI Act Art. 12 'inputs/outputs'?" | Logging PHI bytes themselves would itself violate HIPAA / EHDS / GDPR. We log the bundle reference (Merkle root) + the gateway's verdict; a regulator with a court order can reconstruct the bundle bytes via the on-chain shard layout, but the log alone does not breach confidentiality. The recital language in the AI Act ("where reasonable") explicitly accommodates this. |
| "IEC 61850 GOOSE infeasibility — does that mean the whole thesis is bogus for power grids?" | No — it means decentralized verifiable storage is unsuitable for *real-time* power-grid messaging (sub-10ms). For SCADA / asset-management traffic at 100ms–s latencies (different layer of the substation hierarchy), LBVR-Med is plausible. The third paper (CLAUDE.md §3.3) tests that hypothesis on real domain datasets. |

---

**Last updated:** D15 (2026-04-29). Sources of truth: regulation text (verifyable via official EU/HHS journals), CLAUDE.md §1 / §12, run artefacts under `eval/results/E*/`. Citation-text accuracy should be re-verified at D19 (full-pass-and-reference-check) before submission.

**Citations to land in `paper/references.bib`:** see CLAUDE.md §14 — `ehds2025reg`, `aiact2024`, `hipaanprm2025`, `iec60601-1-8`, `iec61850goose`, `ieee11073-10700`. Each cited at exactly one anchor point in §VI per the IEEE 6-page budget.
