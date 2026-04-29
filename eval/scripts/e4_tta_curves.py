#!/usr/bin/env python3
"""E4 — Time-to-availability post-PUT (Fig 5).

Reads the latest run-*.json under eval/results/E4/ and renders a CDF
per tier of `first_ok_at_ms` (= ms from PUT-return until the first
non-source GET succeeds). Three curves on a shared log-scale x-axis,
plus reference lines at the canonical clinical SLOs (500 ms chart-pull,
2 s radiology open, 10 s alarm).

The plot is the headline §V finding for §VI's "Trautwein 2024 IPFS
PUTs taking 'dozens of seconds to minutes'" framing — LBVR-Med's
hot+warm path stays well under 1 s for the 100-bundle population while
the cold tier extends into 30s+ as expected for chain settlement.

Sim/live provenance is read from the per-tier `live_mode` flag in the
JSON; a 'sim' badge appears in each curve label so reviewers can see
exactly which tiers are calibrated stand-ins vs measured.
"""
from __future__ import annotations
import argparse
import json
import sys
from pathlib import Path

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np


TIER_ORDER = ["hot", "warm", "cold"]
TIER_COLOURS = {
    "hot":  "#1a9850",
    "warm": "#fdae61",
    "cold": "#c0392b",
}
TIER_LABELS = {
    "hot":  "Hot — Pinata",
    "warm": "Warm — Filebase",
    "cold": "Cold — Irys / Arweave",
}

# Reference SLOs from CLAUDE.md §12.
SLO_LINES = [
    (500,   "Chart-pull (500 ms)",      "#2c3e50"),
    (2000,  "Radiology open (2 s)",     "#2c3e50"),
    (10000, "Alarm IEC 60601-1-8 (10 s)", "#2c3e50"),
]


def find_latest_run(out_dir: Path) -> Path:
    runs = sorted(out_dir.glob("run-*.json"))
    if not runs:
        raise FileNotFoundError(f"no run-*.json in {out_dir}")
    return runs[-1]


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    p.add_argument("--source", "--run", dest="source", type=Path,
                   help="explicit run-*.json path")
    p.add_argument("--out-dir", type=Path, default=Path("eval/results/E4"),
                   help="search dir when --source is omitted")
    p.add_argument("--fig-dir", type=Path, default=Path("paper/figures"),
                   help="output directory for PDF + PNG")
    args = p.parse_args()

    run_path = args.source if args.source else find_latest_run(args.out_dir)
    rec = json.loads(run_path.read_text())
    samples = rec.get("samples", [])
    if not samples:
        print(f"E4: no samples in {run_path}", file=sys.stderr)
        return 1

    # Collect first_ok_at_ms per tier + record sim/live for the legend badge.
    by_tier: dict[str, list[int]] = {t: [] for t in TIER_ORDER}
    live_mode: dict[str, bool] = {t: False for t in TIER_ORDER}
    for s in samples:
        for tier in TIER_ORDER:
            row = s.get("tiers", {}).get(tier)
            if not row:
                continue
            by_tier[tier].append(row["first_ok_at_ms"])
            if row.get("live_mode"):
                live_mode[tier] = True

    fig, ax = plt.subplots(figsize=(7.0, 4.4))

    for tier in TIER_ORDER:
        xs = sorted(by_tier[tier])
        if not xs:
            continue
        # Empirical CDF: y = (1..N)/N at each x point.
        ys = np.arange(1, len(xs) + 1) / len(xs)
        badge = "live" if live_mode[tier] else "sim"
        label = f"{TIER_LABELS[tier]} (n={len(xs)}, {badge})"
        ax.plot(xs, ys, color=TIER_COLOURS[tier], linewidth=2.0,
                marker="o", markersize=3, label=label)

    # SLO reference lines.
    for (x_ms, lbl, c) in SLO_LINES:
        ax.axvline(x_ms, color=c, linestyle=":", linewidth=0.8, alpha=0.4)
        ax.text(x_ms, 0.03, " " + lbl, ha="left", va="bottom",
                fontsize=7.5, color="#404040", rotation=90)

    ax.set_xscale("log")
    ax.set_xlabel("Time since PUT-return (ms, log scale)")
    ax.set_ylabel("Fraction of bundles reachable")
    ax.set_title("E4 — Time-to-Availability post-PUT, by tier")
    ax.set_ylim(0, 1.02)
    ax.set_xlim(50, 600_000)
    ax.grid(linestyle="--", linewidth=0.3, alpha=0.6)
    ax.legend(loc="lower right", fontsize=8, framealpha=0.95)

    # Sim/live provenance footer.
    sim_tiers = [t for t in TIER_ORDER if not live_mode[t]]
    live_tiers = [t for t in TIER_ORDER if live_mode[t]]
    if sim_tiers:
        cfg = rec.get("config", {})
        cal_summary = (
            f"Sim-mode tiers: {','.join(sim_tiers)}. "
            f"Calibration: hot prop {cfg.get('hot_prop_p50_ms','?')}/{cfg.get('hot_prop_p99_ms','?')}ms, "
            f"warm {cfg.get('warm_prop_p50_ms','?')}/{cfg.get('warm_prop_p99_ms','?')}ms, "
            f"cold {cfg.get('cold_prop_p50_ms','?')}/{cfg.get('cold_prop_p99_ms','?')}ms (P50/P99 propagation)."
        )
        fig.text(0.5, -0.06, cal_summary, ha="center", fontsize=7.5,
                 color="#404040")
    if live_tiers:
        fig.text(0.5, -0.10, f"Live-mode tiers: {','.join(live_tiers)}",
                 ha="center", fontsize=7.5, color="#404040")

    fig.tight_layout()

    args.fig_dir.mkdir(parents=True, exist_ok=True)
    pdf_path = args.fig_dir / "E4_tta_curves.pdf"
    png_path = args.fig_dir / "E4_tta_curves.png"
    fig.savefig(pdf_path, bbox_inches="tight", metadata={"CreationDate": None})
    fig.savefig(png_path, bbox_inches="tight", dpi=180)
    plt.close(fig)

    print(f"E4: wrote {pdf_path}")
    print(f"E4: wrote {png_path}")

    # Echo the table for §V.
    print()
    print(f"  {'tier':<6} {'n':>3} {'put_p50':>9} {'first_ok_p50':>13} "
          f"{'first_ok_p95':>13} {'first_ok_p99':>13} {'timeouts':>9}")
    for tier in TIER_ORDER:
        xs = by_tier[tier]
        if not xs:
            continue
        puts = [s["tiers"][tier]["put_latency_ms"] for s in samples
                if tier in s.get("tiers", {})]
        timeouts = sum(1 for s in samples
                       if s.get("tiers", {}).get(tier, {}).get("timed_out"))
        arr = np.array(xs, dtype=np.int64)
        put_arr = np.array(puts, dtype=np.int64)
        print(f"  {tier:<6} {len(xs):>3} "
              f"{int(np.percentile(put_arr, 50)):>7}ms "
              f"{int(np.percentile(arr, 50)):>11}ms "
              f"{int(np.percentile(arr, 95)):>11}ms "
              f"{int(np.percentile(arr, 99)):>11}ms "
              f"{timeouts:>9}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
