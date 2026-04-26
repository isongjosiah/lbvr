package provenance

import (
	"errors"
	"testing"
)

// TestStaticKeyResolver_HappyPath: round-trip through StaticKeyResolver.
func TestStaticKeyResolver_HappyPath(t *testing.T) {
	var pk [PublicKeySize]byte
	pk[0] = 0x42
	r := StaticKeyResolver{"did:lbvr:test": pk}
	got, err := r.Resolve("did:lbvr:test")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != pk {
		t.Errorf("pk mismatch: got %x want %x", got, pk)
	}
}

// TestStaticKeyResolver_Unknown: an unknown DID returns an error
// that mentions the DID — important for verifier logs.
func TestStaticKeyResolver_Unknown(t *testing.T) {
	r := StaticKeyResolver{}
	_, err := r.Resolve("did:lbvr:missing")
	if err == nil {
		t.Fatal("expected error for unknown DID")
	}
}

// TestStaticAnchorResolver_HappyPath: SetAnchor + Resolve round-trip.
func TestStaticAnchorResolver_HappyPath(t *testing.T) {
	var bid, rid, ph [32]byte
	bid[0] = 1
	rid[0] = 2
	ph[0] = 3

	r := StaticAnchorResolver{}
	r.SetAnchor(bid, rid, ph, 999)
	gotPH, gotBlock, err := r.Resolve(bid, rid)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if gotPH != ph {
		t.Errorf("hash mismatch")
	}
	if gotBlock != 999 {
		t.Errorf("block: got %d want 999", gotBlock)
	}
}

// TestStaticAnchorResolver_Missing: ErrNoAnchor sentinel.
func TestStaticAnchorResolver_Missing(t *testing.T) {
	var bid, rid [32]byte
	r := StaticAnchorResolver{}
	_, _, err := r.Resolve(bid, rid)
	if !errors.Is(err, ErrNoAnchor) {
		t.Fatalf("expected ErrNoAnchor, got %v", err)
	}
}

// TestVerifier_NilGuards: calling Verify with nil verifier or nil
// KeyResolver returns a real error — never panics.
func TestVerifier_NilGuards(t *testing.T) {
	var v *Verifier
	_, err := v.Verify([]byte(`{}`))
	if err == nil {
		t.Fatal("expected error on nil verifier")
	}
	v2 := &Verifier{}
	_, err = v2.Verify([]byte(`{}`))
	if err == nil {
		t.Fatal("expected error on nil KeyResolver")
	}
}

// TestVerifier_ParseFailure: malformed JSON → parse_failed.
func TestVerifier_ParseFailure(t *testing.T) {
	v := &Verifier{Keys: StaticKeyResolver{}, AllowOffline: true}
	res, _ := v.Verify([]byte(`{not-json`))
	if res.Valid {
		t.Fatal("expected invalid")
	}
	if res.FailureReason != "parse_failed" {
		t.Errorf("expected parse_failed, got %q", res.FailureReason)
	}
}

// TestVerifier_RequiresAnchorResolverWhenChainMode: AllowOffline=false
// and Anchors=nil → error. Documents intent: chain mode without an
// anchor source is a misconfiguration, not a silent skip.
func TestVerifier_RequiresAnchorResolverWhenChainMode(t *testing.T) {
	v := &Verifier{Keys: StaticKeyResolver{}}
	// Document needs at least a bundle entity for IDs to extract;
	// supply the bare minimum.
	doc := []byte(`{
        "@context":"x","prefix":{},
        "entity":{"lbvr:bundle/00000000":{"prov:type":"lbvr:FHIRBundle"}},
        "activity":{"lbvr:retrieval/00000000":{"prov:type":"lbvr:VerifiedRetrieval"}},
        "agent":{}
    }`)
	_, err := v.Verify(doc)
	if err == nil {
		t.Fatal("expected error on chain mode with nil Anchors")
	}
}
