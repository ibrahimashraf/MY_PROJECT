package snapshot

import (
	"bytes"
	"encoding/binary"
	"encoding/json/v2"
	"fmt"
	"io"
)

// WorldSnapshotHeader identifies the snapshot binary format version.
type WorldSnapshotHeader struct {
	Magic   [4]byte  // "INTG"
	Version uint32   // Schema version (currently 1)
	Tick    uint64   // World simulation tick
	TimeS   float64  // World time in seconds
	Count   uint32   // Number of entity records
}

// EntityRecord stores compact binary state of one physics entity.
type EntityRecord struct {
	ID              uint32
	PosX, PosY, PosZ float64
	VelX, VelY, VelZ float64
	Mass            float64
	IsStatic        uint8  // 0 = dynamic, 1 = static
}

var magic = [4]byte{'I', 'N', 'T', 'G'}

// Serialize encodes a complete world snapshot to binary writer.
func Serialize(w io.Writer, tick uint64, timeS float64, entities []EntityRecord) error {
	hdr := WorldSnapshotHeader{
		Magic:   magic,
		Version: 1,
		Tick:    tick,
		TimeS:   timeS,
		Count:   uint32(len(entities)),
	}

	if err := binary.Write(w, binary.LittleEndian, hdr); err != nil {
		return fmt.Errorf("snapshot: header write failed: %w", err)
	}

	for _, e := range entities {
		if err := binary.Write(w, binary.LittleEndian, e); err != nil {
			return fmt.Errorf("snapshot: entity write failed (id=%d): %w", e.ID, err)
		}
	}
	return nil
}

// Deserialize decodes a binary snapshot and returns the header and entity records.
func Deserialize(r io.Reader) (WorldSnapshotHeader, []EntityRecord, error) {
	var hdr WorldSnapshotHeader
	if err := binary.Read(r, binary.LittleEndian, &hdr); err != nil {
		return hdr, nil, fmt.Errorf("snapshot: header read failed: %w", err)
	}

	if hdr.Magic != magic {
		return hdr, nil, fmt.Errorf("snapshot: invalid magic %v", hdr.Magic)
	}
	if hdr.Version != 1 {
		return hdr, nil, fmt.Errorf("snapshot: unsupported version %d", hdr.Version)
	}

	entities := make([]EntityRecord, hdr.Count)
	for i := uint32(0); i < hdr.Count; i++ {
		if err := binary.Read(r, binary.LittleEndian, &entities[i]); err != nil {
			return hdr, nil, fmt.Errorf("snapshot: entity %d read failed: %w", i, err)
		}
	}
	return hdr, entities, nil
}

// RoundTrip is a convenience helper that serializes and immediately deserializes in memory.
func RoundTrip(tick uint64, timeS float64, entities []EntityRecord) (WorldSnapshotHeader, []EntityRecord, error) {
	var buf bytes.Buffer
	if err := Serialize(&buf, tick, timeS, entities); err != nil {
		return WorldSnapshotHeader{}, nil, err
	}
	return Deserialize(&buf)
}

// WorldJSONSnapshot represents human-readable JSON world snapshot with v2 annotations.
type WorldJSONSnapshot struct {
	Tick     uint64         `json:"tick,string"`
	TimeS    float64        `json:"time_s"`
	Count    int            `json:"count"`
	Entities []EntityRecord `json:"entities,omitzero"`
}

// SerializeJSON writes snapshot to an io.Writer using encoding/json/v2 zero-allocation streaming with canonical determinism.
func SerializeJSON(w io.Writer, tick uint64, timeS float64, entities []EntityRecord) error {
	snap := WorldJSONSnapshot{
		Tick:     tick,
		TimeS:    timeS,
		Count:    len(entities),
		Entities: entities,
	}
	return json.MarshalWrite(w, snap, json.Deterministic(true))
}

// DeserializeJSON reads snapshot from an io.Reader using encoding/json/v2 with strict unknown-member rejection.
func DeserializeJSON(r io.Reader) (WorldJSONSnapshot, error) {
	var snap WorldJSONSnapshot
	err := json.UnmarshalRead(r, &snap, json.RejectUnknownMembers(true))
	return snap, err
}

