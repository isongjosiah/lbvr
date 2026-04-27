// E2 bench tests. Three flavours mirroring cmd/bench/e9:
//
//   TestRun_Smoke           — small invocation; verifies wiring + JSON shape.
//   TestBaselines_AllPositive — every baseline emits strictly-positive samples.
//   TestE2Full              — full bench, gated by -short. Run via:
//                              go test -run TestE2Full -timeout 30m
//                            (matches CLAUDE.md §10 D14 invocation pattern).

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestRun_Smoke runs a tiny end-to-end invocation so we catch wiring
// bugs (Recover signature drift, JSON schema typos, env.json fields,
// baseline interface mismatches) without needing the full 100×10 run.
func TestRun_Smoke(t *testing.T) {
	outDir := t.TempDir()
	sizesCSV := "../../../eval/results/synthea-100000/sizes.csv"
	if _, err := os.Stat(sizesCSV); err != nil {
		// Fallback: synthesise a small distribution covering the
		// regions we care about (small bundles only — keep the test fast).
		fallback := filepath.Join(outDir, "sizes.csv")
		f, err := os.Create(fallback)
		if err != nil {
			t.Fatalf("create fallback CSV: %v", err)
		}
		if _, err := f.WriteString("filename,size_bytes\n"); err != nil {
			t.Fatalf("write header: %v", err)
		}
		for i := 0; i < 50; i++ {
			line := fmt.Sprintf("test_%d.json,%d\n", i, 200000+i*8000)
			if _, err := f.WriteString(line); err != nil {
				t.Fatalf("write row: %v", err)
			}
		}
		f.Close()
		sizesCSV = fallback
	}

	if err := run(5, 2, 1, outDir, 2000, sizesCSV); err != nil {
		t.Fatalf("run: %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("read outDir: %v", err)
	}
	var foundRun, foundEnv bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "run-") && strings.HasSuffix(e.Name(), ".json") {
			foundRun = true
			path := filepath.Join(outDir, e.Name())
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			var rec runRecord
			if err := json.Unmarshal(b, &rec); err != nil {
				t.Fatalf("unmarshal %s: %v", path, err)
			}
			if rec.SchemaVersion != 1 {
				t.Errorf("schema_version = %d, want 1", rec.SchemaVersion)
			}
			if len(rec.Samples) != 5 {
				t.Errorf("samples = %d, want 5", len(rec.Samples))
			}
			// Per-bundle: every mode in allModes must be present with
			// 2 runs each (reps=2).
			for _, s := range rec.Samples {
				for _, m := range allModes {
					runs, ok := s.ModeRuns[m]
					if !ok {
						t.Errorf("bundle %d missing mode %q", s.BundleIdx, m)
						continue
					}
					if len(runs) != 2 {
						t.Errorf("bundle %d mode %s: runs = %d, want 2", s.BundleIdx, m, len(runs))
					}
					for _, r := range runs {
						if r.LatencyNS <= 0 {
							t.Errorf("bundle %d mode %s: non-positive latency %d", s.BundleIdx, m, r.LatencyNS)
						}
						if m == "lbvr" && r.Mode == "failure" {
							t.Errorf("bundle %d lbvr reported failure", s.BundleIdx)
						}
					}
				}
			}
			// Sanity: 5 modes total = 1 lbvr + 4 baselines.
			if len(allModes) != 5 {
				t.Errorf("allModes = %d, want 5", len(allModes))
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
			if env.BenchID != "E2" {
				t.Errorf("bench_id = %q, want E2", env.BenchID)
			}
			if len(env.TierDistributions) != 3 {
				t.Errorf("tier_distributions = %d entries, want 3", len(env.TierDistributions))
			}
			if len(env.Baselines) != 4 {
				t.Errorf("baselines = %d entries, want 4", len(env.Baselines))
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

// TestBaselines_AllPositive — every baseline must emit strictly-positive
// latency samples; lognormal sampling is positive by construction, but
// this guards against a future re-calibration that collapses to zero
// (e.g. a sigma so large that the floor clamp dominates).
func TestBaselines_AllPositive(t *testing.T) {
	for _, b := range newBaselines(7) {
		ctx := context.Background()
		for i := 0; i < 100; i++ {
			lat, err := b.Fetch(ctx)
			if err != nil {
				t.Fatalf("baseline %s rep %d: %v", b.Name(), i, err)
			}
			if lat <= 0 {
				t.Errorf("baseline %s rep %d: non-positive latency %v", b.Name(), i, lat)
			}
		}
	}
}

// TestE2Full runs the full E2 bench against the real eval/results/
// directory. Skipped under -short so the unit-test loop stays fast;
// invoked explicitly via:
//
//	go test -run TestE2Full -timeout 30m ./cmd/bench/e2/...
//
// Defaults match `make bench-E2` once that target lands; can be
// overridden via BENCH_E2_* env vars.
func TestE2Full(t *testing.T) {
	if testing.Short() {
		t.Skip("E2 full run skipped under -short")
	}
	n := envInt("BENCH_E2_N", 100)
	reps := envInt("BENCH_E2_REPS", 10)
	seed := envInt("BENCH_E2_SEED", 42)
	sloMS := envInt("BENCH_E2_SLO_MS", 2000)
	outDir := envStr("BENCH_E2_OUT_DIR", "../../../eval/results/E2")
	sizesCSV := envStr("BENCH_E2_SIZES_CSV", "../../../eval/results/synthea-100000/sizes.csv")

	if err := run(n, reps, int64(seed), outDir, sloMS, sizesCSV); err != nil {
		t.Fatalf("E2 full run: %v", err)
	}
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envStr(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}
