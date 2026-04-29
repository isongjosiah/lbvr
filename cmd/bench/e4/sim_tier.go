// Calibrated lognormal stand-in for the three tier backends, sized for
// E4's question — "from PUT-return, when does the bundle become globally
// reachable?" Two distinct distributions per tier:
//
//	put_latency       — time the Put RPC takes to return the CID
//	propagation_delay — additional time before a non-source gateway can
//	                    Get the CID. Models DHT propagation (Pinata),
//	                    cross-region S3 sync (Filebase), or chain
//	                    settlement (Arweave/Irys).
//
// The bench's TTA observation is propagation_delay (= first_ok_at_ms).
// We track put_latency separately so the JSON has both axes — paper
// figures may want to plot end-to-end (put + propagation) or just the
// post-PUT slope.
//
// CALIBRATION SOURCES (per tier):
//
//	Hot — Pinata (private gateway pinning):
//	  put_latency P50/P99   = 80ms / 400ms     (E9 sim_tier — read latency
//	                                            against a warm cache; PUT
//	                                            is similar order)
//	  propagation P50/P99   = 50ms / 500ms     (Pinata's dedicated gateway
//	                                            indexes within ~ms; the
//	                                            tail covers occasional DHT
//	                                            spikes when the public IPFS
//	                                            network is queried)
//
//	Warm — Filebase (S3 + IPFS pinning):
//	  put_latency P50/P99   = 120ms / 600ms    (E9 sim_tier)
//	  propagation P50/P99   = 200ms / 3000ms   (S3 cross-region eventual
//	                                            consistency; bounded by AWS
//	                                            region replication)
//
//	Cold — Irys (Arweave bundler):
//	  put_latency P50/P99   = 500ms / 8000ms   (E9 sim_tier — accepts the
//	                                            upload to Irys node)
//	  propagation P50/P99   = 5000ms / 60000ms (chain settlement on
//	                                            Arweave; Irys docs note
//	                                            ~minutes typical)
//
// The numbers are conservative and consistent with E9's sim — when live
// runs land they replace these without invalidating the methodology.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	mrand "math/rand"
	"sync"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

// Lognormal calibration for Put latency (matches E9). mu = ln(P50_ns);
// sigma = (ln(P99_ns) - mu) / 2.326.
type lnPair struct {
	muNs   float64
	sigma  float64
	p50ms  int
	p99ms  int
	tag    string
}

var (
	hotPut = lnPair{muNs: 18.1974681, sigma: 0.6919175, p50ms: 80, p99ms: 400, tag: "hot-put"}
	warmPut = lnPair{muNs: 18.6028196, sigma: 0.6919175, p50ms: 120, p99ms: 600, tag: "warm-put"}
	coldPut = lnPair{muNs: 20.0312124, sigma: 1.1925473, p50ms: 500, p99ms: 8000, tag: "cold-put"}

	// Propagation: time after Put-return before a non-source gateway can Get.
	// Hot: Pinata's dedicated gateway returns near-immediately (~ms cache),
	// but we want the global-reachability number — i.e., what a different
	// IPFS gateway would observe. DHT propagation is the long tail.
	hotProp = lnPair{muNs: 17.7274581, sigma: 0.9904189, p50ms: 50, p99ms: 500, tag: "hot-prop"}
	// Warm: S3 multi-region eventual consistency ranges 100ms→few seconds.
	warmProp = lnPair{muNs: 19.1138269, sigma: 1.1648131, p50ms: 200, p99ms: 3000, tag: "warm-prop"}
	// Cold: Arweave settlement, lognormal heavy tail.
	coldProp = lnPair{muNs: 22.3327037, sigma: 1.0683523, p50ms: 5000, p99ms: 60000, tag: "cold-prop"}
)

// sample returns one lognormal draw in nanoseconds.
func (p lnPair) sample(rng *mrand.Rand) int64 {
	z := rng.NormFloat64()
	return int64(math.Exp(p.muNs + p.sigma*z))
}

// simTTA implements tiers.Client + a propagation-delay model. Every Put
// records (cid → readyAt = put-complete + propagation_delay). Get returns
// ErrNotReady before readyAt, the synthesised payload after.
type simTTA struct {
	name      string
	tierClass uint8

	putDist  lnPair
	propDist lnPair

	// rng is gated by a mutex because Put + Get may be invoked from
	// different goroutines (the caller drives polling concurrently with
	// background uploads in a future version of the harness).
	rngMu sync.Mutex
	rng   *mrand.Rand

	// store: cid → (readyAt, payload). The payload is just sha256(cid)
	// expanded to the original size — we don't need fidelity, only that
	// Get returns the same bytes for the same CID.
	storeMu sync.Mutex
	store   map[string]simEntry
}

type simEntry struct {
	readyAt   time.Time
	sizeBytes int
}

// ErrNotReady is returned by Get when called before propagation completes.
// Callers (the polling loop) treat this as a normal "not yet" signal and
// retry; it is *not* a permanent error.
var ErrNotReady = errors.New("simTTA: bundle not yet propagated")

// NewSimTTA returns a sim-mode tier with the given calibration tag.
//
// Each call to Put samples both put_latency and propagation_delay.
// put_latency gates how long Put blocks; propagation_delay determines
// when subsequent Get calls succeed (relative to Put-return time).
func NewSimTTA(name string, tierClass uint8, putDist, propDist lnPair, seed int64) *simTTA {
	return &simTTA{
		name:      name,
		tierClass: tierClass,
		putDist:   putDist,
		propDist:  propDist,
		rng:       mrand.New(mrand.NewSource(seed)),
		store:     map[string]simEntry{},
	}
}

func (s *simTTA) Name() string      { return s.name }
func (s *simTTA) TierClass() uint8  { return s.tierClass }

// Put blocks for a sampled put_latency, then returns. The CID is a
// deterministic hash of the input — same content → same CID, matching
// the IPFS contract. After Put returns, the bundle is *registered* but
// not yet *reachable*; Get returns ErrNotReady until the propagation
// timer elapses.
func (s *simTTA) Put(ctx context.Context, data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("simTTA: empty payload")
	}
	s.rngMu.Lock()
	putNs := s.putDist.sample(s.rng)
	propNs := s.propDist.sample(s.rng)
	s.rngMu.Unlock()

	// Block for put_latency. The propagation timer starts the moment
	// Put returns — readyAt = (now + put) + prop = "(when Put returns)
	// + prop", which we capture just before unblocking.
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(time.Duration(putNs)):
	}

	digest := sha256.Sum256(data)
	cid := "sim:" + hex.EncodeToString(digest[:])

	readyAt := time.Now().Add(time.Duration(propNs))
	s.storeMu.Lock()
	s.store[cid] = simEntry{readyAt: readyAt, sizeBytes: len(data)}
	s.storeMu.Unlock()

	return cid, nil
}

// Get returns the bundle if propagation has completed, or ErrNotReady.
// The bytes are deterministic from the CID (cheaper than retaining the
// original payload — the bench discards it after Put).
func (s *simTTA) Get(ctx context.Context, cid string) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	s.storeMu.Lock()
	entry, ok := s.store[cid]
	s.storeMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("simTTA: cid %s not found", cid)
	}
	if time.Now().Before(entry.readyAt) {
		return nil, ErrNotReady
	}
	// Synthesise payload deterministically from CID.
	out := make([]byte, entry.sizeBytes)
	digest := sha256.Sum256([]byte(cid))
	for i := 0; i < entry.sizeBytes; i++ {
		out[i] = digest[i%len(digest)]
	}
	return out, nil
}

// Stat is a cheap "is it in the store" probe — does NOT require
// propagation completion. Used by the polling loop's fast-path checks
// when Get would be wasteful.
func (s *simTTA) Stat(ctx context.Context, cid string) (*tiers.Stat, error) {
	s.storeMu.Lock()
	entry, ok := s.store[cid]
	s.storeMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("simTTA: cid %s not found", cid)
	}
	return &tiers.Stat{
		CID:       cid,
		SizeBytes: int64(entry.sizeBytes),
		StoredAt:  entry.readyAt.Add(-time.Duration(s.propDist.muNs)),
	}, nil
}

// Delete is a no-op success — the bench never deletes bundles, but the
// tiers.Client interface requires the method.
func (s *simTTA) Delete(ctx context.Context, cid string) error {
	s.storeMu.Lock()
	delete(s.store, cid)
	s.storeMu.Unlock()
	return nil
}
