package structural

import (
	"math"
	"testing"
)

func TestWindSpeedAtHeightPowerLaw(t *testing.T) {
	// At the reference height the profile returns vRef unchanged.
	if got := WindSpeedAtHeight(10.0, 10.0, 10.0, 0.143); got != 10.0 {
		t.Fatalf("at z=z0 expected vRef=10, got %v", got)
	}

	// Open terrain: v(20) = 10 * 2^0.143 ≈ 11.0407 m/s.
	got := WindSpeedAtHeight(10.0, 20.0, 10.0, 0.143)
	expected := 10.0 * math.Pow(2.0, 0.143)
	if math.Abs(got-expected)/expected > 1e-12 {
		t.Fatalf("power-law mismatch: got %v, expected %v", got, expected)
	}

	// ASCE 7 urban profile grows faster than open terrain.
	urban := WindSpeedAtHeight(10.0, 100.0, 10.0, 0.33)
	open := WindSpeedAtHeight(10.0, 100.0, 10.0, 0.143)
	if urban <= open {
		t.Fatalf("urban exponent must scale stronger than open terrain: %v <= %v", urban, open)
	}

	// Monotonic in height.
	below := WindSpeedAtHeight(10.0, 15.0, 10.0, 0.143)
	above := WindSpeedAtHeight(10.0, 25.0, 10.0, 0.143)
	if below >= above {
		t.Fatalf("profile must be monotonically increasing: %v >= %v", below, above)
	}
}

func TestWindSpeedAtHeightGuards(t *testing.T) {
	if got := WindSpeedAtHeight(-5.0, 30.0, 10.0, 0.143); got != 0.0 {
		t.Fatalf("negative vRef must yield 0, got %v", got)
	}
	if got := WindSpeedAtHeight(10.0, 30.0, 0.0, 0.143); got != 0.0 {
		t.Fatalf("non-positive z0 must yield 0, got %v", got)
	}
	if got := WindSpeedAtHeight(10.0, 30.0, 10.0, 0.0); got != 0.0 {
		t.Fatalf("non-positive alpha must yield 0, got %v", got)
	}
	if got := WindSpeedAtHeight(10.0, 5.0, 10.0, 0.143); got != 10.0 {
		t.Fatalf("z below reference must clamp to vRef, got %v", got)
	}
	if math.IsNaN(WindSpeedAtHeight(10.0, 5.0, 10.0, 0.143)) {
		t.Fatal("guard path must not leak NaN")
	}
}

func TestTurbulenceIntensity(t *testing.T) {
	// Open terrain: z=30m, z0=0.03m -> ln(1000) ≈ 6.908 -> I_v ≈ 0.1447.
	iv := TurbulenceIntensity(30.0, 0.03)
	expected := 1.0 / math.Log(30.0/0.03)
	if math.Abs(iv-expected)/expected > 1e-12 {
		t.Fatalf("turbulence intensity mismatch: got %v, expected %v", iv, expected)
	}
	if iv <= 0 || iv >= 1.0 {
		t.Fatalf("open-terrain turbulence out of range: %v", iv)
	}

	// Below roughness height caps at 1.0.
	if got := TurbulenceIntensity(0.01, 0.03); got != 1.0 {
		t.Fatalf("z<=z0 must cap at 1.0, got %v", got)
	}
}

func TestGustPeakPressure(t *testing.T) {
	// v=20 m/s, I_v=0.15 -> G=2.05 -> q_p = 0.5*1.225*(2.05*20)^2 ≈ 1029.28 Pa.
	qp := GustPeakPressure(1.225, 20.0, 0.15)
	expected := 0.5 * 1.225 * ((1.0 + 7.0*0.15) * 20.0) * ((1.0 + 7.0*0.15) * 20.0)
	if qp <= 900.0 || qp >= 1100.0 {
		t.Fatalf("peak pressure out of expected range: %v Pa", qp)
	}
	if math.Abs(qp-expected)/expected > 1e-12 {
		t.Fatalf("peak pressure mismatch: got %v, expected %v", qp, expected)
	}

	// Default air density when non-positive supplied.
	qpDefault := GustPeakPressure(0.0, 20.0, 0.15)
	if math.Abs(qpDefault-expected)/expected > 1e-12 {
		t.Fatalf("default density mismatch: got %v, expected %v", qpDefault, expected)
	}
}
