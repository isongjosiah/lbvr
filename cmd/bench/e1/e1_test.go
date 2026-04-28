package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRun_Smoke_1K — drives the bench at the smallest scale through a
// tiny invocation so we catch wiring bugs (Ingester signature drift,
// JSON schema typos, env.json fields) without waiting for the full
// matrix. Uses a 50-row synthesised CSV (small bundle sizes) so the
// test finishes in a few seconds even with concurrency=4.
func TestRun_Smoke_1K(t *testing.T) {
	outDir := t.TempDir()
	bundleDir := t.TempDir()
	csv := smallSizesCSV(t, 1500)

	if err := run(42, 4, outDir, csv, bundleDir, []int{50}); err != nil {
		t.Fatalf("run: %v", err)
	}

	// Locate run-*.json + env.json.
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("read outDir: %v", err)
	}
	var foundRun, foundEnv bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "run-") && strings.HasSuffix(e.Name(), ".json") {
			foundRun = true
			b, err := os.ReadFile(filepath.Join(outDir, e.Name()))
			if err != nil {
				t.Fatalf("read run JSON: %v", err)
			}
			var rec runRecord
			if err := json.Unmarshal(b, &rec); err != nil {
				t.Fatalf("unmarshal run JSON: %v", err)
			}
			if rec.SchemaVersion != 1 {
				t.Errorf("schema_version = %d, want 1", rec.SchemaVersion)
			}
			// At least one live scale row + the projected rows for
			// 1000/10000/100000 (whichever didn't run). The smoke
			// test asks for {50} explicitly; the projection logic
			// only fires for the canonical scaleSizes list, so
			// scaleStat at corpus_size=50 must exist live.
			haveLive := false
			for _, s := range rec.Scales {
				if s.CorpusSize == 50 && !s.Projected {
					haveLive = true
					if s.NBundles != 50 {
						t.Errorf("scale 50: NBundles = %d, want 50", s.NBundles)
					}
					if s.WallSeconds <= 0 {
						t.Errorf("scale 50: non-positive WallSeconds %v", s.WallSeconds)
					}
					if s.ThroughputMBPS <= 0 {
						t.Errorf("scale 50: non-positive throughput %v", s.ThroughputMBPS)
					}
					if s.ThroughputBundlesPerMin <= 0 {
						t.Errorf("scale 50: non-positive bundles/min %v", s.ThroughputBundlesPerMin)
					}
					if s.StageP50ms.TotalMs <= 0 {
						t.Errorf("scale 50: non-positive P50 total %v", s.StageP50ms.TotalMs)
					}
				}
			}
			if !haveLive {
				t.Errorf("no live scaleStat with corpus_size=50; got %d entries", len(rec.Scales))
			}
		}
		if e.Name() == "env.json" {
			foundEnv = true
			b, err := os.ReadFile(filepath.Join(outDir, e.Name()))
			if err != nil {
				t.Fatalf("read env.json: %v", err)
			}
			var env envJSON
			if err := json.Unmarshal(b, &env); err != nil {
				t.Fatalf("unmarshal env.json: %v", err)
			}
			if env.BenchID != "E1" {
				t.Errorf("bench_id = %q, want E1", env.BenchID)
			}
			if len(env.TierDistributions) != 3 {
				t.Errorf("tier_distributions = %d entries, want 3", len(env.TierDistributions))
			}
		}
	}
	if !foundRun {
		t.Error("no run-*.json written")
	}
	if !foundEnv {
		t.Error("no env.json written")
	}
}

// TestStageTimings_Sum — sanity check that the per-stage P50s roughly
// add up to the total P50. The pipeline has merkle → seal → encode →
// upload → register; each stage runs serially within Ingest, so the
// stage P50s should sum to within 20% of the total P50 across the
// distribution. (Strict equality fails because the per-bundle stage
// breakdown of one outlier doesn't have to match the per-bundle total
// of another outlier; we're checking distribution-level sanity.)
func TestStageTimings_Sum(t *testing.T) {
	outDir := t.TempDir()
	bundleDir := t.TempDir()
	csv := smallSizesCSV(t, 500)

	if err := run(7, 4, outDir, csv, bundleDir, []int{120}); err != nil {
		t.Fatalf("run: %v", err)
	}
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("read outDir: %v", err)
	}
	var rec runRecord
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "run-") && strings.HasSuffix(e.Name(), ".json") {
			b, err := os.ReadFile(filepath.Join(outDir, e.Name()))
			if err != nil {
				t.Fatalf("read run JSON: %v", err)
			}
			if err := json.Unmarshal(b, &rec); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			break
		}
	}
	if len(rec.Scales) == 0 {
		t.Fatal("no scales in run record")
	}

	for _, s := range rec.Scales {
		if s.Projected {
			continue
		}
		stages := s.StageP50ms.MerkleMs + s.StageP50ms.SealMs +
			s.StageP50ms.EncodeMs + s.StageP50ms.UploadMs +
			s.StageP50ms.RegisterMs
		total := s.StageP50ms.TotalMs
		if total <= 0 {
			t.Errorf("scale %d: non-positive P50 total %v", s.CorpusSize, total)
			continue
		}
		// |stages - total| / total <= 0.20
		ratio := (stages - total) / total
		if ratio < -0.20 || ratio > 0.20 {
			t.Errorf("scale %d: stage sum %.2f vs total %.2f (delta %.1f%%)",
				s.CorpusSize, stages, total, 100*ratio)
		}
	}
}

// TestE1Full — gated by -short so the unit-test loop stays fast.
// Documented invocation: `go test -run TestE1Full -timeout 90m`.
// Reads BENCH_E1_SCALES (default "1000,10000") + BENCH_E1_SEED (42).
func TestE1Full(t *testing.T) {
	if testing.Short() {
		t.Skip("E1 full run skipped under -short")
	}
	seed := int64(42)
	if v := os.Getenv("BENCH_E1_SEED"); v != "" {
		var n int64
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			seed = n
		}
	}
	concurrency := 8
	if v := os.Getenv("BENCH_E1_CONCURRENCY"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			concurrency = n
		}
	}
	scales := runScales
	if v := os.Getenv("BENCH_E1_SCALES"); v != "" {
		s, err := parseScales(v)
		if err != nil {
			t.Fatalf("parseScales: %v", err)
		}
		scales = s
	}
	outDir := envStrTest("BENCH_E1_OUT_DIR", "../../../eval/results/E1")
	sizesCSV := envStrTest("BENCH_E1_SIZES_CSV", "../../../eval/results/synthea-100000/sizes.csv")
	bundleDir := os.Getenv("BENCH_E1_BUNDLE_DIR") // empty → run() picks $TMPDIR

	if _, err := os.Stat(sizesCSV); err != nil {
		t.Skipf("E1 full run requires %s", sizesCSV)
	}

	if err := run(seed, concurrency, outDir, sizesCSV, bundleDir, scales); err != nil {
		t.Fatalf("E1 full run: %v", err)
	}
}

// envStrTest returns os.Getenv(key) or def when unset. Stripped-down
// version of the shared helper in cmd/bench/e9/run_full_test.go.
func envStrTest(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

// smallSizesCSV writes a synthetic sizes.csv with `rows` entries in the
// 200 KB - 600 KB range (above the §4.5 floor, fast to seal). Returns
// the absolute path. Used by the smoke + sum-check tests so neither
// requires the real 100K corpus to be present.
func smallSizesCSV(t *testing.T, rows int) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "sizes.csv")
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("create CSV: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString("filename,size_bytes\n"); err != nil {
		t.Fatalf("write header: %v", err)
	}
	rng := rand.New(rand.NewSource(int64(rows)))
	for i := 0; i < rows; i++ {
		size := 200000 + rng.Intn(400000) // 200 KB - 600 KB
		if _, err := fmt.Fprintf(f, "row%05d.json,%d\n", i, size); err != nil {
			t.Fatalf("write row: %v", err)
		}
	}
	return p
}
