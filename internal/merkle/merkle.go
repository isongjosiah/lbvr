// Package merkle implements the chunked SHA-256 Merkle tree used as the
// integrity root for LBVR-Med FHIR bundles (see CLAUDE.md §4.2).
//
// A payload is split into fixed-size chunks (ChunkSize = 16 KB); the final
// chunk may be shorter. Each chunk hashes to a leaf; internal nodes are
// SHA-256(left || right). When a level has an odd number of nodes, the last
// node is duplicated before pairing ("Bitcoin-style" duplication).
//
// Why the duplication pattern is acceptable here: Bitcoin-style duplication
// is known to permit a second-preimage attack when the verifier does not
// authenticate the leaf count (CVE-2012-2459 class). In LBVR-Med the total
// chunk count is bound to the bundle via the on-chain CIDRegistry entry
// (see CLAUDE.md §4.2 step 6 and §4.5 shard layout) and Verify requires it
// as an input, so an attacker cannot forge a proof by changing tree width.
package merkle

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
)

// ChunkSize is the Merkle-tree leaf size in bytes. Matches CLAUDE.md §4.2.
const ChunkSize = 16 * 1024

// Tree holds the hashes of every level of the Merkle tree in bottom-up order.
// levels[0] is the leaf row; levels[len-1] is a one-element row containing
// the root. numChunks is the real leaf count (i.e. before any duplication
// padding at each level).
type Tree struct {
	levels    [][][32]byte
	numChunks int
}

// Build streams r in ChunkSize-byte chunks and returns the resulting tree.
// An empty reader is a valid input and yields a tree with zero chunks and
// a zero root (the all-zeroes [32]byte); callers that treat empty payloads
// as invalid should check NumChunks() == 0.
func Build(r io.Reader) (*Tree, error) {
	if r == nil {
		return nil, errors.New("merkle: nil reader")
	}
	var leaves [][32]byte
	buf := make([]byte, ChunkSize)
	for {
		n, err := io.ReadFull(r, buf)
		switch {
		case err == nil:
			leaves = append(leaves, sha256.Sum256(buf[:n]))
		case errors.Is(err, io.ErrUnexpectedEOF):
			// Short final chunk.
			leaves = append(leaves, sha256.Sum256(buf[:n]))
		case errors.Is(err, io.EOF):
			// Clean EOF after a full chunk read on the previous iteration.
		default:
			return nil, fmt.Errorf("merkle: read: %w", err)
		}
		if err != nil {
			break
		}
	}
	return buildFromLeaves(leaves), nil
}

func buildFromLeaves(leaves [][32]byte) *Tree {
	t := &Tree{numChunks: len(leaves)}
	if len(leaves) == 0 {
		// Empty payload: single empty level so Root() returns the zero hash.
		t.levels = [][][32]byte{{}}
		return t
	}
	t.levels = append(t.levels, leaves)
	cur := leaves
	for len(cur) > 1 {
		if len(cur)%2 == 1 {
			cur = append(cur, cur[len(cur)-1]) // duplicate last — see package doc
		}
		next := make([][32]byte, len(cur)/2)
		for i := 0; i < len(cur); i += 2 {
			next[i/2] = hashPair(cur[i], cur[i+1])
		}
		t.levels = append(t.levels, next)
		cur = next
	}
	return t
}

// Root returns the Merkle root. For an empty tree the zero value is returned.
func (t *Tree) Root() [32]byte {
	if t == nil || len(t.levels) == 0 {
		return [32]byte{}
	}
	top := t.levels[len(t.levels)-1]
	if len(top) == 0 {
		return [32]byte{}
	}
	return top[0]
}

// NumChunks reports the real leaf count (no duplication padding).
func (t *Tree) NumChunks() int { return t.numChunks }

// Proof returns the sibling path for the chunk at chunkIdx, ordered from the
// leaf level upward. Each step's sibling is the node that must be combined
// with the current node to produce the parent at the next level.
func (t *Tree) Proof(chunkIdx int) ([][32]byte, error) {
	if t == nil || t.numChunks == 0 {
		return nil, errors.New("merkle: empty tree")
	}
	if chunkIdx < 0 || chunkIdx >= t.numChunks {
		return nil, fmt.Errorf("merkle: chunkIdx %d out of range [0,%d)", chunkIdx, t.numChunks)
	}
	if t.numChunks == 1 {
		return [][32]byte{}, nil // single leaf is the root
	}
	proof := make([][32]byte, 0, len(t.levels)-1)
	idx := chunkIdx
	for lvl := 0; lvl < len(t.levels)-1; lvl++ {
		row := t.levels[lvl]
		sibIdx := idx ^ 1
		if sibIdx >= len(row) {
			// Level had odd width; duplication applies — sibling is self.
			proof = append(proof, row[idx])
		} else {
			proof = append(proof, row[sibIdx])
		}
		idx /= 2
	}
	return proof, nil
}

// Verify recomputes the root from chunk + proof and compares against the
// expected root. totalChunks must match the Build-time chunk count so that
// the verifier can determine which levels duplicated their tail.
func Verify(root [32]byte, chunkIdx int, chunk []byte, proof [][32]byte, totalChunks int) bool {
	if totalChunks <= 0 || chunkIdx < 0 || chunkIdx >= totalChunks {
		return false
	}
	expectedProofLen := treeDepth(totalChunks)
	if len(proof) != expectedProofLen {
		return false
	}
	cur := sha256.Sum256(chunk)
	idx := chunkIdx
	levelWidth := totalChunks
	for _, sib := range proof {
		// Reconstruct: if idx is the duplicated tail, sibling == self.
		var left, right [32]byte
		if idx%2 == 0 {
			left, right = cur, sib
		} else {
			left, right = sib, cur
		}
		cur = hashPair(left, right)
		idx /= 2
		if levelWidth%2 == 1 {
			levelWidth++
		}
		levelWidth /= 2
	}
	return cur == root
}

// treeDepth returns the number of internal levels above the leaves for a
// tree with n leaves, accounting for odd-width duplication at each level.
func treeDepth(n int) int {
	if n <= 1 {
		return 0
	}
	depth := 0
	for n > 1 {
		if n%2 == 1 {
			n++
		}
		n /= 2
		depth++
	}
	return depth
}

func hashPair(l, r [32]byte) [32]byte {
	var buf [64]byte
	copy(buf[:32], l[:])
	copy(buf[32:], r[:])
	return sha256.Sum256(buf[:])
}
