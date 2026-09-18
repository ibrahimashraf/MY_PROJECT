package parameters_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"integin/pkg/rulesengine/parameters"
)

func TestASMEB30HookEvaluation(t *testing.T) {
	// Nominal base: throat 100 mm, saddle 50 mm
	validInput := parameters.HookInspectionInput{
		NominalThroatMM:  100.0,
		MeasuredThroatMM: 102.0, // 2% stretch <= 5%
		NominalSaddleMM:  50.0,
		MeasuredSaddleMM: 48.0,  // 4% wear <= 10%
		TwistDeg:         3.0,   // 3° <= 10°
		HasCracks:        false,
		LatchOperational: true,
	}

	verdict, err := parameters.EvaluateASMEB30Hook(validInput)
	require.NoError(t, err)
	assert.True(t, verdict.Passed)
	assert.InDelta(t, 2.0, verdict.ThroatStretchPct, 1e-4)
	assert.InDelta(t, 4.0, verdict.SaddleWearPct, 1e-4)
	assert.Empty(t, verdict.Violations)

	// Case 1: Throat stretch breach (> 5%)
	stretched := validInput
	stretched.MeasuredThroatMM = 106.0 // 6% stretch
	verdict, err = parameters.EvaluateASMEB30Hook(stretched)
	require.NoError(t, err)
	assert.False(t, verdict.Passed)
	assert.NotEmpty(t, verdict.Violations)

	// Case 2: Saddle wear breach (> 10%)
	worn := validInput
	worn.MeasuredSaddleMM = 44.0 // 12% wear
	verdict, err = parameters.EvaluateASMEB30Hook(worn)
	require.NoError(t, err)
	assert.False(t, verdict.Passed)

	// Case 3: Twist breach (> 10°)
	twisted := validInput
	twisted.TwistDeg = 11.5
	verdict, err = parameters.EvaluateASMEB30Hook(twisted)
	require.NoError(t, err)
	assert.False(t, verdict.Passed)

	// Case 4: Crack detected
	cracked := validInput
	cracked.HasCracks = true
	verdict, err = parameters.EvaluateASMEB30Hook(cracked)
	require.NoError(t, err)
	assert.False(t, verdict.Passed)

	// Case 5: Latch inoperative
	latchBroken := validInput
	latchBroken.LatchOperational = false
	verdict, err = parameters.EvaluateASMEB30Hook(latchBroken)
	require.NoError(t, err)
	assert.False(t, verdict.Passed)

	// Case 6: NaN/Inf rejection
	nanInput := validInput
	nanInput.MeasuredThroatMM = math.NaN()
	_, err = parameters.EvaluateASMEB30Hook(nanInput)
	assert.Error(t, err)
}

func TestISO5817WeldEvaluation(t *testing.T) {
	// Base: thickness t = 10 mm, width b = 20 mm
	// Level B: undercut <= 0.5 mm, excess <= 3.0 mm (1 + 0.1*20), porosity <= 1.0%
	validLevelB := parameters.WeldInspectionInput{
		Level:               parameters.WeldQualityLevelB,
		NominalThicknessMM:  10.0,
		WeldWidthMM:         20.0,
		MeasuredUndercutMM:  0.4,
		ExcessWeldHeightMM:  2.5,
		PorosityAreaPct:     0.8,
		HasCracks:           false,
		HasLackOfFusion:     false,
		HasIncompletePenetr: false,
	}

	verdict, err := parameters.EvaluateISO5817Weld(validLevelB)
	require.NoError(t, err)
	assert.True(t, verdict.Passed)
	assert.Equal(t, 0.5, verdict.MaxAllowUndercut)
	assert.Equal(t, 3.0, verdict.MaxAllowExcess)

	// Level B breach on undercut (0.6 mm > 0.5 mm)
	breachUndercut := validLevelB
	breachUndercut.MeasuredUndercutMM = 0.6
	verdict, err = parameters.EvaluateISO5817Weld(breachUndercut)
	require.NoError(t, err)
	assert.False(t, verdict.Passed)

	// Same weld in Level C passes (Level C limit is 1.0 mm)
	breachUndercut.Level = parameters.WeldQualityLevelC
	verdict, err = parameters.EvaluateISO5817Weld(breachUndercut)
	require.NoError(t, err)
	assert.True(t, verdict.Passed)

	// Crack strictly prohibited
	crackWeld := validLevelB
	crackWeld.HasCracks = true
	verdict, err = parameters.EvaluateISO5817Weld(crackWeld)
	require.NoError(t, err)
	assert.False(t, verdict.Passed)

	// NaN rejection
	nanWeld := validLevelB
	nanWeld.NominalThicknessMM = math.NaN()
	_, err = parameters.EvaluateISO5817Weld(nanWeld)
	assert.Error(t, err)
}

func TestOSHA1926Rigging(t *testing.T) {
	// Safety Factors
	dfWire, err := parameters.RequiredDesignFactor(parameters.SlingTypeWireRopeIWRC)
	require.NoError(t, err)
	assert.Equal(t, 5.0, dfWire)

	dfChain, err := parameters.RequiredDesignFactor(parameters.SlingTypeAlloyChain)
	require.NoError(t, err)
	assert.Equal(t, 4.0, dfChain)

	dfSynth, err := parameters.RequiredDesignFactor(parameters.SlingTypeSyntheticWeb)
	require.NoError(t, err)
	assert.Equal(t, 5.0, dfSynth)

	// Sling Tension: 10 tonnes, 2 legs at 60°
	// T = 10 / (2 * sin(60°)) = 10 / (2 * 0.866025) = 5.7735 tonnes
	tension60, err := parameters.OSHASlingTension(10.0, 2, 60.0)
	require.NoError(t, err)
	assert.InDelta(t, 5.7735, tension60, 1e-3)

	// Sling Tension: 10 tonnes, 2 legs at 30°
	// T = 10 / (2 * sin(30°)) = 10 / (2 * 0.5) = 10.0 tonnes
	tension30, err := parameters.OSHASlingTension(10.0, 2, 30.0)
	require.NoError(t, err)
	assert.InDelta(t, 10.0, tension30, 1e-3)

	// Sling angle < 30° must fail closed
	_, err = parameters.OSHASlingTension(10.0, 2, 29.5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "critical 30° minimum")

	// Temperature limits
	ok, msg, err := parameters.EvaluateOSHATemperature(parameters.SlingTypeSyntheticWeb, 150.0)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, msg, err = parameters.EvaluateOSHATemperature(parameters.SlingTypeSyntheticWeb, 195.0)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Contains(t, msg, "exceeds 180.0°F limit")

	ok, msg, err = parameters.EvaluateOSHATemperature(parameters.SlingTypeAlloyChain, 1100.0)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Contains(t, msg, "exceeds 1000.0°F permanent discard ceiling")
}

func TestBS7121Parameters(t *testing.T) {
	// 1. Lift Categorization per BS 7121-1 Table 2
	basicInput := parameters.LiftComplexityInput{
		KnownWeight:          true,
		KnownCentreOfGravity: true,
		DesignatedLiftPoints: true,
		ContainsFluids:       false,
		IsFragile:            false,
		StableWhenLanded:     true,
		ClearLineOfSight:     true,
		LiftAtHeight:         false,
		NearPowerLines:       false,
		MultipleCranes:       false,
		PersonLifting:        false,
		OverWaterOrRail:      false,
		SailAreaRatioM2PerT:  0.5,
	}
	assert.Equal(t, parameters.BS7121LiftBasic, parameters.CategorizeBS7121Lift(basicInput))

	// Blind lift over obstruction -> Intermediate
	intermediateInput := basicInput
	intermediateInput.ClearLineOfSight = false
	assert.Equal(t, parameters.BS7121LiftIntermediate, parameters.CategorizeBS7121Lift(intermediateInput))

	// Blind lift at height -> Complex
	complexInput := basicInput
	complexInput.ClearLineOfSight = false
	complexInput.LiftAtHeight = true
	assert.Equal(t, parameters.BS7121LiftComplex, parameters.CategorizeBS7121Lift(complexInput))

	// Multi-crane lift automatically Complex
	tandemInput := basicInput
	tandemInput.MultipleCranes = true
	assert.Equal(t, parameters.BS7121LiftComplex, parameters.CategorizeBS7121Lift(tandemInput))

	// 2. Wind Speeds per BS 7121-1 Annex D
	windNormal, err := parameters.PermissibleWindSpeed(14.0, 0.5)
	require.NoError(t, err)
	assert.Equal(t, 14.0, windNormal)

	// Sail area > 1.0 m²/tonne clamps to 9.8 m/s
	windSail, err := parameters.PermissibleWindSpeed(14.0, 1.8)
	require.NoError(t, err)
	assert.Equal(t, 9.8, windSail)

	// 3. Multi-Crane Down-Rating (Clause 14.2.9)
	f1, err := parameters.MultiCraneCapacityFactor(1, false)
	require.NoError(t, err)
	assert.Equal(t, 1.0, f1)

	f2, err := parameters.MultiCraneCapacityFactor(2, false)
	require.NoError(t, err)
	assert.Equal(t, 0.80, f2)

	f3, err := parameters.MultiCraneCapacityFactor(3, false)
	require.NoError(t, err)
	assert.Equal(t, 0.67, f3)

	fInst, err := parameters.MultiCraneCapacityFactor(2, true)
	require.NoError(t, err)
	assert.Equal(t, 1.0, fInst)

	// 4. Mobile Crane Levelling (BS 7121-2-3 §9.4)
	lvlOk, err := parameters.EvaluateMobileCraneLevel(0.3)
	require.NoError(t, err)
	assert.True(t, lvlOk)

	_, err = parameters.EvaluateMobileCraneLevel(0.65)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds ±0.50% levelling tolerance")

	// 5. Tower Crane Climbing Wind (BS 7121-5)
	climbOk, err := parameters.EvaluateTowerCraneClimbingWind(8.5)
	require.NoError(t, err)
	assert.True(t, climbOk)

	_, err = parameters.EvaluateTowerCraneClimbingWind(11.2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds climbing ceiling 10.0 m/s")

	// 6. Hydraulic Gantry Differential Height (BS 7121-13)
	gantryOk, err := parameters.EvaluateHydraulicGantryDifferential(15.0, 10000.0)
	require.NoError(t, err)
	assert.True(t, gantryOk)

	gantryFail, err := parameters.EvaluateHydraulicGantryDifferential(28.0, 10000.0)
	assert.Error(t, err)
	assert.False(t, gantryFail)

	// 7. Pipelayer Stability SWL (BS 7121-14 §4.7)
	// Tipping load = 50 tonnes -> SWL limit = 0.714 * 50 = 35.7 tonnes
	pipeOk, util, err := parameters.EvaluatePipelayerStability(30.0, 50.0)
	require.NoError(t, err)
	assert.True(t, pipeOk)
	assert.InDelta(t, 30.0/35.7, util, 1e-3)

	pipeFail, _, err := parameters.EvaluatePipelayerStability(38.0, 50.0)
	require.NoError(t, err)
	assert.False(t, pipeFail)

	// 8. Statutory Intervals
	assert.Equal(t, 6, parameters.BS7121ThoroughExamIntervalPersonLiftMonths)
	assert.Equal(t, 6, parameters.BS7121ThoroughExamIntervalAccessoryMonths)
	assert.Equal(t, 12, parameters.BS7121ThoroughExamIntervalGoodsCraneMonths)
	assert.Equal(t, 10, parameters.BS7121MajorReviewIntervalNormalYears)
	assert.Equal(t, 5, parameters.BS7121MajorReviewIntervalIntensiveYears)

	// 9. Gantry Track Pressure (BS 7121-13 §12)
	pressure, err := parameters.BS7121GantryTrackPressure(100.0, 5.0) // 100 t / 5 m² = 20 t/m²
	require.NoError(t, err)
	assert.InDelta(t, 20.0, pressure, 1e-4)

	// 10. Pipelayer Lowering-In Cradle Load (BS 7121-14 §10)
	// 500 kg/m, 12 m spacing, 1.1 shock factor -> (500 * 12 * 1.1) / 1000 = 6.6 tonnes
	cradleLoad, err := parameters.BS7121PipelayerLoweringInCradleLoad(500.0, 12.0, 1.1)
	require.NoError(t, err)
	assert.InDelta(t, 6.6, cradleLoad, 1e-4)
}
