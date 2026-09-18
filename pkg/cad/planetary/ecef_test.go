package planetary

import (
	"math"
	"testing"
)

func TestECEFRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
		lon  float64
		alt  float64
	}{
		{"Equator", 0, 0, 0},
		{"Poles", 90, 0, 0},
		{"Everest", 27.9881, 86.9250, 8848.86},
		{"Mariana Trench", 11.3493, 142.1996, -10984},
		{"Random City", 40.7128, -74.0060, 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			x, y, z := LatLonAltToECEF(tc.lat, tc.lon, tc.alt)
			lat, lon, alt := ECEFToLatLonAlt(x, y, z)

			latDiff := math.Abs(tc.lat - lat)
			lonDiff := math.Abs(tc.lon - lon)
			altDiff := math.Abs(tc.alt - alt)

			// Sub-millimeter check: ~0.1mm is ~9e-10 degrees at equator
			if latDiff > 1e-8 {
				t.Errorf("Lat round-trip failed: got %v, want %v", lat, tc.lat)
			}
			if tc.name != "Poles" && lonDiff > 1e-8 {
				t.Errorf("Lon round-trip failed: got %v, want %v", lon, tc.lon)
			}
			if altDiff > 0.0001 { // 0.1mm
				t.Errorf("Alt round-trip failed: got %v, want %v", alt, tc.alt)
			}
		})
	}
}

func TestHorizonCull(t *testing.T) {
	// Camera in orbit along +Z at 10,000 km from center
	cam := Camera64{X: 0, Y: 0, Z: 10000000}

	// Node on near side along +Z at equator/surface (should NOT be culled)
	nearNode := [3]float64{0, 0, WGS84b}
	if HorizonCull(cam, nearNode, 1000) {
		t.Errorf("Near side node must not be culled")
	}

	// Node on far side along -Z at surface (MUST be culled)
	farNode := [3]float64{0, 0, -WGS84b}
	if !HorizonCull(cam, farNode, 1000) {
		t.Errorf("Far side node on opposite hemisphere must be culled")
	}
}

func TestBuildTagScheduler(t *testing.T) {
	s := &BuildTagScheduler{}
	n1 := &QuadNode{Level: 1}
	n2 := &QuadNode{Level: 2}
	s.Enqueue(n1)
	s.Enqueue(n2)

	var processed []*QuadNode
	s.Process(func(node *QuadNode) {
		processed = append(processed, node)
	})

	if len(processed) != 2 || processed[0] != n1 || processed[1] != n2 {
		t.Errorf("Queue processing failed: got %v", processed)
	}
	if len(s.Queue) != 0 {
		t.Errorf("Queue must be drained after process, remaining: %d", len(s.Queue))
	}
}
