package registry

import (
	"context"
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"
)

// fixed clock for deterministic timestamp assertions
func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func newRecord() BundleRecord {
	return BundleRecord{
		MerkleRoot: [32]byte{0x01, 0x02, 0x03},
		NumChunks:  4,
		Shards: [ShardCount]ShardPlacement{
			{CID: "QmHot", Tier: TierHot},
			{CID: "QmWarm", Tier: TierWarm},
			{CID: "ar-cold", Tier: TierCold},
		},
		Owner:    "0x0000000000000000000000000000000000000001",
		PolicyID: [32]byte{0xff},
	}
}

func TestMock_RegisterAndGet(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	m := NewMock()
	m.SetClock(fixedClock(now))

	id := [32]byte{0xaa, 0xbb}
	if err := m.RegisterBundle(ctx, id, newRecord()); err != nil {
		t.Fatalf("register: %v", err)
	}

	got, err := m.GetRecord(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.MerkleRoot != ([32]byte{0x01, 0x02, 0x03}) {
		t.Fatalf("merkleRoot mismatch: %x", got.MerkleRoot)
	}
	if got.NumChunks != 4 {
		t.Fatalf("numChunks mismatch: %d", got.NumChunks)
	}
	if got.Shards[0].CID != "QmHot" || got.Shards[0].Tier != TierHot {
		t.Fatalf("shard 0 mismatch: %+v", got.Shards[0])
	}
	if got.Shards[2].Tier != TierCold {
		t.Fatalf("shard 2 tier wrong: %d", got.Shards[2].Tier)
	}
	if !got.RegisteredAt.Equal(now) {
		t.Fatalf("registeredAt = %v, want %v", got.RegisteredAt, now)
	}
	if !got.LastMigratedAt.IsZero() {
		t.Fatalf("lastMigratedAt should be zero on register, got %v", got.LastMigratedAt)
	}

	// Mutating returned struct must not affect storage (Mock returns a copy).
	got.MerkleRoot = [32]byte{0xde, 0xad}
	again, _ := m.GetRecord(ctx, id)
	if again.MerkleRoot != ([32]byte{0x01, 0x02, 0x03}) {
		t.Fatal("returned record was aliased to internal storage")
	}
}

func TestMock_DuplicateRegistration(t *testing.T) {
	t.Parallel()
	m := NewMock()
	id := [32]byte{0x01}
	if err := m.RegisterBundle(context.Background(), id, newRecord()); err != nil {
		t.Fatal(err)
	}
	err := m.RegisterBundle(context.Background(), id, newRecord())
	if !errors.Is(err, ErrAlreadyRegistered) {
		t.Fatalf("err = %v, want ErrAlreadyRegistered", err)
	}
}

func TestMock_GetMissing(t *testing.T) {
	t.Parallel()
	m := NewMock()
	if _, err := m.GetRecord(context.Background(), [32]byte{0x42}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetRecord on missing: %v", err)
	}
	if _, err := m.GetShardLayout(context.Background(), [32]byte{0x42}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetShardLayout on missing: %v", err)
	}
	if err := m.UpdateShardLayout(context.Background(), [32]byte{0x42}, newRecord().Shards); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateShardLayout on missing: %v", err)
	}
}

func TestMock_RegisterRejectsZeroChunks(t *testing.T) {
	t.Parallel()
	m := NewMock()
	rec := newRecord()
	rec.NumChunks = 0
	err := m.RegisterBundle(context.Background(), [32]byte{0x01}, rec)
	if !errors.Is(err, ErrNumChunksZero) {
		t.Fatalf("err = %v, want ErrNumChunksZero", err)
	}
}

func TestMock_RegisterRejectsEmptyShardCID(t *testing.T) {
	t.Parallel()
	m := NewMock()
	rec := newRecord()
	rec.Shards[1].CID = ""
	err := m.RegisterBundle(context.Background(), [32]byte{0x01}, rec)
	if !errors.Is(err, ErrEmptyShardCID) {
		t.Fatalf("err = %v, want ErrEmptyShardCID", err)
	}
}

func TestMock_GetShardLayout(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m := NewMock()
	id := [32]byte{0x01}
	if err := m.RegisterBundle(ctx, id, newRecord()); err != nil {
		t.Fatal(err)
	}
	shards, err := m.GetShardLayout(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if shards[1].CID != "QmWarm" {
		t.Fatalf("warm CID = %q", shards[1].CID)
	}
}

func TestMock_UpdateShardLayout(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	migAt := now.Add(time.Hour)
	m := NewMock()
	m.SetClock(fixedClock(now))

	id := [32]byte{0x01}
	if err := m.RegisterBundle(ctx, id, newRecord()); err != nil {
		t.Fatal(err)
	}

	m.SetClock(fixedClock(migAt))
	newShards := [ShardCount]ShardPlacement{
		{CID: "QmHotV2", Tier: TierHot},
		{CID: "QmWarmV2", Tier: TierWarm},
		{CID: "ar-coldV2", Tier: TierCold},
	}
	if err := m.UpdateShardLayout(ctx, id, newShards); err != nil {
		t.Fatal(err)
	}
	got, err := m.GetRecord(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Shards[0].CID != "QmHotV2" {
		t.Fatalf("post-migration shards[0].CID = %q", got.Shards[0].CID)
	}
	if !got.LastMigratedAt.Equal(migAt) {
		t.Fatalf("lastMigratedAt = %v, want %v", got.LastMigratedAt, migAt)
	}

	// Empty CID rejected on update too.
	bad := newShards
	bad[0].CID = ""
	if err := m.UpdateShardLayout(ctx, id, bad); !errors.Is(err, ErrEmptyShardCID) {
		t.Fatalf("update with empty CID: %v", err)
	}
}

func TestMock_ContextCancellation(t *testing.T) {
	t.Parallel()
	m := NewMock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.RegisterBundle(ctx, [32]byte{0x01}, newRecord()); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestMock_ConcurrentRegistration(t *testing.T) {
	t.Parallel()
	m := NewMock()
	const n = 64
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			var id [32]byte
			id[0] = byte(i)
			rec := newRecord()
			rec.MerkleRoot[0] = byte(i)
			if err := m.RegisterBundle(context.Background(), id, rec); err != nil {
				t.Errorf("concurrent register %d: %v", i, err)
			}
		}()
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		var id [32]byte
		id[0] = byte(i)
		if _, err := m.GetRecord(context.Background(), id); err != nil {
			t.Errorf("get %d: %v", i, err)
		}
	}
}

func TestKeccak256_KAT(t *testing.T) {
	t.Parallel()
	cases := []struct{ in, want string }{
		// Known Ethereum keccak256 vectors.
		{"", "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"},
		{"abc", "4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45"},
		{"The quick brown fox jumps over the lazy dog",
			"4d741b6f1eb29cb2a9b9911c82f56fa8d73b04959d3d9d222895df6c0b28aa15"},
	}
	for _, c := range cases {
		got := Keccak256([]byte(c.in))
		if hex.EncodeToString(got[:]) != c.want {
			t.Fatalf("keccak256(%q) = %x, want %s", c.in, got, c.want)
		}
	}
}

func TestBundleID_DerivedFromAddrAndRoot(t *testing.T) {
	t.Parallel()
	addr := [20]byte{}
	addr[19] = 0x01 // 0x0000...0001
	root := [32]byte{0xaa, 0xbb, 0xcc}

	id := BundleID(addr, root)

	// Reproduce the derivation by hand.
	var packed [52]byte
	copy(packed[:20], addr[:])
	copy(packed[20:], root[:])
	expected := Keccak256(packed[:])
	if id != expected {
		t.Fatalf("BundleID = %x, want %x", id, expected)
	}

	// Different root yields different bundleId (collision-resistance smoke test).
	root2 := root
	root2[0] = 0xab
	if BundleID(addr, root2) == id {
		t.Fatal("BundleID should differ for different roots")
	}
}

func TestValidateShards(t *testing.T) {
	t.Parallel()
	good := newRecord().Shards
	if err := ValidateShards(good); err != nil {
		t.Fatalf("good shards rejected: %v", err)
	}
	bad := good
	bad[2].CID = ""
	if err := ValidateShards(bad); !errors.Is(err, ErrEmptyShardCID) {
		t.Fatalf("expected ErrEmptyShardCID, got %v", err)
	}
}

func TestNewChain_StubReturnsErr(t *testing.T) {
	t.Parallel()
	c, err := NewChain(context.Background(), "http://x", "0xabc", "0xkey")
	if err != nil {
		t.Fatalf("NewChain: %v", err)
	}
	if err := c.RegisterBundle(context.Background(), [32]byte{}, newRecord()); !errors.Is(err, ErrChainNotImplemented) {
		t.Fatalf("RegisterBundle stub err = %v, want ErrChainNotImplemented", err)
	}
	if _, err := c.GetRecord(context.Background(), [32]byte{}); !errors.Is(err, ErrChainNotImplemented) {
		t.Fatalf("GetRecord stub err = %v", err)
	}
	if _, err := c.GetShardLayout(context.Background(), [32]byte{}); !errors.Is(err, ErrChainNotImplemented) {
		t.Fatalf("GetShardLayout stub err = %v", err)
	}
	if err := c.UpdateShardLayout(context.Background(), [32]byte{}, newRecord().Shards); !errors.Is(err, ErrChainNotImplemented) {
		t.Fatalf("UpdateShardLayout stub err = %v", err)
	}
}

func TestNewChain_RejectsEmptyArgs(t *testing.T) {
	t.Parallel()
	if _, err := NewChain(context.Background(), "", "0xabc", "0xkey"); err == nil {
		t.Fatal("expected error for empty rpcURL")
	}
	if _, err := NewChain(context.Background(), "http://x", "", "0xkey"); err == nil {
		t.Fatal("expected error for empty contractAddr")
	}
	if _, err := NewChain(context.Background(), "http://x", "0xabc", ""); err == nil {
		t.Fatal("expected error for empty privKey")
	}
}
