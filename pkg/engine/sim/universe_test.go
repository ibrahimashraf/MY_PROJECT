package sim

import (
	"context"
	"testing"
	"time"
)

func TestMultiAgentUniverseSimulation(t *testing.T) {
	world := NewUniverseWorld()

	agent1 := world.SpawnAgent("agent-001", RoleSurveyor, SimVec3{0, 0, 0}, SimVec3{10, 0, 0})
	agent2 := world.SpawnAgent("agent-002", RoleBuilder, SimVec3{0, 5, 0}, SimVec3{0, 5, 0})

	// Step simulation by 1.0 second (5 m/s speed)
	world.Step(1.0)

	if agent1.Position.X < 4.9 || agent1.Position.X > 5.1 {
		t.Fatalf("Agent 1 position mismatch after 1s: got %v", agent1.Position.X)
	}
	if agent1.Status != "MOVING" {
		t.Fatalf("Expected agent 1 moving, got %v", agent1.Status)
	}

	if agent2.Status != "GOAL_REACHED" {
		t.Fatalf("Agent 2 should have reached goal immediately, got %v", agent2.Status)
	}

	// Test 60Hz loop ticker with context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = world.RunUniverseLoop(ctx, 60)
	if world.Tick == 0 {
		t.Fatal("Expected world ticks during loop")
	}
}

func TestContinuousCollisionDetection(t *testing.T) {
	// Ray from (0, 0, 0) to (10, 0, 0), radius 0.5
	// Sphere obstacle at (5, 0, 0), radius 1.0
	start := SimVec3{0, 0, 0}
	end := SimVec3{10, 0, 0}
	obs := SimVec3{5, 0, 0}

	hit, tImpact := ContinuousCollisionCheck(start, end, 0.5, obs, 1.0)
	if !hit {
		t.Fatal("Expected continuous collision hit")
	}
	// Total collision distance: 5 - 1.5 = 3.5m -> t = 3.5 / 10 = 0.35
	if tImpact < 0.34 || tImpact > 0.36 {
		t.Fatalf("Impact parameter t mismatch: expected ~0.35, got %v", tImpact)
	}

	// Miss path: ray moving in Y axis
	missEnd := SimVec3{0, 10, 0}
	missHit, _ := ContinuousCollisionCheck(start, missEnd, 0.5, obs, 1.0)
	if missHit {
		t.Fatal("Expected miss on perpendicular path")
	}
}
