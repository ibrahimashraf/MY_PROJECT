package geology

import (
	"testing"
)

func TestHydraulicErosionSimulation(t *testing.T) {
	w, h := 64, 64
	hm := NewHeightmap(w, h)

	// Create an inclined slope
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			elevation := float64(x)*2.0 + float64(y)*1.5
			hm.Set(x, y, elevation)
		}
	}

	cfg := DefaultErosionConfig()
	// Simulate 500 droplets
	hm.SimulateHydraulicErosion(500, cfg, 42)

	var modifiedCells int
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			original := float64(x)*2.0 + float64(y)*1.5
			if hm.Get(x, y) != original {
				modifiedCells++
			}
		}
	}

	if modifiedCells == 0 {
		t.Fatal("Erosion simulation had no effect on heightmap cells")
	}
}
