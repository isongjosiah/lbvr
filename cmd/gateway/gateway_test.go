package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/crypto"
	"github.com/isongjosiah/lbvr-med/internal/erasure"
	"github.com/isongjosiah/lbvr-med/internal/gateway"
	"github.com/isongjosiah/lbvr-med/internal/merkle"
	"github.com/isongjosiah/lbvr-med/internal/provenance"
	"github.com/isongjosiah/lbvr-med/internal/registry"
	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

// quietLogger discards log output so test runs stay readable.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fixture is one prepared bundle with everything the gateway expects:
// plaintext FHIR, a bundleID, the registry record, the sidecar entry,
// and the three tiers each holding the right shard.
type fixture struct {
	plaintext []byte
	bundleID  [32]byte
	root      [32]byte
	numChunks uint32
	key       [32]byte
	paddedLen uint32
	lastChunk uint32
	shards    [3][]byte
	cids      [3]string
	hot       *inMemTier
	warm      *inMemTier
	cold      *inMemTier
	reg       *registry.Mock
	sidecar   *InMemorySidecar
}

// newFixture builds a synthetic FHIR bundle, runs it through the ingest
// math (Merkle → seal → erasure → Put), and returns everything the
// gateway tests need to drive a single retrieval.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	// Build a payload that spans more than one Merkle chunk so the
	// decrypt loop iterates non-trivially.
	body := strings.Repeat(`{"r":"Observation","status":"final","code":"8302-2","val":175.4},`, 800)
	plaintext := []byte(`{"resourceType":"Bundle","entry":[` + strings.TrimSuffix(body, ",") + `]}`)
	if len(plaintext) < 2*merkle.ChunkSize+1 {
		// Force at least 3 chunks (2 full + 1 short) so lastChunk math is
		// genuinely exercised.
		pad := strings.Repeat(" ", 2*merkle.ChunkSize+1-len(plaintext))
		plaintext = append(plaintext, []byte(pad)...)
	}

	tree, err := merkle.Build(bytes.NewReader(plaintext))
	if err != nil {
		t.Fatalf("merkle build: %v", err)
	}
	root := tree.Root()
	numChunks := uint32(tree.NumChunks())
	if numChunks < 2 {
		t.Fatalf("test expects ≥2 chunks, got %d", numChunks)
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	encrypted, lastChunk := sealAll(t, key, plaintext)
	shards, paddedLen, err := erasure.Encode(encrypted)
	if err != nil {
		t.Fatalf("erasure encode: %v", err)
	}

	hot := newInMemTier("pinata", tiers.TierHot)
	warm := newInMemTier("filebase", tiers.TierWarm)
	cold := newInMemTier("arweave", tiers.TierCold)
	ctx := context.Background()
	hotCID, err := hot.Put(ctx, shards[0])
	if err != nil {
		t.Fatalf("hot put: %v", err)
	}
	warmCID, err := warm.Put(ctx, shards[1])
	if err != nil {
		t.Fatalf("warm put: %v", err)
	}
	coldCID, err := cold.Put(ctx, shards[2])
	if err != nil {
		t.Fatalf("cold put: %v", err)
	}

	bundleID := registry.BundleID([20]byte{}, root)
	reg := registry.NewMock()
	rec := registry.BundleRecord{
		MerkleRoot: root,
		NumChunks:  numChunks,
		Shards: [registry.ShardCount]registry.ShardPlacement{
			{CID: hotCID, Tier: registry.TierHot},
			{CID: warmCID, Tier: registry.TierWarm},
			{CID: coldCID, Tier: registry.TierCold},
		},
		Owner:    "0x0000000000000000000000000000000000000000",
		PolicyID: [32]byte{},
	}
	if err := reg.RegisterBundle(ctx, bundleID, rec); err != nil {
		t.Fatalf("register: %v", err)
	}

	sc := NewInMemorySidecar()
	sc.Set(bundleID, Entry{Key: key, PaddedLen: uint32(paddedLen), LastChunkBytes: lastChunk})

	return &fixture{
		plaintext: plaintext,
		bundleID:  bundleID,
		root:      root,
		numChunks: numChunks,
		key:       key,
		paddedLen: uint32(paddedLen),
		lastChunk: lastChunk,
		shards:    shards,
		cids:      [3]string{hotCID, warmCID, coldCID},
		hot:       hot,
		warm:      warm,
		cold:      cold,
		reg:       reg,
		sidecar:   sc,
	}
}

// sealAll mirrors the ingest path's sealing (loop SealChunk over 16-KiB
// plaintext blocks). Returns the concatenated ciphertext and the
// plaintext size of the trailing chunk.
func sealAll(t *testing.T, key [32]byte, plain []byte) ([]byte, uint32) {
	t.Helper()
	out := make([]byte, 0, len(plain)+(len(plain)/merkle.ChunkSize+1)*sealOverhead)
	var lastChunk uint32
	for off := 0; off < len(plain); off += merkle.ChunkSize {
		end := off + merkle.ChunkSize
		if end > len(plain) {
			end = len(plain)
		}
		chunk := plain[off:end]
		sealed, err := crypto.SealChunk(key, chunk)
		if err != nil {
			t.Fatalf("seal: %v", err)
		}
		out = append(out, sealed...)
		lastChunk = uint32(len(chunk))
	}
	return out, lastChunk
}

// newGateway wraps NewGateway with the standard test budget.
func newGateway(t *testing.T, f *fixture, slo time.Duration) *Gateway {
	t.Helper()
	gw, err := NewGateway(GatewayOpts{
		Hot:         f.hot,
		Warm:        f.warm,
		Cold:        f.cold,
		Registry:    f.reg,
		Sidecar:     f.sidecar,
		Logger:      quietLogger(),
		SLOBudget:   slo,
		GetDeadline: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewGateway: %v", err)
	}
	return gw
}

// doGet drives one retrieval through the gateway and returns the
// recorder.
func doGet(gw *Gateway, bundleID [32]byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/bundle/"+hex.EncodeToString(bundleID[:]), nil)
	rr := httptest.NewRecorder()
	gw.Routes().ServeHTTP(rr, req)
	return rr
}

func TestServeBundle_HappyPath_FastPath(t *testing.T) {
	f := newFixture(t)
	gw := newGateway(t, f, 2*time.Second)

	rr := doGet(gw, f.bundleID)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if !bytes.Equal(rr.Body.Bytes(), f.plaintext) {
		t.Fatalf("body mismatch: got %d bytes, want %d", rr.Body.Len(), len(f.plaintext))
	}
	if got := rr.Header().Get("X-LBVR-Recovery-Mode"); got != "fast" {
		t.Fatalf("recovery-mode header: got %q, want fast", got)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/fhir+json" {
		t.Fatalf("content-type: got %q", got)
	}
}

func TestServeBundle_SingleShardLoss_D0(t *testing.T) {
	f := newFixture(t)
	f.hot.failGet = func(string) error { return errors.New("404") }
	gw := newGateway(t, f, 2*time.Second)

	rr := doGet(gw, f.bundleID)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if !bytes.Equal(rr.Body.Bytes(), f.plaintext) {
		t.Fatalf("body mismatch")
	}
	if got := rr.Header().Get("X-LBVR-Recovery-Mode"); got != "slow" {
		t.Fatalf("recovery-mode header: got %q, want slow", got)
	}
}

func TestServeBundle_SingleShardLoss_D1(t *testing.T) {
	f := newFixture(t)
	f.warm.failGet = func(string) error { return errors.New("404") }
	gw := newGateway(t, f, 2*time.Second)

	rr := doGet(gw, f.bundleID)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if !bytes.Equal(rr.Body.Bytes(), f.plaintext) {
		t.Fatalf("body mismatch")
	}
	if got := rr.Header().Get("X-LBVR-Recovery-Mode"); got != "slow" {
		t.Fatalf("recovery-mode header: got %q, want slow", got)
	}
}

func TestServeBundle_SingleShardLoss_P0(t *testing.T) {
	f := newFixture(t)
	f.cold.failGet = func(string) error { return errors.New("404") }
	gw := newGateway(t, f, 2*time.Second)

	rr := doGet(gw, f.bundleID)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if !bytes.Equal(rr.Body.Bytes(), f.plaintext) {
		t.Fatalf("body mismatch")
	}
	// P0 dead but D0+D1 sufficed → still fast path.
	if got := rr.Header().Get("X-LBVR-Recovery-Mode"); got != "fast" {
		t.Fatalf("recovery-mode header: got %q, want fast (D0+D1 still met SLO)", got)
	}
}

func TestServeBundle_DoubleShardLoss_Returns503(t *testing.T) {
	f := newFixture(t)
	f.hot.failGet = func(string) error { return errors.New("404 hot") }
	f.warm.failGet = func(string) error { return errors.New("404 warm") }
	gw := newGateway(t, f, 2*time.Second)

	rr := doGet(gw, f.bundleID)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var body errorBody
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("body json: %v", err)
	}
	if body.Error != "recovery_failed" {
		t.Fatalf("error kind: got %q want recovery_failed", body.Error)
	}
}

// TestServeBundle_TamperedShard_Returns502 flips a byte in the warm
// tier's stored shard. With D0 and D1 both arriving in time the gateway
// concatenates them without erasure decode; the AES-GCM auth tag on the
// chunk that overlaps the corrupted byte fails, surfacing as a 502
// (decrypt_failed). This exercises the breach-trail path before the
// Merkle reconstruction even runs.
func TestServeBundle_TamperedShard_Returns502(t *testing.T) {
	f := newFixture(t)
	f.warm.tamper = func(b []byte) []byte {
		// Flip a byte deep in the shard so it lands inside a sealed
		// chunk, not in the trailing zero pad.
		out := append([]byte(nil), b...)
		if len(out) > 100 {
			out[100] ^= 0xFF
		}
		return out
	}
	gw := newGateway(t, f, 2*time.Second)

	rr := doGet(gw, f.bundleID)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("want 502, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var body errorBody
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("body json: %v", err)
	}
	// Either decrypt_failed (GCM auth) or merkle_mismatch is acceptable;
	// in practice the per-chunk AES-GCM tag fires first.
	if body.Error != "decrypt_failed" && body.Error != "merkle_mismatch" {
		t.Fatalf("unexpected error kind %q", body.Error)
	}
}

func TestServeBundle_SidecarMissing_Returns503(t *testing.T) {
	f := newFixture(t)
	// Replace sidecar with an empty one but keep the registry record.
	gw, err := NewGateway(GatewayOpts{
		Hot:         f.hot,
		Warm:        f.warm,
		Cold:        f.cold,
		Registry:    f.reg,
		Sidecar:     NewInMemorySidecar(),
		Logger:      quietLogger(),
		SLOBudget:   2 * time.Second,
		GetDeadline: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewGateway: %v", err)
	}

	rr := doGet(gw, f.bundleID)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", rr.Code)
	}
	var body errorBody
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("body json: %v", err)
	}
	if body.Error != "sidecar_missing" {
		t.Fatalf("error kind: got %q want sidecar_missing", body.Error)
	}
}

func TestServeBundle_RegistryMissing_Returns404(t *testing.T) {
	f := newFixture(t)
	// Use a fresh empty registry so GetRecord returns ErrNotFound.
	gw, err := NewGateway(GatewayOpts{
		Hot:         f.hot,
		Warm:        f.warm,
		Cold:        f.cold,
		Registry:    registry.NewMock(),
		Sidecar:     f.sidecar,
		Logger:      quietLogger(),
		SLOBudget:   2 * time.Second,
		GetDeadline: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewGateway: %v", err)
	}
	rr := doGet(gw, f.bundleID)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rr.Code)
	}
}

func TestServeBundle_SlowColdTier_FastPath(t *testing.T) {
	f := newFixture(t)
	// Make cold tier slower than the SLO budget; D0+D1 should win and the
	// cold-tier context should be cancelled.
	f.cold.slowGet(2 * time.Second)
	gw := newGateway(t, f, 200*time.Millisecond)

	start := time.Now()
	rr := doGet(gw, f.bundleID)
	elapsed := time.Since(start)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if elapsed > time.Second {
		t.Fatalf("fast path took too long: %v", elapsed)
	}
	if got := rr.Header().Get("X-LBVR-Recovery-Mode"); got != "fast" {
		t.Fatalf("recovery-mode: got %q want fast", got)
	}
	// Give the cancelled goroutine a moment to record cancellation.
	time.Sleep(50 * time.Millisecond)
	if !f.cold.wasCancelled() {
		t.Fatalf("cold-tier Get was not cancelled")
	}
}

func TestServeBundle_SlowWarmTier_SlowPath(t *testing.T) {
	f := newFixture(t)
	// Warm sleeps past SLO; cold returns instantly. Recovery should
	// proceed via D0+P0 reconstruction.
	f.warm.slowGet(2 * time.Second)
	gw := newGateway(t, f, 100*time.Millisecond)

	rr := doGet(gw, f.bundleID)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if !bytes.Equal(rr.Body.Bytes(), f.plaintext) {
		t.Fatalf("body mismatch")
	}
	if got := rr.Header().Get("X-LBVR-Recovery-Mode"); got != "slow" {
		t.Fatalf("recovery-mode: got %q want slow", got)
	}
}

func TestRecover_StatsReportLatencies(t *testing.T) {
	f := newFixture(t)
	// Force one shard to fail so ShardErrors[0] is populated; the other
	// two return successfully.
	f.hot.failGet = func(string) error { return errors.New("404 hot") }

	encrypted, stats, err := gateway.Recover(
		context.Background(),
		[3]tiers.Client{f.hot, f.warm, f.cold},
		f.cids,
		int(f.paddedLen),
		2*time.Second,
	)
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if stats.Mode != gateway.RecoverySlowPath {
		t.Fatalf("mode: got %v want slow", stats.Mode)
	}
	if stats.ShardErrors[0] == nil {
		t.Fatalf("ShardErrors[0] should be set")
	}
	// Surviving shards should have non-negative latency recorded.
	if stats.ShardLatencies[1] < 0 {
		t.Fatalf("ShardLatencies[1] should be set, got %v", stats.ShardLatencies[1])
	}
	// Decode happened.
	if stats.DecodeNanos == 0 {
		t.Fatalf("DecodeNanos should be > 0")
	}
	// And the encrypted bundle round-trips back to the seal-time bytes.
	if len(encrypted) != int(f.paddedLen) {
		t.Fatalf("encrypted length: got %d want %d", len(encrypted), f.paddedLen)
	}
}

// TestParseBundleID exercises the validator end-to-end so handler
// rejection of malformed IDs is captured.
func TestParseBundleID(t *testing.T) {
	good := strings.Repeat("ab", 32)
	if _, err := parseBundleID(good); err != nil {
		t.Fatalf("good id rejected: %v", err)
	}
	if _, err := parseBundleID("0x" + good); err != nil {
		t.Fatalf("good id with 0x rejected: %v", err)
	}
	if _, err := parseBundleID("short"); err == nil {
		t.Fatalf("short id accepted")
	}
	if _, err := parseBundleID(strings.Repeat("zz", 32)); err == nil {
		t.Fatalf("non-hex id accepted")
	}
}

func TestServeBundle_BadBundleID_Returns400(t *testing.T) {
	f := newFixture(t)
	gw := newGateway(t, f, 2*time.Second)
	req := httptest.NewRequest(http.MethodGet, "/bundle/notvalid", nil)
	rr := httptest.NewRecorder()
	gw.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestHealthz(t *testing.T) {
	f := newFixture(t)
	gw := newGateway(t, f, 2*time.Second)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	gw.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
}

// newGatewayWithProvenance wires the standard fixture with a freshly-
// generated 2-key BLS quorum, a mock anchor, and the provenance config
// that flips the gateway from retrieval-only to "emit a signed and
// anchored PROV doc per retrieval." Returns the gateway, the mockAnchor
// (so tests can read back what was anchored), and the matching
// StaticKeyResolver / signer DIDs (so tests can construct a Verifier).
func newGatewayWithProvenance(t *testing.T, f *fixture, slo time.Duration) (*Gateway, *mockAnchor, provenance.StaticKeyResolver, []string) {
	t.Helper()

	const numSigners = 2
	keys := make([][32]byte, numSigners)
	dids := make([]string, numSigners)
	agents := make([]provenance.GatewayAgent, numSigners)
	keyResolver := provenance.StaticKeyResolver{}
	for i := 0; i < numSigners; i++ {
		kp, err := provenance.GenerateKey()
		if err != nil {
			t.Fatalf("keygen %d: %v", i, err)
		}
		keys[i] = kp.PrivateBytes
		dids[i] = "did:lbvr:gw-" + hex.EncodeToString(kp.PublicBytes[:4])
		agents[i] = provenance.GatewayAgent{
			ProvType:  "prov:SoftwareAgent",
			Role:      "retrieval_gateway",
			Version:   "test",
			PublicKey: "0x" + hex.EncodeToString(kp.PublicBytes[:]),
		}
		keyResolver[dids[i]] = kp.PublicBytes
	}

	anchor := newMockAnchor()
	gw, err := NewGateway(GatewayOpts{
		Hot:           f.hot,
		Warm:          f.warm,
		Cold:          f.cold,
		Registry:      f.reg,
		Sidecar:       f.sidecar,
		Logger:        quietLogger(),
		SLOBudget:     slo,
		GetDeadline:   5 * time.Second,
		SignerKeys:    keys,
		SignerDIDs:    dids,
		GatewayAgents: agents,
		Requester: provenance.RequesterAgent{
			ProvType:    "prov:Person",
			Role:        "clinician",
			Institution: "did:lbvr:hosp-test",
			AuthzPolicy: "EHDS-Art44-test",
		},
		QuorumThreshold: 2,
		Anchor:          anchor,
		AnchorContract:  "0xMock",
	})
	if err != nil {
		t.Fatalf("NewGateway: %v", err)
	}
	return gw, anchor, keyResolver, dids
}

// truncateID returns a [32]byte where only the first 4 bytes are the
// originals (rest zero). Mirrors verifier.extractIDs which keys the
// AnchorResolver by truncated form (PROV node IDs only encode 4 bytes
// per generator.shortID).
func truncateID(full [32]byte) [32]byte {
	var out [32]byte
	copy(out[:4], full[:4])
	return out
}

func TestServeBundle_EmitsValidProvenance(t *testing.T) {
	f := newFixture(t)
	gw, anchor, keyResolver, _ := newGatewayWithProvenance(t, f, 2*time.Second)

	// 1) Drive a normal retrieval; expect 200 + retrievalID header.
	rr := doGet(gw, f.bundleID)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if !bytes.Equal(rr.Body.Bytes(), f.plaintext) {
		t.Fatal("response body != plaintext")
	}
	retrievalIDHex := rr.Header().Get("X-LBVR-Retrieval-ID")
	if retrievalIDHex == "" {
		t.Fatal("X-LBVR-Retrieval-ID header missing")
	}
	if len(retrievalIDHex) != 64 {
		t.Fatalf("retrievalID hex must be 64 chars, got %d", len(retrievalIDHex))
	}
	retrievalIDBytes, err := hex.DecodeString(retrievalIDHex)
	if err != nil {
		t.Fatalf("retrievalID hex: %v", err)
	}
	var retrievalID [32]byte
	copy(retrievalID[:], retrievalIDBytes)

	// 2) Fetch the PROV doc via the side endpoint.
	provReq := httptest.NewRequest(http.MethodGet, "/prov/"+retrievalIDHex, nil)
	provRR := httptest.NewRecorder()
	gw.Routes().ServeHTTP(provRR, provReq)
	if provRR.Code != http.StatusOK {
		t.Fatalf("prov fetch want 200, got %d (body=%s)", provRR.Code, provRR.Body.String())
	}
	if provRR.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("prov content-type = %q, want application/json", provRR.Header().Get("Content-Type"))
	}
	provBytes := provRR.Body.Bytes()
	if len(provBytes) == 0 {
		t.Fatal("prov body is empty")
	}

	// 3) Look up the anchor the gateway recorded; rekey it for the
	//    verifier (which uses truncated IDs per extractIDs).
	rec, err := anchor.Get(context.Background(), f.bundleID, retrievalID)
	if err != nil {
		t.Fatalf("anchor lookup: %v", err)
	}
	anchors := provenance.StaticAnchorResolver{}
	anchors.SetAnchor(truncateID(f.bundleID), truncateID(retrievalID), rec.ProvHash, rec.BlockNumber)

	// 4) Verify.
	v := &provenance.Verifier{Keys: keyResolver, Anchors: anchors}
	res, err := v.Verify(provBytes)
	if err != nil {
		t.Fatalf("Verify err: %v", err)
	}
	if !res.Valid {
		t.Fatalf("Verify Valid=false: %s; checks=%+v", res.FailureReason, res.SignatureChecks)
	}
	if res.AnchoredBlock != rec.BlockNumber {
		t.Fatalf("AnchoredBlock = %d, want %d", res.AnchoredBlock, rec.BlockNumber)
	}
	if len(res.SignatureChecks) < 2 {
		t.Fatalf("want ≥2 signature checks (entity + activity), got %d", len(res.SignatureChecks))
	}
	for _, ck := range res.SignatureChecks {
		if !ck.Valid {
			t.Fatalf("signature check %q failed: %s", ck.NodeID, ck.Reason)
		}
	}
}

func TestServeBundle_ProvenanceTamperingDetected(t *testing.T) {
	f := newFixture(t)
	gw, anchor, keyResolver, _ := newGatewayWithProvenance(t, f, 2*time.Second)

	rr := doGet(gw, f.bundleID)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	retrievalIDHex := rr.Header().Get("X-LBVR-Retrieval-ID")
	retrievalIDBytes, _ := hex.DecodeString(retrievalIDHex)
	var retrievalID [32]byte
	copy(retrievalID[:], retrievalIDBytes)

	provReq := httptest.NewRequest(http.MethodGet, "/prov/"+retrievalIDHex, nil)
	provRR := httptest.NewRecorder()
	gw.Routes().ServeHTTP(provRR, provReq)
	provBytes := provRR.Body.Bytes()

	// Tamper: flip a byte inside the canonical doc.
	tampered := append([]byte(nil), provBytes...)
	for i, b := range tampered {
		if b == '"' && i+1 < len(tampered) && tampered[i+1] != '@' {
			// flip a non-control byte well inside the content
			tampered[i+10] ^= 0x01
			break
		}
	}

	rec, _ := anchor.Get(context.Background(), f.bundleID, retrievalID)
	anchors := provenance.StaticAnchorResolver{}
	anchors.SetAnchor(truncateID(f.bundleID), truncateID(retrievalID), rec.ProvHash, rec.BlockNumber)

	v := &provenance.Verifier{Keys: keyResolver, Anchors: anchors}
	res, err := v.Verify(tampered)
	if err == nil && res.Valid {
		t.Fatal("expected Verify to reject tampered doc")
	}
}
