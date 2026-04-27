// Package main is the E6 / E6b Byzantine-withstand bench harness.
//
// CLAUDE.md §5 (threat model) / §8 rows E6, E6b / contribution Tier 2 #4
// (tier-aware Byzantine withstand). Drives gateway.Recover against
// adversarial sim-tiers across the canonical adversary fractions
// {0, 0.10, 0.33, 0.50, 0.67}. Produces the raw JSON consumed by
// eval/scripts/e6_byzantine_curve.py (Fig 7a) and
// eval/scripts/e6b_detection_gap.py (Fig 7b).
//
// Two adversary modes:
//
//   - uniform        - byzantine replicas serve corrupt bytes on every Get.
//                      Distributed uniformly across {hot, warm, cold}.
//                      Models a generic compromised-pinning-service threat.
//   - tier-selective - byzantine replicas serve honest bytes when ctx is
//                      tagged "por_challenge" but corrupt bytes during
//                      retrievals. Biased to the cold tier (cheapest to
//                      compromise; metadata-correlated per §5). Models
//                      the §5 metadata-correlated adversary that evades
//                      standard PoR cadences.
//
// For each (mode, fraction, bundle) combination we run -reps retrievals
// and (in tier-selective mode) one matching PoR challenge per rep. The
// detection gap is PoR-success-rate − retrieval-success-rate; large gap
// means PoR cadence is blind to a real attack.
//
// Decision rule for "byzantine fraction":
//
//	The fraction names the SHARE of the global replica pool (3*N replicas
//	for N bundles) that is byzantine. For uniform mode the byzantine
//	replicas are spread across tiers via per-replica Bernoulli(fraction).
//	For tier-selective the same target fraction is met BUT byzantine
//	replicas are biased to the cold tier first, then warm, then hot —
//	mirroring the §5 framing (cold tier = longest-lived, lowest-cost-to-
//	subvert audit target). Per-bundle byzantine assignment is recorded so
//	the figure can show the realised tier breakdown, not the target.

package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/crypto"
	"github.com/isongjosiah/lbvr-med/internal/erasure"
	"github.com/isongjosiah/lbvr-med/internal/gateway"
	"github.com/isongjosiah/lbvr-med/internal/merkle"
	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

// Adversary fractions per CLAUDE.md §8 row E6: {0, 0.10, 0.33, 0.50, 0.67}.
var defaultFractions = []float64{0.0, 0.10, 0.33, 0.50, 0.67}

const maxInputBytes = 1 << 30

// modeName maps the -mode CLI flag onto the byzantineMode enum + a
// human-readable label that lands in the JSON.
var modeName = map[string]struct {
	mode  byzantineMode
	label string
}{
	"uniform":        {byzantineUniform, "uniform"},
	"tier-selective": {byzantineTierSelective, "tier_selective"},
}

// runRecord is the top-level JSON shape (schema_version=1).
type runRecord struct {
	SchemaVersion int              `json:"schema_version"`
	RunID         string           `json:"run_id"`
	StartedAt     string           `json:"started_at"`
	CompletedAt   string           `json:"completed_at"`
	Config        runConfig        `json:"config"`
	Fractions     []fractionStat   `json:"fractions"`
	Bundles       []bundleManifest `json:"bundles"` // small per-bundle metadata for reproducibility
}

type runConfig struct {
	Mode              string              `json:"mode"`
	NumBundles        int                 `json:"num_bundles"`
	RepsPerFraction   int                 `json:"reps_per_fraction"`
	Seed              int64               `json:"seed"`
	SLOBudgetMS       int                 `json:"slo_budget_ms"`
	Fractions         []float64           `json:"adversary_fractions"`
	TierDistributions map[string]distSpec `json:"tier_distributions"`
	SamplingPolicy    string              `json:"sampling_policy"`
	ColdTierMechanism string              `json:"cold_tier_mechanism"`
}

// fractionStat aggregates per-fraction outcomes. PoR fields are populated
// only in tier-selective mode; in uniform mode they remain at their zero
// value (json marshals them as 0/empty).
type fractionStat struct {
	AdversaryFraction    float64            `json:"adversary_fraction"`
	Mode                 string             `json:"mode"`
	NRetrievals          int                `json:"n_retrievals"`
	NRetrievalSuccess    int                `json:"n_retrieval_success"`
	RetrievalSuccessRate float64            `json:"retrieval_success_rate"`
	NPoRChallenges       int                `json:"n_por_challenges,omitempty"`
	NPoRSuccess          int                `json:"n_por_success,omitempty"`
	PoRSuccessRate       float64            `json:"por_success_rate,omitempty"`
	DetectionGap         float64            `json:"detection_gap,omitempty"`
	TierBreakdown        map[string]float64 `json:"tier_breakdown"`
	FailureBreakdown     map[string]int     `json:"failure_breakdown"`
}

// bundleManifest is the minimal per-bundle record needed to map a result
// back to a row in sizes.csv. We deliberately omit per-rep data here —
// the figure operates on aggregate counts only and per-rep arrays would
// bloat the JSON to ~MB scale at -bundles 100 -reps 10.
type bundleManifest struct {
	Idx       int    `json:"idx"`
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes"`
	PaddedLen int    `json:"padded_len"`
	ShardSize int    `json:"shard_size"`
}

// envJSON mirrors CLAUDE.md §8 environment-fingerprint contract plus the
// bench-specific calibration block.
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
	Mode              string              `json:"mode"`
	SamplingPolicy    string              `json:"sampling_policy"`
	ColdTierMechanism string              `json:"cold_tier_mechanism"`
	TierDistributions map[string]distSpec `json:"tier_distributions"`
}

func main() {
	var (
		modeFlag     = flag.String("mode", "uniform", "adversary mode: uniform | tier-selective")
		nFlag        = flag.Int("bundles", 100, "number of bundles to sample")
		repsFlag     = flag.Int("reps", 10, "retrievals per bundle per fraction")
		seedFlag     = flag.Int64("seed", 42, "RNG seed (sampler + simTier streams + adversary placement)")
		outDirFlag   = flag.String("out-dir", "eval/results/E6", "output directory")
		sloMSFlag    = flag.Int("slo-ms", 2000, "fast-path SLO budget in milliseconds")
		sizesCSVFlag = flag.String("sizes-csv", "eval/results/synthea-100000/sizes.csv", "path to validated size distribution")
	)
	flag.Parse()

	if err := run(*modeFlag, *nFlag, *repsFlag, *seedFlag, *outDirFlag, *sloMSFlag, *sizesCSVFlag, defaultFractions); err != nil {
		log.Fatalf("bench-E6: %v", err)
	}
}

func run(modeStr string, nBundles, reps int, seed int64, outDir string, sloMS int, sizesCSV string, fractions []float64) error {
	wallStart := time.Now().UTC()
	commit := shortCommit()
	runID := wallStart.Format("20060102-150405") + "-" + modeStr + "-" + commit

	mn, ok := modeName[modeStr]
	if !ok {
		return fmt.Errorf("bench-E6: unknown -mode %q (want uniform | tier-selective)", modeStr)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	bundles, err := SampleBundles(sizesCSV, nBundles, seed)
	if err != nil {
		return err
	}

	sloBudget := time.Duration(sloMS) * time.Millisecond

	// One simTier per tier class. Adversary placement is per-bundle/per-
	// fraction (we flip SetByzantine before each rep block), so the tier
	// instances persist across the whole run.
	hot := newSimTier("pinata-sim", tiers.TierHot, seed+1, mn.mode)
	warm := newSimTier("filebase-sim", tiers.TierWarm, seed+2, mn.mode)
	cold := newSimTier("arweave-sim", tiers.TierCold, seed+3, mn.mode)
	tierArr := [3]tiers.Client{hot, warm, cold}
	simArr := [3]*simTier{hot, warm, cold}

	// Per-bundle setup: encrypt + erasure-encode + Put across all 3 tiers.
	// See bundleState (file-scope) for the field doc.
	bundleStates := make([]bundleState, 0, nBundles)

	ctx := context.Background()
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

		// Honest plaintext Merkle root — this is the ground truth a real
		// gateway compares to. The bench mirrors that contract.
		tree, err := merkle.Build(strings.NewReader(string(plaintext)))
		if err != nil {
			return fmt.Errorf("merkle.Build bundle %d: %w", i, err)
		}
		merkleRoot := tree.Root()
		numChunks := tree.NumChunks()
		// lastChunkBytes mirrors handler.go: the size of the final
		// plaintext chunk (between 1 and ChunkSize).
		lastChunkBytes := len(plaintext) - (numChunks-1)*merkle.ChunkSize
		if lastChunkBytes <= 0 || lastChunkBytes > merkle.ChunkSize {
			return fmt.Errorf("bundle %d: lastChunkBytes %d out of range", i, lastChunkBytes)
		}

		key, err := crypto.GenerateKey()
		if err != nil {
			return fmt.Errorf("crypto.GenerateKey: %w", err)
		}
		encrypted, err := sealAll(key, plaintext)
		if err != nil {
			return fmt.Errorf("sealAll bundle %d: %w", i, err)
		}

		shards, paddedLen, err := erasure.Encode(encrypted)
		if err != nil {
			return fmt.Errorf("erasure.Encode bundle %d: %w", i, err)
		}

		bs := bundleState{
			paddedLen:   paddedLen,
			merkleRoot:  merkleRoot,
			key:         key,
			numChunks:   numChunks,
			lastChunkSz: lastChunkBytes,
		}
		for k := 0; k < 3; k++ {
			cid, err := tierArr[k].Put(ctx, shards[k])
			if err != nil {
				return fmt.Errorf("Put bundle %d shard %d: %w", i, k, err)
			}
			bs.cids[k] = cid
			bs.shardHash[k] = sha256.Sum256(shards[k])
		}
		bs.manifest = bundleManifest{
			Idx:       i,
			Filename:  b.Filename,
			SizeBytes: b.SizeBytes,
			PaddedLen: paddedLen,
			ShardSize: len(shards[0]),
		}
		bundleStates = append(bundleStates, bs)

		if (i+1)%25 == 0 || i == nBundles-1 {
			log.Printf("bench-E6 setup: %d/%d bundles ingested", i+1, nBundles)
		}
	}

	// Run each adversary fraction in turn.
	stats := make([]fractionStat, 0, len(fractions))
	for _, frac := range fractions {
		// Assign byzantine flags per replica for THIS fraction. Per
		// design: fraction names the global share, distributed per
		// mode rules (uniform vs cold-biased). The assignment is
		// reproducible from (seed, fraction).
		assignSeed := seed + int64(frac*1e6) + 7919 // arbitrary salt
		bzn := assignByzantine(len(bundleStates), frac, mn.mode, assignSeed)

		// Tally counters.
		var (
			nRet, nRetOK  int
			nPoR, nPoROK  int
			tierByzCount  = [3]int{}
			tierByzTotal  = [3]int{}
			failBreakdown = map[string]int{
				"recovery_failed": 0,
				"merkle_mismatch": 0,
				"decrypt_failed":  0,
				"recovered_clean": 0,
			}
		)

		for bi, bs := range bundleStates {
			// Apply this bundle's adversary mask. We flip the
			// per-tier flag based on whether THIS bundle's replica
			// for that tier is byzantine.
			for k := 0; k < 3; k++ {
				simArr[k].SetByzantine(bzn[bi][k])
				tierByzTotal[k]++
				if bzn[bi][k] {
					tierByzCount[k]++
				}
			}

			for r := 0; r < reps; r++ {
				// Run retrieval and PoR challenge concurrently when the
				// mode requires both. Sequential execution doubles the
				// per-rep wall-clock because the cold-tier latency is
				// paid twice (once during Recover's cold goroutine,
				// once during PoR's cold Get). With concurrent reps the
				// cold waits overlap and the bench fits the 60-min
				// budget at the §8 row-E6b configuration.
				//
				// Goroutine safety: simTier guards storeMu and rngMu;
				// the only shared state is the byzantine atomic.Bool
				// (set per-bundle, not per-rep) and the LRU rng. Both
				// are concurrency-safe.
				type retOutcome struct {
					ok     bool
					reason string
				}
				retCh := make(chan retOutcome, 1)
				porCh := make(chan bool, 1)

				go func() {
					rctx := withCallPurpose(ctx, callPurposeRetrieval)
					ok, reason := doRetrieval(rctx, tierArr, bs, sloBudget)
					retCh <- retOutcome{ok: ok, reason: reason}
				}()
				if mn.mode == byzantineTierSelective {
					go func() {
						pctx := withCallPurpose(ctx, callPurposePoRChallenge)
						porCh <- doPoRChallenge(pctx, simArr, bs)
					}()
				}

				ro := <-retCh
				nRet++
				if ro.ok {
					nRetOK++
					failBreakdown["recovered_clean"]++
				} else {
					failBreakdown[ro.reason]++
				}
				if mn.mode == byzantineTierSelective {
					porOK := <-porCh
					nPoR++
					if porOK {
						nPoROK++
					}
				}
			}
		}

		// Reset all tiers to honest before the next fraction so
		// per-fraction setup remains independent.
		for k := 0; k < 3; k++ {
			simArr[k].SetByzantine(false)
		}

		fs := fractionStat{
			AdversaryFraction: frac,
			Mode:              mn.label,
			NRetrievals:       nRet,
			NRetrievalSuccess: nRetOK,
			TierBreakdown:     map[string]float64{},
			FailureBreakdown:  failBreakdown,
		}
		if nRet > 0 {
			fs.RetrievalSuccessRate = float64(nRetOK) / float64(nRet)
		}
		tierLabels := [3]string{"hot", "warm", "cold"}
		for k, lbl := range tierLabels {
			if tierByzTotal[k] > 0 {
				fs.TierBreakdown[lbl] = float64(tierByzCount[k]) / float64(tierByzTotal[k])
			}
		}
		if mn.mode == byzantineTierSelective {
			fs.NPoRChallenges = nPoR
			fs.NPoRSuccess = nPoROK
			if nPoR > 0 {
				fs.PoRSuccessRate = float64(nPoROK) / float64(nPoR)
			}
			fs.DetectionGap = fs.PoRSuccessRate - fs.RetrievalSuccessRate
		}
		stats = append(stats, fs)

		log.Printf(
			"bench-E6: f=%.2f retrieval_success=%.3f por_success=%.3f gap=%.3f tier_byz=hot=%.2f warm=%.2f cold=%.2f",
			frac, fs.RetrievalSuccessRate, fs.PoRSuccessRate, fs.DetectionGap,
			fs.TierBreakdown["hot"], fs.TierBreakdown["warm"], fs.TierBreakdown["cold"],
		)
	}

	wallEnd := time.Now().UTC()

	dists := map[string]distSpec{
		"hot":  distSpecFor(tiers.TierHot),
		"warm": distSpecFor(tiers.TierWarm),
		"cold": distSpecFor(tiers.TierCold),
	}
	manifests := make([]bundleManifest, 0, len(bundleStates))
	for _, bs := range bundleStates {
		manifests = append(manifests, bs.manifest)
	}
	rec := runRecord{
		SchemaVersion: 1,
		RunID:         runID,
		StartedAt:     wallStart.Format(time.RFC3339Nano),
		CompletedAt:   wallEnd.Format(time.RFC3339Nano),
		Config: runConfig{
			Mode:              mn.label,
			NumBundles:        nBundles,
			RepsPerFraction:   reps,
			Seed:              seed,
			SLOBudgetMS:       sloMS,
			Fractions:         fractions,
			TierDistributions: dists,
			SamplingPolicy:    "uniform-random over eval/results/synthea-100000/sizes.csv (proportional to measured 100K distribution)",
			ColdTierMechanism: "in-process simTier (lognormal); Toxiproxy + live tiers deferred per docs/eval-protocol.md §3",
		},
		Fractions: stats,
		Bundles:   manifests,
	}

	runJSONPath := filepath.Join(outDir, "run-"+runID+".json")
	if err := writeJSON(runJSONPath, rec); err != nil {
		return err
	}
	log.Printf("bench-E6: wrote %s", runJSONPath)

	envPath := filepath.Join(outDir, "env-"+mn.label+".json")
	if err := writeJSON(envPath, envJSON{
		CommitHash:        commit,
		GoVersion:         runtime.Version(),
		OS:                runtime.GOOS,
		Kernel:            unameOrEmpty(),
		CPU:               cpuOrEmpty(),
		NetworkPath:       "in-process simTier (no network egress)",
		WallStart:         wallStart.Format(time.RFC3339Nano),
		WallEnd:           wallEnd.Format(time.RFC3339Nano),
		BenchID:           "E6",
		Mode:              mn.label,
		SamplingPolicy:    rec.Config.SamplingPolicy,
		ColdTierMechanism: rec.Config.ColdTierMechanism,
		TierDistributions: dists,
	}); err != nil {
		return err
	}
	log.Printf("bench-E6: wrote %s", envPath)

	printSummary(rec, wallEnd.Sub(wallStart))
	return nil
}

// bundleState holds the per-bundle fixture the bench needs across
// fractions: CIDs to retrieve, the honest per-shard SHA so the PoR
// challenge has a known-good comparator, and the AES key + Merkle root
// so the retrieval verifier can mirror cmd/gateway/handler.go's contract.
type bundleState struct {
	manifest    bundleManifest
	cids        [3]string
	shardHash   [3][32]byte // SHA-256 of honest shard bytes; PoR reference
	paddedLen   int
	merkleRoot  [32]byte // honest plaintext Merkle root; retrieval-success ground truth
	key         [32]byte // kept so we can decrypt at retrieval verify time
	numChunks   int
	lastChunkSz int
}

// doRetrieval issues one Recover, then mirrors cmd/gateway/handler.go's
// post-Recover gates: AES-GCM decrypt (catches per-chunk corruption) and
// Merkle re-derivation against the registry root. Either failure means
// the recovered bytes diverge from the honest reference — i.e. the
// adversary corrupted a shard.
//
// Returns (success, reason) where reason is one of:
//
//	"recovered_clean"  - decrypt + Merkle both pass (the success case).
//	"recovery_failed"  - gateway.Recover errored (e.g. RS decode parity
//	                     mismatch when 2 corrupted shards land on the
//	                     slow path together).
//	"decrypt_failed"   - Recover succeeded but AES-GCM auth tag failed —
//	                     the byzantine tier returned bytes that no longer
//	                     decrypt under the bundle key. Most uniform-mode
//	                     fast-path corruptions land here.
//	"merkle_mismatch"  - decrypt succeeded but the re-derived Merkle root
//	                     differs from the honest reference. Possible when
//	                     RS Decode reconstructs from a corrupted shard
//	                     and somehow produces auth-passing bytes; rare in
//	                     practice but the gateway's last-line check.
func doRetrieval(ctx context.Context, ts [3]tiers.Client, bs bundleState, sloBudget time.Duration) (bool, string) {
	// Use the same per-request deadline shape the gateway uses.
	rctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	encrypted, _, err := gateway.Recover(rctx, ts, bs.cids, bs.paddedLen, sloBudget)
	if err != nil {
		return false, "recovery_failed"
	}

	// Decrypt and rebuild Merkle root — same gate the production gateway
	// applies in handler.go. AES-GCM auth catches a corrupted ciphertext
	// because the per-chunk tag fails; if decrypt succeeds but the
	// re-derived root differs, that is the breach signal.
	plaintext, derr := decryptBundle(bs.key, encrypted, bs.numChunks, bs.lastChunkSz, bs.paddedLen)
	if derr != nil {
		return false, "decrypt_failed"
	}
	tree, terr := merkle.Build(strings.NewReader(string(plaintext)))
	if terr != nil {
		return false, "merkle_mismatch"
	}
	if root := tree.Root(); root != bs.merkleRoot {
		return false, "merkle_mismatch"
	}
	return true, "recovered_clean"
}

// doPoRChallenge models a sampled PoR audit per CLAUDE.md §4.4: each
// replica is asked to return its shard bytes; we hash and compare to the
// recorded honest shard hash. In tier-selective mode the byzantine
// replicas serve honest bytes here, so PoR success ≈ 100% even when
// retrievals are degraded.
//
// Returns true iff ALL three shards verify. A real PoR audit samples one
// shard per challenge round; we audit all three per call so the bench's
// per-fraction sample size matches retrievals 1:1 — the figure plots
// rates, not counts, so this is a fair comparison.
//
// The three shard fetches are issued in parallel (mirrors how a real
// auditor would batch challenges) so wall-clock per challenge is the
// max-of-three latencies, not the sum. Without this the bench wall time
// is dominated by sequential cold-tier waits and overshoots the 60-min
// budget at the §8 row-E6b configuration.
func doPoRChallenge(ctx context.Context, sims [3]*simTier, bs bundleState) bool {
	pctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	type result struct {
		ok  bool
		idx int
	}
	results := make(chan result, 3)
	for k := 0; k < 3; k++ {
		k := k
		go func() {
			got, err := sims[k].Get(pctx, bs.cids[k])
			if err != nil {
				results <- result{ok: false, idx: k}
				return
			}
			h := sha256.Sum256(got)
			results <- result{ok: h == bs.shardHash[k], idx: k}
		}()
	}
	allOK := true
	for i := 0; i < 3; i++ {
		r := <-results
		if !r.ok {
			allOK = false
			// Don't early-return: we want all three goroutines to
			// finish so they don't leak across rep boundaries. The
			// 30s ctx still bounds total wall.
		}
	}
	return allOK
}

// assignByzantine produces a [N][3]bool mask: bzn[i][k] == true iff
// bundle i's replica on tier k is byzantine. Total byzantine count
// targets fraction × N × 3 (rounded), then is distributed per mode:
//
//   - uniform: every replica is independently Bernoulli(fraction). At
//     N*3=300 replicas the realised fraction tracks the target ±2σ.
//   - tier-selective: byzantine replicas are biased to cold first, then
//     warm, then hot — modelling §5's metadata-correlated adversary
//     where the cold tier (longest-lived, lowest-velocity) is the
//     cheapest audit-evading target.
//
// Determinism: the same (n, fraction, mode, seed) always yields the
// same mask, so the JSON figures are reproducible.
func assignByzantine(n int, fraction float64, mode byzantineMode, seed int64) [][3]bool {
	bzn := make([][3]bool, n)
	if fraction <= 0 || n == 0 {
		return bzn
	}

	switch mode {
	case byzantineUniform:
		// Independent Bernoulli per replica. Use a deterministic LCG
		// driven from seed so we don't drag math/rand global state.
		s := uint64(seed) | 1
		for i := 0; i < n; i++ {
			for k := 0; k < 3; k++ {
				s = s*6364136223846793005 + 1442695040888963407
				// Take 32 bits of state, scale to [0,1).
				u := float64((s>>32)&0xFFFFFFFF) / float64(1<<32)
				if u < fraction {
					bzn[i][k] = true
				}
			}
		}
		return bzn

	case byzantineTierSelective:
		// Total budget = fraction of all replicas. Allocate cold
		// first; once cold is saturated, warm; then hot. This makes
		// the §5 metadata-correlated adversary explicit in the JSON
		// (tier_breakdown.cold > tier_breakdown.warm > tier_breakdown.hot).
		total := int(float64(n*3)*fraction + 0.5)
		// Cap per tier at n.
		coldCount := total
		if coldCount > n {
			coldCount = n
		}
		warmCount := total - coldCount
		if warmCount > n {
			warmCount = n
		}
		hotCount := total - coldCount - warmCount
		if hotCount > n {
			hotCount = n
		}

		// Pick which bundle indices receive byzantine replicas at
		// each tier — pseudo-random shuffle per tier so the
		// assignment doesn't always hit the first bundles.
		pickIdx := func(tierSeed int64, count int) []int {
			s := uint64(tierSeed) | 1
			perm := make([]int, n)
			for i := range perm {
				perm[i] = i
			}
			// Fisher-Yates with the LCG.
			for i := n - 1; i > 0; i-- {
				s = s*6364136223846793005 + 1442695040888963407
				j := int((s >> 32) % uint64(i+1))
				perm[i], perm[j] = perm[j], perm[i]
			}
			return perm[:count]
		}
		for _, i := range pickIdx(seed+1, coldCount) {
			bzn[i][2] = true
		}
		for _, i := range pickIdx(seed+2, warmCount) {
			bzn[i][1] = true
		}
		for _, i := range pickIdx(seed+3, hotCount) {
			bzn[i][0] = true
		}
		return bzn

	default:
		return bzn
	}
}

// sealAll seals plaintext in 16-KiB blocks and concatenates the sealed
// outputs — same shape as cmd/gateway sealAll and cmd/bench/e9 sealAll.
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

// decryptBundle mirrors cmd/gateway/handler.go decryptBundle: walks the
// sealed wire format chunk-by-chunk and reassembles the plaintext.
const sealOverhead = crypto.NonceSize + 16
const fullSealedChunkSize = merkle.ChunkSize + sealOverhead

func decryptBundle(key [32]byte, encrypted []byte, numChunks, lastChunkBytes, paddedLen int) ([]byte, error) {
	if numChunks <= 0 {
		return nil, fmt.Errorf("decrypt: numChunks must be > 0, got %d", numChunks)
	}
	if lastChunkBytes <= 0 || lastChunkBytes > merkle.ChunkSize {
		return nil, fmt.Errorf("decrypt: lastChunkBytes %d out of range (0,%d]", lastChunkBytes, merkle.ChunkSize)
	}
	expectedLen := (numChunks-1)*fullSealedChunkSize + lastChunkBytes + sealOverhead
	if expectedLen != paddedLen {
		return nil, fmt.Errorf("decrypt: layout mismatch (expected %d, got %d)", expectedLen, paddedLen)
	}
	if len(encrypted) < expectedLen {
		return nil, fmt.Errorf("decrypt: encrypted shorter than expected (%d < %d)", len(encrypted), expectedLen)
	}
	enc := encrypted[:expectedLen]
	out := make([]byte, 0, (numChunks-1)*merkle.ChunkSize+lastChunkBytes)
	off := 0
	for i := 0; i < numChunks; i++ {
		var sealedSize int
		if i == numChunks-1 {
			sealedSize = lastChunkBytes + sealOverhead
		} else {
			sealedSize = fullSealedChunkSize
		}
		pt, err := crypto.OpenChunk(key, enc[off:off+sealedSize])
		if err != nil {
			return nil, fmt.Errorf("decrypt: chunk %d: %w", i, err)
		}
		out = append(out, pt...)
		off += sealedSize
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

func printSummary(rec runRecord, wall time.Duration) {
	fmt.Printf("\nbench-E6 (%s mode) — %d bundles × %d reps × %d fractions; wall=%s\n",
		rec.Config.Mode, rec.Config.NumBundles, rec.Config.RepsPerFraction,
		len(rec.Config.Fractions), wall.Round(time.Millisecond),
	)
	fmt.Printf("  %-10s  %-10s  %-10s  %-10s  %-30s\n",
		"fraction", "retrieval", "por", "gap", "tier_byz (hot/warm/cold)",
	)
	for _, f := range rec.Fractions {
		fmt.Printf("  %-10.2f  %-10.3f  %-10.3f  %-10.3f  %-30s\n",
			f.AdversaryFraction, f.RetrievalSuccessRate,
			f.PoRSuccessRate, f.DetectionGap,
			fmt.Sprintf("%.2f / %.2f / %.2f",
				f.TierBreakdown["hot"], f.TierBreakdown["warm"], f.TierBreakdown["cold"]),
		)
	}
}
