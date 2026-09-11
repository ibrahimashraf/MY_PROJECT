package ecs

import (
	"math"
	"testing"
)

func TestPlanetaryECSPhysicsAndSpatialQuery(t *testing.T) {
	world := NewPlanetaryECSWorld(10.0) // 10m grid cell

	// Spawn stationary ground beacon at (0, 0, 0)
	id1 := world.SpawnEntity([3]float64{0, 0, 0}, [3]float64{0, 0, 0}, 100.0, true)

	// Spawn falling crate at (0, 0, 50)
	id2 := world.SpawnEntity([3]float64{0, 0, 50.0}, [3]float64{0, 0, 0}, 10.0, false)

	// Step physics by 1 second
	world.ParallelPhysicsStep(1.0)

	crate := world.Physics[id2]
	// Gravity -9.80665 m/s^2 -> PosZ should be 50 - 9.80665 ≈ 40.19m
	expectedZ := 50.0 - 9.80665
	if math.Abs(crate.PosZ-expectedZ) > 1e-3 {
		t.Fatalf("Crate PosZ mismatch: expected %v, got %v", expectedZ, crate.PosZ)
	}

	// Query around (0, 0, 0) within 5m radius -> should only find beacon (id1)
	nearby := world.QueryEntitiesInRadius(0, 0, 0, 5.0)
	if len(nearby) != 1 || nearby[0] != id1 {
		t.Fatalf("Expected only beacon in 5m radius, got %v", nearby)
	}

	// Query with 60m radius -> should find both
	all := world.QueryEntitiesInRadius(0, 0, 0, 60.0)
	if len(all) != 2 {
		t.Fatalf("Expected 2 entities in 60m radius, got %d", len(all))
	}
}
