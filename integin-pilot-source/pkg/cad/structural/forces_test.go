package structural

import (
	"math"
	"testing"
)

func TestForce3DAndTorque(t *testing.T) {
	// Vertical load: 1000 N downward (-Z)
	f := Force3D{Fx: 0, Fy: 0, Fz: -1000}
	if math.Abs(f.Magnitude()-1000.0) > 1e-6 {
		t.Fatalf("Magnitude mismatch: %v", f.Magnitude())
	}

	// Lever arm: 5m along +X axis
	arm := [3]float64{5.0, 0, 0}
	torque := f.TorqueAtPoint(arm)
	// r x F = (5, 0, 0) x (0, 0, -1000) = (0, -(-5000), 0) = (0, 5000, 0) in standard right hand rule:
	// My = rz*Fx - rx*Fz = 0 - 5*(-1000) = +5000 Nm
	if math.Abs(torque[1]-5000.0) > 1e-6 {
		t.Fatalf("Torque My mismatch: expected 5000, got %v", torque[1])
	}
}

func TestCentrifugalAndDynamicHoistForce(t *testing.T) {
	// 50-ton load slewing at 0.1 rad/s (~1 rpm) at radius 30m
	mass := 50000.0
	omega := 0.1
	radius := 30.0

	fCent := CentrifugalForce(mass, omega, radius)
	// Fc = 50000 * 0.01 * 30 = 15,000 N = 15 kN
	if math.Abs(fCent-15000.0) > 1e-3 {
		t.Fatalf("Centrifugal force mismatch: expected 15000, got %v", fCent)
	}

	// Dynamic hoist: 50 tons hoisting with 0.5 m/s^2 accel and 1.2 DAF
	fDyn := DynamicHoistForce(mass, 0.5, 1.2)
	// (9.80665 + 0.5) * 50000 * 1.2 = 10.30665 * 60000 = 618,399 N
	expectedDyn := (9.80665 + 0.5) * 50000 * 1.2
	if math.Abs(fDyn-expectedDyn) > 1e-3 {
		t.Fatalf("Dynamic hoist force mismatch: expected %v, got %v", expectedDyn, fDyn)
	}
}

func TestMultiLegSlingTension(t *testing.T) {
	// 100 kN total weight, 4-leg sling at 30 degrees (pi/6) from vertical
	totalWeight := 100000.0
	angle := math.Pi / 6.0 // 30 deg -> cos(30 deg) = sqrt(3)/2 ≈ 0.866025

	tension, err := MultiLegSlingTension(totalWeight, 4, angle)
	if err != nil {
		t.Fatalf("Sling tension calculation failed: %v", err)
	}

	expectedTension := 100000.0 / (4.0 * math.Cos(angle))
	if math.Abs(tension-expectedTension) > 1e-3 {
		t.Fatalf("Sling tension mismatch: expected %v, got %v", expectedTension, tension)
	}

	// Unsafe flat angle (>84 degrees -> cos < 0.1)
	_, unsafeErr := MultiLegSlingTension(totalWeight, 4, math.Pi*0.48)
	if unsafeErr == nil {
		t.Fatal("Expected error on dangerous flat sling angle")
	}
}
