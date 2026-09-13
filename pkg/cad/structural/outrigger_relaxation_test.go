package structural

import (
	"math"
	"testing"
)

func TestOutriggerSymmetricCentralLoad(t *testing.T) {
	// 100 kN centred at origin on a 4m x 4m pad square: every pad carries
	// exactly 25 kN of pure compression, no lift-off.
	load := 100000.0
	pads := [4][2]float64{{2, 2}, {-2, 2}, {-2, -2}, {2, -2}}

	sol, err := Outrigger4Solve(load, [2]float64{0, 0}, pads)
	if err != nil {
		t.Fatalf("solver error: %v", err)
	}
	if sol.State != Stable4Point {
		t.Fatalf("expected Stable4Point, got %v", sol.State)
	}
	if sol.ActiveCount != 4 {
		t.Fatalf("expected 4 active pads, got %d", sol.ActiveCount)
	}
	for i := 0; i < 4; i++ {
		if !sol.Active[i] {
			t.Fatalf("pad %d unexpectedly lifted off", i)
		}
		if math.Abs(sol.Reactions[i]-25000) > 1e-6 {
			t.Fatalf("pad %d reaction = %v, want 25000", i, sol.Reactions[i])
		}
	}
	if sol.Residual > 1e-6 {
		t.Fatalf("equilibrium witness residual too large: %e", sol.Residual)
	}
}

func TestOutriggerEccentricLiftOffToTripod(t *testing.T) {
	// 100 kN placed over pad 1 (+x, +y). The planar distribution predicts
	// tension on the opposite diagonal pad; relaxation must resolve to the
	// {1,2,3} tripod with the lifted pad exactly at zero.
	load := 100000.0
	pads := [4][2]float64{{2, 2}, {-2, 2}, {-2, -2}, {2, -2}}
	pos := [2]float64{1.5, 1.5}

	el := elasticReaction4(load, pos, pads, 0)
	if el <= 0 {
		t.Fatalf("test setup wrong: pad 0 elastic reaction %v should be positive", el)
	}

	sol, err := Outrigger4Solve(load, pos, pads)
	if err != nil {
		t.Fatalf("solver error: %v", err)
	}
	if sol.State != Stable3PointTripod {
		t.Fatalf("expected Stable3PointTripod, got %v", sol.State)
	}
	if sol.ActiveCount != 3 {
		t.Fatalf("expected 3 active pads, got %d", sol.ActiveCount)
	}

	// Diagonal pad 4 (idx 3, -x,-y) must be the lifted one.
	if sol.Active[3] {
		t.Fatal("opposite-diagonal pad 3 expected to lift off")
	}
	if math.Abs(sol.Reactions[3]) > 1e-6 {
		t.Fatalf("lifted pad reaction = %v, want 0", sol.Reactions[3])
	}

	// Known analytic tripod solution for {0,1,2}:
	//   r0 = 87500 N, r1 = 0 N, r2 = 12500 N.
	for i, want := range []float64{87500.0, 0.0, 12500.0} {
		if sol.Active[i] && math.Abs(sol.Reactions[i]-want) > 1e-3 {
			t.Fatalf("pad %d reaction = %v, want %v (tol 1e-3)", i, sol.Reactions[i], want)
		}
	}
	if sol.Residual > 1e-6 {
		t.Fatalf("tripod equilibrium witness residual too large: %e", sol.Residual)
	}
}

func TestOutriggerExtremeOverturning(t *testing.T) {
	// 100 kN positioned well outside the support polygon: no tripod can carry
	// it without tension, so the system is flagged as tipping/unstable.
	load := 100000.0
	pads := [4][2]float64{{2, 2}, {-2, 2}, {-2, -2}, {2, -2}}
	pos := [2]float64{6, 6}

	sol, err := Outrigger4Solve(load, pos, pads)
	if err != nil {
		t.Fatalf("solver error: %v", err)
	}
	if sol.State != UnstableTipping {
		t.Fatalf("expected UnstableTipping, got %v", sol.State)
	}
	if sol.Residual > 1e-6 {
		t.Fatalf("residual should be small for the rocker solution, got %e", sol.Residual)
	}
}

func TestOutriggerUQSensitivity(t *testing.T) {
	// 200 kN centred load, 5% load sigma + 0.1 m position sigma + 0.05 m pad
	// sigma. All four pads are symmetrically loaded, so sigma_P must be
	// positive and dominated by the load term: expected
	// sigma_P = dP/dL * sigma_L = (1/4)*10kN = 2.5kN minimum.
	load := 200000.0
	pads := [4][2]float64{{2.5, 2.5}, {-2.5, 2.5}, {-2.5, -2.5}, {2.5, -2.5}}
	in := OutriggerInputUQ{
		TotalLoadN:   load,
		LoadPos:      [2]float64{0, 0},
		Pads:         pads,
		SigmaLoad:    0.05 * load,
		SigmaLoadPos: [2]float64{0.1, 0.1},
		SigmaPad:     0.05,
	}

	res, err := OutriggerUQ(in)
	if err != nil {
		t.Fatalf("UQ error: %v", err)
	}
	if !res.IsStable {
		t.Fatal("centred load must be stable")
	}
	for i := 0; i < 4; i++ {
		if res.LiftOff[i] {
			t.Fatalf("pad %d should not lift off", i)
		}
		if math.Abs(res.Reactions[i]-50000) > 1e-6 {
			t.Fatalf("pad %d reaction = %v, want 50000", i, res.Reactions[i])
		}
		// 5% load sigma propagates as (1/4)*sigma_load = 2500 N through dP/dL.
		if res.StdDevs[i] < 2500 {
			t.Fatalf("pad %d sigma_P = %v, want >= 2500", i, res.StdDevs[i])
		}
		if !isFinite(res.StdDevs[i]) {
			t.Fatalf("pad %d sigma_P is not finite: %v", i, res.StdDevs[i])
		}
		pk := CharacteristicUpperBound(res.Reactions[i], res.StdDevs[i])
		if pk <= res.Reactions[i] {
			t.Fatalf("characteristic upper bound %v must exceed reaction %v", pk, res.Reactions[i])
		}
	}
}

func isFinite(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}

func BenchmarkOutrigger4Solve(b *testing.B) {
	load := 500000.0
	pads := [4][2]float64{{3, 3}, {-3, 3}, {-3, -3}, {3, -3}}
	pos := [2]float64{0.5, -1.2}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Outrigger4Solve(load, pos, pads); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOutriggerUQ(b *testing.B) {
	in := OutriggerInputUQ{
		TotalLoadN:   500000.0,
		LoadPos:      [2]float64{0.5, -1.2},
		Pads:         [4][2]float64{{3, 3}, {-3, 3}, {-3, -3}, {3, -3}},
		SigmaLoad:    25000.0,
		SigmaLoadPos: [2]float64{0.1, 0.1},
		SigmaPad:     0.05,
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := OutriggerUQ(in); err != nil {
			b.Fatal(err)
		}
	}
}
