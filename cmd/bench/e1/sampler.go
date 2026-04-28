// Bundle sampling for E1 — derived from cmd/bench/e9/sampler.go but
// adapted for the §8 ingest-throughput contract: the figure plots
// throughput "across the full corpus at this scale", so we want EVERY
// row up to the requested corpus_size, not a stratified sub-sample.
//
// Decision (CLAUDE.md §10 D9, docs/eval-protocol.md §3): proportional
// to the measured 100K size distribution. Reading the first N rows of a
// shuffled sizes.csv IS proportional sampling because every row is one
// observation. The shuffle is seeded so re-runs are deterministic.

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

// SampleBundles reads sizesCSV and returns the first `corpusSize` bundles
// from a seeded shuffle. Deterministic for a given seed.
//
// Differs from cmd/bench/e9/SampleBundles in two ways:
//
//  1. The argument is named corpusSize (not n) to match the §8 ingest-
//     throughput contract. The caller passes 1000 / 10000 / 100000.
//  2. corpusSize == len(rows) is allowed and returns every row. (The e9
//     variant requires len(rows) > n strictly because it samples a
//     small subset; here we want the whole corpus at the largest scale.)
func SampleBundles(sizesCSV string, corpusSize int, seed int64) ([]SampledBundle, error) {
	if corpusSize <= 0 {
		return nil, fmt.Errorf("SampleBundles: corpusSize must be > 0, got %d", corpusSize)
	}
	rows, err := loadSizesCSV(sizesCSV)
	if err != nil {
		return nil, err
	}
	if len(rows) < corpusSize {
		return nil, fmt.Errorf("SampleBundles: requested %d but only %d rows in %s", corpusSize, len(rows), sizesCSV)
	}

	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(rows), func(i, j int) { rows[i], rows[j] = rows[j], rows[i] })
	return rows[:corpusSize], nil
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
