# LBVR-Med — Journal Roadmap (IEEE JBHI)

> **Target submission:** mid-June 2027 (~13-month runway from 2026-05-26).
> **Venue:** IEEE Journal of Biomedical and Health Informatics (Regular Paper).
> **Working branch:** `journal/jbhi` (off `main` at tag `icufn2026-submission`).
> **Conference paper status:** submitted to IEEE ICUFN 2026 on D21 (2026-05-08); see `docs/pending.md` for the closing punch-list.
>
> This file is the forward-looking analogue of `docs/pending.md`. Maintain at the end of every session that opens or closes a journal-extension item.

---

## Locked decisions

| Decision | Value | Source |
|---|---|---|
| Scope | Full CLAUDE.md §3.2 — DICOM workload, 5 tiers, FL retrieval-quorum attestation, C2PA profile | CLAUDE.md §3.2 |
| Target venue | IEEE JBHI Regular Paper | CLAUDE.md §3.2 |
| Target submission | **2027-06-15** (mid-June 2027) | Confirmed 2026-05-26 |
| Staffing | Solo (Josiah) | Confirmed 2026-05-26 |
| Phase E gate | Run E15 (Fang VC-PoR viability check) **before** committing to the full vector-commitment PoR upgrade | Confirmed 2026-05-26 |
| Working branch | `journal/jbhi` off `icufn2026-submission` tag | This doc |
| Erasure parameter (5-tier) | RS(3,5) — 3 data + 2 parity, one shard per tier; tolerates 2-tier failure | CLAUDE.md §3.2 implies, §4.5 leaves open |
| Provenance quorum | 3-of-N BLS (up from 2-of-N) | CLAUDE.md §4.6 noted as journal upgrade |

## Open decisions (resolve before the relevant phase begins)

| ID | Question | Needed by | Notes |
|---|---|---|---|
| **O1** | Filecoin FVM sealing latency policy: pre-seal corpus before benchmarks, OR measure sealing latency as a distinct metric and exclude from end-to-end retrieval CDFs? | Start of Phase B | Pre-seal keeps the recovery CDFs clean and comparable to the conference; measuring exposes a real production-deployment number. Recommendation: **measure as separate metric** (it's a novel data point) AND pre-seal for the recovery CDFs so they remain comparable. |
| ~~O2~~ | ~~MIMIC-IV credentialing — apply now or defer?~~ | ~~June 2026~~ | **RESOLVED 2026-05-26: apply now (Phase A6).** Cheap to start, expensive to skip later. |
| **O3** | DICOM corpus exact source: NIH ChestX-ray14 re-wrapped as synthetic DICOM (CLAUDE.md §3.2), or also pull TCIA imaging if convenient? | Start of Phase C | CLAUDE.md says ChestX-ray14. Keep to that unless TCIA is trivially additive. |
| **O4** | FL attestation primitive: standalone BLS quorum receipts, OR reuse the existing PROV-JSON signing layer with a new `attestation_type: "fl_consumption"`? | Start of Phase D | Recommendation: **reuse PROV-JSON layer**. Less new crypto code, interoperates with verifier. |
| **O5** | C2PA library choice: official C2PA Rust SDK with FFI, or pure-Go re-implementation of the manifest subset we need? | Start of Phase C | Rust SDK is canonical but adds FFI; pure-Go is lighter but spec drift risk. |

---

## Phase A — Branch & foundation (June 2026)

Closes conference debt so the journal work doesn't inherit it. Also de-risks every chain-touching experiment downstream.

| ID | Item | Owner | Effort | Status |
|---|---|---|---|---|
| A1 | Tag conference submission (`icufn2026-submission` at 011937f), open `journal/jbhi` branch | Either | 30 min | **✅ DONE 2026-05-26** — commits `f4ee65c` (camera-ready on main), `c6b44ae` (roadmap on journal/jbhi); tag + branch pushed to origin |
| A2 | **J5.1 — abigen-bound chain client** | Claude | done in ~3 h | **✅ DONE 2026-05-27** — commits `fb83d3f` (bindings + go-ethereum dep + Go 1.22→1.24 bump), `1ddb19a` (chain.go + anchor.go wiring + Cardona→Chain config rename), `feb3943` (integration test). Live PureChain smoke: anchor tx `0xaa8c158a18a234aa23b77b88cabd86dc7316bbf310aeec52d47b2ab8ee669a9c` at block 1,237,482, gasUsed=108,264 (matches paper §V-A spot-check exactly) |
| A3 | P2.1 — deterministic sim-tier under `t.Short()` (replace lognormal flake source) | Claude | 1–2 h | pending |
| A4 | P2.2 — Filebase Put timeout root-cause + fix | Claude | half-day | pending |
| A5 | P2.4 — Irys/Sepolia wallet funding | Josiah | 1 h + wait | pending |
| A6 | **MIMIC-IV credentialing application** (per O2 resolution) | Josiah | half-day to apply, ~3 months to land | pending |
| A7 | This doc + CLAUDE.md §3.2 pointer to it | Claude | 30 min | **✅ DONE 2026-05-26** — commit `c6b44ae` |

**Phase A exit criterion:** All conference debt closed; chain client live; `journal/jbhi` branch is the working trunk.

---

## Phase B — 5-tier substrate expansion (July–August 2026)

Most code-heavy phase. Adds Kubo (local) and Filecoin FVM as the two new tiers, parameterizes the erasure layer for RS(3,5), and rewires placement policy + on-chain schema.

| ID | Item | Owner | Effort | Acceptance |
|---|---|---|---|---|
| B1 | `internal/tiers/kubo/` — local Kubo node client matching existing tier interface | Claude | 2 days | Unit + integration tests; `Put/Get/Stat/Delete` parity with `pinata/filebase/arweave` clients |
| B2 | `internal/tiers/filecoin/` — Filecoin FVM client | Claude | 4–5 days | Same interface; resolves O1 (sealing latency policy) |
| B3 | Erasure parameterization `internal/erasure/` (k, n) — support RS(3,5) | Claude | 2 days | Existing RS(2,3) tests still pass; new RS(3,5) tests cover all 2-tier failure permutations |
| B4 | Placement policy revision `internal/policy/` for 5 tiers + RS(3,5): D0→Pinata, D1→Kubo, D2→Filebase, P0→Arweave, P1→Filecoin | Claude | 1 day | Layout decision documented in `docs/erasure-design.md` revision |
| B5 | `CIDRegistry.sol` schema bump: `ShardPlacement[3]` → variable-length `ShardPlacement[]` (v2 contract; migration script for any anchored conference bundles) | Claude | 2–3 days | Foundry tests pass; deployed to PureChain; abigen bindings regenerated; v1 contract still readable for archival |
| B6 | Revise `docs/erasure-design.md` for RS(3,5) — rationale, failure-mode matrix (single + double tier), placement decision | Claude | 1 day | Doc internally consistent with B3/B4 |

**Phase B exit criterion:** Full ingest pipeline produces RS(3,5) layouts across all 5 tiers; retrieval gateway can recover from any single OR double tier outage.

---

## Phase C — DICOM workload + C2PA profile (September 2026)

Independent of B's chain-side work but depends on B's tier interface. Adds the imaging half of the workload and the C2PA provenance profile for that half.

| ID | Item | Owner | Effort | Acceptance |
|---|---|---|---|---|
| C1 | NIH ChestX-ray14 acquisition + re-wrap script → synthetic DICOM under `eval/dicom/` | Josiah + Claude | 2 days | Corpus statistics (size distribution, count) logged like Synthea D2 validation |
| C2 | DICOM ingest path: verify chunked-Merkle + AES-GCM + erasure pass through unchanged for DICOM binaries | Claude | 1 day | Property test: round-trip integrity for DICOM headers AND pixel data |
| C3 | C2PA profile spec → `docs/c2pa-spec.md` | Claude | 2 days | Maps C2PA assertion model onto LBVR-Med BLS quorum sigs; preserves PROV-JSON for FHIR side; resolves O5 (Rust SDK vs pure-Go) |
| C4 | `internal/provenance/c2pa/` — C2PA manifest emission for DICOM retrievals | Claude | 3–4 days | Manifest validates against C2PA Public Validator; on-chain anchor parity with PROV-JSON path |
| C5 | Verifier extension: parse + validate C2PA manifests; cross-check against on-chain anchor | Claude | 2 days | Tampering tests analogous to E-PROV; ≥7 detection categories |

**Phase C exit criterion:** Mixed FHIR+DICOM ingest works end-to-end; FHIR retrievals emit PROV-JSON, DICOM retrievals emit C2PA, both verifiable.

---

## Phase D — FL retrieval-quorum attestation (October 2026)

The "verifiable retrieval-quorum attestation for federated inference" layer from CLAUDE.md §3.2. Conceptually closest to existing FL portfolio (XS-FedPRS / RV-FedPRS).

| ID | Item | Owner | Effort | Acceptance |
|---|---|---|---|---|
| D1 | Design doc `docs/fl-attestation-spec.md` — core proposal: `EHDSPolicy{bundle_class, permitted_purposes[]}` published on-chain; client emits `FLConsumptionReceipt{round_id, consumed_bundle_ids[], purpose, BLS_sig}` aggregated quorum-style | Claude | 2 days | Spec resolves O4 (standalone vs reuse PROV layer); diagrams cover happy path + tampering |
| D2 | EHDS policy schema + on-chain `PolicyRegistry.sol` | Claude | 2 days | Foundry tests; deployed to PureChain |
| D3 | Minimal FL client stub (Go) — drives retrievals + emits receipts; NOT a full FedAvg implementation | Claude | 3 days | Smoke test: 1 round, 3 clients, all receipts verify |
| D4 | `AuditorLog.sol` extension: `FLAttestation` event + verification | Claude | 1 day | Foundry tests; gas budget ≤80k per attestation |
| D5 | Verifier extension: validate attestation chains end-to-end + tamper-detection tests | Claude | 2 days | ≥5 attack categories detected (forged receipt, replayed receipt, off-policy bundle consumption, quorum-subthreshold, signer not in roster) |

**Phase D exit criterion:** FL client stub can prove it consumed only EHDS-permitted shards across multiple rounds; verifier rejects all five attack categories.

---

## Phase E — Cryptographic upgrades (November 2026)

Strengthens primitives ahead of the eval re-run so F measures the upgraded system. **Gated by E15 viability check.**

| ID | Item | Owner | Effort | Acceptance |
|---|---|---|---|---|
| E1 | **E15 viability check (run BEFORE swapping)** — implement minimal Fang 2024 VC-PoR prototype against 100 bundles; measure prove time, verify gas, on-chain footprint vs simplified Fang | Claude | 3–4 days | `eval/results/E15-viability/` JSON; written verdict in this doc |
| E2 | **Gate decision** based on E1: (a) prove time ≤2s AND verify gas ≤500k → proceed with full swap (E3+E4); (b) prove time >2s OR verify gas >500k → ship simplified Fang in the journal too, document VC-PoR as future work | Josiah | 1 h | Decision logged in this doc with rationale |
| E3 | (Conditional on E2=proceed) Full Fang 2024 VC-PoR — polynomial commitments over BLS12-381, replaces simplified Merkle+BLS scheme | Claude | 1 week | Round-trip tests pass; existing PoR tests still pass for backward compat |
| E4 | (Conditional on E2=proceed) `PoRVerifier.sol` upgrade for VC verification | Claude | 3 days | Foundry tests; deployed to PureChain |
| E5 | Provenance quorum: 2-of-N → 3-of-N (independent of E2 gate) | Claude | 1 day | Verifier rejects 2-of-N signed docs as below threshold |

**Phase E exit criterion:** E15 result documented; PoR upgrade either landed (with new gas/time numbers) or formally deferred to third paper with rationale.

---

## Phase F — Evaluation matrix expansion (December 2026 – February 2027)

Bulk of measurement effort. Re-runs conference experiments at the new configuration plus journal-specific additions. Three months gives slack for holiday season + experiment debugging.

### F1 — Re-runs of conference experiments under 5-tier RS(3,5) + upgraded crypto

| Exp | What changes vs conference | Expected runtime |
|---|---|---|
| E1 | Add Kubo + Filecoin to tier sweep | ~8h |
| E2 | 5-tier retrieval CDF + new baselines (consider adding Storj-tardigrade, IPFS Cluster) | ~10h |
| E3 | Same WAN matrix, 5 tiers, RS(3,5) | ~12h |
| E5 | New gas numbers if E2 swap landed; otherwise unchanged | ~2h |
| E6 / E6b | Byzantine sweep over 5 tiers | ~10h |
| E9 / E9-multi | RS(3,5) recovery — now includes double-tier failure modes (previously failure modes) | ~5h |
| E10 | Cross-SLO re-overlay with new E3 data | ~30min post-processing |
| E-PROV | 3-of-N quorum overhead | ~3h |

### F2 — New experiments

| ID | Experiment | Output | Effort |
|---|---|---|---|
| E11 | DICOM ingest/retrieval throughput vs FHIR (corpus comparison) | Fig: ingest/retrieval throughput by corpus type | ~4h |
| E12 | 5-tier RS(3,5) recovery CDF — single AND double-tier failure modes | Fig: recovery CDF expansion | ~4h |
| E13 | FL attestation overhead per training round (1, 3, 10 clients) | Fig: per-round overhead + Table: receipt sizes | ~3h |
| E14 | C2PA manifest overhead vs PROV-JSON (per DICOM retrieval) | Table: side-by-side | ~2h |
| E15 | Already run in Phase E1 — figure renders here | Fig: VC-PoR vs simplified | post-processing |
| E16 | Live-mode E4 across all 5 tiers (closes P2.4) | Fig: TTA per tier | ~6h |

### F3 — Cross-domain hooks

Lightweight data captures that prime the third (cross-domain) paper without doing its full eval. Specifically: instrument the gateway to emit SLO-domain tags so the same E3 data can be re-analyzed per regime without rerunning.

**Phase F exit criterion:** All E* JSON committed under `eval/results/E*/`; all figures render at journal column width.

---

## Phase G — JBHI paper writing (March – early May 2027)

| ID | Item | Owner | Effort |
|---|---|---|---|
| G1 | Audit conference paper for keep/extend/replace. Most likely outcome: §III/§IV expand significantly; §V wholly replaced; §VI extended with cross-domain validation hooks | Either | 2 days |
| G2 | IEEE JBHI format wiring (~10–12 pages typical for Regular Papers, IEEEtran transmag style) | Claude | 1 day |
| G3 | Section drafts | Claude (drafts) → Josiah (review) | ~4 weeks |
| G4 | Re-positioning vs Fang 2024 — since we either implement the full VC scheme (E2=proceed) or document why we stopped short (E2=defer), the framing changes accordingly | Claude | 1 day |
| G5 | Figure pass at journal column width; ensure E-series figures from F are consistent in style | Claude | 3 days |
| G6 | Internal review rounds — at least 2 full passes | Josiah | 1 week each |
| G7 | Pre-submission verification gate (analogous to `docs/pending.md` Tier 4) | Either | 1 day |

**Phase G exit criterion:** Submission-ready PDF; all claims traceable to `eval/results/`; all citations resolve.

---

## Phase H — Submission + buffer (mid-May – mid-June 2027)

| ID | Item | Owner | Effort |
|---|---|---|---|
| H1 | ScholarOne submission setup (account, manuscript metadata, suggested reviewers, cover letter) | Josiah | 1 day |
| H2 | Submit | Josiah | 1 day |
| H3 | Buffer for unforeseen polish / reviewer comments triggered by self-review | — | ~3 weeks |

**Target submission: 2027-06-15.**

---

## Calendar at a glance

| Month | Phase | Major milestone |
|---|---|---|
| Jun 2026 | A | Chain client live; MIMIC-IV applied; conference debt closed |
| Jul–Aug 2026 | B | 5-tier substrate live; RS(3,5) recovery proven |
| Sep 2026 | C | DICOM + C2PA path working |
| Oct 2026 | D | FL attestation working end-to-end |
| Nov 2026 | E | E15 viability run; PoR upgrade decision made |
| Dec 2026 – Feb 2027 | F | Eval matrix completes (re-runs + new experiments) |
| Mar – early May 2027 | G | JBHI paper drafted + reviewed |
| Mid-May – mid-Jun 2027 | H | Submitted |

---

## Risk register

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Filecoin FVM sealing latency makes Phase B experiments awkward | High | Medium | O1 decision — measure as separate metric, pre-seal for recovery CDFs |
| MIMIC-IV credentialing rejected or delayed past Phase F | Medium | Low | Synthea + DICOM is sufficient for journal; MIMIC is third-paper gravy |
| E15 viability check shows VC-PoR is impractical | Medium | Medium | Phase E gate exists exactly for this; document as "simplified Fang sufficient for clinical SLOs, VC-PoR is future work" |
| Solo execution + scope creep → schedule slip | High | High | This doc is the scope ceiling. Any new contribution requires explicit promotion from "third paper / future work" to a tracked phase here. |
| C2PA Rust SDK FFI quirks block C5 | Low | Low | Fall back to pure-Go subset implementation per O5 |
| Eval re-runs take longer than expected (compounded over 5 tiers) | Medium | Medium | Phase F has 3-month budget specifically to absorb this |

---

## Maintenance

Update this doc at the end of every session that closes or opens an item. Items move from `pending` to done by being struck through OR moved to a Closed section. New items get appended to the appropriate phase. If a new contribution candidate emerges, it goes into Risk Register's "scope creep" mitigation flow — explicit promotion required.

**Last updated:** 2026-05-27 (Phase A1 + A2 complete; A3–A6 still open).
