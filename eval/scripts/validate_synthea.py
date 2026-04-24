#!/usr/bin/env python3
"""Validate a Synthea FHIR R4 corpus: bundle count, structural soundness, size distribution.

Usage:
    python validate_synthea.py eval/synthea/upstream/output-1000/fhir/

Checks:
  1. All files parse as JSON.
  2. Each file has resourceType=Bundle and type=transaction|collection.
  3. Bundle entries include at least one Patient resource (sanity).
  4. Emit size distribution histogram (helps plan erasure coding §4.5: bundles < 128KB
     should not be used for RS(2,3) tests per CLAUDE.md).
"""
from __future__ import annotations
import argparse
import json
import os
import sys
from collections import Counter
from pathlib import Path


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("fhir_dir", type=Path, help="Synthea output/fhir directory")
    parser.add_argument("--sample", type=int, default=5, help="N bundles to structurally validate")
    parser.add_argument("--min-bundles", type=int, default=100)
    args = parser.parse_args()

    if not args.fhir_dir.is_dir():
        print(f"✗ {args.fhir_dir} is not a directory", file=sys.stderr)
        return 2

    json_files = sorted(args.fhir_dir.glob("*.json"))
    patient_files = [p for p in json_files if not p.name.startswith(("hospitalInformation", "practitionerInformation"))]
    n = len(patient_files)
    print(f"→ found {n} patient bundles (+ {len(json_files) - n} metadata files)")
    if n < args.min_bundles:
        print(f"✗ expected ≥{args.min_bundles} bundles, got {n}", file=sys.stderr)
        return 1

    # Size distribution
    sizes = [p.stat().st_size for p in patient_files]
    sizes.sort()
    kb = lambda b: b / 1024
    print("→ size distribution:")
    print(f"    min  = {kb(sizes[0]):>9.1f} KB")
    print(f"    P50  = {kb(sizes[n // 2]):>9.1f} KB")
    print(f"    P95  = {kb(sizes[int(n * 0.95)]):>9.1f} KB")
    print(f"    max  = {kb(sizes[-1]):>9.1f} KB")
    print(f"    mean = {kb(sum(sizes) / n):>9.1f} KB")
    below_128kb = sum(1 for s in sizes if s < 128 * 1024)
    print(f"→ bundles < 128 KB (unfit for RS(2,3) per CLAUDE.md §4.5): {below_128kb}/{n} ({100*below_128kb/n:.1f}%)")

    # Structural sample
    print(f"→ sampling {min(args.sample, n)} bundles for FHIR structure checks")
    resource_types: Counter[str] = Counter()
    for path in patient_files[:: max(1, n // args.sample)][: args.sample]:
        try:
            doc = json.loads(path.read_text())
        except json.JSONDecodeError as e:
            print(f"✗ {path.name}: invalid JSON ({e})", file=sys.stderr)
            return 1
        if doc.get("resourceType") != "Bundle":
            print(f"✗ {path.name}: resourceType={doc.get('resourceType')!r}, expected 'Bundle'", file=sys.stderr)
            return 1
        if doc.get("type") not in ("transaction", "collection", "batch"):
            print(f"✗ {path.name}: Bundle.type={doc.get('type')!r}", file=sys.stderr)
            return 1
        entries = doc.get("entry", []) or []
        types = [e.get("resource", {}).get("resourceType") for e in entries]
        resource_types.update(t for t in types if t)
        if "Patient" not in types:
            print(f"✗ {path.name}: no Patient resource in bundle", file=sys.stderr)
            return 1

    print("→ resource types observed (top 10):")
    for rt, c in resource_types.most_common(10):
        print(f"    {rt:<30} {c}")

    print(f"✓ {n} bundles valid, {below_128kb} under the 128KB erasure floor")
    return 0


if __name__ == "__main__":
    sys.exit(main())
