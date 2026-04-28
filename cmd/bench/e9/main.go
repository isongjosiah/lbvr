// Package main is the E9 erasure-recovery bench harness.
//
// CLAUDE.md §4.5 / §8 / docs/erasure-design.md §6 / docs/eval-protocol.md §2
// row E9. Measures internal/gateway.Recover across four failure modes:
//
//	baseline  — all three shards available; fast path expected
//	D0_lost   — hot tier dropped; slow path with cold-tier fetch
//	D1_lost   — warm tier dropped; slow path with cold-tier fetch
//	P0_lost   — cold tier dropped; fast path (parity unused)
//
// Open decisions resolved for this run (Claude session 2026-04-25):
//
//   1. Sampling: uniform-random over the 114,693 measured 100K bundles
//      in eval/results/synthea-100000/sizes.csv → proportional to the
//      measured distribution, including the 11.3% tail above 5 MB up to
//      ~98 MB. Stratified-to-§4.5-bands would clip the tail and bias the
//      figure toward the median band.
//
//   2. Cold-tier mechanism: in-process simTier with a calibrated
//      lognormal latency distribution (see sim_tier.go). NOT a
//      live-Arweave measurement — env.json names the family and
//      parameters so the figure cannot be misread. Toxiproxy + live
//      tiers comes after .env keys land per docs/eval-protocol.md §3.
//
// Bundle bytes are synthesised via crypto/rand: we measure storage-fabric
// latency, not FHIR parser speed. Each synthesised bundle is sealed via
// internal/crypto.SealChunk at 16-KiB granularity (matching merkle.ChunkSize)
// then erasure.Encode, so paddedLen and shard sizes match the real ingest
// path. We deliberately skip merkle.Build because gateway.Recover reads
// shard bytes only — Merkle verification happens after Recover returns,
// outside our measurement window.

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

// modesDefault is the canonical single-failure list (E9 / §8 Fig 10).
// Extended modes for double-failure (E9-multi / §8 Fig 10b) are listed
// after; -modes selects which subset to run. `modes` itself is the
// active selection for this run (mutated in main after flag parse).
var modesDefault = []string{"baseline", "D0_lost", "D1_lost", "P0_lost"}
var modesMulti = []string{"D0D1_lost", "D0P0_lost", "D1P0_lost"}
var modesAll = append(append([]string{}, modesDefault...), modesMulti...)
var modes = modesDefault

// Per-bundle setup cap. erasure.MaxInputBytes is 1 GiB; defensive cap
// to avoid accidental allocator blowup if a synthesised bundle ever
// asks for more. Synthea's measured max is ~98 MB.
const maxInputBytes = 1 << 30

// runRecord is the top-level JSON shape (schema_version=1).
type runRecord struct {
	SchemaVersion int          `json:"schema_version"`
	RunID         string       `json:"run_id"`
	StartedAt     string       `json:"started_at"`
	CompletedAt   string       `json:"completed_at"`
	Config        runConfig    `json:"config"`
	Samples       []sampleStat `json:"samples"`
}

type runConfig struct {
	NumBundles        int                 `json:"num_bundles"`
	RepsPerMode       int                 `json:"reps_per_mode"`
	Seed              int64               `json:"seed"`
	SLOBudgetMS       int                 `json:"slo_budget_ms"`
	Modes             []string            `json:"modes"`
	TierDistributions map[string]distSpec `json:"tier_distributions"`
	SamplingPolicy    string              `json:"sampling_policy"`
	ColdTierMechanism string              `json:"cold_tier_mechanism"`
}

type sampleStat struct {
	BundleIdx int                 `json:"bundle_idx"`
	Filename  string              `json:"filename"`
	SizeBytes int64               `json:"size_bytes"`
	PaddedLen int                 `json:"padded_len"`
	ShardSize int                 `json:"shard_size"`
	ModeRuns  map[string][]modeRu `json:"mode_runs"`
}

// modeRu — one Recover invocation outcome. Tagged "ru" not "run" so
// `go vet` doesn't flag the unused naming clash with runRecord.
type modeRu struct {
	LatencyNS int64  `json:"latency_ns"`
	Mode      string `json:"mode"` // "fast" / "slow" / "failure"
}

// envJSON mirrors CLAUDE.md §8 environment-fingerprint contract plus
// the bench-specific calibration block so the figure is unambiguous.
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
	ColdTierMechanism string              `json:"cold_tier_mechanism"`
	TierDistributions map[string]distSpec `json:"tier_distributions"`
}

func main() {
	var (
		nFlag        = flag.Int("n", 100, "number of bundles to sample")
		repsFlag     = flag.Int("reps", 10, "repetitions per mode per bundle")
		seedFlag     = flag.Int64("seed", 42, "RNG seed (sampler + simTier streams)")
		outDirFlag   = flag.String("out-dir", "eval/results/E9", "output directory")
		sloMSFlag    = flag.Int("slo-ms", 2000, "fast-path SLO budget in milliseconds")
		sizesCSVFlag = flag.String("sizes-csv", "eval/results/synthea-100000/sizes.csv", "path to validated size distribution")
		modesFlag    = flag.String("modes", "default", "mode set: 'default' (single-failure E9), 'multi' (double-failure E9-multi), 'all', or comma-separated mode names")
	)
	flag.Parse()

	switch *modesFlag {
	case "default", "":
		modes = modesDefault
	case "multi":
		modes = modesMulti
	case "all":
		modes = modesAll
	default:
		modes = strings.Split(*modesFlag, ",")
	}

	if err := run(*nFlag, *repsFlag, *seedFlag, *outDirFlag, *sloMSFlag, *sizesCSVFlag); err != nil {
		log.Fatalf("bench-E9: %v", err)
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
	samples := make([]sampleStat, 0, nBundles)

	// Per-tier independent streams so failure injection is deterministic.
	// Offsets (1, 2, 3) keep the streams disjoint from the sampler's seed.
	hot := newSimTier("pinata-sim", tiers.TierHot, seed+1)
	warm := newSimTier("filebase-sim", tiers.TierWarm, seed+2)
	cold := newSimTier("arweave-sim", tiers.TierCold, seed+3)
	tierArr := [3]tiers.Client{hot, warm, cold}
	simArr := [3]*simTier{hot, warm, cold}

	for i, b := range bundles {
		if b.SizeBytes <= 0 {
			return fmt.Errorf("bundle %d (%s) has non-positive size %d", i, b.Filename, b.SizeBytes)
		}
		if b.SizeBytes > maxInputBytes {
			return fmt.Errorf("bundle %d (%s) size %d exceeds maxInputBytes %d", i, b.Filename, b.SizeBytes, maxInputBytes)
		}

		// Build the encrypted bundle. We seal at merkle.ChunkSize
		// granularity so paddedLen matches the real ingest path. The
		// last chunk is short; that's fine — SealChunk handles it.
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
		// Defensive: zero the key after we use it. The bench never
		// logs it; this is belt-and-braces (CLAUDE.md §13 hard
		// constraints).
		for j := range key {
			key[j] = 0
		}

		shards, paddedLen, err := erasure.Encode(encrypted)
		if err != nil {
			return fmt.Errorf("erasure.Encode bundle %d: %w", i, err)
		}

		// Distribute shards across tiers — D0→hot, D1→warm, P0→cold
		// per docs/erasure-design.md §4.
		ctx := context.Background()
		var cids [3]string
		for k := 0; k < 3; k++ {
			cid, err := tierArr[k].Put(ctx, shards[k])
			if err != nil {
				return fmt.Errorf("Put bundle %d shard %d: %w", i, k, err)
			}
			cids[k] = cid
		}
		shardSize := len(shards[0])

		stat := sampleStat{
			BundleIdx: i,
			Filename:  b.Filename,
			SizeBytes: b.SizeBytes,
			PaddedLen: paddedLen,
			ShardSize: shardSize,
			ModeRuns:  make(map[string][]modeRu, len(modes)),
		}

		// Per-mode measurement. We allocate the slice once per bundle
		// and reset drops between modes so failure injection is
		// independent of run order.
		for _, mode := range modes {
			runs := make([]modeRu, 0, reps)
			for r := 0; r < reps; r++ {
				applyMode(mode, simArr)
				start := time.Now()
				_, recStats, recErr := gateway.Recover(ctx, tierArr, cids, paddedLen, sloBudget)
				latency := time.Since(start)

				// Sanity: baseline must never fail. If it does, the
				// rest of the run is meaningless.
				if mode == "baseline" && recErr != nil {
					return fmt.Errorf("bench-E9: baseline failed bundle %d rep %d: %v", i, r, recErr)
				}
				if mode == "baseline" && recStats.Mode == gateway.RecoveryFailure {
					return fmt.Errorf("bench-E9: baseline reported RecoveryFailure bundle %d rep %d", i, r)
				}

				runs = append(runs, modeRu{
					LatencyNS: latency.Nanoseconds(),
					Mode:      recStats.Mode.String(),
				})
				resetTiers(simArr)
			}
			stat.ModeRuns[mode] = runs
		}
		samples = append(samples, stat)

		if (i+1)%10 == 0 || i == nBundles-1 {
			log.Printf("bench-E9: %d/%d bundles done (%.1f%%)", i+1, nBundles, 100.0*float64(i+1)/float64(nBundles))
		}
	}

	wallEnd := time.Now().UTC()

	// Compose the run record.
	dists := map[string]distSpec{
		"hot":  distSpecFor(tiers.TierHot),
		"warm": distSpecFor(tiers.TierWarm),
		"cold": distSpecFor(tiers.TierCold),
	}
	rec := runRecord{
		SchemaVersion: 1,
		RunID:         runID,
		StartedAt:     wallStart.Format(time.RFC3339Nano),
		CompletedAt:   wallEnd.Format(time.RFC3339Nano),
		Config: runConfig{
			NumBundles:        nBundles,
			RepsPerMode:       reps,
			Seed:              seed,
			SLOBudgetMS:       sloMS,
			Modes:             modes,
			TierDistributions: dists,
			SamplingPolicy:    "uniform-random over eval/results/synthea-100000/sizes.csv (proportional to measured 100K distribution)",
			ColdTierMechanism: "in-process simTier (lognormal); Toxiproxy + live tiers deferred per docs/eval-protocol.md §3",
		},
		Samples: samples,
	}

	runJSONPath := filepath.Join(outDir, "run-"+runID+".json")
	if err := writeJSON(runJSONPath, rec); err != nil {
		return err
	}
	log.Printf("bench-E9: wrote %s", runJSONPath)

	envPath := filepath.Join(outDir, "env.json")
	if err := writeJSON(envPath, envJSON{
		CommitHash:        commit,
		GoVersion:         runtime.Version(),
		OS:                runtime.GOOS,
		Kernel:            unameOrEmpty(),
		CPU:               cpuOrEmpty(),
		NetworkPath:       "in-process simTier (no network egress)",
		WallStart:         wallStart.Format(time.RFC3339Nano),
		WallEnd:           wallEnd.Format(time.RFC3339Nano),
		BenchID:           "E9",
		SamplingPolicy:    rec.Config.SamplingPolicy,
		ColdTierMechanism: rec.Config.ColdTierMechanism,
		TierDistributions: dists,
	}); err != nil {
		return err
	}
	log.Printf("bench-E9: wrote %s", envPath)

	printSummary(rec, wallEnd.Sub(wallStart))
	return nil
}

// applyMode flips the right simTier into Drop() for the named failure mode.
// Double-failure modes (E9-multi) drop two tiers; the recovery state machine
// is expected to RecoveryFailure as soon as it can prove insufficiency.
func applyMode(mode string, sims [3]*simTier) {
	switch mode {
	case "baseline":
		// no drops
	case "D0_lost":
		sims[0].Drop()
	case "D1_lost":
		sims[1].Drop()
	case "P0_lost":
		sims[2].Drop()
	case "D0D1_lost":
		sims[0].Drop()
		sims[1].Drop()
	case "D0P0_lost":
		sims[0].Drop()
		sims[2].Drop()
	case "D1P0_lost":
		sims[1].Drop()
		sims[2].Drop()
	}
}

func resetTiers(sims [3]*simTier) {
	for _, s := range sims {
		s.Reset()
	}
}

// sealAll seals plaintext in 16-KiB blocks and concatenates the sealed
// outputs — same shape as cmd/gateway sealAll. Output length is
// len(plaintext) + numChunks*(NonceSize+GCMTag = 28); erasure.Encode
// pads to a multiple of 2 internally.
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

// cpuOrEmpty pulls the model name from /proc/cpuinfo. Best-effort; the
// figure does not depend on it but env.json should record it.
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

// printSummary emits the one-liner stdout summary so a bench operator
// can confirm the run shape at a glance without firing up the post-
// processor.
func printSummary(rec runRecord, wall time.Duration) {
	flat := make(map[string][]int64, len(rec.Config.Modes))
	for _, m := range rec.Config.Modes {
		flat[m] = make([]int64, 0, rec.Config.NumBundles*rec.Config.RepsPerMode)
	}
	for _, s := range rec.Samples {
		for m, runs := range s.ModeRuns {
			for _, r := range runs {
				flat[m] = append(flat[m], r.LatencyNS)
			}
		}
	}
	parts := []string{}
	for _, m := range rec.Config.Modes {
		xs := flat[m]
		sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] })
		p50 := pct(xs, 0.50)
		p99 := pct(xs, 0.99)
		parts = append(parts, fmt.Sprintf("%s P50=%dms P99=%dms", m, p50/1_000_000, p99/1_000_000))
	}
	fmt.Printf(
		"bench-E9: %d bundles × %d reps × %d modes; %s; wall=%s\n",
		rec.Config.NumBundles, rec.Config.RepsPerMode, len(rec.Config.Modes),
		strings.Join(parts, "; "),
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
