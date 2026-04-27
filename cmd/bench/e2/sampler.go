// Bundle sampling for E2 — copy of cmd/bench/e9/sampler.go so the two
// benches sample from the same 100K Synthea distribution with the same
// reproducibility contract. Decision (CLAUDE.md §10 D9, docs/eval-protocol.md §3,
// Claude session 2026-04-25): proportional to measured 100K size distribution.
// Uniform-random over rows in eval/results/synthea-100000/sizes.csv IS
// proportional sampling because every row is one observation.

package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
)

// SampledBundle is one row from sizes.csv plus the recorded byte size.
// We do not need the actual FHIR JSON — the bench measures storage-fabric
// latency, not parser speed; the bytes are synthesised in the driver.
type SampledBundle struct {
	Filename  string
	SizeBytes int64
}

// SampleBundles reads sizesCSV and returns n bundles via uniform-random
// selection. Deterministic for a given seed.
//
// Sampling without replacement: picking with replacement at n=100 over
// 114 693 rows gives ~4% chance of any duplicate, low but non-zero;
// without replacement is cleaner and what the §V narrative will assume.
func SampleBundles(sizesCSV string, n int, seed int64) ([]SampledBundle, error) {
	if n <= 0 {
		return nil, fmt.Errorf("SampleBundles: n must be > 0, got %d", n)
	}
	rows, err := loadSizesCSV(sizesCSV)
	if err != nil {
		return nil, err
	}
	if len(rows) < n {
		return nil, fmt.Errorf("SampleBundles: requested %d but only %d rows in %s", n, len(rows), sizesCSV)
	}

	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(rows), func(i, j int) { rows[i], rows[j] = rows[j], rows[i] })
	return rows[:n], nil
}

// loadSizesCSV parses the (filename,size_bytes) layout written by
// eval/scripts/plot_size_distribution.py. Errors out with a clear
// pointer if the file is missing.
func loadSizesCSV(path string) ([]SampledBundle, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf(
				"SampleBundles: %s missing — run `make plot-synthea-100k` first",
				path,
			)
		}
		return nil, fmt.Errorf("SampleBundles: open %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("SampleBundles: read header: %w", err)
	}
	if len(header) < 2 || header[0] != "filename" || header[1] != "size_bytes" {
		return nil, fmt.Errorf("SampleBundles: unexpected header %v in %s", header, path)
	}

	var out []SampledBundle
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("SampleBundles: read row: %w", err)
		}
		if len(rec) < 2 {
			continue
		}
		size, err := strconv.ParseInt(rec[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("SampleBundles: parse size %q: %w", rec[1], err)
		}
		out = append(out, SampledBundle{Filename: rec[0], SizeBytes: size})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("SampleBundles: %s contains no rows", path)
	}
	return out, nil
}
