// Package main is the E5 PoR-cost bench harness.
//
// CLAUDE.md §4.4 / §8 row "E5". Measures two halves:
//
//	Half A — Go-side prove time (responder constructs Merkle proof + BLS sig).
//	Half B — Solidity verify-side gas (postChallenge / respondToChallenge / recordVerdict).
//
// Sweep dimension is Merkle proof depth, which is determined by bundle size
// (numChunks = ceil(size / 16 KiB), depth = ceil(log2(numChunks))). Six
// canonical depths cover the empirical Synthea distribution; see sampler.go.
//
// Run flow:
//
//  1. For each sweep point, drive the prove path N times against an in-process
//     synthesised tree. Record per-rep merkle/sign/total nanoseconds.
//  2. Once at run start, shell out to `forge test --match-contract
//     PoRVerifierGas -vv`. The Foundry test contract emits one
//     `E5_GAS,<fn>, <depth>, <gas>` line per (function, depth); we regex-parse
//     them into the run JSON.
//  3. Emit run-<id>.json + env.json to -out-dir.
//
// Realism notes:
//   - Forge runs in-memory EVM at the Cancun fork; Polygon zkEVM Cardona is
//     opcode-equivalent at the same fork, so the gas numbers match what a
//     live Cardona tx would consume within ±1%. A footnote in §V documents
//     this; once Cardona is funded we re-run a sample sweep against the live
//     contract for a parity check.
//   - Tree-build time is excluded from the per-rep prove time (the responder
//     already holds the tree; rebuild is amortised over many challenges).
//     buildTreeNS lands in the row metadata so a reviewer can verify.
//   - One BLS keypair per sweep point, reused across reps — matches deployed
//     gateway behaviour, keeps GenerateKey cost out of the per-rep timer.

package main

import (
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
	"time"
)

// runRecord is the top-level JSON shape (schema_version=1).
type runRecord struct {
	SchemaVersion int       `json:"schema_version"`
	RunID         string    `json:"run_id"`
	StartedAt     string    `json:"started_at"`
	CompletedAt   string    `json:"completed_at"`
	Config        runConfig `json:"config"`
	Rows          []row     `json:"rows"`
}

type runConfig struct {
	Reps             int    `json:"reps_per_depth"`
	Seed             int64  `json:"seed"`
	ForgeEVMVersion  string `json:"forge_evm_version"`
	ForgeMatchTest   string `json:"forge_match_test"`
	GasContract      string `json:"gas_contract"`
	GasFunctionsList string `json:"gas_functions_list"`
}

type row struct {
	Depth          int         `json:"depth"`
	NumChunks      int         `json:"num_chunks"`
	BundleBytes    int         `json:"bundle_bytes"`
	BuildTreeNS    int64       `json:"build_tree_ns"`
	ProveTimeNS    Percentiles `json:"prove_time_ns"`
	MerkleProofNS  Percentiles `json:"merkle_proof_ns"`
	SignNS         Percentiles `json:"sign_ns"`
	GasPostChallenge uint64 `json:"gas_post_challenge"`
	GasRespond       uint64 `json:"gas_respond_to_challenge"`
	GasRecordVerdict uint64 `json:"gas_record_verdict"`
}

type envJSON struct {
	CommitHash      string `json:"commit_hash"`
	GoVersion       string `json:"go_version"`
	OS              string `json:"os"`
	Kernel          string `json:"kernel"`
	CPU             string `json:"cpu"`
	NetworkPath     string `json:"network_path"`
	WallStart       string `json:"wall_start"`
	WallEnd         string `json:"wall_end"`
	BenchID         string `json:"bench_id"`
	ForgeEVMVersion string `json:"forge_evm_version"`
	ForgeBinary     string `json:"forge_binary"`
}

func main() {
	var (
		repsFlag     = flag.Int("reps", 1000, "prove-time reps per sweep point")
		seedFlag     = flag.Int64("seed", 42, "RNG seed (chunkIdx + nonce streams)")
		outDirFlag   = flag.String("out-dir", "eval/results/E5", "output directory")
		forgeBinFlag = flag.String("forge-bin", defaultForgeBin(), "path to forge binary")
		contractsFlag = flag.String("contracts-root", "contracts", "path to Foundry project root")
		matchTestFlag = flag.String("match-test", "", "optional --match-test regex (e.g., \"depth_(3|5|7)\") to subset the sweep")
	)
	flag.Parse()

	if err := run(*repsFlag, *seedFlag, *outDirFlag, *forgeBinFlag, *contractsFlag, *matchTestFlag); err != nil {
		log.Fatalf("bench-E5: %v", err)
	}
}

func run(reps int, seed int64, outDir, forgeBin, contractsRoot, matchTest string) error {
	wallStart := time.Now().UTC()
	commit := shortCommit()
	runID := wallStart.Format("20060102-150405") + "-" + commit

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	// --- Half B (gas) — runs once up front, fail fast if forge is broken.
	log.Printf("bench-E5: invoking forge test (PoRVerifierGas) ...")
	gasStart := time.Now()
	gas, err := runForgeGas(forgeBin, contractsRoot, matchTest)
	if err != nil {
		return fmt.Errorf("forge gas: %w", err)
	}
	log.Printf("bench-E5: forge done in %s; %d gas measurements across %d functions",
		time.Since(gasStart).Round(time.Millisecond),
		gasDepthCount(gas),
		len(gas.ByFn))

	// --- Half A (prove time) — per sweep point, reps reps.
	rng := mrand.New(mrand.NewSource(seed))
	rows := make([]row, 0, len(sweepPoints))

	for _, pt := range sweepPoints {
		// Honour matchTest by filtering sweep points that the gas pass
		// didn't exercise. Otherwise we'd report prove-time for a depth
		// whose gas column is zero.
		if matchTest != "" && !gasHasDepth(gas, pt.Depth) {
			log.Printf("bench-E5: skipping %s (excluded by --match-test)", pt)
			continue
		}

		log.Printf("bench-E5: prove-time pass for %s (reps=%d) ...", pt, reps)
		t0 := time.Now()
		pr, err := runProve(rng, pt, reps)
		if err != nil {
			return fmt.Errorf("prove %s: %w", pt, err)
		}
		log.Printf("bench-E5: %s done in %s (build_tree=%dms)",
			pt, time.Since(t0).Round(time.Millisecond), pr.BuildTreeNS/1e6)

		gasPost, _ := gas.Lookup("post", pt.Depth)
		gasResp, _ := gas.Lookup("respond", pt.Depth)
		gasVerd, _ := gas.Lookup("verdict", pt.Depth)

		rows = append(rows, row{
			Depth:            pt.Depth,
			NumChunks:        pt.NumChunks,
			BundleBytes:      pt.BundleSizeBytes(),
			BuildTreeNS:      pr.BuildTreeNS,
			ProveTimeNS:      percentiles(pr.TotalNS),
			MerkleProofNS:    percentiles(pr.MerkleNS),
			SignNS:           percentiles(pr.SignNS),
			GasPostChallenge: gasPost,
			GasRespond:       gasResp,
			GasRecordVerdict: gasVerd,
		})
	}

	wallEnd := time.Now().UTC()

	rec := runRecord{
		SchemaVersion: 1,
		RunID:         runID,
		StartedAt:     wallStart.Format(time.RFC3339Nano),
		CompletedAt:   wallEnd.Format(time.RFC3339Nano),
		Config: runConfig{
			Reps:             reps,
			Seed:             seed,
			ForgeEVMVersion:  "shanghai",
			ForgeMatchTest:   matchTest,
			GasContract:      "PoRVerifierGas",
			GasFunctionsList: "post,respond,verdict",
		},
		Rows: rows,
	}

	runJSONPath := filepath.Join(outDir, "run-"+runID+".json")
	if err := writeJSON(runJSONPath, rec); err != nil {
		return err
	}
	log.Printf("bench-E5: wrote %s", runJSONPath)

	envPath := filepath.Join(outDir, "env.json")
	if err := writeJSON(envPath, envJSON{
		CommitHash:      commit,
		GoVersion:       runtime.Version(),
		OS:              runtime.GOOS,
		Kernel:          unameOrEmpty(),
		CPU:             cpuOrEmpty(),
		NetworkPath:     "N/A — local sim (forge in-memory EVM + Go BLS)",
		WallStart:       wallStart.Format(time.RFC3339Nano),
		WallEnd:         wallEnd.Format(time.RFC3339Nano),
		BenchID:         "E5",
		ForgeEVMVersion: "shanghai",
		ForgeBinary:     forgeBin,
	}); err != nil {
		return err
	}
	log.Printf("bench-E5: wrote %s", envPath)

	printSummary(rec, wallEnd.Sub(wallStart))
	return nil
}

func printSummary(rec runRecord, wall time.Duration) {
	fmt.Println()
	fmt.Printf("bench-E5: %d sweep points; total wall=%s\n", len(rec.Rows), wall.Round(time.Millisecond))
	fmt.Printf("  %-6s %-10s %-12s %-12s %-12s %-12s\n", "depth", "chunks", "prove_p50_us", "prove_p99_us", "gas_respond", "gas_post")
	for _, r := range rec.Rows {
		fmt.Printf("  %-6d %-10d %-12.1f %-12.1f %-12d %-12d\n",
			r.Depth, r.NumChunks,
			float64(r.ProveTimeNS.P50)/1e3,
			float64(r.ProveTimeNS.P99)/1e3,
			r.GasRespond, r.GasPostChallenge)
	}
}

// --- helpers (mirror cmd/bench/eprov for self-containment) -------------

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

func defaultForgeBin() string {
	// Resolution order: explicit -forge-bin flag → $FOUNDRY_BIN → ~/.foundry/bin/forge
	// → bare "forge" on PATH. The bench fails fast if forge can't be found
	// (runForgeGas error) so a missing default is never silent.
	if env := os.Getenv("FOUNDRY_BIN"); env != "" {
		return filepath.Join(env, "forge")
	}
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, ".foundry", "bin", "forge")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "forge"
}

// gasDepthCount reports the total (fn, depth) pairs measured. Used for
// the run-start log line; no semantic role.
func gasDepthCount(g *GasResult) int {
	n := 0
	for _, byD := range g.ByFn {
		n += len(byD)
	}
	return n
}

// gasHasDepth returns true iff at least one (fn, depth) entry covers
// `depth`. Used to skip prove-time for sweep points the gas pass didn't
// exercise (--match-test subset case).
func gasHasDepth(g *GasResult, depth int) bool {
	for _, byD := range g.ByFn {
		if _, ok := byD[depth]; ok {
			return true
		}
	}
	return false
}
