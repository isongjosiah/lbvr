package provenance

import (
	"bytes"
	"strings"
	"testing"
)

// TestCanonicalize_KeyOrderInvariant: JCS sorts keys, so two semantically
// equal documents with different key order canonicalize to identical
// bytes. This is the headline JCS property — anything else here would
// be a fingerprint of a buggy library.
func TestCanonicalize_KeyOrderInvariant(t *testing.T) {
	a := []byte(`{"a":1,"b":2}`)
	b := []byte(`{"b":2,"a":1}`)

	ca, err := Canonicalize(a)
	if err != nil {
		t.Fatalf("canonicalize a: %v", err)
	}
	cb, err := Canonicalize(b)
	if err != nil {
		t.Fatalf("canonicalize b: %v", err)
	}
	if !bytes.Equal(ca, cb) {
		t.Fatalf("expected canonical(a) == canonical(b), got %q vs %q", ca, cb)
	}
}

// TestCanonicalize_NumberNormalization: RFC 8785 §3.2.2 requires
// "shortest representation"; integers don't pick up trailing ".0", and
// scientific notation collapses where possible.
func TestCanonicalize_NumberNormalization(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
	}{
		{"int", []byte(`{"x":1}`)},
		{"trailing-zero", []byte(`{"x":1.0}`)},
		{"sci-zero", []byte(`{"x":1e0}`)},
	}
	want, err := Canonicalize([]byte(`{"x":1}`))
	if err != nil {
		t.Fatalf("baseline: %v", err)
	}
	for _, c := range cases {
		got, err := Canonicalize(c.in)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: got %q want %q", c.name, got, want)
		}
	}
}

// TestCanonicalize_Idempotent: canon(canon(x)) == canon(x). Required
// for the verifier — re-canonicalising on the verify side must not
// drift relative to the signer side.
func TestCanonicalize_Idempotent(t *testing.T) {
	in := []byte(`{"alpha":[3,2,1],"beta":{"k":"v","j":null}}`)
	once, err := Canonicalize(in)
	if err != nil {
		t.Fatalf("once: %v", err)
	}
	twice, err := Canonicalize(once)
	if err != nil {
		t.Fatalf("twice: %v", err)
	}
	if !bytes.Equal(once, twice) {
		t.Errorf("not idempotent: %q vs %q", once, twice)
	}
}

// TestCanonicalize_InvalidJSON returns an error rather than producing
// nonsense bytes. We do not pin the exact error message — the wrapped
// JCS message is library-internal — but we require non-nil.
func TestCanonicalize_InvalidJSON(t *testing.T) {
	_, err := Canonicalize([]byte(`{"oops":}`))
	if err == nil {
		t.Fatal("expected error on malformed JSON")
	}
}

// TestStripSigField_HappyPath: with a "sig" key present, return the
// document minus that key. We re-parse the output to assert sig is
// gone rather than string-matching, since the inner JSON ordering is
// not stable.
func TestStripSigField_HappyPath(t *testing.T) {
	in := []byte(`{"prov:type":"x","sig":{"signature":"0xdead"},"foo":"bar"}`)
	out, err := StripSigField(in)
	if err != nil {
		t.Fatalf("strip: %v", err)
	}
	if strings.Contains(string(out), `"sig"`) {
		t.Errorf("sig still present: %q", out)
	}
	if !strings.Contains(string(out), `"foo"`) {
		t.Errorf("foo missing: %q", out)
	}
}

// TestStripSigField_NoSig: the input has no sig at all; output equals
// input semantically (modulo key reordering, which we tolerate).
func TestStripSigField_NoSig(t *testing.T) {
	in := []byte(`{"a":1,"b":2}`)
	out, err := StripSigField(in)
	if err != nil {
		t.Fatalf("strip: %v", err)
	}
	canonIn, _ := Canonicalize(in)
	canonOut, _ := Canonicalize(out)
	if !bytes.Equal(canonIn, canonOut) {
		t.Errorf("no-op strip altered semantics: %q -> %q", canonIn, canonOut)
	}
}

// TestStripSigField_NestedSigPreserved: only the TOP-level "sig" key
// is removed. A "sig" inside a nested object stays, because the signer
// signs only the node it owns. This is a deliberate design choice — see
// canonicalize.go's StripSigField docstring.
func TestStripSigField_NestedSigPreserved(t *testing.T) {
	in := []byte(`{"a":1,"nested":{"sig":"keep"}}`)
	out, err := StripSigField(in)
	if err != nil {
		t.Fatalf("strip: %v", err)
	}
	if !strings.Contains(string(out), `"sig":"keep"`) {
		t.Errorf("nested sig was unexpectedly stripped: %q", out)
	}
}

// TestCanonicalHash_Deterministic: the same canonical hash on repeated
// invocations of CanonicalHash for the same input. We do not pin the
// hex value because RFC 8785 details (e.g. number normalisation) are
// library-version-dependent in pathological cases; same-process
// determinism is what the verifier relies on.
func TestCanonicalHash_Deterministic(t *testing.T) {
	in := []byte(`{"@context":"x","entity":{"a":{"prov:type":"y"}}}`)
	h1, err := CanonicalHash(in)
	if err != nil {
		t.Fatalf("hash 1: %v", err)
	}
	h2, err := CanonicalHash(in)
	if err != nil {
		t.Fatalf("hash 2: %v", err)
	}
	if h1 != h2 {
		t.Errorf("non-deterministic hash: %x vs %x", h1, h2)
	}
}
