// Bundle sampling for E3. Copied from cmd/bench/e9/sampler.go so the two
// benches share the same proportional-to-measured sampling policy. See
// the §V evaluation discussion: every cell of the matrix gets the same
// bundle set so RTT × loss is the only varying factor (the seed is
// forwarded from the driver).

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
type SampledBundle struct {
	Filename  string
	SizeBytes int64
}

// SampleBundles reads sizesCSV and returns n bundles via uniform-random
// selection. Deterministic for a given seed.
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
// eval/scripts/plot_size_distribution.py.
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
