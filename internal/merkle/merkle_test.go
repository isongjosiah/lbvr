package merkle

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"io"
	mrand "math/rand"
	"testing"
	"testing/quick"
)

// helper: build from a byte slice.
func buildBytes(t *testing.T, b []byte) *Tree {
	t.Helper()
	tr, err := Build(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return tr
}

func TestBuild_ChunkCounts(t *testing.T) {
	cases := []struct {
		name       string
		size       int
		wantChunks int
	}{
		{"empty", 0, 0},
		{"single short chunk", 100, 1},
		{"exactly one chunk", ChunkSize, 1},
		{"boundary + 1 byte", ChunkSize + 1, 2},
		{"two full chunks", 2 * ChunkSize, 2},
		{"three chunks odd", 2*ChunkSize + 1, 3},
		{"four chunks even", 4 * ChunkSize, 4},
		{"five chunks odd", 4*ChunkSize + 7, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := make([]byte, tc.size)
			if _, err := rand.Read(buf); err != nil {
				t.Fatal(err)
			}
			tr := buildBytes(t, buf)
			if got := tr.NumChunks(); got != tc.wantChunks {
				t.Fatalf("NumChunks = %d, want %d", got, tc.wantChunks)
			}
		})
	}
}

func TestBuild_NilReader(t *testing.T) {
	if _, err := Build(nil); err == nil {
		t.Fatal("expected error on nil reader")
	}
}

func TestRoot_EmptyIsZero(t *testing.T) {
	tr := buildBytes(t, nil)
	var zero [32]byte
	if tr.Root() != zero {
		t.Fatalf("empty tree root = %x, want zero", tr.Root())
	}
}

func TestRoot_SingleChunkEqualsLeafHash(t *testing.T) {
	payload := []byte("hello LBVR-Med")
	tr := buildBytes(t, payload)
	want := sha256.Sum256(payload)
	if tr.Root() != want {
		t.Fatalf("root mismatch: got %x want %x", tr.Root(), want)
	}
}

// TestProofVerify_Exhaustive checks that every chunk index in a deliberately
// odd-width tree round-trips through Proof/Verify. 3 leaves exercises the
// last-chunk duplication path at the leaf level; widening to 5 exercises it
// again at level 1.
func TestProofVerify_Exhaustive(t *testing.T) {
	for _, nLeaves := range []int{1, 2, 3, 4, 5, 7, 8, 9, 16, 17} {
		t.Run("", func(t *testing.T) {
			payload := make([]byte, nLeaves*ChunkSize)
			if _, err := rand.Read(payload); err != nil {
				t.Fatal(err)
			}
			tr := buildBytes(t, payload)
			if tr.NumChunks() != nLeaves {
				t.Fatalf("want %d leaves, got %d", nLeaves, tr.NumChunks())
			}
			for i := 0; i < nLeaves; i++ {
				proof, err := tr.Proof(i)
				if err != nil {
					t.Fatalf("Proof(%d): %v", i, err)
				}
				chunk := payload[i*ChunkSize : min((i+1)*ChunkSize, len(payload))]
				if !Verify(tr.Root(), i, chunk, proof, nLeaves) {
					t.Fatalf("verify failed for leaf %d of %d", i, nLeaves)
				}
			}
		})
	}
}

func TestProof_OutOfRange(t *testing.T) {
	tr := buildBytes(t, make([]byte, 3*ChunkSize))
	if _, err := tr.Proof(-1); err == nil {
		t.Error("expected error for negative index")
	}
	if _, err := tr.Proof(3); err == nil {
		t.Error("expected error for idx == numChunks")
	}
	empty := buildBytes(t, nil)
	if _, err := empty.Proof(0); err == nil {
		t.Error("expected error on empty tree")
	}
}

func TestVerify_RejectsTampering(t *testing.T) {
	payload := make([]byte, 5*ChunkSize)
	if _, err := rand.Read(payload); err != nil {
		t.Fatal(err)
	}
	tr := buildBytes(t, payload)
	proof, err := tr.Proof(2)
	if err != nil {
		t.Fatal(err)
	}
	chunk := payload[2*ChunkSize : 3*ChunkSize]

	// baseline: valid
	if !Verify(tr.Root(), 2, chunk, proof, 5) {
		t.Fatal("sanity: expected valid verify")
	}

	// flip a byte in the chunk
	bad := append([]byte(nil), chunk...)
	bad[0] ^= 0x01
	if Verify(tr.Root(), 2, bad, proof, 5) {
		t.Error("tampered chunk verified")
	}

	// wrong index
	if Verify(tr.Root(), 3, chunk, proof, 5) {
		t.Error("wrong chunkIdx verified")
	}

	// wrong total — different tree depth must be rejected on proof-length mismatch.
	// Note: not every totalChunks mismatch will fail here (e.g. 5 vs 6 produce
	// the same proof-length and for some indices the same path nodes); security
	// against that class of confusion is provided by binding totalChunks to the
	// bundle via the on-chain CIDRegistry, per the package doc comment.
	if Verify(tr.Root(), 2, chunk, proof, 9) {
		t.Error("wrong totalChunks (different depth) verified")
	}

	// truncated proof
	if Verify(tr.Root(), 2, chunk, proof[:len(proof)-1], 5) {
		t.Error("truncated proof verified")
	}

	// corrupted proof node
	badProof := append([][32]byte(nil), proof...)
	badProof[0][0] ^= 0x01
	if Verify(tr.Root(), 2, chunk, badProof, 5) {
		t.Error("corrupted proof verified")
	}
}

// TestProofVerify_Property: testing/quick driven round-trip on random
// chunk counts in [1,100] and random target indices. Also checks that a
// single-bit flip in the chunk causes verification to fail.
func TestProofVerify_Property(t *testing.T) {
	f := func(seed int64, rawN uint8, rawIdx uint16) bool {
		r := mrand.New(mrand.NewSource(seed))
		n := int(rawN)%100 + 1 // [1,100]
		// pick a payload of n chunks with the last one partially filled sometimes
		lastLen := ChunkSize
		if n > 1 && r.Intn(2) == 0 {
			lastLen = 1 + r.Intn(ChunkSize-1)
		}
		payload := make([]byte, (n-1)*ChunkSize+lastLen)
		_, _ = io.ReadFull(r, payload)

		tr, err := Build(bytes.NewReader(payload))
		if err != nil || tr.NumChunks() != n {
			return false
		}
		idx := int(rawIdx) % n
		proof, err := tr.Proof(idx)
		if err != nil {
			return false
		}
		start := idx * ChunkSize
		end := start + ChunkSize
		if end > len(payload) {
			end = len(payload)
		}
		chunk := payload[start:end]

		if !Verify(tr.Root(), idx, chunk, proof, n) {
			return false
		}
		// flip a byte -> verify must fail
		bad := append([]byte(nil), chunk...)
		if len(bad) == 0 {
			return true // single empty-chunk case: skip tamper check
		}
		bad[r.Intn(len(bad))] ^= 0x01
		if Verify(tr.Root(), idx, bad, proof, n) {
			return false
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 200}); err != nil {
		t.Fatal(err)
	}
}

func TestTreeDepth(t *testing.T) {
	cases := map[int]int{
		1:  0,
		2:  1,
		3:  2, // odd -> duplicate to 4, one more level
		4:  2,
		5:  3,
		8:  3,
		9:  4,
		16: 4,
	}
	for n, want := range cases {
		if got := treeDepth(n); got != want {
			t.Errorf("treeDepth(%d) = %d, want %d", n, got, want)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
