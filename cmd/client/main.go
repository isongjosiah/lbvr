// Command lbvr-client is the LBVR-Med L1 ingest CLI (CLAUDE.md §4.1).
//
// Subcommands:
//
//	ingest    Encrypt + erasure-encode + cross-tier upload + on-chain register
//	          for one bundle or a corpus directory.
//	version   Print build metadata.
//
// The CLI surface is implemented with stdlib `flag` rather than spf13/cobra
// because the sandboxed build environment cannot fetch new module
// dependencies for D6. CLAUDE.md §7 lists cobra/zap as the canonical
// choices and the interfaces here are deliberately simple enough that
// swapping is a mechanical change once the module graph admits them.
//
// Encoder injection: by default the CLI uses replicaEncoder{} (a 3×
// replication stub). The integrator wires in the real internal/erasure
// encoder in one line — see TODO(integration) in ingest pipeline.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/config"
	"github.com/isongjosiah/lbvr-med/internal/ingest"
	"github.com/isongjosiah/lbvr-med/internal/registry"
	"github.com/isongjosiah/lbvr-med/internal/tiers"
	"github.com/isongjosiah/lbvr-med/internal/tiers/arweave"
	"github.com/isongjosiah/lbvr-med/internal/tiers/filebase"
	"github.com/isongjosiah/lbvr-med/internal/tiers/pinata"
)

// defaultClientAddr is the placeholder Ethereum address used as the
// "owner" field in BundleRecord and as the prefix in the bundleID
// derivation (CLAUDE.md §4.2 step 6 expects keccak256(clientAddr ||
// merkleRoot)). Real signing replaces this on D12+ once the Cardona key
// is provisioned.
var defaultClientAddr = [20]byte{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 1,
}

func main() {
	if len(os.Args) < 2 {
		printRootUsage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "ingest":
		os.Exit(runIngest(os.Args[2:]))
	case "version", "-v", "--version":
		printVersion()
		os.Exit(0)
	case "help", "-h", "--help":
		printRootUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", os.Args[1])
		printRootUsage()
		os.Exit(2)
	}
}

func printRootUsage() {
	fmt.Fprint(os.Stderr, `lbvr-client — LBVR-Med ingest CLI

Usage:
  lbvr-client ingest [flags]
  lbvr-client version

Subcommands:
  ingest    Ingest a single bundle or a directory of FHIR bundles.
  version   Print build version and commit.

Run 'lbvr-client ingest -h' for ingest flags.
`)
}

func printVersion() {
	v, commit, builtAt := buildInfo()
	fmt.Printf("lbvr-client %s (commit %s, built %s, go %s)\n",
		v, commit, builtAt, runtime.Version())
}

// buildInfo extracts version + VCS hash from runtime/debug.ReadBuildInfo.
// When run via `go run` the values are best-effort placeholders; the
// production binary built via `go build -trimpath` carries the real values.
func buildInfo() (version, commit, builtAt string) {
	version, commit, builtAt = "dev", "unknown", "unknown"
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		version = bi.Main.Version
	}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) >= 7 {
				commit = s.Value[:7]
			}
		case "vcs.time":
			builtAt = s.Value
		}
	}
	return
}

// runIngest parses ingest flags and executes one of three modes:
// single-bundle, corpus-dir, or dry-run inspection.
func runIngest(args []string) int {
	fs := flag.NewFlagSet("ingest", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		bundle      string
		corpusDir   string
		policy      string
		envPath     string
		manifestDir string
		runID       string
		concurrency int
		dryRun      bool
		jsonLogs    bool
		clientAddr  string
	)
	fs.StringVar(&bundle, "bundle", "", "path to a single FHIR bundle JSON")
	fs.StringVar(&corpusDir, "corpus-dir", "", "directory of FHIR bundle JSONs (recursive)")
	fs.StringVar(&policy, "policy", "lbvr://policy/clinician-read",
		"policy reference; hashed to a 32-byte policyId on-chain")
	fs.StringVar(&envPath, "env", ".env", "path to .env file")
	fs.StringVar(&manifestDir, "manifest-dir", "",
		"if set, write per-bundle manifest JSONs here; default is "+
			"eval/results/ingest-<runID>/")
	fs.StringVar(&runID, "run-id", "",
		"identifier for this run; defaults to a timestamp")
	fs.IntVar(&concurrency, "concurrency", runtime.NumCPU(),
		"corpus-dir worker count (clamped to [1,32])")
	fs.BoolVar(&dryRun, "dry-run", false,
		"plan + log only; skip Put and RegisterBundle")
	fs.BoolVar(&jsonLogs, "json", false, "emit logs as JSON instead of text")
	fs.StringVar(&clientAddr, "client-addr", "",
		"override the placeholder client address (40-hex chars, no 0x); "+
			"D6 default is the all-zero-with-trailing-1 stub")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if (bundle == "" && corpusDir == "") || (bundle != "" && corpusDir != "") {
		fmt.Fprintln(os.Stderr, "exactly one of --bundle or --corpus-dir is required")
		fs.Usage()
		return 2
	}

	logger := newLogger(jsonLogs)

	cfg, err := config.Load(envPath)
	if err != nil {
		logger.Error("config load failed", slog.String("err", err.Error()))
		return 1
	}

	addr := defaultClientAddr
	if clientAddr != "" {
		parsed, err := parseAddrHex(clientAddr)
		if err != nil {
			logger.Error("invalid client-addr", slog.String("err", err.Error()))
			return 2
		}
		addr = parsed
	}

	// Tier clients. In dry-run we still construct them so a config bug
	// surfaces — but we skip Put. If construction itself fails (e.g. no
	// Pinata JWT) the dry-run silently substitutes a no-network stub
	// because the spec says "log what would happen." Production runs MUST
	// have credentials; dry-run is for plan inspection.
	hot, warm, cold, err := buildTierClients(cfg, dryRun)
	if err != nil {
		logger.Error("tier client construction failed", slog.String("err", err.Error()))
		return 1
	}

	// Registry: prefer the chain client when CIDRegistryAddress is set
	// AND we are not in dry-run, but the chain client is currently a stub
	// (D12 wiring). In dry-run or when address is missing, we fall back
	// to the in-memory Mock so the CLI is still demoable end-to-end.
	reg, regKind, err := buildRegistry(cfg, dryRun)
	if err != nil {
		logger.Error("registry construction failed", slog.String("err", err.Error()))
		return 1
	}
	logger.Info("registry ready", slog.String("kind", regKind))

	if runID == "" {
		runID = time.Now().UTC().Format("20060102-150405")
	}
	if manifestDir == "" {
		manifestDir = "eval/results/ingest-" + runID
	}

	policyID := registry.Keccak256([]byte(policy))

	ing, err := ingest.NewIngester(ingest.IngesterOpts{
		Hot:         hot,
		Warm:        warm,
		Cold:        cold,
		Registry:    reg,
		Encoder:     erasureEncoder{}, // RS(2,3) via internal/erasure
		ClientAddr:  addr,
		ManifestDir: manifestDir,
		Logger:      logger,
	})
	if err != nil {
		logger.Error("ingester construction failed", slog.String("err", err.Error()))
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	installSignalHandler(ctx, cancel, logger)

	switch {
	case bundle != "":
		_, err := ing.Ingest(ctx, ingest.IngestRequest{
			Path:     bundle,
			PolicyID: policyID,
			DryRun:   dryRun,
		})
		if err != nil {
			logger.Error("ingest failed", slog.String("err", err.Error()))
			return 1
		}
		return 0

	case corpusDir != "":
		results, err := ing.IngestCorpus(ctx, corpusDir, policyID, concurrency, dryRun)
		ok := 0
		for _, r := range results {
			if r != nil {
				ok++
			}
		}
		logger.Info("corpus ingest done",
			slog.Int("ok", ok),
			slog.Int("total", len(results)),
			slog.String("manifestDir", manifestDir),
		)
		if err != nil {
			logger.Error("corpus had failures", slog.String("err", err.Error()))
			return 1
		}
		return 0
	}
	return 0
}

// parseAddrHex parses a 40-character hex string (with or without 0x
// prefix) into a 20-byte Ethereum address.
func parseAddrHex(s string) ([20]byte, error) {
	s = strings.TrimPrefix(strings.ToLower(s), "0x")
	if len(s) != 40 {
		return [20]byte{}, fmt.Errorf("address must be 20 bytes (40 hex chars), got %d chars", len(s))
	}
	var out [20]byte
	for i := 0; i < 20; i++ {
		hi, err := hexNib(s[2*i])
		if err != nil {
			return out, err
		}
		lo, err := hexNib(s[2*i+1])
		if err != nil {
			return out, err
		}
		out[i] = hi<<4 | lo
	}
	return out, nil
}

func hexNib(c byte) (byte, error) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', nil
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, nil
	default:
		return 0, fmt.Errorf("invalid hex char %q", c)
	}
}

// buildTierClients constructs the three tier clients. In dry-run mode any
// missing credential becomes a no-network NopClient — the CLI is still
// useful for sanity-checking the bundle fingerprints and the placement
// manifest without touching the network.
func buildTierClients(cfg *config.Config, dryRun bool) (tiers.Client, tiers.Client, tiers.Client, error) {
	hot, hotErr := pinata.New(cfg)
	warm, warmErr := filebase.New(cfg)
	cold, coldErr := arweave.New(cfg)

	if dryRun {
		// In dry-run the upload code path is not executed, but the
		// Ingester constructor still validates that each client
		// implements the right TierClass(). Provide stubs whenever the
		// real client could not be built so the constructor succeeds
		// without leaking secrets via the error path.
		var h, w, c tiers.Client
		if hotErr == nil {
			h = hot
		} else {
			h = nopClient{name: "pinata-stub", tier: tiers.TierHot}
		}
		if warmErr == nil {
			w = warm
		} else {
			w = nopClient{name: "filebase-stub", tier: tiers.TierWarm}
		}
		if coldErr == nil {
			c = cold
		} else {
			c = nopClient{name: "arweave-stub", tier: tiers.TierCold}
		}
		return h, w, c, nil
	}

	if hotErr != nil {
		return nil, nil, nil, fmt.Errorf("pinata: %w", hotErr)
	}
	if warmErr != nil {
		return nil, nil, nil, fmt.Errorf("filebase: %w", warmErr)
	}
	if coldErr != nil {
		return nil, nil, nil, fmt.Errorf("arweave: %w", coldErr)
	}
	return hot, warm, cold, nil
}

// buildRegistry returns either a chain-backed client (when configured) or
// the in-memory Mock. Returns the kind for logging.
//
// Note: the chain client is a stub until D12. We still construct it when
// the operator has set CID_REGISTRY_ADDRESS so the resulting
// ErrChainNotImplemented is the loudest possible signal that further
// wiring is needed; otherwise we silently use Mock.
func buildRegistry(cfg *config.Config, dryRun bool) (registry.Client, string, error) {
	if cfg.CIDRegistryAddress != "" && cfg.ChainRPCURL != "" && cfg.ChainPrivateKey != "" && !dryRun {
		c, err := registry.NewChain(context.Background(),
			cfg.ChainRPCURL, cfg.CIDRegistryAddress, cfg.ChainPrivateKey)
		if err != nil {
			return nil, "", err
		}
		return c, "chain (stub)", nil
	}
	return registry.NewMock(), "mock", nil
}

// nopClient is the dry-run substitute used when a tier credential is
// missing. Methods return errors; the Ingester does not call Put on
// dry-run, but we still surface a clear error if the integrator forgets
// the dry-run flag.
type nopClient struct {
	name string
	tier uint8
}

func (n nopClient) Name() string     { return n.name }
func (n nopClient) TierClass() uint8 { return n.tier }
func (n nopClient) Put(_ context.Context, _ []byte) (string, error) {
	return "", fmt.Errorf("%s: nop (dry-run-only stub)", n.name)
}
func (n nopClient) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, fmt.Errorf("%s: nop", n.name)
}
func (n nopClient) Stat(_ context.Context, _ string) (*tiers.Stat, error) {
	return nil, fmt.Errorf("%s: nop", n.name)
}
func (n nopClient) Delete(_ context.Context, _ string) error { return fmt.Errorf("%s: nop", n.name) }

// newLogger returns a slog.Logger configured for either text or JSON
// output, both writing to stderr (so stdout stays free for any future
// machine-consumable output the CLI grows).
func newLogger(jsonOut bool) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if jsonOut {
		return slog.New(slog.NewJSONHandler(os.Stderr, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, opts))
}

// installSignalHandler cancels ctx on SIGINT/SIGTERM so a partially-
// completed corpus run shuts down cleanly instead of orphaning HTTP
// connections to the tier backends.
func installSignalHandler(ctx context.Context, cancel context.CancelFunc, logger *slog.Logger) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case s := <-sigCh:
			logger.Warn("signal received, cancelling", slog.String("signal", s.String()))
			cancel()
		case <-ctx.Done():
		}
	}()
}
