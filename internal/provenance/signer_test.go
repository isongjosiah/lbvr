package provenance

import (
	"bytes"
	"errors"
	"testing"
)

// TestPubkeyFromPrivate_MatchesGenerateKey: the helper used by callers
// that load BLS keys from .env (gateway main loop) must produce the
// exact pubkey that GenerateKey would have stored alongside the
// private key. A drift here would make every signature we produce
// from .env-loaded keys silently fail downstream verification.
func TestPubkeyFromPrivate_MatchesGenerateKey(t *testing.T) {
	for i := 0; i < 16; i++ {
		kp, err := GenerateKey()
		if err != nil {
			t.Fatalf("genkey[%d]: %v", i, err)
		}
		got, err := PubkeyFromPrivate(kp.PrivateBytes)
		if err != nil {
			t.Fatalf("PubkeyFromPrivate[%d]: %v", i, err)
		}
		if !bytes.Equal(got[:], kp.PublicBytes[:]) {
			t.Fatalf("iter %d: derived pubkey diverges from GenerateKey pubkey", i)
		}
	}
}

// TestPubkeyFromPrivate_RejectsZero: a zero scalar would produce the
// identity element and silently accept any signature.
func TestPubkeyFromPrivate_RejectsZero(t *testing.T) {
	var zero [PrivateKeySize]byte
	if _, err := PubkeyFromPrivate(zero); err == nil {
		t.Fatal("expected error on zero scalar")
	}
}

// TestGenerateKey_Distinct: 100 successive keys, all unique. A
// duplicate would imply broken entropy or a shared internal scratch
// buffer — both are silent and devastating, so this is worth a real
// test rather than a smoke check.
func TestGenerateKey_Distinct(t *testing.T) {
	const n = 100
	seen := make(map[[PrivateKeySize]byte]struct{}, n)
	for i := 0; i < n; i++ {
		kp, err := GenerateKey()
		if err != nil {
			t.Fatalf("genkey[%d]: %v", i, err)
		}
		if _, dup := seen[kp.PrivateBytes]; dup {
			t.Fatalf("duplicate private key at %d", i)
		}
		seen[kp.PrivateBytes] = struct{}{}
	}
}

// TestSignVerify_RoundTrip: sign with one key, verify with the same
// pubkey. The bedrock test — if this fails, nothing else in the
// package works.
func TestSignVerify_RoundTrip(t *testing.T) {
	kp, err := GenerateKey()
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	msg := []byte("hello LBVR-Med")
	sig, err := Sign(kp.PrivateBytes, msg)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if err := Verify(kp.PublicBytes, msg, sig); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

// TestVerify_TamperedMessage: flip one bit in the message — verify
// must fail. Confirms the BLS hash binds the message correctly.
func TestVerify_TamperedMessage(t *testing.T) {
	kp, err := GenerateKey()
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	msg := []byte("hello LBVR-Med")
	sig, err := Sign(kp.PrivateBytes, msg)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	tampered := append([]byte{}, msg...)
	tampered[0] ^= 0x01
	err = Verify(kp.PublicBytes, tampered, sig)
	if !errors.Is(err, ErrSignatureInvalid) {
		t.Fatalf("expected ErrSignatureInvalid, got %v", err)
	}
}

// TestVerify_TamperedSignature: flip one byte in the signature → fail.
// We pick a byte that won't produce an invalid compressed point (which
// would error at decode); the last byte is safest.
func TestVerify_TamperedSignature(t *testing.T) {
	kp, err := GenerateKey()
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	msg := []byte("hello LBVR-Med")
	sig, err := Sign(kp.PrivateBytes, msg)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sig[SignatureSize-1] ^= 0x01
	err = Verify(kp.PublicBytes, msg, sig)
	// Two acceptable failure modes: ErrSignatureInvalid (tampered
	// signature still on the curve, fails the pairing) or a decode
	// error (tampered signature no longer decompresses).
	if err == nil {
		t.Fatal("expected verify failure on tampered signature")
	}
}

// TestAggregateSignatures_VerifyWithAggregatedPubkey: the canonical
// quorum flow. Two signers sign the same message; aggregated sig
// verifies against aggregated pubkey.
func TestAggregateSignatures_VerifyWithAggregatedPubkey(t *testing.T) {
	kp1, err := GenerateKey()
	if err != nil {
		t.Fatalf("kp1: %v", err)
	}
	kp2, err := GenerateKey()
	if err != nil {
		t.Fatalf("kp2: %v", err)
	}
	msg := []byte("quorum-attested message")

	sig1, err := Sign(kp1.PrivateBytes, msg)
	if err != nil {
		t.Fatalf("sign1: %v", err)
	}
	sig2, err := Sign(kp2.PrivateBytes, msg)
	if err != nil {
		t.Fatalf("sign2: %v", err)
	}

	aggSig, err := AggregateSignatures([][SignatureSize]byte{sig1, sig2})
	if err != nil {
		t.Fatalf("agg sig: %v", err)
	}
	aggPub, err := AggregatePublicKeys([][PublicKeySize]byte{kp1.PublicBytes, kp2.PublicBytes})
	if err != nil {
		t.Fatalf("agg pub: %v", err)
	}

	if err := Verify(aggPub, msg, aggSig); err != nil {
		t.Fatalf("verify agg: %v", err)
	}
}

// TestAggregateSignatures_SubsetPubkeyFails: aggregate two sigs but
// verify against only one pubkey → fail. This is the security
// invariant: a verifier cannot accept a 2-of-2 quorum signature
// while only knowing one signer.
func TestAggregateSignatures_SubsetPubkeyFails(t *testing.T) {
	kp1, _ := GenerateKey()
	kp2, _ := GenerateKey()
	msg := []byte("test")
	sig1, _ := Sign(kp1.PrivateBytes, msg)
	sig2, _ := Sign(kp2.PrivateBytes, msg)
	aggSig, err := AggregateSignatures([][SignatureSize]byte{sig1, sig2})
	if err != nil {
		t.Fatalf("agg: %v", err)
	}
	if err := Verify(kp1.PublicBytes, msg, aggSig); err == nil {
		t.Fatal("expected verify failure when aggregated sig checked against single pubkey")
	}
}

// TestAggregateSignatures_Empty returns the sentinel.
func TestAggregateSignatures_Empty(t *testing.T) {
	_, err := AggregateSignatures(nil)
	if !errors.Is(err, ErrEmptyAggregation) {
		t.Fatalf("expected ErrEmptyAggregation, got %v", err)
	}
	_, err = AggregatePublicKeys(nil)
	if !errors.Is(err, ErrEmptyAggregation) {
		t.Fatalf("expected ErrEmptyAggregation, got %v", err)
	}
}

// TestSign_DeterministicForSameKeyAndMessage: BLS is deterministic
// (no randomness in signing). Same key + message → byte-identical
// signature. Verifies our wrapper isn't accidentally injecting
// randomness somewhere.
func TestSign_DeterministicForSameKeyAndMessage(t *testing.T) {
	kp, _ := GenerateKey()
	msg := []byte("determinism-check")
	sig1, err := Sign(kp.PrivateBytes, msg)
	if err != nil {
		t.Fatalf("sign1: %v", err)
	}
	sig2, err := Sign(kp.PrivateBytes, msg)
	if err != nil {
		t.Fatalf("sign2: %v", err)
	}
	if !bytes.Equal(sig1[:], sig2[:]) {
		t.Errorf("signatures differ: %x vs %x", sig1, sig2)
	}
}

// TestSign_RejectsZeroKey: a zero scalar would yield identity-element
// signatures that trivially "verify" against the zero pubkey. We
// reject up front rather than emit a security-broken sig.
func TestSign_RejectsZeroKey(t *testing.T) {
	var zero [PrivateKeySize]byte
	_, err := Sign(zero, []byte("anything"))
	if err == nil {
		t.Fatal("expected error on zero private key")
	}
}
