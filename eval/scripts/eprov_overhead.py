#!/usr/bin/env python3
"""Post-processor for E-PROV cryptographic-provenance overhead.

Reads the latest run-*.json under eval/results/E-PROV/ (or an explicit
path), computes per-stage P50/P95/P99 in MICROSECONDS (most stages are
sub-millisecond), and emits two figures:

  paper/figures/E_PROV_overhead.{pdf,png}
        Horizontal bar chart per stage; bar length = P50, error bar = P95.
  paper/figures/E_PROV_tamper_detection.{pdf,png}
        Table figure: case, n_tested, n_caught, detection_rate.

CLAUDE.md §4.6 / §8 contract: figure must be reproducible from the raw
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


# Stable per-stage colours; gradient blue → orange so a reviewer scans
# left-to-right by intuition (cheap → expensive).
STAGE_ORDER = ["gen", "canon", "sign", "anchor", "setroot", "verify"]
STAGE_COLOURS = {
    "gen":     "#9ecae1",  # pale blue — cheapest
    "canon":   "#6baed6",
    "sign":    "#fdae61",  # orange — BLS
    "anchor":  "#d7191c",  # red — wall-clock dominated
    "setroot": "#fdae61",
    "verify":  "#fd8d3c",  # darker orange — BLS again
}
STAGE_FIELD = {
    "gen":     "t_gen_ns",
    "canon":   "t_canon_ns",
    "sign":    "t_sign_ns",
    "anchor":  "t_anchor_ns",
    "setroot": "t_setroot_ns",
    "verify":  "t_verify_ns",
}

TAMPER_CASES = [
    "happy",
    "hash_tamper",
    "sig_tamper",
    "signer_substitute",
    "quorum_reduce",
    "missing_sig",
    "timestamp_tamper",
]


def find_latest_run(out_dir: Path) -> Path:
    runs = sorted(out_dir.glob("run-*.json"))
    if not runs:
        raise FileNotFoundError(f"no run-*.json in {out_dir}")
    return runs[-1]  # timestamped prefix sorts lexicographically


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    p.add_argument("--source", "--run", dest="source", type=Path,
                   help="explicit run-*.json path")
    p.add_argument("--out-dir", type=Path, default=Path("eval/results/E-PROV"),
                   help="search dir when --source is omitted")
    p.add_argument("--fig-dir", type=Path, default=Path("paper/figures"),
                   help="output directory for PDF + PNG")
    args = p.parse_args()

    run_path = args.source if args.source else find_latest_run(args.out_dir)
    rec = json.loads(run_path.read_text())
    samples = rec.get("samples", [])
    if not samples:
        print(f"! no samples in {run_path}", file=sys.stderr)
        return 1

    cfg = rec.get("config", {})
    n = cfg.get("n", len(samples))

    # Per-stage latency arrays in microseconds.
    by_stage: dict[str, np.ndarray] = {}
    for st, field in STAGE_FIELD.items():
        xs = np.asarray([s.get(field, 0) for s in samples], dtype=np.int64) / 1e3
        by_stage[st] = xs

    # ── stdout: stage table ─────────────────────────────────────────
    print(f"E-PROV run: {rec.get('run_id', '?')}  ({n} iterations)")
    print(f"  source: {run_path}")
    print(f"  anchor model: {cfg.get('anchor_model', '?')}")
    print()
    print(f"  {'stage':<8}  {'P50_us':>10}  {'P95_us':>10}  {'P99_us':>10}  {'mean_us':>10}")
    for st in STAGE_ORDER:
        xs = by_stage[st]
        print(f"  {st:<8}  {np.percentile(xs, 50):>10.2f}  "
              f"{np.percentile(xs, 95):>10.2f}  "
              f"{np.percentile(xs, 99):>10.2f}  "
              f"{xs.mean():>10.2f}")

    # ── stdout: tampering detection table ───────────────────────────
    by_case: dict[str, list[dict]] = {c: [] for c in TAMPER_CASES}
    for s in samples:
        c = s.get("tampering")
        if c in by_case:
            by_case[c].append(s)

    case_rows = []
    print()
    print(f"  {'case':<22}  {'n_tested':>8}  {'n_caught':>8}  {'rate':>8}")
    for c in TAMPER_CASES:
        rows = by_case[c]
        n_tested = len(rows)
        # 'happy' — caught == verified clean (Valid=true). Others —
        # caught == verifier rejected (Valid=false).
        if c == "happy":
            n_caught = sum(1 for r in rows if r.get("valid"))
        else:
            n_caught = sum(1 for r in rows if not r.get("valid"))
        rate = (n_caught / n_tested) if n_tested else 0.0
        print(f"  {c:<22}  {n_tested:>8d}  {n_caught:>8d}  {rate*100:>7.1f}%")
        case_rows.append((c, n_tested, n_caught, rate))

    # ── figure 1: per-stage overhead bar chart ──────────────────────
    fig, ax = plt.subplots(figsize=(7.6, 4.4))
    p50 = [np.percentile(by_stage[st], 50) for st in STAGE_ORDER]
    p95 = [np.percentile(by_stage[st], 95) for st in STAGE_ORDER]
    err = np.maximum(np.asarray(p95) - np.asarray(p50), 0.0)
    y = np.arange(len(STAGE_ORDER))
    colours = [STAGE_COLOURS[st] for st in STAGE_ORDER]
    ax.barh(y, p50, xerr=[np.zeros_like(err), err],
            color=colours, edgecolor="#222", linewidth=0.5,
            capsize=4, error_kw={"elinewidth": 1.0, "capthick": 1.0})
    ax.set_yticks(y)
    ax.set_yticklabels(STAGE_ORDER)
    ax.invert_yaxis()  # gen at top, verify at bottom
    ax.set_xscale("log")
    ax.set_xlabel("Latency (microseconds, log scale) — bar = P50, whisker = P95")
    ax.set_title(f"E-PROV per-stage overhead — {n} iterations")
    ax.grid(True, axis="x", alpha=0.25, which="both")

    # Annotate each bar with its P50 numerically (right-aligned just
    # past the bar end). Helps reviewers read sub-ms values that are
    # hard to compare on log-scale.
    for i, (val, e) in enumerate(zip(p50, err)):
        label = f"{val:,.1f} us" if val < 1000 else f"{val/1000:.2f} ms"
        ax.text(val + e, i, "  " + label, va="center", fontsize=9)

    fig.tight_layout()
    args.fig_dir.mkdir(parents=True, exist_ok=True)
    pdf_path = args.fig_dir / "E_PROV_overhead.pdf"
    png_path = args.fig_dir / "E_PROV_overhead.png"
    fig.savefig(pdf_path, format="pdf")
    fig.savefig(png_path, format="png", dpi=150)
    plt.close(fig)
    print()
    print(f"→ wrote {pdf_path}")
    print(f"→ wrote {png_path}")

    # ── figure 2: tampering-detection table ─────────────────────────
    fig2, ax2 = plt.subplots(figsize=(7.6, 2.6))
    ax2.axis("off")
    table_data = [["case", "n_tested", "n_caught", "detection rate"]]
    for c, n_tested, n_caught, rate in case_rows:
        table_data.append([c, str(n_tested), str(n_caught), f"{rate*100:.1f}%"])

    table = ax2.table(cellText=table_data, loc="center", cellLoc="left",
                      colWidths=[0.34, 0.14, 0.14, 0.18])
    table.auto_set_font_size(False)
    table.set_fontsize(9)
    table.scale(1.0, 1.4)

    # Header row styling.
    for col in range(4):
        cell = table[(0, col)]
        cell.set_facecolor("#dddddd")
        cell.set_text_props(weight="bold")

    # Colour-code detection rate column: green if 100%, red otherwise.
    for row_idx, (_, _, _, rate) in enumerate(case_rows, start=1):
        cell = table[(row_idx, 3)]
        cell.set_facecolor("#c7e9c0" if rate >= 1.0 else "#fcae91")

    ax2.set_title(f"E-PROV tamper detection — {n} iterations", fontsize=11, pad=12)
    fig2.tight_layout()
    pdf2 = args.fig_dir / "E_PROV_tamper_detection.pdf"
    png2 = args.fig_dir / "E_PROV_tamper_detection.png"
    fig2.savefig(pdf2, format="pdf")
    fig2.savefig(png2, format="png", dpi=150)
    plt.close(fig2)
    print(f"→ wrote {pdf2}")
    print(f"→ wrote {png2}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
