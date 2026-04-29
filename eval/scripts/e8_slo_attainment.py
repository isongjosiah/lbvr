#!/usr/bin/env python3
"""E8 — End-to-end SLO compliance synthesis.

CLAUDE.md §8 row E8: synthesises E1 (ingest), E2 (retrieval baseline),
E3 (retrieval under WAN stress), E4 (time-to-availability post-PUT) into
a single SLO attainment table. Output: paper/figures/E8_slo_attainment
.{pdf,png} + eval/results/E8/run-<id>.json.

Columns reference the four contributing experiments. Cell-by-cell
verdicts roll up to a composite verdict that reflects the strictest
limiting axis. The table is the §V "we meet clinical SLOs end-to-end"
talking point; without it the paper would fall back to "we meet the
retrieval-side SLO" (E10's claim) and lose the cross-pipeline framing.

E4 is on the D15 schedule and not yet committed. Until its results land,
the TTA column reads "pending" and the composite verdict ignores TTA.
A footnote in §V notes this; once E4 commits the script picks up the
new data automatically.
"""
from __future__ import annotations
import argparse
import datetime as dt
import glob
import json
import subprocess
import sys
from pathlib import Path

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np


# Clinical SLO thresholds (ms). Sources in CLAUDE.md §14 references.
# Ordered tightest-first so the table reads "most demanding" → "easiest".
SLOS = [
    ("Clinical chart-pull",            500,   "iec60601-cdss"),
    ("Clinical radiology open",        2000,  "iec60601-pacs"),
    ("IEC 60601-1-8 alarm",            10000, "iec60601-alarm"),
]

# E3 stress cell: 5% sustained loss × 200 ms RTT — the hardest cell in
# the E3 matrix that doesn't outright break the system. (10 ms × 5%
# loss is technically the worst absolute P99, but 200 ms × 5% is the
# more representative "WAN under adversity" picture per E3's analysis.)
STRESS_LOSS = 0.05
STRESS_RTT_MS = 200

# Verdict bands. PASS if P99 ≤ threshold; STRETCH if ≤ 2× threshold;
# FAIL otherwise. The "stretch" band exists because clinical SLOs are
# typically point targets (P99) but operations teams treat 2× as the
# gradual-degradation band before alarms fire.
VERDICT_COLOURS = {
    "PASS":    "#1a9850",
    "STRETCH": "#fdae61",
    "FAIL":    "#c0392b",
    "PEND":    "#9e9e9e",
}


def newest(pattern: str, root: Path) -> Path | None:
    candidates = sorted(root.glob(pattern))
    return candidates[-1] if candidates else None


def latencies_ms_from_e2(path: Path, mode: str = "lbvr") -> np.ndarray:
    doc = json.loads(path.read_text())
    out: list[float] = []
    for sample in doc["samples"]:
        runs = sample["mode_runs"].get(mode) or sample["mode_runs"].get(mode.upper())
        if not runs:
            continue
        for r in runs:
            out.append(r["latency_ns"] / 1e6)
    return np.array(out, dtype=np.float64)


def find_e3_stress_cell(path: Path) -> dict | None:
    """Return the {p99_ms, fast_path_pct, ...} dict for the canonical
    stress cell, or None if the matrix doesn't include it."""
    doc = json.loads(path.read_text())
    for c in doc["cells"]:
        if c["rtt_ms"] == STRESS_RTT_MS and abs(c["loss_rate"] - STRESS_LOSS) < 1e-6:
            return c["stats"]
    return None


def e1_ingest_p99_ms(path: Path) -> float | None:
    """Pull the largest scale's P99 total ingest time. We use the largest
    *measured* (not projected) scale — projections inflate P99 with
    extrapolation noise."""
    doc = json.loads(path.read_text())
    measured = [s for s in doc.get("scales", []) if not s.get("projected", False)]
    if not measured:
        return None
    largest = max(measured, key=lambda s: s["corpus_size"])
    return largest["stage_p99_ms"]["total"]


def e4_tta_p99_ms(path: Path | None) -> float | None:
    """Hot-tier first_ok_at_ms P99. Hot defines LBVR's fast-path TTA;
    warm and cold are documented in the E4 run-*.json but not rolled into
    the SLO verdict (the §V text explains the choice — clinician-perceived
    TTA is bounded by the fastest tier holding a shard, which is hot)."""
    if path is None:
        return None
    doc = json.loads(path.read_text())
    hot_first_oks: list[int] = []
    for s in doc.get("samples", []):
        hot = s.get("tiers", {}).get("hot")
        if not hot:
            continue
        hot_first_oks.append(hot["first_ok_at_ms"])
    if not hot_first_oks:
        return None
    return float(np.percentile(np.array(hot_first_oks, dtype=np.int64), 99))


def verdict(p99_ms: float, threshold_ms: float) -> str:
    if p99_ms <= threshold_ms:
        return "PASS"
    if p99_ms <= 2.0 * threshold_ms:
        return "STRETCH"
    return "FAIL"


def composite_verdict(verdicts: list[str]) -> str:
    """Strictest verdict wins, ignoring PEND cells."""
    real = [v for v in verdicts if v != "PEND"]
    if not real:
        return "PEND"
    if "FAIL" in real:
        return "FAIL"
    if "STRETCH" in real:
        return "STRETCH"
    return "PASS"


def short_commit() -> str:
    try:
        out = subprocess.check_output(["git", "rev-parse", "--short", "HEAD"], text=True)
        return out.strip()
    except subprocess.CalledProcessError:
        return "nogit"


def main() -> int:
    repo = Path(__file__).resolve().parent.parent.parent
    p = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    p.add_argument("--out-dir", type=Path, default=repo / "eval" / "results" / "E8")
    p.add_argument("--fig-dir", type=Path, default=repo / "paper" / "figures")
    args = p.parse_args()

    e1_path = newest("run-*.json", repo / "eval" / "results" / "E1")
    e2_path = newest("run-*.json", repo / "eval" / "results" / "E2")
    e3_path = newest("run-*.json", repo / "eval" / "results" / "E3")
    e4_path = newest("run-*.json", repo / "eval" / "results" / "E4")

    if not e2_path or not e3_path:
        print("E8: requires E2 and E3 runs; one or both missing", file=sys.stderr)
        return 1

    print(f"E8 sources:")
    print(f"  E1: {e1_path.relative_to(repo) if e1_path else '(missing)'}")
    print(f"  E2: {e2_path.relative_to(repo)}")
    print(f"  E3: {e3_path.relative_to(repo)}")
    print(f"  E4: {e4_path.relative_to(repo) if e4_path else '(pending — D15)'}")

    # --- per-axis derivations ---
    ingest_p99 = e1_ingest_p99_ms(e1_path) if e1_path else None
    lbvr_lat = latencies_ms_from_e2(e2_path)
    if lbvr_lat.size == 0:
        print("E8: no LBVR latencies in E2", file=sys.stderr)
        return 1
    retrieval_p99 = float(np.percentile(lbvr_lat, 99))

    stress_cell = find_e3_stress_cell(e3_path)
    if stress_cell is None:
        print(f"E8: E3 missing the {STRESS_RTT_MS}ms × {STRESS_LOSS:.0%} stress cell",
              file=sys.stderr)
        return 1
    stress_p99 = stress_cell["p99_ms"]

    tta_p99 = e4_tta_p99_ms(e4_path)

    # --- per-SLO verdicts ---
    rows = []
    for label, threshold_ms, slo_id in SLOS:
        v_ingest = verdict(ingest_p99, threshold_ms) if ingest_p99 is not None else "PEND"
        v_retrieval = verdict(retrieval_p99, threshold_ms)
        v_stress = verdict(stress_p99, threshold_ms)
        v_tta = "PEND" if tta_p99 is None else verdict(tta_p99, threshold_ms)
        # Composite excludes ingest deliberately: clinical SLOs are
        # retrieval-side; ingest is a separate column for visibility but
        # not a roll-up factor (a workflow can pre-stage data, decoupling
        # ingest latency from clinician-perceived latency).
        v_composite = composite_verdict([v_retrieval, v_stress, v_tta])
        rows.append({
            "slo_id": slo_id,
            "label": label,
            "threshold_ms": threshold_ms,
            "ingest_p99_ms": ingest_p99,
            "ingest_verdict": v_ingest,
            "retrieval_p99_ms": retrieval_p99,
            "retrieval_verdict": v_retrieval,
            "stress_p99_ms": stress_p99,
            "stress_verdict": v_stress,
            "tta_p99_ms": tta_p99,
            "tta_verdict": v_tta,
            "composite_verdict": v_composite,
        })

    # --- print table to stdout for §V draft ---
    print()
    print(f"  {'SLO':<26} {'thr':>6} {'ingest':>10} {'lbvr':>10} {'stress':>10} {'tta':>8} {'verdict':>8}")
    for r in rows:
        ingest_cell = f"{r['ingest_p99_ms']:.0f}/{r['ingest_verdict']}" if r['ingest_p99_ms'] else "PEND"
        tta_cell = f"{r['tta_p99_ms']:.0f}/{r['tta_verdict']}" if r['tta_p99_ms'] else "pending"
        print(f"  {r['label']:<26} {r['threshold_ms']:>5}ms "
              f"{ingest_cell:>10} "
              f"{r['retrieval_p99_ms']:>4.0f}ms/{r['retrieval_verdict']:<7} "
              f"{r['stress_p99_ms']:>4.0f}ms/{r['stress_verdict']:<7} "
              f"{tta_cell:>8} "
              f"{r['composite_verdict']:>8}")

    # --- figure: matplotlib table with verdict-coloured cells ---
    fig, ax = plt.subplots(figsize=(8.6, 2.6 + 0.45 * len(rows)))
    ax.axis("off")

    headers = [
        "SLO", "Threshold (ms)",
        "Ingest P99\n(E1, 10K)",
        "LBVR Retrieval P99\n(E2, baseline)",
        f"Stressed P99\n(E3, {STRESS_RTT_MS}ms × {int(STRESS_LOSS*100)}%)",
        "TTA P99\n(E4, pending)",
        "Composite",
    ]

    cell_text = []
    cell_colours = []
    for r in rows:
        ingest_text = f"{r['ingest_p99_ms']:.0f} / {r['ingest_verdict']}" if r['ingest_p99_ms'] else "—"
        tta_text = "pending" if r['tta_p99_ms'] is None else f"{r['tta_p99_ms']:.0f} / {r['tta_verdict']}"
        cell_text.append([
            r["label"].replace("\n", " "),
            f"{r['threshold_ms']}",
            ingest_text,
            f"{r['retrieval_p99_ms']:.0f} / {r['retrieval_verdict']}",
            f"{r['stress_p99_ms']:.0f} / {r['stress_verdict']}",
            tta_text,
            r["composite_verdict"],
        ])
        cell_colours.append([
            "#f6f6f6", "#f6f6f6",
            VERDICT_COLOURS.get(r["ingest_verdict"], "#f6f6f6"),
            VERDICT_COLOURS.get(r["retrieval_verdict"], "#f6f6f6"),
            VERDICT_COLOURS.get(r["stress_verdict"], "#f6f6f6"),
            VERDICT_COLOURS.get(r["tta_verdict"], "#f6f6f6"),
            VERDICT_COLOURS.get(r["composite_verdict"], "#f6f6f6"),
        ])

    table = ax.table(
        cellText=cell_text,
        colLabels=headers,
        cellColours=cell_colours,
        loc="center",
        cellLoc="center",
    )
    table.auto_set_font_size(False)
    table.set_fontsize(8)
    table.scale(1.0, 1.6)

    # Bold + dark header row.
    for i, key in enumerate(headers):
        cell = table[(0, i)]
        cell.set_facecolor("#34495e")
        cell.set_text_props(color="white", weight="bold", fontsize=8)

    # Whiten verdict-cell text where contrast against the colour fails.
    for r in range(len(rows)):
        for c in (2, 3, 4, 5, 6):
            cell = table[(r + 1, c)]
            v = cell_text[r][c].split(" / ")[-1] if " / " in cell_text[r][c] else cell_text[r][c]
            if cell_colours[r][c] in ("#1a9850", "#c0392b"):
                cell.set_text_props(color="white", weight="bold", fontsize=8)
            elif cell_colours[r][c] == "#fdae61":
                cell.set_text_props(color="#1a1a1a", weight="bold", fontsize=8)
            else:
                cell.set_text_props(color="#1a1a1a", fontsize=8)

    fig.suptitle(
        "E8 — End-to-end SLO attainment (composite of E1–E4 axes)",
        fontsize=10, weight="bold", y=0.96,
    )
    fig.text(
        0.5, 0.04,
        f"Cells show P99 latency (ms) / verdict. PASS ≤ threshold; STRETCH ≤ 2×; FAIL > 2×. "
        f"Composite excludes ingest (decouplable via pre-staging).",
        ha="center", fontsize=7.5, color="#404040",
    )

    args.fig_dir.mkdir(parents=True, exist_ok=True)
    pdf_path = args.fig_dir / "E8_slo_attainment.pdf"
    png_path = args.fig_dir / "E8_slo_attainment.png"
    fig.savefig(pdf_path, bbox_inches="tight", metadata={"CreationDate": None})
    fig.savefig(png_path, bbox_inches="tight", dpi=180)
    plt.close(fig)
    print()
    print(f"E8: wrote {pdf_path}")
    print(f"E8: wrote {png_path}")

    # --- JSON sidecar (machine-readable; CLAUDE.md §8 reproducibility) ---
    args.out_dir.mkdir(parents=True, exist_ok=True)
    now = dt.datetime.now(dt.timezone.utc)
    run_id = now.strftime("%Y%m%d-%H%M%S") + "-" + short_commit()
    sidecar = {
        "schema_version": 1,
        "run_id": run_id,
        "started_at": now.isoformat(),
        "completed_at": now.isoformat(),
        "config": {
            "stress_cell_rtt_ms": STRESS_RTT_MS,
            "stress_cell_loss_rate": STRESS_LOSS,
            "verdict_stretch_factor": 2.0,
            "composite_excludes_ingest": True,
            "composite_treats_pending_axes_as_neutral": True,
        },
        "sources": {
            "e1": str(e1_path.relative_to(repo)) if e1_path else None,
            "e2": str(e2_path.relative_to(repo)),
            "e3": str(e3_path.relative_to(repo)),
            "e4": str(e4_path.relative_to(repo)) if e4_path else None,
        },
        "rows": rows,
    }
    sidecar_path = args.out_dir / f"run-{run_id}.json"
    sidecar_path.write_text(json.dumps(sidecar, indent=2) + "\n")
    print(f"E8: wrote {sidecar_path.relative_to(repo)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
