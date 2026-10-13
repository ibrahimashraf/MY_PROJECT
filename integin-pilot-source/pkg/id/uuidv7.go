// Package id implements RFC 9562 UUIDv7 identifiers and the INTEGIN
// Sovereign ID authority primitives.
package id

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	uuidV7Length = 36
	maxSequence  = 0x0FFF // 12-bit rand_a sub-millisecond counter ceiling
)

var (
	// ErrInvalidUUID is returned when a string is malformed or is not a
	// version-7 / variant-2 UUID.
	ErrInvalidUUID = errors.New("id: malformed or non-v7 UUID string")

	mu         sync.Mutex
	lastMillis int64
	lastSeq    uint16
)

// NewV7 generates an RFC 9562 UUIDv7 stamped with the current UTC wall clock.
// The returned value is strictly increasing across successive calls.
func NewV7() (string, error) {
	return NewV7FromTime(time.Now().UTC())
}

// NewV7FromTime generates an RFC 9562 UUIDv7 for the given instant.
//
// Layout: 48-bit big-endian unix-ms timestamp, 4-bit version (0b0111),
// 12-bit monotonic sequence counter within the same millisecond, 2-bit
// variant (0b10), and 62 cryptographically random bits.
//
// When 4096 identifiers are emitted within one millisecond the generator
// waits for the clock to advance so the sequence counter never wraps and
// strict ordering is preserved.
func NewV7FromTime(t time.Time) (string, error) {
	millis := t.UnixMilli()

	mu.Lock()
	if millis != lastMillis {
		lastMillis = millis
		lastSeq = 0
	} else if lastSeq < maxSequence {
		lastSeq++
	} else {
		for {
			millis = time.Now().UTC().UnixMilli()
			if millis != lastMillis {
				lastMillis = millis
				lastSeq = 0
				break
			}
		}
	}
	seq := lastSeq
	mu.Unlock()

	var raw [16]byte
	raw[0] = byte(millis >> 40)
	raw[1] = byte(millis >> 32)
	raw[2] = byte(millis >> 24)
	raw[3] = byte(millis >> 16)
	raw[4] = byte(millis >> 8)
	raw[5] = byte(millis)
	raw[6] = 0x70 | byte(seq>>8)
	raw[7] = byte(seq)

	var randBits [8]byte
	if _, err := rand.Read(randBits[:]); err != nil {
		return "", fmt.Errorf("id: crypto/rand read: %w", err)
	}
	raw[8] = 0x80 | (randBits[0] & 0x3F)
	copy(raw[9:], randBits[1:])

	return formatV7(&raw), nil
}

// ParseTime extracts the millisecond-precision timestamp embedded in a
// UUIDv7 string and validates its version and variant.
func ParseTime(uuidStr string) (time.Time, error) {
	var raw [16]byte
	if err := parseUUID(uuidStr, &raw); err != nil {
		return time.Time{}, err
	}
	if raw[6]>>4 != 0x7 {
		return time.Time{}, ErrInvalidUUID
	}
	if raw[8]>>6 != 0b10 {
		return time.Time{}, ErrInvalidUUID
	}
	millis := int64(raw[0])<<40 | int64(raw[1])<<32 | int64(raw[2])<<24 |
		int64(raw[3])<<16 | int64(raw[4])<<8 | int64(raw[5])
	return time.UnixMilli(millis).UTC(), nil
}

// IsValidV7 reports whether the string is a canonical, well-formed UUIDv7
// (hyphen placement, hex digits, version 7 and variant 2).
func IsValidV7(uuidStr string) bool {
	_, err := ParseTime(uuidStr)
	return err == nil
}

// formatV7 writes the 16 raw bytes as a canonical 8-4-4-4-12 dashed,
// lowercase hex string. The sole heap allocation is the returned string.
func formatV7(raw *[16]byte) string {
	var buf [uuidV7Length]byte
	hex.Encode(buf[0:8], raw[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], raw[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], raw[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], raw[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], raw[10:16])
	return string(buf[:])
}

// parseUUID decodes a dashed hex string into 16 bytes, validating exact
// length and hyphen placement without allocating.
func parseUUID(s string, out *[16]byte) error {
	if len(s) != uuidV7Length {
		return ErrInvalidUUID
	}
	var hexSrc [32]byte
	j := 0
	for i := 0; i < len(s); i++ {
		switch i {
		case 8, 13, 18, 23:
			if s[i] != '-' {
				return ErrInvalidUUID
			}
			continue
		}
		hexSrc[j] = s[i]
		j++
	}
	if j != 32 {
		return ErrInvalidUUID
	}
	if _, err := hex.Decode(out[:], hexSrc[:]); err != nil {
		return ErrInvalidUUID
	}
	return nil
}
