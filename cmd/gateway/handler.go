// Retrieval HTTP handler — wires registry → sidecar → recovery → decrypt
// → Merkle verify into one request lifecycle (CLAUDE.md §4.3).
//
// HTTP status policy:
//   400 → malformed bundleID
//   404 → registry has no record for bundleID
//   502 → tier returned tampered data (decrypt or Merkle verify failed) —
//         this is the breach trail; structured-log every field.
//   503 → operational failure: sidecar entry missing or <2 shards available
//   200 → plaintext FHIR bundle in response body

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/crypto"
	"github.com/isongjosiah/lbvr-med/internal/merkle"
	"github.com/isongjosiah/lbvr-med/internal/registry"
	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

// sealOverhead is the AES-GCM-per-chunk size growth: 12-byte nonce +
// 16-byte tag. Matches internal/crypto.SealChunk's wire format.
const sealOverhead = crypto.NonceSize + 16

// fullSealedChunkSize is the on-wire size of a fully-populated chunk.
const fullSealedChunkSize = merkle.ChunkSize + sealOverhead

// Gateway holds the per-process retrieval dependencies.
type Gateway struct {
	tiers       [3]tiers.Client
	registry    registry.Client
	sidecar     Sidecar
	logger      *slog.Logger
	sloBudget   time.Duration
	getDeadline time.Duration // per-request total deadline (parent ctx)
}

// GatewayOpts is the constructor input.
type GatewayOpts struct {
	Hot         tiers.Client
	Warm        tiers.Client
	Cold        tiers.Client
	Registry    registry.Client
	Sidecar     Sidecar
	Logger      *slog.Logger
	SLOBudget   time.Duration
	GetDeadline time.Duration
}

// NewGateway validates dependencies and returns a ready-to-serve gateway.
func NewGateway(opts GatewayOpts) (*Gateway, error) {
	if opts.Hot == nil || opts.Warm == nil || opts.Cold == nil {
		return nil, errors.New("gateway: tier clients are required")
	}
	if opts.Registry == nil {
		return nil, errors.New("gateway: registry client is required")
	}
	if opts.Sidecar == nil {
		return nil, errors.New("gateway: sidecar is required")
	}
	if opts.SLOBudget <= 0 {
		return nil, errors.New("gateway: SLOBudget must be positive")
	}
	if opts.GetDeadline <= 0 {
		return nil, errors.New("gateway: GetDeadline must be positive")
	}
	if opts.Hot.TierClass() != tiers.TierHot ||
		opts.Warm.TierClass() != tiers.TierWarm ||
		opts.Cold.TierClass() != tiers.TierCold {
		return nil, fmt.Errorf("gateway: tier-class mismatch (hot=%d warm=%d cold=%d)",
			opts.Hot.TierClass(), opts.Warm.TierClass(), opts.Cold.TierClass())
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Gateway{
		tiers:       [3]tiers.Client{opts.Hot, opts.Warm, opts.Cold},
		registry:    opts.Registry,
		sidecar:     opts.Sidecar,
		logger:      logger,
		sloBudget:   opts.SLOBudget,
		getDeadline: opts.GetDeadline,
	}, nil
}

// Routes returns the configured ServeMux. Health is registered here too
// so main.go does not need to import this package's constants.
func (g *Gateway) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", g.serveHealth)
	mux.HandleFunc("GET /bundle/{bundleID}", g.serveBundle)
	return mux
}

func (g *Gateway) serveHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// errorBody renders a small JSON envelope so curl / fetch / clients can
// branch on status without parsing prose.
type errorBody struct {
	Error  string `json:"error"`
	Detail string `json:"detail,omitempty"`
}

func writeJSONError(w http.ResponseWriter, status int, kind, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorBody{Error: kind, Detail: detail})
}

// serveBundle is the GET /bundle/{bundleID} handler.
func (g *Gateway) serveBundle(w http.ResponseWriter, r *http.Request) {
	raw := r.PathValue("bundleID")
	bundleID, err := parseBundleID(raw)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad_bundle_id", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), g.getDeadline)
	defer cancel()

	rec, err := g.registry.GetRecord(ctx, bundleID)
	if err != nil {
		if errors.Is(err, registry.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "bundle_not_found", "registry has no record for this bundleID")
			return
		}
		g.logger.Warn("registry lookup failed",
			slog.String("bundleId", raw),
			slog.String("err", err.Error()))
		writeJSONError(w, http.StatusBadGateway, "registry_error", err.Error())
		return
	}

	entry, err := g.sidecar.Lookup(ctx, bundleID)
	if err != nil {
		if errors.Is(err, ErrSidecarMissing) {
			writeJSONError(w, http.StatusServiceUnavailable, "sidecar_missing",
				"on-chain record present but gateway sidecar entry is absent; ingest manifest may not have been replicated to this gateway")
			return
		}
		g.logger.Warn("sidecar lookup failed",
			slog.String("bundleId", raw),
			slog.String("err", err.Error()))
		writeJSONError(w, http.StatusBadGateway, "sidecar_error", err.Error())
		return
	}

	cids := [3]string{rec.Shards[0].CID, rec.Shards[1].CID, rec.Shards[2].CID}
	encrypted, stats, err := Recover(ctx, g.tiers, cids, int(entry.PaddedLen), g.sloBudget)
	if err != nil {
		g.logger.Warn("recovery failed",
			slog.String("bundleId", raw),
			slog.String("err", err.Error()),
			slog.String("shardErrs", shardErrsString(stats.ShardErrors)))
		writeJSONError(w, http.StatusServiceUnavailable, "recovery_failed", err.Error())
		return
	}

	plaintext, perr := decryptBundle(entry.Key, encrypted, int(rec.NumChunks), int(entry.LastChunkBytes), int(entry.PaddedLen))
	if perr != nil {
		// Decryption failure on a successfully-recovered ciphertext is a
		// breach signal — the shards opened by the tier were tampered.
		g.logger.Error("decrypt failed (potential breach)",
			slog.String("bundleId", raw),
			slog.String("hotCID", cids[0]),
			slog.String("warmCID", cids[1]),
			slog.String("coldCID", cids[2]),
			slog.String("err", perr.Error()))
		writeJSONError(w, http.StatusBadGateway, "decrypt_failed", perr.Error())
		return
	}

	// Re-derive Merkle root from the decrypted plaintext and compare to the
	// authority on chain. A mismatch with valid GCM authentication means
	// the registry entry, the sidecar, or the upload-time pipeline is out
	// of sync — log the breach trail and return 502.
	tree, err := merkle.Build(strings.NewReader(string(plaintext)))
	if err != nil {
		g.logger.Error("merkle rebuild failed",
			slog.String("bundleId", raw),
			slog.String("err", err.Error()))
		writeJSONError(w, http.StatusBadGateway, "merkle_rebuild_failed", err.Error())
		return
	}
	if root := tree.Root(); root != rec.MerkleRoot {
		g.logger.Error("merkle mismatch (potential breach)",
			slog.String("bundleId", raw),
			slog.String("hotCID", cids[0]),
			slog.String("warmCID", cids[1]),
			slog.String("coldCID", cids[2]),
			slog.String("expected", hex.EncodeToString(rec.MerkleRoot[:])),
			slog.String("got", hex.EncodeToString(root[:])))
		writeJSONError(w, http.StatusBadGateway, "merkle_mismatch",
			"recomputed Merkle root does not match registry record")
		return
	}

	w.Header().Set("Content-Type", "application/fhir+json")
	w.Header().Set("X-LBVR-Recovery-Mode", stats.Mode.String())
	w.Header().Set("X-LBVR-Decode-Latency-Us", strconv.FormatInt(stats.DecodeNanos/1000, 10))
	w.Header().Set("X-LBVR-Shard-Latency-Us", shardLatHeader(stats.ShardLatencies))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(plaintext)

	g.logger.Info("served bundle",
		slog.String("bundleId", raw),
		slog.String("recoveryMode", stats.Mode.String()),
		slog.Int64("decodeUs", stats.DecodeNanos/1000),
		slog.Int("plainBytes", len(plaintext)))
}

// parseBundleID accepts 64 hex chars with or without a 0x prefix. The
// strict-length check rejects ambiguous short forms before they reach the
// registry.
func parseBundleID(raw string) ([32]byte, error) {
	s := strings.TrimPrefix(strings.ToLower(raw), "0x")
	if len(s) != 64 {
		return [32]byte{}, fmt.Errorf("bundleID must be 32 bytes (64 hex chars), got %d", len(s))
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return [32]byte{}, fmt.Errorf("bundleID hex decode: %w", err)
	}
	var out [32]byte
	copy(out[:], b)
	return out, nil
}

// decryptBundle walks the encrypted bundle in (numChunks-1) full sealed
// chunks of fullSealedChunkSize each, then one final sealed chunk of
// (lastChunkBytes + sealOverhead) bytes. Concatenated plaintexts are the
// original FHIR bundle.
//
// The function validates the wire layout against (paddedLen, numChunks,
// lastChunkBytes) before attempting any decryption so a sidecar/registry
// drift is caught with a precise error rather than an opaque GCM auth
// failure deep in the loop.
func decryptBundle(key [32]byte, encrypted []byte, numChunks, lastChunkBytes, paddedLen int) ([]byte, error) {
	if numChunks <= 0 {
		return nil, fmt.Errorf("decrypt: numChunks must be > 0, got %d", numChunks)
	}
	if lastChunkBytes <= 0 || lastChunkBytes > merkle.ChunkSize {
		return nil, fmt.Errorf("decrypt: lastChunkBytes %d out of range (0,%d]", lastChunkBytes, merkle.ChunkSize)
	}
	expectedLen := (numChunks-1)*fullSealedChunkSize + lastChunkBytes + sealOverhead
	if expectedLen != paddedLen {
		return nil, fmt.Errorf("decrypt: encrypted-bundle layout inconsistent with sidecar metadata (expected %d, got %d)", expectedLen, paddedLen)
	}
	if len(encrypted) < expectedLen {
		return nil, fmt.Errorf("decrypt: encrypted shorter than expected (%d < %d)", len(encrypted), expectedLen)
	}
	// We only consume expectedLen bytes — anything trailing was zero-pad
	// from the erasure shard alignment and should already have been
	// trimmed by erasure.Decode, but we are defensive.
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

// shardLatHeader formats per-tier latencies as comma-separated
// microseconds; -1 is rendered as "-1" so a downstream metric can
// distinguish "tier never returned" from "tier returned at 0us".
func shardLatHeader(lats [3]time.Duration) string {
	parts := make([]string, 3)
	for i, l := range lats {
		if l == notReturned {
			parts[i] = "-1"
		} else {
			parts[i] = strconv.FormatInt(l.Microseconds(), 10)
		}
	}
	return strings.Join(parts, ",")
}

// shardErrsString condenses the per-shard errors for one log line.
func shardErrsString(errs [3]error) string {
	parts := make([]string, 3)
	for i, e := range errs {
		if e == nil {
			parts[i] = "ok"
		} else {
			parts[i] = e.Error()
		}
	}
	return strings.Join(parts, " | ")
}
