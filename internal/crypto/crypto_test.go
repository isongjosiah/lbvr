package crypto

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"testing"
)

func TestGenerateKey_NonZeroAndDistinct(t *testing.T) {
	k1, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	var zero [32]byte
	if k1 == zero {
		t.Fatal("GenerateKey returned zero key")
	}
	k2, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	if k1 == k2 {
		t.Fatal("two GenerateKey calls returned the same key")
	}
}

func TestSealOpen_RoundTrip(t *testing.T) {
	cases := []struct {
		name string
		size int
	}{
		{"empty", 0},
		{"1 byte", 1},
		{"small", 64},
		{"AES block boundary", 16},
		{"chunk-sized 16 KB", 16 * 1024},
		{"chunk + 1", 16*1024 + 1},
	}
	key, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pt := make([]byte, tc.size)
			if _, err := rand.Read(pt); err != nil {
				t.Fatal(err)
			}
			sealed, err := SealChunk(key, pt)
			if err != nil {
				t.Fatalf("SealChunk: %v", err)
			}
			if len(sealed) != NonceSize+len(pt)+16 {
				t.Fatalf("sealed len = %d, want %d", len(sealed), NonceSize+len(pt)+16)
			}
			got, err := OpenChunk(key, sealed)
			if err != nil {
				t.Fatalf("OpenChunk: %v", err)
			}
			if !bytes.Equal(got, pt) {
				t.Fatalf("round-trip mismatch: got %x want %x", got, pt)
			}
		})
	}
}

func TestSealChunk_NonceFreshness(t *testing.T) {
	key, _ := GenerateKey()
	pt := []byte("identical plaintext")
	s1, _ := SealChunk(key, pt)
	s2, _ := SealChunk(key, pt)
	if bytes.Equal(s1[:NonceSize], s2[:NonceSize]) {
		t.Fatal("two seals produced the same nonce")
	}
	if bytes.Equal(s1, s2) {
		t.Fatal("two seals of the same plaintext produced identical output")
	}
}

func TestOpenChunk_Tampering(t *testing.T) {
	key, _ := GenerateKey()
	sealed, err := SealChunk(key, []byte("secret FHIR payload"))
	if err != nil {
		t.Fatal(err)
	}

	// baseline
	if _, err := OpenChunk(key, sealed); err != nil {
		t.Fatalf("baseline open failed: %v", err)
	}

	// flip a ciphertext byte (skip past nonce)
	bad := append([]byte(nil), sealed...)
	bad[NonceSize+1] ^= 0x01
	if _, err := OpenChunk(key, bad); err == nil {
		t.Error("tampered ciphertext opened without error")
	}

	// flip a tag byte (last 16)
	bad = append([]byte(nil), sealed...)
	bad[len(bad)-1] ^= 0x01
	if _, err := OpenChunk(key, bad); err == nil {
		t.Error("tampered tag opened without error")
	}

	// flip a nonce byte -> auth fails
	bad = append([]byte(nil), sealed...)
	bad[0] ^= 0x01
	if _, err := OpenChunk(key, bad); err == nil {
		t.Error("tampered nonce opened without error")
	}
}

func TestOpenChunk_TooShort(t *testing.T) {
	key, _ := GenerateKey()
	if _, err := OpenChunk(key, []byte{1, 2, 3}); err == nil {
		t.Error("expected error on undersized input")
	}
	if _, err := OpenChunk(key, nil); err == nil {
		t.Error("expected error on nil input")
	}
}

func TestOpenChunk_WrongKey(t *testing.T) {
	k1, _ := GenerateKey()
	k2, _ := GenerateKey()
	sealed, _ := SealChunk(k1, []byte("xxx"))
	if _, err := OpenChunk(k2, sealed); err == nil {
		t.Error("open with wrong key succeeded")
	}
}

// NIST CAVP AES-256-GCM known-answer tests.
//
// Source: NIST CAVP AES Algorithm Validation Suite, file
// "gcmEncryptExtIV256.rsp" from the GCM test vector archive
// (csrc.nist.gov / Cryptographic Algorithm Validation Program).
// These specific vectors are Count=0 of the first [Keylen=256, IVlen=96,
// PTlen=0, AADlen=0, Taglen=128] group, and Count=0 of the
// [PTlen=128, AADlen=0, Taglen=128] group. They are the canonical
// AES-256-GCM anchors also used by Go's crypto/cipher test suite.
func TestKAT_NIST_CAVP(t *testing.T) {
	cases := []struct {
		name, key, iv, pt, aad, ct, tag string
	}{
		{
			name: "PTlen=0",
			key:  "b52c505a37d78eda5dd34f20c22540ea1b58963cf8e5bf8ffa85f9f2492505b4",
			iv:   "516c33929df5a3284ff463d7",
			pt:   "",
			aad:  "",
			ct:   "",
			tag:  "bdc1ac884d332457a1d2664f168c76f0",
		},
		{
			name: "PTlen=128",
			key:  "31bdadd96698c204aa9ce1448ea94ae1fb4a9a0b3c9d773b51bb1822666b8f22",
			iv:   "0d18e06c7c725ac9e362e1ce",
			pt:   "2db5168e932556f8089a0622981d017d",
			aad:  "",
			ct:   "fa4362189661d163fcd6a56d8bf0405a",
			tag:  "d636ac1bbedd5cc3ee727dc2ab4a9489",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			keyB := mustHex(t, tc.key)
			if len(keyB) != 32 {
				t.Fatalf("bad key len %d", len(keyB))
			}
			var key [32]byte
			copy(key[:], keyB)
			iv := mustHex(t, tc.iv)
			pt := mustHex(t, tc.pt)
			aad := mustHex(t, tc.aad)
			wantCT := mustHex(t, tc.ct)
			wantTag := mustHex(t, tc.tag)

			got, err := sealWithNonce(key, iv, pt, aad)
			if err != nil {
				t.Fatal(err)
			}
			wantCombined := append(append([]byte(nil), wantCT...), wantTag...)
			if !bytes.Equal(got, wantCombined) {
				t.Fatalf("KAT mismatch:\n got %x\nwant %x", got, wantCombined)
			}
		})
	}
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
