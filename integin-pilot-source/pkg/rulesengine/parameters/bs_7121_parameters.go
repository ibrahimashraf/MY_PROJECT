package parameters

import (
	"fmt"
	"math"

	"cel.dev/cel-go/cel"
)

// BS 7121 Suite: Code of Practice for Safe Use of Cranes
// Codifies normative parameters across:
// - BS 7121-1:2016 (General - Lift categories, wind cutoffs, proximity, multi-crane derating)
// - BS 7121-2-1:2012 (Inspection, maintenance & thorough examination - General intervals & major reviews)
// - BS 7121-2-3:2012 (Inspection & thorough examination - Mobile cranes, leveling & NDT)
// - BS 7121-2-4:2025 / BS 7121-4:2010 (Loader cranes & lorry loaders, tilt & interlocks)
// - BS 7121-2-7:2012+A1:2015 (Overhead travelling cranes, portal cranes & hoists, brake proof tests)
// - BS 7121-2-9:2013 (Cargo handling & container cranes, twistlocks & storm stowage)
// - BS 7121-3:2017+A1:2019 (Mobile cranes - outrigger mat sizing, pick-and-carry)
// - BS 7121-5:2019 (Tower cranes - in-service wind, climbing cutoff, weather-vaning, clearances)
// - BS 7121-13:2009 (Hydraulic gantry lifting systems - differential height, track slope & deflection)
// - BS 7121-14:2005 (Side-boom pipelayers - tipping stability factor, rope factor & drum wraps)

type BS7121LiftCategory string

const (
	BS7121LiftBasic        BS7121LiftCategory = "BASIC"
	BS7121LiftIntermediate BS7121LiftCategory = "INTERMEDIATE"
	BS7121LiftComplex      BS7121LiftCategory = "COMPLEX"
)

// BS 7121-1:2016 Constants
const (
	BS7121GeneralMaxWindSpeedMS       = 14.0 // Standard operational cutoff (approx 31 mph)
	BS7121SailAreaMaxWindSpeedMS      = 9.8  // High sail area cutoff (> 1.0 m²/tonne)
	BS7121PowerLineClearancePylonM    = 15.0 // Min distance to lines on metal towers/pylons
	BS7121PowerLineClearanceWoodPoleM = 9.0  // Min distance to lines on wooden poles
	BS7121MinTrappingClearanceM       = 0.6  // 600 mm clearance around rotating superstructure
	BS7121MultiCraneDerate2Factor     = 0.80 // 20% down-rating for 2-crane lift (0.80 capacity factor)
	BS7121MultiCraneDerate3Factor     = 0.67 // 33% down-rating for 3+-crane lift (0.67 capacity factor)
)

// BS 7121-2-1:2012 Examination Intervals (LOLER 1998 Reg 9 parity)
const (
	BS7121ThoroughExamIntervalPersonLiftMonths = 6
	BS7121ThoroughExamIntervalAccessoryMonths  = 6
	BS7121ThoroughExamIntervalGoodsCraneMonths = 12
	BS7121MajorReviewIntervalNormalYears       = 10
	BS7121MajorReviewIntervalIntensiveYears    = 5
)

// BS 7121-2-3:2012 Mobile Cranes
const (
	BS7121MobileCraneMaxLevelSlopePct = 0.5 // Level to within ±0.5% slope (Clause 9.4)
	BS7121SupplementaryTestYears      = 4   // 4-yearly supplementary testing
)

// BS 7121-2-4 / BS 7121-4 Loader Cranes & Lorry Loaders
const (
	BS7121LoaderCraneMaxChassisTiltDeg = 5.0 // Maximum chassis tilt angle
	BS7121LoaderCraneStandardTiltDeg   = 3.0 // Recommended operational tilt
)

// BS 7121-2-7:2012+A1:2015 Overhead Cranes
const (
	BS7121OverheadStaticBrakeProofMultiplier  = 1.25 // 125% proof load held 10 min (Clause 9.2.3)
	BS7121OverheadDynamicBrakeProofMultiplier = 1.10 // 110% dynamic test
	BS7121OverheadTripLimiterMultiplier       = 1.10 // 110% overload trip threshold
)

// BS 7121-2-9:2013 Container Cranes
const (
	BS7121ContainerTwistlockMaxCycles      = 250000 // Discard twistlock after 250k cycles
	BS7121ContainerTwistlockMaxWearMM      = 2.0    // Discard twistlock if neck wear > 2.0 mm
	BS7121ContainerGaleClampWindSpeedMS    = 20.0   // Engage storm rail clamps at >= 20 m/s
	BS7121ContainerMaxGroovedFleetAngleDeg = 1.5    // Max 1.5° fleet angle on grooved drums
)

// BS 7121-3:2017+A1:2019 Mobile Cranes
const (
	BS7121MobileCornerReactionFraction  = 0.85 // 85% of total mass on single outrigger during slew
	BS7121PickAndCarryMaxSpeedMS        = 0.40 // 0.4 m/s max travel speed during pick-and-carry
	BS7121PickAndCarryMaxGroundSlopeDeg = 1.0  // 1.0° max travel ground slope
)

// BS 7121-5:2019 Tower Cranes
const (
	BS7121TowerCraneClimbingMaxWindMS       = 10.0 // Max wind speed for climbing/jacking (22 mph)
	BS7121TowerCraneInServiceMaxWindMS      = 15.0 // Standard in-service cutoff (unless OEM chart higher)
	BS7121TowerAntiCollisionHorizClearanceM = 3.0  // Min 3.0 m horizontal clearance between overlapping cranes
	BS7121TowerAntiCollisionVertClearanceM  = 2.0  // Min 2.0 m vertical clearance between overlapping cranes
	BS7121AerodromeConsultationRadiusKM     = 6.0  // 6 km aerodrome notification radius
	BS7121AerodromeConsultationHeightM      = 10.0 // 10 m height threshold
)

// BS 7121-13:2009 Hydraulic Gantry Lifting Systems
const (
	BS7121GantryMaxDifferentialHeightMM  = 25.0        // Max 25 mm differential height between gantries
	BS7121GantryMaxDifferentialSpanRatio = 0.0025      // 0.25% of span max differential height
	BS7121GantryMaxTrackSlopePct         = 0.5         // Max 0.5% (1:200) longitudinal slope
	BS7121GantryMaxTrackDeflectionRatio  = 1.0 / 600.0 // Span / 600 max track deflection under load
	BS7121GantryMaxSideShiftLateralRatio = 0.03        // 3% max vertical SWL as lateral load
)

// BS 7121-14:2005 Side-Boom Pipelayers
const (
	BS7121PipelayerStabilitySWLFactor = 0.714 // Rated SWL <= 0.714 * tipping balance load (Clause 4.7)
	BS7121PipelayerISORatedCapacity   = 0.850 // ISO 8813 rating <= 85% of tipping load
	BS7121PipelayerRopeSafetyFactor   = 5.0   // 1/5th ultimate rope strength (FoS 5.0, Clause 4.8)
	BS7121PipelayerMinDrumWraps       = 3     // Minimum 3 full wraps on drum (Clause 5)
	BS7121PipelayerTestSiteSlopePct   = 1.0   // Test site level within 1.0% (Clause 4.3)
)

// LiftComplexityInput represents the parameters determining lift classification in BS 7121-1 Table 2.
type LiftComplexityInput struct {
	KnownWeight          bool    `json:"known_weight"`
	KnownCentreOfGravity bool    `json:"known_cog"`
	DesignatedLiftPoints bool    `json:"designated_lift_points"`
	ContainsFluids       bool    `json:"contains_fluids"`
	IsFragile            bool    `json:"is_fragile"`
	StableWhenLanded     bool    `json:"stable_when_landed"`
	ClearLineOfSight     bool    `json:"clear_line_of_sight"`
	LiftAtHeight         bool    `json:"lift_at_height"`
	NearPowerLines       bool    `json:"near_power_lines"`
	MultipleCranes       bool    `json:"multiple_cranes"`
	PersonLifting        bool    `json:"person_lifting"`
	OverWaterOrRail      bool    `json:"over_water_or_rail"`
	SailAreaRatioM2PerT  float64 `json:"sail_area_ratio_m2_per_t"`
}

// CategorizeBS7121Lift categorizes the lift into Basic, Intermediate, or Complex per BS 7121-1:2016 Table 2.
func CategorizeBS7121Lift(in LiftComplexityInput) BS7121LiftCategory {
	// Any multi-crane, person-lifting, powerline proximity, or complex hazard forces Complex lift
	if in.MultipleCranes || in.PersonLifting || in.NearPowerLines || in.OverWaterOrRail {
		return BS7121LiftComplex
	}

	loadComplexity := 1 // L1: Known weight, designated points, central CoG, no fluids, stable
	if !in.KnownWeight || !in.KnownCentreOfGravity || !in.DesignatedLiftPoints || in.ContainsFluids || in.IsFragile || !in.StableWhenLanded {
		if !in.KnownWeight && !in.KnownCentreOfGravity && !in.DesignatedLiftPoints {
			loadComplexity = 3 // L3: Estimated weight & CoG, no points, fluids/fragile
		} else {
			loadComplexity = 2 // L2: Intermediate load complexity
		}
	}

	envComplexity := 1 // E1: Ground to ground, clear sight, no obstructions
	if !in.ClearLineOfSight || in.LiftAtHeight {
		if !in.ClearLineOfSight && in.LiftAtHeight {
			envComplexity = 3 // E3: Blind landing at height
		} else {
			envComplexity = 2 // E2: Over obstruction or blind landing
		}
	}

	if in.SailAreaRatioM2PerT > 1.0 {
		// High sail area elevates environmental complexity
		if envComplexity < 2 {
			envComplexity = 2
		}
	}

	// Decision Matrix per BS 7121-1:2016 Table 2
	if loadComplexity >= 3 || envComplexity >= 3 {
		return BS7121LiftComplex
	}
	if loadComplexity == 2 || envComplexity == 2 {
		return BS7121LiftIntermediate
	}
	return BS7121LiftBasic
}

// PermissibleWindSpeed determines maximum permissible in-service wind speed per BS 7121-1 Annex D.
func PermissibleWindSpeed(manufacturerMaxMS, sailAreaRatioM2PerT float64) (float64, error) {
	if err := rejectParamNaN("wind speed inputs", manufacturerMaxMS, sailAreaRatioM2PerT); err != nil {
		return 0, err
	}
	if manufacturerMaxMS <= 0 {
		return 0, fmt.Errorf("parameters: manufacturer wind limit must be > 0, got %g", manufacturerMaxMS)
	}
	if sailAreaRatioM2PerT < 0 {
		return 0, fmt.Errorf("parameters: sail area ratio must be >= 0, got %g", sailAreaRatioM2PerT)
	}

	limit := manufacturerMaxMS
	if limit > BS7121GeneralMaxWindSpeedMS {
		limit = BS7121GeneralMaxWindSpeedMS
	}

	if sailAreaRatioM2PerT > 1.0 {
		// Load presents large sail area: clamp to 9.8 m/s per BS 7121-1 Annex D
		if limit > BS7121SailAreaMaxWindSpeedMS {
			limit = BS7121SailAreaMaxWindSpeedMS
		}
	}
	return limit, nil
}

// MultiCraneCapacityFactor returns the mandatory down-rating multiplier for tandem lifts per BS 7121-1 Clause 14.2.9.
func MultiCraneCapacityFactor(craneCount int, allFactorsInstrumented bool) (float64, error) {
	if craneCount < 1 {
		return 0, fmt.Errorf("parameters: crane count must be >= 1, got %d", craneCount)
	}
	if craneCount == 1 {
		return 1.0, nil
	}
	if allFactorsInstrumented {
		// If continuous electronic load/tilt monitoring is in place, up to rated capacity is permitted
		return 1.0, nil
	}
	if craneCount == 2 {
		return BS7121MultiCraneDerate2Factor, nil // 0.80 (20% down-rating)
	}
	return BS7121MultiCraneDerate3Factor, nil // 0.67 (33% down-rating for 3 or more cranes)
}

// EvaluatePipelayerStability checks rated load against tipping load per BS 7121-14 Clause 4.7.
func EvaluatePipelayerStability(appliedLoadT, tippingLoadT float64) (bool, float64, error) {
	if err := rejectParamNaN("pipelayer loads", appliedLoadT, tippingLoadT); err != nil {
		return false, 0, err
	}
	if tippingLoadT <= 0 {
		return false, 0, fmt.Errorf("parameters: tipping load must be > 0, got %g", tippingLoadT)
	}
	if appliedLoadT < 0 {
		return false, 0, fmt.Errorf("parameters: applied load must be >= 0, got %g", appliedLoadT)
	}

	maxAllowable := BS7121PipelayerStabilitySWLFactor * tippingLoadT // 0.714 * tippingLoad
	utilization := appliedLoadT / maxAllowable
	return utilization <= 1.0, utilization, nil
}

// EvaluateHydraulicGantryDifferential checks jacking height difference per BS 7121-13 Clause 13.
func EvaluateHydraulicGantryDifferential(diffHeightMM, spanMM float64) (bool, error) {
	if err := rejectParamNaN("gantry dimensions", diffHeightMM, spanMM); err != nil {
		return false, err
	}
	if diffHeightMM < 0 {
		return false, fmt.Errorf("parameters: differential height must be >= 0, got %g", diffHeightMM)
	}
	if spanMM <= 0 {
		return false, fmt.Errorf("parameters: gantry span must be > 0, got %g", spanMM)
	}

	maxBySpan := spanMM * BS7121GantryMaxDifferentialSpanRatio
	allowable := math.Min(BS7121GantryMaxDifferentialHeightMM, maxBySpan)

	if diffHeightMM > allowable {
		return false, fmt.Errorf("BS 7121-13: differential height %.1f mm exceeds allowable %.1f mm", diffHeightMM, allowable)
	}
	return true, nil
}

// EvaluateTowerCraneClimbingWind verifies wind speed prior to tower crane climbing/jacking per BS 7121-5.
func EvaluateTowerCraneClimbingWind(measuredWindMS float64) (bool, error) {
	if err := rejectParamNaN("climbing wind", measuredWindMS); err != nil {
		return false, err
	}
	if measuredWindMS < 0 {
		return false, fmt.Errorf("parameters: wind speed must be >= 0, got %g", measuredWindMS)
	}
	if measuredWindMS > BS7121TowerCraneClimbingMaxWindMS {
		return false, fmt.Errorf("BS 7121-5: measured wind %.1f m/s exceeds climbing ceiling 10.0 m/s (climbing prohibited)", measuredWindMS)
	}
	return true, nil
}

// EvaluateMobileCraneLevel checks setup levelness per BS 7121-2-3 Clause 9.4 (within ±0.5% slope).
func EvaluateMobileCraneLevel(slopePct float64) (bool, error) {
	if err := rejectParamNaN("crane slope", slopePct); err != nil {
		return false, err
	}
	absSlope := math.Abs(slopePct)
	if absSlope > BS7121MobileCraneMaxLevelSlopePct {
		return false, fmt.Errorf("BS 7121-2-3 §9.4: crane slope %.2f%% exceeds ±0.50%% levelling tolerance", absSlope)
	}
	return true, nil
}

// BS7121GantryTrackPressure calculates track ground bearing pressure per BS 7121-13 Clause 12.
func BS7121GantryTrackPressure(verticalLoadT, bearingAreaM2 float64) (float64, error) {
	if err := rejectParamNaN("gantry track pressure", verticalLoadT, bearingAreaM2); err != nil {
		return 0, err
	}
	if verticalLoadT <= 0 {
		return 0, fmt.Errorf("parameters: vertical load must be > 0, got %g", verticalLoadT)
	}
	if bearingAreaM2 <= 0 {
		return 0, fmt.Errorf("parameters: bearing area must be > 0, got %g", bearingAreaM2)
	}
	return verticalLoadT / bearingAreaM2, nil
}

// BS7121PipelayerLoweringInCradleLoad calculates pipelayer load per cradle during pipeline lowering-in per BS 7121-14 Clause 10.
func BS7121PipelayerLoweringInCradleLoad(pipeUnitWeightKgPerM, cradleSpacingM, shockFactor float64) (float64, error) {
	if err := rejectParamNaN("pipelayer lowering-in load", pipeUnitWeightKgPerM, cradleSpacingM, shockFactor); err != nil {
		return 0, err
	}
	if pipeUnitWeightKgPerM <= 0 {
		return 0, fmt.Errorf("parameters: pipe unit weight must be > 0, got %g", pipeUnitWeightKgPerM)
	}
	if cradleSpacingM <= 0 {
		return 0, fmt.Errorf("parameters: cradle spacing must be > 0, got %g", cradleSpacingM)
	}
	if shockFactor < 1.0 {
		return 0, fmt.Errorf("parameters: shock factor must be >= 1.0, got %g", shockFactor)
	}
	// Total load in tonnes = (weight/m * spacing * shockFactor) / 1000
	return (pipeUnitWeightKgPerM * cradleSpacingM * shockFactor) / 1000.0, nil
}

// BS7121GateVars declares CEL typed variable mappings for BS 7121 rules.
func BS7121GateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"wind_speed_ms":        cel.DoubleType,
		"max_permissible_wind": cel.DoubleType,
		"clearance_m":          cel.DoubleType,
		"crane_slope_pct":      cel.DoubleType,
		"gantry_diff_mm":       cel.DoubleType,
		"pipelayer_util":       cel.DoubleType,
	}
}
