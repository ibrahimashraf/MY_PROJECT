package snapshot

import (
	"bytes"
	"math"
	"testing"
)

func TestSnapshotRoundTrip(t *testing.T) {
	entities := []EntityRecord{
		{ID: 1, PosX: 1000.5, PosY: -500.25, PosZ: 75.0, VelX: 1.0, VelY: 0, VelZ: -9.8, Mass: 50.0, IsStatic: 0},
		{ID: 2, PosX: 0, PosY: 0, PosZ: 0, Mass: 1e6, IsStatic: 1},
	}

	hdr, loaded, err := RoundTrip(42, 3.14159, entities)
	if err != nil {
		t.Fatalf("Snapshot roundtrip failed: %v", err)
	}
	if hdr.Tick != 42 || math.Abs(hdr.TimeS-3.14159) > 1e-10 {
		t.Fatalf("Header mismatch: tick=%d, time=%v", hdr.Tick, hdr.TimeS)
	}
	if len(loaded) != 2 {
		t.Fatalf("Expected 2 entities, got %d", len(loaded))
	}
	if math.Abs(loaded[0].PosX-1000.5) > 1e-9 {
		t.Fatalf("Entity PosX mismatch: %v", loaded[0].PosX)
	}
	if loaded[1].IsStatic != 1 {
		t.Fatal("Static entity flag mismatch")
	}
}

func TestJSONSnapshotRoundTrip(t *testing.T) {
	entities := []EntityRecord{
		{ID: 1, PosX: 100.0, PosY: 200.0, PosZ: 300.0, Mass: 45.5, IsStatic: 0},
	}

	var buf bytes.Buffer
	if err := SerializeJSON(&buf, 101, 1.25, entities); err != nil {
		t.Fatalf("SerializeJSON failed: %v", err)
	}

	snap, err := DeserializeJSON(&buf)
	if err != nil {
		t.Fatalf("DeserializeJSON failed: %v", err)
	}

	if snap.Tick != 101 || snap.Count != 1 || len(snap.Entities) != 1 {
		t.Fatalf("JSON snapshot payload mismatch: %+v", snap)
	}
}

