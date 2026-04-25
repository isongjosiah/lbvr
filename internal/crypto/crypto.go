// Package crypto provides the AES-256-GCM chunk envelope used by LBVR-Med
// clients (see CLAUDE.md §4.2 step 3). One key per bundle; each chunk is
// sealed under a fresh random 12-byte nonce that is prepended to the
// ciphertext-and-tag output.
//
// Key wrapping (consortium KMS / HSM) is deliberately stubbed — the
// conference scope in CLAUDE.md §3.1 keeps the KMS as a stub, and the
// journal extension (§3.2) will implement it.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// NonceSize is the GCM nonce width in bytes. 96 bits is the NIST SP 800-38D
// recommended size and is what crypto/cipher.NewGCM returns.
const NonceSize = 12

// KeySize is the AES-256 key length in bytes.
const KeySize = 32

// GenerateKey returns a cryptographically random 256-bit key.
//
// TODO(journal): wrap the generated key under the consortium KMS root key
// (CLAUDE.md §3.2). Conference scope treats the bundle key as in-memory only.
func GenerateKey() ([32]byte, error) {
	var k [KeySize]byte
	if _, err := io.ReadFull(rand.Reader, k[:]); err != nil {
		return k, fmt.Errorf("crypto: key gen: %w", err)
	}
	return k, nil
}

// SealChunk encrypts plaintext with key and returns nonce||ciphertext||tag.
// A fresh 96-bit nonce is drawn from crypto/rand for every call; never
// call this twice with the same key+nonce pair.
func SealChunk(key [32]byte, plaintext []byte) ([]byte, error) {
	aead, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	var nonce [NonceSize]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, fmt.Errorf("crypto: nonce: %w", err)
	}
	// Seal writes into a dst that already contains nonce, so the output is
	// a single contiguous slice of nonce||ciphertext||tag — no extra copy.
	out := make([]byte, NonceSize, NonceSize+len(plaintext)+aead.Overhead())
	copy(out, nonce[:])
	out = aead.Seal(out, nonce[:], plaintext, nil)
	return out, nil
}

// OpenChunk inverts SealChunk. It returns an error if sealed is shorter than
// the nonce+tag overhead or if authentication fails (tampering).
func OpenChunk(key [32]byte, sealed []byte) ([]byte, error) {
	aead, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(sealed) < NonceSize+aead.Overhead() {
		return nil, errors.New("crypto: sealed chunk too short")
	}
	nonce := sealed[:NonceSize]
	ct := sealed[NonceSize:]
	pt, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: open: %w", err)
	}
	return pt, nil
}

// sealWithNonce is an internal variant used only by KAT tests. It is not
// exported because callers should never pick their own nonce in production —
// misuse (reuse) destroys GCM's security.
func sealWithNonce(key [32]byte, nonce []byte, plaintext, aad []byte) ([]byte, error) {
	aead, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("crypto: nonce must be %d bytes", NonceSize)
	}
	return aead.Seal(nil, nonce, plaintext, aad), nil
}

func newGCM(key [32]byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("crypto: aes: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: gcm: %w", err)
	}
	return aead, nil
}
