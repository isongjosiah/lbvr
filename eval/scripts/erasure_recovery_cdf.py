#!/usr/bin/env python3
"""Post-processor for E9 erasure-recovery latency.

Reads the latest run-*.json under eval/results/E9/ (or an explicit path),
computes the empirical CDF per mode (flattened across bundles), and emits
paper/figures/E9_erasure_recovery_cdf.{pdf,png}.

CLAUDE.md §4.5 / §8 contract: figure must be reproducible from the raw
JSON. Re-running on the same input must produce byte-identical output.
"""
from __future__ import annotations
import argparse
import json
import sys
from pathlib import Path

import matplotlib
matplotlib.use("Agg")  # headless
import matplotlib.pyplot as plt
import numpy as np


# Stable per-mode colours; chosen for distinct hue + matplotlib defaults.
MODE_COLOURS = {
    "baseline": "#2c7bb6",  # blue — fast path, no failure
    "P0_lost":  "#abd9e9",  # light blue — fast path, parity drop (looks like baseline)
    "D0_lost":  "#fdae61",  # orange — slow path, hot tier dropped
    "D1_lost":  "#d7191c",  # red — slow path, warm tier dropped
}

# Order on the legend matches the §V narrative: baseline first, parity
# (also fast-path) next, then the two slow-path modes.
MODE_ORDER = ["baseline", "P0_lost", "D0_lost", "D1_lost"]


def find_latest_run(out_dir: Path) -> Path:
    runs = sorted(out_dir.glob("run-*.json"))
    if not runs:
        raise FileNotFoundError(f"no run-*.json in {out_dir}")
    return runs[-1]  # timestamped prefix sorts lexicographically


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    p.add_argument("--run", type=Path, help="explicit run-*.json path")
    p.add_argument("--out-dir", type=Path, default=Path("eval/results/E9"),
                   help="search dir when --run is omitted")
    p.add_argument("--fig-dir", type=Path, default=Path("paper/figures"),
                   help="output directory for PDF + PNG")
    args = p.parse_args()

    run_path = args.run if args.run else find_latest_run(args.out_dir)
    rec = json.loads(run_path.read_text())

    # Extract per-mode latency arrays (ms).
    by_mode: dict[str, np.ndarray] = {}
    for m in MODE_ORDER:
        xs: list[int] = []
        for sample in rec.get("samples", []):
            runs = sample.get("mode_runs", {}).get(m, [])
            xs.extend(r["latency_ns"] for r in runs)
        if not xs:
            print(f"! mode {m!r} has no runs in {run_path}", file=sys.stderr)
            continue
        by_mode[m] = np.asarray(xs, dtype=np.int64) / 1e6  # ns → ms

    if not by_mode:
        print(f"! no recoverable data in {run_path}", file=sys.stderr)
        return 1

    cfg = rec.get("config", {})
    n = cfg.get("num_bundles", 0)
    reps = cfg.get("reps_per_mode", 0)
    slo_ms = cfg.get("slo_budget_ms", 2000)

    # Print P50/P95/P99 per mode (stdout summary; matches CLAUDE.md §8
    # "scripts must be idempotent" by writing the same numbers each run).
    print(f"E9 run: {rec.get('run_id', '?')}  ({n} bundles × {reps} reps)")
    print(f"  source: {run_path}")
    print(f"  cold-tier mechanism: {cfg.get('cold_tier_mechanism', '?')}")
    print()
    print(f"  {'mode':<10}  {'P50_ms':>10}  {'P95_ms':>10}  {'P99_ms':>10}  {'N':>6}")
    for m in MODE_ORDER:
        if m not in by_mode:
            continue
        xs = by_mode[m]
        p50 = float(np.percentile(xs, 50))
        p95 = float(np.percentile(xs, 95))
        p99 = float(np.percentile(xs, 99))
        print(f"  {m:<10}  {p50:>10.2f}  {p95:>10.2f}  {p99:>10.2f}  {len(xs):>6d}")

    # Plot.
    fig, ax = plt.subplots(figsize=(7.2, 4.6))
    for m in MODE_ORDER:
        if m not in by_mode:
            continue
        xs = np.sort(by_mode[m])
        ys = np.arange(1, len(xs) + 1) / len(xs)
        ax.plot(xs, ys, label=m, color=MODE_COLOURS[m], linewidth=1.6)
    ax.set_xscale("log")
    ax.set_xlabel("Recovery latency (ms, log scale)")
    ax.set_ylabel("CDF")
    title_n = n if n else "?"
    title_reps = reps if reps else "?"
    ax.set_title(f"Erasure recovery latency — {title_n} bundles × {title_reps} reps")
    ax.grid(True, alpha=0.25, which="both")

    # SLO line.
    ax.axvline(slo_ms, color="black", linestyle="--", linewidth=1.0,
               label=f"SLO budget ({slo_ms} ms)")

    ax.legend(loc="lower right", fontsize=9, framealpha=0.95)
    ax.set_ylim(0, 1.02)
    fig.tight_layout()

    args.fig_dir.mkdir(parents=True, exist_ok=True)
    pdf_path = args.fig_dir / "E9_erasure_recovery_cdf.pdf"
    png_path = args.fig_dir / "E9_erasure_recovery_cdf.png"
    fig.savefig(pdf_path, format="pdf")
    fig.savefig(png_path, format="png", dpi=150)
    plt.close(fig)
    print()
    print(f"→ wrote {pdf_path}")
    print(f"→ wrote {png_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
