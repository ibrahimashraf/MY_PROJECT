package snapshot

import (
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
