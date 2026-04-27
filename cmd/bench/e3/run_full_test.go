// Driver test that invokes the full E3 bench against the real outputs
// directory. Skipped under -short so the unit-test loop stays fast;
// invoked explicitly with `go test -run TestE3Full -timeout 90m` from
// the D14 runbook (CLAUDE.md §10 D14). All flags read from BENCH_E3_*
// env vars so the operator can override defaults without recompiling.

package main

import (
	"os"
	"strconv"
	"testing"
)

func TestE3Full(t *testing.T) {
	if testing.Short() {
		t.Skip("E3 full run skipped under -short")
	}
	n := envInt("BENCH_E3_BUNDLES", 50)
	reps := envInt("BENCH_E3_REPS", 30)
	seed := envInt("BENCH_E3_SEED", 42)
	sloMS := envInt("BENCH_E3_SLO_MS", 2000)
	outDir := envStr("BENCH_E3_OUT_DIR", "../../../eval/results/E3")
	sizesCSV := envStr("BENCH_E3_SIZES_CSV", "../../../eval/results/synthea-100000/sizes.csv")

	if err := run(n, reps, int64(seed), outDir, sloMS, sizesCSV); err != nil {
		t.Fatalf("E3 full run: %v", err)
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
