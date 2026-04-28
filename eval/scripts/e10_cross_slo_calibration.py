#!/usr/bin/env python3
"""E10 — Cross-SLO calibration. Post-processes E2's LBVR latency
distribution against multiple safety-critical-domain SLO thresholds
to identify which decentralized-storage-fabric configurations are
feasible per regime.

CLAUDE.md §8 row E10 + §1 four-contribution claim #3. Produces Fig 11.
"""
from __future__ import annotations
import json
import sys
from pathlib import Path

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np

# Domain SLO thresholds, sources in CLAUDE.md §14 references.
# Each tuple: (label, threshold_ms, regime, note).
SLO_REGIMES = [
    ("Power-grid GOOSE\n(IEC 61850)",      4,    "infeasible",      "trip-class real-time"),
    ("Clinical chart-pull",                500,  "stretch",         "EHR open + scroll"),
    ("Clinical radiology open",            2000, "core",            "PACS image read"),
    ("Scientific-DT\n(interTwin)",         5000, "comfortable",     "Manzi 2025 typical"),
    ("Clinical alarm\n(IEC 60601-1-8)",    10000,"comfortable",     "high-priority alert"),
]

# Colour by regime classification.
COLOURS = {
    "infeasible":   "#c0392b",
    "stretch":      "#e67e22",
    "core":         "#2c7bb6",
    "comfortable":  "#1a9850",
}


def load_lbvr_latencies_ms(e2_path: Path) -> np.ndarray:
    """Pull every LBVR-mode rep latency from E2's run JSON."""
    doc = json.loads(e2_path.read_text())
    out: list[float] = []
    for sample in doc["samples"]:
        runs = sample["mode_runs"].get("lbvr") or sample["mode_runs"].get("LBVR")
        if not runs:
            continue
        for r in runs:
            out.append(r["latency_ns"] / 1e6)  # ns → ms
    return np.array(out, dtype=np.float64)


def newest(pattern: str, root: Path) -> Path:
    candidates = sorted(root.glob(pattern))
    if not candidates:
        sys.exit(f"no files matching {pattern} under {root}")
    return candidates[-1]


def main() -> int:
    repo = Path(__file__).resolve().parent.parent.parent
    e2_path = newest("run-*.json", repo / "eval" / "results" / "E2")
    print(f"E10 source: {e2_path.relative_to(repo)}")

    lat_ms = load_lbvr_latencies_ms(e2_path)
    if lat_ms.size == 0:
        sys.exit("no LBVR latencies found in E2 run")
    print(f"LBVR samples: n={lat_ms.size}, P50={np.percentile(lat_ms,50):.1f}ms, "
          f"P95={np.percentile(lat_ms,95):.1f}ms, P99={np.percentile(lat_ms,99):.1f}ms, "
          f"max={lat_ms.max():.1f}ms")

    # Per-regime: % of LBVR retrievals meeting that SLO.
    rows = []
    for label, thresh_ms, regime, note in SLO_REGIMES:
        meeting = float((lat_ms <= thresh_ms).sum()) / lat_ms.size
        rows.append((label, thresh_ms, regime, note, meeting))

    # Sort by threshold ascending so the bar chart reads strictness-first.
    rows.sort(key=lambda r: r[1])

    print()
    print(f"  {'Regime':<32} {'SLO (ms)':>10} {'% meeting':>10} {'class':<14} note")
    print("  " + "-" * 90)
    for label, thresh, regime, note, meet in rows:
        print(f"  {label.replace(chr(10),' '):<32} {thresh:>10} {meet*100:>9.1f}% {regime:<14} {note}")

    # --- Figure: horizontal bar chart, one bar per regime, height = %
    #     meeting that SLO. Colour by regime classification. ---
    fig, ax = plt.subplots(figsize=(9, 4.5), dpi=150)
    labels = [r[0] for r in rows]
    pcts = [r[4] * 100 for r in rows]
    classes = [r[2] for r in rows]
    bar_colours = [COLOURS[c] for c in classes]
    y = np.arange(len(labels))

    bars = ax.barh(y, pcts, color=bar_colours, edgecolor="black", linewidth=0.5)
    ax.set_yticks(y)
    ax.set_yticklabels(labels, fontsize=9)
    ax.set_xlabel("% of LBVR-Med retrievals meeting the regime's SLO", fontsize=10)
    ax.set_xlim(0, 105)
    ax.set_title(
        f"Cross-SLO calibration — LBVR-Med fast path (n={lat_ms.size:,} samples, "
        f"P50 {np.percentile(lat_ms,50):.0f}ms / P99 {np.percentile(lat_ms,99):.0f}ms)",
        fontsize=10,
    )
    for bar, pct, regime in zip(bars, pcts, classes):
        x = bar.get_width()
        ax.annotate(
            f"{pct:.1f}%",
            xy=(min(x + 1, 102), bar.get_y() + bar.get_height() / 2),
            ha="left", va="center", fontsize=9,
        )
    ax.grid(True, axis="x", alpha=0.3)

    # Legend by regime class.
    handles = [
        plt.Rectangle((0, 0), 1, 1, color=COLOURS["infeasible"]),
        plt.Rectangle((0, 0), 1, 1, color=COLOURS["stretch"]),
        plt.Rectangle((0, 0), 1, 1, color=COLOURS["core"]),
        plt.Rectangle((0, 0), 1, 1, color=COLOURS["comfortable"]),
    ]
    ax.legend(
        handles,
        ["infeasible (LBVR cannot meet)", "stretch (P99 close to SLO)",
         "core target (clinical)", "comfortable (well within SLO)"],
        loc="lower right", fontsize=8, framealpha=0.9,
    )
    fig.tight_layout()

    out_dir = repo / "paper" / "figures"
    out_dir.mkdir(parents=True, exist_ok=True)
    pdf = out_dir / "E10_cross_slo_calibration.pdf"
    png = out_dir / "E10_cross_slo_calibration.png"
    fig.savefig(pdf, format="pdf")
    fig.savefig(png, format="png", dpi=150)
    plt.close(fig)
    print(f"\n→ wrote {pdf.relative_to(repo)}")
    print(f"→ wrote {png.relative_to(repo)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
