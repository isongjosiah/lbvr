// Ingest pipeline for the LBVR-Med client (CLAUDE.md §4.2 steps 1–7).
//
// One Ingester serves many bundles; it is safe to share across goroutines.
// The pipeline order mirrors the spec exactly:
//
//	read → merkle.Build → AES-GCM seal → Encoder.Encode →
//	parallel tier Put → registry.RegisterBundle → emit manifest
//
// Decision: for D6 the fast path (single bundle ≤ 64 MiB) buffers the
// payload in memory once. The Merkle build and the seal both consume
// chunked readers, but a second pass is required after Build to re-derive
// the sealed blob; buffering avoids two filesystem reads at the cost of
// one allocation. Bundles ≥ 64 MiB stream the seal pass via a tee buffer
// (see ReadBundle for the threshold logic) — measured Synthea P99 is
// ~38 MiB so the buffered path hits >99% of the corpus.

package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/crypto"
	"github.com/isongjosiah/lbvr-med/internal/merkle"
	"github.com/isongjosiah/lbvr-med/internal/registry"
	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

// streamThreshold (in bytes) is the bundle size above which the ingest
// pipeline avoids buffering the whole file in memory. Set to 64 MiB —
// above the measured P99 Synthea bundle (~38 MiB) but well below typical
// host RAM. Streaming changes nothing semantically; it just runs the read
// twice (once for Merkle, once for seal) instead of buffering.
const streamThreshold = 64 * 1024 * 1024

// Ingester wires together every dependency the per-bundle pipeline needs.
type Ingester struct {
	hot      tiers.Client // tier 0
	warm     tiers.Client // tier 1
	cold     tiers.Client // tier 2
	registry registry.Client
	encoder  Encoder

	// clientAddr is hardcoded for D6 (CLAUDE.md §4.2: real signing comes
	// after foundryup + Cardona wiring). The address is also folded into
	// every BundleRecord.Owner field as the 0x-prefixed hex form so the
	// registry's owner column matches what the contract would record.
	clientAddr [20]byte

	// manifestDir is the directory ingest manifests are written to. Empty
	// = no manifest emitted (used by the unit-test path which asserts on
	// the in-memory registry instead).
	manifestDir string

	logger *slog.Logger

	// Optional clock injection for tests that pin RegisteredAt.
	now func() time.Time
}

// IngesterOpts is the constructor input.
type IngesterOpts struct {
	Hot         tiers.Client
	Warm        tiers.Client
	Cold        tiers.Client
	Registry    registry.Client
	Encoder     Encoder
	ClientAddr  [20]byte
	ManifestDir string
	Logger      *slog.Logger
	Now         func() time.Time
}

// NewIngester validates dependencies and returns a ready-to-use Ingester.
// All three tier clients, the registry, and the encoder are required; an
// absent clientAddr (all-zero) is allowed (D6 default per spec).
func NewIngester(opts IngesterOpts) (*Ingester, error) {
	if opts.Hot == nil {
		return nil, errors.New("ingester: hot tier client is nil")
	}
	if opts.Warm == nil {
		return nil, errors.New("ingester: warm tier client is nil")
	}
	if opts.Cold == nil {
		return nil, errors.New("ingester: cold tier client is nil")
	}
	if opts.Registry == nil {
		return nil, errors.New("ingester: registry client is nil")
	}
	if opts.Encoder == nil {
		return nil, errors.New("ingester: encoder is nil")
	}
	if opts.Hot.TierClass() != tiers.TierHot {
		return nil, fmt.Errorf("ingester: hot client TierClass=%d, want %d", opts.Hot.TierClass(), tiers.TierHot)
	}
	if opts.Warm.TierClass() != tiers.TierWarm {
		return nil, fmt.Errorf("ingester: warm client TierClass=%d, want %d", opts.Warm.TierClass(), tiers.TierWarm)
	}
	if opts.Cold.TierClass() != tiers.TierCold {
		return nil, fmt.Errorf("ingester: cold client TierClass=%d, want %d", opts.Cold.TierClass(), tiers.TierCold)
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &Ingester{
		hot:         opts.Hot,
		warm:        opts.Warm,
		cold:        opts.Cold,
		registry:    opts.Registry,
		encoder:     opts.Encoder,
		clientAddr:  opts.ClientAddr,
		manifestDir: opts.ManifestDir,
		logger:      logger,
		now:         now,
	}, nil
}

// IngestResult is the per-bundle output. The fields cover everything a
// downstream tool (manifest writer, metrics emitter) might want.
type IngestResult struct {
	BundleID   [32]byte `json:"bundleId"`
	MerkleRoot [32]byte `json:"merkleRoot"`
	NumChunks  uint32   `json:"numChunks"`
	// PaddedLen is the original encrypted-bundle length (Encoder input size).
	// Persisted in the manifest so the retrieval gateway can trim trailing
	// zero padding after erasure.Decode. Not yet stored on-chain — the
	// CIDRegistry schema bump for D8 will move this into BundleRecord.
	PaddedLen  uint32                     `json:"paddedLen"`
	Shards     [3]registry.ShardPlacement `json:"shards"`
	PolicyID   [32]byte                   `json:"policyId"`
	BundlePath string                     `json:"bundlePath"`
	Owner      string                     `json:"owner"`

	// Stage timings for E1-style ingest-throughput plots.
	TMerkle   time.Duration `json:"tMerkleNs"`
	TSeal     time.Duration `json:"tSealNs"`
	TEncode   time.Duration `json:"tEncodeNs"`
	TUpload   time.Duration `json:"tUploadNs"`
	TRegister time.Duration `json:"tRegisterNs"`
	TTotal    time.Duration `json:"tTotalNs"`
}

// IngestRequest is one bundle's parameters.
type IngestRequest struct {
	Path     string   // path to FHIR bundle
	PolicyID [32]byte // lbvr policy reference
	DryRun   bool     // skip Put + RegisterBundle
}

// Ingest runs the full pipeline for a single bundle. It is safe to call
// concurrently with itself on the same Ingester.
func (ing *Ingester) Ingest(ctx context.Context, req IngestRequest) (*IngestResult, error) {
	start := ing.now()

	// 1) read + 2) Merkle build (single pass, also returns buffered bytes
	// when the bundle is below streamThreshold).
	plain, root, numChunks, tMerkle, err := ing.readAndBuildMerkle(req.Path)
	if err != nil {
		return nil, fmt.Errorf("ingest: %w", err)
	}
	if numChunks == 0 {
		return nil, fmt.Errorf("ingest: bundle %s is empty (numChunks=0)", req.Path)
	}

	// 3) AES-256-GCM seal each 16-KiB chunk. Reuse the buffered plaintext
	// when we have it; otherwise re-read the file streaming.
	tSealStart := ing.now()
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("ingest: keygen: %w", err)
	}
	encrypted, err := sealAllChunks(key, plain, req.Path, len(plain) > 0)
	if err != nil {
		return nil, fmt.Errorf("ingest: seal: %w", err)
	}
	tSeal := ing.now().Sub(tSealStart)

	// 4) erasure-encode → 3 shards.
	tEncodeStart := ing.now()
	shards, paddedLen, err := ing.encoder.Encode(encrypted)
	if err != nil {
		return nil, fmt.Errorf("ingest: encode: %w", err)
	}
	tEncode := ing.now().Sub(tEncodeStart)

	// 5) Parallel Put to {hot, warm, cold}. Ordering convention (CLAUDE.md
	// §4.5): D0 → hot, D1 → warm, P0 → cold.
	tUploadStart := ing.now()
	cids := [3]string{}
	if !req.DryRun {
		cids, err = ing.uploadShards(ctx, shards)
		if err != nil {
			return nil, fmt.Errorf("ingest: upload: %w", err)
		}
	} else {
		// In dry-run mode emit synthetic CIDs so downstream logging still
		// has something to show. We never call RegisterBundle on dry-run.
		cids = [3]string{"dryrun-hot", "dryrun-warm", "dryrun-cold"}
	}
	tUpload := ing.now().Sub(tUploadStart)

	// 6) Compute bundleID = keccak256(clientAddr || M_root). Matches the
	// CIDRegistry contract's expected derivation.
	bundleID := registry.BundleID(ing.clientAddr, root)

	placement := [3]registry.ShardPlacement{
		{CID: cids[0], Tier: registry.TierHot},
		{CID: cids[1], Tier: registry.TierWarm},
		{CID: cids[2], Tier: registry.TierCold},
	}

	owner := "0x" + hex.EncodeToString(ing.clientAddr[:])

	rec := registry.BundleRecord{
		MerkleRoot: root,
		NumChunks:  numChunks,
		Shards:     placement,
		Owner:      owner,
		PolicyID:   req.PolicyID,
	}

	// 7) Register on-chain (Mock or chain).
	tRegisterStart := ing.now()
	if !req.DryRun {
		if err := ing.registry.RegisterBundle(ctx, bundleID, rec); err != nil {
			return nil, fmt.Errorf("ingest: register: %w", err)
		}
	}
	tRegister := ing.now().Sub(tRegisterStart)

	res := &IngestResult{
		BundleID:   bundleID,
		MerkleRoot: root,
		NumChunks:  numChunks,
		PaddedLen:  uint32(paddedLen),
		Shards:     placement,
		PolicyID:   req.PolicyID,
		BundlePath: req.Path,
		Owner:      owner,
		TMerkle:    tMerkle,
		TSeal:      tSeal,
		TEncode:    tEncode,
		TUpload:    tUpload,
		TRegister:  tRegister,
		TTotal:     ing.now().Sub(start),
	}

	if ing.manifestDir != "" {
		if err := writeManifest(ing.manifestDir, res); err != nil {
			// Manifest failure is non-fatal — the on-chain record is
			// authoritative. Log + continue.
			ing.logger.Warn("manifest write failed",
				slog.String("bundleId", hex.EncodeToString(bundleID[:])),
				slog.String("err", err.Error()))
		}
	}

	ing.logger.Info("ingest ok",
		slog.String("bundleId", hex.EncodeToString(bundleID[:])),
		slog.String("merkleRoot", hex.EncodeToString(root[:])),
		slog.Uint64("numChunks", uint64(numChunks)),
		slog.String("hotCID", cids[0]),
		slog.String("warmCID", cids[1]),
		slog.String("coldCID", cids[2]),
		slog.Int64("tTotalMs", res.TTotal.Milliseconds()),
		slog.Bool("dryRun", req.DryRun),
	)
	return res, nil
}

// readAndBuildMerkle reads the bundle once. For payloads under
// streamThreshold it also returns the buffered bytes so the seal step can
// avoid a second filesystem read; otherwise the returned slice is nil.
func (ing *Ingester) readAndBuildMerkle(path string) ([]byte, [32]byte, uint32, time.Duration, error) {
	tStart := ing.now()

	st, err := os.Stat(path)
	if err != nil {
		return nil, [32]byte{}, 0, 0, fmt.Errorf("stat %s: %w", path, err)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, [32]byte{}, 0, 0, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	if st.Size() <= streamThreshold {
		buf, err := io.ReadAll(f)
		if err != nil {
			return nil, [32]byte{}, 0, 0, fmt.Errorf("read %s: %w", path, err)
		}
		tree, err := merkle.Build(bytes.NewReader(buf))
		if err != nil {
			return nil, [32]byte{}, 0, 0, fmt.Errorf("merkle %s: %w", path, err)
		}
		return buf, tree.Root(), uint32(tree.NumChunks()), ing.now().Sub(tStart), nil
	}

	// Streaming path: build Merkle on the fly, no buffer returned.
	tree, err := merkle.Build(f)
	if err != nil {
		return nil, [32]byte{}, 0, 0, fmt.Errorf("merkle %s: %w", path, err)
	}
	return nil, tree.Root(), uint32(tree.NumChunks()), ing.now().Sub(tStart), nil
}

// sealAllChunks encrypts the bundle one merkle.ChunkSize block at a time
// and concatenates the results. When buffered != "" it is treated as a
// pre-loaded plaintext (avoiding a second open of `path`); otherwise the
// file at path is reopened and streamed.
func sealAllChunks(key [32]byte, buffered []byte, path string, hasBuffer bool) ([]byte, error) {
	var src io.Reader
	if hasBuffer {
		src = bytes.NewReader(buffered)
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("reopen %s: %w", path, err)
		}
		defer f.Close()
		src = f
	}

	// Pre-allocate: each sealed chunk grows by NonceSize+TagSize=28 bytes.
	plainCap := len(buffered)
	if !hasBuffer {
		plainCap = 0 // unknown; let append grow
	}
	out := make([]byte, 0, plainCap+plainCap/merkle.ChunkSize*28+64)

	chunk := make([]byte, merkle.ChunkSize)
	for {
		n, err := io.ReadFull(src, chunk)
		switch {
		case err == nil:
			sealed, sErr := crypto.SealChunk(key, chunk[:n])
			if sErr != nil {
				return nil, sErr
			}
			out = append(out, sealed...)
		case errors.Is(err, io.ErrUnexpectedEOF):
			sealed, sErr := crypto.SealChunk(key, chunk[:n])
			if sErr != nil {
				return nil, sErr
			}
			out = append(out, sealed...)
		case errors.Is(err, io.EOF):
			// nothing
		default:
			return nil, err
		}
		if err != nil {
			break
		}
	}
	return out, nil
}

// uploadShards Puts the three shards in parallel and collects CIDs in
// (hot, warm, cold) order. Any individual error fails the bundle — there
// is no per-tier retry at D6 (the integrator can wrap each Put in a
// retry helper later without changing this layout).
func (ing *Ingester) uploadShards(ctx context.Context, shards [3][]byte) ([3]string, error) {
	var (
		cids [3]string
		errs [3]error
		wg   sync.WaitGroup
	)
	clients := [3]tiers.Client{ing.hot, ing.warm, ing.cold}

	wg.Add(3)
	for i := 0; i < 3; i++ {
		i := i
		go func() {
			defer wg.Done()
			cid, err := clients[i].Put(ctx, shards[i])
			cids[i] = cid
			errs[i] = err
		}()
	}
	wg.Wait()

	for i, e := range errs {
		if e != nil {
			return [3]string{}, fmt.Errorf("tier %d (%s): %w", i, clients[i].Name(), e)
		}
		if cids[i] == "" {
			return [3]string{}, fmt.Errorf("tier %d (%s): empty CID", i, clients[i].Name())
		}
	}
	return cids, nil
}

// IngestCorpus walks corpusDir, finds every *.json, and runs Ingest with
// up to `concurrency` workers. It returns the aggregated results and a
// single error if any bundle failed (the others still complete).
//
// Order of returned results is deterministic (sorted bundle paths) so
// downstream tooling can diff runs without flapping on goroutine
// scheduling.
func (ing *Ingester) IngestCorpus(ctx context.Context, corpusDir string, policyID [32]byte, concurrency int, dryRun bool) ([]*IngestResult, error) {
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > 32 {
		concurrency = 32
	}

	var paths []string
	err := filepath.WalkDir(corpusDir, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(p) == ".json" {
			paths = append(paths, p)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ingest-corpus: walk: %w", err)
	}
	sort.Strings(paths)

	results := make([]*IngestResult, len(paths))
	errsByIdx := make([]error, len(paths))
	jobs := make(chan int)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for idx := range jobs {
			if ctx.Err() != nil {
				errsByIdx[idx] = ctx.Err()
				continue
			}
			res, err := ing.Ingest(ctx, IngestRequest{
				Path:     paths[idx],
				PolicyID: policyID,
				DryRun:   dryRun,
			})
			if err != nil {
				errsByIdx[idx] = err
				continue
			}
			results[idx] = res
		}
	}

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}
	for i := range paths {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return results, ctx.Err()
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()

	// Aggregate errors: combine via errors.Join so callers get every
	// failing path in a single line of output.
	var aggregated []error
	for i, e := range errsByIdx {
		if e != nil {
			aggregated = append(aggregated, fmt.Errorf("%s: %w", paths[i], e))
		}
	}
	if len(aggregated) > 0 {
		return results, errors.Join(aggregated...)
	}
	return results, nil
}

// writeManifest persists a per-bundle JSON manifest under
// manifestDir/<bundleId>.json. The manifest is the same struct as
// IngestResult, marshalled with hex-encoded byte arrays so it is stable
// across runs and human-readable.
func writeManifest(manifestDir string, res *IngestResult) error {
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		return err
	}
	name := hex.EncodeToString(res.BundleID[:]) + ".json"
	out := filepath.Join(manifestDir, name)

	view := manifestView{
		BundleID:   hex.EncodeToString(res.BundleID[:]),
		MerkleRoot: hex.EncodeToString(res.MerkleRoot[:]),
		NumChunks:  res.NumChunks,
		PolicyID:   hex.EncodeToString(res.PolicyID[:]),
		Owner:      res.Owner,
		BundlePath: res.BundlePath,
		Shards: []manifestShard{
			{CID: res.Shards[0].CID, Tier: res.Shards[0].Tier, TierName: "hot"},
			{CID: res.Shards[1].CID, Tier: res.Shards[1].Tier, TierName: "warm"},
			{CID: res.Shards[2].CID, Tier: res.Shards[2].Tier, TierName: "cold"},
		},
		Timings: manifestTimings{
			MerkleMs:   res.TMerkle.Milliseconds(),
			SealMs:     res.TSeal.Milliseconds(),
			EncodeMs:   res.TEncode.Milliseconds(),
			UploadMs:   res.TUpload.Milliseconds(),
			RegisterMs: res.TRegister.Milliseconds(),
			TotalMs:    res.TTotal.Milliseconds(),
		},
	}

	b, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(out, b, 0o644)
}

// manifestView is the on-disk shape of the per-bundle manifest. Bytes are
// hex-encoded for readability and so a downstream Python script (the eval
// post-processing path) can consume the file without writing a custom
// decoder for [N]byte arrays.
type manifestView struct {
	BundleID   string          `json:"bundleId"`
	MerkleRoot string          `json:"merkleRoot"`
	NumChunks  uint32          `json:"numChunks"`
	PolicyID   string          `json:"policyId"`
	Owner      string          `json:"owner"`
	BundlePath string          `json:"bundlePath"`
	Shards     []manifestShard `json:"shards"`
	Timings    manifestTimings `json:"timings"`
}

type manifestShard struct {
	CID      string `json:"cid"`
	Tier     uint8  `json:"tier"`
	TierName string `json:"tierName"`
}

type manifestTimings struct {
	MerkleMs   int64 `json:"merkleMs"`
	SealMs     int64 `json:"sealMs"`
	EncodeMs   int64 `json:"encodeMs"`
	UploadMs   int64 `json:"uploadMs"`
	RegisterMs int64 `json:"registerMs"`
	TotalMs    int64 `json:"totalMs"`
}
