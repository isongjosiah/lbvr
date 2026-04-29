package main

// E4 bundle sampling: pick N rows from sizes.csv with a seeded shuffle.
// Identical contract to cmd/bench/e9/sampler.go (proportional to the
// measured 100K Synthea distribution); duplicated here because E4
// doesn't pull in any other e9 internals and a future per-tier sample
// stratification could diverge.

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	mrand "math/rand"
	"os"
	"strconv"
)

// SampledBundle is one row from sizes.csv. The bytes are not loaded —
// the bench synthesises payloads in the driver to avoid pulling 430 GiB
// of FHIR JSON through the test harness.
type SampledBundle struct {
	Filename  string
	SizeBytes int64
}

// loadSizes reads sizes.csv (header row "filename,size_bytes") into a
// slice. The slice is *not* shuffled — caller's responsibility.
func loadSizes(path string) ([]SampledBundle, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if len(header) < 2 || header[0] != "filename" || header[1] != "size_bytes" {
		return nil, fmt.Errorf("unexpected header %v", header)
	}

	var out []SampledBundle
	for {
		row, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}
		size, err := strconv.ParseInt(row[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse size %q: %w", row[1], err)
		}
		out = append(out, SampledBundle{Filename: row[0], SizeBytes: size})
	}
	return out, nil
}

// sampleN returns N bundles drawn proportionally to the measured
// distribution: shuffle the full corpus with the seeded RNG, take the
// first N. This mirrors cmd/bench/e9 — proportional sampling preserves
// the right-skew of the 100K corpus (P50 = 2.23 MB, P99 = 37.9 MB).
func sampleN(all []SampledBundle, n int, rng *mrand.Rand) ([]SampledBundle, error) {
	if n <= 0 {
		return nil, fmt.Errorf("sampleN: n=%d not positive", n)
	}
	if n > len(all) {
		return nil, fmt.Errorf("sampleN: requested %d bundles but corpus only has %d", n, len(all))
	}
	idx := rng.Perm(len(all))[:n]
	out := make([]SampledBundle, n)
	for i, j := range idx {
		out[i] = all[j]
	}
	return out, nil
}
