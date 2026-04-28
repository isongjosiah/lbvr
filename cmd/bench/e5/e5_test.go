package main

import (
	"math"
	mrand "math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/isongjosiah/lbvr-med/internal/merkle"
)

// TestRunProve_Smoke drives a single sweep point with a small number of
// reps to confirm the prove-time path runs end-to-end and produces
// monotonically positive timings.
func TestRunProve_Smoke(t *testing.T) {
	pt := SweepPoint{Depth: 3, NumChunks: 8}
	rng := mrand.New(mrand.NewSource(42))
	pr, err := runProve(rng, pt, 5)
	if err != nil {
		t.Fatalf("runProve: %v", err)
	}
	if pr.BuildBytes != pt.NumChunks*merkle.ChunkSize {
		t.Errorf("BuildBytes=%d want %d", pr.BuildBytes, pt.NumChunks*merkle.ChunkSize)
	}
	if pr.BuildTreeNS <= 0 {
		t.Errorf("BuildTreeNS non-positive: %d", pr.BuildTreeNS)
	}
	for i, ns := range pr.TotalNS {
		if ns <= 0 {
			t.Errorf("rep %d TotalNS non-positive: %d", i, ns)
		}
	}
	for i, ns := range pr.MerkleNS {
		if ns <= 0 {
			t.Errorf("rep %d MerkleNS non-positive: %d", i, ns)
		}
	}
	for i, ns := range pr.SignNS {
		if ns <= 0 {
			t.Errorf("rep %d SignNS non-positive: %d", i, ns)
		}
	}
}

// TestPercentiles_Monotone confirms p50 ≤ p95 ≤ p99 ≤ max and min ≤ p50.
func TestPercentiles_Monotone(t *testing.T) {
	samples := []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	p := percentiles(samples)
	if p.Min > p.P50 || p.P50 > p.P95 || p.P95 > p.P99 || p.P99 > p.Max {
		t.Errorf("non-monotone percentiles: %+v", p)
	}
}

// TestPercentiles_EmptyZero ensures the empty case doesn't panic.
func TestPercentiles_EmptyZero(t *testing.T) {
	p := percentiles(nil)
	if p.P50 != 0 || p.Max != 0 {
		t.Errorf("empty: %+v want zero", p)
	}
}

// TestGasLineRegex confirms the parser tolerates console2.log's irregular
// spacing. The fixture mirrors actual forge stdout we observed when
// validating the gas test against forge 1.5.1.
func TestGasLineRegex(t *testing.T) {
	cases := []struct {
		line    string
		fn      string
		depth   int
		gas     uint64
		matches bool
	}{
		{"  E5_GAS,post, 3 , 122215", "post", 3, 122215, true},
		{"  E5_GAS,respond, 5 , 337645", "respond", 5, 337645, true},
		{"E5_GAS,verdict,11,74944", "verdict", 11, 74944, true},
		{"unrelated log line", "", 0, 0, false},
	}
	for _, c := range cases {
		m := gasLineRegex.FindStringSubmatch(c.line)
		if c.matches != (m != nil) {
			t.Errorf("line=%q match=%v want %v", c.line, m != nil, c.matches)
			continue
		}
		if !c.matches {
			continue
		}
		if m[1] != c.fn {
			t.Errorf("line=%q fn=%q want %q", c.line, m[1], c.fn)
		}
		// Numeric fields converted by the bench loop, but verify the
		// regex captured the right substrings.
		if got := m[2]; got != itoa(c.depth) {
			t.Errorf("line=%q depth=%q want %d", c.line, got, c.depth)
		}
		if got := m[3]; got != utoa(c.gas) {
			t.Errorf("line=%q gas=%q want %d", c.line, got, c.gas)
		}
	}
}

// TestSweepPoint_BundleSize confirms the depth → bytes mapping is exactly
// numChunks * ChunkSize. A reviewer can recompute from the JSON without
// guessing at edge-case rounding.
func TestSweepPoint_BundleSize(t *testing.T) {
	for _, p := range sweepPoints {
		want := int(math.Pow(2, float64(p.Depth))) * merkle.ChunkSize
		if got := p.BundleSizeBytes(); got != want {
			t.Errorf("depth=%d: BundleSizeBytes=%d want %d", p.Depth, got, want)
		}
	}
}

// TestWriteJSON_RoundTrip verifies the JSON serialiser produces a file
// we can read back. Used as a smoke for the run-record output path.
func TestWriteJSON_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.json")
	if err := writeJSON(path, map[string]int{"a": 1}); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("empty file")
	}
}

func itoa(i int) string  { return formatInt(int64(i)) }
func utoa(u uint64) string { return formatInt(int64(u)) }
func formatInt(i int64) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
