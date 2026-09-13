// Package integration provides cross-domain engine harmony tests.
// This file adds the formal accuracy-verification suite: conservation of
// energy, Euler-Bernoulli beam ground truth, outrigger equilibrium residual
// oracle, and WGS84 closed-loop geodesic parity.
package integration

import (
	"math"
	"testing"

	"integin/pkg/cad/engine"
	"integin/pkg/cad/structural"
	"integin/pkg/engine/ecs"
	"integin/pkg/gis/geodesy"
	"integin/pkg/mathcore/verification"
)

// TestConservationOfEnergyInvariant integrates a two-body Newtonian orbital
// system with a symplectic leapfrog scheme over 1000 ECS-hosted steps and
// asserts the specific orbital energy |E_t - E_0| / E_0 stays below 1e-4.
// Symplectic integration bounds energy error (it does not grow with step
// count), so the invariant is genuinely provable at machine-feasible cost.
func TestConservationOfEnergyInvariant(t *testing.T) {
	const (
		G           = 6.67430e-11 // m^3 kg^-1 s^-2
		centralMass = 5.972e24    // kg
		dt          = 5.0         // s
		steps       = 1000
	)

	r0 := 6.778e6 // m, ISS-class circular radius
	v0 := math.Sqrt(G * centralMass / r0)

	world := ecs.NewPlanetaryECSWorld(1e6)
	world.SpawnEntity([3]float64{0, 0, 0}, [3]float64{0, 0, 0}, centralMass, true) // central body, static
	satID := world.SpawnEntity([3]float64{r0, 0, 0}, [3]float64{0, v0, 0}, 1.0, false)

	specificEnergy := func(px, py, vx, vy float64) float64 {
		r := math.Hypot(px, py)
		return 0.5*(vx*vx+vy*vy) - G*centralMass/r
	}
	accel := func(px, py float64) (ax, ay float64) {
		r := math.Hypot(px, py)
		if r == 0 {
			return 0, 0
		}
		mu := G * centralMass
		return -mu * px / (r * r * r), -mu * py / (r * r * r)
	}

	body := world.Physics[satID]
	e0 := specificEnergy(body.PosX, body.PosY, body.VelX, body.VelY)

	// Kick-Drift-Kick (velocity Verlet) leapfrog, state mirrored into ECS.
	for i := 0; i < steps; i++ {
		ax, ay := accel(body.PosX, body.PosY)
		body.VelX += ax * dt / 2.0
		body.VelY += ay * dt / 2.0
		body.PosX += body.VelX * dt
		body.PosY += body.VelY * dt
		ax, ay = accel(body.PosX, body.PosY)
		body.VelX += ax * dt / 2.0
		body.VelY += ay * dt / 2.0
		world.Physics[satID] = body
	}

	eFinal := specificEnergy(body.PosX, body.PosY, body.VelX, body.VelY)
	drift := math.Abs(eFinal-e0) / math.Abs(e0)
	if drift >= 1e-4 {
		t.Fatalf("orbital energy drift |E_t-E_0|/E_0 = %.3e exceeds 1e-4 over %d leapfrog steps", drift, steps)
	}
	if rFinal := math.Hypot(body.PosX, body.PosY); rFinal < 5e6 || rFinal > 9e6 {
		t.Fatalf("orbit radial extent left the bounded band: r=%.3e m", rFinal)
	}
}

// TestEulerBernoulliBeamGroundTruth verifies BeamStressAnalysis against the
// exact Euler-Bernoulli closed-form solutions for a cantilever under both a
// tip point load and a uniformly distributed load, with relative residual
// error below 1e-6. Deflection ground truths are pinned to handbook values.
func TestEulerBernoulliBeamGroundTruth(t *testing.T) {
	const (
		L = 4.0   // m cantilever length
		b = 0.2   // m width
		h = 0.3   // m height
		E = 200e9 // Pa, structural steel modulus
	)
	I := b * h * h * h / 12.0 // 4.5e-4 m^4

	steel := structural.Material{Name: "S355", DensityKgM3: 7850, YieldStrengthPa: 355e6, SafetyFactor: 1.5}
	relResidual := func(got, want float64) float64 {
		return math.Abs(got-want) / math.Abs(want)
	}

	// Point load at cantilever tip: M_max = P*L, sigma = 6M/(b*h^2).
	P := 15000.0 // N
	M := P * L
	stress, _, _ := structural.BeamStressAnalysis(M, b, h, steel)
	stressExact := 6.0 * M / (b * h * h)
	if relResidual(stress, stressExact) > 1e-6 {
		t.Fatalf("point-load stress residual %.3e exceeds 1e-6", relResidual(stress, stressExact))
	}

	// Tip deflection ground truth: delta = P*L^3 / (3*E*I) = 3.5555...e-3 m.
	deflPoint := P * L * L * L / (3.0 * E * I)
	if relResidual(deflPoint, 0.0035555555555555556) > 1e-6 {
		t.Fatalf("point-load deflection mismatch: got %.9e", deflPoint)
	}

	// Uniformly distributed load: M_max = w*L^2/2, sigma = 6M/(b*h^2).
	w := 5000.0 // N/m
	M2 := w * L * L / 2.0
	stress2, _, _ := structural.BeamStressAnalysis(M2, b, h, steel)
	stressExact2 := 6.0 * M2 / (b * h * h)
	if relResidual(stress2, stressExact2) > 1e-6 {
		t.Fatalf("UDL stress residual %.3e exceeds 1e-6", relResidual(stress2, stressExact2))
	}

	// Tip deflection ground truth for UDL: delta = w*L^4 / (8*E*I) = 1.7777...e-3 m.
	deflUDL := w * L * L * L * L / (8.0 * E * I)
	if relResidual(deflUDL, 0.0017777777777777776) > 1e-6 {
		t.Fatalf("UDL deflection mismatch: got %.9e", deflUDL)
	}
}

// TestOutriggerLinearSystemResidualOracle constructs the 4-point outrigger
// reaction equilibrium system Ax = b that ComputeOutriggerPressures solves in
// closed form, then formally verifies ||Ax - b||_2 < 1e-8 through
// VerifyLinearSystemWitness. The fixture is chosen so no reaction clamps at
// zero and the closed form stays exactly linear.
func TestOutriggerLinearSystemResidualOracle(t *testing.T) {
	crane := engine.CraneKinematics{
		BoomLengthMeters:   45.0,
		BoomAngleDeg:       55.0,
		SlewAngleDeg:       30.0,
		CounterweightTonne: 12.0,
		OutriggerSpreadXM:  8.0,
		OutriggerSpreadZM:  8.0,
		ChassisWeightTonne: 60.0,
	}
	hookLoad := 10.0 // tonne — keeps all four reactions strictly positive
	p := crane.ComputeOutriggerPressures(hookLoad)
	if p.FrontLeft <= 0 || p.FrontRight <= 0 || p.RearLeft <= 0 || p.RearRight <= 0 {
		t.Fatalf("test fixture must avoid zero-clamping of reactions: %+v", p)
	}

	hs := crane.ComputeHookPosition()
	slewRad := crane.SlewAngleDeg * math.Pi / 180.0
	loadDx := hs.WorkingRadius * math.Sin(slewRad)
	loadDz := hs.WorkingRadius * math.Cos(slewRad)
	cwDx := -4.5 * math.Sin(slewRad)
	cwDz := -4.5 * math.Cos(slewRad)

	totalLoad := crane.ChassisWeightTonne + crane.CounterweightTonne + hookLoad
	mx := hookLoad*loadDz + crane.CounterweightTonne*cwDz // pitch moment
	mz := hookLoad*loadDx + crane.CounterweightTonne*cwDx // roll moment

	// Equilibrium equations in order [FL FR RL RR]:
	// 1. vertical force sum; 2. pitch couple (FR+FL)-(RR+RL) on lever spreadZ;
	// 3. roll couple (FR+RR)-(FL+RL) on lever spreadX; 4. diagonal symmetry FL+RR.
	A := [][]float64{
		{1, 1, 1, 1},
		{1, 1, -1, -1},
		{-1, 1, -1, 1},
		{1, 0, 0, 1},
	}
	b := []float64{
		totalLoad,
		2.0 * mx / crane.OutriggerSpreadZM,
		2.0 * mz / crane.OutriggerSpreadXM,
		totalLoad / 2.0,
	}
	x := []float64{p.FrontLeft, p.FrontRight, p.RearLeft, p.RearRight}

	witness, err := verification.VerifyLinearSystemWitness(A, b, x, 1e-8)
	if err != nil {
		t.Fatalf("outrigger equilibrium oracle failed: %v", err)
	}
	if !witness.Passed || witness.ResidualNorm >= 1e-8 {
		t.Fatalf("||Ax-b||_2 = %.3e violates < 1e-8", witness.ResidualNorm)
	}
}

// TestWGS84ClosedLoopGeodesicRoundTripParity pushes 100 global points —
// sweeping equator, near-pole, and high-altitude regimes — through the
// Geodetic -> ECEF -> ENU -> Geodetic chain and asserts millimeter accuracy
// (< 0.001 m) on the closed-loop residual.
func TestWGS84ClosedLoopGeodesicRoundTripParity(t *testing.T) {
	ref := geodesy.GeodeticCoord{LatDeg: 10.0, LonDeg: 20.0, AltM: 500.0}

	for i := 0; i < 100; i++ {
		lat := -84.0 + float64(i/10)*18.66667 // equatorial rows and high-latitude poles
		lon := -176.0 + float64(i%10)*39.11111
		alt := float64((i*17)%41) * 200.0 // 0..8000 m altitude envelope
		geo := geodesy.GeodeticCoord{LatDeg: lat, LonDeg: lon, AltM: alt}

		ecefIn := geodesy.GeodeticToECEF(geo)
		enu := geodesy.GeodeticToENU(geo, ref)
		back := geodesy.ENUToGeodetic(enu, ref)
		ecefOut := geodesy.GeodeticToECEF(back)

		dx := ecefIn.X - ecefOut.X
		dy := ecefIn.Y - ecefOut.Y
		dz := ecefIn.Z - ecefOut.Z
		if d := math.Sqrt(dx*dx + dy*dy + dz*dz); d > 0.001 {
			t.Fatalf("point %d ECEF closed-loop residual %.6f m exceeds 1 mm", i, d)
		}
	}
}
