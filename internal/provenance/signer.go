package provenance

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"sync"

	bls "github.com/kilic/bls12-381"
)

// DomainSeparationTag is the RFC 9380 hash-to-curve DST. Custom suffix
// "LBVR_2026" prevents cross-protocol attacks: a signature minted under
// this DST cannot be replayed against a future BLS deployment that uses
// the standard "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_" suite, and
// vice-versa. Year suffix gives a clean migration handle if we rotate.
//
// kilic/bls12-381's HashToCurve implements draft-irtf-cfrg-hash-to-curve
// (BLS12381G2_XMD:SHA-256_SSWU_RO_); the DST is the only string the
// caller controls, so encoding the protocol identity here is the
// standards-compliant way to pin a signing context.
const DomainSeparationTag = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_LBVR_2026"

// Signature, public-key, and private-key sizes in bytes. Public key is
// G1 compressed (48 bytes); signature is G2 compressed (96 bytes);
// private key is an Fr scalar (32 bytes).
const (
	PrivateKeySize = 32
	PublicKeySize  = 48
	SignatureSize  = 96
)

// KeyPair is a BLS12-381 keypair. Stored as fixed byte arrays so callers
// never juggle PointG1/Fr internals. Convert at the kilic boundary only.
type KeyPair struct {
	PrivateBytes [PrivateKeySize]byte
	PublicBytes  [PublicKeySize]byte
}

// g1, g2, and one-G1 are constructed once at package init. kilic's G1/G2
// instances embed mutable scratch buffers (see g2.go: tempG2), so they
// are NOT safe for concurrent use; we serialise with a sync.Mutex per
// instance. The mutexes only protect the kilic types, not pure-Go
// computations on the byte slices, so contention is at the millisecond
// scale per signature — fine for the conference benchmark target of
// <10ms signing.
var (
	g1     *bls.G1
	g2     *bls.G2
	g1Gen  *bls.PointG1
	g1Lock sync.Mutex
	g2Lock sync.Mutex
)

func init() {
	g1 = bls.NewG1()
	g2 = bls.NewG2()
	g1Gen = g1.One()
}

// GenerateKey returns a fresh keypair seeded from crypto/rand. Returns
// an error only if crypto/rand fails (e.g. depleted entropy on a
// container with no /dev/urandom).
func GenerateKey() (*KeyPair, error) {
	return generateKeyFrom(rand.Reader)
}

// generateKeyFrom is the testable variant of GenerateKey — tests can
// pass a deterministic reader. NEVER expose this as a public API; the
// security of BLS depends on uniform sk sampling.
func generateKeyFrom(r io.Reader) (*KeyPair, error) {
	sk := bls.NewFr()
	if _, err := sk.Rand(r); err != nil {
		return nil, fmt.Errorf("provenance: keygen: %w", err)
	}
	if sk.IsZero() {
		// Astronomically unlikely (≈ 2^-256), but a zero scalar yields
		// the identity public key, which would silently accept any
		// signature. Fail loud.
		return nil, errors.New("provenance: keygen: zero scalar")
	}

	pkPoint := g1.New()
	g1Lock.Lock()
	g1.MulScalar(pkPoint, g1Gen, sk)
	pkBytes := g1.ToCompressed(pkPoint)
	g1Lock.Unlock()

	kp := &KeyPair{}
	skBytes := sk.ToBytes()
	if len(skBytes) != PrivateKeySize {
		return nil, fmt.Errorf("provenance: keygen: unexpected sk size %d", len(skBytes))
	}
	copy(kp.PrivateBytes[:], skBytes)
	if len(pkBytes) != PublicKeySize {
		return nil, fmt.Errorf("provenance: keygen: unexpected pk size %d", len(pkBytes))
	}
	copy(kp.PublicBytes[:], pkBytes)
	return kp, nil
}

// PubkeyFromPrivate derives the 48-byte compressed G1 public key for a
// given private key, matching the relationship pk = g1 * sk used by
// GenerateKey. Use when keys are loaded from persistent config (e.g.,
// .env GATEWAY_BLS_SK_*) and the caller needs to reconstruct the
// matching pubkey + DID.
func PubkeyFromPrivate(priv [PrivateKeySize]byte) ([PublicKeySize]byte, error) {
	var out [PublicKeySize]byte
	sk := bls.NewFr().FromBytes(priv[:])
	if sk.IsZero() {
		return out, errors.New("provenance: pubkey-from-private: zero private key")
	}
	pkPoint := g1.New()
	g1Lock.Lock()
	g1.MulScalar(pkPoint, g1Gen, sk)
	pkBytes := g1.ToCompressed(pkPoint)
	g1Lock.Unlock()
	if len(pkBytes) != PublicKeySize {
		return out, fmt.Errorf("provenance: pubkey-from-private: unexpected pk size %d", len(pkBytes))
	}
	copy(out[:], pkBytes)
	return out, nil
}

// Sign signs message under priv and returns the 96-byte compressed G2
// signature. The hash-to-curve uses DomainSeparationTag.
//
// scheme: sig = sk * H(message), where H : {0,1}* → G2 via SSWU+isogeny
// per draft-irtf-cfrg-hash-to-curve (kilic exposes this as
// G2.HashToCurve). Verification then checks e(G1_gen, sig) ==
// e(pk, H(m)).
func Sign(priv [PrivateKeySize]byte, message []byte) ([SignatureSize]byte, error) {
	var out [SignatureSize]byte

	sk := bls.NewFr().FromBytes(priv[:])
	if sk.IsZero() {
		return out, errors.New("provenance: sign: zero private key")
	}

	g2Lock.Lock()
	defer g2Lock.Unlock()
	hPoint, err := g2.HashToCurve(message, []byte(DomainSeparationTag))
	if err != nil {
		return out, fmt.Errorf("provenance: hash-to-curve: %w", err)
	}
	sigPoint := g2.New()
	g2.MulScalar(sigPoint, hPoint, sk)
	sigBytes := g2.ToCompressed(sigPoint)
	if len(sigBytes) != SignatureSize {
		return out, fmt.Errorf("provenance: sign: unexpected sig size %d", len(sigBytes))
	}
	copy(out[:], sigBytes)
	return out, nil
}

// AggregateSignatures combines per-signer signatures into one signature
// via G2 point addition. RFC-draft-cfrg-bls-signature §2.7: the
// aggregation function for the basic scheme is the group operation,
// which for G2 is the elliptic-curve addition.
//
// Returns ErrEmptyAggregation if no signatures are provided — an
// aggregation of zero signatures would be the identity element, which
// trivially verifies the empty quorum and is therefore disallowed.
func AggregateSignatures(sigs [][SignatureSize]byte) ([SignatureSize]byte, error) {
	var out [SignatureSize]byte
	if len(sigs) == 0 {
		return out, ErrEmptyAggregation
	}

	g2Lock.Lock()
	defer g2Lock.Unlock()

	acc, err := g2.FromCompressed(sigs[0][:])
	if err != nil {
		return out, fmt.Errorf("provenance: aggregate sigs: decode 0: %w", err)
	}
	for i := 1; i < len(sigs); i++ {
		p, err := g2.FromCompressed(sigs[i][:])
		if err != nil {
			return out, fmt.Errorf("provenance: aggregate sigs: decode %d: %w", i, err)
		}
		g2.Add(acc, acc, p)
	}
	bs := g2.ToCompressed(acc)
	copy(out[:], bs)
	return out, nil
}

// AggregatePublicKeys combines per-signer public keys via G1 point
// addition. Mirror of AggregateSignatures; the verifier needs the
// aggregated pubkey to check the aggregated signature.
func AggregatePublicKeys(pubs [][PublicKeySize]byte) ([PublicKeySize]byte, error) {
	var out [PublicKeySize]byte
	if len(pubs) == 0 {
		return out, ErrEmptyAggregation
	}

	g1Lock.Lock()
	defer g1Lock.Unlock()

	acc, err := g1.FromCompressed(pubs[0][:])
	if err != nil {
		return out, fmt.Errorf("provenance: aggregate pubs: decode 0: %w", err)
	}
	for i := 1; i < len(pubs); i++ {
		p, err := g1.FromCompressed(pubs[i][:])
		if err != nil {
			return out, fmt.Errorf("provenance: aggregate pubs: decode %d: %w", i, err)
		}
		g1.Add(acc, acc, p)
	}
	bs := g1.ToCompressed(acc)
	copy(out[:], bs)
	return out, nil
}

// Verify returns nil iff sig is a valid BLS signature on message under
// aggPub. The aggregated check is:
//
//	e(G1_gen, sig) == e(pk, H(m))
//	⇔ e(G1_gen, sig) · e(-pk, H(m)) == 1
//
// The negated form lets the kilic Engine combine two pairings into one
// product check (Engine.Check), which is ~2× cheaper than two
// independent pairings.
func Verify(aggPub [PublicKeySize]byte, message []byte, sig [SignatureSize]byte) error {
	pkPoint, err := decodePublic(aggPub)
	if err != nil {
		return fmt.Errorf("provenance: verify: decode pk: %w", err)
	}
	sigPoint, err := decodeSignature(sig)
	if err != nil {
		return fmt.Errorf("provenance: verify: decode sig: %w", err)
	}

	g2Lock.Lock()
	hPoint, err := g2.HashToCurve(message, []byte(DomainSeparationTag))
	g2Lock.Unlock()
	if err != nil {
		return fmt.Errorf("provenance: verify: hash-to-curve: %w", err)
	}

	// Engine carries its own G1/G2 scratch space, so we don't need the
	// global locks for the pairing itself; the points were already
	// extracted from kilic types under their respective locks.
	engine := bls.NewEngine()
	engine.AddPair(g1Gen, sigPoint)
	engine.AddPairInv(pkPoint, hPoint)
	if !engine.Check() {
		return ErrSignatureInvalid
	}
	return nil
}

// decodePublic decompresses a 48-byte pubkey, holding the G1 lock. The
// extracted PointG1 owns no further G1-lock-protected state, so it can
// be used by other goroutines (including the pairing engine) without
// further synchronisation.
func decodePublic(pub [PublicKeySize]byte) (*bls.PointG1, error) {
	g1Lock.Lock()
	defer g1Lock.Unlock()
	return g1.FromCompressed(pub[:])
}

// decodeSignature decompresses a 96-byte signature, holding the G2 lock.
func decodeSignature(sig [SignatureSize]byte) (*bls.PointG2, error) {
	g2Lock.Lock()
	defer g2Lock.Unlock()
	return g2.FromCompressed(sig[:])
}

// Sentinel errors. Verifier callers branch on these via errors.Is.
var (
	ErrEmptyAggregation = errors.New("provenance: empty aggregation")
	ErrSignatureInvalid = errors.New("provenance: signature verification failed")
)
