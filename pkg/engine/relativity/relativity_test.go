package relativity

import (
	"math"
	"testing"
)

func TestLorentzTimeDilation(t *testing.T) {
	// ISS speed ~7660 m/s -> very small dilation
	dilated, gamma := TimeDilation(1.0, 7660.0)
	if gamma <= 1.0 {
		t.Fatal("Lorentz factor must be >= 1")
	}
	if math.Abs(dilated-1.0) > 0.01 {
		t.Fatal("ISS dilation should be negligibly small for 1s proper time")
	}
}

func TestSchwarzschildRadius(t *testing.T) {
	// Solar mass: 1.989e30 kg -> rs ≈ 2953 m
	rs := SchwarzschildRadius(1.989e30)
	if rs < 2900 || rs > 3100 {
		t.Fatalf("Solar Schwarzschild radius out of range: %v m", rs)
	}
}

func TestKeplerOrbitalPeriod(t *testing.T) {
	// ISS orbit: a ≈ 6778e3 m, Earth mass = 5.972e24 kg -> T ≈ 5560s
	T := KeplerOrbitalPeriod(6778e3, 5.972e24)
	if T < 5400 || T > 5800 {
		t.Fatalf("ISS orbital period out of expected range: %v s", T)
	}
}

func TestGravitationalTimeDilation(t *testing.T) {
	// Earth surface: factor should be very close to 1 (tiny gravitational redshift)
	factor := GravitationalTimeDilation(5.972e24, 6371e3)
	if factor <= 0 || factor > 1.0 || math.Abs(factor-1.0) > 1e-3 {
		t.Fatalf("Earth surface gravitational time factor unexpected: %v", factor)
	}
}
