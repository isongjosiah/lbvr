#!/usr/bin/env python3
"""Validate a Synthea FHIR R4 corpus: bundle count, structure, size distribution, resource mix.

Emits exit code 0 on success, 1 on validation error, 2 on bad invocation.
If --stats-out is given, writes a machine-readable JSON summary there for the
D2 artifacts pipeline (CLAUDE.md §10). No external deps — only stdlib.

Usage:
    validate_synthea.py <fhir_dir> [--sample N] [--min-bundles N]
                        [--stats-out FILE] [--corpus-size N]
"""
from __future__ import annotations
import argparse
import json
import re
import statistics
import subprocess
import sys
import time
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path


# Known-valid FHIR R4 resource types Synthea is expected to emit. Unknown types
# don't fail validation — they only get reported so a human can spot-check.
FHIR_R4_KNOWN = {
    "AllergyIntolerance", "CarePlan", "CareTeam", "Claim", "Condition", "Device",
    "DiagnosticReport", "DocumentReference", "Encounter", "ExplanationOfBenefit",
    "Goal", "ImagingStudy", "Immunization", "Location", "Media", "Medication",
    "MedicationAdministration", "MedicationRequest", "Observation", "Organization",
    "Patient", "Practitioner", "Procedure", "Provenance", "SupplyDelivery",
    "Bundle", "Composition", "Condition", "Coverage", "HealthcareService",
    "Questionnaire", "QuestionnaireResponse", "RelatedPerson", "Specimen",
    "ServiceRequest", "Task", "Appointment",
}

VALID_BUNDLE_TYPES = {"transaction", "collection", "batch"}


def scan_corpus(fhir_dir: Path) -> tuple[list[Path], list[Path]]:
    json_files = sorted(fhir_dir.glob("*.json"))
    meta = [p for p in json_files if p.name.startswith(("hospitalInformation", "practitionerInformation"))]
    patient = [p for p in json_files if p not in set(meta)]
    return patient, meta


def percentile(sorted_vals: list[int], q: float) -> int:
    if not sorted_vals:
        return 0
    idx = min(int(len(sorted_vals) * q), len(sorted_vals) - 1)
    return sorted_vals[idx]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("fhir_dir", type=Path, help="Synthea output/*/fhir directory")
    parser.add_argument("--sample", type=int, default=500,
                        help="N bundles to structurally validate (default 500)")
    parser.add_argument("--min-bundles", type=int, default=100)
    parser.add_argument("--stats-out", type=Path, help="Write JSON summary to this path")
    parser.add_argument("--corpus-size", type=int, help="Intended patient count (for metadata)")
    args = parser.parse_args()

    if not args.fhir_dir.is_dir():
        print(f"✗ {args.fhir_dir} is not a directory", file=sys.stderr)
        return 2

    started = time.monotonic()
    patient_files, meta_files = scan_corpus(args.fhir_dir)
    n = len(patient_files)
    print(f"→ found {n} patient bundles ({len(meta_files)} metadata)")
    if n < args.min_bundles:
        print(f"✗ expected ≥{args.min_bundles} bundles, got {n}", file=sys.stderr)
        return 1

    # --- size distribution ---
    sizes = sorted(p.stat().st_size for p in patient_files)
    size_stats = {
        "min": sizes[0],
        "p50": percentile(sizes, 0.50),
        "p95": percentile(sizes, 0.95),
        "p99": percentile(sizes, 0.99),
        "max": sizes[-1],
        "mean": int(statistics.mean(sizes)),
        "total": sum(sizes),
    }
    below_128kb = sum(1 for s in sizes if s < 128 * 1024)
    in_erasure_range = sum(1 for s in sizes if 128 * 1024 <= s <= 5 * 1024 * 1024)  # §4.5 floor/ceiling
    above_5mb = sum(1 for s in sizes if s > 5 * 1024 * 1024)

    kb = lambda b: b / 1024
    print("→ size distribution:")
    for k in ("min", "p50", "p95", "p99", "max", "mean"):
        print(f"    {k:<5} = {kb(size_stats[k]):>10.1f} KB")
    print(f"    total = {size_stats['total'] / (1024 ** 3):>10.2f} GiB")
    print(f"→ bundles <128 KB (below RS(2,3) floor, §4.5): {below_128kb}/{n} ({100 * below_128kb / n:.1f}%)")
    print(f"→ bundles in §4.5 range 128KB–5MB:             {in_erasure_range}/{n} ({100 * in_erasure_range / n:.1f}%)")
    print(f"→ bundles >5MB (above §4.5 reference):         {above_5mb}/{n} ({100 * above_5mb / n:.1f}%)")

    # --- structural sample ---
    sample_n = min(args.sample, n)
    step = max(1, n // sample_n)
    sample = patient_files[::step][:sample_n]
    print(f"→ sampling {len(sample)} bundles for FHIR structure checks")

    bundle_types: Counter[str] = Counter()
    resource_types: Counter[str] = Counter()
    unknown_types: Counter[str] = Counter()
    patients_alive = 0
    patients_deceased = 0
    errors: list[str] = []

    for path in sample:
        try:
            doc = json.loads(path.read_text())
        except json.JSONDecodeError as e:
            errors.append(f"{path.name}: invalid JSON ({e})")
            continue
        if doc.get("resourceType") != "Bundle":
            errors.append(f"{path.name}: resourceType={doc.get('resourceType')!r}")
            continue
        btype = doc.get("type")
        bundle_types[btype or "<missing>"] += 1
        if btype not in VALID_BUNDLE_TYPES:
            errors.append(f"{path.name}: Bundle.type={btype!r}")
        entries = doc.get("entry", []) or []
        saw_patient = False
        for ent in entries:
            rt = ent.get("resource", {}).get("resourceType")
            if not rt:
                continue
            resource_types[rt] += 1
            if rt not in FHIR_R4_KNOWN and not re.fullmatch(r"[A-Z][A-Za-z]+", rt):
                unknown_types[rt] += 1
            if rt == "Patient":
                saw_patient = True
                if ent.get("resource", {}).get("deceasedDateTime"):
                    patients_deceased += 1
                else:
                    patients_alive += 1
        if not saw_patient:
            errors.append(f"{path.name}: no Patient resource")

    print(f"→ Bundle.type distribution (sample): {dict(bundle_types)}")
    print(f"→ patient status (sample): {patients_alive} alive, {patients_deceased} deceased")
    print("→ top resource types (sample):")
    for rt, c in resource_types.most_common(10):
        marker = " " if rt in FHIR_R4_KNOWN else "?"
        print(f"   {marker} {rt:<28} {c}")
    if unknown_types:
        print(f"⚠ unknown resource types (not in R4 whitelist): {dict(unknown_types)}")

    if errors:
        print(f"✗ {len(errors)} validation errors (showing first 5):", file=sys.stderr)
        for e in errors[:5]:
            print(f"    {e}", file=sys.stderr)

    # --- JSON output ---
    if args.stats_out:
        try:
            synthea_sha = subprocess.check_output(
                ["git", "-C", str(args.fhir_dir.parent.parent), "rev-parse", "HEAD"],
                stderr=subprocess.DEVNULL,
            ).decode().strip()
        except (subprocess.CalledProcessError, FileNotFoundError):
            synthea_sha = None
        stats = {
            "schema_version": 1,
            "validated_at": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
            "fhir_dir": str(args.fhir_dir),
            "intended_corpus_size": args.corpus_size,
            "bundle_count": n,
            "metadata_file_count": len(meta_files),
            "size_bytes": size_stats,
            "erasure_fit": {
                "below_128kb": below_128kb,
                "in_128kb_to_5mb": in_erasure_range,
                "above_5mb": above_5mb,
            },
            "sample": {
                "size": len(sample),
                "bundle_types": dict(bundle_types),
                "resource_type_histogram": dict(resource_types),
                "unknown_resource_types": dict(unknown_types),
                "patients_alive": patients_alive,
                "patients_deceased": patients_deceased,
            },
            "errors": errors[:50],
            "synthea_git_sha": synthea_sha,
            "elapsed_seconds": round(time.monotonic() - started, 2),
        }
        args.stats_out.parent.mkdir(parents=True, exist_ok=True)
        args.stats_out.write_text(json.dumps(stats, indent=2, sort_keys=True) + "\n")
        print(f"→ wrote {args.stats_out}")

    if errors:
        return 1
    print(f"✓ {n} bundles valid "
          f"({in_erasure_range}/{n} erasure-eligible, "
          f"{below_128kb} below floor, "
          f"{above_5mb} above §4.5 5MB reference)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
