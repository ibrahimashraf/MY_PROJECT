package auditlog

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
)

// Record is a fixed-size 64-byte canonical hash-chain entry. Layout:
//
//	[0:32]   previous hash prefix (raw SHA-256)
//	[32:48]  entry payload hash (first 16 bytes of SHA-256 over the payload)
//	[48:56]  created-at Unix nanoseconds (little-endian)
//	[56:64]  valid-time Unix nanoseconds (little-endian)
//
// Payload = concatenation of EventType + EntityType + EntityID + ActorID + Action.
const (
	recordPrevHashOffset = 0
	recordPayloadOffset  = 32
	recordPayloadLen     = 16
	recordCreatedOffset  = 48
	recordValidOffset    = 56

	maxPayloadLen = 1024
)

var (
	ErrRecordInvalidPrevHash = errors.New("audit log: previous hash must be 64 hex characters")
	ErrRecordPayloadTooLong  = errors.New("audit log: entry payload exceeds 1024 bytes")
)

type Record [64]byte

// AppendRecord builds the canonical 64-byte record for an entry, chained on
// prevHashHex (a 64-hex-char string such as GenesisHash()). Zero allocations.
func AppendRecord(prevHashHex string, e Entry) (Record, error) {
	var rec Record

	var prevHexBuf [64]byte
	if n := copy(prevHexBuf[:], prevHashHex); n != 64 {
		return rec, ErrRecordInvalidPrevHash
	}
	if _, err := hex.Decode(rec[recordPrevHashOffset:recordPayloadOffset], prevHexBuf[:]); err != nil {
		return rec, ErrRecordInvalidPrevHash
	}

	total := len(e.EventType) + len(e.EntityType) + len(e.EntityID) + len(e.ActorID) + len(e.Action)
	if total > maxPayloadLen {
		return rec, ErrRecordPayloadTooLong
	}
	var payload [maxPayloadLen]byte
	pn := copy(payload[:], e.EventType)
	pn += copy(payload[pn:], e.EntityType)
	pn += copy(payload[pn:], e.EntityID)
	pn += copy(payload[pn:], e.ActorID)
	pn += copy(payload[pn:], e.Action)

	digest := sha256.Sum256(payload[:pn])
	copy(rec[recordPayloadOffset:recordCreatedOffset], digest[:recordPayloadLen])

	created := e.CreatedAt.UnixNano()
	binary.LittleEndian.PutUint64(rec[recordCreatedOffset:recordValidOffset], uint64(created))

	valid := e.ValidTime
	if valid.IsZero() {
		valid = e.CreatedAt
	}
	binary.LittleEndian.PutUint64(rec[recordValidOffset:], uint64(valid.UnixNano()))

	return rec, nil
}

func (r Record) PrevHashPrefix() (prev [32]byte) {
	copy(prev[:], r[recordPrevHashOffset:recordPayloadOffset])
	return
}

func (r Record) PayloadHash() (digest [16]byte) {
	copy(digest[:], r[recordPayloadOffset:recordCreatedOffset])
	return
}

func (r Record) CreatedUnixNanos() int64 {
	return int64(binary.LittleEndian.Uint64(r[recordCreatedOffset:recordValidOffset]))
}

func (r Record) ValidUnixNanos() int64 {
	return int64(binary.LittleEndian.Uint64(r[recordValidOffset:]))
}
