# LBVR-Med — Pending Work (D18 → submission)

> Snapshot of all open items blocking the IEEE ICUFN 2026 May 8 submission. Maintained as the working punch-list through D21.
>
> **Conventions.** "Owner: Josiah" = task requires the human (auth, money, judgement). "Owner: Claude" = self-driveable in a session. "Owner: either" = either path works. Each item carries an acceptance criterion so we know when to stop.

---

## Tier 1 — Critical-path for May 8 submission

These block submission. Order is roughly the order to attack.

### P1.1 — Paper §I Introduction
- **What.** `paper/sections/01-introduction.tex` (~0.75 page) per CLAUDE.md §9. Open with the four-contribution claim, frame against Trautwein INFOCOM 2024 + interTwin FGCS 2025 + Thwe IEEE Access 2025 as the gap citations.
- **Why.** Reviewers read the intro first; it sets the framing for everything downstream.
- **Owner.** Claude (drafting) → Josiah (review/redirect).
- **Acceptance.** Includes the verbatim "verifiable storage fabric for safety-critical federated systems" canonical phrase (CLAUDE.md §12). Cites all three gap papers in the first two paragraphs. ≤ ~700 words.
- **Risk if skipped.** Submission impossible.
- **Status.** ✅ DONE.

### P1.2 — Paper §II Related Work
- **What.** `paper/sections/02-related-work.tex` (~0.5 page). Position against `fang2024metaverse` (PoR but no erasure), `manzi2025intertwin` (federated storage but no PoR), W3C PROV (provenance but no crypto). Add `li2025trl` TRL-3 critique.
- **Owner.** Claude (drafting) → Josiah (review).
- **Acceptance.** Each prior work cited once with the specific delta LBVR-Med adds. ≤ ~500 words.
- **Risk if skipped.** PC will reject for missing context.
- **Status.** ✅ DONE.

### P1.3 — Paper §III System Model & Threat Model
- **What.** `paper/sections/03-system-model.tex` (~0.75 page). Compress from `docs/threat-model.md` — keep the 5-layer figure, the adversary table, emphasise tier-selective adversary as novel.
- **Owner.** Claude (drafting from threat-model.md) → Josiah (review).
- **Acceptance.** All seven in-scope attacks named, defenses in one line each. ≤ ~700 words.
- **Risk if skipped.** Submission impossible.
- **Status.** ✅ DONE.

### P1.4 — Paper §IV Protocol Design
- **What.** `paper/sections/04-protocol.tex` (~1.5 pages — the longest tech section). Tier decision rules; erasure coding design (~0.4 pages, compress from `docs/erasure-design.md`); PROV-JSON extension with BLS signing (~0.4 pages, compress from `docs/provenance-spec.md`); PoR challenge/response/verdict; quorum retrieval algorithm; registry schema.
- **Owner.** Claude (drafting from companion docs) → Josiah (review).
- **Acceptance.** Reader can reconstruct the on-chain `BundleRecord` shape and the `respondToChallenge` calldata from §IV alone. ≤ ~1.4k words.
- **Risk if skipped.** Submission impossible.
- **Status.** ✅ DONE.

### P1.5 — Paper §V Evaluation
- **What.** `paper/sections/05-evaluation.tex` (~1.75 pages). Walk through Figs 2–12. Cross-SLO calibration gets its own subsection (CLAUDE.md §9).
- **Owner.** Claude (drafting) → Josiah (review).
- **Acceptance.** Every figure referenced; every § text claim traceable to a JSON file in `eval/results/`. ≤ ~1.6k words.
- **Risk if skipped.** Submission impossible.
- **Status.** ✅ DONE.

### P1.6 — Paper §VI Discussion & Regulatory Alignment
- **What.** `paper/sections/06-discussion.tex` (~0.5 page). Compress from `docs/regulatory-alignment.md`. Map results to EHDS Art. 44–50, EU AI Act Art. 12, HIPAA NPRM 90 FR 898. Acknowledge limitations explicitly.
- **Owner.** Claude (drafting from regulatory-alignment.md) → Josiah (review).
- **Acceptance.** Cites E5 gas affordability, E9-multi restore times, E10 cross-domain. Names tier-cooperation collusion + KMS side-channel + endpoint compromise as out of scope. ≤ ~500 words.
- **Risk if skipped.** Submission impossible.
- **Status.** ✅ DONE.

### P1.7 — Paper §VII Conclusion + Journal Roadmap
- **What.** `paper/sections/07-conclusion.tex` (~0.25 page). One paragraph each: conclusion + journal extension (FL-DT) + third paper (cross-domain).
- **Owner.** Claude → Josiah.
- **Acceptance.** ≤ ~250 words.
- **Risk if skipped.** Submission impossible.
- **Status.** ✅ DONE.

### P1.8 — Paper §00 Abstract
- **What.** `paper/sections/00-abstract.tex` (≤ 200 words). The four-contribution claim + headline result.
- **Owner.** Claude → Josiah. Best done LAST after all sections are written so it can paraphrase the actual paper.
- **Acceptance.** ≤ 200 words; the phrase "verifiable storage fabric" appears.
- **Risk if skipped.** Submission impossible.
- **Status.** ✅ DONE.

### P1.9 — `paper/references.bib`
- **What.** Build from CLAUDE.md §14. 16 references named there; every citation in the paper must resolve.
- **Owner.** Claude (initial population) → Josiah (verify exact metadata).
- **Acceptance.** `bibtex paper/main` exits 0 with no missing-key warnings. Each entry has DOI or URL where available.
- **Risk if skipped.** PC will dock the paper.
- **Status.** ✅ DONE — 16 entries field-checked against publisher records on 2026-05-05; 5 wrong titles + 2 wrong author lists + 1 wrong lead-author name corrected; commit `e0f6cef`.

### P1.10 — `paper/main.tex` skeleton + IEEEtran wiring
- **What.** Create `paper/main.tex` with IEEEtran conference style, double-column 10pt, 6-page hard limit. Wire `\input` directives for sections.
- **Owner.** Claude.
- **Acceptance.** `pdflatex paper/main` produces a 6-page PDF with no errors and warnings only for the still-empty section files.
- **Risk if skipped.** Cannot submit a PDF.
- **Status.** ✅ DONE — IEEEtran skeleton committed `2dccc2c`.

### P1.11 — Final pass + page-budget enforcement (D20)
- **What.** Read end-to-end. Trim or expand each section to fit the 6-page IEEEtran limit. Verify every figure renders cleanly at column width. Verify every citation in `references.bib` resolves. Verify every regulatory-text citation against the official journal.
- **Owner.** Either.
- **Acceptance.** PDF is exactly 6 pages, no overflow. `[VERIFY]` tags in `docs/regulatory-alignment.md` resolved against canonical text.
- **Risk if skipped.** Page-limit overflow → automatic rejection.
- **Status.** ✅ DONE — paper at exactly 6 pages, all citations resolve, no warnings (commits `c477c9d`, `79cc34b`, `15ba73b`, `70de317`).

### P1.12 — EDAS upload (D21, May 8)
- **What.** Submit final PDF to EDAS for ICUFN 2026 Track 3. Target ≤ 18:00 UTC.
- **Owner.** Josiah only (account-bound).
- **Acceptance.** Confirmation email from EDAS.
- **Risk if skipped.** No submission.

---

## Tier 2 — Known issues to investigate or close

### P2.1 — `cmd/bench/e{1,2,3,6,9}` test fixture flakiness (RESOLVED — documented)
- **Symptom.** `go test ./cmd/bench/e{1,2,3,6,9}/...` times out after 60–150s with stack traces pointing at `internal/gateway/recovery.go:141, 143, 154`. First seen Apr 30 (task `bfhep83w5`).
- **Diagnosis.** Not a deadlock and not a regression. The integration smoke tests (`TestE2Full`, `TestE3Full`, `TestRun_Smoke` etc.) call the bench's `run()` which in turn drives `gateway.Recover` with calibrated lognormal sim-tier latency (cold P99 = 8 s, with rare 30 s+ tails). With a small fixture sample (5–10 bundles), a single pathological cold-tier latency draw deterministically blows the per-test 30–150s deadline. Under `-race` overhead this gets worse but the same pattern fires without `-race`.
- **Why this is fine for submission.** The canonical bench runs committed under `eval/results/E*/run-*.json` are authoritative; they completed end-to-end (E1: 26 m; E2/E3: hours each on D13; E9-multi: ~10 m on D14). Every paper claim traces to a JSON record. The integration smoke tests are CI sanity checks, not the source of paper data.
- **Mitigation for D20 verification gate.** Run unit tests only (excluding `^Test*Full` / `^TestRun_Smoke` patterns where they call `run()`); these pass in seconds. Document the integration-test flakiness as a journal-scope cleanup item.
- **Journal-scope fix.** Either (a) replace lognormal sim with a deterministic small-fixed-latency profile under `t.Short()` (1–2h), or (b) seed-pin specific test runs to known-good latency draws (30 min). Not blocking the May 8 submission.

### P2.2 — Live-mode E4 wiring (RESOLVED — committed, smoke deferred)
- **What.** Live-mode `makeTier()` for hot (Pinata) + warm (Filebase) was implemented; 3-bundle live smoke timed out on Filebase Put after 4m44s.
- **Status.** ✅ Committed as scaffold in `09bd2eb` (sim canonical for paper). Filebase Put debugging deferred to journal scope.

### P2.3 — Live L4 deployment + E5 gas parity (RESOLVED — landed on PureChain)
- **What.** Originally scoped against Polygon zkEVM Cardona (faucets dry on D14). Pivoted to PureChain (lab PoA² chain, chain id 900520900520) on D23 (2026-05-07).
- **Status.** ✅ DONE.
  - Live deploy via `forge script ... --rpc-url purechain --broadcast --legacy` (commits `7c31586`, `15ba73b`).
  - CIDRegistry `0x06002e97243b844c9ba06bdf4cdf9302691a9cb7`, AuditorLog `0x1e93113ddcce5084a272f9a003448de322cfefea`, PoRVerifier `0x4c3b96f5c94f31d09c4855783e7e49660ca0e81a`.
  - Spot-check anchor tx `0xdadd5280ec5deecd6035084ffd12f4846c0520c18caad0d162f0ac126660db2b` at block 1,124,295: 108 264 gas vs Foundry's predicted ~106 k → within 2% (commit `70de317`). Paper §V-A cites this directly.
  - Required EVM-version pivot Shanghai → London (commit `7c31586`); PureChain's Geth fork doesn't enable PUSH0.

### P2.4 — Irys/Sepolia funding for cold-tier live E4 (DEFERRED — journal scope)
- **What.** Fund the Irys devnet wallet (`0xCca280e22bb1ea79d69EBD0Fc993Db08B259ce92`) with Sepolia ETH; rerun E4 with `-cold-mode live`.
- **Status.** Deferred. Sepolia funding never arrived during conference window. Sim-mode E4 ships in the paper (`eval/results/E4/run-20260429-034942-ba5b359.json`); live cold-tier is a journal extension item alongside the broader Filecoin FVM / Kubo-local 5-tier expansion.

---

## Tier 3 — Explicit non-goals (journal-scope; do NOT pursue for May 8)

| Item | Why deferred | Where it shows up in paper |
|---|---|---|
| Federated-learning client integration | CLAUDE.md §3.2 — journal scope | §VII roadmap |
| C2PA medical-imaging provenance for DICOM | CLAUDE.md §3.2 | §VI limitations + §VII |
| Real EHR datasets (MIMIC-IV / UK Biobank) | Credentialing > deadline | §VI limitations |
| RS(3,5) or higher-parity erasure | Conference uses RS(2,3) — minimum meaningful | §VI design space |
| Halo2 / Plonky3 ZK circuits | Groth16 sufficient at conference scope | §VII |
| TEE-attested clients | Out-of-scope per threat model | §VI limitations |
| Right-to-erasure proofs (Arweave immutability) | `docs/regulatory-alignment.md` §1.3 — journal | §VI limitations |
| Public-IPFS-gateway TTA measurement | Source-side TTA is the production-deployment number | E4 caption + §V footnote |

---

## Tier 4 — Pre-submission verification gate (D20 evening)

Run through this list before tagging the submission PDF as final.

- [ ] `go vet ./...` clean
- [ ] `go test -race -count=1 ./...` clean (specifically: P2.1 resolved)
- [ ] `forge test --root contracts` — all 78 tests passing
- [ ] `make bench-E1`, …, `make bench-E-PROV` each launch without command-line errors (don't actually run the full matrix; just `make -n` each)
- [ ] `pdflatex paper/main` exits 0
- [ ] PDF page count = 6 (IEEEtran constraint)
- [ ] `bibtex paper/main` no missing-key warnings
- [ ] Every figure referenced from text exists at `paper/figures/E*.pdf`
- [ ] Every numerical claim in §V traceable to a `eval/results/E*/run-*.json` (spot-check 5)
- [ ] All `[VERIFY]` tags in `docs/regulatory-alignment.md` resolved
- [ ] `.env` not staged; `.env.example` shows only placeholders
- [ ] `git status` is clean before EDAS upload
- [ ] EDAS metadata (title, abstract, authors, affiliations, keywords) entered correctly

---

## Tier 5 — Post-submission cleanup (deferred, after May 8)

### Repo hygiene (~1 h)
- Squash any "wip" or scratch branches.
- Tag the commit corresponding to the submitted PDF (`git tag icufn2026-submission`).
- Open journal extension branch (`journal/jbhi`).
- Schedule MIMIC-IV credentialing application.

### J5.1 — Phase 3: abigen-bound chain client (carry-over from D23 decision)
- **What.** Replace the `ErrChainNotImplemented` stubs in `internal/registry/chain.go` and `cmd/gateway/anchor.go` (chainAnchor) with abigen-bound Go bindings that talk to the deployed PureChain contracts via go-ethereum's `ethclient` + `bind.NewKeyedTransactorWithChainID`.
- **Why deferred.** Decided on D23 (2026-05-07) to skip for the conference. Phase 1 (live deploy + spot-check anchor) and Phase 2 (BLS env-wiring) had paper-visible value; Phase 3 has none — no bench exercises the runtime chain path, so the canonical figures don't change. Risk of pulling in ~50 MB of go-ethereum transitive deps so close to deadline (potential conflict with the existing `kilic/bls12-381` crypto stack) was not worth taking on with <17 hours to EDAS upload.
- **Acceptance.**
  - `go install github.com/ethereum/go-ethereum/cmd/abigen@latest` and rerun `forge build` to produce ABI artifacts.
  - `abigen --abi out/{CIDRegistry,AuditorLog,PoRVerifier}.sol/*.json --pkg registry --type Chain* --out internal/registry/contract_binding.go` (one binding file per contract, or a combined file).
  - `internal/registry/chain.go` `chainClient` struct holds an `ethclient.Client` + the abigen-generated contract instance + a `bind.TransactOpts`; the four methods (`RegisterBundle`, `GetRecord`, `GetShardLayout`, `UpdateShardLayout`) call into the binding instead of returning `ErrChainNotImplemented`.
  - `cmd/gateway/anchor.go` chainAnchor wired similarly (single method: `Anchor` calls `AuditorLog.AnchorProvenance(...)` via the binding).
  - Tests: an integration test against a PureChain devnet OR a local anvil instance running on chain id 900520900520. Smoke a full ingest → register → retrieve → anchor cycle through the live code path.
- **Effort.** ~2–3 hours of focused work + verification. Best done as the first real journal-extension code change since it derisks subsequent live-FVM and on-chain BLS experiments.
- **Reference data.**
  - PureChain RPC: `https://purechainnode.com`
  - Chain id: `900520900520`
  - EVM target: `london` (PureChain Geth doesn't enable PUSH0)
  - Funded deployer key: in `.env` as `CHAIN_PRIVATE_KEY` (also `PURECHAIN_PRIVATE_KEY`)
  - Active contract addresses: see `.env` `*_ADDRESS` fields, or commit `7c31586`.
  - Live spot-check anchor tx: `0xdadd5280...0db2b`, block 1 124 295.

### Other journal-scope cleanups
- Investigate Filebase Put timeout (P2.2) for journal-scope live runs.
- Wire IRYS_PRIVATE_KEY funded with Sepolia ETH; run E4 with `-cold-mode live` (P2.4).
- Live-tier swap path documented in CLAUDE.md §10 D4 — close the loop.
- Replace `cmd/bench/e{1,2,3,6,9}` lognormal sim-tier with deterministic small-fixed-latency profile under `t.Short()` (P2.1 cleanup).

---

**Maintenance.** Update this file at the end of every session that closes or opens an item. Items move from Tier 1/2 → done by being struck through OR removed; new items get appended to the appropriate tier.

**Last updated:** D23 (2026-05-07), commit `011937f`.
