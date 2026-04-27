#!/usr/bin/env python3
"""Post-processor for E6 Byzantine-withstand (uniform adversary).

Reads the latest run-*-uniform-*.json under eval/results/E6/ (or an
explicit path), extracts the per-fraction retrieval-success rate, and
emits paper/figures/E6_byzantine_uniform.{pdf,png}.

The figure is the headline measurement for CLAUDE.md §8 row E6 / Tier 2
contribution #4 (tier-aware Byzantine withstand): success rate vs the
fraction of byzantine replicas in the global pool. RS(2,3) tolerates one
failed shard per bundle, so the curve should stay near 1.0 below f≈0.33,
drop sharply through f=0.5 (point at which the expected number of
failures per bundle crosses 1.0), and approach 0 by f=0.67.

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
    """Find the most recent run-*.json under out_dir whose filename
    contains the given mode token (we encode mode into the run-id so the
    same directory holds both uniform and tier-selective runs).
    """
    runs = sorted(p for p in out_dir.glob("run-*.json") if mode_token in p.name)
    if not runs:
        raise FileNotFoundError(
            f"no run-*-{mode_token}-*.json in {out_dir}; run "
            f"`go run ./cmd/bench/e6 -mode uniform` first"
        )
    return runs[-1]  # timestamped prefix sorts lexicographically


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

    run_path = args.run if args.run else find_latest_run(args.out_dir, "uniform")
    rec = json.loads(run_path.read_text())

    fractions_data = rec.get("fractions", [])
    if not fractions_data:
        print(f"! no fractions data in {run_path}", file=sys.stderr)
        return 1

    # Sort by fraction so the line plot is monotonic on x.
    fractions_data = sorted(fractions_data, key=lambda f: f["adversary_fraction"])
    xs = np.array([f["adversary_fraction"] for f in fractions_data], dtype=float)
    ys = np.array([f["retrieval_success_rate"] for f in fractions_data], dtype=float)

    cfg = rec.get("config", {})
    n_bundles = cfg.get("num_bundles", 0)
    reps = cfg.get("reps_per_fraction", 0)

    # Stdout summary — operator confirms shape before opening the figure.
    print(f"E6 uniform run: {rec.get('run_id', '?')}")
    print(f"  source: {run_path}")
    print(f"  config: {n_bundles} bundles × {reps} reps × {len(xs)} fractions")
    print()
    print(f"  {'fraction':>10}  {'retrieval_success':>18}  {'tier_byz (h/w/c)':>22}")
    for f in fractions_data:
        tb = f.get("tier_breakdown", {})
        tier_str = (
            f"{tb.get('hot', 0):.2f}/{tb.get('warm', 0):.2f}/{tb.get('cold', 0):.2f}"
        )
        print(
            f"  {f['adversary_fraction']:>10.2f}  "
            f"{f['retrieval_success_rate']:>18.3f}  "
            f"{tier_str:>22}"
        )

    # Locate the threshold where success drops below 50% — annotated on
    # the figure. Linear interpolation between adjacent fractions if the
    # crossing is not exactly on a measured x.
    crossing = None
    for i in range(1, len(xs)):
        if ys[i - 1] >= 0.5 > ys[i]:
            # Linear interp.
            x0, x1 = xs[i - 1], xs[i]
            y0, y1 = ys[i - 1], ys[i]
            if y0 != y1:
                crossing = x0 + (0.5 - y0) * (x1 - x0) / (y1 - y0)
            else:
                crossing = (x0 + x1) / 2
            break

    fig, ax = plt.subplots(figsize=(7.2, 4.6))
    ax.plot(
        xs,
        ys,
        marker="o",
        markersize=7,
        linewidth=1.8,
        color="#d7191c",
        label="Retrieval success (RS(2,3) decode + Merkle re-verify)",
    )

    # Theoretical reference: byzantine corruption is NOT the same as
    # availability failure. RS(2,3) tolerates one shard *erased* (the
    # erasure-decode case), but a byzantine replica returning length-
    # preserving corrupted bytes pollutes the fast path because gateway
    # parity-verifies only when all 3 shards are present. In practice
    # the fast path only fetches D0+D1, so retrieval succeeds iff both
    # data shards are honest: (1-f)^2 for uniform Bernoulli(f).
    #
    # This is a non-obvious distinction the §V narrative will call out
    # explicitly: erasure parity gives availability redundancy, not
    # integrity redundancy under byzantine adversaries — that's the
    # job of the Merkle re-verify in the post-Recover gate.
    f_dense = np.linspace(0, max(xs), 200)
    theory_fast = (1 - f_dense) ** 2
    ax.plot(
        f_dense,
        theory_fast,
        linestyle="--",
        linewidth=1.2,
        color="#2c7bb6",
        alpha=0.75,
        label="(1-f)² fast-path expectation (D0+D1 both honest)",
    )

    # 50% reference line + crossing annotation.
    ax.axhline(
        0.5,
        color="black",
        linestyle=":",
        linewidth=0.9,
        alpha=0.6,
    )
    if crossing is not None:
        ax.axvline(
            crossing,
            color="black",
            linestyle=":",
            linewidth=0.9,
            alpha=0.6,
        )
        ax.annotate(
            f"50% threshold ≈ f={crossing:.2f}",
            xy=(crossing, 0.5),
            xytext=(crossing + 0.02, 0.55),
            fontsize=9,
            arrowprops=dict(arrowstyle="->", color="black", lw=0.7),
        )

    ax.set_xlabel("Fraction of byzantine replicas (global pool, uniform across tiers)")
    ax.set_ylabel("Retrieval success rate")
    ax.set_title(
        f"Byzantine withstand — uniform adversary "
        f"({n_bundles} bundles × {reps} reps per fraction)"
    )
    ax.set_xlim(-0.02, max(xs) + 0.05)
    ax.set_ylim(-0.02, 1.05)
    ax.grid(True, alpha=0.25)
    ax.legend(loc="lower left", fontsize=9, framealpha=0.95)
    fig.tight_layout()

    args.fig_dir.mkdir(parents=True, exist_ok=True)
    pdf_path = args.fig_dir / "E6_byzantine_uniform.pdf"
    png_path = args.fig_dir / "E6_byzantine_uniform.png"
    fig.savefig(pdf_path, format="pdf")
    fig.savefig(png_path, format="png", dpi=150)
    plt.close(fig)
    print()
    print(f"→ wrote {pdf_path}")
    print(f"→ wrote {png_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
