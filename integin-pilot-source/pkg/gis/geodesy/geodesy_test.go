package geodesy

import (
	"math"
	"testing"
)

func TestWGS84ECEFRoundTrip(t *testing.T) {
	// Eiffel Tower, Paris: 48.8584° N, 2.2945° E, Altitude 300m
	eiffel := GeodeticCoord{
		LatDeg: 48.8584,
		LonDeg: 2.2945,
		AltM:   300.0,
	}

	ecef := GeodeticToECEF(eiffel)
	if ecef.X == 0 || ecef.Y == 0 || ecef.Z == 0 {
		t.Fatalf("ECEF conversion returned zero coordinates: %v", ecef)
	}

	// Reverse conversion via Bowring algorithm
	recovered := ECEFToGeodetic(ecef)

	if math.Abs(recovered.LatDeg-eiffel.LatDeg) > 1e-7 {
		t.Fatalf("Latitude roundtrip mismatch: expected %v, got %v", eiffel.LatDeg, recovered.LatDeg)
	}
	if math.Abs(recovered.LonDeg-eiffel.LonDeg) > 1e-7 {
		t.Fatalf("Longitude roundtrip mismatch: expected %v, got %v", eiffel.LonDeg, recovered.LonDeg)
	}
	if math.Abs(recovered.AltM-eiffel.AltM) > 1e-3 {
		t.Fatalf("Altitude roundtrip mismatch: expected %v, got %v", eiffel.AltM, recovered.AltM)
	}
}

func TestHaversineGreatCircleDistance(t *testing.T) {
	// London (51.5074° N, 0.1278° W) to Paris (48.8566° N, 2.3522° E)
	london := GeodeticCoord{LatDeg: 51.5074, LonDeg: -0.1278}
	paris := GeodeticCoord{LatDeg: 48.8566, LonDeg: 2.3522}

	distM := HaversineDistance(london, paris)
	// Known distance is ~343 to 344 km
	distKm := distM / 1000.0
	if distKm < 340.0 || distKm > 346.0 {
		t.Fatalf("Great circle distance out of expected range: %v km", distKm)
	}
}

func TestSlippyTileAddressing(t *testing.T) {
	// Greenwich Meridian / Equator at zoom 10
	tile := LatLonToSlippyTile(0.0, 0.0, 10)
	if tile.Zoom != 10 {
		t.Fatalf("Expected zoom 10, got %d", tile.Zoom)
	}

	// Midpoint tile at zoom 10 is 512, 512
	if tile.X != 512 || tile.Y != 512 {
		t.Fatalf("Expected tile (512, 512), got (%d, %d)", tile.X, tile.Y)
	}

	north, west, south, east := SlippyTileToBounds(tile)
	if west > 0 || east < 0 {
		t.Fatalf("Tile bounds should straddle prime meridian: W=%v, E=%v", west, east)
	}
	if north < 0 || south > 0 {
		t.Fatalf("Tile bounds should straddle equator: N=%v, S=%v", north, south)
	}
}

func TestSphericalPolygonArea(t *testing.T) {
	// 1 degree square at the equator (roughly 111 km x 111 km ≈ 12,300 km^2)
	quad := []GeodeticCoord{
		{LatDeg: 0, LonDeg: 0},
		{LatDeg: 0, LonDeg: 1},
		{LatDeg: 1, LonDeg: 1},
		{LatDeg: 1, LonDeg: 0},
	}

	areaM2, err := SphericalPolygonArea(quad)
	if err != nil {
		t.Fatalf("Polygon area calculation failed: %v", err)
	}

	areaKm2 := areaM2 / 1e6
	// Expected ~12,300 to 12,400 km^2
	if areaKm2 < 12000 || areaKm2 > 12500 {
		t.Fatalf("Polygon area mismatch: expected ~12364 km^2, got %v km^2", areaKm2)
	}
}
