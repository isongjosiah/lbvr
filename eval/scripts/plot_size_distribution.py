#!/usr/bin/env python3
"""Plot bundle-size distribution for a Synthea FHIR corpus.

CLAUDE.md §10 D2 requires a "bundle size distribution plot" as a D2 deliverable.
This script emits:
  - size_distribution.pdf  — log-scale histogram with 128 KB RS(2,3) floor +
    5 MB §4.5 reference ceiling annotated.
  - size_distribution.png  — same plot as PNG (for quick review).
  - sizes.csv              — per-bundle (filename,size_bytes) raw data.
  - size_stats.json        — summary stats, same schema as validator's erasure_fit.

Intended to be invoked via: make plot-synthea-{1k,10k,100k}
"""
from __future__ import annotations
import argparse
import csv
import json
import statistics
import sys
from datetime import datetime, timezone
from pathlib import Path

import matplotlib
matplotlib.use("Agg")  # no display on a headless WSL box
import matplotlib.pyplot as plt
import numpy as np


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    p.add_argument("fhir_dir", type=Path)
    p.add_argument("--out-dir", type=Path, required=True)
    p.add_argument("--corpus-size", type=int, help="Intended patient count (for titles)")
    args = p.parse_args()

    if not args.fhir_dir.is_dir():
        print(f"✗ {args.fhir_dir} is not a directory", file=sys.stderr)
        return 2

    json_files = sorted(args.fhir_dir.glob("*.json"))
    # exclude hospitalInformation* and practitionerInformation* metadata files
    patient_files = [f for f in json_files if not f.name.startswith(("hospitalInformation", "practitionerInformation"))]
    n = len(patient_files)
    if n == 0:
        print(f"✗ no patient bundles in {args.fhir_dir}", file=sys.stderr)
        return 1

    sizes = np.array([f.stat().st_size for f in patient_files], dtype=np.int64)
    args.out_dir.mkdir(parents=True, exist_ok=True)

    # --- raw CSV (enables post-hoc analysis without regenerating) ---
    csv_path = args.out_dir / "sizes.csv"
    with csv_path.open("w", newline="") as fh:
        w = csv.writer(fh)
        w.writerow(["filename", "size_bytes"])
        for f, s in zip(patient_files, sizes):
            w.writerow([f.name, int(s)])
    print(f"→ wrote {csv_path}  ({n} rows)")

    # --- summary stats JSON ---
    below_128kb = int((sizes < 128 * 1024).sum())
    in_range = int(((sizes >= 128 * 1024) & (sizes <= 5 * 1024 * 1024)).sum())
    above_5mb = int((sizes > 5 * 1024 * 1024).sum())
    stats = {
        "generated_at": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "intended_corpus_size": args.corpus_size,
        "bundle_count": n,
        "size_bytes": {
            "min": int(sizes.min()),
            "p05": int(np.percentile(sizes, 5)),
            "p50": int(np.percentile(sizes, 50)),
            "p95": int(np.percentile(sizes, 95)),
            "p99": int(np.percentile(sizes, 99)),
            "max": int(sizes.max()),
            "mean": int(sizes.mean()),
            "total": int(sizes.sum()),
        },
        "erasure_fit": {
            "below_128kb": below_128kb,
            "in_128kb_to_5mb": in_range,
            "above_5mb": above_5mb,
        },
    }
    stats_path = args.out_dir / "size_stats.json"
    stats_path.write_text(json.dumps(stats, indent=2, sort_keys=True) + "\n")
    print(f"→ wrote {stats_path}")

    # --- histogram ---
    fig, ax = plt.subplots(figsize=(7.2, 4.2))
    # log-spaced bins between the smallest and largest observed size
    lo, hi = max(sizes.min(), 1), sizes.max()
    bins = np.logspace(np.log10(lo), np.log10(hi), 60)
    ax.hist(sizes, bins=bins, color="#3b6fb6", edgecolor="black", linewidth=0.4)
    ax.set_xscale("log")
    ax.set_xlabel("FHIR bundle size (bytes, log scale)")
    ax.set_ylabel("bundle count")
    corpus_label = f"Synthea {args.corpus_size:,}" if args.corpus_size else f"Synthea ({n} bundles)"
    ax.set_title(f"{corpus_label} — FHIR R4 bundle size distribution")
    ax.grid(True, alpha=0.25, which="both")

    # annotations: §4.5 floor (128 KB) and ceiling (5 MB)
    ax.axvline(128 * 1024, color="crimson", linestyle="--", linewidth=1.2,
               label=f"§4.5 RS(2,3) floor (128 KB): {below_128kb} below")
    ax.axvline(5 * 1024 * 1024, color="darkorange", linestyle="--", linewidth=1.2,
               label=f"§4.5 reference ceiling (5 MB): {above_5mb} above")
    ax.axvline(float(np.percentile(sizes, 50)), color="black", linestyle=":", linewidth=1.0,
               label=f"P50 ({int(np.percentile(sizes, 50) / 1024)} KB)")
    ax.legend(loc="upper right", fontsize=8, framealpha=0.9)
    fig.tight_layout()

    pdf_path = args.out_dir / "size_distribution.pdf"
    png_path = args.out_dir / "size_distribution.png"
    fig.savefig(pdf_path, format="pdf")
    fig.savefig(png_path, format="png", dpi=150)
    plt.close(fig)
    print(f"→ wrote {pdf_path}")
    print(f"→ wrote {png_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
