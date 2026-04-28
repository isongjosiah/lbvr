// Cross-tier recovery state machine — heart of the LBVR-Med D8 gateway
// (CLAUDE.md §4.3 / §4.5; docs/erasure-design.md §6).
//
// Three goroutines fan GETs across {hot, warm, cold} in parallel, each
// with an independent cancellable context. As soon as both data shards
// (D0+D1) arrive within the SLO budget the cold-tier context is cancelled
// and we return without erasure decode (fast path). Otherwise we wait for
// any 2 of 3, run erasure.Decode, and report slow path. If all 3 contexts
// terminate with <2 successes, we return RecoveryFailure.
//
// Context discipline matters here: a leaked goroutine = a leaked HTTP
// connection in production. Every per-shard context is cancelled on every
// exit path; the parent ctx still bounds total runtime.

// Package gateway holds the retrieval-path recovery state machine
// extracted from cmd/gateway so the bench harness can import it without
// duplicating the logic it intends to measure.
package gateway

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/erasure"
	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

// RecoveryMode names which logical path the gateway took for one bundle.
type RecoveryMode uint8

const (
	// RecoveryFastPath: D0+D1 returned within sloBudget; P0 unused.
	RecoveryFastPath RecoveryMode = iota
	// RecoverySlowPath: any 2 of 3 used; required reconstruction.
	RecoverySlowPath
	// RecoveryFailure: <2 shards available within parent ctx.
	RecoveryFailure
)

// String renders the mode for headers/logs.
func (m RecoveryMode) String() string {
	switch m {
	case RecoveryFastPath:
		return "fast"
	case RecoverySlowPath:
		return "slow"
	case RecoveryFailure:
		return "failure"
	default:
		return "unknown"
	}
}

// NotReturned sentinel for ShardLatencies entries that never produced a
// final state (parent ctx died first). We use -1 instead of zero so the
// distinction between "instant return" and "no return" is unambiguous.
const NotReturned time.Duration = -1

// RecoveryStats reports per-shard outcome. Decode timing is zero when the
// fast path skipped reconstruction.
type RecoveryStats struct {
	Mode           RecoveryMode
	ShardLatencies [3]time.Duration // -1 = not returned
	ShardErrors    [3]error
	DecodeNanos    int64
}

// shardResult is the channel payload one fetcher sends.
type shardResult struct {
	idx     int
	data    []byte
	err     error
	latency time.Duration
}

// Recover fans GETs across all 3 tiers in parallel, picks the cheapest
// viable reconstruction path, and returns the encrypted-bundle bytes
// (still AES-GCM-sealed; the gateway decrypts after Recover returns).
//
// sloBudget is the fast-path budget. If D0 and D1 both arrive within
// sloBudget, Recover returns immediately without waiting for P0 (cancels
// the cold-tier context). Otherwise it waits for any 2 of 3 to arrive,
// then erasure-decodes.
//
// Decision: a fast-path candidate is "in time" iff its arrival time
// (now()-start) is <= sloBudget. If D0 returns at SLO+1ms and D1 at
// SLO+2ms, the fast-path window has expired — we still take the cheaper
// path (no decode) but mark it RecoverySlowPath because we exceeded the
// SLO; the gateway uses this header to alert on SLO breaches.
//
// Note on ordering: a fast cold-tier arrival does NOT short-circuit a
// pending data-shard. If P0 arrives first with D0 in flight, we keep
// waiting for D0+D1 until either both arrive (fast path) or sloBudget
// elapses (commit to slow path). Otherwise a fast cold tier would
// pollute every retrieval onto the decode path and obscure the SLO.
func Recover(
	ctx context.Context,
	tiers [3]tiers.Client,
	cids [3]string,
	paddedLen int,
	sloBudget time.Duration,
) (encrypted []byte, stats RecoveryStats, err error) {
	stats = RecoveryStats{
		ShardLatencies: [3]time.Duration{NotReturned, NotReturned, NotReturned},
	}
	if paddedLen <= 0 {
		return nil, stats, fmt.Errorf("gateway: invalid paddedLen %d", paddedLen)
	}

	// Per-shard cancellable contexts so the fast path can drop the cold
	// fetch immediately. All derive from ctx so the parent kill-switch
	// still propagates.
	var (
		sctxs    [3]context.Context
		scancels [3]context.CancelFunc
	)
	for i := 0; i < 3; i++ {
		sctxs[i], scancels[i] = context.WithCancel(ctx)
	}
	// Defensive: if we exit early on parent-ctx cancel, make sure every
	// per-shard goroutine receives the cancel signal too.
	defer func() {
		for i := 0; i < 3; i++ {
			scancels[i]()
		}
	}()

	results := make(chan shardResult, 3)
	start := time.Now()

	// Launch one fetcher per tier. Each fetcher is responsible for
	// emitting exactly one shardResult — even on context cancellation —
	// so the consumer's "received N" count is reliable.
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, gErr := tiers[i].Get(sctxs[i], cids[i])
			results <- shardResult{
				idx:     i,
				data:    data,
				err:     gErr,
				latency: time.Since(start),
			}
		}()
	}
	// Drain in the background once we leave; channel stays open long
	// enough for cancelled goroutines to flush their result.
	go func() { wg.Wait(); close(results) }()

	// Collect arrivals. State transitions:
	//
	//   (a) D0+D1 both succeeded → fast path (no decode); cancel P0.
	//   (b) Any 2 of {D0,D1,P0} succeeded AND the SLO budget has elapsed
	//       (or the third shard already failed) → slow path with decode.
	//   (c) All 3 results received with <2 successes → failure.
	//
	// Crucially, when only D0+P0 (or D1+P0) have arrived but the third
	// outstanding shard is still in flight under sloBudget, we keep
	// waiting — it might be a fast-path candidate. Without this rule a
	// quick cold tier would force every retrieval onto the slow path even
	// when D0+D1 are seconds away.
	var (
		shards   [3][]byte
		gotOK    [3]bool
		got      [3]bool // received result (success or error)
		received int
	)
	const maxResults = 3

	// Helper: would waiting longer plausibly let us reach fast path?
	canStillReachFast := func() bool {
		// Fast path needs both data shards succeeded.
		if gotOK[0] && gotOK[1] {
			return false // already there or past it
		}
		// If either data shard has already returned an error, fast path
		// is impossible regardless of how long we wait.
		if got[0] && !gotOK[0] {
			return false
		}
		if got[1] && !gotOK[1] {
			return false
		}
		return true
	}

	// sloTimer fires once the fast-path window expires; after that, two
	// arrivals in any combination are enough to commit to slow path.
	sloTimer := time.NewTimer(sloBudget)
	defer sloTimer.Stop()
	sloElapsed := false

	for received < maxResults {
		select {
		case <-ctx.Done():
			if okCount(gotOK) >= 2 {
				return finishSlow(shards, paddedLen, &stats)
			}
			stats.Mode = RecoveryFailure
			return nil, stats, ctx.Err()

		case <-sloTimer.C:
			sloElapsed = true
			// If we already have 2 of 3, commit to slow path now.
			if okCount(gotOK) >= 2 {
				cancelOutstanding(scancels, got)
				return finishSlowOrLateFast(shards, paddedLen, &stats)
			}
			// Otherwise keep waiting — we need at least 2 successes.

		case r, ok := <-results:
			if !ok {
				if okCount(gotOK) >= 2 {
					return finishSlowOrLateFast(shards, paddedLen, &stats)
				}
				stats.Mode = RecoveryFailure
				return nil, stats, errors.New("gateway: no shards returned")
			}
			received++
			got[r.idx] = true
			stats.ShardLatencies[r.idx] = r.latency
			stats.ShardErrors[r.idx] = r.err
			if r.err == nil && r.data != nil {
				shards[r.idx] = r.data
				gotOK[r.idx] = true
			}

			// Fast path: both data shards succeeded.
			if gotOK[0] && gotOK[1] {
				slower := stats.ShardLatencies[0]
				if stats.ShardLatencies[1] > slower {
					slower = stats.ShardLatencies[1]
				}
				scancels[2]() // we won't need P0
				if slower <= sloBudget {
					return finishFast(shards, paddedLen, &stats)
				}
				// Both data shards in but past SLO — still no decode,
				// but report slow-path so the SLO-breach signal fires.
				return finishFastButLate(shards, paddedLen, &stats)
			}

			// Slow path commit: we have 2 of 3 successes AND either the
			// SLO window has elapsed OR fast path is no longer reachable
			// (one of the data shards has erroneously returned). In
			// either case, waiting longer cannot improve outcome.
			if okCount(gotOK) >= 2 && (sloElapsed || !canStillReachFast()) {
				cancelOutstanding(scancels, got)
				return finishSlowOrLateFast(shards, paddedLen, &stats)
			}

			// Early failure: if even with the still-outstanding shard
			// succeeding we cannot reach 2 successes, fail immediately.
			// Without this, a double-data-shard failure (D0+D1 both
			// errored) would still wait the full cold-tier latency
			// budget for P0 — measured P99 of 8s on the calibrated
			// lognormal — before returning RecoveryFailure. This is
			// what makes the §8 E9-multi "detection time" measurement
			// useful: detection becomes max(error_time_D0, error_time_D1)
			// rather than max(_, _, cold_tier_latency).
			if okCount(gotOK)+(maxResults-received) < 2 {
				cancelOutstanding(scancels, got)
				stats.Mode = RecoveryFailure
				return nil, stats, errors.New("gateway: insufficient shards (early detection)")
			}
		}
	}

	// All 3 results consumed; if we reach here we did not have ≥2
	// successes (otherwise we'd have returned inside the loop).
	stats.Mode = RecoveryFailure
	return nil, stats, errors.New("gateway: insufficient shards for reconstruction")
}

// cancelOutstanding cancels the per-shard contexts of any shard whose
// result has not yet been delivered. The fetcher goroutine still emits
// one shardResult after cancel propagates, so the channel stays
// drainable.
func cancelOutstanding(cancels [3]context.CancelFunc, got [3]bool) {
	for i := 0; i < 3; i++ {
		if !got[i] {
			cancels[i]()
		}
	}
}

// finishSlowOrLateFast picks between two finish modes when we have 2
// successes after the SLO window: if the two are D0+D1 we can join
// directly (no decode), otherwise we must RS-decode. Either way the
// reported mode is slow because the SLO budget was already exceeded.
func finishSlowOrLateFast(shards [3][]byte, paddedLen int, stats *RecoveryStats) ([]byte, RecoveryStats, error) {
	if shards[0] != nil && shards[1] != nil {
		return finishFastButLate(shards, paddedLen, stats)
	}
	return finishSlow(shards, paddedLen, stats)
}

// finishFast: D0+D1 within SLO; concatenate and skip decode.
func finishFast(shards [3][]byte, paddedLen int, stats *RecoveryStats) ([]byte, RecoveryStats, error) {
	out, err := joinDataShards(shards[0], shards[1], paddedLen)
	if err != nil {
		// Shard sizes inconsistent → fall back to RS decode which validates.
		return finishSlow(shards, paddedLen, stats)
	}
	stats.Mode = RecoveryFastPath
	return out, *stats, nil
}

// finishFastButLate: same shape as fast path (D0+D1 present, no decode
// needed) but the slower of the two missed sloBudget. Reported as slow
// so the SLO-breach signal isn't masked.
func finishFastButLate(shards [3][]byte, paddedLen int, stats *RecoveryStats) ([]byte, RecoveryStats, error) {
	out, err := joinDataShards(shards[0], shards[1], paddedLen)
	if err != nil {
		return finishSlow(shards, paddedLen, stats)
	}
	stats.Mode = RecoverySlowPath
	return out, *stats, nil
}

// finishSlow: invoke RS(2,3) reconstruct from any 2 of 3 surviving shards.
func finishSlow(shards [3][]byte, paddedLen int, stats *RecoveryStats) ([]byte, RecoveryStats, error) {
	t0 := time.Now()
	out, err := erasure.Decode(shards, paddedLen)
	stats.DecodeNanos = time.Since(t0).Nanoseconds()
	if err != nil {
		stats.Mode = RecoveryFailure
		return nil, *stats, fmt.Errorf("gateway: erasure decode: %w", err)
	}
	stats.Mode = RecoverySlowPath
	return out, *stats, nil
}

// joinDataShards concatenates D0+D1 and trims to paddedLen. Both shards
// must be the same length (RS(2,3) invariant) and paddedLen must lie in
// the half-open interval ((shardLen-1)*2, shardLen*2] — same check the
// erasure package runs.
func joinDataShards(d0, d1 []byte, paddedLen int) ([]byte, error) {
	if len(d0) != len(d1) {
		return nil, fmt.Errorf("gateway: shard size mismatch %d vs %d", len(d0), len(d1))
	}
	shardLen := len(d0)
	if paddedLen <= 0 || paddedLen > shardLen*2 || paddedLen <= (shardLen-1)*2 {
		return nil, fmt.Errorf("gateway: paddedLen %d inconsistent with shard size %d", paddedLen, shardLen)
	}
	out := make([]byte, paddedLen)
	copy(out, d0)
	if paddedLen > shardLen {
		copy(out[shardLen:], d1[:paddedLen-shardLen])
	}
	return out, nil
}

func okCount(b [3]bool) int {
	n := 0
	for _, v := range b {
		if v {
			n++
		}
	}
	return n
}
