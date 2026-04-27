#!/usr/bin/env python3
"""Post-processor for E2 retrieval-latency CDF.

Reads the latest run-*.json under eval/results/E2/ (or an explicit path),
computes the empirical CDF per mode (flattened across bundles), and emits
paper/figures/E2_retrieval_latency_cdf.{pdf,png}.

CLAUDE.md §8 row E2 ("Retrieval latency CDF, 3 tiers × baseline
{Pinata-only, S3, Storj, ipfs.io} — fast path") + reproducibility
contract: figure must be reproducible from the raw JSON. Re-running on
the same input must produce byte-identical output.
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


# Per-mode visual style. LBVR fast path gets a heavier line + saturated
# colour so the eye anchors on it; baselines are lighter.
MODE_STYLE = {
    "lbvr":        {"colour": "#1a1a1a", "linewidth": 2.4, "linestyle": "-",  "label": "LBVR-Med (verifiable fast path)"},
    "s3":          {"colour": "#2c7bb6", "linewidth": 1.3, "linestyle": "-",  "label": "AWS S3 (single-tier)"},
    "pinata_only": {"colour": "#fdae61", "linewidth": 1.3, "linestyle": "-",  "label": "Pinata only (single-tier)"},
    "storj":       {"colour": "#7b3294", "linewidth": 1.3, "linestyle": "--", "label": "Storj (single-tier)"},
    "ipfsio":      {"colour": "#d7191c", "linewidth": 1.3, "linestyle": ":",  "label": "Public ipfs.io gateway"},
}

# Order on the legend matches the §V narrative: lbvr first, then
# centralised baselines, then decentralised baselines.
MODE_ORDER = ["lbvr", "s3", "pinata_only", "storj", "ipfsio"]

# Clinical SLO thresholds — annotated as vertical dashed lines so
# reviewers can read attainment at a glance. Numbers match CLAUDE.md
# §12 ("anchor every claim ... to a specific clinical SLO").
SLOS = [
    (500,  "Chart pull (500 ms)"),
    (2000, "Radiology open (2 s)"),
]


def find_latest_run(out_dir: Path) -> Path:
    runs = sorted(out_dir.glob("run-*.json"))
    if not runs:
        raise FileNotFoundError(f"no run-*.json in {out_dir}")
    return runs[-1]  # timestamped prefix sorts lexicographically


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    p.add_argument("--source", "--run", dest="run", type=Path,
                   help="explicit run-*.json path")
    p.add_argument("--out-dir", type=Path, default=Path("eval/results/E2"),
                   help="search dir when --source is omitted")
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

    # Print P50/P95/P99 per mode.
    print(f"E2 run: {rec.get('run_id', '?')}  ({n} bundles × {reps} reps)")
    print(f"  source: {run_path}")
    print(f"  cold-tier mechanism: {cfg.get('cold_tier_mechanism', '?')}")
    print()
    print(f"  {'mode':<14}  {'P50_ms':>10}  {'P95_ms':>10}  {'P99_ms':>10}  {'N':>6}")
    for m in MODE_ORDER:
        if m not in by_mode:
            continue
        xs = by_mode[m]
        p50 = float(np.percentile(xs, 50))
        p95 = float(np.percentile(xs, 95))
        p99 = float(np.percentile(xs, 99))
        print(f"  {m:<14}  {p50:>10.2f}  {p95:>10.2f}  {p99:>10.2f}  {len(xs):>6d}")

    # Plot.
    fig, ax = plt.subplots(figsize=(7.6, 4.8))
    for m in MODE_ORDER:
        if m not in by_mode:
            continue
        st = MODE_STYLE[m]
        xs = np.sort(by_mode[m])
        ys = np.arange(1, len(xs) + 1) / len(xs)
        ax.plot(
            xs, ys,
            label=st["label"],
            color=st["colour"],
            linewidth=st["linewidth"],
            linestyle=st["linestyle"],
        )

    # SLO annotations — drawn UNDER the curves with low alpha so they
    # don't dominate the plot but remain readable.
    for ms, label in SLOS:
        ax.axvline(ms, color="#555555", linestyle="--", linewidth=0.9, alpha=0.7)
        ax.text(
            ms, 0.04, " " + label,
            rotation=90, va="bottom", ha="left",
            fontsize=8, color="#555555",
        )

    ax.set_xscale("log")
    ax.set_xlabel("Retrieval latency (ms, log scale)")
    ax.set_ylabel("CDF")
    title_n = n if n else "?"
    title_reps = reps if reps else "?"
    ax.set_title(
        f"Retrieval latency CDF — LBVR-Med vs single-tier baselines "
        f"({title_n} bundles × {title_reps} reps)"
    )
    ax.grid(True, alpha=0.25, which="both")
    ax.set_ylim(0, 1.02)
    ax.legend(loc="lower right", fontsize=8.5, framealpha=0.95)
    fig.tight_layout()

    args.fig_dir.mkdir(parents=True, exist_ok=True)
    pdf_path = args.fig_dir / "E2_retrieval_latency_cdf.pdf"
    png_path = args.fig_dir / "E2_retrieval_latency_cdf.png"
    fig.savefig(pdf_path, format="pdf")
    fig.savefig(png_path, format="png", dpi=150)
    plt.close(fig)
    print()
    print(f"→ wrote {pdf_path}")
    print(f"→ wrote {png_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
