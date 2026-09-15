// Package bce provides bounds-check-elimination friendly byte utilities for
// cryptographic tokens, digests, and byte-swapping hot paths.
//
// Every helper is organized so the Go compiler can prove slice and array
// accesses are in-bounds before the hot loop: a single hoisted guard
// (_ = b[7]) or an upfront length check, followed by min-length re-slicing.
// The result is zero heap allocations and zero per-iteration bounds checks,
// which TestBCEVerified checks against the compiler's own BCE debug pass.
package bce

// ReadUint64LE decodes a little-endian uint64 from b.
//
// BCE: the upfront length guard plus the hoisted _ = b[7] makes all eight
// b[i] reads provably in-bounds, so only a single bounds check is emitted.
// It reports false when b holds fewer than 8 bytes.
func ReadUint64LE(b []byte) (uint64, bool) {
	if len(b) < 8 {
		return 0, false
	}
	_ = b[7]
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56, true
}

// ReadUint64BE decodes a big-endian uint64 from b.
//
// BCE: identical hoisting strategy to ReadUint64LE.
// It reports false when b holds fewer than 8 bytes.
func ReadUint64BE(b []byte) (uint64, bool) {
	if len(b) < 8 {
		return 0, false
	}
	_ = b[7]
	return uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
		uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7]), true
}

// WriteUint64LE encodes v into b in little-endian order.
//
// BCE: the _ = b[7] hoist lets the compiler prove every b[i] store is
// in-bounds after the single length guard, emitting no per-store checks.
// It reports false when b holds fewer than 8 bytes.
func WriteUint64LE(b []byte, v uint64) bool {
	if len(b) < 8 {
		return false
	}
	_ = b[7]
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
	b[4] = byte(v >> 32)
	b[5] = byte(v >> 40)
	b[6] = byte(v >> 48)
	b[7] = byte(v >> 56)
	return true
}

// WriteUint64BE encodes v into b in big-endian order.
//
// BCE: identical hoisting strategy to WriteUint64LE.
// It reports false when b holds fewer than 8 bytes.
func WriteUint64BE(b []byte, v uint64) bool {
	if len(b) < 8 {
		return false
	}
	_ = b[7]
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
	return true
}

// XORBytes XORs a and b into dst in place of per-byte, and returns the number
// of bytes written, which is the minimum length of a and b.
//
// BCE: every buffer is re-sliced to the shared minimum length n before the
// loop, giving the compiler exact upper bounds (i < n <= len(buf)), so no
// per-iteration bounds checks are emitted. dst must hold at least n bytes,
// matching crypto/subtle.XORBytes semantics; it panics otherwise.
func XORBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if n == 0 {
		return 0
	}
	if len(dst) < n {
		panic("bce: XORBytes: destination buffer too short")
	}
	dst, a, b = dst[:n], a[:n], b[:n]
	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}
	return n
}

// ConstantTimeCompare32 reports whether x and y have identical contents.
//
// The comparison takes constant time with respect to the contents of x and
// y: ranging over fixed-size arrays eliminates every bounds check (the loop
// index is provably < 32, the array length), and the only data-dependent
// operation is an OR-accumulation, so there is no input-dependent branch to
// mispredict. Both pointers must be non-nil.
func ConstantTimeCompare32(x, y *[32]byte) bool {
	var diff byte
	for i, xb := range x {
		diff |= xb ^ y[i]
	}
	return diff == 0
}

// ConstantTimeCompare64 is the 64-byte variant of ConstantTimeCompare32.
// Both pointers must be non-nil.
func ConstantTimeCompare64(x, y *[64]byte) bool {
	var diff byte
	for i, xb := range x {
		diff |= xb ^ y[i]
	}
	return diff == 0
}

// hexDigits maps a 4-bit value to its lowercase hex ASCII byte.
var hexDigits = [16]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}

// hex2 packs the two hex characters of every byte value (high nibble in the
// high byte), so the encode loop needs a single table lookup per source byte.
//
// Built once at init; the nibble masks/shifts are applied to a byte value, so
// the table build itself is immune to bounds checks too.
var hex2 = func() (t [256]uint16) {
	for i := 0; i < 256; i++ {
		v := byte(i)
		t[i] = uint16(hexDigits[v&0x0f]) | uint16(hexDigits[v>>4])<<8
	}
	return t
}()

// HexEncode32 writes the 64-character lowercase hex encoding of src to dst.
//
// BCE: dst is re-sliced to exactly 64 bytes after the length check and src is
// a fixed-size array, so src[i] and dst[i*2]/dst[i*2+1] are provably in-bounds
// inside the loop; the lookup hex2[c] is indexed by a byte and needs no check.
// It reports false when dst holds fewer than 64 bytes.
func HexEncode32(dst []byte, src *[32]byte) bool {
	if len(dst) < 64 {
		return false
	}
	dst = dst[:64]
	for i, c := range src {
		v := hex2[c]
		dst[i*2] = byte(v >> 8)
		dst[i*2+1] = byte(v)
	}
	return true
}
