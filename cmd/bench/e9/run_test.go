package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRun_EndToEnd drives the bench through a tiny invocation so we
// catch wiring bugs (Recover signature drift, JSON schema typos, env.json
// fields) without needing to wait the full 100×10 run. Uses the real
// 100K sizes.csv if present; otherwise synthesises a 50-row CSV so the
// test still runs in a fresh checkout.
func TestRun_EndToEnd(t *testing.T) {
	outDir := t.TempDir()
	sizesCSV := "../../../eval/results/synthea-100000/sizes.csv"
	if _, err := os.Stat(sizesCSV); err != nil {
		// Fallback: synthesise a small distribution covering the
		// regions we care about (small bundles only — the test should
		// finish quickly).
		fallback := filepath.Join(outDir, "sizes.csv")
		f, err := os.Create(fallback)
		if err != nil {
			t.Fatalf("create fallback CSV: %v", err)
		}
		if _, err := f.WriteString("filename,size_bytes\n"); err != nil {
			t.Fatalf("write header: %v", err)
		}
		for i := 0; i < 50; i++ {
			// 200 KB - 600 KB: above the §4.5 floor, fast to seal.
			row := []byte("test")
			_ = row
			line := []byte{}
			line = append(line, []byte("test_")...)
			line = appendInt(line, i)
			line = append(line, []byte(".json,")...)
			line = appendInt(line, 200000+i*8000)
			line = append(line, '\n')
			if _, err := f.Write(line); err != nil {
				t.Fatalf("write row: %v", err)
			}
		}
		f.Close()
		sizesCSV = fallback
	}

	if err := run(3, 2, 42, outDir, 2000, sizesCSV); err != nil {
		t.Fatalf("run: %v", err)
	}

	// Confirm the run JSON + env.json landed in outDir.
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
			if len(rec.Samples) != 3 {
				t.Errorf("samples = %d, want 3", len(rec.Samples))
			}
			for _, s := range rec.Samples {
				for _, m := range modes {
					runs := s.ModeRuns[m]
					if len(runs) != 2 {
						t.Errorf("bundle %d mode %s: runs = %d, want 2", s.BundleIdx, m, len(runs))
					}
					for _, r := range runs {
						if r.LatencyNS <= 0 {
							t.Errorf("bundle %d mode %s: non-positive latency %d", s.BundleIdx, m, r.LatencyNS)
						}
						if m == "baseline" && r.Mode == "failure" {
							t.Errorf("bundle %d baseline reported failure", s.BundleIdx)
						}
					}
				}
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
			if env.BenchID != "E9" {
				t.Errorf("bench_id = %q, want E9", env.BenchID)
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

// appendInt avoids importing strconv for one call. Negative values not
// supported (sizes are always positive).
func appendInt(b []byte, n int) []byte {
	if n == 0 {
		return append(b, '0')
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return append(b, buf[i:]...)
}
