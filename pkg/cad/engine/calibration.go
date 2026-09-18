package engine

import (
	"errors"
	"math"

	"integin/pkg/cad/dxf"
)

var (
	ErrZeroCalibrationDistance = errors.New("zero distance between calibration reference points")
	ErrInvalidRealDistance     = errors.New("real distance must be a positive finite number")
)

// Calibration defines a 2-point reference mapping between drawing units and real-world meters.
type Calibration struct {
	Point1             dxf.Point3D `json:"point1"`
	Point2             dxf.Point3D `json:"point2"`
	RealDistanceMeters float64     `json:"real_distance_meters"`
	ScaleFactor        float64     `json:"scale_factor"` // real meters per drawing unit
}

// NewCalibration creates a valid calibration from two reference points and known physical length.
func NewCalibration(p1, p2 dxf.Point3D, realDistanceMeters float64) (*Calibration, error) {
	if realDistanceMeters <= 0 || math.IsNaN(realDistanceMeters) || math.IsInf(realDistanceMeters, 0) {
		return nil, ErrInvalidRealDistance
	}

	dx := p2.X - p1.X
	dy := p2.Y - p1.Y
	dz := p2.Z - p1.Z
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)

	if dist <= 1e-9 {
		return nil, ErrZeroCalibrationDistance
	}

	return &Calibration{
		Point1:             p1,
		Point2:             p2,
		RealDistanceMeters: realDistanceMeters,
		ScaleFactor:        realDistanceMeters / dist,
	}, nil
}

// DrawingToReal converts drawing units to real-world meters.
func (c *Calibration) DrawingToReal(drawingUnits float64) float64 {
	if c == nil {
		return drawingUnits
	}
	return drawingUnits * c.ScaleFactor
}

// RealToDrawing converts real-world meters back to drawing units.
func (c *Calibration) RealToDrawing(realMeters float64) float64 {
	if c == nil || c.ScaleFactor <= 1e-9 {
		return 0
	}
	return realMeters / c.ScaleFactor
}

// MeasurementResult contains measured delta and euclidean lengths.
type MeasurementResult struct {
	DrawingDistance    float64 `json:"drawing_distance"`
	RealDistanceMeters float64 `json:"real_distance_meters"`
	DeltaX             float64 `json:"delta_x"`
	DeltaY             float64 `json:"delta_y"`
	DeltaZ             float64 `json:"delta_z"`
}

// MeasureDistance computes 3D Euclidean and component distances with optional real-world scaling.
func MeasureDistance(p1, p2 dxf.Point3D, cal *Calibration) MeasurementResult {
	dx := p2.X - p1.X
	dy := p2.Y - p1.Y
	dz := p2.Z - p1.Z
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)

	realDist := dist
	if cal != nil {
		realDist = cal.DrawingToReal(dist)
	}

	return MeasurementResult{
		DrawingDistance:    dist,
		RealDistanceMeters: realDist,
		DeltaX:             dx,
		DeltaY:             dy,
		DeltaZ:             dz,
	}
}

// SnapPointType categorizes geometric snap anchors.
type SnapPointType string

const (
	SnapEndpoint SnapPointType = "ENDPOINT"
	SnapMidpoint SnapPointType = "MIDPOINT"
	SnapCenter   SnapPointType = "CENTER"
)

// SnapPoint holds a 3D coordinate, type, and source entity index.
type SnapPoint struct {
	Point       dxf.Point3D   `json:"point"`
	Type        SnapPointType `json:"type"`
	EntityIndex int           `json:"entity_index"`
}

// ExtractSnapPoints traverses CAD entities and extracts precision object-snap points.
func ExtractSnapPoints(entities []dxf.Entity) []SnapPoint {
	var snaps []SnapPoint

	for i, e := range entities {
		switch v := e.(type) {
		case *dxf.Line:
			snaps = append(snaps,
				SnapPoint{Point: v.Start, Type: SnapEndpoint, EntityIndex: i},
				SnapPoint{Point: v.End, Type: SnapEndpoint, EntityIndex: i},
				SnapPoint{
					Point: dxf.Point3D{
						X: (v.Start.X + v.End.X) * 0.5,
						Y: (v.Start.Y + v.End.Y) * 0.5,
						Z: (v.Start.Z + v.End.Z) * 0.5,
					},
					Type:        SnapMidpoint,
					EntityIndex: i,
				},
			)
		case *dxf.Circle:
			snaps = append(snaps, SnapPoint{Point: v.Center, Type: SnapCenter, EntityIndex: i})
		case *dxf.Arc:
			snaps = append(snaps, SnapPoint{Point: v.Center, Type: SnapCenter, EntityIndex: i})
		case *dxf.Polyline:
			n := len(v.Vertices)
			if n == 0 {
				continue
			}
			for vi, vert := range v.Vertices {
				snaps = append(snaps, SnapPoint{Point: vert.Point, Type: SnapEndpoint, EntityIndex: i})
				if vi < n-1 {
					next := v.Vertices[vi+1].Point
					snaps = append(snaps, SnapPoint{
						Point: dxf.Point3D{
							X: (vert.Point.X + next.X) * 0.5,
							Y: (vert.Point.Y + next.Y) * 0.5,
							Z: (vert.Point.Z + next.Z) * 0.5,
						},
						Type:        SnapMidpoint,
						EntityIndex: i,
					})
				} else if v.IsClosed && n > 1 {
					first := v.Vertices[0].Point
					snaps = append(snaps, SnapPoint{
						Point: dxf.Point3D{
							X: (vert.Point.X + first.X) * 0.5,
							Y: (vert.Point.Y + first.Y) * 0.5,
							Z: (vert.Point.Z + first.Z) * 0.5,
						},
						Type:        SnapMidpoint,
						EntityIndex: i,
					})
				}
			}
		}
	}

	return snaps
}
