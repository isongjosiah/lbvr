package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/isongjosiah/lbvr-med/internal/merkle"
	"github.com/isongjosiah/lbvr-med/internal/registry"
	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

// quietLogger discards log output so test runs stay readable.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// makeBundle writes a synthetic FHIR-shaped JSON to a tmp dir and returns
// the path + raw bytes. It is intentionally not the real Synthea fixture
// because eval/synthea/upstream/output-* is gitignored (CLAUDE.md §6 +
// .gitignore). Size is enough to span more than one merkle.ChunkSize so
// the Merkle tree has > 1 leaf.
func makeBundle(t *testing.T) (string, []byte) {
	t.Helper()

	// Build a JSON ~50 KiB so we get ~4 Merkle leaves. Repeating filler
	// to exceed ChunkSize twice over.
	payload := strings.Repeat(`{"resource":{"resourceType":"Observation","status":"final","code":{"coding":[{"system":"http://loinc.org","code":"8302-2","display":"Body Height"}]},"valueQuantity":{"value":175.4,"unit":"cm"}}},`, 350)
	body := `{"resourceType":"Bundle","type":"transaction","entry":[` + strings.TrimSuffix(payload, ",") + `]}`
	if len(body) < 2*merkle.ChunkSize {
		// Pad to ensure > 1 Merkle leaf irrespective of locale/env.
		body += strings.Repeat(" ", 2*merkle.ChunkSize-len(body)+1)
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "bundle.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p, []byte(body)
}

// newTestIngester wires the in-memory tier clients, the Mock registry,
// and the replicaEncoder into a ready-to-go Ingester.
func newTestIngester(t *testing.T) (*Ingester, *inMemTier, *inMemTier, *inMemTier, *registry.Mock) {
	t.Helper()
	hot := newInMemTier("pinata-mem", tiers.TierHot)
	warm := newInMemTier("filebase-mem", tiers.TierWarm)
	cold := newInMemTier("arweave-mem", tiers.TierCold)
	reg := registry.NewMock()

	ing, err := NewIngester(IngesterOpts{
		Hot:        hot,
		Warm:       warm,
		Cold:       cold,
		Registry:   reg,
		Encoder:    replicaEncoder{},
		ClientAddr: defaultClientAddr,
		Logger:     quietLogger(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return ing, hot, warm, cold, reg
}

func TestIngest_EndToEnd(t *testing.T) {
	t.Parallel()
	bundlePath, plain := makeBundle(t)
	ing, hot, warm, cold, reg := newTestIngester(t)

	res, err := ing.Ingest(context.Background(), IngestRequest{
		Path:     bundlePath,
		PolicyID: registry.Keccak256([]byte("lbvr://policy/test")),
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}

	// 1. The Merkle root in the result equals one re-built from the raw bytes.
	tree, err := merkle.Build(bytes.NewReader(plain))
	if err != nil {
		t.Fatalf("rebuild merkle: %v", err)
	}
	if res.MerkleRoot != tree.Root() {
		t.Fatalf("merkle root mismatch: got %x, want %x", res.MerkleRoot, tree.Root())
	}
	if int(res.NumChunks) != tree.NumChunks() {
		t.Fatalf("numChunks mismatch: got %d, want %d", res.NumChunks, tree.NumChunks())
	}
	if res.NumChunks < 2 {
		t.Fatalf("test bundle should span >1 chunk, got %d", res.NumChunks)
	}

	// 2. Registry has exactly one record under the derived bundleID.
	rec, err := reg.GetRecord(context.Background(), res.BundleID)
	if err != nil {
		t.Fatalf("registry GetRecord: %v", err)
	}
	if rec.MerkleRoot != res.MerkleRoot {
		t.Fatal("stored merkleRoot != result merkleRoot")
	}
	if rec.NumChunks != res.NumChunks {
		t.Fatal("stored numChunks != result numChunks")
	}
	if rec.RegisteredAt.IsZero() {
		t.Fatal("registeredAt should be non-zero on Mock register")
	}

	// 3. BundleID derivation matches the public helper.
	expectedID := registry.BundleID(defaultClientAddr, res.MerkleRoot)
	if expectedID != res.BundleID {
		t.Fatalf("bundleID derivation mismatch: got %x, want %x", res.BundleID, expectedID)
	}

	// 4. Each tier holds the shard at the expected CID.
	wantTiers := []struct {
		client *inMemTier
		idx    int
		tier   uint8
	}{
		{hot, 0, registry.TierHot},
		{warm, 1, registry.TierWarm},
		{cold, 2, registry.TierCold},
	}
	for _, tt := range wantTiers {
		got, err := tt.client.Get(context.Background(), res.Shards[tt.idx].CID)
		if err != nil {
			t.Fatalf("tier %s missing CID %q: %v", tt.client.Name(), res.Shards[tt.idx].CID, err)
		}
		if len(got) == 0 {
			t.Fatalf("tier %s returned empty data for shard %d", tt.client.Name(), tt.idx)
		}
		if res.Shards[tt.idx].Tier != tt.tier {
			t.Fatalf("shard %d tier = %d, want %d", tt.idx, res.Shards[tt.idx].Tier, tt.tier)
		}
		if !strings.HasPrefix(res.Shards[tt.idx].CID, tt.client.prefix) {
			t.Fatalf("shard %d CID %q lacks tier prefix %q", tt.idx, res.Shards[tt.idx].CID, tt.client.prefix)
		}
	}

	// 5. With replicaEncoder all three shards are byte-identical (this
	// will NOT hold once internal/erasure replaces it — flag accordingly).
	hotSnap := hot.snapshot()
	warmSnap := warm.snapshot()
	coldSnap := cold.snapshot()
	if len(hotSnap) != 1 || len(warmSnap) != 1 || len(coldSnap) != 1 {
		t.Fatalf("expected one object per tier, got hot=%d warm=%d cold=%d",
			len(hotSnap), len(warmSnap), len(coldSnap))
	}
	var hotBlob, warmBlob, coldBlob []byte
	for _, v := range hotSnap {
		hotBlob = v
	}
	for _, v := range warmSnap {
		warmBlob = v
	}
	for _, v := range coldSnap {
		coldBlob = v
	}
	if !bytes.Equal(hotBlob, warmBlob) || !bytes.Equal(warmBlob, coldBlob) {
		t.Fatal("replicaEncoder shards must be byte-identical")
	}
	// And the encrypted blob must NOT equal the plaintext (sanity).
	if bytes.Equal(hotBlob, plain) {
		t.Fatal("uploaded shard equals plaintext — encryption did not run")
	}
}

func TestIngest_DryRunSkipsUploadAndRegister(t *testing.T) {
	t.Parallel()
	bundlePath, _ := makeBundle(t)
	ing, hot, warm, cold, reg := newTestIngester(t)

	res, err := ing.Ingest(context.Background(), IngestRequest{
		Path:     bundlePath,
		PolicyID: [32]byte{0xaa},
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("dry-run ingest: %v", err)
	}
	if len(hot.snapshot())+len(warm.snapshot())+len(cold.snapshot()) != 0 {
		t.Fatal("dry-run should not have uploaded any shard")
	}
	if _, err := reg.GetRecord(context.Background(), res.BundleID); !errors.Is(err, registry.ErrNotFound) {
		t.Fatalf("dry-run should not have registered; got err=%v", err)
	}
	for _, s := range res.Shards {
		if !strings.HasPrefix(s.CID, "dryrun-") {
			t.Fatalf("dry-run CID lacks dryrun- prefix: %q", s.CID)
		}
	}
}

func TestIngest_RejectsEmptyBundle(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	ing, _, _, _, _ := newTestIngester(t)
	_, err := ing.Ingest(context.Background(), IngestRequest{Path: p})
	if err == nil {
		t.Fatal("expected error on empty bundle")
	}
}

func TestIngest_TierUploadFailureBubblesUp(t *testing.T) {
	t.Parallel()
	bundlePath, _ := makeBundle(t)
	ing, _, warm, _, reg := newTestIngester(t)
	warm.failPut = func(_ []byte) error { return errors.New("filebase: 503") }

	res, err := ing.Ingest(context.Background(), IngestRequest{Path: bundlePath})
	if err == nil {
		t.Fatal("expected ingest to fail when warm tier rejects upload")
	}
	if !strings.Contains(err.Error(), "filebase-mem") {
		t.Fatalf("error should identify the failing tier: %v", err)
	}
	// Registry must NOT have been written.
	if res != nil {
		t.Fatalf("result should be nil on upload failure, got %+v", res)
	}
	// And the registry has zero records.
	if _, err := reg.GetRecord(context.Background(), [32]byte{}); !errors.Is(err, registry.ErrNotFound) {
		t.Fatalf("registry should be empty; got %v", err)
	}
}

func TestIngestCorpus_RoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const n = 5
	for i := 0; i < n; i++ {
		body := strings.Repeat("x", merkle.ChunkSize+i) // distinct lengths → distinct roots
		p := filepath.Join(dir, "b-"+string(rune('a'+i))+".json")
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	ing, hot, warm, cold, reg := newTestIngester(t)
	results, err := ing.IngestCorpus(context.Background(), dir, [32]byte{0x01}, 4, false)
	if err != nil {
		t.Fatalf("corpus: %v", err)
	}
	if len(results) != n {
		t.Fatalf("expected %d results, got %d", n, len(results))
	}
	for i, r := range results {
		if r == nil {
			t.Fatalf("result %d is nil", i)
		}
		if _, err := reg.GetRecord(context.Background(), r.BundleID); err != nil {
			t.Fatalf("result %d not in registry: %v", i, err)
		}
	}
	if got := len(hot.snapshot()); got != n {
		t.Fatalf("hot shard count = %d, want %d", got, n)
	}
	if got := len(warm.snapshot()); got != n {
		t.Fatalf("warm shard count = %d, want %d", got, n)
	}
	if got := len(cold.snapshot()); got != n {
		t.Fatalf("cold shard count = %d, want %d", got, n)
	}
}

func TestIngest_ManifestEmitted(t *testing.T) {
	t.Parallel()
	bundlePath, _ := makeBundle(t)
	manifestDir := t.TempDir()
	hot := newInMemTier("pinata-mem", tiers.TierHot)
	warm := newInMemTier("filebase-mem", tiers.TierWarm)
	cold := newInMemTier("arweave-mem", tiers.TierCold)
	reg := registry.NewMock()
	ing, err := NewIngester(IngesterOpts{
		Hot:         hot,
		Warm:        warm,
		Cold:        cold,
		Registry:    reg,
		Encoder:     replicaEncoder{},
		ClientAddr:  defaultClientAddr,
		ManifestDir: manifestDir,
		Logger:      quietLogger(),
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := ing.Ingest(context.Background(), IngestRequest{Path: bundlePath})
	if err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(manifestDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("manifest dir has %d files, want 1", len(entries))
	}
	body, err := os.ReadFile(filepath.Join(manifestDir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	var view manifestView
	if err := json.Unmarshal(body, &view); err != nil {
		t.Fatalf("manifest JSON parse: %v", err)
	}
	if view.NumChunks != res.NumChunks {
		t.Fatalf("manifest numChunks = %d, want %d", view.NumChunks, res.NumChunks)
	}
	if len(view.Shards) != 3 {
		t.Fatalf("manifest shards = %d, want 3", len(view.Shards))
	}
	if view.Shards[0].TierName != "hot" || view.Shards[1].TierName != "warm" || view.Shards[2].TierName != "cold" {
		t.Fatalf("manifest tier names wrong: %+v", view.Shards)
	}
}

func TestParseAddrHex(t *testing.T) {
	t.Parallel()
	good, err := parseAddrHex("0x" + strings.Repeat("ab", 20))
	if err != nil {
		t.Fatal(err)
	}
	if good[0] != 0xab || good[19] != 0xab {
		t.Fatalf("parsed bytes wrong: %x", good)
	}
	if _, err := parseAddrHex("notlongenough"); err == nil {
		t.Fatal("expected error on short addr")
	}
	if _, err := parseAddrHex("zz" + strings.Repeat("a", 38)); err == nil {
		t.Fatal("expected error on non-hex addr")
	}
}

func TestReplicaEncoder_PassThrough(t *testing.T) {
	t.Parallel()
	data := []byte("hello world")
	shards, padded, err := (replicaEncoder{}).Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	if padded != len(data) {
		t.Fatalf("paddedLen = %d, want %d", padded, len(data))
	}
	for i, s := range shards {
		if !bytes.Equal(s, data) {
			t.Fatalf("shard %d not a copy of data", i)
		}
	}
	// And mutations to one shard must not propagate to siblings.
	shards[0][0] ^= 0xff
	if bytes.Equal(shards[0], shards[1]) {
		t.Fatal("shards aliased; replicaEncoder must produce independent copies")
	}
}

func TestReplicaEncoder_RejectsEmpty(t *testing.T) {
	t.Parallel()
	if _, _, err := (replicaEncoder{}).Encode(nil); err == nil {
		t.Fatal("expected error on empty input")
	}
}
