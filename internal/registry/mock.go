package registry

import (
	"context"
	"sync"
	"time"
)

// Mock is an in-memory implementation of Client. Safe for concurrent use.
//
// It mirrors the contract's revert paths: re-registering a bundleId returns
// ErrAlreadyRegistered, reading a missing one returns ErrNotFound, and
// shard validation runs on every write path so the Mock and the on-chain
// path reject identical bad inputs.
//
// The Mock fills RegisteredAt / LastMigratedAt with the value returned by
// `now()`, which defaults to time.Now() but is overrideable by tests via
// SetClock. Callers cannot observe a partial write — RegisterBundle only
// commits to the map after every validation passes.
type Mock struct {
	mu      sync.RWMutex
	records map[[32]byte]*BundleRecord
	now     func() time.Time
}

// NewMock constructs an empty Mock with time.Now() as its clock.
func NewMock() *Mock {
	return &Mock{
		records: make(map[[32]byte]*BundleRecord),
		now:     time.Now,
	}
}

// SetClock overrides the timestamp source. Used by tests that assert exact
// RegisteredAt values; production callers should never call this.
func (m *Mock) SetClock(now func() time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.now = now
}

// RegisterBundle commits a new BundleRecord. Returns:
//   - ErrAlreadyRegistered if bundleID is already present.
//   - ErrNumChunksZero if rec.NumChunks == 0 (matches contract revert).
//   - a wrapped ErrEmptyShardCID if any shard's CID is empty.
//
// On success the record is stored with RegisteredAt = now() and
// LastMigratedAt = zero, matching CIDRegistry.registerBundle's storage
// writes.
func (m *Mock) RegisterBundle(ctx context.Context, bundleID [32]byte, rec BundleRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if rec.NumChunks == 0 {
		return ErrNumChunksZero
	}
	if err := ValidateShards(rec.Shards); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.records[bundleID]; ok {
		return ErrAlreadyRegistered
	}

	stored := rec // copy by value so caller's slices/strings are not aliased
	stored.RegisteredAt = m.now().UTC()
	stored.LastMigratedAt = time.Time{} // zero value; matches lastMigratedAt = 0
	m.records[bundleID] = &stored
	return nil
}

// GetRecord returns a copy of the stored record. Returns ErrNotFound if no
// bundle exists at bundleID. The returned pointer is owned by the caller —
// the Mock never hands out an aliased pointer to its internal map.
func (m *Mock) GetRecord(ctx context.Context, bundleID [32]byte) (*BundleRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	rec, ok := m.records[bundleID]
	if !ok {
		return nil, ErrNotFound
	}
	out := *rec // value copy
	return &out, nil
}

// GetShardLayout returns the fixed-size shard array for bundleID. Returns
// ErrNotFound if the bundle is missing.
func (m *Mock) GetShardLayout(ctx context.Context, bundleID [32]byte) ([ShardCount]ShardPlacement, error) {
	if err := ctx.Err(); err != nil {
		return [ShardCount]ShardPlacement{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	rec, ok := m.records[bundleID]
	if !ok {
		return [ShardCount]ShardPlacement{}, ErrNotFound
	}
	return rec.Shards, nil
}

// UpdateShardLayout rewrites the shard array (matches the contract's
// migration path). Updates LastMigratedAt to now(). Returns ErrNotFound if
// the bundle is missing.
//
// MIGRATOR_ROLE access control is intentionally NOT replicated here — the
// Mock has no notion of msg.sender and the consumers (orchestrator,
// retrieval gateway under PoR) are trusted in-process callers in the test
// matrix. The chain client must enforce it.
func (m *Mock) UpdateShardLayout(ctx context.Context, bundleID [32]byte, shards [ShardCount]ShardPlacement) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateShards(shards); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.records[bundleID]
	if !ok {
		return ErrNotFound
	}
	rec.Shards = shards
	rec.LastMigratedAt = m.now().UTC()
	return nil
}
