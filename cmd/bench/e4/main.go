// Package main is the E4 time-to-availability bench harness.
//
// CLAUDE.md §8 row "E4". Measures, for each (bundle, tier) tuple, the
// time from Put-return until a non-source Get can reach the bundle.
// Fig 5 in the paper is a CDF per tier: x = log(time-since-PUT in ms),
// y = fraction of bundles reachable. The expected shape is a fast hot
// curve, a moderate warm curve, and a long cold tail (Arweave settlement).
//
// Three tier-mode hooks: each tier can run in `sim` mode (calibrated
// lognormal stand-in, see sim_tier.go) or `live` mode (real Pinata /
// Filebase / Irys clients). At D15 only sim is wired; the live path is
// drafted via the tiers.Client interface so the swap is mechanical when
// .env funding lands. The mode is captured per-tier in env.json so a
// reviewer can see exactly which tier was synthesised vs measured.
//
// Realism notes:
//   - Polling cadence is logarithmic 100ms → 5min, identical between sim
//     and live runs. Sim mode could short-circuit to the analytically-
//     correct propagation time, but emitting the same shape on both code
//     paths makes parity checks trivial when live runs land.
//   - Per-bundle payload is random bytes synthesised once and reused
//     across the three tiers for that bundle. This keeps per-tier
//     latencies comparable.
//   - 100 bundles × 3 tiers × ≤300s polling budget = ~75min worst-case
//     wall in sim mode (each tier polls in parallel within a bundle,
//     bundles run sequentially). Live mode would dominate at cold-tier
//     settlement which is bounded by the 300s last checkpoint.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	mrand "math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

type runRecord struct {
	SchemaVersion int       `json:"schema_version"`
	RunID         string    `json:"run_id"`
	StartedAt     string    `json:"started_at"`
	CompletedAt   string    `json:"completed_at"`
	Config        runConfig `json:"config"`
	Samples       []sample  `json:"samples"`
}

type runConfig struct {
	NBundles         int      `json:"n_bundles"`
	Seed             int64    `json:"seed"`
	SizesCSV         string   `json:"sizes_csv"`
	HotMode          string   `json:"hot_mode"`
	WarmMode         string   `json:"warm_mode"`
	ColdMode         string   `json:"cold_mode"`
	PollCheckpointMs []int64  `json:"poll_checkpoint_ms"`
	HotPropP50Ms     int      `json:"hot_prop_p50_ms"`
	HotPropP99Ms     int      `json:"hot_prop_p99_ms"`
	WarmPropP50Ms    int      `json:"warm_prop_p50_ms"`
	WarmPropP99Ms    int      `json:"warm_prop_p99_ms"`
	ColdPropP50Ms    int      `json:"cold_prop_p50_ms"`
	ColdPropP99Ms    int      `json:"cold_prop_p99_ms"`
	Notes            string   `json:"notes"`
}

type sample struct {
	BundleIdx int                   `json:"bundle_idx"`
	Filename  string                `json:"filename"`
	SizeBytes int64                 `json:"size_bytes"`
	Tiers     map[string]tierResult `json:"tiers"`
}

type envJSON struct {
	CommitHash  string `json:"commit_hash"`
	GoVersion   string `json:"go_version"`
	OS          string `json:"os"`
	Kernel      string `json:"kernel"`
	CPU         string `json:"cpu"`
	NetworkPath string `json:"network_path"`
	WallStart   string `json:"wall_start"`
	WallEnd     string `json:"wall_end"`
	BenchID     string `json:"bench_id"`
	Notes       string `json:"notes"`
}

func main() {
	var (
		nFlag       = flag.Int("n", 100, "number of bundles to sample")
		seedFlag    = flag.Int64("seed", 42, "RNG seed")
		sizesFlag   = flag.String("sizes-csv", "eval/results/synthea-100000/sizes.csv", "Synthea sizes.csv")
		outDirFlag  = flag.String("out-dir", "eval/results/E4", "output directory")
		hotMode     = flag.String("hot-mode", "sim", "hot tier mode: sim|live")
		warmMode    = flag.String("warm-mode", "sim", "warm tier mode: sim|live")
		coldMode    = flag.String("cold-mode", "sim", "cold tier mode: sim|live")
		notesFlag   = flag.String("notes", "", "free-text notes for env.json (e.g., calibration provenance)")
	)
	flag.Parse()

	if err := run(*nFlag, *seedFlag, *sizesFlag, *outDirFlag, *hotMode, *warmMode, *coldMode, *notesFlag); err != nil {
		log.Fatalf("bench-E4: %v", err)
	}
}

func run(n int, seed int64, sizesCSV, outDir, hotModeStr, warmModeStr, coldModeStr, notes string) error {
	wallStart := time.Now().UTC()
	commit := shortCommit()
	runID := wallStart.Format("20060102-150405") + "-" + commit

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	all, err := loadSizes(sizesCSV)
	if err != nil {
		return fmt.Errorf("load sizes: %w", err)
	}
	rng := mrand.New(mrand.NewSource(seed))
	picked, err := sampleN(all, n, rng)
	if err != nil {
		return fmt.Errorf("sample: %w", err)
	}
	log.Printf("bench-E4: sampled %d bundles from %s (seed=%d)", len(picked), sizesCSV, seed)

	hotClient, err := makeTier("hot", tiers.TierHot, hotModeStr, hotPut, hotProp, seed+1)
	if err != nil {
		return fmt.Errorf("hot tier: %w", err)
	}
	warmClient, err := makeTier("warm", tiers.TierWarm, warmModeStr, warmPut, warmProp, seed+2)
	if err != nil {
		return fmt.Errorf("warm tier: %w", err)
	}
	coldClient, err := makeTier("cold", tiers.TierCold, coldModeStr, coldPut, coldProp, seed+3)
	if err != nil {
		return fmt.Errorf("cold tier: %w", err)
	}

	tierClients := map[string]tiers.Client{
		"hot":  hotClient,
		"warm": warmClient,
		"cold": coldClient,
	}
	tierLive := map[string]bool{
		"hot":  hotModeStr == "live",
		"warm": warmModeStr == "live",
		"cold": coldModeStr == "live",
	}

	checkpoints := make([]int64, len(pollCheckpoints))
	for i, cp := range pollCheckpoints {
		checkpoints[i] = cp.Milliseconds()
	}

	samples := make([]sample, 0, len(picked))
	for i, b := range picked {
		payload, err := synthesisePayload(b.SizeBytes)
		if err != nil {
			return fmt.Errorf("bundle %d: %w", i, err)
		}

		// Per-tier polling runs in parallel for one bundle. Bundles
		// themselves are sequential — running 100 in parallel would
		// hammer live tiers and is unnecessary for sim (the wall-clock
		// gain is worth more for the live mode where total polling
		// budget per bundle is up to 5 min; we'll revisit when live
		// lands).
		var (
			wg      sync.WaitGroup
			results = map[string]tierResult{}
			mu      sync.Mutex
			errs    []error
		)
		for tierName, client := range tierClients {
			wg.Add(1)
			tierName := tierName
			client := client
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()
				res, err := runOneBundleTier(ctx, tierName, client, payload, tierLive[tierName])
				mu.Lock()
				if err != nil {
					errs = append(errs, fmt.Errorf("bundle %d tier %s: %w", i, tierName, err))
				} else {
					results[tierName] = res
				}
				mu.Unlock()
			}()
		}
		wg.Wait()
		if len(errs) > 0 {
			return fmt.Errorf("bundle %d: %d tier errors: %v", i, len(errs), errs[0])
		}

		samples = append(samples, sample{
			BundleIdx: i,
			Filename:  b.Filename,
			SizeBytes: b.SizeBytes,
			Tiers:     results,
		})

		if (i+1)%10 == 0 || i == len(picked)-1 {
			log.Printf("bench-E4: %d/%d bundles done", i+1, len(picked))
		}
	}

	wallEnd := time.Now().UTC()

	rec := runRecord{
		SchemaVersion: 1,
		RunID:         runID,
		StartedAt:     wallStart.Format(time.RFC3339Nano),
		CompletedAt:   wallEnd.Format(time.RFC3339Nano),
		Config: runConfig{
			NBundles:         n,
			Seed:             seed,
			SizesCSV:         sizesCSV,
			HotMode:          hotModeStr,
			WarmMode:         warmModeStr,
			ColdMode:         coldModeStr,
			PollCheckpointMs: checkpoints,
			HotPropP50Ms:     hotProp.p50ms,
			HotPropP99Ms:     hotProp.p99ms,
			WarmPropP50Ms:    warmProp.p50ms,
			WarmPropP99Ms:    warmProp.p99ms,
			ColdPropP50Ms:    coldProp.p50ms,
			ColdPropP99Ms:    coldProp.p99ms,
			Notes:            notes,
		},
		Samples: samples,
	}

	runJSONPath := filepath.Join(outDir, "run-"+runID+".json")
	if err := writeJSON(runJSONPath, rec); err != nil {
		return err
	}
	log.Printf("bench-E4: wrote %s", runJSONPath)

	envPath := filepath.Join(outDir, "env.json")
	netPath := "N/A — sim mode"
	if hotModeStr == "live" || warmModeStr == "live" || coldModeStr == "live" {
		netPath = fmt.Sprintf("partial-live: hot=%s warm=%s cold=%s", hotModeStr, warmModeStr, coldModeStr)
	}
	if err := writeJSON(envPath, envJSON{
		CommitHash:  commit,
		GoVersion:   runtime.Version(),
		OS:          runtime.GOOS,
		Kernel:      unameOrEmpty(),
		CPU:         cpuOrEmpty(),
		NetworkPath: netPath,
		WallStart:   wallStart.Format(time.RFC3339Nano),
		WallEnd:     wallEnd.Format(time.RFC3339Nano),
		BenchID:     "E4",
		Notes:       notes,
	}); err != nil {
		return err
	}
	log.Printf("bench-E4: wrote %s", envPath)

	printSummary(rec, wallEnd.Sub(wallStart))
	return nil
}

// makeTier returns a tiers.Client for the requested mode. Sim returns
// the calibrated stand-in; live is left unimplemented at D15 (returns
// an error if requested), with the integration plan documented in the
// comment.
func makeTier(name string, class uint8, mode string, putDist, propDist lnPair, seed int64) (tiers.Client, error) {
	switch mode {
	case "sim", "":
		return NewSimTTA(name, class, putDist, propDist, seed), nil
	case "live":
		// Live integration plan: instantiate the corresponding
		// internal/tiers/{pinata,filebase,arweave} client from .env,
		// then wrap with a thin TTA-flavored decorator that captures
		// (put_latency, polling-Get latencies). The decorator should
		// preserve the tiers.Client interface so the per-bundle loop
		// in runOneBundleTier stays tier-agnostic.
		return nil, fmt.Errorf("e4: live mode not wired at D15 — see internal/tiers/%s/ for the client; bind it from .env once funding lands", name)
	default:
		return nil, fmt.Errorf("e4: unknown mode %q (want sim|live)", mode)
	}
}

func printSummary(rec runRecord, wall time.Duration) {
	fmt.Println()
	fmt.Printf("bench-E4: %d bundles × 3 tiers; total wall=%s\n", len(rec.Samples), wall.Round(time.Millisecond))
	for _, tier := range []string{"hot", "warm", "cold"} {
		var firstOKs []int64
		var puts []int64
		timeouts := 0
		for _, s := range rec.Samples {
			tr, ok := s.Tiers[tier]
			if !ok {
				continue
			}
			firstOKs = append(firstOKs, tr.FirstOKAtMs)
			puts = append(puts, tr.PutLatencyMs)
			if tr.TimedOut {
				timeouts++
			}
		}
		if len(firstOKs) == 0 {
			continue
		}
		fmt.Printf("  %-6s n=%-3d put_p50=%4dms first_ok_p50=%6dms first_ok_p99=%7dms timeouts=%d\n",
			tier, len(firstOKs),
			percentile(puts, 0.50),
			percentile(firstOKs, 0.50),
			percentile(firstOKs, 0.99),
			timeouts)
	}
}

// percentile returns the p-percentile of a slice. Modifies a copy.
func percentile(xs []int64, p float64) int64 {
	if len(xs) == 0 {
		return 0
	}
	cp := make([]int64, len(xs))
	copy(cp, xs)
	// Insertion sort — n ≤ 100 so anything fancier is overkill.
	for i := 1; i < len(cp); i++ {
		v := cp[i]
		j := i - 1
		for j >= 0 && cp[j] > v {
			cp[j+1] = cp[j]
			j--
		}
		cp[j+1] = v
	}
	idx := int(float64(len(cp)-1) * p)
	return cp[idx]
}

// --- helpers (mirror cmd/bench/eprov / cmd/bench/e5) -------------------

func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return nil
}

func shortCommit() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "nogit"
	}
	return strings.TrimSpace(string(out))
}

func unameOrEmpty() string {
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func cpuOrEmpty() string {
	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}
