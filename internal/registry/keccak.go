package registry

// Keccak-256 (Ethereum / pre-NIST padding 0x01) implementation, pure Go.
//
// Why this lives here: CIDRegistry.sol expects bundleId =
// keccak256(abi.encodePacked(clientId, merkleRoot)). The ingest CLI must
// compute that off-chain. The conventional Go way is golang.org/x/crypto/sha3,
// but that is not yet in the module graph (see go.mod) and the sandboxed
// build environment cannot run `go get` to add it. This single-file
// implementation removes the dependency.
//
// Source: derived from the public-domain reference in the Keccak Code
// Package (https://github.com/XKCP/XKCP) and cross-checked against the
// Ethereum yellow-paper KAT vectors. The padding byte is 0x01 (Keccak),
// not 0x06 (NIST SHA-3) — Ethereum/Solidity uses the former.
//
// TODO(D12): once the module graph admits golang.org/x/crypto, swap this
// out for sha3.NewLegacyKeccak256(). Behaviour must remain bit-identical.

// keccakRC are the round constants for the 24 keccak-f[1600] rounds.
var keccakRC = [24]uint64{
	0x0000000000000001, 0x0000000000008082, 0x800000000000808A, 0x8000000080008000,
	0x000000000000808B, 0x0000000080000001, 0x8000000080008081, 0x8000000000008009,
	0x000000000000008A, 0x0000000000000088, 0x0000000080008009, 0x000000008000000A,
	0x000000008000808B, 0x800000000000008B, 0x8000000000008089, 0x8000000000008003,
	0x8000000000008002, 0x8000000000000080, 0x000000000000800A, 0x800000008000000A,
	0x8000000080008081, 0x8000000000008080, 0x0000000080000001, 0x8000000080008008,
}

// keccakF1600 is the 24-round keccak-f[1600] permutation on a 5x5 lane state.
// Implemented following the canonical θ ρ π χ ι sequence; the rotation
// offsets are inlined as constants in the ρ step.
func keccakF1600(s *[25]uint64) {
	for r := 0; r < 24; r++ {
		// θ
		var c [5]uint64
		c[0] = s[0] ^ s[5] ^ s[10] ^ s[15] ^ s[20]
		c[1] = s[1] ^ s[6] ^ s[11] ^ s[16] ^ s[21]
		c[2] = s[2] ^ s[7] ^ s[12] ^ s[17] ^ s[22]
		c[3] = s[3] ^ s[8] ^ s[13] ^ s[18] ^ s[23]
		c[4] = s[4] ^ s[9] ^ s[14] ^ s[19] ^ s[24]
		var d [5]uint64
		for x := 0; x < 5; x++ {
			d[x] = c[(x+4)%5] ^ rotl(c[(x+1)%5], 1)
		}
		for x := 0; x < 5; x++ {
			for y := 0; y < 5; y++ {
				s[x+5*y] ^= d[x]
			}
		}

		// ρ + π merged: B[y, 2x+3y] = rotl(s[x,y], r[x,y])
		var b [25]uint64
		// hand-unrolled rotation table (FIPS 202 Table 2)
		b[0] = s[0]
		b[10] = rotl(s[1], 1)
		b[20] = rotl(s[2], 62)
		b[5] = rotl(s[3], 28)
		b[15] = rotl(s[4], 27)
		b[16] = rotl(s[5], 36)
		b[1] = rotl(s[6], 44)
		b[11] = rotl(s[7], 6)
		b[21] = rotl(s[8], 55)
		b[6] = rotl(s[9], 20)
		b[7] = rotl(s[10], 3)
		b[17] = rotl(s[11], 10)
		b[2] = rotl(s[12], 43)
		b[12] = rotl(s[13], 25)
		b[22] = rotl(s[14], 39)
		b[23] = rotl(s[15], 41)
		b[8] = rotl(s[16], 45)
		b[18] = rotl(s[17], 15)
		b[3] = rotl(s[18], 21)
		b[13] = rotl(s[19], 8)
		b[14] = rotl(s[20], 18)
		b[24] = rotl(s[21], 2)
		b[9] = rotl(s[22], 61)
		b[19] = rotl(s[23], 56)
		b[4] = rotl(s[24], 14)

		// χ
		for y := 0; y < 5; y++ {
			off := 5 * y
			s[off+0] = b[off+0] ^ ((^b[off+1]) & b[off+2])
			s[off+1] = b[off+1] ^ ((^b[off+2]) & b[off+3])
			s[off+2] = b[off+2] ^ ((^b[off+3]) & b[off+4])
			s[off+3] = b[off+3] ^ ((^b[off+4]) & b[off+0])
			s[off+4] = b[off+4] ^ ((^b[off+0]) & b[off+1])
		}

		// ι
		s[0] ^= keccakRC[r]
	}
}

func rotl(x uint64, n uint) uint64 {
	return (x << n) | (x >> (64 - n))
}

// Keccak256 returns the 32-byte Keccak-256 digest of data using the
// pre-NIST padding rule (delimiter byte 0x01) used by Ethereum / Solidity's
// keccak256 builtin. Output is bit-identical to
// golang.org/x/crypto/sha3.NewLegacyKeccak256(); verified against the
// "" → c5d2460186f7233c... vector and CIDRegistry test vectors.
//
// rate = 1088 bits = 136 bytes (capacity 512 = output 256).
func Keccak256(data []byte) [32]byte {
	const rate = 136 // bytes
	var state [25]uint64
	stateBytes := func(idx int) byte {
		return byte(state[idx/8] >> (8 * uint(idx%8)))
	}

	// Absorb full blocks.
	off := 0
	for len(data)-off >= rate {
		for i := 0; i < rate; i++ {
			state[i/8] ^= uint64(data[off+i]) << (8 * uint(i%8))
		}
		keccakF1600(&state)
		off += rate
	}

	// Pad and absorb final block.
	var pad [rate]byte
	copy(pad[:], data[off:])
	pad[len(data)-off] ^= 0x01 // Keccak (NOT 0x06 — that is NIST SHA-3)
	pad[rate-1] ^= 0x80        // final-block marker
	for i := 0; i < rate; i++ {
		state[i/8] ^= uint64(pad[i]) << (8 * uint(i%8))
	}
	keccakF1600(&state)

	// Squeeze 32 bytes (single squeeze; output < rate).
	var out [32]byte
	for i := 0; i < 32; i++ {
		out[i] = stateBytes(i)
	}
	return out
}

// BundleID computes the canonical bundleId derivation that the contract
// expects: keccak256(clientAddr || merkleRoot). clientAddr is exactly 20
// bytes (an Ethereum address); merkleRoot is exactly 32 bytes. This is the
// same value the contract receives in registerBundle's bundleId argument.
//
// Solidity's abi.encodePacked(address, bytes32) is concatenation without
// length prefixes; our concatenation reproduces that bit-for-bit.
func BundleID(clientAddr [20]byte, merkleRoot [32]byte) [32]byte {
	var buf [52]byte
	copy(buf[:20], clientAddr[:])
	copy(buf[20:], merkleRoot[:])
	return Keccak256(buf[:])
}
