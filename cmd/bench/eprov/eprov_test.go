package main

import (
	"context"
	"encoding/json"
	mrand "math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/provenance"
)

// TestRun_Smoke runs the bench with n=5 / seed=1 and asserts the JSON
// landed and parses. Catches wiring drift (schema typos, output dir
// missing, signOne returning errors on the happy iter).
func TestRun_Smoke(t *testing.T) {
	outDir := t.TempDir()
	if err := run(5, 1, outDir); err != nil {
		t.Fatalf("run: %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("read outDir: %v", err)
	}
	var foundRun, foundEnv bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "run-") && strings.HasSuffix(e.Name(), ".json") {
			foundRun = true
			b, err := os.ReadFile(filepath.Join(outDir, e.Name()))
			if err != nil {
				t.Fatalf("read run-*.json: %v", err)
			}
			var rec runRecord
			if err := json.Unmarshal(b, &rec); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if rec.SchemaVersion != 1 {
				t.Errorf("schema_version = %d, want 1", rec.SchemaVersion)
			}
			if len(rec.Samples) != 5 {
				t.Errorf("samples = %d, want 5", len(rec.Samples))
			}
			for _, s := range rec.Samples {
				if s.TTotalNS <= 0 {
					t.Errorf("iter %d: t_total_ns = %d, want > 0", s.Iter, s.TTotalNS)
				}
				if s.BytesDoc <= 0 {
					t.Errorf("iter %d: bytes_doc = %d, want > 0", s.Iter, s.BytesDoc)
				}
			}
		}
		if e.Name() == "env.json" {
			foundEnv = true
			b, err := os.ReadFile(filepath.Join(outDir, e.Name()))
			if err != nil {
				t.Fatalf("read env.json: %v", err)
			}
			var env envJSON
			if err := json.Unmarshal(b, &env); err != nil {
				t.Fatalf("unmarshal env.json: %v", err)
			}
			if env.BenchID != "E-PROV" {
				t.Errorf("bench_id = %q, want E-PROV", env.BenchID)
			}
		}
	}
	if !foundRun {
		t.Error("no run-*.json written")
	}
	if !foundEnv {
		t.Error("no env.json written")
	}
}

// TestTamperingCases_AllCaught exercises each non-happy tampering mode
// individually and asserts the verifier rejects the document. Catches
// regressions in StripSigField / Canonicalize / Verify wiring.
func TestTamperingCases_AllCaught(t *testing.T) {
	rng := mrand.New(mrand.NewSource(7))
	kp1, err := provenance.GenerateKey()
	if err != nil {
		t.Fatalf("kp1: %v", err)
	}
	kp2, err := provenance.GenerateKey()
	if err != nil {
		t.Fatalf("kp2: %v", err)
	}
	gw1 := provenance.GatewayAgent{
		ProvType: "prov:SoftwareAgent", Role: "retrieval_gateway",
		Version: "lbvr-med-test", PublicKey: hexPrefixed(kp1.PublicBytes[:]),
	}
	gw2 := provenance.GatewayAgent{
		ProvType: "prov:SoftwareAgent", Role: "retrieval_gateway",
		Version: "lbvr-med-test", PublicKey: hexPrefixed(kp2.PublicBytes[:]),
	}
	did1 := "did:lbvr:" + safeIDFragment(gw1.PublicKey[2:10])
	did2 := "did:lbvr:" + safeIDFragment(gw2.PublicKey[2:10])
	keys := provenance.StaticKeyResolver{
		did1: kp1.PublicBytes,
		did2: kp2.PublicBytes,
	}
	anchor := newMockAnchor(7)

	// Each non-happy case must be detected. Iter index doesn't matter
	// for assertions — the case string drives the tamper.
	cases := []string{
		"hash_tamper",
		"sig_tamper",
		"signer_substitute",
		"quorum_reduce",
		"missing_sig",
		"timestamp_tamper",
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			stat, err := signOne(0, c, rng, gw1, gw2, did1, did2, keys, kp1.PrivateBytes, kp2.PrivateBytes, anchor)
			if err != nil {
				t.Fatalf("signOne %s: %v", c, err)
			}
			if stat.Valid {
				t.Fatalf("case %s: expected Valid=false, got valid (no detection)", c)
			}
			if !stat.TamperCaught {
				t.Errorf("case %s: TamperCaught=false but Valid=false (taxonomy bug)", c)
			}
			// Spot-check expected reasons (matches verifier.go sentinels).
			wantReason := map[string]string{
				"hash_tamper":       "hash_mismatch",
				"sig_tamper":        "signature_invalid",
				"signer_substitute": "unknown_signer",
				"quorum_reduce":     "insufficient_quorum",
				"missing_sig":       "missing_sig",
				"timestamp_tamper":  "hash_mismatch",
			}
			if want := wantReason[c]; want != "" && stat.FailureReason != want {
				t.Errorf("case %s: failure_reason = %q, want %q", c, stat.FailureReason, want)
			}
		})
	}

	// Happy must verify clean.
	stat, err := signOne(0, "happy", rng, gw1, gw2, did1, did2, keys, kp1.PrivateBytes, kp2.PrivateBytes, anchor)
	if err != nil {
		t.Fatalf("signOne happy: %v", err)
	}
	if !stat.Valid {
		t.Errorf("happy: expected Valid=true, got reason=%q", stat.FailureReason)
	}
	if !stat.TamperCaught {
		// In happy semantics TamperCaught == Valid (verified clean).
		t.Errorf("happy: TamperCaught=false despite Valid=true (taxonomy bug)")
	}
}

// TestSampleLatency_NonNegative draws 1000 samples from the lognormal
// anchor distribution; all results must be > 0. Guards against future
// recalibration that accidentally makes mu+sigma small enough to round
// to zero (the time.Duration conversion floors at 0 if ns < 1).
func TestSampleLatency_NonNegative(t *testing.T) {
	a := newMockAnchor(99)
	for i := 0; i < 1000; i++ {
		d := a.sampleLatency()
		if d <= 0 {
			t.Fatalf("sample %d: got %v, want > 0", i, d)
		}
		// Sanity: no sample should exceed 60s for these parameters.
		// P99 is 200ms; even rare tail draws shouldn't blow past a
		// minute. If they do, mu/sigma is misconfigured.
		if d > time.Minute {
			t.Errorf("sample %d: implausibly large latency %v", i, d)
		}
	}

	// Verify Anchor itself respects the lognormal wait. We use a
	// short timeout to confirm the wait is observable; the timeout
	// must be > P99 so happy-path anchors don't flake.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var bid, rid, hash [32]byte
	hash[0] = 1 // non-zero
	bid[0] = 2
	rid[0] = 3
	rec, err := a.Anchor(ctx, bid, rid, hash)
	if err != nil {
		t.Fatalf("Anchor: %v", err)
	}
	if rec.BlockNumber == 0 {
		t.Errorf("BlockNumber = 0 — should be ≥ 1")
	}
}
