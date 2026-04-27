// Tests for the E6 / E6b Byzantine-withstand bench harness. The two
// smoke tests catch wiring drift (Recover signature changes, JSON schema
// typos, env.json fields) without paying the cost of the full 100×10×5
// run. TestE6Full is gated on -short so the unit-test loop stays fast;
// the operator runs it explicitly with:
//
//	go test ./cmd/bench/e6 -run TestE6Full -timeout 60m
//
// Mirrors cmd/bench/e9/run_test.go and run_full_test.go.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// fallbackCSV writes a tiny sizes.csv into dir so the smoke tests can
// run in a fresh checkout (without the 100K Synthea corpus). 50 rows is
// more than the 5-bundle ask plus headroom.
func fallbackCSV(t *testing.T, dir string, rows int) string {
	t.Helper()
	p := filepath.Join(dir, "sizes.csv")
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("create fallback CSV: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString("filename,size_bytes\n"); err != nil {
		t.Fatalf("write header: %v", err)
	}
	for i := 0; i < rows; i++ {
		if _, err := fmt.Fprintf(f, "row%04d.json,%d\n", i, 200000+i*8000); err != nil {
			t.Fatalf("write row: %v", err)
		}
	}
	return p
}

// resolveSizesCSV prefers the real 100K corpus if present so the smoke
// test exercises the same code path as the full run, but falls back to
// a synthesised CSV when the corpus is missing.
func resolveSizesCSV(t *testing.T, outDir string) string {
	t.Helper()
	real := "../../../eval/results/synthea-100000/sizes.csv"
	if _, err := os.Stat(real); err == nil {
		return real
	}
	return fallbackCSV(t, outDir, 50)
}

// TestRun_Smoke_Uniform exercises the uniform-adversary path with a
// trimmed config (2 fractions × 5 bundles × 2 reps) so we catch JSON
// schema drift without waiting for the full run.
func TestRun_Smoke_Uniform(t *testing.T) {
	outDir := t.TempDir()
	sizesCSV := resolveSizesCSV(t, outDir)

	fractions := []float64{0.0, 0.33}
	if err := run("uniform", 5, 2, 42, outDir, 2000, sizesCSV, fractions); err != nil {
		t.Fatalf("run uniform: %v", err)
	}

	rec := readRunJSON(t, outDir, "uniform")
	if rec.SchemaVersion != 1 {
		t.Errorf("schema_version = %d, want 1", rec.SchemaVersion)
	}
	if rec.Config.Mode != "uniform" {
		t.Errorf("config.mode = %q, want uniform", rec.Config.Mode)
	}
	if len(rec.Fractions) != len(fractions) {
		t.Fatalf("fractions = %d, want %d", len(rec.Fractions), len(fractions))
	}
	// Sanity: fraction=0 must succeed 100%; fraction>0 has byzantine
	// replicas but the per-replica Bernoulli could (rarely) leave none
	// flipped at small N. Don't assert a specific failure rate here —
	// that's TestE6Full's job. Just check the schema is consistent.
	for _, fs := range rec.Fractions {
		if fs.NRetrievals != 5*2 {
			t.Errorf("fraction %.2f: n_retrievals = %d, want %d", fs.AdversaryFraction, fs.NRetrievals, 5*2)
		}
		if fs.AdversaryFraction == 0 {
			if fs.RetrievalSuccessRate != 1.0 {
				t.Errorf("fraction=0 retrieval_success_rate = %.3f, want 1.0", fs.RetrievalSuccessRate)
			}
		}
		if fs.NPoRChallenges != 0 || fs.NPoRSuccess != 0 {
			t.Errorf("uniform mode emitted PoR counts (%d/%d); should be zero",
				fs.NPoRSuccess, fs.NPoRChallenges)
		}
	}
}

// TestRun_Smoke_TierSelective exercises the tier-selective path with
// the same trimmed config. Asserts that PoR success rate stays at 1.0
// across all fractions (the §5 invariant: byzantine replicas play nice
// during PoR challenges).
func TestRun_Smoke_TierSelective(t *testing.T) {
	outDir := t.TempDir()
	sizesCSV := resolveSizesCSV(t, outDir)

	fractions := []float64{0.0, 0.33}
	if err := run("tier-selective", 5, 2, 42, outDir, 2000, sizesCSV, fractions); err != nil {
		t.Fatalf("run tier-selective: %v", err)
	}

	rec := readRunJSON(t, outDir, "tier-selective")
	if rec.Config.Mode != "tier_selective" {
		t.Errorf("config.mode = %q, want tier_selective", rec.Config.Mode)
	}
	for _, fs := range rec.Fractions {
		if fs.NPoRChallenges != 5*2 {
			t.Errorf("fraction %.2f: n_por_challenges = %d, want %d", fs.AdversaryFraction, fs.NPoRChallenges, 5*2)
		}
		// PoR-success-rate MUST be 1.0 — that is the whole point of
		// tier-selective adversary behaviour.
		if fs.PoRSuccessRate != 1.0 {
			t.Errorf("fraction %.2f: por_success_rate = %.3f, want 1.0",
				fs.AdversaryFraction, fs.PoRSuccessRate)
		}
		// Detection gap = PoR - retrieval; must be non-negative for
		// any fraction (PoR is an upper bound on what a passive
		// auditor would observe).
		if fs.DetectionGap < 0 {
			t.Errorf("fraction %.2f: detection_gap = %.3f, want >= 0",
				fs.AdversaryFraction, fs.DetectionGap)
		}
	}
}

// TestE6Full is the operator-driven full-corpus run. Skipped under
// -short. Reads BENCH_E6_* env vars so the operator can override the
// defaults without recompiling.
func TestE6Full(t *testing.T) {
	if testing.Short() {
		t.Skip("E6 full run skipped under -short")
	}
	mode := envStr("BENCH_E6_MODE", "uniform")
	n := envInt("BENCH_E6_N", 100)
	reps := envInt("BENCH_E6_REPS", 10)
	seed := envInt("BENCH_E6_SEED", 42)
	sloMS := envInt("BENCH_E6_SLO_MS", 2000)
	outDir := envStr("BENCH_E6_OUT_DIR", "../../../eval/results/E6")
	sizesCSV := envStr("BENCH_E6_SIZES_CSV", "../../../eval/results/synthea-100000/sizes.csv")

	if err := run(mode, n, reps, int64(seed), outDir, sloMS, sizesCSV, defaultFractions); err != nil {
		t.Fatalf("E6 full run (%s): %v", mode, err)
	}
}

// readRunJSON locates the run-*.json the bench wrote into outDir for the
// given mode label and decodes it. We tolerate the multi-mode case
// (operator may have run both within the same outDir) by filtering on
// the mode token in the filename.
func readRunJSON(t *testing.T, outDir, modeLabel string) runRecord {
	t.Helper()
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("read outDir: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "run-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		if !strings.Contains(name, "-"+modeLabel+"-") {
			continue
		}
		path := filepath.Join(outDir, name)
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var rec runRecord
		if err := json.Unmarshal(b, &rec); err != nil {
			t.Fatalf("unmarshal %s: %v", path, err)
		}
		return rec
	}
	t.Fatalf("no run-*-%s-*.json under %s", modeLabel, outDir)
	return runRecord{}
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
