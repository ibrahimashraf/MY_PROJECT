package geodesy

import (
	"math"
	"testing"
)

// ecefDistance measures the 3D residual between two geodetic points in ECEF space.
func ecefDistance(a, b GeodeticCoord) float64 {
	ae := GeodeticToECEF(a)
	be := GeodeticToECEF(b)
	dx := ae.X - be.X
	dy := ae.Y - be.Y
	dz := ae.Z - be.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func TestENURoundTripLocalParity(t *testing.T) {
	ref := GeodeticCoord{LatDeg: 10.0, LonDeg: 20.0, AltM: 500.0}
	cases := []GeodeticCoord{
		{LatDeg: 10.0, LonDeg: 20.0, AltM: 500.0},   // the reference itself
		{LatDeg: 10.001, LonDeg: 20.0, AltM: 500.0}, // ~111 m north
		{LatDeg: 10.0, LonDeg: 20.001, AltM: 500.0}, // ~108 m east
		{LatDeg: 10.0001, LonDeg: 20.0002, AltM: 505.0},
		{LatDeg: 9.9955, LonDeg: 19.998, AltM: 480.0},
	}

	for _, c := range cases {
		enu := GeodeticToENU(c, ref)
		back := ENUToGeodetic(enu, ref)
		if d := ecefDistance(c, back); d > 1e-6 {
			t.Fatalf("local ENU round-trip parity broken: %v -> %v (%.3e m)", c, back, d)
		}
	}
}

func TestENUGlobalRoundTripParity(t *testing.T) {
	ref := GeodeticCoord{LatDeg: 10.0, LonDeg: 20.0, AltM: 500.0}

	for i := 0; i < 100; i++ {
		lat := -84.0 + float64(i/10)*18.66667  // -84..+84 sweep, plus implied low-lat rows near equator
		lon := -176.0 + float64(i%10)*39.11111 // full longitude sweep
		alt := float64((i*17)%41) * 200.0      // 0..8000 m high-altitude envelope
		geo := GeodeticCoord{LatDeg: lat, LonDeg: lon, AltM: alt}
		back := ENUToGeodetic(GeodeticToENU(geo, ref), ref)
		if d := ecefDistance(geo, back); d > 0.001 {
			t.Fatalf("point %d (%v) ENU round-trip error %.6f m exceeds 1 mm", i, geo, d)
		}
	}
}

func TestENUPoleRoundTrip(t *testing.T) {
	ref := GeodeticCoord{LatDeg: 10.0, LonDeg: 20.0, AltM: 500.0}
	poles := []GeodeticCoord{
		{LatDeg: 89.9999, LonDeg: 0.0, AltM: 100.0},
		{LatDeg: -89.9999, LonDeg: 179.9999, AltM: 3000.0},
		{LatDeg: 89.9999, LonDeg: 179.9999, AltM: 8000.0},
	}
	for _, p := range poles {
		back := ENUToGeodetic(GeodeticToENU(p, ref), ref)
		if d := ecefDistance(p, back); d > 0.001 {
			t.Fatalf("polar point %v ENU round-trip error %.6f m exceeds 1 mm", p, d)
		}
	}
}

func TestENUAxisConvention(t *testing.T) {
	ref := GeodeticCoord{LatDeg: 0.0, LonDeg: 0.0, AltM: 0.0}

	// Moving 0.001 deg north of the equator ≈ 110.574 m of meridian arc
	// (WGS84 meridional radius at the equator, flattened vs 111.19 km sphere).
	north := GeodeticToENU(GeodeticCoord{LatDeg: 0.001, LonDeg: 0, AltM: 0}, ref)
	if math.Abs(north[1]-110.574) > 0.5 || math.Abs(north[0]) > 0.05 || math.Abs(north[2]) > 0.05 {
		t.Fatalf("NORTH axis wrong: ENU=%v (want N≈110.574)", north)
	}

	// Moving 0.001 deg east of the equator ≈ 111.32 m of parallel arc.
	east := GeodeticToENU(GeodeticCoord{LatDeg: 0, LonDeg: 0.001, AltM: 0}, ref)
	if math.Abs(east[0]-111.32) > 0.5 || math.Abs(east[1]) > 0.05 || math.Abs(east[2]) > 0.05 {
		t.Fatalf("EAST axis wrong: ENU=%v (want E≈111.32)", east)
	}

	// Height gain lands in UP.
	up := GeodeticToENU(GeodeticCoord{LatDeg: 0, LonDeg: 0, AltM: 12.0}, ref)
	if math.Abs(up[2]-12.0) > 1e-6 || math.Abs(up[0]) > 0.05 || math.Abs(up[1]) > 0.05 {
		t.Fatalf("UP axis wrong: ENU=%v (want U=12.0)", up)
	}
}
