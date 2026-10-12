package rulesengine

import (
	"fmt"
	"math"

	"cel.dev/cel-go/cel"
)

// Spatial clearance rules per OSHA 1926.1408 & ASME B30.5
const (
	SpatialPowerlineGateRuleID = "spatial.powerline-clearance"
	SpatialClearanceExpression = `actual_distance_m >= min_required_clearance_m`
)

// Minimum statutory clearance boundaries (metres)
const (
	// OSHA 1926.1408 Table A: Up to 50 kV requires 10 ft (3.048 m) clearance.
	OSHAMinPowerlineClearance50kVM = 3.048
	// ASME B30.5 §5-3.4.5.1: 2 ft (0.61 m) minimum clearance around crane rotating superstructures.
	ASMEMinClearanceToObstacleM = 0.61
)

// Point3D represents Cartesian coordinate in local site frame (meters).
type Point3D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// DistanceTo calculates Euclidean distance between two spatial points.
func (p Point3D) DistanceTo(other Point3D) float64 {
	dx := p.X - other.X
	dy := p.Y - other.Y
	dz := p.Z - other.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// SpatialClearanceResult records the evaluation of a physical standoff distance.
type SpatialClearanceResult struct {
	ActualDistanceM      float64 `json:"actual_distance_m"`
	MinRequiredDistanceM float64 `json:"min_required_distance_m"`
	Passed               bool    `json:"passed"`
}

// CheckPointClearance verifies standoff between crane point and an exclusion point.
func CheckPointClearance(cranePt, obstaclePt Point3D, minRequiredM float64) (SpatialClearanceResult, error) {
	if math.IsNaN(cranePt.X) || math.IsNaN(cranePt.Y) || math.IsNaN(cranePt.Z) ||
		math.IsNaN(obstaclePt.X) || math.IsNaN(obstaclePt.Y) || math.IsNaN(obstaclePt.Z) ||
		math.IsNaN(minRequiredM) {
		return SpatialClearanceResult{}, fmt.Errorf("spatial: NaN coordinate rejected")
	}
	if minRequiredM <= 0 {
		return SpatialClearanceResult{}, fmt.Errorf("spatial: minimum required clearance must be > 0 m")
	}

	dist := cranePt.DistanceTo(obstaclePt)
	return SpatialClearanceResult{
		ActualDistanceM:      dist,
		MinRequiredDistanceM: minRequiredM,
		Passed:               dist >= minRequiredM,
	}, nil
}

// SpatialPowerlineGateVars returns CEL variable bindings for clearance verification.
func SpatialPowerlineGateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"actual_distance_m":        cel.DoubleType,
		"min_required_clearance_m": cel.DoubleType,
	}
}
