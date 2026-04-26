package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

// TestSimTier_PutGetRoundTrip — sanity check the in-memory store and
// CID derivation. Latency is real (lognormal) but bounded by class.
func TestSimTier_PutGetRoundTrip(t *testing.T) {
	s := newSimTier("hot", tiers.TierHot, 1)
	ctx := context.Background()
	cid, err := s.Put(ctx, []byte("hello"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := s.Get(ctx, cid)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("Get = %q, want %q", got, "hello")
	}
}

// TestSimTier_DropReturnsError — Drop() must surface as an error after
// the latency wait so the gateway sees the same shape as a real backend
// returning HTTP 5xx after a TCP/TLS round-trip.
func TestSimTier_DropReturnsError(t *testing.T) {
	s := newSimTier("hot", tiers.TierHot, 2)
	ctx := context.Background()
	cid, err := s.Put(ctx, []byte("data"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	s.Drop()
	_, err = s.Get(ctx, cid)
	if err == nil {
		t.Fatal("Get after Drop should error")
	}
}

// TestSimTier_ResetClearsDrop — Reset must restore normal Get behaviour
// for the next mode. Gateway harness depends on this.
func TestSimTier_ResetClearsDrop(t *testing.T) {
	s := newSimTier("warm", tiers.TierWarm, 3)
	ctx := context.Background()
	cid, err := s.Put(ctx, []byte("payload"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	s.Drop()
	if _, err := s.Get(ctx, cid); err == nil {
		t.Fatal("Get after Drop should error")
	}
	s.Reset()
	got, err := s.Get(ctx, cid)
	if err != nil {
		t.Fatalf("Get after Reset: %v", err)
	}
	if string(got) != "payload" {
		t.Fatalf("Get = %q", got)
	}
}

// TestSimTier_ContextCancelDuringWait — cancellation while the tier is
// in its latency wait must return ctx.Err() and bump WasCancelled().
// Mirrors the cmd/gateway/inmem_tier_test.go contract.
func TestSimTier_ContextCancelDuringWait(t *testing.T) {
	// Cold tier has the longest expected latency, so cancelling
	// shortly after Get starts will reliably interrupt the wait.
	s := newSimTier("cold", tiers.TierCold, 4)
	ctx := context.Background()
	cid, err := s.Put(ctx, []byte("p"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	cctx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(1)
	var gotErr error
	go func() {
		defer wg.Done()
		_, gotErr = s.Get(cctx, cid)
	}()

	// Cancel after a short delay; cold-tier P50 = 500ms so the
	// goroutine will still be in its wait.
	time.Sleep(10 * time.Millisecond)
	cancel()
	wg.Wait()

	if !errors.Is(gotErr, context.Canceled) {
		t.Fatalf("Get err = %v, want context.Canceled", gotErr)
	}
	if !s.WasCancelled() {
		t.Fatal("WasCancelled() = false, want true")
	}
}

// TestSimTier_LatencyIsPositive — lognormal samples are strictly
// positive by construction; this protects against future re-calibration
// that could collapse to zero.
func TestSimTier_LatencyIsPositive(t *testing.T) {
	s := newSimTier("hot", tiers.TierHot, 5)
	for i := 0; i < 1000; i++ {
		d := s.sampleLatency()
		if d <= 0 {
			t.Fatalf("sample %d = %v, want > 0", i, d)
		}
	}
}

// TestSampleBundles_Deterministic — same seed → same selection.
func TestSampleBundles_Deterministic(t *testing.T) {
	csv := writeTempCSV(t, 200)
	a, err := SampleBundles(csv, 10, 7)
	if err != nil {
		t.Fatalf("SampleBundles a: %v", err)
	}
	b, err := SampleBundles(csv, 10, 7)
	if err != nil {
		t.Fatalf("SampleBundles b: %v", err)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("seed-7 selection diverged at %d: %v vs %v", i, a[i], b[i])
		}
	}
}

// TestSampleBundles_DifferentSeeds — sanity that the seed actually
// changes the selection (shuffle is real, not a no-op).
func TestSampleBundles_DifferentSeeds(t *testing.T) {
	csv := writeTempCSV(t, 200)
	a, err := SampleBundles(csv, 10, 7)
	if err != nil {
		t.Fatalf("SampleBundles a: %v", err)
	}
	b, err := SampleBundles(csv, 10, 99)
	if err != nil {
		t.Fatalf("SampleBundles b: %v", err)
	}
	same := true
	for i := range a {
		if a[i] != b[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("seed 7 vs 99 produced identical selection — shuffle is broken")
	}
}

// TestSampleBundles_RequestTooManyErrors — guard against silent
// truncation when n exceeds row count.
func TestSampleBundles_RequestTooManyErrors(t *testing.T) {
	csv := writeTempCSV(t, 5)
	if _, err := SampleBundles(csv, 10, 1); err == nil {
		t.Fatal("expected error when n > rows")
	}
}

func writeTempCSV(t *testing.T, rows int) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "sizes.csv")
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString("filename,size_bytes\n"); err != nil {
		t.Fatalf("write header: %v", err)
	}
	for i := 0; i < rows; i++ {
		if _, err := fmt.Fprintf(f, "row%04d.json,%d\n", i, 100000+i*317); err != nil {
			t.Fatalf("write row: %v", err)
		}
	}
	return p
}
