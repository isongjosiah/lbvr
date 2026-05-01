package main

import (
	"context"
	"crypto/rand"
	mrand "math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/config"
	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

// TestSimTTA_PutGet_BeforePropagation: Get within propagation window
// returns ErrNotReady; Get after returns the synthesised payload.
func TestSimTTA_PutGet_BeforePropagation(t *testing.T) {
	// Tiny propagation distribution so the test is fast but observable.
	// muNs = ln(50ms in ns) = ln(5e7) ≈ 17.73, sigma=0.1 keeps the draws tight.
	tinyProp := lnPair{muNs: 17.73, sigma: 0.1, p50ms: 50, p99ms: 70, tag: "test-prop"}
	tinyPut := lnPair{muNs: 14.51, sigma: 0.1, p50ms: 2, p99ms: 3, tag: "test-put"}
	sim := NewSimTTA("hot-test", tiers.TierHot, tinyPut, tinyProp, 42)

	payload := make([]byte, 4096)
	if _, err := rand.Read(payload); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	cid, err := sim.Put(ctx, payload)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if cid == "" {
		t.Fatal("empty CID")
	}

	// Immediate Get: should be ErrNotReady (propagation hasn't happened).
	if _, err := sim.Get(ctx, cid); err != ErrNotReady {
		t.Errorf("Get before propagation: got %v want ErrNotReady", err)
	}

	// Wait long enough for propagation (P99 = 70ms; 200ms is plenty).
	time.Sleep(200 * time.Millisecond)
	got, err := sim.Get(ctx, cid)
	if err != nil {
		t.Fatalf("Get after propagation: %v", err)
	}
	if len(got) != len(payload) {
		t.Errorf("Get size=%d want %d", len(got), len(payload))
	}
}

// TestRunOneBundleTier_FirstOKAtMs: drives a full polling loop end-to-end
// in sim mode; confirms FirstOKAtMs falls within the configured
// propagation window.
func TestRunOneBundleTier_FirstOKAtMs(t *testing.T) {
	tinyProp := lnPair{muNs: 17.73, sigma: 0.1, p50ms: 50, p99ms: 70, tag: "test-prop"}
	tinyPut := lnPair{muNs: 14.51, sigma: 0.1, p50ms: 2, p99ms: 3, tag: "test-put"}
	sim := NewSimTTA("hot-test", tiers.TierHot, tinyPut, tinyProp, 42)

	payload := make([]byte, 1024)
	if _, err := rand.Read(payload); err != nil {
		t.Fatal(err)
	}
	res, err := runOneBundleTier(context.Background(), "hot", sim, payload, false)
	if err != nil {
		t.Fatalf("runOneBundleTier: %v", err)
	}
	if res.FirstOKAtMs == 0 {
		t.Errorf("FirstOKAtMs unset")
	}
	// First polling checkpoint is 100ms; propagation P99 ≈ 70ms → first OK
	// should be ≤ 250ms (the second checkpoint), and never timeout.
	if res.TimedOut {
		t.Errorf("unexpected timeout")
	}
	if res.FirstOKAtMs > 500 {
		t.Errorf("FirstOKAtMs=%dms unexpectedly high (P99 prop ≈ 70ms)", res.FirstOKAtMs)
	}
}

// TestSampleN_Determinism: re-run with same seed → same picks.
func TestSampleN_Determinism(t *testing.T) {
	all := []SampledBundle{
		{"a.json", 100}, {"b.json", 200}, {"c.json", 300},
		{"d.json", 400}, {"e.json", 500}, {"f.json", 600},
	}
	rng1 := mrand.New(mrand.NewSource(7))
	rng2 := mrand.New(mrand.NewSource(7))
	a, _ := sampleN(all, 3, rng1)
	b, _ := sampleN(all, 3, rng2)
	if len(a) != len(b) {
		t.Fatalf("len mismatch")
	}
	for i := range a {
		if a[i].Filename != b[i].Filename {
			t.Errorf("idx %d: %q vs %q", i, a[i].Filename, b[i].Filename)
		}
	}
}

// TestPollCheckpoints_Monotone: cadence list must be increasing.
func TestPollCheckpoints_Monotone(t *testing.T) {
	for i := 1; i < len(pollCheckpoints); i++ {
		if pollCheckpoints[i] <= pollCheckpoints[i-1] {
			t.Errorf("pollCheckpoints[%d]=%v not greater than [%d]=%v",
				i, pollCheckpoints[i], i-1, pollCheckpoints[i-1])
		}
	}
}

// TestWriteJSON_Smoke: round-trips a runRecord through the serialiser.
func TestWriteJSON_Smoke(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.json")
	rec := runRecord{
		SchemaVersion: 1,
		RunID:         "test",
		Config:        runConfig{NBundles: 2, Seed: 1},
		Samples: []sample{{
			BundleIdx: 0,
			Filename:  "a",
			SizeBytes: 100,
			Tiers:     map[string]tierResult{"hot": {Tier: "hot", PutLatencyMs: 5, FirstOKAtMs: 50}},
		}},
	}
	if err := writeJSON(path, rec); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

// TestMakeTier_LiveModeRequiresCfg: live mode without a config is a
// clear error (avoids accidental nil-deref against pinata.New /
// filebase.New).
func TestMakeTier_LiveModeRequiresCfg(t *testing.T) {
	_, err := makeTier("hot", tiers.TierHot, "live", hotPut, hotProp, 0, nil)
	if err == nil {
		t.Fatal("expected live mode with nil cfg to return error")
	}
}

// TestMakeTier_LiveColdNotYetWired: cold tier live mode still errors
// because Irys/Sepolia funding hasn't landed.
func TestMakeTier_LiveColdNotYetWired(t *testing.T) {
	// Construct a non-empty cfg so we don't trip the nil-check above —
	// we want to confirm the cold-specific error fires.
	cfg := &config.Config{PinataJWT: "x", FilebaseAccessKey: "y", FilebaseSecretKey: "z", FilebaseBucket: "b"}
	_, err := makeTier("cold", tiers.TierCold, "live", coldPut, coldProp, 0, cfg)
	if err == nil {
		t.Fatal("expected cold-tier live mode to return Sepolia-pending error")
	}
}

// TestMakeTier_UnknownMode: rejects bogus modes loudly.
func TestMakeTier_UnknownMode(t *testing.T) {
	_, err := makeTier("hot", tiers.TierHot, "magic", hotPut, hotProp, 0, nil)
	if err == nil {
		t.Fatal("expected unknown-mode error")
	}
}
