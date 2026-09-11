package csg

import "math"

// Point represents a 2D coordinate for CSG polygon clipping.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Polygon represents a closed boundary loop of 2D vertices.
type Polygon struct {
	Vertices []Point `json:"vertices"`
}

// BoundingBox returns the min and max coordinates of the polygon.
func (p Polygon) BoundingBox() (min Point, max Point) {
	if len(p.Vertices) == 0 {
		return Point{}, Point{}
	}
	min = p.Vertices[0]
	max = p.Vertices[0]
	for _, v := range p.Vertices[1:] {
		min.X = math.Min(min.X, v.X)
		min.Y = math.Min(min.Y, v.Y)
		max.X = math.Max(max.X, v.X)
		max.Y = math.Max(max.Y, v.Y)
	}
	return min, max
}

// Area calculates the signed 2D area using the Shoelace formula.
func (p Polygon) Area() float64 {
	n := len(p.Vertices)
	if n < 3 {
		return 0
	}
	area := 0.0
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area += p.Vertices[i].X * p.Vertices[j].Y
		area -= p.Vertices[j].X * p.Vertices[i].Y
	}
	return area * 0.5
}

// ContainsPoint tests whether point pt is strictly inside the polygon using ray-casting.
func (p Polygon) ContainsPoint(pt Point) bool {
	n := len(p.Vertices)
	if n < 3 {
		return false
	}
	inside := false
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		vi, vj := p.Vertices[i], p.Vertices[j]
		if ((vi.Y > pt.Y) != (vj.Y > pt.Y)) &&
			(pt.X < (vj.X-vi.X)*(pt.Y-vi.Y)/(vj.Y-vi.Y)+vi.X) {
			inside = !inside
		}
	}
	return inside
}

// ClipConvex clips polygon subject against a convex clip polygon using Sutherland-Hodgman.
// Pure Go implementation with zero Cgo dependencies (ADV-01 / Hazard 40).
func ClipConvex(subject, clip Polygon) Polygon {
	outputList := subject.Vertices
	if len(outputList) == 0 || len(clip.Vertices) < 3 {
		return Polygon{}
	}

	clipLen := len(clip.Vertices)
	for i := 0; i < clipLen; i++ {
		edgeStart := clip.Vertices[i]
		edgeEnd := clip.Vertices[(i+1)%clipLen]

		inputList := outputList
		outputList = make([]Point, 0, len(inputList))
		if len(inputList) == 0 {
			break
		}

		s := inputList[len(inputList)-1]
		for _, e := range inputList {
			if isInside(e, edgeStart, edgeEnd) {
				if isInside(s, edgeStart, edgeEnd) {
					outputList = append(outputList, e)
				} else {
					outputList = append(outputList, lineIntersection(s, e, edgeStart, edgeEnd))
					outputList = append(outputList, e)
				}
			} else if isInside(s, edgeStart, edgeEnd) {
				outputList = append(outputList, lineIntersection(s, e, edgeStart, edgeEnd))
			}
			s = e
		}
	}

	return Polygon{Vertices: outputList}
}

func isInside(cp, edgeStart, edgeEnd Point) bool {
	// Cross product of edge vector and point vector
	return (edgeEnd.X-edgeStart.X)*(cp.Y-edgeStart.Y)-(edgeEnd.Y-edgeStart.Y)*(cp.X-edgeStart.X) >= -1e-9
}

func lineIntersection(s, e, cp1, cp2 Point) Point {
	a1 := e.Y - s.Y
	b1 := s.X - e.X
	c1 := a1*s.X + b1*s.Y

	a2 := cp2.Y - cp1.Y
	b2 := cp1.X - cp2.X
	c2 := a2*cp1.X + b2*cp1.Y

	det := a1*b2 - a2*b1
	if math.Abs(det) < 1e-9 {
		return s // Parallel
	}

	return Point{
		X: (b2*c1 - b1*c2) / det,
		Y: (a1*c2 - a2*c1) / det,
	}
}
