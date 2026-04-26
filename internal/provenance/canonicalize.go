package provenance

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

// Canonicalize returns the JCS (RFC 8785) canonical encoding of doc.
// Input must be valid JSON; the underlying library returns its own
// descriptive error on parse failure or unsupported numeric values
// (NaN, ±Inf — RFC 8785 §3.2.2 forbids them).
func Canonicalize(doc []byte) ([]byte, error) {
	out, err := jsoncanonicalizer.Transform(doc)
	if err != nil {
		return nil, fmt.Errorf("provenance: jcs: %w", err)
	}
	return out, nil
}

// CanonicalHash returns SHA-256(JCS(doc)). This is the value anchored
// on-chain (truncated to the contract's bytes32 storage). The intermediate
// canonical bytes are not returned; callers that need them should call
// Canonicalize directly.
func CanonicalHash(doc []byte) ([32]byte, error) {
	canonical, err := Canonicalize(doc)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(canonical), nil
}

// StripSigField returns nodeJSON with any top-level "sig" key removed.
// Used by signers and verifiers to compute the message that gets signed —
// a sig field cannot include a hash of itself. Nested "sig" keys (e.g.
// hypothetical sub-objects that also embed signatures) are intentionally
// left intact: the signer signs the node it owns, not its descendants.
//
// The output is JSON, not necessarily canonical; the caller should pass
// it through Canonicalize before hashing. We round-trip through
// json.Unmarshal/Marshal here rather than string-splice the source bytes
// because doing so handles whitespace, comments-via-error, and embedded
// "sig" keys inside string values without ad-hoc parsing.
func StripSigField(nodeJSON []byte) ([]byte, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(nodeJSON, &raw); err != nil {
		return nil, fmt.Errorf("provenance: strip sig: parse: %w", err)
	}
	delete(raw, "sig")
	out, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("provenance: strip sig: marshal: %w", err)
	}
	return out, nil
}
