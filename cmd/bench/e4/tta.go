package main

// Per-bundle PUT + polling-GET measurement. The harness runs once per
// (bundle, tier) tuple; the per-tier results land side-by-side in the
// JSON output so Fig 5 can render three CDFs (one per tier).
//
// Polling cadence is logarithmic: dense early to capture sub-second
// availability (clinical SLO budget), sparse later because cold-tier
// settlement can stretch into minutes. The cadence is the same for
// sim and live runs; sim uses synthetic propagation_delay sampling and
// returns the analytically-correct "first reachable" time, but the
// polling-curve output is preserved for parity with future live runs.

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

// pollCheckpoints is the canonical polling schedule. Each entry is the
// ms-since-PUT-return at which we attempt a Get. The shape covers
// 100ms → 5min, dense early. The list MUST be monotonically increasing.
var pollCheckpoints = []time.Duration{
	100 * time.Millisecond,
	250 * time.Millisecond,
	500 * time.Millisecond,
	1 * time.Second,
	2 * time.Second,
	5 * time.Second,
	10 * time.Second,
	30 * time.Second,
	60 * time.Second,
	120 * time.Second,
	300 * time.Second,
}

// tierResult is one (bundle, tier) measurement. CIDs are not retained —
// the post-processor only cares about timing.
type tierResult struct {
	Tier             string  `json:"tier"`
	PutLatencyMs     int64   `json:"put_latency_ms"`
	FirstOKAtMs      int64   `json:"first_ok_at_ms"`     // ms after PUT return
	FirstOKCheckpoint int64  `json:"first_ok_checkpoint_ms"` // the checkpoint at which we observed first OK
	TimedOut         bool    `json:"timed_out"`           // never reached Get-OK within the polling budget
	LiveMode         bool    `json:"live_mode"`           // true = real Get polling, false = sim-mode synthetic
}

// runOneBundleTier issues a Put + polling-Get loop against `client`
// for one bundle. The result captures (put latency, first-OK time, and
// whether we timed out within the polling budget).
//
// `payload` is supplied by the caller — the harness synthesises random
// bytes once per bundle and feeds the same payload to every tier so the
// per-tier comparison is bundle-fair.
func runOneBundleTier(
	ctx context.Context,
	tierName string,
	client tiers.Client,
	payload []byte,
	liveMode bool,
) (tierResult, error) {
	res := tierResult{Tier: tierName, LiveMode: liveMode}

	// --- Put leg ---------------------------------------------------------
	putStart := time.Now()
	cid, err := client.Put(ctx, payload)
	res.PutLatencyMs = time.Since(putStart).Milliseconds()
	if err != nil {
		return res, fmt.Errorf("Put: %w", err)
	}
	if cid == "" {
		return res, errors.New("Put returned empty cid")
	}

	// putReturnedAt is the time PUT returned. All polling latencies are
	// reported relative to this instant.
	putReturnedAt := time.Now()

	// --- Polling-Get leg -------------------------------------------------
	for _, cp := range pollCheckpoints {
		// Sleep until the next checkpoint. We measure relative to
		// putReturnedAt, not relative to the previous checkpoint, so a
		// slow Get doesn't compound into the next interval.
		target := putReturnedAt.Add(cp)
		now := time.Now()
		if target.After(now) {
			select {
			case <-ctx.Done():
				return res, ctx.Err()
			case <-time.After(target.Sub(now)):
			}
		}

		_, err := client.Get(ctx, cid)
		switch {
		case err == nil:
			// First success. Record the actual elapsed time (not the
			// nominal checkpoint), so we capture sub-checkpoint
			// resolution if the implementation has it.
			res.FirstOKAtMs = time.Since(putReturnedAt).Milliseconds()
			res.FirstOKCheckpoint = cp.Milliseconds()
			return res, nil
		case errors.Is(err, ErrNotReady):
			// Sim path's "not yet propagated"; keep polling.
			continue
		default:
			// Live path: a 404 / timeout / network error usually means
			// "not yet". We don't fail the run — TTA polling is
			// inherently retryable. A persistent error will time out
			// at the last checkpoint and surface via TimedOut.
			continue
		}
	}

	res.TimedOut = true
	res.FirstOKAtMs = pollCheckpoints[len(pollCheckpoints)-1].Milliseconds()
	res.FirstOKCheckpoint = res.FirstOKAtMs
	return res, nil
}

// synthesisePayload draws random bytes for one bundle. Reused across
// the three tiers per bundle so per-tier latencies are comparable.
func synthesisePayload(sizeBytes int64) ([]byte, error) {
	if sizeBytes <= 0 {
		return nil, fmt.Errorf("synthesisePayload: non-positive size %d", sizeBytes)
	}
	out := make([]byte, sizeBytes)
	if _, err := rand.Read(out); err != nil {
		return nil, fmt.Errorf("rand.Read: %w", err)
	}
	return out, nil
}
