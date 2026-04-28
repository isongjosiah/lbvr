#!/usr/bin/env python3
"""Post-processor for E1 ingest-throughput bars (Fig. 2).

Reads the latest run-*.json under eval/results/E1/ (or an explicit path)
and emits paper/figures/E1_ingest_throughput.{pdf,png}.

CLAUDE.md §8 row E1 + reproducibility contract: figure must be
reproducible from the raw JSON. Re-running on the same input must
produce byte-identical output.

Layout:
  - Left subplot: throughput in MiB/sec, one bar per corpus size.
    Each bar is stacked-fraction across the five timed stages
    (merkle / seal / encode / upload / register) so the eye reads the
    dominant stage at a glance. Stack heights sum to 1.0; stage shares
    are computed from the per-stage P50 contributions.
  - Right subplot: bundles per minute, one bar per corpus size, plain
    bars (no stack — the breakdown lives in the left plot).
  - Each bar is annotated with the wall-clock duration. Projected scales
    (100K) are tagged "(projected)".
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

# Stage stacking palette. Order matches the pipeline order so the visual
# left-to-right matches the §IV narrative (read → merkle → seal → encode
# → upload → register).
STAGE_ORDER = ["merkle", "seal", "encode", "upload", "register"]
STAGE_COLOURS = {
    "merkle":   "#2c7bb6",  # blue — SHA-256 over chunks
    "seal":     "#abd9e9",  # light blue — AES-GCM
    "encode":   "#fdae61",  # orange — RS(2,3)
    "upload":   "#d7191c",  # red — parallel cross-tier Put (dominant)
    "register": "#7b3294",  # purple — Mock RegisterBundle
}

# Long names for the legend.
STAGE_LABELS = {
    "merkle":   "Merkle build",
    "seal":     "AES-GCM seal",
    "encode":   "RS(2,3) encode",
    "upload":   "Cross-tier upload",
    "register": "Registry write",
}


def find_latest_run(out_dir: Path) -> Path:
    runs = sorted(out_dir.glob("run-*.json"))
    if not runs:
        raise FileNotFoundError(f"no run-*.json in {out_dir}")
    return runs[-1]  # timestamped prefix sorts lexicographically


def fmt_corpus_label(n: int) -> str:
    """Format 1000 → "1K", 10000 → "10K", 100000 → "100K"."""
    if n >= 1000 and n % 1000 == 0:
        return f"{n // 1000}K"
    return str(n)


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    p.add_argument("--source", "--run", dest="run", type=Path,
                   help="explicit run-*.json path")
    p.add_argument("--out-dir", type=Path, default=Path("eval/results/E1"),
                   help="search dir when --source is omitted")
    p.add_argument("--fig-dir", type=Path, default=Path("paper/figures"),
                   help="output directory for PDF + PNG")
    args = p.parse_args()

    run_path = args.run if args.run else find_latest_run(args.out_dir)
    rec = json.loads(run_path.read_text())

    scales = rec.get("scales", [])
    if not scales:
        print(f"! no scales in {run_path}", file=sys.stderr)
        return 1

    # Sort by corpus size so the bars read 1K → 10K → 100K.
    scales = sorted(scales, key=lambda s: s["corpus_size"])
    cfg = rec.get("config", {})

    # Print P50/P99 stage table for the operator.
    print(f"E1 run: {rec.get('run_id', '?')}")
    print(f"  source: {run_path}")
    print(f"  cold-tier mechanism: {cfg.get('cold_tier_mechanism', '?')}")
    print(f"  concurrency: {cfg.get('concurrency', '?')}")
    print()
    hdr = f"  {'scale':<10}  {'n':>6}  {'GiB':>6}  {'wall_s':>8}  {'MiB/s':>8}  {'b/min':>8}  proj"
    print(hdr)
    for s in scales:
        proj = "yes" if s.get("projected") else "no"
        print(f"  {fmt_corpus_label(s['corpus_size']):<10}  "
              f"{s['n_bundles']:>6}  "
              f"{s['total_bytes']/(1<<30):>6.2f}  "
              f"{s['wall_seconds']:>8.1f}  "
              f"{s['throughput_mbps']:>8.2f}  "
              f"{s['throughput_bundles_per_min']:>8.1f}  "
              f"{proj}")
    print()
    print("  Stage P50 (ms) per scale:")
    sub_hdr = f"  {'scale':<10}  " + "  ".join(f"{st:>9}" for st in STAGE_ORDER) + f"  {'total':>9}"
    print(sub_hdr)
    for s in scales:
        if s.get("projected"):
            continue
        st = s["stage_p50_ms"]
        print(f"  {fmt_corpus_label(s['corpus_size']):<10}  " +
              "  ".join(f"{st[k]:>9.2f}" for k in STAGE_ORDER) +
              f"  {st['total']:>9.2f}")

    # Build the figure.
    fig, (ax_l, ax_r) = plt.subplots(1, 2, figsize=(11.0, 4.6))

    labels = [fmt_corpus_label(s["corpus_size"]) for s in scales]
    x = np.arange(len(scales))
    width = 0.62

    # Left: stacked-fraction MiB/s bars. Each bar's full height is the
    # measured throughput; the stage segments are scaled so each
    # segment's height corresponds to its share of the total stage
    # budget (P50). Sum of stage P50s ≈ total P50 — see the test
    # harness's TestStageTimings_Sum sanity check.
    for i, s in enumerate(scales):
        st = s["stage_p50_ms"]
        share_total = sum(st[k] for k in STAGE_ORDER)
        if share_total <= 0:
            share_total = 1.0  # avoid divide-by-zero on a degenerate run
        bottom = 0.0
        for stage in STAGE_ORDER:
            frac = st[stage] / share_total
            seg_height = frac * s["throughput_mbps"]
            ax_l.bar(
                x[i], seg_height, width,
                bottom=bottom,
                color=STAGE_COLOURS[stage],
                edgecolor="black", linewidth=0.5,
                hatch=("////" if s.get("projected") else None),
                label=STAGE_LABELS[stage] if i == 0 else None,
            )
            bottom += seg_height
        # Annotate the bar top with wall-clock and projected tag.
        annotation = f"{s['wall_seconds']:.0f}s"
        if s.get("projected"):
            annotation += f"\n(proj. from {fmt_corpus_label(s.get('projected_from_scale', 0))})"
        ax_l.text(
            x[i], s["throughput_mbps"] * 1.02,
            annotation,
            ha="center", va="bottom", fontsize=8.5,
        )

    ax_l.set_xticks(x)
    ax_l.set_xticklabels(labels)
    ax_l.set_xlabel("Corpus size (patients)")
    ax_l.set_ylabel("Ingest throughput (MiB/s)")
    ax_l.set_title("Throughput with stage-share stacking (P50)")
    ax_l.grid(True, axis="y", alpha=0.25)
    # Headroom for the wall-clock annotations.
    ymax_l = max((s["throughput_mbps"] for s in scales), default=1) * 1.20
    ax_l.set_ylim(0, ymax_l)
    ax_l.legend(loc="upper right", fontsize=8.5, framealpha=0.95)

    # Right: bundles per minute. Plain bars; the wall-clock annotation
    # repeats so each subplot is independently readable.
    bpm = [s["throughput_bundles_per_min"] for s in scales]
    bar_colours = [
        "#888888" if s.get("projected") else "#1a1a1a" for s in scales
    ]
    bar_hatches = [
        "////" if s.get("projected") else "" for s in scales
    ]
    for i, (s, col, hatch) in enumerate(zip(scales, bar_colours, bar_hatches)):
        ax_r.bar(
            x[i], s["throughput_bundles_per_min"], width,
            color=col, edgecolor="black", linewidth=0.5, hatch=hatch,
        )
        annotation = f"{s['throughput_bundles_per_min']:.0f} b/min"
        if s.get("projected"):
            annotation += f"\n(proj. from {fmt_corpus_label(s.get('projected_from_scale', 0))})"
        else:
            annotation += f"\n{s['wall_seconds']:.0f}s wall"
        ax_r.text(
            x[i], s["throughput_bundles_per_min"] * 1.02,
            annotation,
            ha="center", va="bottom", fontsize=8.5,
        )

    ax_r.set_xticks(x)
    ax_r.set_xticklabels(labels)
    ax_r.set_xlabel("Corpus size (patients)")
    ax_r.set_ylabel("Bundles per minute")
    ax_r.set_title("Throughput in bundles/min")
    ax_r.grid(True, axis="y", alpha=0.25)
    ymax_r = max(bpm) * 1.25 if bpm else 1
    ax_r.set_ylim(0, ymax_r)

    fig.suptitle(
        "Ingest throughput across corpus scales — RS(2,3) cross-tier upload",
        fontsize=12.5,
    )
    fig.tight_layout()

    args.fig_dir.mkdir(parents=True, exist_ok=True)
    pdf_path = args.fig_dir / "E1_ingest_throughput.pdf"
    png_path = args.fig_dir / "E1_ingest_throughput.png"
    fig.savefig(pdf_path, format="pdf")
    fig.savefig(png_path, format="png", dpi=150)
    plt.close(fig)
    print()
    print(f"→ wrote {pdf_path}")
    print(f"→ wrote {png_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
