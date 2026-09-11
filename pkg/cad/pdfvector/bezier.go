package pdfvector

import (
	"math"

	"integin/pkg/cad/dxf"
)

// LinearizeCubicBezier converts a cubic Bézier curve into a sequence of line segments.
// P0 is start point, P1 & P2 are control points, P3 is end point.
// chordTolerance is the maximum allowable distance between the curve and approximating chord.
func LinearizeCubicBezier(p0, p1, p2, p3 dxf.Point3D, chordTolerance float64) []dxf.Point3D {
	if chordTolerance <= 0 {
		chordTolerance = 0.05 // default 50 microns / 0.05 mm
	}

	points := []dxf.Point3D{p0}
	subdivideBezier(p0, p1, p2, p3, chordTolerance, 0, 10, &points)
	return points
}

func subdivideBezier(p0, p1, p2, p3 dxf.Point3D, tolerance float64, depth, maxDepth int, out *[]dxf.Point3D) {
	// Calculate flatness (deviation of control points from the baseline P0->P3)
	d1 := pointLineDistance(p1, p0, p3)
	d2 := pointLineDistance(p2, p0, p3)

	if (d1+d2 <= tolerance) || depth >= maxDepth {
		*out = append(*out, p3)
		return
	}

	// De Casteljau midpoint subdivision
	// Level 1 midpoints
	p01 := midpoint(p0, p1)
	p12 := midpoint(p1, p2)
	p23 := midpoint(p2, p3)

	// Level 2 midpoints
	p012 := midpoint(p01, p12)
	p123 := midpoint(p12, p23)

	// Level 3 midpoint on the curve
	p0123 := midpoint(p012, p123)

	// Recursively subdivide left and right halves
	subdivideBezier(p0, p01, p012, p0123, tolerance, depth+1, maxDepth, out)
	subdivideBezier(p0123, p123, p23, p3, tolerance, depth+1, maxDepth, out)
}

func midpoint(a, b dxf.Point3D) dxf.Point3D {
	return dxf.Point3D{
		X: (a.X + b.X) * 0.5,
		Y: (a.Y + b.Y) * 0.5,
		Z: (a.Z + b.Z) * 0.5,
	}
}

func pointLineDistance(p, a, b dxf.Point3D) float64 {
	dx := b.X - a.X
	dy := b.Y - a.Y
	l2 := dx*dx + dy*dy
	if l2 == 0 {
		return math.Hypot(p.X-a.X, p.Y-a.Y)
	}

	// Project p onto line segment ab
	t := ((p.X-a.X)*dx + (p.Y-a.Y)*dy) / l2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}

	projX := a.X + t*dx
	projY := a.Y + t*dy
	return math.Hypot(p.X-projX, p.Y-projY)
}
