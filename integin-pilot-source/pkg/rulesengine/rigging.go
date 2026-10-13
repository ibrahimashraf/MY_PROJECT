package rulesengine

import (
	"fmt"
	"math"
)

// CriticalSlingAngleDeg is the minimum permitted sling angle from horizontal.
// Below this angle the 2D vector tension exceeds the standard derating
// lockout and the lift is refused.
const CriticalSlingAngleDeg = 30.0

// CriticalSlingAngleLockout is the CEL-shaped rule ID surfaced in results.
const CriticalSlingAngleLockout = "rigging.vector.critical-angle-lockout"

// SlingTension2Leg computes the load per leg for a 2-legged sling carrying a
// total load at a given sling angle from the horizontal:
//
//	T = Load / (2 * sin(angle))
//
// The load is shared evenly across symmetric legs. A non-positive angle or a
// load below zero returns a refused result: a 0-degree sling is infinite
// tension (pure horizontal pull) and must lock out the lift.
func SlingTension2Leg(totalLoadT float64, angleFromHorizontalDeg float64) (tensionPerLegT float64, passed bool, lockout bool, err error) {
	if math.IsNaN(totalLoadT) || math.IsNaN(angleFromHorizontalDeg) ||
		math.IsInf(totalLoadT, 0) || math.IsInf(angleFromHorizontalDeg, 0) {
		return 0, false, true, fmt.Errorf("rigging: NaN/Inf input rejected")
	}
	if totalLoadT <= 0 {
		return 0, false, false, fmt.Errorf("rigging: total load must be > 0, got %g", totalLoadT)
	}
	if angleFromHorizontalDeg <= 0 {
		return 0, false, true, fmt.Errorf("rigging: angle must be > 0 deg, got %g (pure horizontal pull)", angleFromHorizontalDeg)
	}
	if math.Sin(angleFromHorizontalDeg*(math.Pi/180)) <= 0 {
		return 0, false, true, fmt.Errorf("rigging: degenerate sling angle %g deg", angleFromHorizontalDeg)
	}
	if angleFromHorizontalDeg < CriticalSlingAngleDeg {
		return 0, false, true, fmt.Errorf("rigging: sling angle %g deg below critical %g deg (derating lockout)", angleFromHorizontalDeg, CriticalSlingAngleDeg)
	}
	tension := totalLoadT / (2 * math.Sin(angleFromHorizontalDeg*(math.Pi/180)))
	return tension, true, false, nil
}
