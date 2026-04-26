// Command lbvr-verify is the standalone PROV-JSON verifier
// (CLAUDE.md §4.6 / docs/provenance-spec.md §7.1). External
// auditors run this against a downloaded prov.json + a key file
// (DID → BLS pubkey) + an anchor file (or the on-chain RPC, once
// the chain client lands on D12).
//
// Subcommands:
//
//	verify   Verify a single PROV-JSON document.
//	keygen   Emit a fresh BLS keypair (private + public + DID).
//
// Output of `verify` matches the spec §7.1 example verbatim so a
// human can read it without grepping; the exit code is 0 on VALID
// and 1 on INVALID. Machine-readable callers should pass --json.
//
// Per CLAUDE.md §13.1 we use stdlib `flag` rather than cobra to keep
// the conference-scope dependency graph minimal; the CLI surface is
// small enough that a subcommand swap is mechanical.
package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/isongjosiah/lbvr-med/internal/provenance"
)

func main() {
	if len(os.Args) < 2 {
		printRootUsage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "verify":
		os.Exit(runVerify(os.Args[2:]))
	case "keygen":
		os.Exit(runKeygen(os.Args[2:]))
	case "help", "-h", "--help":
		printRootUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", os.Args[1])
		printRootUsage()
		os.Exit(2)
	}
}

func printRootUsage() {
	fmt.Fprint(os.Stderr, `lbvr-verify — LBVR-Med PROV-JSON verifier

Usage:
  lbvr-verify verify --prov-doc PATH --keys-file PATH --anchors-file PATH
  lbvr-verify verify --prov-doc PATH --keys-file PATH --offline
  lbvr-verify keygen

Subcommands:
  verify   Verify a signed PROV-JSON document against an anchor.
  keygen   Emit a fresh BLS12-381 keypair as JSON to stdout.

Run 'lbvr-verify verify -h' for verify flags.
`)
}

// runVerify is the verify subcommand handler. Returns the process exit
// code: 0 = VALID, 1 = INVALID, 2 = misuse / IO error.
func runVerify(args []string) int {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		provDoc     string
		keysFile    string
		anchorsFile string
		rpcURL      string
		contract    string
		offline     bool
		jsonOut     bool
	)
	fs.StringVar(&provDoc, "prov-doc", "", "path to the PROV-JSON document to verify (required)")
	fs.StringVar(&keysFile, "keys-file", "", "path to JSON map of DID → 0x-hex pubkey (required)")
	fs.StringVar(&anchorsFile, "anchors-file", "", "path to JSON anchor table (offline mode)")
	fs.StringVar(&rpcURL, "rpc", "", "Ethereum RPC URL (chain mode; not yet implemented)")
	fs.StringVar(&contract, "contract", "", "AuditorLog contract address (chain mode)")
	fs.BoolVar(&offline, "offline", false, "skip on-chain anchor check (testing only)")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON result")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if provDoc == "" || keysFile == "" {
		fmt.Fprintln(os.Stderr, "--prov-doc and --keys-file are required")
		fs.Usage()
		return 2
	}
	if !offline && anchorsFile == "" && rpcURL == "" {
		fmt.Fprintln(os.Stderr, "specify either --anchors-file (offline mode) or --rpc + --contract (chain mode), or pass --offline to skip the anchor check")
		return 2
	}

	docBytes, err := os.ReadFile(provDoc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read prov-doc: %v\n", err)
		return 2
	}
	keys, err := loadKeysFile(keysFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load keys: %v\n", err)
		return 2
	}

	v := &provenance.Verifier{Keys: keys, AllowOffline: offline}
	if !offline {
		if anchorsFile != "" {
			anchors, err := loadAnchorsFile(anchorsFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "load anchors: %v\n", err)
				return 2
			}
			v.Anchors = anchors
		} else {
			v.Anchors = chainAnchorResolver{rpcURL: rpcURL, contract: contract}
		}
	}

	result, err := v.Verify(docBytes)
	if err != nil && !isVerifierFailure(err) {
		// Distinguish "verifier completed; document invalid" (err is
		// nil or a known sentinel like ErrNoAnchor) from real IO /
		// misuse errors. The former is a normal result; the latter
		// gets the exit code 2.
		fmt.Fprintf(os.Stderr, "verify: %v\n", err)
		return 2
	}

	if jsonOut {
		emitJSON(result)
	} else {
		emitHuman(result, docBytes)
	}
	if result == nil || !result.Valid {
		return 1
	}
	return 0
}

// runKeygen emits a fresh keypair as JSON. Use this to register a new
// gateway node — pipe the output to AuditorLog.registerGatewayKey.
//
// SECURITY: the private key is printed to stdout. NEVER pipe this to
// a file or shared terminal. Production deployments should generate
// keys on the gateway host directly.
func runKeygen(args []string) int {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	kp, err := provenance.GenerateKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "keygen: %v\n", err)
		return 1
	}

	pubHex := "0x" + hex.EncodeToString(kp.PublicBytes[:])
	privHex := "0x" + hex.EncodeToString(kp.PrivateBytes[:])
	// DID derivation must match generator.gatewayNodeID — we cannot
	// import that helper because it's package-private. Reproduced
	// here verbatim: first 8 hex chars of pubkey (after 0x).
	did := "did:lbvr:gw-" + pubHex[2:10]

	out := map[string]string{
		"did":         did,
		"public_key":  pubHex,
		"private_key": privHex,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
		return 1
	}
	return 0
}

// emitHuman prints the spec §7.1 textual report. We re-parse the doc
// to display the bundle/retrieval ID, recovery mode, latency, and
// signers — everything a human auditor needs at a glance.
func emitHuman(res *provenance.Result, docBytes []byte) {
	fmt.Println("Verifying provenance document...")

	// Extract a few display fields from the doc directly. We do not
	// fail loud on a parse error here — Verify already bailed if
	// the doc was malformed.
	probe := struct {
		Activity map[string]json.RawMessage `json:"activity"`
		Entity   map[string]json.RawMessage `json:"entity"`
	}{}
	_ = json.Unmarshal(docBytes, &probe)

	for id, raw := range probe.Entity {
		if !strings.HasPrefix(id, "lbvr:bundle/") {
			continue
		}
		fmt.Printf("  Bundle ID:     %s\n", strings.TrimPrefix(id, "lbvr:bundle/"))
		var b struct {
			Sig provenance.SignatureBlock `json:"sig"`
		}
		if err := json.Unmarshal(raw, &b); err == nil && b.Sig.Algorithm != "" {
			// nothing extra to display from the entity sig today
			_ = b
		}
		break
	}
	for id, raw := range probe.Activity {
		if !strings.HasPrefix(id, "lbvr:retrieval/") {
			continue
		}
		fmt.Printf("  Retrieval ID:  %s\n", strings.TrimPrefix(id, "lbvr:retrieval/"))
		var act struct {
			RecoveryMode string                    `json:"lbvr:recoveryMode"`
			ShardsUsed   []string                  `json:"lbvr:shardsUsed"`
			LatencyMs    int64                     `json:"lbvr:latencyMs"`
			Sig          provenance.SignatureBlock `json:"sig"`
		}
		if err := json.Unmarshal(raw, &act); err == nil {
			fmt.Printf("  Recovery mode: %s\n", act.RecoveryMode)
			fmt.Printf("  Shards used:   %s\n", strings.Join(act.ShardsUsed, ", "))
			fmt.Printf("  Latency:       %d ms\n", act.LatencyMs)
			if act.Sig.Algorithm != "" {
				fmt.Printf("  Signed by:     %s (quorum %d/%d)\n",
					strings.Join(act.Sig.Signers, ", "),
					act.Sig.QuorumThreshold, len(act.Sig.Signers),
				)
			}
		}
		break
	}

	fmt.Println()
	fmt.Println("Canonicalization: JCS-RFC8785")
	if res != nil {
		fmt.Printf("Document hash:    %s\n", "0x"+hex.EncodeToString(res.ComputedHash[:]))
	}
	if res != nil && res.AnchoredBlock > 0 {
		fmt.Printf("On-chain anchor:  block %d MATCH\n", res.AnchoredBlock)
	}

	if res != nil && len(res.SignatureChecks) > 0 {
		fmt.Println()
		fmt.Println("Signature verification:")
		for _, c := range res.SignatureChecks {
			status := "VALID"
			if !c.Valid {
				status = "INVALID (" + c.Reason + ")"
			}
			fmt.Printf("  %s signature (%s): %s\n", c.NodeKind, c.AttestationType, status)
		}
	}

	fmt.Println()
	if res != nil && res.Valid {
		fmt.Println("Result: VALID")
	} else if res != nil {
		fmt.Printf("Result: INVALID — %s\n", res.FailureReason)
	} else {
		fmt.Println("Result: ERROR")
	}
}

// emitJSON prints the Result as JSON for machine consumers. Useful in
// CI / batch verification scripts.
func emitJSON(res *provenance.Result) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(res); err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
	}
}

// loadKeysFile reads a JSON object mapping DID → 0x-hex pubkey and
// converts it into a StaticKeyResolver. Unknown / malformed entries
// fail the entire load — partial loads would mask configuration bugs.
func loadKeysFile(path string) (provenance.StaticKeyResolver, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries map[string]string
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	out := make(provenance.StaticKeyResolver, len(entries))
	for did, pkHex := range entries {
		if !strings.HasPrefix(pkHex, "0x") {
			return nil, fmt.Errorf("%s: pubkey missing 0x prefix", did)
		}
		raw, err := hex.DecodeString(pkHex[2:])
		if err != nil {
			return nil, fmt.Errorf("%s: invalid hex: %w", did, err)
		}
		if len(raw) != provenance.PublicKeySize {
			return nil, fmt.Errorf("%s: pubkey size %d (want %d)", did, len(raw), provenance.PublicKeySize)
		}
		var pk [provenance.PublicKeySize]byte
		copy(pk[:], raw)
		out[did] = pk
	}
	return out, nil
}

// loadAnchorsFile reads a JSON file containing a list of anchor entries
// and returns a StaticAnchorResolver. The file format is:
//
//	[
//	  {"bundle_id":"0x...","retrieval_id":"0x...","prov_hash":"0x...","block_number":1234},
//	  ...
//	]
//
// We index by the FIRST 4 BYTES of each ID because the verifier
// extracts only those from the document's short-form node IDs (see
// verifier.extractIDs). This is deliberate for offline mode; chain
// mode uses the full 32-byte IDs derived from the URL.
func loadAnchorsFile(path string) (provenance.StaticAnchorResolver, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []struct {
		BundleID    string `json:"bundle_id"`
		RetrievalID string `json:"retrieval_id"`
		ProvHash    string `json:"prov_hash"`
		BlockNumber uint64 `json:"block_number"`
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	out := provenance.StaticAnchorResolver{}
	for i, e := range entries {
		bid, err := decodeID(e.BundleID)
		if err != nil {
			return nil, fmt.Errorf("entry %d bundle_id: %w", i, err)
		}
		rid, err := decodeID(e.RetrievalID)
		if err != nil {
			return nil, fmt.Errorf("entry %d retrieval_id: %w", i, err)
		}
		ph, err := decodeID(e.ProvHash)
		if err != nil {
			return nil, fmt.Errorf("entry %d prov_hash: %w", i, err)
		}
		// Truncate IDs to first 4 bytes for the offline-mode short-form
		// keying (see verifier.extractIDs comment).
		var bidShort, ridShort [32]byte
		copy(bidShort[:], bid[:4])
		copy(ridShort[:], rid[:4])
		out.SetAnchor(bidShort, ridShort, ph, e.BlockNumber)
	}
	return out, nil
}

// decodeID parses a 0x-prefixed hex string into a 32-byte ID.
func decodeID(s string) ([32]byte, error) {
	var out [32]byte
	if !strings.HasPrefix(s, "0x") {
		return out, errors.New("missing 0x prefix")
	}
	raw, err := hex.DecodeString(s[2:])
	if err != nil {
		return out, err
	}
	if len(raw) != 32 {
		return out, fmt.Errorf("expected 32 bytes, got %d", len(raw))
	}
	copy(out[:], raw)
	return out, nil
}

// chainAnchorResolver is the on-chain (RPC) variant. Stubbed until D12
// when the AuditorLog contract bindings are generated; for now any
// invocation returns ErrChainNotImplemented so the failure mode is
// loud rather than silent.
type chainAnchorResolver struct {
	rpcURL   string
	contract string
}

// ErrChainNotImplemented signals that --rpc / --contract paths are not
// yet wired. D12 lands the AuditorLog bindings.
var ErrChainNotImplemented = errors.New("chain anchor resolver not implemented (D12)")

// Resolve always errors. Kept as a method receiver so the type
// satisfies provenance.AnchorResolver; the verifier surface stays
// compile-stable when D12 implements the body.
func (c chainAnchorResolver) Resolve(bundleID, retrievalID [32]byte) ([32]byte, uint64, error) {
	return [32]byte{}, 0, ErrChainNotImplemented
}

// isVerifierFailure returns true for errors that represent "the
// verifier ran successfully and decided invalid" (vs IO / parse
// errors). These map to exit code 1, not 2.
func isVerifierFailure(err error) bool {
	return errors.Is(err, provenance.ErrNoAnchor) ||
		errors.Is(err, provenance.ErrHashMismatch) ||
		errors.Is(err, provenance.ErrSignatureInvalid)
}
