#!/usr/bin/env python3
"""E9-multi — double-tier failure detection latency.

Reads `eval/results/E9-multi/run-*.json` (output of `cmd/bench/e9
-modes multi`) and plots the per-mode failure-detection latency CDFs.
The contrast between modes is the headline: when both data shards
fail (D0D1_lost) the recovery state machine's early-fail fires once
two errors arrive (~100-200ms); when one data shard plus parity fail
(D0P0_lost / D1P0_lost) detection waits on the cold-tier error
return (~500ms median, ~8s P99 tail).

Companion to eval/scripts/erasure_recovery_cdf.py (Fig 10) — produces
Fig 10b per CLAUDE.md §8 row E9-multi.
"""
from __future__ import annotations
import json
import sys
from pathlib import Path

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np

MODE_STYLE = {
    "D0D1_lost": {"colour": "#1a9850", "linewidth": 2.4, "linestyle": "-",
                  "label": "D0+D1 lost (both data shards) — early-fail"},
    "D0P0_lost": {"colour": "#e67e22", "linewidth": 2.0, "linestyle": "-",
                  "label": "D0+P0 lost (hot + parity) — waits for cold error"},
    "D1P0_lost": {"colour": "#c0392b", "linewidth": 2.0, "linestyle": "--",
                  "label": "D1+P0 lost (warm + parity) — waits for cold error"},
}
MODE_ORDER = ["D0D1_lost", "D0P0_lost", "D1P0_lost"]


def load(path: Path) -> dict:
    return json.loads(path.read_text())


def newest(pattern: str, root: Path) -> Path:
    cands = sorted(root.glob(pattern))
    if not cands:
        sys.exit(f"no files matching {pattern} under {root}")
    return cands[-1]


def main() -> int:
    repo = Path(__file__).resolve().parent.parent.parent
    src = newest("run-*.json", repo / "eval" / "results" / "E9-multi")
    print(f"E9-multi source: {src.relative_to(repo)}")

    doc = load(src)
    n_bundles = doc["config"]["num_bundles"]
    reps = doc["config"]["reps_per_mode"]
    slo_ms = doc["config"]["slo_budget_ms"]

    # Flatten per-mode latencies in ms across all (bundle, rep) pairs.
    per_mode: dict[str, np.ndarray] = {}
    for mode in MODE_ORDER:
        latencies = []
        for sample in doc["samples"]:
            runs = sample["mode_runs"].get(mode, [])
            for r in runs:
                latencies.append(r["latency_ns"] / 1e6)
        if latencies:
            per_mode[mode] = np.array(latencies, dtype=np.float64)

    if not per_mode:
        sys.exit("no E9-multi modes found in run JSON")

    # Print per-mode percentiles.
    print()
    print(f"  {'mode':<14} {'P50_ms':>10} {'P95_ms':>10} {'P99_ms':>10} {'max_ms':>10} {'N':>6}")
    print("  " + "-" * 64)
    for mode in MODE_ORDER:
        if mode not in per_mode:
            continue
        a = per_mode[mode]
        print(f"  {mode:<14} {np.percentile(a,50):>10.1f} {np.percentile(a,95):>10.1f} "
              f"{np.percentile(a,99):>10.1f} {a.max():>10.1f} {a.size:>6}")

    # --- Figure: overlaid CDFs on log-x. ---
    fig, ax = plt.subplots(figsize=(8, 4.5), dpi=150)
    for mode in MODE_ORDER:
        if mode not in per_mode:
            continue
        a = np.sort(per_mode[mode])
        y = np.arange(1, a.size + 1) / a.size
        st = MODE_STYLE[mode]
        ax.plot(a, y, label=st["label"], color=st["colour"],
                linewidth=st["linewidth"], linestyle=st["linestyle"])

    ax.set_xscale("log")
    ax.set_xlabel("Failure-detection latency (ms, log scale)", fontsize=10)
    ax.set_ylabel("CDF", fontsize=10)
    ax.set_title(
        f"Double-tier failure detection — {n_bundles} bundles × {reps} reps per mode",
        fontsize=10,
    )
    ax.grid(True, alpha=0.3, which="both")
    ax.axvline(slo_ms, linestyle=":", color="black", linewidth=1.0,
               label=f"SLO budget ({slo_ms} ms)")
    ax.legend(loc="upper left", fontsize=8, framealpha=0.95)
    fig.tight_layout()

    pdf = repo / "paper" / "figures" / "E9_multi_double_failure_cdf.pdf"
    png = repo / "paper" / "figures" / "E9_multi_double_failure_cdf.png"
    fig.savefig(pdf, format="pdf")
    fig.savefig(png, format="png", dpi=150)
    plt.close(fig)
    print(f"\n→ wrote {pdf.relative_to(repo)}")
    print(f"→ wrote {png.relative_to(repo)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
