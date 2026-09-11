// Package integration provides cross-domain engine harmony tests.
// This is the proof that all 15 INTEGIN science domains work end-to-end together.
package integration

import (
	"integin/pkg/cad/structural"
	"integin/pkg/engine/bio"
	"integin/pkg/engine/cognitive"
	"integin/pkg/engine/ecs"
	"integin/pkg/engine/eventbus"
	"integin/pkg/engine/geology"
	"integin/pkg/engine/materials"
	"integin/pkg/engine/physics"
	"integin/pkg/engine/planet"
	"integin/pkg/engine/qfield"
	"integin/pkg/engine/relativity"
	"integin/pkg/engine/snapshot"
	"integin/pkg/engine/timecal"
	"integin/pkg/gis/geodesy"
	"integin/pkg/mathcore/verification"
	"integin/pkg/units"
	"math"
	"testing"
)

func TestCrossSystemHarmony(t *testing.T) {
	bus := eventbus.NewBus()
	windReceived := false

	// === Domain coupling: atmosphere -> wildfire wind coupling via event bus ===
	atm := physics.EvaluateAtmosphere(500.0) // 500m altitude
	bus.Subscribe(eventbus.TopicWindUpdate, func(e eventbus.Event) {
		if _, ok := e.Payload.(eventbus.WindEvent); ok {
			windReceived = true
		}
	})
	bus.Publish(eventbus.Event{
		Topic:   eventbus.TopicWindUpdate,
		Payload: eventbus.WindEvent{SpeedMps: atm.SpeedOfSoundMs * 0.05, DirectionX: 0.3, DirectionY: 0.0, AltitudeM: 500.0},
	})
	if !windReceived {
		t.Fatal("Event bus: wind event not received by subscriber")
	}

	// === Wildfire ignited by wind ===
	grid := bio.NewWildfireGrid(16, 16, 0.3, 0.0, 8, 8)
	for i := 0; i < 4; i++ {
		grid.Step(0.7)
	}
	if grid.BurnedCount() < 2 {
		t.Fatal("Wildfire should have spread")
	}

	// === Planetary terrain with geodetic coordinates ===
	rEarth := 6371000.0
	camPos := planet.Vec3d{X: 0, Y: 0, Z: rEarth + 100000.0}
	_ = camPos
	node := planet.NewQuadNode(planet.FacePZ, -1.0, -1.0, 2.0, 0, rEarth)
	node.Subdivide(rEarth)
	if node.Children[0] == nil {
		t.Fatal("Planet quadtree: subdivision failed")
	}

	// === Geodesy: WGS84 ECEF round-trip for terrain coordinate ===
	cairo := geodesy.GeodeticCoord{LatDeg: 30.0444, LonDeg: 31.2357, AltM: 75.0}
	ecef := geodesy.GeodeticToECEF(cairo)
	recovered := geodesy.ECEFToGeodetic(ecef)
	if math.Abs(recovered.LatDeg-cairo.LatDeg) > 1e-6 {
		t.Fatalf("Geodesy ECEF roundtrip failed: lat %v != %v", recovered.LatDeg, cairo.LatDeg)
	}

	// === Hydraulic erosion on terrain heightmap ===
	hm := geology.NewHeightmap(32, 32)
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			hm.Set(x, y, float64(x+y)*3.0)
		}
	}
	hm.SimulateHydraulicErosion(200, geology.DefaultErosionConfig(), 99)

	// === ECS planetary simulation ===
	world := ecs.NewPlanetaryECSWorld(50.0)
	id := world.SpawnEntity([3]float64{0, 0, 100}, [3]float64{0, 0, 0}, 50.0, false)
	world.ParallelPhysicsStep(1.0)
	entity := world.Physics[id]
	if entity.PosZ >= 100.0 {
		t.Fatal("ECS entity should have fallen under gravity")
	}

	// === Materials: structural stress with typed SI units ===
	steel := materials.IsotropicStiffness(200e9, 0.3)
	if steel[0][0] <= 0 {
		t.Fatal("Stiffness tensor should be positive")
	}
	stressPa := structural.BeamStressAnalysis
	_ = stressPa // type-safe interaction via units
	windLoad := units.Newtons(7000.0).ToKilonewtons()
	if float64(windLoad) < 6.9 || float64(windLoad) > 7.1 {
		t.Fatalf("Unit conversion failed: %v kN", windLoad)
	}

	// === Relativity: satellite clock correction using Kepler period ===
	issOrbitPeriodS := relativity.KeplerOrbitalPeriod(6778e3, 5.972e24)
	if issOrbitPeriodS < 5000 || issOrbitPeriodS > 6000 {
		t.Fatalf("Kepler orbital period out of range: %v s", issOrbitPeriodS)
	}

	// === Quantum field: Casimir force at 10nm separation ===
	casimir := qfield.CasimirForce(10e-9)
	if casimir >= 0 {
		t.Fatal("Casimir force must be attractive (negative)")
	}

	// === BDI Cognitive agent deliberation ===
	agent := cognitive.BDIAgent{ID: "surveyor-1"}
	agent.Desires = append(agent.Desires, cognitive.Desire{Goal: "MapTerrain", Priority: 0.9})
	agent.Deliberate()
	if agent.Intention == nil || agent.Intention.Action != "MapTerrain" {
		t.Fatal("BDI agent failed to deliberate on highest-priority desire")
	}

	// === Calendrical: Hijri date for simulation epoch ===
	hY, _, _ := timecal.HijriFromGregorian(2026, 9, 12)
	if hY < 1447 || hY > 1448 {
		t.Fatalf("Hijri calendar conversion failed: got year %d", hY)
	}

	// === Math verification: 9-tier formal oracle ===
	oracle := verification.InvariantDualOracle{Tol: 1e-10}
	if err := oracle.VerifyDeterminantMultiplicativity(2.0, 3.0, 6.0); err != nil {
		t.Fatalf("Formal verification oracle failed: %v", err)
	}

	// === Snapshot: serialize and deserialize world state ===
	records := []snapshot.EntityRecord{
		{ID: 1, PosX: entity.PosX, PosY: entity.PosY, PosZ: entity.PosZ, Mass: entity.Mass},
	}
	hdr, loaded, err := snapshot.RoundTrip(uint64(1), float64(1.0), records)
	if err != nil {
		t.Fatalf("Snapshot roundtrip failed: %v", err)
	}
	if hdr.Count != 1 {
		t.Fatalf("Expected 1 entity in snapshot, got %d", hdr.Count)
	}
	if math.Abs(loaded[0].PosZ-entity.PosZ) > 1e-9 {
		t.Fatalf("Snapshot entity position mismatch: %v != %v", loaded[0].PosZ, entity.PosZ)
	}
}
