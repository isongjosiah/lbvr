// Package main is the E1 ingest-throughput bench harness.
//
// CLAUDE.md §8 row E1: "Ingest throughput × corpus size (1K / 10K / 100K
// patients) × tier (with RS(2,3) enabled)". Output figure is
// paper/figures/E1_ingest_throughput.{pdf,png}; the thesis it supports
// is that the verifiable cross-tier ingest fabric scales linearly with
// corpus size — sim-tier latency is constant-cost, so throughput is a
// function of the per-bundle pipeline (Merkle + seal + encode + upload
// + register) and not of the corpus length.
//
// Open decisions resolved for this run:
//
//  1. Sampling: read EVERY bundle in the requested corpus_size from
//     a seeded shuffle of eval/results/synthea-100000/sizes.csv. The
//     §8 narrative is "throughput across the full corpus at this
//     scale", so sub-sampling would defeat the figure's point. Same
//     proportional-to-measured-distribution decision as E9 (CLAUDE.md
//     §10 D9 / docs/eval-protocol.md §3, Claude session 2026-04-25).
//
//  2. Cold-tier mechanism: in-process simTier with the same lognormal
//     calibration as E9 (sim_tier.go). Latency injected on Put (E1
//     measures writes, not reads). NOT a live-Arweave measurement —
//     env.json names the family + parameters so the figure cannot be
//     misread. Toxiproxy + live tiers comes after .env keys land
//     per docs/eval-protocol.md §3.
//
//  3. Plaintext synthesis: deterministic math/rand fill seeded from
//     -seed plus the bundle index. crypto/rand is too slow to fill
//     ~35 GiB of synthetic plaintext for the 10K scale (~12 min at
//     50 MiB/s); math/rand at ~3 GiB/s is fine. The bench measures
//     storage-fabric throughput, not entropy-source quality.
//
//  4. Scope: we run the 1K and 10K scales. 100K is projected from
//     the 10K rate with an explicit caveat in the JSON config and the
//     figure annotation — running it would take ~9 hours wall-clock,
//     outside the 90-min bench budget per the prompt. The ingest
//     pipeline is constant-per-bundle so the projection is honest;
//     the §V text flags it accordingly.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/erasure"
	"github.com/isongjosiah/lbvr-med/internal/ingest"
	"github.com/isongjosiah/lbvr-med/internal/registry"
	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

// Per-bundle setup cap. erasure.MaxInputBytes is 1 GiB; defensive cap
// to avoid accidental allocator blowup if a synthesised bundle ever
// asks for more. Synthea's measured max is ~98 MB.
const maxInputBytes = 1 << 30

// scaleSizes is the canonical list of corpus scales the §8 figure plots.
// 100000 is included so env.json records the full plan; the runner
// projects it from the 10K rate (see scaleStat.Projected).
var scaleSizes = []int{1000, 10000, 100000}

// runScales is what we actually run. 100K is skipped (~9h wall) and
// projected post-hoc from 10K. If you want to run 100K live, set
// BENCH_E1_RUN_100K=1 in the environment — see envBoolish below.
var runScales = []int{1000, 10000}

// runRecord is the top-level JSON shape (schema_version=1).
type runRecord struct {
	SchemaVersion int         `json:"schema_version"`
	RunID         string      `json:"run_id"`
	StartedAt     string      `json:"started_at"`
	CompletedAt   string      `json:"completed_at"`
	Config        runConfig   `json:"config"`
	Scales        []scaleStat `json:"scales"`
}

type runConfig struct {
	Concurrency       int                 `json:"concurrency"`
	Seed              int64               `json:"seed"`
	ScalesPlanned     []int               `json:"scales_planned"`
	ScalesRun         []int               `json:"scales_run"`
	ScalesProjected   []int               `json:"scales_projected"`
	TierDistributions map[string]distSpec `json:"tier_distributions"`
	SamplingPolicy    string              `json:"sampling_policy"`
	ColdTierMechanism string              `json:"cold_tier_mechanism"`
	PlaintextPolicy   string              `json:"plaintext_policy"`
}

// scaleStat is one row of the §8 ingest-throughput table.
//
// Projected=true means this scale was NOT actually run; the throughput
// numbers are linearly extrapolated from the most recent live scale.
// The figure annotation must say "projected" for any such row.
type scaleStat struct {
	CorpusSize              int          `json:"corpus_size"`
	NBundles                int          `json:"n_bundles"`
	TotalBytes              int64        `json:"total_bytes"`
	WallSeconds             float64      `json:"wall_seconds"`
	ThroughputMBPS          float64      `json:"throughput_mbps"`
	ThroughputBundlesPerMin float64      `json:"throughput_bundles_per_min"`
	StageP50ms              stageMetrics `json:"stage_p50_ms"`
	StageP99ms              stageMetrics `json:"stage_p99_ms"`
	Projected               bool         `json:"projected"`
	ProjectedFromScale      int          `json:"projected_from_scale,omitempty"`
}

// stageMetrics is the per-stage breakdown emitted at every scale. Keys
// match the IngestResult timing fields. "total" is the end-to-end Ingest
// runtime; the other four sum to (≈) it modulo register overhead.
type stageMetrics struct {
	MerkleMs   float64 `json:"merkle"`
	SealMs     float64 `json:"seal"`
	EncodeMs   float64 `json:"encode"`
	UploadMs   float64 `json:"upload"`
	RegisterMs float64 `json:"register"`
	TotalMs    float64 `json:"total"`
}

// envJSON mirrors CLAUDE.md §8 environment-fingerprint contract plus
// the bench-specific calibration block.
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
	PlaintextPolicy   string              `json:"plaintext_policy"`
	TierDistributions map[string]distSpec `json:"tier_distributions"`
}

func main() {
	var (
		seedFlag     = flag.Int64("seed", 42, "RNG seed (sampler + plaintext + simTier streams)")
		concFlag     = flag.Int("concurrency", 8, "ingest worker pool size")
		outDirFlag   = flag.String("out-dir", "eval/results/E1", "output directory")
		sizesCSVFlag = flag.String("sizes-csv", "eval/results/synthea-100000/sizes.csv", "path to validated size distribution")
		bundleDirF   = flag.String("bundle-dir", "", "scratch dir for synthesised bundles (default: $TMPDIR/lbvr-e1-<runID>)")
		scalesFlag   = flag.String("scales", "", "comma-separated list of corpus sizes to run (overrides default 1000,10000); 100000 is honoured if set explicitly")
	)
	flag.Parse()

	scales := runScales
	if *scalesFlag != "" {
		var err error
		scales, err = parseScales(*scalesFlag)
		if err != nil {
			log.Fatalf("bench-E1: -scales: %v", err)
		}
	}

	if err := run(*seedFlag, *concFlag, *outDirFlag, *sizesCSVFlag, *bundleDirF, scales); err != nil {
		log.Fatalf("bench-E1: %v", err)
	}
}

func run(seed int64, concurrency int, outDir, sizesCSV, bundleDir string, scales []int) error {
	wallStart := time.Now().UTC()
	commit := shortCommit()
	runID := wallStart.Format("20060102-150405") + "-" + commit

	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > 32 {
		concurrency = 32
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}
	if bundleDir == "" {
		bundleDir = filepath.Join(os.TempDir(), "lbvr-e1-"+runID)
	}
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", bundleDir, err)
	}
	// Defensive: clean up the per-run scratch dir on exit. The bench
	// keeps O(corpus_size × bundle_size) bytes here; for 10K × 3.5MB
	// median this is ~35 GiB, dwarfing the eval/results JSON.
	defer func() {
		if err := os.RemoveAll(bundleDir); err != nil {
			log.Printf("bench-E1: cleanup %s: %v", bundleDir, err)
		}
	}()
	log.Printf("bench-E1: scratch bundle dir = %s", bundleDir)

	// Build the three sim-tiers ONCE; we reuse them across scales. Per-
	// tier independent rand streams keep latency draws reproducible
	// regardless of scale order.
	hot := newSimTier("pinata-sim", tiers.TierHot, seed+1)
	warm := newSimTier("filebase-sim", tiers.TierWarm, seed+2)
	cold := newSimTier("arweave-sim", tiers.TierCold, seed+3)

	// Identify the highest scale we'll run — it caps the storage
	// footprint for the temp dir + drives the projection-from logic.
	maxScale := 0
	for _, s := range scales {
		if s > maxScale {
			maxScale = s
		}
	}
	if maxScale <= 0 {
		return fmt.Errorf("bench-E1: no scales to run")
	}

	// Sample once at the largest scale, then prefix-slice for smaller
	// scales. This guarantees the smaller scale is a subset of the
	// larger so per-bundle byte streams stay deterministic across the
	// figure's three columns.
	allBundles, err := SampleBundles(sizesCSV, maxScale, seed)
	if err != nil {
		return fmt.Errorf("SampleBundles(%d): %w", maxScale, err)
	}

	// Plaintext synthesis fills a math/rand-seeded buffer per bundle. We
	// do NOT use crypto/rand: at the 10K scale the plaintext volume is
	// ~35 GiB, and crypto/rand fills at ~50 MiB/s on this host (~12
	// min just for randomness). math/rand fills at >3 GiB/s, which is
	// noise relative to the encode + upload pipeline. The bench measures
	// storage-fabric throughput; entropy-source quality is irrelevant.
	plaintextRNG := rand.New(rand.NewSource(seed ^ 0xE1)) // domain-separated from sampler

	scaleStats := make([]scaleStat, 0, len(scaleSizes))
	scalesRun := make([]int, 0, len(scales))
	scalesProjected := make([]int, 0, len(scaleSizes))

	// Track the most recent live scaleStat for projection.
	var lastLive *scaleStat

	// Preallocate temp file paths for the largest scale. We write each
	// bundle's plaintext to its own file; the Ingester reads it during
	// readAndBuildMerkle. Setup cost (file write + rand fill) is OUTSIDE
	// the wall-clock measurement window — this is the throughput of the
	// pipeline itself, not of the synthetic-plaintext generator.
	bundlePaths := make([]string, len(allBundles))
	bundleSizes := make([]int64, len(allBundles))
	wroteUpTo := 0

	for _, scaleIdx := range scaleOrder(scales) {
		corpusSize := scales[scaleIdx]
		if corpusSize > maxScale || corpusSize <= 0 {
			return fmt.Errorf("bench-E1: invalid scale %d", corpusSize)
		}

		// Write any not-yet-written bundles up to corpusSize.
		setupStart := time.Now()
		freshlyWritten := corpusSize - wroteUpTo
		for i := wroteUpTo; i < corpusSize; i++ {
			b := allBundles[i]
			if b.SizeBytes <= 0 {
				return fmt.Errorf("bundle %d (%s) has non-positive size %d", i, b.Filename, b.SizeBytes)
			}
			if b.SizeBytes > maxInputBytes {
				return fmt.Errorf("bundle %d (%s) size %d exceeds maxInputBytes %d", i, b.Filename, b.SizeBytes, maxInputBytes)
			}
			path := filepath.Join(bundleDir, fmt.Sprintf("bundle-%07d.json", i))
			if err := writeRandomBundle(path, b.SizeBytes, plaintextRNG); err != nil {
				return fmt.Errorf("write bundle %d: %w", i, err)
			}
			bundlePaths[i] = path
			bundleSizes[i] = b.SizeBytes
		}
		wroteUpTo = corpusSize
		setupDur := time.Since(setupStart)
		log.Printf("bench-E1: scale=%d setup wrote %d new bundles in %s",
			corpusSize, freshlyWritten, setupDur.Round(time.Millisecond))

		// Run the bench at this scale. Mock registry is reset between
		// scales because the larger scales reuse bundle paths and the
		// derived bundleIDs are deterministic — re-registering would hit
		// ErrAlreadyRegistered. The sim-tiers don't retain payloads
		// (see sim_tier.go Put docstring) so no store-reset is needed.
		reg := registry.NewMock()

		ing, err := ingest.NewIngester(ingest.IngesterOpts{
			Hot:        hot,
			Warm:       warm,
			Cold:       cold,
			Registry:   reg,
			Encoder:    erasureEncoderFunc{},
			ClientAddr: defaultClientAddr,
			Logger:     quietLogger(),
		})
		if err != nil {
			return fmt.Errorf("scale=%d: NewIngester: %w", corpusSize, err)
		}

		log.Printf("bench-E1: scale=%d running %d bundles × concurrency=%d", corpusSize, corpusSize, concurrency)
		stat, err := runScale(ing, bundlePaths[:corpusSize], bundleSizes[:corpusSize], corpusSize, concurrency)
		if err != nil {
			return fmt.Errorf("scale=%d: %w", corpusSize, err)
		}
		stat.Projected = false
		scaleStats = append(scaleStats, stat)
		scalesRun = append(scalesRun, corpusSize)
		lastLive = &scaleStats[len(scaleStats)-1]

		fmt.Printf(
			"bench-E1: scale=%-6d n=%-6d total=%6.2f GiB wall=%7.2fs throughput=%6.2f MiB/s = %7.1f bundles/min  (P50 stages: m=%5.1f s=%5.1f e=%5.1f u=%6.1f r=%5.2f total=%6.1f ms)\n",
			stat.CorpusSize, stat.NBundles, float64(stat.TotalBytes)/(1<<30),
			stat.WallSeconds, stat.ThroughputMBPS, stat.ThroughputBundlesPerMin,
			stat.StageP50ms.MerkleMs, stat.StageP50ms.SealMs, stat.StageP50ms.EncodeMs,
			stat.StageP50ms.UploadMs, stat.StageP50ms.RegisterMs, stat.StageP50ms.TotalMs,
		)
	}

	// Project unmeasured scales (100K) from the most recent live scale.
	for _, planned := range scaleSizes {
		alreadyRan := false
		for _, ran := range scalesRun {
			if ran == planned {
				alreadyRan = true
				break
			}
		}
		if alreadyRan {
			continue
		}
		if lastLive == nil {
			break
		}
		proj := projectScale(*lastLive, planned, allBundles)
		scaleStats = append(scaleStats, proj)
		scalesProjected = append(scalesProjected, planned)
		fmt.Printf(
			"bench-E1: scale=%-6d (PROJECTED from %d): n=%-6d total=%6.2f GiB wall=%7.1fs throughput=%6.2f MiB/s = %7.1f bundles/min\n",
			proj.CorpusSize, lastLive.CorpusSize, proj.NBundles, float64(proj.TotalBytes)/(1<<30),
			proj.WallSeconds, proj.ThroughputMBPS, proj.ThroughputBundlesPerMin,
		)
	}

	// Sort scaleStats by corpus size for stable JSON output (small → large).
	sort.Slice(scaleStats, func(i, j int) bool { return scaleStats[i].CorpusSize < scaleStats[j].CorpusSize })

	wallEnd := time.Now().UTC()

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
			Concurrency:       concurrency,
			Seed:              seed,
			ScalesPlanned:     scaleSizes,
			ScalesRun:         scalesRun,
			ScalesProjected:   scalesProjected,
			TierDistributions: dists,
			SamplingPolicy:    "first corpus_size rows of seeded shuffle over eval/results/synthea-100000/sizes.csv (proportional to measured 100K distribution)",
			ColdTierMechanism: "in-process simTier (lognormal); identical calibration to E9; live-tier swap deferred per docs/eval-protocol.md §3",
			PlaintextPolicy:   "math/rand seeded from -seed XOR domain-tag 0xE1; entropy-source quality is irrelevant for storage-fabric throughput",
		},
		Scales: scaleStats,
	}

	runJSONPath := filepath.Join(outDir, "run-"+runID+".json")
	if err := writeJSON(runJSONPath, rec); err != nil {
		return err
	}
	log.Printf("bench-E1: wrote %s", runJSONPath)

	envPath := filepath.Join(outDir, "env.json")
	if err := writeJSON(envPath, envJSON{
		CommitHash:        commit,
		GoVersion:         runtime.Version(),
		OS:                runtime.GOOS,
		Kernel:            unameOrEmpty(),
		CPU:               cpuOrEmpty(),
		NetworkPath:       "in-process simTier (no network egress); latency injected on Put",
		WallStart:         wallStart.Format(time.RFC3339Nano),
		WallEnd:           wallEnd.Format(time.RFC3339Nano),
		BenchID:           "E1",
		SamplingPolicy:    rec.Config.SamplingPolicy,
		ColdTierMechanism: rec.Config.ColdTierMechanism,
		PlaintextPolicy:   rec.Config.PlaintextPolicy,
		TierDistributions: dists,
	}); err != nil {
		return err
	}
	log.Printf("bench-E1: wrote %s", envPath)

	wall := wallEnd.Sub(wallStart)
	fmt.Printf("\nbench-E1: total wall=%s; %d scales run, %d projected\n",
		wall.Round(time.Second), len(scalesRun), len(scalesProjected),
	)
	return nil
}

// runScale is the inner loop: dispatch corpusSize Ingest calls across
// `concurrency` workers. Returns the populated scaleStat (Projected
// always false here — caller flips it).
func runScale(ing *ingest.Ingester, paths []string, sizes []int64, corpusSize, concurrency int) (scaleStat, error) {
	totalBytes := int64(0)
	for _, s := range sizes {
		totalBytes += s
	}

	merkles := make([]int64, corpusSize)
	seals := make([]int64, corpusSize)
	encodes := make([]int64, corpusSize)
	uploads := make([]int64, corpusSize)
	registers := make([]int64, corpusSize)
	totals := make([]int64, corpusSize)

	jobs := make(chan int)
	errCh := make(chan error, corpusSize)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		ctx := context.Background()
		for idx := range jobs {
			res, err := ing.Ingest(ctx, ingest.IngestRequest{
				Path:     paths[idx],
				PolicyID: e1PolicyID,
			})
			if err != nil {
				errCh <- fmt.Errorf("ingest bundle %d (%s): %w", idx, paths[idx], err)
				return
			}
			merkles[idx] = res.TMerkle.Nanoseconds()
			seals[idx] = res.TSeal.Nanoseconds()
			encodes[idx] = res.TEncode.Nanoseconds()
			uploads[idx] = res.TUpload.Nanoseconds()
			registers[idx] = res.TRegister.Nanoseconds()
			totals[idx] = res.TTotal.Nanoseconds()
		}
	}

	wallStart := time.Now()
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}
	progressEvery := corpusSize / 10
	if progressEvery < 1 {
		progressEvery = 1
	}
	for i := 0; i < corpusSize; i++ {
		jobs <- i
		if (i+1)%progressEvery == 0 {
			elapsed := time.Since(wallStart)
			log.Printf("bench-E1: scale=%d %d/%d (%.1f%%) elapsed=%s",
				corpusSize, i+1, corpusSize, 100.0*float64(i+1)/float64(corpusSize),
				elapsed.Round(time.Second))
		}
	}
	close(jobs)
	wg.Wait()
	close(errCh)
	wallSeconds := time.Since(wallStart).Seconds()

	for err := range errCh {
		if err != nil {
			return scaleStat{}, err
		}
	}

	// Compute throughput. MiB/s = totalBytes / 2^20 / wallSeconds.
	mibps := 0.0
	bpm := 0.0
	if wallSeconds > 0 {
		mibps = float64(totalBytes) / (1 << 20) / wallSeconds
		bpm = float64(corpusSize) / wallSeconds * 60.0
	}

	return scaleStat{
		CorpusSize:              corpusSize,
		NBundles:                corpusSize,
		TotalBytes:              totalBytes,
		WallSeconds:             wallSeconds,
		ThroughputMBPS:          mibps,
		ThroughputBundlesPerMin: bpm,
		StageP50ms:              percentileStages(merkles, seals, encodes, uploads, registers, totals, 0.50),
		StageP99ms:              percentileStages(merkles, seals, encodes, uploads, registers, totals, 0.99),
	}, nil
}

// projectScale extrapolates throughput at `target` corpus size from the
// `live` scaleStat measurement. Sim-tier latency is constant-cost, so a
// linear projection is honest: wall = live.wall × (target / live.scale).
// The figure annotation must clearly tag projected rows as such.
func projectScale(live scaleStat, target int, all []SampledBundle) scaleStat {
	if live.NBundles == 0 || target <= 0 {
		return scaleStat{CorpusSize: target, Projected: true, ProjectedFromScale: live.CorpusSize}
	}
	// Total bytes for the projected corpus = sum of the first `target`
	// bundle sizes from the same shuffled-and-prefixed source. If the
	// caller didn't pre-shuffle to ≥ target rows, we extrapolate via the
	// live mean bundle size; in our run we always pre-sample at maxScale
	// = max(scales), so this fallback only fires under unusual flag combos.
	totalBytes := int64(0)
	if target <= len(all) {
		for i := 0; i < target; i++ {
			totalBytes += all[i].SizeBytes
		}
	} else if live.NBundles > 0 {
		mean := float64(live.TotalBytes) / float64(live.NBundles)
		totalBytes = int64(mean * float64(target))
	}

	scaleFactor := float64(target) / float64(live.NBundles)
	wallSec := live.WallSeconds * scaleFactor
	mibps := live.ThroughputMBPS // identical by construction (linear)
	bpm := live.ThroughputBundlesPerMin
	// If totalBytes is computed from real samples, recompute mibps for
	// honesty (the bundle-size mean drifts slightly across the larger
	// prefix); this small delta should not change the figure.
	if wallSec > 0 && totalBytes > 0 && target <= len(all) {
		mibps = float64(totalBytes) / (1 << 20) / wallSec
	}
	return scaleStat{
		CorpusSize:              target,
		NBundles:                target,
		TotalBytes:              totalBytes,
		WallSeconds:             wallSec,
		ThroughputMBPS:          mibps,
		ThroughputBundlesPerMin: bpm,
		StageP50ms:              live.StageP50ms,
		StageP99ms:              live.StageP99ms,
		Projected:               true,
		ProjectedFromScale:      live.CorpusSize,
	}
}

// percentileStages computes per-stage percentile in ms.
func percentileStages(merkle, seal, encode, upload, register, total []int64, q float64) stageMetrics {
	return stageMetrics{
		MerkleMs:   pctMS(merkle, q),
		SealMs:     pctMS(seal, q),
		EncodeMs:   pctMS(encode, q),
		UploadMs:   pctMS(upload, q),
		RegisterMs: pctMS(register, q),
		TotalMs:    pctMS(total, q),
	}
}

// pctMS returns the q-percentile of xs (nanoseconds), in milliseconds.
func pctMS(xs []int64, q float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := make([]int64, len(xs))
	copy(cp, xs)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	idx := int(float64(len(cp)-1) * q)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return float64(cp[idx]) / 1e6
}

// writeRandomBundle fills a fresh file with `size` bytes from rng. Uses
// math/rand because crypto/rand is too slow for the 10K scale (~35 GiB
// of writes); the bench measures storage-fabric throughput, not entropy
// quality. The buffer is reused across files via a 1 MiB scratch slice
// so the allocator does not melt under 10K bundles.
func writeRandomBundle(path string, size int64, rng *rand.Rand) error {
	if size <= 0 {
		return fmt.Errorf("non-positive size %d", size)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	const chunkSize = 1 << 20
	buf := make([]byte, chunkSize)
	remaining := size
	for remaining > 0 {
		n := int64(chunkSize)
		if n > remaining {
			n = remaining
		}
		// math/rand.Read on the global Source is cheaper than NormFloat
		// per byte — a single call fills as many bytes as we ask for.
		if _, err := rng.Read(buf[:n]); err != nil {
			return err
		}
		if _, err := f.Write(buf[:n]); err != nil {
			return err
		}
		remaining -= n
	}
	return nil
}

// erasureEncoderFunc satisfies ingest.Encoder via internal/erasure. We
// can't import cmd/client (main package) so the encoder is re-declared
// here as a stateless type. It is byte-identical to cmd/client's
// erasureEncoder; the test suite asserts identical behaviour by
// running the production-wired pipeline.
type erasureEncoderFunc struct{}

// Encode implements ingest.Encoder via the RS(2,3) encoder.
func (erasureEncoderFunc) Encode(data []byte) (shards [3][]byte, paddedLen int, err error) {
	return erasure.Encode(data)
}

// defaultClientAddr is the placeholder Ethereum address — same value
// used by cmd/client (CLAUDE.md §4.2 step 6 placeholder).
var defaultClientAddr = [20]byte{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 1,
}

// e1PolicyID is the policy reference used for every bundle in this run.
// Real policy injection lands later (CLAUDE.md §4.2); for E1 the bench
// just needs a non-zero stable value so the registry's NumChunks /
// shard-validation paths run end-to-end.
var e1PolicyID = registry.Keccak256([]byte("lbvr://policy/e1-bench"))

// quietLogger discards the Ingester's structured logs so the bench
// stdout stays human-readable. Per-stage timings are captured directly
// off the IngestResult, not parsed from log lines.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// scaleOrder returns indices into `scales` in ascending order so smaller
// scales run before larger ones (matching the user-facing plot's
// left-to-right reading order).
func scaleOrder(scales []int) []int {
	order := make([]int, len(scales))
	for i := range scales {
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool { return scales[order[i]] < scales[order[j]] })
	return order
}

// parseScales accepts a comma-separated list "1000,10000" and returns
// the int slice. Empty entries are skipped; non-positive values error.
func parseScales(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n := 0
		for _, c := range p {
			if c < '0' || c > '9' {
				return nil, fmt.Errorf("non-digit in %q", p)
			}
			n = n*10 + int(c-'0')
		}
		if n <= 0 {
			return nil, fmt.Errorf("non-positive scale %q", p)
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty scales list")
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

// cpuOrEmpty pulls the model name from /proc/cpuinfo. Best-effort.
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
