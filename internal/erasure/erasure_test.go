package erasure

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	mrand "math/rand"
	"testing"
)

// fixedSeed makes property-style sub-tests reproducible — same payloads each
// run, so a CI failure is replayable without hunting for a seed.
const fixedSeed int64 = 0x1B_4ABCDEF

// roundTripSizes covers the FHIR-bundle distribution measured on Synthea 3.3
// (CLAUDE.md §3.1, §4.5): floor / typical / tail. 14 MiB ≈ P95, 38 MiB ≈ P99,
// 98 MiB ≈ measured max. The 1-byte case stresses padding and the smallest-
// possible RS(2,3) encoding.
var roundTripSizes = []int{
	1,              // single byte — smallest possible; pads to 2
	1 << 10,        // 1 KiB
	16 * 1024,      // 16 KiB — one Merkle chunk
	64 * 1024,      // 64 KiB — small bundle
	128 * 1024,     // 128 KiB — §4.5 erasure floor
	1 << 20,        // 1 MiB
	5 * (1 << 20),  // 5 MiB — §4.5 medium-band ceiling
	14 * (1 << 20), // 14 MiB — measured P95
	38 * (1 << 20), // 38 MiB — measured P99
	98 * (1 << 20), // 98 MiB — measured max
}

func randBytes(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	return b
}

// seededBytes uses math/rand seeded once so that the same sub-test sees the
// same payload across runs — failures are reproducible without printing seeds.
func seededBytes(seed int64, n int) []byte {
	r := mrand.New(mrand.NewSource(seed))
	b := make([]byte, n)
	_, _ = r.Read(b)
	return b
}

func TestEncode_RoundTrip(t *testing.T) {
	for _, size := range roundTripSizes {
		size := size
		t.Run(fmt.Sprintf("size=%d", size), func(t *testing.T) {
			data := seededBytes(fixedSeed+int64(size), size)
			shards, paddedLen, err := Encode(data)
			if err != nil {
				t.Fatalf("Encode(%d): %v", size, err)
			}
			// paddedLen is the original length; shard width is ceil(size/2).
			if paddedLen != size {
				t.Fatalf("paddedLen = %d, want %d (original len)", paddedLen, size)
			}
			wantShard := (size + dataShards - 1) / dataShards
			for i, s := range shards {
				if len(s) != wantShard {
					t.Fatalf("shard[%d] len = %d, want %d", i, len(s), wantShard)
				}
			}
			got, err := Decode(shards, paddedLen)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if !bytes.Equal(got, data) {
				t.Fatalf("round-trip mismatch at size %d", size)
			}
		})
	}
}

// drop returns shards with index i replaced by nil — convenience for the
// single-shard-loss tests.
func drop(in [3][]byte, i int) [3][]byte {
	out := in
	out[i] = nil
	return out
}

func TestDecode_SingleShardLoss_D0(t *testing.T) {
	data := randBytes(t, 1<<20) // 1 MiB
	shards, paddedLen, err := Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(drop(shards, 0), paddedLen)
	if err != nil {
		t.Fatalf("Decode without D0: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("recovered bytes differ after D0 loss")
	}
}

func TestDecode_SingleShardLoss_D1(t *testing.T) {
	data := randBytes(t, 1<<20)
	shards, paddedLen, err := Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(drop(shards, 1), paddedLen)
	if err != nil {
		t.Fatalf("Decode without D1: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("recovered bytes differ after D1 loss")
	}
}

// P0 loss is the fast path: D0+D1 alone reconstruct trivially. Decode must
// still succeed and return identical bytes.
func TestDecode_SingleShardLoss_P0(t *testing.T) {
	data := randBytes(t, 1<<20)
	shards, paddedLen, err := Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(drop(shards, 2), paddedLen)
	if err != nil {
		t.Fatalf("Decode without P0: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("recovered bytes differ after P0 loss")
	}
}

func TestDecode_DoubleShardLoss_Fails(t *testing.T) {
	data := randBytes(t, 64*1024)
	shards, paddedLen, err := Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	pairs := [][2]int{{0, 1}, {0, 2}, {1, 2}}
	for _, p := range pairs {
		bad := shards
		bad[p[0]] = nil
		bad[p[1]] = nil
		_, err := Decode(bad, paddedLen)
		if !errors.Is(err, ErrInsufficientShards) {
			t.Errorf("Decode dropping {%d,%d}: got %v, want ErrInsufficientShards", p[0], p[1], err)
		}
	}
}

// Tampering detection at the erasure layer is best-effort: it only triggers
// when all 3 shards are present after reconstruction (so Verify can compare
// recomputed parity against the provided P0). The Merkle layer in
// internal/merkle is the authoritative integrity check; this test just
// confirms Decode does not silently return corrupt bytes when redundancy
// would have caught the tamper.
func TestDecode_TamperedShard_Fails(t *testing.T) {
	data := randBytes(t, 64*1024)
	shards, paddedLen, err := Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	// Flip a byte in D1 (index 1). All 3 shards remain present.
	shards[1][0] ^= 0xFF
	_, err = Decode(shards, paddedLen)
	if err == nil {
		t.Fatal("Decode of tampered shard set returned nil error")
	}
}

func TestEncode_RejectsEmpty(t *testing.T) {
	if _, _, err := Encode(nil); !errors.Is(err, ErrEmptyInput) {
		t.Errorf("Encode(nil): got %v, want ErrEmptyInput", err)
	}
	if _, _, err := Encode([]byte{}); !errors.Is(err, ErrEmptyInput) {
		t.Errorf("Encode([]byte{}): got %v, want ErrEmptyInput", err)
	}
}

// TestEncode_RejectsOversize allocates 1 GiB+1 to confirm the cap. Slow on
// machines with <2 GiB free RAM; skip with -short. The cap exists to prevent
// accidental allocator blowup from a corrupted size header — see erasure.go.
func TestEncode_RejectsOversize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 1 GiB allocation in short mode")
	}
	big := make([]byte, MaxInputBytes+1)
	_, _, err := Encode(big)
	if !errors.Is(err, ErrInputTooLarge) {
		t.Errorf("Encode(MaxInputBytes+1): got %v, want ErrInputTooLarge", err)
	}
}

// Input of size 7 -> padded to 8 internally -> shardLen = 4. Confirms the
// right-pad arithmetic in Encode produces equal-sized shards even when the
// input length is not a multiple of dataShards.
func TestEncode_ShardSizesEqual(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7}
	shards, paddedLen, err := Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	// paddedLen carries the original length so Decode can trim trailing pad.
	if paddedLen != 7 {
		t.Errorf("paddedLen = %d, want 7 (original len)", paddedLen)
	}
	for i, s := range shards {
		if len(s) != 4 {
			t.Errorf("shard[%d] len = %d, want 4", i, len(s))
		}
	}
}

// Decode must trim the right-padding so the returned slice has exactly the
// caller's original length, not the shard-aligned length.
func TestDecode_TrimsTrailingPadding(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7} // odd size -> 1 byte of zero padding
	shards, paddedLen, err := Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(shards, paddedLen)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(data) {
		t.Fatalf("Decode returned %d bytes, want %d", len(got), len(data))
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("Decode = %v, want %v", got, data)
	}
}

// TestDecode_RejectsBadSizes covers paddedLen sanity checks and the
// shard-size-mismatch path.
func TestDecode_RejectsBadSizes(t *testing.T) {
	data := randBytes(t, 1024) // shardLen = 512
	shards, paddedLen, err := Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(shards, 0); err == nil {
		t.Error("Decode with paddedLen=0 returned nil error")
	}
	if _, err := Decode(shards, -1); err == nil {
		t.Error("Decode with paddedLen=-1 returned nil error")
	}
	// paddedLen too large for the supplied shards: shardLen=512, dataShards=2,
	// so max valid paddedLen is 1024. 2048 must be rejected.
	if _, err := Decode(shards, 2048); err == nil {
		t.Error("Decode with oversize paddedLen returned nil error")
	}
	// paddedLen too small (would imply a different, smaller shard size).
	if _, err := Decode(shards, 1); err == nil {
		t.Error("Decode with too-small paddedLen returned nil error")
	}

	// shard-size mismatch: truncate D1.
	bad := shards
	bad[1] = bad[1][:len(bad[1])-1]
	if _, err := Decode(bad, paddedLen); !errors.Is(err, ErrShardSizeMismatch) {
		t.Errorf("Decode with truncated D1: got %v, want ErrShardSizeMismatch", err)
	}
}
