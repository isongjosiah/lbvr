// Package por holds the Proof-of-Retrievability primitives used by the
// auditor (challenge issuer) and responders (storage replicas) per
// CLAUDE.md §4.4.
//
// Scope at D14 (E5 bench): just the responder-side prove path.
//
//	auditor side  (postChallenge / recordVerdict): contracts/src/PoRVerifier.sol
//	responder side (build proof + BLS sign):       this package
//	verifier side (off-chain BLS check):           internal/provenance.Verify
//
// On-wire shapes intentionally mirror the Solidity respondToChallenge
// arguments — each field maps 1:1 to a calldata slot. Adding new fields
// here without touching the Solidity ABI is a contract drift; keep them
// aligned.
//
// CVE-2012-2459 note (carried over from internal/merkle): TotalChunks
// must come from the registry's BundleRecord.numChunks, not from the
// responder. The Solidity contract enforces this; we expose the field
// here only so callers can construct a Response struct that round-trips
// through the contract ABI.
package por

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/isongjosiah/lbvr-med/internal/merkle"
	"github.com/isongjosiah/lbvr-med/internal/provenance"
)

// Challenge is the auditor's challenge tuple. Field order matches
// postChallenge calldata order so an abigen-bound caller can pass the
// struct directly without rearrangement.
type Challenge struct {
	BundleID       [32]byte
	ShardIdx       uint32
	ChunkIdx       uint32
	Nonce          [32]byte
	ResponseWindow uint64 // seconds until the auditor may declare timeout
}

// Response is the responder's proof of retrievability, matching
// respondToChallenge's calldata. The Merkle proof is verified on-chain;
// the BLS signature is stored on-chain and verified off-chain by the
// auditor before recordVerdict.
type Response struct {
	ChunkHash   [32]byte               // sha256(chunk_i)
	MerkleProof [][32]byte             // sibling path, leaf→root
	BLSSig      [provenance.SignatureSize]byte
	TotalChunks uint32 // copied from registry.BundleRecord.numChunks
}

// ProveMessage is the BLS-signed payload for a PoR response: the
// challenged chunk concatenated with the auditor's nonce. The auditor
// reconstructs this message off-chain (chunk re-fetched from the
// responding tier; nonce read from the on-chain Challenge) and verifies
// the BLS signature against the responder's aggregate pubkey.
//
// The output buffer is freshly allocated; callers may retain it.
func ProveMessage(chunk []byte, nonce [32]byte) []byte {
	out := make([]byte, len(chunk)+len(nonce))
	copy(out, chunk)
	copy(out[len(chunk):], nonce[:])
	return out
}

// Prove builds a Response for the given (tree, chunk, chunkIdx, challenge)
// pair. `priv` is the responder's BLS private key.
//
// The caller is responsible for ensuring `chunk` is the actual bytes at
// chunkIdx — Prove does NOT re-derive chunks from the tree (the tree
// only stores leaf hashes, not payloads).
func Prove(tree *merkle.Tree, chunk []byte, chunkIdx int, challenge Challenge, priv [provenance.PrivateKeySize]byte) (*Response, error) {
	if tree == nil {
		return nil, errors.New("por: tree is nil")
	}
	if tree.NumChunks() == 0 {
		return nil, errors.New("por: tree is empty")
	}
	if chunkIdx < 0 || chunkIdx >= tree.NumChunks() {
		return nil, fmt.Errorf("por: chunkIdx %d out of range [0,%d)", chunkIdx, tree.NumChunks())
	}
	if uint32(chunkIdx) != challenge.ChunkIdx {
		return nil, fmt.Errorf("por: chunkIdx mismatch: tree=%d challenge=%d", chunkIdx, challenge.ChunkIdx)
	}

	proof, err := tree.Proof(chunkIdx)
	if err != nil {
		return nil, fmt.Errorf("por: build merkle proof: %w", err)
	}

	sig, err := provenance.Sign(priv, ProveMessage(chunk, challenge.Nonce))
	if err != nil {
		return nil, fmt.Errorf("por: BLS sign: %w", err)
	}

	return &Response{
		ChunkHash:   sha256.Sum256(chunk),
		MerkleProof: proof,
		BLSSig:      sig,
		TotalChunks: uint32(tree.NumChunks()),
	}, nil
}
