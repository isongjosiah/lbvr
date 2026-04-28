#!/usr/bin/env python3
"""Post-processor for E5 PoR proof cost (Fig 6).

Reads the latest run-*.json under eval/results/E5/ (or an explicit path)
and emits a two-panel figure:

  Top panel: prove-time bars per Merkle proof depth (μs, P50 ± P95).
             Constant-time across depths — visualises that responder
             cost is dominated by BLS sign, not proof construction.
  Bottom panel: verify-side gas per depth, three stacked bars
             (post / respond / verdict). Only `respond` grows with
             depth; the other two are flat.

The figure pair lands in paper/figures/E5_por_cost_bars.{pdf,png} for
inclusion in §V of the paper.

CLAUDE.md §4.4 / §8 contract: figure must be reproducible from the raw
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


# Color scheme aligned with eprov_overhead.py — reviewer continuity.
COLOR_PROVE = "#fdae61"   # orange — BLS-dominated
COLOR_POST = "#9ecae1"    # pale blue — cheapest
COLOR_RESPOND = "#fd8d3c" # darker orange — Merkle proof verification
COLOR_VERDICT = "#74c476" # green — bookkeeping


def find_latest_run(out_dir: Path) -> Path:
    runs = sorted(out_dir.glob("run-*.json"))
    if not runs:
        raise FileNotFoundError(f"no run-*.json in {out_dir}")
    return runs[-1]


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    p.add_argument("--source", "--run", dest="source", type=Path,
                   help="explicit run-*.json path")
    p.add_argument("--out-dir", type=Path, default=Path("eval/results/E5"),
                   help="search dir when --source is omitted")
    p.add_argument("--fig-dir", type=Path, default=Path("paper/figures"),
                   help="output directory for PDF + PNG")
    args = p.parse_args()

    run_path = args.source if args.source else find_latest_run(args.out_dir)
    rec = json.loads(run_path.read_text())
    rows = rec.get("rows", [])
    if not rows:
        print(f"E5: no rows in {run_path}", file=sys.stderr)
        return 1

    rows.sort(key=lambda r: r["depth"])
    depths = [r["depth"] for r in rows]
    chunks = [r["num_chunks"] for r in rows]
    prove_p50_us = [r["prove_time_ns"]["p50"] / 1e3 for r in rows]
    prove_p95_us = [r["prove_time_ns"]["p95"] / 1e3 for r in rows]
    gas_post = [r["gas_post_challenge"] for r in rows]
    gas_respond = [r["gas_respond_to_challenge"] for r in rows]
    gas_verdict = [r["gas_record_verdict"] for r in rows]

    # --- figure layout: two stacked panels, shared x-axis ---
    fig, (ax_prove, ax_gas) = plt.subplots(
        2, 1, figsize=(7.0, 5.4), sharex=True,
        gridspec_kw={"height_ratios": [1.0, 1.4]},
    )

    x = np.arange(len(depths))

    # Top: prove time
    err_low = [p50 - p50 for p50 in prove_p50_us]  # P50 anchor
    err_high = [p95 - p50 for p50, p95 in zip(prove_p50_us, prove_p95_us)]
    ax_prove.bar(
        x, prove_p50_us, color=COLOR_PROVE, edgecolor="#7f7f7f", linewidth=0.5,
        yerr=[err_low, err_high], capsize=3, label="prove (P50, error → P95)",
    )
    ax_prove.set_ylabel("prove time (μs)")
    ax_prove.set_title("E5 — PoR responder cost vs Merkle proof depth")
    ax_prove.grid(axis="y", linestyle="--", linewidth=0.3, alpha=0.6)
    ax_prove.legend(loc="upper right", fontsize=8, framealpha=0.9)

    # Annotate: prove time is roughly constant (BLS-dominated).
    median_prove = np.median(prove_p50_us)
    ax_prove.axhline(median_prove, color="#7f7f7f", linewidth=0.6,
                     linestyle=":", alpha=0.7)
    ax_prove.text(
        len(depths) - 1, median_prove,
        f"  median P50 = {median_prove:.0f} μs (BLS sign-dominated)",
        ha="right", va="bottom", fontsize=7.5, color="#404040",
    )

    # Bottom: gas — three grouped bars per depth
    width = 0.27
    ax_gas.bar(x - width, [g / 1000 for g in gas_post], width,
               color=COLOR_POST, edgecolor="#7f7f7f", linewidth=0.5,
               label="postChallenge")
    ax_gas.bar(x, [g / 1000 for g in gas_respond], width,
               color=COLOR_RESPOND, edgecolor="#7f7f7f", linewidth=0.5,
               label="respondToChallenge")
    ax_gas.bar(x + width, [g / 1000 for g in gas_verdict], width,
               color=COLOR_VERDICT, edgecolor="#7f7f7f", linewidth=0.5,
               label="recordVerdict")
    ax_gas.set_ylabel("gas (×1000)")
    ax_gas.set_xlabel("Merkle proof depth (= ⌈log₂(numChunks)⌉)")
    ax_gas.set_xticks(x)
    ax_gas.set_xticklabels([f"{d}\n({c}c)" for d, c in zip(depths, chunks)])
    ax_gas.grid(axis="y", linestyle="--", linewidth=0.3, alpha=0.6)
    ax_gas.legend(loc="upper left", fontsize=8, framealpha=0.9, ncol=3)

    # Annotate respond's slope (the only depth-dependent function).
    if len(rows) >= 2:
        slope = (gas_respond[-1] - gas_respond[0]) / max(1, depths[-1] - depths[0])
        ax_gas.text(
            0.02, 0.96,
            f"respond slope ≈ {slope/1000:+.1f}k gas / depth level",
            transform=ax_gas.transAxes, fontsize=7.5, color="#404040",
            verticalalignment="top",
        )

    fig.tight_layout()

    args.fig_dir.mkdir(parents=True, exist_ok=True)
    pdf_path = args.fig_dir / "E5_por_cost_bars.pdf"
    png_path = args.fig_dir / "E5_por_cost_bars.png"
    fig.savefig(pdf_path, bbox_inches="tight", metadata={"CreationDate": None})
    fig.savefig(png_path, bbox_inches="tight", dpi=180)
    plt.close(fig)

    print(f"E5: wrote {pdf_path}")
    print(f"E5: wrote {png_path}")

    # Echo the table for the §V draft.
    print()
    print(f"  {'depth':>5} {'chunks':>7} {'prove_p50_us':>13} {'prove_p99_us':>13} "
          f"{'gas_post':>10} {'gas_respond':>11} {'gas_verdict':>11}")
    for r in rows:
        print(f"  {r['depth']:>5} {r['num_chunks']:>7} "
              f"{r['prove_time_ns']['p50']/1e3:>13.1f} "
              f"{r['prove_time_ns']['p99']/1e3:>13.1f} "
              f"{r['gas_post_challenge']:>10} "
              f"{r['gas_respond_to_challenge']:>11} "
              f"{r['gas_record_verdict']:>11}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
