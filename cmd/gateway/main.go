// Command lbvr-gateway is the LBVR-Med L3 retrieval gateway (CLAUDE.md
// §4.1, §4.3). It serves GET /bundle/{bundleID} by parallel-fetching the
// three RS(2,3) shards across the hot/warm/cold tiers, optionally erasure
// -decoding, decrypting, and re-verifying against the on-chain Merkle
// root.
//
// Subcommands:
//
//	serve     Start the HTTP server (default).
//	version   Print build metadata.
//
// stdlib `flag` is used (matching cmd/client) because the sandboxed build
// environment cannot fetch new module dependencies. CLAUDE.md §7 lists
// cobra/zap as the canonical choices and the surface here is small enough
// that swapping is mechanical.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"encoding/hex"

	"github.com/isongjosiah/lbvr-med/internal/config"
	"github.com/isongjosiah/lbvr-med/internal/provenance"
	"github.com/isongjosiah/lbvr-med/internal/registry"
	"github.com/isongjosiah/lbvr-med/internal/tiers"
	"github.com/isongjosiah/lbvr-med/internal/tiers/arweave"
	"github.com/isongjosiah/lbvr-med/internal/tiers/filebase"
	"github.com/isongjosiah/lbvr-med/internal/tiers/pinata"
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(runServe(nil))
	}
	switch os.Args[1] {
	case "serve":
		os.Exit(runServe(os.Args[2:]))
	case "version", "-v", "--version":
		printVersion()
		os.Exit(0)
	case "help", "-h", "--help":
		printRootUsage()
		os.Exit(0)
	default:
		// Unrecognised first arg: treat as serve flag for ergonomics
		// (e.g. `lbvr-gateway --addr :9000`).
		if isFlagLike(os.Args[1]) {
			os.Exit(runServe(os.Args[1:]))
		}
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", os.Args[1])
		printRootUsage()
		os.Exit(2)
	}
}

func isFlagLike(s string) bool {
	return len(s) > 0 && s[0] == '-'
}

func printRootUsage() {
	fmt.Fprint(os.Stderr, `lbvr-gateway — LBVR-Med retrieval gateway

Usage:
  lbvr-gateway serve [flags]
  lbvr-gateway version

Subcommands:
  serve     Start the HTTP server (default).
  version   Print build version and commit.

Run 'lbvr-gateway serve -h' for serve flags.
`)
}

func printVersion() {
	v, commit, builtAt := buildInfo()
	fmt.Printf("lbvr-gateway %s (commit %s, built %s, go %s)\n",
		v, commit, builtAt, runtime.Version())
}

// buildInfo extracts version + VCS hash from runtime/debug.ReadBuildInfo.
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

func runServe(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		addr          string
		envPath       string
		sloMs         int
		readTimeoutMs int
		jsonLogs      bool
		manifestDir   string
	)
	fs.StringVar(&addr, "addr", ":8080", "listen address (host:port)")
	fs.StringVar(&envPath, "env", ".env", "path to .env file")
	fs.IntVar(&sloMs, "slo-ms", 2000, "fast-path SLO budget in milliseconds (IEC-60601-friendly)")
	fs.IntVar(&readTimeoutMs, "read-timeout-ms", 5000, "per-request total deadline (ms) — bounds parent ctx for tier Gets")
	fs.BoolVar(&jsonLogs, "json", false, "emit logs as JSON instead of text")
	fs.StringVar(&manifestDir, "manifest-dir", "",
		"directory of per-bundle sidecar JSONs; required for production, omitted = empty in-memory sidecar (degraded demo)")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	logger := newLogger(jsonLogs)

	cfg, err := config.Load(envPath)
	if err != nil {
		logger.Error("config load failed", slog.String("err", err.Error()))
		return 1
	}

	hot, warm, cold, err := buildTierClients(cfg)
	if err != nil {
		logger.Error("tier client construction failed", slog.String("err", err.Error()))
		return 1
	}

	reg, regKind, err := buildRegistry(cfg)
	if err != nil {
		logger.Error("registry construction failed", slog.String("err", err.Error()))
		return 1
	}
	logger.Info("registry ready", slog.String("kind", regKind))

	var sc Sidecar
	if manifestDir == "" {
		logger.Warn("no --manifest-dir; running with empty in-memory sidecar — every retrieval will fail with sidecar_missing until entries are pushed in-process",
			slog.String("hint", "set --manifest-dir to the ingest manifest directory"))
		sc = NewInMemorySidecar()
	} else {
		if _, statErr := os.Stat(manifestDir); statErr != nil {
			logger.Error("manifest-dir not accessible", slog.String("dir", manifestDir), slog.String("err", statErr.Error()))
			return 1
		}
		sc = NewFileSidecar(manifestDir)
		logger.Info("sidecar ready", slog.String("dir", manifestDir))
	}

	provBundle, err := buildProvenance(logger)
	if err != nil {
		logger.Error("provenance setup failed", slog.String("err", err.Error()))
		return 1
	}

	gw, err := NewGateway(GatewayOpts{
		Hot:             hot,
		Warm:            warm,
		Cold:            cold,
		Registry:        reg,
		Sidecar:         sc,
		Logger:          logger,
		SLOBudget:       time.Duration(sloMs) * time.Millisecond,
		GetDeadline:     time.Duration(readTimeoutMs) * time.Millisecond,
		SignerKeys:      provBundle.signerKeys,
		SignerDIDs:      provBundle.signerDIDs,
		GatewayAgents:   provBundle.gatewayAgents,
		Requester:       provBundle.requester,
		QuorumThreshold: provBundle.quorumThreshold,
		Anchor:          provBundle.anchor,
		AnchorContract:  provBundle.anchorContract,
	})
	if err != nil {
		logger.Error("gateway construction failed", slog.String("err", err.Error()))
		return 1
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           gw.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		// ReadTimeout / WriteTimeout intentionally omitted — the per-handler
		// context already enforces the request deadline via getDeadline.
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	installSignalHandler(ctx, cancel, logger, srv)

	logger.Info("listening",
		slog.String("addr", addr),
		slog.Int("sloMs", sloMs),
		slog.Int("readTimeoutMs", readTimeoutMs))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server failed", slog.String("err", err.Error()))
		return 1
	}
	logger.Info("server stopped cleanly")
	return 0
}

// buildTierClients constructs production tier clients. Unlike the ingest
// CLI there is no dry-run substitute — a gateway with stub tier clients
// would silently 503 every request, which is worse than failing fast.
func buildTierClients(cfg *config.Config) (tiers.Client, tiers.Client, tiers.Client, error) {
	hot, err := pinata.New(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("pinata: %w", err)
	}
	warm, err := filebase.New(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("filebase: %w", err)
	}
	cold, err := arweave.New(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("arweave: %w", err)
	}
	return hot, warm, cold, nil
}

// buildRegistry returns the chain client when configured, otherwise the
// in-memory Mock with a loud warning. Mock-mode gateway is useful for
// local smoke tests but will obviously not see any real on-chain ingests.
func buildRegistry(cfg *config.Config) (registry.Client, string, error) {
	if cfg.CIDRegistryAddress != "" && cfg.CardonaRPCURL != "" && cfg.CardonaPrivateKey != "" {
		c, err := registry.NewChain(context.Background(),
			cfg.CardonaRPCURL, cfg.CIDRegistryAddress, cfg.CardonaPrivateKey)
		if err != nil {
			return nil, "", err
		}
		return c, "chain (stub)", nil
	}
	return registry.NewMock(), "mock", nil
}

func newLogger(jsonOut bool) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if jsonOut {
		return slog.New(slog.NewJSONHandler(os.Stderr, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, opts))
}

// installSignalHandler triggers an http.Server.Shutdown with a 5s drain on
// SIGINT/SIGTERM. The drain ensures in-flight retrievals get a chance to
// finish their tier fetches rather than orphaning HTTP connections.
func installSignalHandler(ctx context.Context, cancel context.CancelFunc, logger *slog.Logger, srv *http.Server) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case s := <-sigCh:
			logger.Warn("signal received, shutting down", slog.String("signal", s.String()))
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := srv.Shutdown(shutdownCtx); err != nil {
				logger.Error("graceful shutdown failed", slog.String("err", err.Error()))
			}
			cancel()
		case <-ctx.Done():
		}
	}()
}

// provenanceBundle bundles the per-process provenance dependencies built
// at gateway startup. Conference-scope keys are ephemeral (generated
// fresh per process); production loads them from .env via configured
// GATEWAY_BLS_SK_{1,2} (TODO when those env vars are wired up).
type provenanceBundle struct {
	signerKeys      [][32]byte
	signerDIDs      []string
	gatewayAgents   []provenance.GatewayAgent
	requester       provenance.RequesterAgent
	quorumThreshold int
	anchor          AnchorClient
	anchorContract  string
}

// buildProvenance generates 2 ephemeral BLS keypairs and constructs the
// matching gateway agents + a mock anchor client. The on-chain anchor
// path (chainAnchor against AuditorLog) is wired in D12 once forge
// build + abigen have produced the contract bindings.
func buildProvenance(logger *slog.Logger) (provenanceBundle, error) {
	const numSigners = 2
	keys := make([][32]byte, numSigners)
	dids := make([]string, numSigners)
	agents := make([]provenance.GatewayAgent, numSigners)
	for i := 0; i < numSigners; i++ {
		kp, err := provenance.GenerateKey()
		if err != nil {
			return provenanceBundle{}, fmt.Errorf("buildProvenance: keygen %d: %w", i, err)
		}
		keys[i] = kp.PrivateBytes
		// DID convention: did:lbvr:gw- + first 8 hex chars of pubkey.
		dids[i] = "did:lbvr:gw-" + hex.EncodeToString(kp.PublicBytes[:4])
		agents[i] = provenance.GatewayAgent{
			ProvType:  "prov:SoftwareAgent",
			Role:      "retrieval_gateway",
			Version:   "lbvr-gateway-0.1.0",
			PublicKey: "0x" + hex.EncodeToString(kp.PublicBytes[:]),
		}
	}
	logger.Info("provenance keys generated (ephemeral)",
		slog.Int("signers", numSigners),
		slog.String("did1", dids[0]),
		slog.String("did2", dids[1]))

	return provenanceBundle{
		signerKeys:    keys,
		signerDIDs:    dids,
		gatewayAgents: agents,
		requester: provenance.RequesterAgent{
			ProvType:    "prov:Person",
			Role:        "clinician",
			Institution: "did:lbvr:hosp-default",
			AuthzPolicy: "EHDS-Art44-primary-use",
		},
		quorumThreshold: 2,
		anchor:          newMockAnchor(),
		anchorContract:  "0xMockAuditorLog0000000000000000000000000000",
	}, nil
}
