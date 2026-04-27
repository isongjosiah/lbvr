// Package main is the E3 RTT × loss matrix bench harness.
//
// CLAUDE.md §8 row E3: P50/P95/P99 retrieval × RTT {10, 50, 200 ms} ×
// loss {0, 1, 5%}. docs/eval-protocol.md §2 row E3 → Fig. 4 — RTT × loss
// heatmap. Thesis: the storage fabric remains under SLO across plausible
// WAN conditions; degradation is graceful and predictable.
//
// Differs from E9 along two axes:
//
//   1. The failure model is uniform WAN impairment, not per-tier hard
//      drops. Every tier sees the same extraRTT and lossRate per cell;
//      this matches the assumption that a degraded WAN degrades everyone.
//
//   2. The figure is a 3×3 heatmap of P99 latency, not a CDF. Each cell
//      runs reps_per_cell retrievals through gateway.Recover and reports
//      P50/P95/P99 plus fast/slow/failure-path percentages.
//
// Bundle bytes are synthesised via crypto/rand (E9 convention): we
// measure storage-fabric latency, not FHIR parser speed. Each synthesised
// bundle is sealed via internal/crypto.SealChunk at 16-KiB granularity
// then erasure.Encode, so paddedLen and shard sizes match the real
// ingest path. We deliberately skip merkle.Build — gateway.Recover reads
// shard bytes only.

package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/crypto"
	"github.com/isongjosiah/lbvr-med/internal/erasure"
	"github.com/isongjosiah/lbvr-med/internal/gateway"
	"github.com/isongjosiah/lbvr-med/internal/merkle"
	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

// Matrix axes — locked to CLAUDE.md §8 row E3.
var (
	rttCells  = []time.Duration{10 * time.Millisecond, 50 * time.Millisecond, 200 * time.Millisecond}
	lossCells = []float64{0.0, 0.01, 0.05}
)

// Per-bundle setup cap (defensive vs allocator blowup). Synthea max ~98 MB.
const maxInputBytes = 1 << 30

// runRecord is the top-level JSON shape (schema_version=1).
type runRecord struct {
	SchemaVersion int        `json:"schema_version"`
	RunID         string     `json:"run_id"`
	StartedAt     string     `json:"started_at"`
	CompletedAt   string     `json:"completed_at"`
	Config        runConfig  `json:"config"`
	Cells         []cellData `json:"cells"`
}

type runConfig struct {
	NumBundles        int                 `json:"num_bundles"`
	RepsPerCell       int                 `json:"reps_per_cell"`
	Seed              int64               `json:"seed"`
	SLOBudgetMS       int                 `json:"slo_budget_ms"`
	RTTCellsMS        []int               `json:"rtt_cells_ms"`
	LossCells         []float64           `json:"loss_cells"`
	TierDistributions map[string]distSpec `json:"tier_distributions"`
	SamplingPolicy    string              `json:"sampling_policy"`
	WANMechanism      string              `json:"wan_mechanism"`
}

type cellData struct {
	RTTms    int          `json:"rtt_ms"`
	LossRate float64      `json:"loss_rate"`
	Samples  []sampleStat `json:"samples"`
	Stats    cellStats    `json:"stats"`
}

type cellStats struct {
	P50ms       float64 `json:"p50_ms"`
	P95ms       float64 `json:"p95_ms"`
	P99ms       float64 `json:"p99_ms"`
	FastPathPct float64 `json:"fast_path_pct"`
	SlowPathPct float64 `json:"slow_path_pct"`
	FailurePct  float64 `json:"failure_pct"`
	NSamples    int     `json:"n_samples"`
}

type sampleStat struct {
	BundleIdx int    `json:"bundle_idx"`
	LatencyNS int64  `json:"latency_ns"`
	Mode      string `json:"mode"` // "fast" / "slow" / "failure"
}

// envJSON mirrors CLAUDE.md §8 environment-fingerprint contract.
type envJSON struct {
	CommitHash        string              `json:"commit_hash"`
	GoVersion         string              `json:"go_version"`
	OS                string              `json:"os"`
	Kernel            string              `json:"kernel"`
	CPU               string              `json:"cpu"`
	NetworkPath       string              `json:"network_path"`
	WallStart         string              `json:"wall_start"`
	WallEnd           string              `json:"wall_end"`
	BenchID           string              `json:"bench_id"`
	SamplingPolicy    string              `json:"sampling_policy"`
	WANMechanism      string              `json:"wan_mechanism"`
	TierDistributions map[string]distSpec `json:"tier_distributions"`
}

func main() {
	var (
		nFlag        = flag.Int("bundles", 50, "number of bundles to sample (per-cell, same set across cells)")
		repsFlag     = flag.Int("reps", 30, "repetitions per cell per bundle")
		seedFlag     = flag.Int64("seed", 42, "RNG seed (sampler + simTier streams)")
		outDirFlag   = flag.String("out-dir", "eval/results/E3", "output directory")
		sloMSFlag    = flag.Int("slo-ms", 2000, "fast-path SLO budget in milliseconds")
		sizesCSVFlag = flag.String("sizes-csv", "eval/results/synthea-100000/sizes.csv", "path to validated size distribution")
	)
	flag.Parse()

	if err := run(*nFlag, *repsFlag, *seedFlag, *outDirFlag, *sloMSFlag, *sizesCSVFlag); err != nil {
		log.Fatalf("bench-E3: %v", err)
	}
}

func run(nBundles, reps int, seed int64, outDir string, sloMS int, sizesCSV string) error {
	wallStart := time.Now().UTC()
	commit := shortCommit()
	runID := wallStart.Format("20060102-150405") + "-" + commit

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	bundles, err := SampleBundles(sizesCSV, nBundles, seed)
	if err != nil {
		return err
	}

	sloBudget := time.Duration(sloMS) * time.Millisecond

	// Build the three sim-tiers ONCE; we reuse them across cells, just
	// re-configuring extraRTT/lossRate per cell. Independent rand streams
	// per tier keep WAN-loss draws reproducible regardless of cell order.
	hot := newSimTier("pinata-sim", tiers.TierHot, seed+1)
	warm := newSimTier("filebase-sim", tiers.TierWarm, seed+2)
	cold := newSimTier("arweave-sim", tiers.TierCold, seed+3)
	tierArr := [3]tiers.Client{hot, warm, cold}
	simArr := [3]*simTier{hot, warm, cold}

	// Encode every bundle once and store its shards on the sim tiers
	// (no WAN impairment during setup — Put is impairment-free by design).
	type encoded struct {
		bundleIdx int
		filename  string
		paddedLen int
		cids      [3]string
	}
	enc := make([]encoded, 0, nBundles)
	for i, b := range bundles {
		if b.SizeBytes <= 0 {
			return fmt.Errorf("bundle %d (%s) has non-positive size %d", i, b.Filename, b.SizeBytes)
		}
		if b.SizeBytes > maxInputBytes {
			return fmt.Errorf("bundle %d (%s) size %d exceeds maxInputBytes %d", i, b.Filename, b.SizeBytes, maxInputBytes)
		}
		plaintext := make([]byte, b.SizeBytes)
		if _, err := rand.Read(plaintext); err != nil {
			return fmt.Errorf("rand.Read: %w", err)
		}
		key, err := crypto.GenerateKey()
		if err != nil {
			return fmt.Errorf("crypto.GenerateKey: %w", err)
		}
		encrypted, err := sealAll(key, plaintext)
		if err != nil {
			return fmt.Errorf("sealAll bundle %d: %w", i, err)
		}
		for j := range key {
			key[j] = 0
		}
		shards, paddedLen, err := erasure.Encode(encrypted)
		if err != nil {
			return fmt.Errorf("erasure.Encode bundle %d: %w", i, err)
		}
		ctx := context.Background()
		var cids [3]string
		for k := 0; k < 3; k++ {
			cid, err := tierArr[k].Put(ctx, shards[k])
			if err != nil {
				return fmt.Errorf("Put bundle %d shard %d: %w", i, k, err)
			}
			cids[k] = cid
		}
		enc = append(enc, encoded{
			bundleIdx: i,
			filename:  b.Filename,
			paddedLen: paddedLen,
			cids:      cids,
		})
	}
	log.Printf("bench-E3: %d bundles encoded + stored across 3 sim tiers", len(enc))

	// Walk the 9-cell matrix.
	cells := make([]cellData, 0, len(rttCells)*len(lossCells))
	totalCells := len(rttCells) * len(lossCells)
	cellNum := 0
	for _, rtt := range rttCells {
		for _, loss := range lossCells {
			cellNum++
			// Apply impairment uniformly across all three tiers — WAN
			// degrades everyone (per the bench thesis statement).
			for _, s := range simArr {
				s.SetWAN(rtt, loss)
			}

			samples := make([]sampleStat, 0, len(enc)*reps)
			for _, e := range enc {
				for r := 0; r < reps; r++ {
					ctx := context.Background()
					start := time.Now()
					_, recStats, _ := gateway.Recover(ctx, tierArr, e.cids, e.paddedLen, sloBudget)
					latency := time.Since(start)
					samples = append(samples, sampleStat{
						BundleIdx: e.bundleIdx,
						LatencyNS: latency.Nanoseconds(),
						Mode:      recStats.Mode.String(),
					})
				}
			}

			stats := summariseCell(samples)
			cells = append(cells, cellData{
				RTTms:    int(rtt / time.Millisecond),
				LossRate: loss,
				Samples:  samples,
				Stats:    stats,
			})
			log.Printf(
				"bench-E3: cell %d/%d (rtt=%dms loss=%.2f) → P99=%.0fms fast=%.1f%% slow=%.1f%% fail=%.1f%%",
				cellNum, totalCells, int(rtt/time.Millisecond), loss,
				stats.P99ms, stats.FastPathPct, stats.SlowPathPct, stats.FailurePct,
			)
		}
	}

	wallEnd := time.Now().UTC()

	dists := map[string]distSpec{
		"hot":  distSpecFor(tiers.TierHot),
		"warm": distSpecFor(tiers.TierWarm),
		"cold": distSpecFor(tiers.TierCold),
	}
	rttCellsMS := make([]int, len(rttCells))
	for i, d := range rttCells {
		rttCellsMS[i] = int(d / time.Millisecond)
	}
	wanDesc := "in-process simTier with extraRTT (uniform across tiers) + per-Get loss-coin"

	rec := runRecord{
		SchemaVersion: 1,
		RunID:         runID,
		StartedAt:     wallStart.Format(time.RFC3339Nano),
		CompletedAt:   wallEnd.Format(time.RFC3339Nano),
		Config: runConfig{
			NumBundles:        nBundles,
			RepsPerCell:       reps,
			Seed:              seed,
			SLOBudgetMS:       sloMS,
			RTTCellsMS:        rttCellsMS,
			LossCells:         append([]float64(nil), lossCells...),
			TierDistributions: dists,
			SamplingPolicy:    "uniform-random over eval/results/synthea-100000/sizes.csv (proportional to measured 100K distribution)",
			WANMechanism:      wanDesc,
		},
		Cells: cells,
	}

	runJSONPath := filepath.Join(outDir, "run-"+runID+".json")
	if err := writeJSON(runJSONPath, rec); err != nil {
		return err
	}
	log.Printf("bench-E3: wrote %s", runJSONPath)

	envPath := filepath.Join(outDir, "env.json")
	if err := writeJSON(envPath, envJSON{
		CommitHash:        commit,
		GoVersion:         runtime.Version(),
		OS:                runtime.GOOS,
		Kernel:            unameOrEmpty(),
		CPU:               cpuOrEmpty(),
		NetworkPath:       "in-process simTier (no network egress); WAN impairment via extraRTT + per-Get loss-coin",
		WallStart:         wallStart.Format(time.RFC3339Nano),
		WallEnd:           wallEnd.Format(time.RFC3339Nano),
		BenchID:           "E3",
		SamplingPolicy:    rec.Config.SamplingPolicy,
		WANMechanism:      wanDesc,
		TierDistributions: dists,
	}); err != nil {
		return err
	}
	log.Printf("bench-E3: wrote %s", envPath)

	printSummary(rec, wallEnd.Sub(wallStart))
	return nil
}

// summariseCell flattens samples → P50/P95/P99 + path-mode percentages.
func summariseCell(samples []sampleStat) cellStats {
	if len(samples) == 0 {
		return cellStats{}
	}
	xs := make([]int64, len(samples))
	var fast, slow, fail int
	for i, s := range samples {
		xs[i] = s.LatencyNS
		switch s.Mode {
		case "fast":
			fast++
		case "slow":
			slow++
		case "failure":
			fail++
		}
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] })
	n := float64(len(samples))
	return cellStats{
		P50ms:       float64(pct(xs, 0.50)) / 1e6,
		P95ms:       float64(pct(xs, 0.95)) / 1e6,
		P99ms:       float64(pct(xs, 0.99)) / 1e6,
		FastPathPct: 100.0 * float64(fast) / n,
		SlowPathPct: 100.0 * float64(slow) / n,
		FailurePct:  100.0 * float64(fail) / n,
		NSamples:    len(samples),
	}
}

// sealAll seals plaintext in 16-KiB blocks (matches cmd/gateway sealAll).
func sealAll(key [32]byte, plaintext []byte) ([]byte, error) {
	out := make([]byte, 0, len(plaintext)+(len(plaintext)/merkle.ChunkSize+1)*(crypto.NonceSize+16))
	for off := 0; off < len(plaintext); off += merkle.ChunkSize {
		end := off + merkle.ChunkSize
		if end > len(plaintext) {
			end = len(plaintext)
		}
		sealed, err := crypto.SealChunk(key, plaintext[off:end])
		if err != nil {
			return nil, err
		}
		out = append(out, sealed...)
	}
	return out, nil
}

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

// printSummary emits a small text-mode 3×3 of P99 ms + fast-path% so the
// operator can confirm the matrix shape at a glance.
func printSummary(rec runRecord, wall time.Duration) {
	// Index cells by (rtt, loss) for the grid render.
	type key struct {
		rtt  int
		loss float64
	}
	idx := make(map[key]cellStats, len(rec.Cells))
	for _, c := range rec.Cells {
		idx[key{c.RTTms, c.LossRate}] = c.Stats
	}
	fmt.Println()
	fmt.Println("E3 P99 latency (ms) and fast-path % matrix:")
	// Header row: RTT columns.
	fmt.Printf("%-10s", "loss \\ rtt")
	for _, r := range rec.Config.RTTCellsMS {
		fmt.Printf("  %-22s", fmt.Sprintf("%d ms", r))
	}
	fmt.Println()
	for _, l := range rec.Config.LossCells {
		fmt.Printf("%-10s", fmt.Sprintf("%.0f%%", l*100))
		for _, r := range rec.Config.RTTCellsMS {
			st := idx[key{r, l}]
			fmt.Printf("  %-22s", fmt.Sprintf("P99=%6.0f fast=%5.1f%%", st.P99ms, st.FastPathPct))
		}
		fmt.Println()
	}
	fmt.Printf("\nbench-E3: %d cells × %d bundles × %d reps; wall=%s\n",
		len(rec.Cells), rec.Config.NumBundles, rec.Config.RepsPerCell,
		wall.Round(time.Millisecond),
	)
}

// pct picks a percentile from a pre-sorted slice. Empty slice → 0.
func pct(xs []int64, q float64) int64 {
	if len(xs) == 0 {
		return 0
	}
	idx := int(float64(len(xs)-1) * q)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(xs) {
		idx = len(xs) - 1
	}
	return xs[idx]
}
