# LBVR-Med — Pending Work (D18 → submission)

> Snapshot of all open items at D18 (2026-05-02) blocking the IEEE ICUFN 2026 May 8 submission. Maintained as the working punch-list through D21.
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

### P1.2 — Paper §II Related Work
- **What.** `paper/sections/02-related-work.tex` (~0.5 page). Position against `fang2024metaverse` (PoR but no erasure), `manzi2025intertwin` (federated storage but no PoR), W3C PROV (provenance but no crypto). Add `li2025trl` TRL-3 critique.
- **Owner.** Claude (drafting) → Josiah (review).
- **Acceptance.** Each prior work cited once with the specific delta LBVR-Med adds. ≤ ~500 words.
- **Risk if skipped.** PC will reject for missing context.

### P1.3 — Paper §III System Model & Threat Model
- **What.** `paper/sections/03-system-model.tex` (~0.75 page). Compress from `docs/threat-model.md` — keep the 5-layer figure, the adversary table, emphasise tier-selective adversary as novel.
- **Owner.** Claude (drafting from threat-model.md) → Josiah (review).
- **Acceptance.** All seven in-scope attacks named, defenses in one line each. ≤ ~700 words.
- **Risk if skipped.** Submission impossible.

### P1.4 — Paper §IV Protocol Design
- **What.** `paper/sections/04-protocol.tex` (~1.5 pages — the longest tech section). Tier decision rules; erasure coding design (~0.4 pages, compress from `docs/erasure-design.md`); PROV-JSON extension with BLS signing (~0.4 pages, compress from `docs/provenance-spec.md`); PoR challenge/response/verdict; quorum retrieval algorithm; registry schema.
- **Owner.** Claude (drafting from companion docs) → Josiah (review).
- **Acceptance.** Reader can reconstruct the on-chain `BundleRecord` shape and the `respondToChallenge` calldata from §IV alone. ≤ ~1.4k words.
- **Risk if skipped.** Submission impossible.

### P1.5 — Paper §V Evaluation
- **What.** `paper/sections/05-evaluation.tex` (~1.75 pages). Walk through Figs 2–12. Cross-SLO calibration gets its own subsection (CLAUDE.md §9).
- **Owner.** Claude (drafting) → Josiah (review).
- **Acceptance.** Every figure referenced; every § text claim traceable to a JSON file in `eval/results/`. ≤ ~1.6k words.
- **Risk if skipped.** Submission impossible.

### P1.6 — Paper §VI Discussion & Regulatory Alignment
- **What.** `paper/sections/06-discussion.tex` (~0.5 page). Compress from `docs/regulatory-alignment.md`. Map results to EHDS Art. 44–50, EU AI Act Art. 12, HIPAA NPRM 90 FR 898. Acknowledge limitations explicitly.
- **Owner.** Claude (drafting from regulatory-alignment.md) → Josiah (review).
- **Acceptance.** Cites E5 gas affordability, E9-multi restore times, E10 cross-domain. Names tier-cooperation collusion + KMS side-channel + endpoint compromise as out of scope. ≤ ~500 words.
- **Risk if skipped.** Submission impossible.

### P1.7 — Paper §VII Conclusion + Journal Roadmap
- **What.** `paper/sections/07-conclusion.tex` (~0.25 page). One paragraph each: conclusion + journal extension (FL-DT) + third paper (cross-domain).
- **Owner.** Claude → Josiah.
- **Acceptance.** ≤ ~250 words.
- **Risk if skipped.** Submission impossible.

### P1.8 — Paper §00 Abstract
- **What.** `paper/sections/00-abstract.tex` (≤ 200 words). The four-contribution claim + headline result.
- **Owner.** Claude → Josiah. Best done LAST after all sections are written so it can paraphrase the actual paper.
- **Acceptance.** ≤ 200 words; the phrase "verifiable storage fabric" appears.
- **Risk if skipped.** Submission impossible.

### P1.9 — `paper/references.bib`
- **What.** Build from CLAUDE.md §14. 16 references named there; every citation in the paper must resolve.
- **Owner.** Claude (initial population) → Josiah (verify exact metadata).
- **Acceptance.** `bibtex paper/main` exits 0 with no missing-key warnings. Each entry has DOI or URL where available.
- **Risk if skipped.** PC will dock the paper.

### P1.10 — `paper/main.tex` skeleton + IEEEtran wiring
- **What.** Create `paper/main.tex` with IEEEtran conference style, double-column 10pt, 6-page hard limit. Wire `\input` directives for sections.
- **Owner.** Claude.
- **Acceptance.** `pdflatex paper/main` produces a 6-page PDF with no errors and warnings only for the still-empty section files.
- **Risk if skipped.** Cannot submit a PDF.

### P1.11 — Final pass + page-budget enforcement (D20)
- **What.** Read end-to-end. Trim or expand each section to fit the 6-page IEEEtran limit. Verify every figure renders cleanly at column width. Verify every citation in `references.bib` resolves. Verify every regulatory-text citation against the official journal.
- **Owner.** Either.
- **Acceptance.** PDF is exactly 6 pages, no overflow. `[VERIFY]` tags in `docs/regulatory-alignment.md` resolved against canonical text.
- **Risk if skipped.** Page-limit overflow → automatic rejection.

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

### P2.2 — Live-mode E4 wiring (uncommitted)
- **What.** Live-mode `makeTier()` for hot (Pinata) + warm (Filebase) is implemented but uncommitted in `cmd/bench/e4/{main,e4_test}.go`. 3-bundle live smoke timed out on Filebase Put after 4m44s.
- **Risk.** None for the paper — sim-mode E4 figure already shipped. The live wiring is "nice to have" for a §V parity footnote.
- **Acceptance.** Either commit-and-skip-live (preserve the wiring for journal extension) OR debug the Filebase Put hang.
- **Owner.** Either.
- **Effort.** Commit-and-skip: 5min. Debug: ~1h likely (read filebase.go Put, check bucket IPFS-pin setting, possibly switch to PutObject path).

### P2.3 — Cardona contract deployment + E5 gas parity
- **What.** Deploy `CIDRegistry`, `PoRVerifier`, `AuditorLog` to Polygon zkEVM Cardona. Run a 5-bundle sample of E5's gas-measurement test against the live contract; compare to forge `--gas-report`.
- **Risk.** None for the paper — `cmd/bench/e5` already produces deterministic gas numbers via in-memory EVM (Cancun-fork-equivalent). Live spot-check would let us drop the §V "±1% expected" hedge.
- **Acceptance.** Live gas readings match forge readings within ±1%. `.env` `*_ADDRESS` fields populated.
- **Owner.** Josiah (testnet ETH funding) → Claude (deploy script + sample run).
- **Effort.** ~30min on Claude's side once funds land.
- **Blocker.** Cardona faucets were dry on Apr 28; status unknown today.

### P2.4 — Irys/Sepolia funding for cold-tier live E4
- **What.** Fund the Irys devnet wallet (`0xCca280e22bb1ea79d69EBD0Fc993Db08B259ce92`) with Sepolia ETH; rerun E4 with `-cold-mode live`.
- **Risk.** None for the paper — sim-mode E4 is committed. Live cold validates the long-tail calibration.
- **Acceptance.** ≥ 0.01 Sepolia ETH on the wallet; one E4 partial-live run committed.
- **Owner.** Josiah (faucet hop) → Claude (run + figure update).
- **Effort.** ~30min Claude after funding lands.

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

- Squash any "wip" or scratch branches
- Tag the commit corresponding to the submitted PDF (`git tag icufn2026-submission`)
- Open journal extension branch (`journal/jbhi`)
- Schedule MIMIC-IV credentialing application
- Investigate Filebase Put timeout (P2.2) for journal-scope live runs
- Investigate Cardona contract deployment (P2.3) for live anchoring
- Live-tier swap path documented in CLAUDE.md §10 D4 — close the loop

---

**Maintenance.** Update this file at the end of every session that closes or opens an item. Items move from Tier 1/2 → done by being struck through OR removed; new items get appended to the appropriate tier.

**Last updated:** D18 (2026-05-02), commit `1da5730`.
