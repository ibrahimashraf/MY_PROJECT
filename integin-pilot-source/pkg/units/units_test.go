package units

import (
	"math"
	"testing"
)

func TestUnitConversions(t *testing.T) {
	if math.Abs(float64(Meters(1000).ToKilometers())-1.0) > 1e-9 {
		t.Fatal("m->km conversion failed")
	}
	if math.Abs(float64(Pascals(355e6).ToMegapascals())-355.0) > 1e-6 {
		t.Fatal("Pa->MPa conversion failed")
	}
	if math.Abs(float64(Celsius(100).ToKelvin())-373.15) > 1e-6 {
		t.Fatal("C->K conversion failed")
	}
	if math.Abs(float64(Degrees(180).ToRadians())-math.Pi) > 1e-9 {
		t.Fatal("deg->rad conversion failed")
	}
	v := MetersPerSecond(343.0)
	mach := v.MachNumber(MetersPerSecond(343.0))
	if math.Abs(mach-1.0) > 1e-9 {
		t.Fatal("Mach 1.0 check failed")
	}
}
