#!/usr/bin/env python3
"""E7 — Storage cost analysis. Analytical, no benchmark run required.

Compares LBVR-Med's RS(2,3) cross-tier placement against four
architectural alternatives at the measured 100K-corpus scale (430 GiB
plaintext per docs/eval-protocol.md). Produces Fig 8.

Tier rates are indicative as of 2026-Q1; the script writes the
assumptions block into the run JSON so a reviewer can re-compute.
Live-deployment numbers should replace these with measured invoices.
"""
from __future__ import annotations
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np

# 100K Synthea corpus measured plaintext size, in GiB.
# Source: eval/results/synthea-100000/validation.json (mean 3.93 MB ×
# 114,693 bundles ≈ 430.25 GiB).
PLAINTEXT_GIB = 430.25

# Indicative monthly $/GiB rates as of 2026-Q1. Document the source.
# Hot/warm/cold sources mirror CLAUDE.md §7 dependency choices.
RATES = {
    "pinata":   0.200,    # Pinata Picnic plan: $20/mo per 100 GB.
    "filebase": 0.006,    # Filebase: $5.99/TB/mo = $0.006/GiB/mo.
    "arweave":  0.029,    # Irys/Arweave amortised perpetual at 60-month
                          # horizon: $1.74/GiB ÷ 60 ≈ $0.029/GiB/mo.
                          # The sunk cost is perpetual storage; we
                          # amortise so the 30-day comparison is
                          # apples-to-apples with monthly tiers.
    "s3":       0.023,    # AWS S3 Standard: $0.023/GiB/mo.
}

# Architectures compared. Each row has (label, blow-up factor,
# per-replica tier mapping that produces the cost).
def cost_for(arch_label: str, replicas) -> tuple[float, list[tuple[str, float, float]]]:
    """replicas is a list of (tier_name, GiB) tuples — pure replicas
    (whole copies) or RS shards (fractional copies). Returns total
    monthly cost + per-replica breakdown."""
    breakdown = []
    total = 0.0
    for tier, gib in replicas:
        rate = RATES[tier]
        cost = gib * rate
        breakdown.append((tier, gib, cost))
        total += cost
    return total, breakdown

ARCHITECTURES = [
    ("Pinata-only (no redundancy)",
     [("pinata", PLAINTEXT_GIB)]),
    ("S3 single-region",
     [("s3", PLAINTEXT_GIB)]),
    ("S3 3-region replication",
     [("s3", 3 * PLAINTEXT_GIB)]),
    ("Naive 3× hot replication\n(Pinata × 3)",
     [("pinata", 3 * PLAINTEXT_GIB)]),
    ("LBVR-Med RS(2,3)\n(hot+warm+cold)",
     [("pinata",   PLAINTEXT_GIB / 2),  # D0
      ("filebase", PLAINTEXT_GIB / 2),  # D1
      ("arweave",  PLAINTEXT_GIB / 2)]),  # P0
]


def main() -> int:
    rows = []
    for label, replicas in ARCHITECTURES:
        total, breakdown = cost_for(label, replicas)
        rows.append((label, total, breakdown))

    repo = Path(__file__).resolve().parent.parent.parent
    out_dir = repo / "eval" / "results" / "E7"
    out_dir.mkdir(parents=True, exist_ok=True)

    # JSON record of inputs + result so a reviewer can re-compute.
    record = {
        "schema_version": 1,
        "computed_at": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "plaintext_gib": PLAINTEXT_GIB,
        "rates_per_gib_per_month": RATES,
        "architectures": [
            {
                "label": label.replace("\n", " "),
                "monthly_total_usd": round(total, 2),
                "breakdown": [
                    {"tier": t, "gib": round(g, 2), "monthly_usd": round(c, 2)}
                    for t, g, c in breakdown
                ],
            }
            for label, total, breakdown in rows
        ],
    }
    (out_dir / "e7_costs.json").write_text(json.dumps(record, indent=2) + "\n")
    print(f"→ wrote {(out_dir / 'e7_costs.json').relative_to(repo)}")

    print()
    print(f"  {'Architecture':<48} {'$/month':>10}")
    print("  " + "-" * 60)
    for label, total, _ in rows:
        print(f"  {label.replace(chr(10), ' '):<48} {total:>10.2f}")

    # --- Figure: stacked horizontal bars; one bar per architecture,
    #     stacked by tier. Shows both the absolute $ and the source
    #     of the cost (which tier dominates). ---
    fig, ax = plt.subplots(figsize=(9, 4.5), dpi=150)
    labels = [r[0] for r in rows]
    y = np.arange(len(labels))
    tier_palette = {
        "pinata":   "#3b6fb6",  # blue (hot)
        "filebase": "#7fb050",  # green (warm)
        "arweave":  "#c0392b",  # red (cold)
        "s3":       "#888888",  # grey
    }
    # Draw each tier as a stacked segment.
    left = np.zeros(len(rows))
    used_tiers: set[str] = set()
    for tier in ["pinata", "filebase", "arweave", "s3"]:
        widths = []
        for _, _, breakdown in rows:
            w = sum(c for t, _, c in breakdown if t == tier)
            widths.append(w)
        widths_arr = np.array(widths)
        if widths_arr.sum() == 0:
            continue
        ax.barh(y, widths_arr, left=left, color=tier_palette[tier],
                edgecolor="black", linewidth=0.4, label=tier)
        left += widths_arr
        used_tiers.add(tier)

    # Total cost label at the right of each bar.
    for i, (_, total, _) in enumerate(rows):
        ax.annotate(f"${total:.0f}/mo",
                    xy=(total + max(left.max(), 1) * 0.01, i),
                    ha="left", va="center", fontsize=9)

    ax.set_yticks(y)
    ax.set_yticklabels(labels, fontsize=9)
    ax.set_xlabel(f"Monthly storage cost (USD) — 100K corpus, {PLAINTEXT_GIB:.0f} GiB plaintext",
                  fontsize=10)
    ax.set_xlim(0, left.max() * 1.18)
    ax.set_title(
        "Storage-cost comparison — LBVR-Med RS(2,3) vs centralized + naive-replication baselines",
        fontsize=10,
    )
    ax.grid(True, axis="x", alpha=0.3)
    ax.legend(loc="lower right", fontsize=8, framealpha=0.9, title="tier")
    fig.tight_layout()

    pdf = repo / "paper" / "figures" / "E7_storage_cost.pdf"
    png = repo / "paper" / "figures" / "E7_storage_cost.png"
    fig.savefig(pdf, format="pdf")
    fig.savefig(png, format="png", dpi=150)
    plt.close(fig)
    print(f"\n→ wrote {pdf.relative_to(repo)}")
    print(f"→ wrote {png.relative_to(repo)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
