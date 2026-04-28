package por

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"testing"

	"github.com/isongjosiah/lbvr-med/internal/merkle"
	"github.com/isongjosiah/lbvr-med/internal/provenance"
)

// proveTestFixture: a small bundle, deterministic chunk bytes, fresh BLS keypair.
func proveTestFixture(t *testing.T, numChunks int) (*merkle.Tree, [][]byte, *provenance.KeyPair) {
	t.Helper()
	payload := make([]byte, numChunks*merkle.ChunkSize)
	if _, err := rand.Read(payload); err != nil {
		t.Fatalf("rand: %v", err)
	}
	tree, err := merkle.Build(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("merkle.Build: %v", err)
	}
	if tree.NumChunks() != numChunks {
		t.Fatalf("numChunks: got %d want %d", tree.NumChunks(), numChunks)
	}
	chunks := make([][]byte, numChunks)
	for i := 0; i < numChunks; i++ {
		end := (i + 1) * merkle.ChunkSize
		if end > len(payload) {
			end = len(payload)
		}
		chunks[i] = payload[i*merkle.ChunkSize : end]
	}
	kp, err := provenance.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return tree, chunks, kp
}

func TestProve_RoundTrip(t *testing.T) {
	for _, n := range []int{1, 2, 5, 8, 17, 64, 257} {
		t.Run("", func(t *testing.T) {
			tree, chunks, kp := proveTestFixture(t, n)
			chunkIdx := n / 2
			ch := Challenge{
				BundleID: [32]byte{0xab, 0xcd},
				ShardIdx: 1,
				ChunkIdx: uint32(chunkIdx),
				Nonce:    [32]byte{0x01, 0x02, 0x03},
			}
			resp, err := Prove(tree, chunks[chunkIdx], chunkIdx, ch, kp.PrivateBytes)
			if err != nil {
				t.Fatalf("Prove: %v", err)
			}

			// Merkle: chunk hash + proof reconstructs root.
			if got := sha256.Sum256(chunks[chunkIdx]); got != resp.ChunkHash {
				t.Errorf("chunkHash mismatch")
			}
			if !merkle.Verify(tree.Root(), chunkIdx, chunks[chunkIdx], resp.MerkleProof, n) {
				t.Errorf("merkle.Verify rejected a freshly built proof for n=%d idx=%d", n, chunkIdx)
			}

			// BLS: signature verifies against the responder's pubkey + spec'd message.
			msg := ProveMessage(chunks[chunkIdx], ch.Nonce)
			if err := provenance.Verify(kp.PublicBytes, msg, resp.BLSSig); err != nil {
				t.Errorf("provenance.Verify: %v", err)
			}

			// Tree-width binding: TotalChunks must equal NumChunks().
			if resp.TotalChunks != uint32(n) {
				t.Errorf("TotalChunks=%d want %d", resp.TotalChunks, n)
			}
		})
	}
}

func TestProve_RejectsMismatchedIdx(t *testing.T) {
	tree, chunks, kp := proveTestFixture(t, 8)
	ch := Challenge{ChunkIdx: 3}
	_, err := Prove(tree, chunks[2], 2, ch, kp.PrivateBytes) // chunkIdx 2 vs challenge 3
	if err == nil {
		t.Fatal("expected error on chunkIdx ≠ challenge.ChunkIdx")
	}
}

func TestProve_RejectsOutOfRange(t *testing.T) {
	tree, chunks, kp := proveTestFixture(t, 4)
	ch := Challenge{ChunkIdx: 4}
	_, err := Prove(tree, chunks[0], 4, ch, kp.PrivateBytes) // 4 == numChunks → out of range
	if err == nil {
		t.Fatal("expected out-of-range error")
	}
}

func TestProveMessage_Shape(t *testing.T) {
	chunk := []byte("hello")
	nonce := [32]byte{0xff, 0x00}
	out := ProveMessage(chunk, nonce)
	if len(out) != len(chunk)+32 {
		t.Fatalf("len=%d want %d", len(out), len(chunk)+32)
	}
	if !bytes.Equal(out[:len(chunk)], chunk) {
		t.Errorf("chunk prefix mismatch")
	}
	if !bytes.Equal(out[len(chunk):], nonce[:]) {
		t.Errorf("nonce suffix mismatch")
	}
}
