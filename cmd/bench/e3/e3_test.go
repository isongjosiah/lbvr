// Unit tests for the E3 bench. Mirrors the cmd/bench/e9 test layout.
//
//	TestRun_Smoke              — small driver invocation; assert JSON written
//	                             + parses + cells == expected count.
//	TestSimTier_LossRateBoundaries — sim Get loss-coin sanity at 0.0 and 1.0.
//	TestE3Full                 — gated by testing.Short(); invoked via
//	                             `go test -run TestE3Full -timeout 90m` for
//	                             the full bench run.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

// TestRun_Smoke drives the bench through a minimal invocation. Goal:
// catch wiring bugs (Recover signature drift, JSON schema typos, env.json
// fields) without waiting for the full 50×30×9 run. Synthesises a small
// sizes.csv if the real one is absent so the test runs in a fresh checkout.
//
// Note the Smoke variant runs the FULL 9-cell matrix because the matrix
// dimensions are part of the contract — but uses 5 bundles × 5 reps per
// cell to keep wall-clock low. CLAUDE.md §10 calls out: every cell must
// be exercised by tests.
func TestRun_Smoke(t *testing.T) {
	outDir := t.TempDir()
	sizesCSV := "../../../eval/results/synthea-100000/sizes.csv"
	if _, err := os.Stat(sizesCSV); err != nil {
		// Fallback: synthesise a small distribution (small bundles only
		// so the seal+encode setup stays fast).
		fallback := filepath.Join(outDir, "sizes.csv")
		f, err := os.Create(fallback)
		if err != nil {
			t.Fatalf("create fallback CSV: %v", err)
		}
		if _, err := f.WriteString("filename,size_bytes\n"); err != nil {
			t.Fatalf("write header: %v", err)
		}
		for i := 0; i < 50; i++ {
			if _, err := fmt.Fprintf(f, "test_%d.json,%d\n", i, 200000+i*8000); err != nil {
				t.Fatalf("write row: %v", err)
			}
		}
		f.Close()
		sizesCSV = fallback
	}

	if err := run(5, 5, 42, outDir, 2000, sizesCSV); err != nil {
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
			wantCells := len(rttCells) * len(lossCells)
			if len(rec.Cells) != wantCells {
				t.Errorf("cells = %d, want %d (3 RTT × 3 loss)", len(rec.Cells), wantCells)
			}
			for _, c := range rec.Cells {
				wantSamples := 5 * 5 // bundles × reps
				if len(c.Samples) != wantSamples {
					t.Errorf("cell rtt=%d loss=%.2f: samples=%d, want %d",
						c.RTTms, c.LossRate, len(c.Samples), wantSamples)
				}
				if c.Stats.NSamples != wantSamples {
					t.Errorf("cell rtt=%d loss=%.2f: stats.n_samples=%d, want %d",
						c.RTTms, c.LossRate, c.Stats.NSamples, wantSamples)
				}
				// 0%-loss cells must have zero failures (the gateway can
				// always recover from D0+D1 at minimum).
				if c.LossRate == 0.0 && c.Stats.FailurePct > 0 {
					t.Errorf("cell rtt=%d loss=0: failure_pct=%.1f, want 0",
						c.RTTms, c.Stats.FailurePct)
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
			if env.BenchID != "E3" {
				t.Errorf("bench_id = %q, want E3", env.BenchID)
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

// TestSimTier_LossRateBoundaries — Get must never error at lossRate=0
// (modulo lognormal-tier dropped/not-found, which we exclude by storing
// real bytes first), and must always error at lossRate=1.0.
//
// We use the warm tier (P50=120ms, P99=600ms) so 1000 samples complete
// in a bounded time even with the full lognormal wait. extraRTT is set
// to 0 so the test wall-clock is purely lognormal-bounded.
func TestSimTier_LossRateBoundaries(t *testing.T) {
	ctx := context.Background()
	const N = 1000

	// Loss = 0: Get must always succeed.
	{
		s := newSimTier("warm-loss0", tiers.TierWarm, 11)
		// To keep the test fast, drop lognormal to a near-zero
		// distribution by patching to hot params. Acceptable here
		// because we're exercising the loss-coin contract, not the
		// distribution itself.
		s.muDist = hotMu - 4 // ~ exp(14) ns ≈ 1.2 ms median
		s.sigma = 0.1
		s.SetWAN(0, 0.0)

		cid, err := s.Put(ctx, []byte("payload"))
		if err != nil {
			t.Fatalf("Put: %v", err)
		}
		errs := 0
		for i := 0; i < N; i++ {
			if _, err := s.Get(ctx, cid); err != nil {
				errs++
			}
		}
		if errs != 0 {
			t.Fatalf("lossRate=0: got %d errors over %d Gets, want 0", errs, N)
		}
	}

	// Loss = 1.0: every Get must return the WAN-drop sentinel.
	{
		s := newSimTier("warm-loss1", tiers.TierWarm, 12)
		s.muDist = hotMu - 4
		s.sigma = 0.1
		s.SetWAN(0, 1.0)

		cid, err := s.Put(ctx, []byte("payload"))
		if err != nil {
			t.Fatalf("Put: %v", err)
		}
		ok := 0
		drops := 0
		other := 0
		for i := 0; i < N; i++ {
			_, err := s.Get(ctx, cid)
			if err == nil {
				ok++
			} else if strings.Contains(err.Error(), "WAN drop") {
				drops++
			} else {
				other++
			}
		}
		if ok != 0 || other != 0 {
			t.Fatalf("lossRate=1.0: ok=%d drops=%d other=%d, want ok=0 drops=%d other=0",
				ok, drops, N, other)
		}
		if drops != N {
			t.Fatalf("lossRate=1.0: drops=%d, want %d", drops, N)
		}
	}
}
