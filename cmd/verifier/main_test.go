package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/provenance"
)

// TestMain_VerifyOffline_HappyPath: build the binary in a temp dir,
// generate a signed PROV doc on the fly, write a keys-file, and check
// the CLI returns exit 0 with VALID. This is the closest thing to an
// end-to-end smoke test for the binary surface.
func TestMain_VerifyOffline_HappyPath(t *testing.T) {
	dir := t.TempDir()

	// 1. Build the binary.
	bin := filepath.Join(dir, "lbvr-verify")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("build: %v", err)
	}

	// 2. Generate a signed document. We use the same fixture-style
	// setup as provenance_e2e_test.go but without that test's
	// helpers (different package).
	docBytes, did1, did2, kp1, kp2 := mintTestDocument(t)

	// 3. Write the keys file.
	keysPath := filepath.Join(dir, "keys.json")
	keysContent, _ := json.Marshal(map[string]string{
		did1: "0x" + hex.EncodeToString(kp1[:]),
		did2: "0x" + hex.EncodeToString(kp2[:]),
	})
	if err := os.WriteFile(keysPath, keysContent, 0o600); err != nil {
		t.Fatalf("write keys: %v", err)
	}

	// 4. Write the doc.
	docPath := filepath.Join(dir, "prov.json")
	if err := os.WriteFile(docPath, docBytes, 0o600); err != nil {
		t.Fatalf("write doc: %v", err)
	}

	// 5. Run the verifier in offline mode.
	cmd := exec.Command(bin, "verify",
		"--prov-doc", docPath,
		"--keys-file", keysPath,
		"--offline",
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		t.Fatalf("verify: %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "Result: VALID") {
		t.Errorf("expected 'Result: VALID' in output:\n%s", out.String())
	}
}

// TestMain_VerifyOffline_InvalidDoc: a tampered document yields exit 1
// and 'Result: INVALID'. We mint a doc, modify the latency field, and
// confirm the CLI catches it via signature verification (the offline
// mode skips the anchor check, so the failure must come from the BLS
// signature mismatch).
func TestMain_VerifyOffline_InvalidDoc(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "lbvr-verify")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("build: %v", err)
	}

	docBytes, did1, did2, kp1, kp2 := mintTestDocument(t)
	tampered := bytes.Replace(docBytes, []byte(`"lbvr:latencyMs":1500`), []byte(`"lbvr:latencyMs":42`), 1)
	if bytes.Equal(tampered, docBytes) {
		t.Fatal("tamper failed — fixture latency value changed")
	}

	keysPath := filepath.Join(dir, "keys.json")
	keysContent, _ := json.Marshal(map[string]string{
		did1: "0x" + hex.EncodeToString(kp1[:]),
		did2: "0x" + hex.EncodeToString(kp2[:]),
	})
	_ = os.WriteFile(keysPath, keysContent, 0o600)

	docPath := filepath.Join(dir, "prov.json")
	_ = os.WriteFile(docPath, tampered, 0o600)

	cmd := exec.Command(bin, "verify",
		"--prov-doc", docPath,
		"--keys-file", keysPath,
		"--offline",
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected ExitError, got %v\noutput:\n%s", err, out.String())
	}
	if ee.ExitCode() != 1 {
		t.Errorf("expected exit 1, got %d\noutput:\n%s", ee.ExitCode(), out.String())
	}
	if !strings.Contains(out.String(), "INVALID") {
		t.Errorf("expected INVALID in output:\n%s", out.String())
	}
}

// TestMain_Keygen: keygen should emit JSON with did/public_key/
// private_key fields, all 0x-hex of the correct lengths.
func TestMain_Keygen(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "lbvr-verify")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("build: %v", err)
	}

	cmd := exec.Command(bin, "keygen")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("keygen: %v\n%s", err, out.String())
	}

	var entry struct {
		DID        string `json:"did"`
		PublicKey  string `json:"public_key"`
		PrivateKey string `json:"private_key"`
	}
	if err := json.Unmarshal(out.Bytes(), &entry); err != nil {
		t.Fatalf("parse keygen output: %v\nraw: %s", err, out.String())
	}
	if !strings.HasPrefix(entry.DID, "did:lbvr:gw-") {
		t.Errorf("DID = %q (want did:lbvr:gw- prefix)", entry.DID)
	}
	if len(entry.PublicKey) != 2+2*provenance.PublicKeySize {
		t.Errorf("public key length = %d (want %d)", len(entry.PublicKey), 2+2*provenance.PublicKeySize)
	}
	if len(entry.PrivateKey) != 2+2*provenance.PrivateKeySize {
		t.Errorf("private key length = %d (want %d)", len(entry.PrivateKey), 2+2*provenance.PrivateKeySize)
	}
}

// mintTestDocument generates a valid signed PROV document for the CLI
// smoke tests. Returns the canonical bytes plus the two DIDs and the
// matching pubkey bytes (used to write the keys file).
//
// We implement this here rather than reusing the e2e_test fixture
// because the test binaries live in different packages.
func mintTestDocument(t *testing.T) (
	docBytes []byte,
	did1, did2 string,
	pub1, pub2 [provenance.PublicKeySize]byte,
) {
	t.Helper()

	kp1, err := provenance.GenerateKey()
	if err != nil {
		t.Fatalf("kp1: %v", err)
	}
	kp2, err := provenance.GenerateKey()
	if err != nil {
		t.Fatalf("kp2: %v", err)
	}
	pub1 = kp1.PublicBytes
	pub2 = kp2.PublicBytes

	gw1 := provenance.GatewayAgent{
		ProvType: "prov:SoftwareAgent", Role: "retrieval_gateway",
		Version:   "lbvr-med-test",
		PublicKey: "0x" + hex.EncodeToString(kp1.PublicBytes[:]),
	}
	gw2 := provenance.GatewayAgent{
		ProvType: "prov:SoftwareAgent", Role: "retrieval_gateway",
		Version:   "lbvr-med-test",
		PublicKey: "0x" + hex.EncodeToString(kp2.PublicBytes[:]),
	}
	did1 = "did:lbvr:gw-" + gw1.PublicKey[2:10]
	did2 = "did:lbvr:gw-" + gw2.PublicKey[2:10]

	var bid, rid, mr [32]byte
	for i := range bid {
		bid[i] = byte(i)
	}

	in := provenance.GenerateInput{
		BundleID:         bid,
		MerkleRoot:       mr,
		BundleSizeBytes:  4096,
		FHIRResourceType: "Patient",
		ShardLayout: [3]provenance.ShardPlacement{
			{CID: "Qm0", Tier: "pinata", ShardRoot: "0x00"},
			{CID: "Qm1", Tier: "filebase", ShardRoot: "0x01"},
			{CID: "Qm2", Tier: "arweave", ShardRoot: "0x02"},
		},
		RetrievalID:  rid,
		StartedAt:    time.Date(2026, 4, 30, 14, 32, 15, 0, time.UTC),
		EndedAt:      time.Date(2026, 4, 30, 14, 32, 17, 0, time.UTC),
		RecoveryMode: "fast_path",
		ShardsUsed:   []string{"D0", "D1"},
		LatencyMs:    1500,
		Requester: provenance.RequesterAgent{
			ProvType: "prov:Person", Role: "clinician",
			Institution: "did:lbvr:hosp-1",
			AuthzPolicy: "EHDS-Art44",
		},
		Gateways:        []provenance.GatewayAgent{gw1, gw2},
		QuorumThreshold: 2,
	}

	doc, err := provenance.Generate(in)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if err := doc.Sign(
		[][provenance.PrivateKeySize]byte{kp1.PrivateBytes, kp2.PrivateBytes},
		[]string{did1, did2},
		2,
	); err != nil {
		t.Fatalf("sign: %v", err)
	}
	if err := doc.SetRoot(nil); err != nil {
		t.Fatalf("setRoot: %v", err)
	}
	docBytes, err = doc.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return
}
