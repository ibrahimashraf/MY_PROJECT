package structural

import (
	"math"
	"testing"
)

func TestFourPointStaticEquilibrium_Symmetric(t *testing.T) {
	// Total load: 400 kN centered at (0, 0)
	totalLoad := 400000.0
	loadPos := [2]float64{0, 0}

	// 4 pads placed symmetrically at 4 corners (+-5m, +-5m)
	padPositions := [4][2]float64{
		{-5.0, -5.0}, // Pad 0 (SW)
		{5.0, -5.0},  // Pad 1 (SE)
		{5.0, 5.0},   // Pad 2 (NE)
		{-5.0, 5.0},  // Pad 3 (NW)
	}

	// Equal pad soil stiffness: 10 MN/m each
	padStiffness := [4]float64{1e7, 1e7, 1e7, 1e7}

	reactions, err := FourPointStaticEquilibrium(totalLoad, loadPos, padPositions, padStiffness)
	if err != nil {
		t.Fatalf("FourPointStaticEquilibrium failed: %v", err)
	}

	// Centered load must distribute equally: 100 kN to each pad
	for i, r := range reactions {
		if math.Abs(r-100000.0) > 1.0 {
			t.Errorf("Pad %d reaction = %v N, want 100000 N", i, r)
		}
	}
}

func TestFourPointStaticEquilibrium_EccentricLoad(t *testing.T) {
	totalLoad := 400000.0
	// Load shifted towards +X (2.5m, 0m)
	loadPos := [2]float64{2.5, 0}

	padPositions := [4][2]float64{
		{-5.0, -5.0},
		{5.0, -5.0},
		{5.0, 5.0},
		{-5.0, 5.0},
	}
	padStiffness := [4]float64{1e7, 1e7, 1e7, 1e7}

	reactions, err := FourPointStaticEquilibrium(totalLoad, loadPos, padPositions, padStiffness)
	if err != nil {
		t.Fatalf("FourPointStaticEquilibrium failed: %v", err)
	}

	sum := reactions[0] + reactions[1] + reactions[2] + reactions[3]
	if math.Abs(sum-totalLoad) > 1e-3 {
		t.Fatalf("Total reaction sum %v != total load %v", sum, totalLoad)
	}

	// East pads (Pad 1 & 2 at X = +5) must take 3/4 of the load (150 kN each), West pads take 1/4 (50 kN each)
	if math.Abs(reactions[1]-150000.0) > 1.0 || math.Abs(reactions[2]-150000.0) > 1.0 {
		t.Errorf("East pads expected 150000 N, got R1=%v, R2=%v", reactions[1], reactions[2])
	}
	if math.Abs(reactions[0]-50000.0) > 1.0 || math.Abs(reactions[3]-50000.0) > 1.0 {
		t.Errorf("West pads expected 50000 N, got R0=%v, R3=%v", reactions[0], reactions[3])
	}
}

func TestWireRopeBendingDerating(t *testing.T) {
	// D/d = 25 -> 100% efficiency
	eff, err := WireRopeBendingDerating(0.5, 0.02) // ratio = 25
	if err != nil || eff != 1.0 {
		t.Errorf("Expected 1.0 at D/d=25, got %v (err: %v)", eff, err)
	}

	// D/d = 4 -> efficiency = 1 - 0.5/sqrt(4) = 1 - 0.25 = 0.75 (75%)
	eff4, err := WireRopeBendingDerating(0.08, 0.02)
	if err != nil || math.Abs(eff4-0.75) > 1e-4 {
		t.Errorf("Expected 0.75 at D/d=4, got %v (err: %v)", eff4, err)
	}
}

func TestShacklePointLoadingDerating(t *testing.T) {
	eff0, _ := ShacklePointLoadingDerating(0)
	if eff0 != 1.0 {
		t.Errorf("Expected 1.0 for inline shackle, got %v", eff0)
	}

	eff45, _ := ShacklePointLoadingDerating(45)
	if math.Abs(eff45-0.70) > 1e-4 {
		t.Errorf("Expected 0.70 at 45 deg, got %v", eff45)
	}

	eff90, _ := ShacklePointLoadingDerating(90)
	if math.Abs(eff90-0.50) > 1e-4 {
		t.Errorf("Expected 0.50 at 90 deg, got %v", eff90)
	}
}

func TestTandemLiftLoadDistribution(t *testing.T) {
	t.Run("Symmetric50_50", func(t *testing.T) {
		// CoG centered between hooks: total span 8.0 m, 4.0 m to each hook.
		result, err := TandemLiftLoadDistribution(600000.0, 4.0, 4.0, 600000.0, 600000.0)
		if err != nil {
			t.Fatalf("TandemLiftLoadDistribution failed: %v", err)
		}
		if math.Abs(result.Crane1LoadN-300000.0) > 1e-3 {
			t.Errorf("Crane1 load = %v N, want 300000 N", result.Crane1LoadN)
		}
		if math.Abs(result.Crane2LoadN-300000.0) > 1e-3 {
			t.Errorf("Crane2 load = %v N, want 300000 N", result.Crane2LoadN)
		}
		if math.Abs(result.SharePct1-50.0) > 1e-3 || math.Abs(result.SharePct2-50.0) > 1e-3 {
			t.Errorf("Shares = %v%%/%v%%, want 50/50", result.SharePct1, result.SharePct2)
		}
	})

	t.Run("AsymmetricCoGCloserToCrane1", func(t *testing.T) {
		// CoG 2.0 m from hook 1, 3.0 m from hook 2 (total span 5.0 m).
		// Crane 1 (closer to CoG) takes the larger share: W1 = W * L2/L = 60%.
		result, err := TandemLiftLoadDistribution(500000.0, 2.0, 3.0, 400000.0, 400000.0)
		if err != nil {
			t.Fatalf("TandemLiftLoadDistribution failed: %v", err)
		}
		if math.Abs(result.Crane1LoadN-300000.0) > 1e-3 {
			t.Errorf("Crane1 load = %v N, want 300000 N", result.Crane1LoadN)
		}
		if math.Abs(result.Crane2LoadN-200000.0) > 1e-3 {
			t.Errorf("Crane2 load = %v N, want 200000 N", result.Crane2LoadN)
		}
		if math.Abs(result.SharePct1-60.0) > 1e-3 || math.Abs(result.SharePct2-40.0) > 1e-3 {
			t.Errorf("Shares = %v%%/%v%%, want 60/40", result.SharePct1, result.SharePct2)
		}
		if math.Abs((result.Crane1LoadN+result.Crane2LoadN)-500000.0) > 1e-3 {
			t.Errorf("Hook loads do not sum to total weight: %v", result.Crane1LoadN+result.Crane2LoadN)
		}
	})

	t.Run("RatedCapacitySafetyFactor", func(t *testing.T) {
		// Crane 2 rated below its share: safety flag must flip to false.
		result, err := TandemLiftLoadDistribution(500000.0, 2.0, 3.0, 350000.0, 150000.0)
		if err != nil {
			t.Fatalf("TandemLiftLoadDistribution failed: %v", err)
		}
		// Crane 1 share 300000 N <= 350000 N rated -> safe.
		if !result.SafeCapacity1 {
			t.Errorf("Crane1 (300000 N vs rating 350000 N) expected safe, got %v", result.SafeCapacity1)
		}
		// Crane 2 share 200000 N > 150000 N rated -> unsafe.
		if result.SafeCapacity2 {
			t.Errorf("Crane2 (200000 N vs rating 150000 N) expected unsafe, got %v", result.SafeCapacity2)
		}
	})
}

func TestVesselUpendingKinematics(t *testing.T) {
	const (
		weight = 2000000.0 // 2000 kN vessel
		length = 40.0      // 40 m between Head trunnion and Tail lug
		cog    = 20.0      // CoG centered: 20 m from Tail lug
	)

	t.Run("Horizontal0Deg", func(t *testing.T) {
		// Baseline: load split purely by CoG ratio -> 50/50.
		head, tail, err := VesselUpendingKinematics(weight, length, cog, 0.0)
		if err != nil {
			t.Fatalf("VesselUpendingKinematics failed: %v", err)
		}
		if math.Abs(head-1000000.0) > 1e-3 {
			t.Errorf("Head load = %v N, want 1000000 N", head)
		}
		if math.Abs(tail-1000000.0) > 1e-3 {
			t.Errorf("Tail load = %v N, want 1000000 N", tail)
		}
	})

	t.Run("MidUpending45Deg", func(t *testing.T) {
		// Head gains the loads that unload from the tail as cos(45deg) scales
		// the horizontal lever: head = W*(0.5) + W*(0.5)*(1-cos45).
		head, tail, err := VesselUpendingKinematics(weight, length, cog, 45.0)
		if err != nil {
			t.Fatalf("VesselUpendingKinematics failed: %v", err)
		}
		wantHead := 1000000.0 + 1000000.0*(1.0-math.Cos(45.0*math.Pi/180.0))
		if math.Abs(head-wantHead) > 1e-3 {
			t.Errorf("Head load = %v N, want %v N", head, wantHead)
		}
		if math.Abs(tail-(weight-head)) > 1e-3 {
			t.Errorf("Tail load = %v N, want %v N", tail, weight-head)
		}
		// Progressive shift: head takes more than the horizontal baseline but
		// less than the full 100% reached at 90 degrees.
		if head <= 1000000.0 || head >= weight {
			t.Errorf("Head load %v N not strictly between 50%% and 100%% of weight", head)
		}
	})

	t.Run("Vertical90Deg", func(t *testing.T) {
		// Fully vertical: Head crane carries 100%, Tail unloads to 0.
		head, tail, err := VesselUpendingKinematics(weight, length, cog, 90.0)
		if err != nil {
			t.Fatalf("VesselUpendingKinematics failed: %v", err)
		}
		if math.Abs(head-weight) > 1e-3 {
			t.Errorf("Head load = %v N, want %v N", head, weight)
		}
		if math.Abs(tail-0.0) > 1e-3 {
			t.Errorf("Tail load = %v N, want 0 N", tail)
		}
	})
}
