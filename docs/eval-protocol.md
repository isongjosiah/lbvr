# Evaluation protocol

How to reproduce every figure in the LBVR-Med paper. CLAUDE.md §8 is the
authoritative experiment table; this doc is the operator runbook. Each
experiment lands in `eval/results/E{n}/` with raw JSON + an `env.json`
fingerprint per CLAUDE.md §8 ("environment fingerprint").

## 0. Prerequisites

- 100K Synthea corpus generated and validated. Lives at
  `eval/synthea/upstream/output-100000/fhir/` (symlinked to `/mnt/d/...`
  per the §3.1 disk-budget note). Validation artefacts in
  `eval/results/synthea-100000/`.
- `.env` populated with Pinata + Filebase + Irys + Cardona credentials.
- Foundry toolchain installed (`foundryup`); `forge build` produces the
  ABI artefact for the registry's Go bindings.
- Cardona faucet has funded the deployer key.
- Toxiproxy daemon running on the loopback interface for fault injection
  (CLAUDE.md §13: `eval/toxiproxy/*.yaml` configs).

## 1. Naming + reproducibility contract

- Each `make bench-E{n}` writes raw JSON under `eval/results/E{n}/`
  named `run-<UTC-timestamp>-<short-commit>.json`.
- Each run also writes `eval/results/E{n}/env.json` matching CLAUDE.md
  §8 schema (`commit_hash, go_version, os, kernel, cpu, network_path,
  wall_start, wall_end`).
- Post-processing scripts in `eval/scripts/` read the raw JSON and emit
  PDFs into `paper/figures/`. Scripts must be idempotent: re-running on
  the same raw data must produce byte-identical figures.

## 2. Experiment matrix

| ID | Make target | Raw-data shape | Post-processing | Figure |
|---|---|---|---|---|
| E1 | `bench-E1` | `{tier, corpus_size, throughput_mbps, latency_p50/95/99}` per (tier × size) | `eval/scripts/throughput_bars.py` | Fig. 2 — ingest throughput |
| E2 | `bench-E2` | per-request `{tier, latency_ns}`, ≥10K samples per tier | `eval/scripts/generate_cdf.py` | Fig. 3 — retrieval latency CDF |
| E3 | `bench-E3` | per-request `{rtt, loss, p50, p95, p99}` per condition | `eval/scripts/heatmap.py` | Fig. 4 — RTT × loss heatmap |
| E4 | `bench-E4` | per-PUT `{tier, t0, t_global_reachable}` | `eval/scripts/availability_curve.py` | Fig. 5 — availability vs time |
| E5 | `bench-E5` | per-bundle `{size, prove_ms, verify_ms, gas}` | `eval/scripts/por_cost_bars.py` | Fig. 6 — PoR cost |
| E6 | `bench-E6` | `{pct_malicious, success_rate}` over 5 conditions | `eval/scripts/byzantine_curve.py` | Fig. 7a |
| E6b | `bench-E6b` | tier-selective adversary (per CLAUDE.md threat-model row) | `eval/scripts/detection_gap.py` | Fig. 7b |
| E7 | `bench-E7` | analytical (no benchmark — pure cost model) | `eval/scripts/storage_cost_bars.py` | Fig. 8 |
| E8 | `bench-E8` | derived from E1–E4 (no separate run) | `eval/scripts/slo_attainment_table.py` | Fig. 9 |
| E9 | `bench-E9` | per-bundle `{mode, paddedLen, t_baseline, t_recover_D0/D1/P0, t_detect_P0_dead}` | `eval/scripts/erasure_recovery_cdf.py` | Fig. 10 |
| E9-multi | `bench-E9-multi` | per-bundle `{shards_lost, t_detect_failure}` | `eval/scripts/double_failure.py` | Fig. 10b |
| E10 | `bench-E10` | post-processing of E3 raw data overlaid against SLO thresholds | `eval/scripts/slo_calibration.py` | Fig. 11 |
| E-PROV | `bench-E-PROV` | per-retrieval `{t_gen, t_canon, t_sign, t_anchor, anchor_gas, t_verify}` + tamper-detection rates | `eval/scripts/prov_overhead.py` | Fig. 12 + Table 2 |

## 3. Open decisions (must resolve before listed deadline)

- **E9 sample composition (deadline: D9 per CLAUDE.md §4.5)**:
  - **Option A — stratified to §4.5 bands**: 10 small <500 KB + 60 medium
    500 KB–2 MB + 30 large 2–5 MB. Clean figures, all bundles inside the
    illustrative-arithmetic band, but excludes the measured 11.3% tail.
  - **Option B — proportional to measured distribution**: sample
    proportionally from the 100K validated bundles, including the ≥ 5 MB
    tail up to ~98 MB. Honest representation; tail-bundles stress the
    cold-tier recovery path more than median.
  - Status: **open**. Once chosen, document the rationale here and
    reference it from the §V evaluation discussion.

- **Cold-tier upload mechanism for E9 (deadline: D9)**:
  - `internal/tiers/arweave.Put` is currently a documented stub (D4
    agent report). Three options for E9:
    - Wire `goar` (~50 MB transitive deps via go-ethereum); produces real
      Arweave tx ids and real upload latency.
    - Hand-roll ANS-104 data-item construction + Ed25519 signing;
      smaller dep footprint but ~300 lines of new crypto code.
    - Drive the cold tier in E9 via Toxiproxy stubs that mimic Irys
      latency/availability distributions (measured upstream); does not
      produce real Arweave artefacts but isolates the storage-fabric
      properties under measurement.
  - Status: **open**. Lean (per Claude session 2026-04-25 discussion):
    Toxiproxy stub for E9, real Irys deferred to journal extension.

## 4. Standard pre-run checklist

For every experiment:

1. `git status` — must be clean. The commit hash goes into `env.json`.
2. `go test ./... -race -count=1` — must be green.
3. `forge test --root contracts` — must be green if the experiment touches
   PoR or the registry.
4. `make synthea-status-100k` — confirm corpus is validated.
5. Toxiproxy daemon up if the experiment names a `eval/toxiproxy/*.yaml`
   config.
6. Note the wall-clock start; confirm no other workload on the host
   (CLAUDE.md §13: "Never run eval experiments on a laptop with other
   workloads").

## 5. Per-experiment runbooks

### E1 — Ingest throughput

> *To be written when `cmd/bench/E1.go` lands (D12 per §10).*

Skeleton command:
```
make bench-E1 CORPUS=eval/synthea/upstream/output-100000/fhir CONCURRENCY=8
```

[…remaining experiments stubbed; runbooks land alongside each
`cmd/bench/E{n}.go` implementation in D12-D14…]
