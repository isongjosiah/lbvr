package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

// inMemTier is the gateway's test-side tiers.Client. Same shape as the
// ingest CLI's inMemTier but with extra knobs (failGet, slowGet, and a
// wasCancelled flag) so tests can drive the recovery state machine
// through every branch.
type inMemTier struct {
	mu     sync.RWMutex
	store  map[string][]byte
	name   string
	tier   uint8
	prefix string

	// failGet, when non-nil, replaces the stored value with the supplied
	// error. Useful for "shard missing" tests.
	failGet func(cid string) error

	// getDelay, when > 0, blocks Get for this long before returning. Each
	// blocked Get wakes either when getDelay elapses OR when its context
	// cancels — the latter increments cancelObserved.
	getDelay        time.Duration
	cancelObserved  atomic.Int32
	getCallObserved atomic.Int32

	// tamper, when non-nil, mutates a copy of the stored value before
	// returning. Used by the "tampered shard" test to force AES-GCM auth
	// failure or Merkle mismatch.
	tamper func([]byte) []byte
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
	h := sha256.Sum256(data)
	cid := m.prefix + hex.EncodeToString(h[:])
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	m.store[cid] = cp
	return cid, nil
}

// Get honours the configured slowGet delay and respects context
// cancellation. Order of checks: failGet (synchronous error) wins over
// any delay, so the failure mode tests do not need to wait.
func (m *inMemTier) Get(ctx context.Context, cid string) ([]byte, error) {
	m.getCallObserved.Add(1)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if m.failGet != nil {
		if err := m.failGet(cid); err != nil {
			return nil, err
		}
	}
	if m.getDelay > 0 {
		select {
		case <-time.After(m.getDelay):
		case <-ctx.Done():
			m.cancelObserved.Add(1)
			return nil, ctx.Err()
		}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.store[cid]
	if !ok {
		return nil, errors.New("inMemTier: not found")
	}
	out := make([]byte, len(v))
	copy(out, v)
	if m.tamper != nil {
		out = m.tamper(out)
	}
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

// slowGet is a fluent helper for tests that want a tier to sleep before
// returning.
func (m *inMemTier) slowGet(d time.Duration) {
	m.getDelay = d
}

// wasCancelled reports whether at least one Get observed its context
// cancel before its delay elapsed. Used by SlowColdTier_FastPath to
// assert the cold-tier fetch was dropped.
func (m *inMemTier) wasCancelled() bool {
	return m.cancelObserved.Load() > 0
}
