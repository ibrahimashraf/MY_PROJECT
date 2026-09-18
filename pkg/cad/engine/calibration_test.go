package engine

import (
	"math"
	"testing"

	"integin/pkg/cad/dxf"
)

func TestNewCalibration(t *testing.T) {
	p1 := dxf.Point3D{X: 0, Y: 0, Z: 0}
	p2 := dxf.Point3D{X: 100, Y: 0, Z: 0}

	// 100 drawing units = 10.0 real meters -> scale = 0.1 m/unit
	cal, err := NewCalibration(p1, p2, 10.0)
	if err != nil {
		t.Fatalf("Expected valid calibration, got err: %v", err)
	}
	if math.Abs(cal.ScaleFactor-0.1) > 1e-9 {
		t.Errorf("Expected scale 0.1, got %f", cal.ScaleFactor)
	}

	// Conversion round-trip
	realM := cal.DrawingToReal(250.0)
	if math.Abs(realM-25.0) > 1e-9 {
		t.Errorf("Expected 25.0m, got %f", realM)
	}
	drawUnits := cal.RealToDrawing(25.0)
	if math.Abs(drawUnits-250.0) > 1e-9 {
		t.Errorf("Expected 250.0 units, got %f", drawUnits)
	}

	// Zero distance error check
	_, errZero := NewCalibration(p1, p1, 10.0)
	if errZero != ErrZeroCalibrationDistance {
		t.Errorf("Expected ErrZeroCalibrationDistance, got %v", errZero)
	}

	// Invalid real distance checks
	_, errNeg := NewCalibration(p1, p2, -5.0)
	if errNeg != ErrInvalidRealDistance {
		t.Errorf("Expected ErrInvalidRealDistance for negative, got %v", errNeg)
	}
	_, errNaN := NewCalibration(p1, p2, math.NaN())
	if errNaN != ErrInvalidRealDistance {
		t.Errorf("Expected ErrInvalidRealDistance for NaN, got %v", errNaN)
	}
}

func TestMeasureDistance(t *testing.T) {
	p1 := dxf.Point3D{X: 10, Y: 20, Z: 0}
	p2 := dxf.Point3D{X: 40, Y: 60, Z: 0}

	// dx = 30, dy = 40 => dist = 50
	mNoCal := MeasureDistance(p1, p2, nil)
	if math.Abs(mNoCal.DrawingDistance-50.0) > 1e-9 {
		t.Errorf("Expected distance 50, got %f", mNoCal.DrawingDistance)
	}
	if math.Abs(mNoCal.RealDistanceMeters-50.0) > 1e-9 {
		t.Errorf("Expected real distance 50 without cal, got %f", mNoCal.RealDistanceMeters)
	}

	cal, err := NewCalibration(dxf.Point3D{X: 0, Y: 0, Z: 0}, dxf.Point3D{X: 100, Y: 0, Z: 0}, 2.0)
	if err != nil {
		t.Fatalf("Failed calibration: %v", err)
	}
	mCal := MeasureDistance(p1, p2, cal)
	// 50 drawing units * (2 / 100) = 1.0 meter
	if math.Abs(mCal.RealDistanceMeters-1.0) > 1e-9 {
		t.Errorf("Expected calibrated distance 1.0m, got %f", mCal.RealDistanceMeters)
	}
	if math.Abs(mCal.DeltaX-30.0) > 1e-9 || math.Abs(mCal.DeltaY-40.0) > 1e-9 {
		t.Errorf("Expected deltas (30, 40), got (%f, %f)", mCal.DeltaX, mCal.DeltaY)
	}
}

func TestExtractSnapPoints(t *testing.T) {
	entities := []dxf.Entity{
		&dxf.Line{
			Start: dxf.Point3D{X: 0, Y: 0, Z: 0},
			End:   dxf.Point3D{X: 10, Y: 0, Z: 0},
		},
		&dxf.Circle{
			Center: dxf.Point3D{X: 20, Y: 5, Z: 0},
			Radius: 5.0,
		},
		&dxf.Polyline{
			Vertices: []dxf.Vertex{
				{Point: dxf.Point3D{X: 0, Y: 0, Z: 0}},
				{Point: dxf.Point3D{X: 0, Y: 10, Z: 0}},
				{Point: dxf.Point3D{X: 10, Y: 10, Z: 0}},
			},
			IsClosed: true,
		},
	}

	snaps := ExtractSnapPoints(entities)
	if len(snaps) == 0 {
		t.Fatal("Expected snap points, got 0")
	}

	// Line: 2 endpoints + 1 midpoint = 3
	// Circle: 1 center = 1
	// Polyline (closed, 3 vertices): 3 endpoints + 3 midpoints = 6
	// Total = 10
	if len(snaps) != 10 {
		t.Fatalf("Expected 10 snap points, got %d", len(snaps))
	}

	// Check circle center
	foundCenter := false
	for _, s := range snaps {
		if s.Type == SnapCenter && s.Point.X == 20 && s.Point.Y == 5 {
			foundCenter = true
			break
		}
	}
	if !foundCenter {
		t.Errorf("Failed to find circle center snap point")
	}
}
