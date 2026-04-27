#!/usr/bin/env python3
"""Post-processor for E6b — tier-selective adversary detection gap.

Reads the latest run-*-tier-selective-*.json under eval/results/E6/ (or
an explicit path) and emits paper/figures/E6b_detection_gap.{pdf,png}.

The figure is the headline measurement for CLAUDE.md §5 / §8 row E6b /
Tier 2 contribution #4 (tier-aware Byzantine withstand). Two curves:

  - PoR success rate vs adversary fraction.
    Should hover near 1.0 for every f because the byzantine replicas
    pass PoR challenges by design (the §5 metadata-correlated threat).
  - Retrieval success rate vs adversary fraction.
    Degrades with f because the same replicas serve corrupt bytes during
    real reads.

The shaded area between them IS the detection gap: every percent of gap
is a percent of retrievals an auditor relying solely on PoR would miss.
The figure annotates this directly so the §V narrative needs only one
sentence.

CLAUDE.md §8 contract: figure must be reproducible from the raw JSON.
Re-running on the same input produces byte-identical output.
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


def find_latest_run(out_dir: Path, mode_token: str) -> Path:
    runs = sorted(p for p in out_dir.glob("run-*.json") if mode_token in p.name)
    if not runs:
        raise FileNotFoundError(
            f"no run-*-{mode_token}-*.json in {out_dir}; run "
            f"`go run ./cmd/bench/e6 -mode tier-selective` first"
        )
    return runs[-1]


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    p.add_argument("--run", type=Path, help="explicit run-*.json path")
    p.add_argument(
        "--out-dir",
        type=Path,
        default=Path("eval/results/E6"),
        help="search dir when --run is omitted",
    )
    p.add_argument(
        "--fig-dir",
        type=Path,
        default=Path("paper/figures"),
        help="output directory for PDF + PNG",
    )
    args = p.parse_args()

    run_path = args.run if args.run else find_latest_run(args.out_dir, "tier-selective")
    rec = json.loads(run_path.read_text())

    fractions_data = rec.get("fractions", [])
    if not fractions_data:
        print(f"! no fractions data in {run_path}", file=sys.stderr)
        return 1

    fractions_data = sorted(fractions_data, key=lambda f: f["adversary_fraction"])
    xs = np.array([f["adversary_fraction"] for f in fractions_data], dtype=float)
    ys_por = np.array([f.get("por_success_rate", 0.0) for f in fractions_data], dtype=float)
    ys_ret = np.array([f["retrieval_success_rate"] for f in fractions_data], dtype=float)
    gaps = ys_por - ys_ret

    cfg = rec.get("config", {})
    n_bundles = cfg.get("num_bundles", 0)
    reps = cfg.get("reps_per_fraction", 0)

    # Stdout summary.
    print(f"E6b tier-selective run: {rec.get('run_id', '?')}")
    print(f"  source: {run_path}")
    print(f"  config: {n_bundles} bundles × {reps} reps × {len(xs)} fractions")
    print()
    print(
        f"  {'fraction':>10}  {'por':>8}  {'retrieval':>10}  "
        f"{'gap':>8}  {'tier_byz (h/w/c)':>22}"
    )
    for f in fractions_data:
        tb = f.get("tier_breakdown", {})
        tier_str = (
            f"{tb.get('hot', 0):.2f}/{tb.get('warm', 0):.2f}/{tb.get('cold', 0):.2f}"
        )
        print(
            f"  {f['adversary_fraction']:>10.2f}  "
            f"{f.get('por_success_rate', 0):>8.3f}  "
            f"{f['retrieval_success_rate']:>10.3f}  "
            f"{f.get('detection_gap', 0):>8.3f}  "
            f"{tier_str:>22}"
        )

    # Pick a "representative" fraction to annotate — use the one with
    # the largest gap (visually informative without being the f=0 zero
    # case or the f=0.67 collapse case).
    if len(gaps) > 0:
        ann_idx = int(np.argmax(gaps))
    else:
        ann_idx = 0

    fig, ax = plt.subplots(figsize=(7.6, 4.8))

    # PoR curve — flat near 1.0 by design.
    ax.plot(
        xs,
        ys_por,
        marker="s",
        markersize=7,
        linewidth=1.8,
        color="#2c7bb6",
        label="PoR challenge success (auditor-observed health)",
    )
    # Retrieval curve — drops with f.
    ax.plot(
        xs,
        ys_ret,
        marker="o",
        markersize=7,
        linewidth=1.8,
        color="#d7191c",
        label="Retrieval success (clinician-observed health)",
    )

    # Detection-gap shading. fill_between only matters where PoR > retrieval;
    # at f=0 they should coincide.
    ax.fill_between(
        xs,
        ys_ret,
        ys_por,
        where=ys_por >= ys_ret,
        color="#fdae61",
        alpha=0.35,
        label="Detection gap (§5 metadata-correlated adversary)",
    )

    # Annotate the largest-gap point.
    if 0 <= ann_idx < len(xs):
        xa = xs[ann_idx]
        gap = gaps[ann_idx]
        ya = (ys_por[ann_idx] + ys_ret[ann_idx]) / 2
        ax.annotate(
            f"max gap = {gap*100:.1f}%\n"
            f"at f = {xa:.2f}\n"
            f"(invisible to PoR; corrupts {gap*100:.0f}% of retrievals)",
            xy=(xa, ya),
            xytext=(xa + 0.08, ya - 0.05),
            fontsize=8.5,
            ha="left",
            arrowprops=dict(arrowstyle="->", color="black", lw=0.7),
            bbox=dict(boxstyle="round,pad=0.4", facecolor="white", alpha=0.9, edgecolor="#666"),
        )

    ax.set_xlabel("Fraction of byzantine replicas (global pool, biased to cold tier)")
    ax.set_ylabel("Success rate")
    ax.set_title(
        f"Tier-selective Byzantine withstand — PoR vs retrieval gap "
        f"({n_bundles} bundles × {reps} reps per fraction)"
    )
    ax.set_xlim(-0.02, max(xs) + 0.05)
    ax.set_ylim(-0.02, 1.08)
    ax.grid(True, alpha=0.25)
    ax.legend(loc="lower left", fontsize=8.5, framealpha=0.95)

    # Footer: motivates the tier-aware audit cadence in §V discussion.
    fig.text(
        0.5, 0.005,
        "Motivates tier-aware PoR cadence: cold-tier audits must be at retrieval call rate, not the 30-day default.",
        ha="center", fontsize=7.5, style="italic", color="#444",
    )

    fig.tight_layout(rect=[0, 0.02, 1, 1])

    args.fig_dir.mkdir(parents=True, exist_ok=True)
    pdf_path = args.fig_dir / "E6b_detection_gap.pdf"
    png_path = args.fig_dir / "E6b_detection_gap.png"
    fig.savefig(pdf_path, format="pdf")
    fig.savefig(png_path, format="png", dpi=150)
    plt.close(fig)
    print()
    print(f"→ wrote {pdf_path}")
    print(f"→ wrote {png_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
