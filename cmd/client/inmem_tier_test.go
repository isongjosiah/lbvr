package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

// inMemTier is a tiers.Client backed by a map[string][]byte for tests. The
// CID is sha256(data) hex-encoded with a tier-name prefix so the three
// instances produce distinguishable CIDs (matching the spec's "stable CID
// derived from sha256(data)" hint).
//
// Concurrency-safe; the Ingester Puts to all three in parallel.
type inMemTier struct {
	mu     sync.RWMutex
	store  map[string][]byte
	name   string
	tier   uint8
	prefix string // CID prefix; identifies the tier in the test assertions

	// failPut, when non-nil, is consulted before every Put; if it returns
	// non-nil that error is returned. Used by failure-mode tests.
	failPut func(data []byte) error
}

func newInMemTier(name string, tier uint8) *inMemTier {
	return &inMemTier{
		store:  make(map[string][]byte),
		name:   name,
		tier:   tier,
		prefix: name + "-",
	}
}

func (m *inMemTier) Name() string     { return m.name }
func (m *inMemTier) TierClass() uint8 { return m.tier }

func (m *inMemTier) Put(ctx context.Context, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", errors.New("inMemTier: empty data")
	}
	if m.failPut != nil {
		if err := m.failPut(data); err != nil {
			return "", err
		}
	}
	h := sha256.Sum256(data)
	cid := m.prefix + hex.EncodeToString(h[:])

	m.mu.Lock()
	defer m.mu.Unlock()
	// independent copy — caller may reuse the buffer
	cp := make([]byte, len(data))
	copy(cp, data)
	m.store[cid] = cp
	return cid, nil
}

func (m *inMemTier) Get(ctx context.Context, cid string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.store[cid]
	if !ok {
		return nil, errors.New("inMemTier: not found")
	}
	out := make([]byte, len(v))
	copy(out, v)
	return out, nil
}

func (m *inMemTier) Stat(ctx context.Context, cid string) (*tiers.Stat, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.store[cid]
	if !ok {
		return nil, errors.New("inMemTier: not found")
	}
	return &tiers.Stat{CID: cid, SizeBytes: int64(len(v)), StoredAt: time.Now()}, nil
}

func (m *inMemTier) Delete(ctx context.Context, cid string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, cid)
	return nil
}

func (m *inMemTier) snapshot() map[string][]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string][]byte, len(m.store))
	for k, v := range m.store {
		cp := make([]byte, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}
