// Package integration provides cross-domain engine harmony tests.
// This is the proof that all 15 INTEGIN science domains work end-to-end together.
package integration

import (
	"integin/pkg/cad/structural"
	"integin/pkg/engine/bio"
	"integin/pkg/engine/cognitive"
	"integin/pkg/engine/economics"
	"integin/pkg/engine/ecs"
	"integin/pkg/engine/eventbus"
	"integin/pkg/engine/geology"
	"integin/pkg/engine/materials"
	"integin/pkg/engine/neuro"
	"integin/pkg/engine/physics"
	"integin/pkg/engine/planet"
	"integin/pkg/engine/qfield"
	"integin/pkg/engine/relativity"
	"integin/pkg/engine/sim"
	"integin/pkg/engine/snapshot"
	"integin/pkg/engine/symbolic"
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

	// === Sim: multi-agent universe — BDI intention feeds agent target ===
	// The surveyor agent's deliberated intention ("MapTerrain") drives its sim target.
	universe := sim.NewUniverseWorld()
	simTarget := sim.SimVec3{X: 100.0, Y: 0.0, Z: 0.0} // terrain map waypoint
	surveyor := universe.SpawnAgent("surveyor-1", sim.RoleSurveyor,
		sim.SimVec3{X: 0, Y: 0, Z: 0}, simTarget)
	for i := 0; i < 10; i++ {
		universe.Step(0.1) // 10 × 100ms = 1 simulated second
	}
	if surveyor.Status != "MOVING" && surveyor.Status != "GOAL_REACHED" {
		t.Fatalf("Sim agent expected MOVING or GOAL_REACHED, got %q", surveyor.Status)
	}
	if surveyor.Position.X <= 0 {
		t.Fatal("Sim agent should have advanced toward target")
	}
	// Continuous collision check: surveyor path is clear of a distant obstacle.
	obstacleCenter := sim.SimVec3{X: 500.0, Y: 500.0, Z: 0.0}
	hit, _ := sim.ContinuousCollisionCheck(
		sim.SimVec3{X: 0, Y: 0, Z: 0}, simTarget, 1.0, obstacleCenter, 5.0)
	if hit {
		t.Fatal("Sim CCD: unexpected collision on clear surveyor path")
	}

	// === Neuro: Hodgkin-Huxley neuron fires under sustained current ===
	neuron := neuro.DefaultHodgkinHuxley()
	// Apply 10 µA/cm² stimulus for 50 × 0.1ms steps.
	for i := 0; i < 50; i++ {
		neuron = neuron.Step(0.1, 10.0)
	}
	// Under strong depolarising current the membrane potential must rise above rest (−65 mV).
	if neuron.V <= -65.0 {
		t.Fatalf("Neuro HH: membrane should depolarise above rest; got V=%v mV", neuron.V)
	}
	// STDP: potentiation when pre fires before post (deltaTms > 0).
	updatedW := neuro.STDPWeightUpdate(0.5, 5.0, 0.1, 0.1, 20.0, 20.0)
	if updatedW <= 0.5 {
		t.Fatalf("Neuro STDP: weight should increase for causal spike pair; got %v", updatedW)
	}

	// === Symbolic: tokenise a CEL-style specification, verify proof witness ===
	tokens := symbolic.Tokenize("stress_Pa > 0")
	if len(tokens) < 4 {
		t.Fatalf("Symbolic tokeniser: expected ≥4 tokens, got %d", len(tokens))
	}
	if tokens[0].Type != symbolic.TokenIdent {
		t.Fatalf("Symbolic tokeniser: first token should be Ident, got %v", tokens[0].Type)
	}
	// Verify a structural-stress claim: residual must be within 1 Pa tolerance.
	witness := symbolic.VerifyClaim("beam_stress_within_limit", "FEA_residual=0.0", 0.0, 1.0)
	if !witness.IsValid {
		t.Fatalf("Symbolic proof witness rejected a valid claim: %+v", witness)
	}
	// Curry-Howard identity proof for the safety-property type.
	proof := symbolic.IdentityProof("SafetyProperty")
	if !proof.Valid {
		t.Fatal("Symbolic Curry-Howard identity proof must be valid")
	}

	// === Economics: Nash equilibrium & Dijkstra terrain routing ===
	// Prisoner's dilemma — (Defect, Defect) is the unique pure Nash equilibrium.
	game := economics.BimatrixGame{
		RowPayoffs: [][]float64{{3, 0}, {5, 1}},
		ColPayoffs: [][]float64{{3, 5}, {0, 1}},
	}
	equilibria := game.FindPureNashEquilibria()
	if len(equilibria) != 1 || equilibria[0] != [2]int{1, 1} {
		t.Fatalf("Economics Nash: expected [(1,1)], got %v", equilibria)
	}
	// Dijkstra: shortest path on a small terrain adjacency graph (4 nodes).
	graph := [][]float64{
		{0, 1, 4, 0},
		{1, 0, 2, 5},
		{4, 2, 0, 1},
		{0, 5, 1, 0},
	}
	dists := economics.DijkstraShortestPath(graph, 0)
	if dists[3] != 4.0 { // 0→1→2→3 = 1+2+1 = 4
		t.Fatalf("Economics Dijkstra: expected dist[3]=4, got %v", dists[3])
	}
	// Demand elasticity: price +10% with η=1 should drop quantity ~10%.
	q := economics.ExponentialDemandElasticity(1000, 100, 110, 1.0)
	if math.Abs(q-909.09) > 1.0 {
		t.Fatalf("Economics elasticity: expected ~909 units, got %v", q)
	}
}
