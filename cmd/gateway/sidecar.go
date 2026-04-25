// Sidecar carries the data the gateway needs but the on-chain CIDRegistry
// doesn't yet store: the AES-256-GCM bundle key and the pre-erasure
// encrypted-bundle length (paddedLen for erasure.Decode).
//
// Conference scope: keys live in plaintext sidecar JSONs alongside the
// ingest manifests. The retrieval gateway is treated as the consortium
// KMS-equivalent (CLAUDE.md §3.1 — KMS is stubbed). The journal extension
// replaces this with real KMS unwrapping.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Sidecar is the lookup interface every retrieval needs after the
// registry returns its on-chain record.
type Sidecar interface {
	Lookup(ctx context.Context, bundleID [32]byte) (Entry, error)
}

// Entry is one bundle's gateway-only metadata.
type Entry struct {
	// Key is the AES-256 bundle key produced at ingest by crypto.GenerateKey.
	Key [32]byte
	// PaddedLen is the encrypted-bundle length erasure.Decode trims to.
	PaddedLen uint32
	// LastChunkBytes is the size of the last *plaintext* chunk; required to
	// know each sealed-chunk boundary during the decrypt loop.
	LastChunkBytes uint32
}

// ErrSidecarMissing is returned when no entry exists for bundleID. The
// gateway turns this into an HTTP 503 (operational state, not a permanent
// failure — the on-chain record is intact, only the off-chain key store
// is missing the entry).
var ErrSidecarMissing = errors.New("gateway: sidecar entry missing")

// InMemorySidecar holds entries in a map. Used by tests and by the
// production gateway when --manifest-dir is omitted (degraded demo mode).
type InMemorySidecar struct {
	mu      sync.RWMutex
	entries map[[32]byte]Entry
}

// NewInMemorySidecar returns an empty in-memory sidecar.
func NewInMemorySidecar() *InMemorySidecar {
	return &InMemorySidecar{entries: make(map[[32]byte]Entry)}
}

// Set inserts (or overwrites) one entry. Safe for concurrent use.
func (s *InMemorySidecar) Set(bundleID [32]byte, e Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[bundleID] = e
}

// Lookup implements Sidecar.
func (s *InMemorySidecar) Lookup(ctx context.Context, bundleID [32]byte) (Entry, error) {
	if err := ctx.Err(); err != nil {
		return Entry{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[bundleID]
	if !ok {
		return Entry{}, ErrSidecarMissing
	}
	return e, nil
}

// FileSidecar reads <bundleID-hex>.json files from a directory at lookup
// time. No global preload because sidecar dirs grow during long-running
// gateway processes (every fresh ingest adds an entry).
type FileSidecar struct {
	dir string
}

// NewFileSidecar binds to dir; the directory must exist (MkdirAll is the
// caller's responsibility).
func NewFileSidecar(dir string) *FileSidecar {
	return &FileSidecar{dir: dir}
}

// fileSidecarRecord is the on-disk shape; bytes are hex-encoded so the
// file is grep-able and stable across runs.
type fileSidecarRecord struct {
	Key            string `json:"key"`
	PaddedLen      uint32 `json:"paddedLen"`
	LastChunkBytes uint32 `json:"lastChunkBytes"`
}

// Lookup implements Sidecar against the on-disk JSON files.
func (s *FileSidecar) Lookup(ctx context.Context, bundleID [32]byte) (Entry, error) {
	if err := ctx.Err(); err != nil {
		return Entry{}, err
	}
	name := hex.EncodeToString(bundleID[:]) + ".json"
	path := filepath.Join(s.dir, name)
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Entry{}, ErrSidecarMissing
		}
		return Entry{}, fmt.Errorf("sidecar: read %s: %w", path, err)
	}
	var rec fileSidecarRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return Entry{}, fmt.Errorf("sidecar: parse %s: %w", path, err)
	}
	keyHex := strings.TrimPrefix(rec.Key, "0x")
	if len(keyHex) != 64 {
		return Entry{}, fmt.Errorf("sidecar: %s key must be 32 bytes (64 hex chars), got %d", path, len(keyHex))
	}
	keyBytes, err := hex.DecodeString(keyHex)
	if err != nil {
		return Entry{}, fmt.Errorf("sidecar: %s decode key: %w", path, err)
	}
	var entry Entry
	copy(entry.Key[:], keyBytes)
	entry.PaddedLen = rec.PaddedLen
	entry.LastChunkBytes = rec.LastChunkBytes
	return entry, nil
}
