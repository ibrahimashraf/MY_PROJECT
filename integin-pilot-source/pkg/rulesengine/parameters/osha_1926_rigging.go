package parameters

import (
	"fmt"
	"math"

	"cel.dev/cel-go/cel"
)

// OSHA 29 CFR 1926.251: Rigging Equipment for Material Handling
//
// Minimum Design Factors:
// - Wire rope slings: 5.0 (5:1)
// - Alloy steel chain slings: 4.0 (4:1)
// - Synthetic web & round slings: 5.0 (5:1)
// - Rigging hardware & shackles: 5.0 (5:1)
//
// Operating Temperature Ceilings:
// - Synthetic web (nylon/polyester): 180°F (82.2°C) max
// - Fiber core wire rope: 180°F (82.2°C) max
// - IWRC wire rope: 400°F (204.4°C) max
// - Alloy steel chain: 400°F (204.4°C) without derating; 1000°F (537.8°C) absolute maximum with manufacturer derating; > 1000°F prohibited.
//
// Critical Sling Angle Lockout:
// - Sling angle to horizontal theta < 30.0 degrees is prohibited.

type RiggingSlingType string

const (
	SlingTypeWireRopeIWRC RiggingSlingType = "WIRE_ROPE_IWRC"
	SlingTypeWireRopeFC   RiggingSlingType = "WIRE_ROPE_FC"
	SlingTypeAlloyChain   RiggingSlingType = "ALLOY_STEEL_CHAIN"
	SlingTypeSyntheticWeb RiggingSlingType = "SYNTHETIC_WEB"
	SlingTypeSyntheticRnd RiggingSlingType = "SYNTHETIC_ROUND"
)

const (
	OSHAMinDesignFactorWireRope  = 5.0
	OSHAMinDesignFactorChain     = 4.0
	OSHAMinDesignFactorSynthetic = 5.0

	OSHACriticalSlingAngleDeg = 30.0 // Minimum allowable horizontal sling angle
	OSHAMaxTempSyntheticWebF  = 180.0
	OSHAMaxTempFiberCoreWireF = 180.0
	OSHAMaxTempIWRCWireF      = 400.0
	OSHAMaxTempAlloyChainF    = 1000.0
)

// CEL expressions for OSHA 1926.251
const (
	OSHASlingAngleExpression = `horizontal_angle_deg >= 30.0`
	OSHAChainTempExpression  = `temp_f <= 1000.0`
	OSHASynthTempExpression  = `temp_f <= 180.0`
)

// RequiredDesignFactor returns the mandatory OSHA safety factor for the sling type.
func RequiredDesignFactor(slingType RiggingSlingType) (float64, error) {
	switch slingType {
	case SlingTypeWireRopeIWRC, SlingTypeWireRopeFC:
		return OSHAMinDesignFactorWireRope, nil
	case SlingTypeAlloyChain:
		return OSHAMinDesignFactorChain, nil
	case SlingTypeSyntheticWeb, SlingTypeSyntheticRnd:
		return OSHAMinDesignFactorSynthetic, nil
	default:
		return 0, fmt.Errorf("parameters: unknown rigging sling type %q", slingType)
	}
}

// OSHASlingTension calculates the tension per leg given total weight, leg count, and angle to horizontal.
// Formula: T = W / (N * sin(theta))
func OSHASlingTension(weightT float64, legCount int, angleDeg float64) (float64, error) {
	if err := rejectParamNaN("rigging tension", weightT, angleDeg); err != nil {
		return 0, err
	}
	if weightT <= 0 {
		return 0, fmt.Errorf("parameters: load weight must be > 0, got %g", weightT)
	}
	if legCount <= 0 {
		return 0, fmt.Errorf("parameters: leg count must be >= 1, got %d", legCount)
	}
	if angleDeg <= 0 || angleDeg > 90 {
		return 0, fmt.Errorf("parameters: horizontal angle must be in (0, 90], got %g°", angleDeg)
	}
	if angleDeg < OSHACriticalSlingAngleDeg {
		return 0, fmt.Errorf("parameters: OSHA 1926.251(c)(5) violation: sling angle %g° is below critical 30° minimum", angleDeg)
	}

	rad := angleDeg * (math.Pi / 180.0)
	sinVal := math.Sin(rad)
	tension := weightT / (float64(legCount) * sinVal)
	return tension, nil
}

// EvaluateOSHATemperature checks environmental temperature compatibility for rigging hardware.
func EvaluateOSHATemperature(slingType RiggingSlingType, tempF float64) (bool, string, error) {
	if math.IsNaN(tempF) || math.IsInf(tempF, 0) {
		return false, "", fmt.Errorf("parameters: invalid NaN/Inf temperature")
	}

	switch slingType {
	case SlingTypeSyntheticWeb, SlingTypeSyntheticRnd:
		if tempF > OSHAMaxTempSyntheticWebF {
			return false, fmt.Sprintf("OSHA 1926.251(e)(6): synthetic sling exposed to %.1f°F, exceeds 180.0°F limit", tempF), nil
		}
	case SlingTypeWireRopeFC:
		if tempF > OSHAMaxTempFiberCoreWireF {
			return false, fmt.Sprintf("OSHA 1926.251(c)(6): fiber core wire rope exposed to %.1f°F, exceeds 180.0°F limit", tempF), nil
		}
	case SlingTypeWireRopeIWRC:
		if tempF > OSHAMaxTempIWRCWireF {
			return false, fmt.Sprintf("OSHA 1926.251(c)(6): IWRC wire rope exposed to %.1f°F, exceeds 400.0°F limit", tempF), nil
		}
	case SlingTypeAlloyChain:
		if tempF > OSHAMaxTempAlloyChainF {
			return false, fmt.Sprintf("OSHA 1926.251(b)(6): alloy steel chain exposed to %.1f°F, exceeds 1000.0°F permanent discard ceiling", tempF), nil
		}
	default:
		return false, "", fmt.Errorf("parameters: unknown sling type %q", slingType)
	}
	return true, "", nil
}

// OSHARiggingGateVars returns CEL typed variable mappings for rigging checks.
func OSHARiggingGateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"horizontal_angle_deg": cel.DoubleType,
		"temp_f":               cel.DoubleType,
		"design_factor":        cel.DoubleType,
		"leg_count":            cel.IntType,
		"load_weight_t":        cel.DoubleType,
	}
}
