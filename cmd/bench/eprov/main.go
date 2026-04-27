// Package main is the E-PROV cryptographic-provenance bench harness.
//
// CLAUDE.md §4.6 / §8 row "E-PROV"; docs/provenance-spec.md §8. Measures
// per-stage latency for the full provenance lifecycle on synthesised
// retrieval events:
//
//	gen     — provenance.Generate (build PROV-JSON struct from inputs)
//	sign    — provenance.(*Document).Sign (BLS pairings × N signed nodes)
//	canon   — provenance.(*Document).Marshal (json + JCS canonicalisation)
//	anchor  — mockAnchor.Anchor (calibrated lognormal latency, see anchor.go)
//	setroot — provenance.(*Document).SetRoot + final Marshal
//	verify  — provenance.Verifier.Verify (parse + canon + sha256 + 2× BLS verify)
//
// Round-robin tampering scheme: each iteration picks one of {happy,
// hash_tamper, sig_tamper, signer_substitute, quorum_reduce, missing_sig,
// timestamp_tamper}; happy verifies clean, the others mutate the
// canonical doc (or the SignatureBlock) and re-verify, recording whether
// the verifier flagged it. Detection rate per case lands in the JSON +
// post-processor table.
//
// Realism notes:
//   - 2 ephemeral BLS keys are reused across all iterations — mirrors a
//     deployed gateway with stable quorum keys, NOT a per-retrieval
//     keygen cost. Keygen latency is out of scope for this experiment.
//   - The mock anchor is local; t_anchor reflects calibrated round-trip
//     latency (anchor.go), not a live RPC. env.json names the model.
//   - Inputs are synthesised, not drawn from real bundles. The provenance
//     payload is constant-shape across bundle sizes (it carries shard
//     metadata, not bundle bytes), so size sweep is unnecessary for
//     E-PROV — a single-shape sample is sufficient. (Contrast with E9,
//     where bundle bytes drive shard size and therefore latency.)

package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	mrand "math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/provenance"
)

// tamperingCases is the round-robin order. "happy" must be first so the
// 0th iteration is a clean baseline (smoke for the rest of the run).
var tamperingCases = []string{
	"happy",
	"hash_tamper",
	"sig_tamper",
	"signer_substitute",
	"quorum_reduce",
	"missing_sig",
	"timestamp_tamper",
}

// runRecord is the top-level JSON shape (schema_version=1).
type runRecord struct {
	SchemaVersion int          `json:"schema_version"`
	RunID         string       `json:"run_id"`
	StartedAt     string       `json:"started_at"`
	CompletedAt   string       `json:"completed_at"`
	Config        runConfig    `json:"config"`
	Samples       []sampleStat `json:"samples"`
}

type runConfig struct {
	N               int     `json:"n"`
	Seed            int64   `json:"seed"`
	AnchorModel     string  `json:"anchor_model"`
	KeyCount        int     `json:"key_count"`
	QuorumThreshold int     `json:"quorum_threshold"`
	AnchorMuNs      float64 `json:"anchor_mu_ns"`
	AnchorSigma     float64 `json:"anchor_sigma"`
	AnchorP50ms     int     `json:"anchor_p50_ms"`
	AnchorP99ms     int     `json:"anchor_p99_ms"`
}

// sampleStat is one iteration's record. All times are in ns; bytes are
// raw byte counts (post-canonicalisation since Marshal returns canonical
// bytes).
type sampleStat struct {
	Iter           int    `json:"iter"`
	Tampering      string `json:"tampering"`
	TamperCaught   bool   `json:"tamper_caught"`
	FailureReason  string `json:"failure_reason"`
	Valid          bool   `json:"valid"`
	BytesPreAnchor int    `json:"bytes_pre_anchor"`
	BytesDoc       int    `json:"bytes_doc"`
	TGenNS         int64  `json:"t_gen_ns"`
	TSignNS        int64  `json:"t_sign_ns"`
	TCanonNS       int64  `json:"t_canon_ns"`
	TAnchorNS      int64  `json:"t_anchor_ns"`
	TSetRootNS     int64  `json:"t_setroot_ns"`
	TVerifyNS      int64  `json:"t_verify_ns"`
	TTotalNS       int64  `json:"t_total_ns"`
}

// envJSON mirrors CLAUDE.md §8 environment-fingerprint contract plus
// the bench-specific bench_id / anchor calibration.
type envJSON struct {
	CommitHash  string `json:"commit_hash"`
	GoVersion   string `json:"go_version"`
	OS          string `json:"os"`
	Kernel      string `json:"kernel"`
	CPU         string `json:"cpu"`
	NetworkPath string `json:"network_path"`
	WallStart   string `json:"wall_start"`
	WallEnd     string `json:"wall_end"`
	BenchID     string `json:"bench_id"`
	AnchorModel string `json:"anchor_model"`
}

func main() {
	var (
		nFlag      = flag.Int("n", 1000, "number of iterations")
		seedFlag   = flag.Int64("seed", 42, "RNG seed (input synth + anchor stream)")
		outDirFlag = flag.String("out-dir", "eval/results/E-PROV", "output directory")
	)
	flag.Parse()

	if err := run(*nFlag, *seedFlag, *outDirFlag); err != nil {
		log.Fatalf("bench-E-PROV: %v", err)
	}
}

func run(n int, seed int64, outDir string) error {
	wallStart := time.Now().UTC()
	commit := shortCommit()
	runID := wallStart.Format("20060102-150405") + "-" + commit

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	// Two ephemeral BLS keys, reused across iterations. GenerateKey
	// uses crypto/rand internally; we record nothing about their bytes
	// in the JSON.
	kp1, err := provenance.GenerateKey()
	if err != nil {
		return fmt.Errorf("keygen 1: %w", err)
	}
	kp2, err := provenance.GenerateKey()
	if err != nil {
		return fmt.Errorf("keygen 2: %w", err)
	}
	gw1 := provenance.GatewayAgent{
		ProvType: "prov:SoftwareAgent", Role: "retrieval_gateway",
		Version: "lbvr-med-bench", PublicKey: hexPrefixed(kp1.PublicBytes[:]),
	}
	gw2 := provenance.GatewayAgent{
		ProvType: "prov:SoftwareAgent", Role: "retrieval_gateway",
		Version: "lbvr-med-bench", PublicKey: hexPrefixed(kp2.PublicBytes[:]),
	}
	did1 := "did:lbvr:" + safeIDFragment(gw1.PublicKey[2:10])
	did2 := "did:lbvr:" + safeIDFragment(gw2.PublicKey[2:10])

	keys := provenance.StaticKeyResolver{
		did1: kp1.PublicBytes,
		did2: kp2.PublicBytes,
	}

	// Bench-private RNG for input synthesis. anchor stream uses seed+1
	// so the two streams are disjoint — mirrors the per-tier offset
	// idiom in cmd/bench/e9.
	inputRNG := mrand.New(mrand.NewSource(seed))
	anchor := newMockAnchor(seed + 1)

	samples := make([]sampleStat, 0, n)
	tamperCounts := map[string]int{}
	tamperCaughtCounts := map[string]int{}

	for i := 0; i < n; i++ {
		tCase := tamperingCases[i%len(tamperingCases)]
		tamperCounts[tCase]++

		stat, err := signOne(i, tCase, inputRNG, gw1, gw2, did1, did2, keys, kp1.PrivateBytes, kp2.PrivateBytes, anchor)
		if err != nil {
			return fmt.Errorf("iter %d (%s): %w", i, tCase, err)
		}
		if stat.TamperCaught {
			tamperCaughtCounts[tCase]++
		}
		samples = append(samples, stat)

		if (i+1)%100 == 0 || i == n-1 {
			log.Printf("bench-E-PROV: %d/%d iters done (%.1f%%)", i+1, n, 100.0*float64(i+1)/float64(n))
		}
	}

	wallEnd := time.Now().UTC()

	rec := runRecord{
		SchemaVersion: 1,
		RunID:         runID,
		StartedAt:     wallStart.Format(time.RFC3339Nano),
		CompletedAt:   wallEnd.Format(time.RFC3339Nano),
		Config: runConfig{
			N:               n,
			Seed:            seed,
			AnchorModel:     "mock-immediate (calibrated lognormal: 30ms median, 200ms P99 — see cmd/bench/eprov/anchor.go)",
			KeyCount:        2,
			QuorumThreshold: 2,
			AnchorMuNs:      anchorMu,
			AnchorSigma:     anchorSigma,
			AnchorP50ms:     anchorP50ms,
			AnchorP99ms:     anchorP99ms,
		},
		Samples: samples,
	}

	runJSONPath := filepath.Join(outDir, "run-"+runID+".json")
	if err := writeJSON(runJSONPath, rec); err != nil {
		return err
	}
	log.Printf("bench-E-PROV: wrote %s", runJSONPath)

	envPath := filepath.Join(outDir, "env.json")
	if err := writeJSON(envPath, envJSON{
		CommitHash:  commit,
		GoVersion:   runtime.Version(),
		OS:          runtime.GOOS,
		Kernel:      unameOrEmpty(),
		CPU:         cpuOrEmpty(),
		NetworkPath: "N/A — local sim",
		WallStart:   wallStart.Format(time.RFC3339Nano),
		WallEnd:     wallEnd.Format(time.RFC3339Nano),
		BenchID:     "E-PROV",
		AnchorModel: rec.Config.AnchorModel,
	}); err != nil {
		return err
	}
	log.Printf("bench-E-PROV: wrote %s", envPath)

	printSummary(rec, tamperCounts, tamperCaughtCounts, wallEnd.Sub(wallStart))
	return nil
}

// signOne measures one iteration's stages and runs the chosen tampering.
// Returns a fully populated sampleStat. The sealing of t_anchor inside
// a context.Background() is intentional: we don't want a stray ctx
// cancel during a benchmark to terminate the lognormal wait early and
// underreport latency. Anchor failures are surfaced as iteration errors.
// See file-level comment for stage breakdown.
func signOne(
	iter int,
	tCase string,
	rng *mrand.Rand,
	gw1, gw2 provenance.GatewayAgent,
	did1, did2 string,
	keys provenance.StaticKeyResolver,
	priv1, priv2 [provenance.PrivateKeySize]byte,
	anchor *mockAnchor,
) (sampleStat, error) {
	in := synthInput(rng, gw1, gw2)
	stat := sampleStat{Iter: iter, Tampering: tCase}

	// 1. Generate.
	t0 := time.Now()
	doc, err := provenance.Generate(in)
	stat.TGenNS = time.Since(t0).Nanoseconds()
	if err != nil {
		return stat, fmt.Errorf("Generate: %w", err)
	}

	// 2. Sign.
	t0 = time.Now()
	if err := doc.Sign(
		[][provenance.PrivateKeySize]byte{priv1, priv2},
		[]string{did1, did2},
		2,
	); err != nil {
		return stat, fmt.Errorf("Sign: %w", err)
	}
	stat.TSignNS = time.Since(t0).Nanoseconds()

	// 3. Canon (Marshal returns JCS-canonical bytes). bytes_pre_anchor
	// is the size of the doc with no provenanceRoot embedded — what
	// the gateway hashes for the on-chain anchor.
	t0 = time.Now()
	preAnchor, err := doc.Marshal()
	stat.TCanonNS = time.Since(t0).Nanoseconds()
	if err != nil {
		return stat, fmt.Errorf("Marshal pre-anchor: %w", err)
	}
	stat.BytesPreAnchor = len(preAnchor)

	// Compute provHash from the canonical bytes — sha256 is too cheap
	// to bother charging to a stage.
	provHash, err := provenance.CanonicalHash(preAnchor)
	if err != nil {
		return stat, fmt.Errorf("CanonicalHash: %w", err)
	}

	// 4. Anchor — calibrated lognormal latency on Anchor() so this
	// stage models the on-chain submit-receipt round-trip.
	ctx := context.Background()
	t0 = time.Now()
	anchorRec, err := anchor.Anchor(ctx, in.BundleID, in.RetrievalID, provHash)
	stat.TAnchorNS = time.Since(t0).Nanoseconds()
	if err != nil {
		return stat, fmt.Errorf("Anchor: %w", err)
	}

	// 5. SetRoot (which recomputes the hash internally — same canonical
	// path, plus attaching anchor metadata) + final Marshal for the
	// shipped document.
	t0 = time.Now()
	if err := doc.SetRoot(&provenance.ProvenanceRoot{
		AnchoredOnChain: true,
		ChainTxHash:     anchorRec.TxHash,
		BlockNumber:     anchorRec.BlockNumber,
		AnchorContract:  "0xMockE-PROVAnchor000000000000000000000000",
		AnchoredAt:      time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}); err != nil {
		return stat, fmt.Errorf("SetRoot: %w", err)
	}
	finalDoc, err := doc.Marshal()
	stat.TSetRootNS = time.Since(t0).Nanoseconds()
	if err != nil {
		return stat, fmt.Errorf("Marshal final: %w", err)
	}
	stat.BytesDoc = len(finalDoc)

	// Build resolvers for the verifier. Anchor resolver keys on the
	// short 4-byte prefix because that's what extractIDs recovers from
	// the doc node IDs (see internal/provenance/verifier.go::extractIDs).
	var shortBundle, shortRetrieval [32]byte
	copy(shortBundle[:], in.BundleID[:4])
	copy(shortRetrieval[:], in.RetrievalID[:4])

	anchors := provenance.StaticAnchorResolver{}
	anchors.SetAnchor(shortBundle, shortRetrieval, provHash, anchorRec.BlockNumber)

	// Apply tampering BEFORE verify — we want to measure verify cost
	// even on the unhappy path because that reflects what a real
	// auditor would do.
	verifyDoc, freshAnchor, freshKeys, err := applyTamper(tCase, finalDoc, keys, anchors, shortBundle, shortRetrieval)
	if err != nil {
		return stat, fmt.Errorf("tamper %s: %w", tCase, err)
	}

	// 6. Verify.
	v := &provenance.Verifier{Keys: freshKeys, Anchors: freshAnchor}
	t0 = time.Now()
	res, _ := v.Verify(verifyDoc)
	stat.TVerifyNS = time.Since(t0).Nanoseconds()

	if res != nil {
		stat.Valid = res.Valid
		stat.FailureReason = res.FailureReason
	}

	switch tCase {
	case "happy":
		stat.TamperCaught = stat.Valid // happy = caught means "verified clean"
	default:
		stat.TamperCaught = !stat.Valid
	}

	stat.TTotalNS = stat.TGenNS + stat.TSignNS + stat.TCanonNS + stat.TAnchorNS + stat.TSetRootNS + stat.TVerifyNS
	return stat, nil
}

// applyTamper returns the (possibly tampered) document and the
// resolvers that should be used to verify it. For cases that change
// the document hash, we re-anchor against the tampered doc so the
// verifier's hash check passes and we exercise the signature path —
// EXCEPT for hash_tamper and timestamp_tamper, which deliberately
// leave the original anchor in place (so the verifier catches them at
// the hash step).
func applyTamper(
	tCase string,
	doc []byte,
	keys provenance.StaticKeyResolver,
	anchors provenance.StaticAnchorResolver,
	shortBundle, shortRetrieval [32]byte,
) ([]byte, provenance.AnchorResolver, provenance.KeyResolver, error) {
	switch tCase {
	case "happy":
		return doc, anchors, keys, nil

	case "hash_tamper":
		// Bump latencyMs digit to falsify a performance claim. Hash
		// changes → mismatch against original anchor.
		out := bumpLatencyDigit(doc)
		return out, anchors, keys, nil

	case "sig_tamper":
		// Flip a signature nibble; re-anchor so we exercise the BLS
		// path rather than the hash path.
		out := mutateSignatureNibble(doc)
		fresh, err := canonicalHashWithoutRoot(out)
		if err != nil {
			return nil, nil, nil, err
		}
		freshAnchors := provenance.StaticAnchorResolver{}
		freshAnchors.SetAnchor(shortBundle, shortRetrieval, fresh, 1)
		return out, freshAnchors, keys, nil

	case "signer_substitute":
		// Replace one DID with an unknown imposter; re-anchor.
		// "imposter" is unknown to the resolver → unknown_signer.
		out := substituteFirstDID(doc, "did:lbvr:imposter")
		fresh, err := canonicalHashWithoutRoot(out)
		if err != nil {
			return nil, nil, nil, err
		}
		freshAnchors := provenance.StaticAnchorResolver{}
		freshAnchors.SetAnchor(shortBundle, shortRetrieval, fresh, 1)
		return out, freshAnchors, keys, nil

	case "quorum_reduce":
		// Rewrite signers list down to one entry, leave threshold=2.
		out, err := reduceSigners(doc)
		if err != nil {
			return nil, nil, nil, err
		}
		fresh, err := canonicalHashWithoutRoot(out)
		if err != nil {
			return nil, nil, nil, err
		}
		freshAnchors := provenance.StaticAnchorResolver{}
		freshAnchors.SetAnchor(shortBundle, shortRetrieval, fresh, 1)
		return out, freshAnchors, keys, nil

	case "missing_sig":
		// Strip the bundle-entity sig field; re-anchor so we hit
		// missing_sig instead of hash_mismatch.
		out, err := stripBundleSig(doc)
		if err != nil {
			return nil, nil, nil, err
		}
		fresh, err := canonicalHashWithoutRoot(out)
		if err != nil {
			return nil, nil, nil, err
		}
		freshAnchors := provenance.StaticAnchorResolver{}
		freshAnchors.SetAnchor(shortBundle, shortRetrieval, fresh, 1)
		return out, freshAnchors, keys, nil

	case "timestamp_tamper":
		// Bump signed_at year. Original anchor is left in place so
		// the hash check fails (expected: hash_mismatch).
		out := []byte(strings.Replace(string(doc), `"signed_at":"2026-`, `"signed_at":"2099-`, 1))
		// Fallback if year already differs (unlikely):
		if len(out) == len(doc) {
			out = []byte(strings.Replace(string(doc), `"signed_at":"`, `"signed_at":"X`, 1))
		}
		return out, anchors, keys, nil
	}

	return nil, nil, nil, fmt.Errorf("unknown tampering case %q", tCase)
}

// synthInput builds a randomised GenerateInput. RNG is the bench's
// per-run seeded source so the same -seed produces the same sequence.
// crypto/rand is NOT used here because we want determinism; the
// bench's "random" inputs are about input variety, not cryptographic
// secrecy.
func synthInput(rng *mrand.Rand, gw1, gw2 provenance.GatewayAgent) provenance.GenerateInput {
	var bundleID, retrievalID, merkleRoot [32]byte
	rngFill(rng, bundleID[:])
	rngFill(rng, retrievalID[:])
	rngFill(rng, merkleRoot[:])

	// 46-char CID = baseline IPFS v0 CID length ("Qm" + 44 base58
	// chars). Generated as random base58-ish characters so JCS doesn't
	// choke on null bytes; verifier doesn't decode CIDs.
	cid := func() string {
		b := make([]byte, 22)
		rngFill(rng, b)
		return "Qm" + base58Encode(b)[:44]
	}

	// shardRoot is just hex-prefixed bytes — values aren't validated.
	shardRoot := func() string {
		var b [32]byte
		rngFill(rng, b[:])
		return hexPrefixed(b[:])
	}

	// Deterministic timing inside one iteration; LatencyMs varies
	// across iterations.
	latency := int64(500 + rng.Intn(2500))
	start := time.Date(2026, 4, 30, 14, 32, 0, 0, time.UTC).Add(time.Duration(rng.Intn(3600)) * time.Second)
	end := start.Add(time.Duration(latency) * time.Millisecond)

	return provenance.GenerateInput{
		BundleID:         bundleID,
		MerkleRoot:       merkleRoot,
		BundleSizeBytes:  int64(67_000 + rng.Intn(15_000_000)),
		FHIRResourceType: "Bundle",
		ShardLayout: [3]provenance.ShardPlacement{
			{CID: cid(), Tier: "pinata", ShardRoot: shardRoot()},
			{CID: cid(), Tier: "filebase", ShardRoot: shardRoot()},
			{CID: cid(), Tier: "arweave", ShardRoot: shardRoot()},
		},
		RetrievalID:  retrievalID,
		StartedAt:    start,
		EndedAt:      end,
		RecoveryMode: "fast_path",
		ShardsUsed:   []string{"D0", "D1"},
		RSDecode:     false,
		LatencyMs:    latency,
		Requester: provenance.RequesterAgent{
			ProvType: "prov:Person", Role: "clinician",
			Institution: "did:lbvr:hosp-1",
			AuthzPolicy: "EHDS-Art44-primary-use",
		},
		Gateways:        []provenance.GatewayAgent{gw1, gw2},
		QuorumThreshold: 2,
	}
}

// canonicalHashWithoutRoot recomputes the verifier-style hash for a
// tampered doc; matches the helper of the same name in
// internal/provenance/provenance_e2e_test.go.
func canonicalHashWithoutRoot(doc []byte) ([32]byte, error) {
	var d provenance.Document
	if err := json.Unmarshal(doc, &d); err != nil {
		return [32]byte{}, err
	}
	d.ProvenanceRoot = nil
	bs, err := json.Marshal(d)
	if err != nil {
		return [32]byte{}, err
	}
	return provenance.CanonicalHash(bs)
}

// mutateSignatureNibble flips the first hex nibble after "signature":"0x.
func mutateSignatureNibble(doc []byte) []byte {
	const marker = `"signature":"0x`
	idx := strings.Index(string(doc), marker)
	if idx == -1 {
		return doc
	}
	pos := idx + len(marker)
	if pos >= len(doc) {
		return doc
	}
	b := append([]byte(nil), doc...)
	if b[pos] == 'f' || b[pos] == 'F' {
		b[pos] = '0'
	} else {
		b[pos] = 'f'
	}
	return b
}

// bumpLatencyDigit finds `"lbvr:latencyMs":<digits>` and increments
// the first digit (or wraps 9→1). This guarantees the resulting JSON
// remains syntactically valid AND is byte-different from the original
// → hash mismatch on the verifier's anchor check.
func bumpLatencyDigit(doc []byte) []byte {
	const marker = `"lbvr:latencyMs":`
	idx := strings.Index(string(doc), marker)
	if idx == -1 {
		// Fallback: flip a byte after `"lbvr:` somewhere. Should
		// never happen in practice — JCS preserves the key.
		alt := strings.Index(string(doc), `"lbvr:`)
		if alt == -1 {
			return doc
		}
		out := append([]byte(nil), doc...)
		out[alt+1] = 'X'
		return out
	}
	pos := idx + len(marker)
	if pos >= len(doc) {
		return doc
	}
	b := append([]byte(nil), doc...)
	c := b[pos]
	if c < '0' || c > '9' {
		// No digit — try the next byte (e.g. value starts with '-').
		pos++
		if pos >= len(b) {
			return doc
		}
		c = b[pos]
		if c < '0' || c > '9' {
			return doc
		}
	}
	if c == '9' {
		b[pos] = '1'
	} else {
		b[pos] = c + 1
	}
	return b
}

// substituteFirstDID rewrites the first signer DID it finds in any sig
// block to imposterDID. Re-marshals through the Document struct so the
// resulting bytes remain valid JSON.
func substituteFirstDID(doc []byte, imposterDID string) []byte {
	var d provenance.Document
	if err := json.Unmarshal(doc, &d); err != nil {
		return doc
	}
	bundleID, err := findBundleID(&d)
	if err != nil {
		return doc
	}
	out, ok := rewriteFirstSigner(d.Entity, bundleID, imposterDID)
	if !ok {
		return doc
	}
	d.Entity[bundleID] = out
	bs, err := json.Marshal(d)
	if err != nil {
		return doc
	}
	return bs
}

func reduceSigners(doc []byte) ([]byte, error) {
	var d provenance.Document
	if err := json.Unmarshal(doc, &d); err != nil {
		return nil, err
	}
	bundleID, err := findBundleID(&d)
	if err != nil {
		return nil, err
	}
	var node map[string]json.RawMessage
	if err := json.Unmarshal(d.Entity[bundleID], &node); err != nil {
		return nil, err
	}
	var sigBlock provenance.SignatureBlock
	if err := json.Unmarshal(node["sig"], &sigBlock); err != nil {
		return nil, err
	}
	if len(sigBlock.Signers) < 2 {
		return nil, errors.New("reduceSigners: need ≥2 signers")
	}
	sigBlock.Signers = sigBlock.Signers[:1]
	sigJSON, err := json.Marshal(sigBlock)
	if err != nil {
		return nil, err
	}
	node["sig"] = sigJSON
	bundleJSON, err := json.Marshal(node)
	if err != nil {
		return nil, err
	}
	d.Entity[bundleID] = bundleJSON
	return json.Marshal(d)
}

func stripBundleSig(doc []byte) ([]byte, error) {
	var d provenance.Document
	if err := json.Unmarshal(doc, &d); err != nil {
		return nil, err
	}
	bundleID, err := findBundleID(&d)
	if err != nil {
		return nil, err
	}
	stripped, err := provenance.StripSigField(d.Entity[bundleID])
	if err != nil {
		return nil, err
	}
	d.Entity[bundleID] = stripped
	return json.Marshal(d)
}

func rewriteFirstSigner(store map[string]json.RawMessage, nodeID, imposterDID string) (json.RawMessage, bool) {
	var node map[string]json.RawMessage
	if err := json.Unmarshal(store[nodeID], &node); err != nil {
		return nil, false
	}
	var sigBlock provenance.SignatureBlock
	if err := json.Unmarshal(node["sig"], &sigBlock); err != nil {
		return nil, false
	}
	if len(sigBlock.Signers) == 0 {
		return nil, false
	}
	sigBlock.Signers[0] = imposterDID
	sigJSON, err := json.Marshal(sigBlock)
	if err != nil {
		return nil, false
	}
	node["sig"] = sigJSON
	out, err := json.Marshal(node)
	if err != nil {
		return nil, false
	}
	return out, true
}

func findBundleID(d *provenance.Document) (string, error) {
	for id, raw := range d.Entity {
		var probe struct {
			Type string `json:"prov:type"`
		}
		if err := json.Unmarshal(raw, &probe); err == nil && probe.Type == "lbvr:FHIRBundle" {
			return id, nil
		}
	}
	return "", errors.New("no FHIRBundle entity")
}

// rngFill is a deterministic alternative to crypto/rand.Read using a
// math/rand source seeded from the run's seed flag.
func rngFill(rng *mrand.Rand, b []byte) {
	for i := 0; i < len(b); i += 8 {
		v := rng.Uint64()
		end := i + 8
		if end > len(b) {
			end = len(b)
		}
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], v)
		copy(b[i:end], buf[:end-i])
	}
}

// base58Encode is a minimal Bitcoin-alphabet encoder. We don't need
// round-trip fidelity — the verifier doesn't decode CIDs — so we encode
// raw bytes by indexing into the alphabet 6 bits at a time. Result is
// alphanumeric, JCS-safe, and deterministic from the input.
func base58Encode(b []byte) string {
	const alpha = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	out := make([]byte, 0, (len(b)*8)/6+2)
	bits := uint64(0)
	nbits := 0
	for _, x := range b {
		bits = (bits << 8) | uint64(x)
		nbits += 8
		for nbits >= 6 {
			nbits -= 6
			idx := int((bits >> uint(nbits)) & 0x3f)
			if idx >= len(alpha) {
				idx = idx % len(alpha)
			}
			out = append(out, alpha[idx])
		}
	}
	if nbits > 0 {
		idx := int((bits << uint(6-nbits)) & 0x3f)
		if idx >= len(alpha) {
			idx = idx % len(alpha)
		}
		out = append(out, alpha[idx])
	}
	// Pad/trim to a stable length so synthInput's [:44] slice always
	// has enough bytes. 44 chars × ~5.85 bits/char ≈ 257 bits → need
	// ≥33 input bytes. We get called with 22 → ~30 chars; pad with
	// alphabet[0] to 44.
	for len(out) < 44 {
		out = append(out, alpha[0])
	}
	return string(out)
}

func hexPrefixed(b []byte) string {
	out := make([]byte, 2+hex.EncodedLen(len(b)))
	out[0] = '0'
	out[1] = 'x'
	hex.Encode(out[2:], b)
	return string(out)
}

func safeIDFragment(s string) string {
	if s == "" {
		return "anon"
	}
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z',
			c >= 'A' && c <= 'Z',
			c >= '0' && c <= '9',
			c == '-', c == '_':
			out = append(out, c)
		default:
			out = append(out, '-')
		}
	}
	return string(out)
}

func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return nil
}

func shortCommit() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "nogit"
	}
	return strings.TrimSpace(string(out))
}

func unameOrEmpty() string {
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func cpuOrEmpty() string {
	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func printSummary(rec runRecord, counts, caught map[string]int, wall time.Duration) {
	stages := []string{"gen", "sign", "canon", "anchor", "setroot", "verify"}
	getter := map[string]func(s sampleStat) int64{
		"gen":     func(s sampleStat) int64 { return s.TGenNS },
		"sign":    func(s sampleStat) int64 { return s.TSignNS },
		"canon":   func(s sampleStat) int64 { return s.TCanonNS },
		"anchor":  func(s sampleStat) int64 { return s.TAnchorNS },
		"setroot": func(s sampleStat) int64 { return s.TSetRootNS },
		"verify":  func(s sampleStat) int64 { return s.TVerifyNS },
	}

	parts := []string{}
	for _, st := range stages {
		xs := make([]int64, 0, len(rec.Samples))
		for _, s := range rec.Samples {
			xs = append(xs, getter[st](s))
		}
		sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] })
		p50 := pct(xs, 0.50)
		p99 := pct(xs, 0.99)
		parts = append(parts, fmt.Sprintf("%s P50=%dus P99=%dus", st, p50/1_000, p99/1_000))
	}

	totalCaught := 0
	totalShould := 0
	for c, n := range counts {
		if c == "happy" {
			continue // happy is "caught" iff it verifies clean — different semantics
		}
		totalShould += n
		totalCaught += caught[c]
	}
	rate := 0.0
	if totalShould > 0 {
		rate = 100.0 * float64(totalCaught) / float64(totalShould)
	}

	fmt.Printf(
		"bench-E-PROV: %d iterations; %s; tamper detection: %d/%d (%.1f%%); wall=%s\n",
		rec.Config.N, strings.Join(parts, "; "),
		totalCaught, totalShould, rate,
		wall.Round(time.Millisecond),
	)
}

func pct(xs []int64, q float64) int64 {
	if len(xs) == 0 {
		return 0
	}
	idx := int(float64(len(xs)-1) * q)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(xs) {
		idx = len(xs) - 1
	}
	return xs[idx]
}

// crypto/rand is imported elsewhere (anchor.go); silence unused-import
// false positives in case of future refactor.
var _ = rand.Reader
