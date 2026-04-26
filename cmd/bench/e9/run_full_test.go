// Driver test that invokes the full E9 bench against the real outputs
// directory. Skipped under -short so the unit-test loop stays fast;
// invoked explicitly with `go test -run E9Full -timeout 60m` from the
// D9 runbook (CLAUDE.md §10 D9). All flags read from BENCH_E9_* env
// vars so the operator can override defaults without recompiling.

package main

import (
	"os"
	"strconv"
	"testing"
)

func TestE9Full(t *testing.T) {
	if testing.Short() {
		t.Skip("E9 full run skipped under -short")
	}
	n := envInt("BENCH_E9_N", 100)
	reps := envInt("BENCH_E9_REPS", 10)
	seed := envInt("BENCH_E9_SEED", 42)
	sloMS := envInt("BENCH_E9_SLO_MS", 2000)
	outDir := envStr("BENCH_E9_OUT_DIR", "../../../eval/results/E9")
	sizesCSV := envStr("BENCH_E9_SIZES_CSV", "../../../eval/results/synthea-100000/sizes.csv")

	if err := run(n, reps, int64(seed), outDir, sloMS, sizesCSV); err != nil {
		t.Fatalf("E9 full run: %v", err)
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
