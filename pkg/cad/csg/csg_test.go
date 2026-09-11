package csg

import (
	"math"
	"testing"
)

func TestPolygonCSGClipping(t *testing.T) {
	// Subject: 10x10 square from (0,0) to (10,10)
	subject := Polygon{
		Vertices: []Point{
			{X: 0, Y: 0},
			{X: 10, Y: 0},
			{X: 10, Y: 10},
			{X: 0, Y: 10},
		},
	}

	// Clip: 10x10 square from (5,5) to (15,15)
	clip := Polygon{
		Vertices: []Point{
			{X: 5, Y: 5},
			{X: 15, Y: 5},
			{X: 15, Y: 15},
			{X: 5, Y: 15},
		},
	}

	// 1. Check initial areas
	subArea := math.Abs(subject.Area())
	if subArea != 100.0 {
		t.Fatalf("Expected subject area 100.0, got %f", subArea)
	}

	// 2. Intersect via Sutherland-Hodgman
	intersection := ClipConvex(subject, clip)
	intArea := math.Abs(intersection.Area())

	// Overlap should be 5x5 square from (5,5) to (10,10) => Area = 25.0
	expectedArea := 25.0
	if math.Abs(intArea-expectedArea) > 1e-4 {
		t.Fatalf("Expected intersection area %f, got %f", expectedArea, intArea)
	}

	// 3. Verify Point Containment
	if !intersection.ContainsPoint(Point{X: 7.5, Y: 7.5}) {
		t.Errorf("Intersection must contain point (7.5, 7.5)")
	}
	if intersection.ContainsPoint(Point{X: 2.0, Y: 2.0}) {
		t.Errorf("Intersection must NOT contain point (2.0, 2.0)")
	}
}
