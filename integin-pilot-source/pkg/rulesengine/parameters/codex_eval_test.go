package parameters_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"integin/pkg/rulesengine/parameters"
)

func TestEvaluateStability(t *testing.T) {
	// STABILITY_TIPPING_PERCENT_MAX (ASME B30.6 tipping limit <= 85%)
	valid := parameters.StabilityInput{LoadPercent: 80.0, TippingPercentMax: 85.0}
	verdict, err := parameters.EvaluateStability(valid)
	require.NoError(t, err)
	assert.True(t, verdict.Passed)
	assert.Empty(t, verdict.Violations)

	invalid := parameters.StabilityInput{LoadPercent: 90.0, TippingPercentMax: 85.0}
	verdict, err = parameters.EvaluateStability(invalid)
	require.NoError(t, err)
	assert.False(t, verdict.Passed)
	assert.Contains(t, verdict.Violations[0], "Load percent exceeds")

	// NaN test
	_, err = parameters.EvaluateStability(parameters.StabilityInput{LoadPercent: math.NaN(), TippingPercentMax: 85.0})
	require.Error(t, err)
}

func TestEvaluateWindSpeed(t *testing.T) {
	// WIND_SPEED_MAX (ASME B30.8 wind cutoff <= 40 mph)
	valid := parameters.WindSpeedInput{WindSpeedMPH: 35.0, WindSpeedMax: 40.0}
	verdict, err := parameters.EvaluateWindSpeed(valid)
	require.NoError(t, err)
	assert.True(t, verdict.Passed)

	invalid := parameters.WindSpeedInput{WindSpeedMPH: 45.0, WindSpeedMax: 40.0}
	verdict, err = parameters.EvaluateWindSpeed(invalid)
	require.NoError(t, err)
	assert.False(t, verdict.Passed)
}

func TestEvaluateTemperature(t *testing.T) {
	// TEMPERATURE_LIMIT (ASME B30.7 winch operating temp bounds)
	valid := parameters.TemperatureInput{TemperatureF: 100.0, TempMin: -20.0, TempMax: 140.0}
	verdict, err := parameters.EvaluateTemperature(valid)
	require.NoError(t, err)
	assert.True(t, verdict.Passed)

	tooHot := parameters.TemperatureInput{TemperatureF: 150.0, TempMin: -20.0, TempMax: 140.0}
	verdict, err = parameters.EvaluateTemperature(tooHot)
	require.NoError(t, err)
	assert.False(t, verdict.Passed)

	tooCold := parameters.TemperatureInput{TemperatureF: -30.0, TempMin: -20.0, TempMax: 140.0}
	verdict, err = parameters.EvaluateTemperature(tooCold)
	require.NoError(t, err)
	assert.False(t, verdict.Passed)
}

func TestEvaluateSafetyFactor(t *testing.T) {
	// SAFETY_FACTOR_MIN (rigging safety factors)
	valid := parameters.SafetyFactorInput{SafetyFactor: 5.5, SafetyFactorMin: 5.0}
	verdict, err := parameters.EvaluateSafetyFactor(valid)
	require.NoError(t, err)
	assert.True(t, verdict.Passed)

	invalid := parameters.SafetyFactorInput{SafetyFactor: 4.5, SafetyFactorMin: 5.0}
	verdict, err = parameters.EvaluateSafetyFactor(invalid)
	require.NoError(t, err)
	assert.False(t, verdict.Passed)
}

func TestCodexGateVars(t *testing.T) {
	stabilityVars := parameters.CodexStabilityGateVars()
	assert.Contains(t, stabilityVars, "load_percent")
	assert.Contains(t, stabilityVars, "tipping_percent_max")

	windVars := parameters.CodexWindSpeedGateVars()
	assert.Contains(t, windVars, "wind_speed_mph")
	assert.Contains(t, windVars, "wind_speed_max")

	tempVars := parameters.CodexTemperatureGateVars()
	assert.Contains(t, tempVars, "temperature_f")
	assert.Contains(t, tempVars, "temp_min")
	assert.Contains(t, tempVars, "temp_max")

	sfVars := parameters.CodexSafetyFactorGateVars()
	assert.Contains(t, sfVars, "safety_factor")
	assert.Contains(t, sfVars, "safety_factor_min")
}

func TestCodexOverrides(t *testing.T) {
	// Case 1: Overridden Stability Tipping (e.g. Special Engineered Lift Approval)
	overrideInput := parameters.StabilityInput{
		LoadPercent:       88.0,
		TippingPercentMax: 85.0,
		Override: parameters.OverrideToggle{
			AllowOverride: true,
			Reason:        "Engineered heavy lift with outrigger load monitoring and calm wind window",
			AuthorizedBy:  "Eng. Ibrahim (Lead Technical Authority)",
		},
	}
	verdict, err := parameters.EvaluateStability(overrideInput)
	require.NoError(t, err)
	assert.True(t, verdict.Passed)
	assert.True(t, verdict.OverrideApplied)
	assert.NotEmpty(t, verdict.Violations) // Violation is retained for transparent audit trail
	assert.Contains(t, verdict.AuditNote, "OVERRIDE APPLIED")
	assert.Contains(t, verdict.AuditNote, "Eng. Ibrahim")

	// Case 2: Wind Speed Override
	windOverride := parameters.WindSpeedInput{
		WindSpeedMPH: 43.0,
		WindSpeedMax: 40.0,
		Override: parameters.OverrideToggle{
			AllowOverride: true,
			Reason:        "Low-profile concrete block lift at ground level; no sail area",
			AuthorizedBy:  "Appointed Person Site Sign-off",
		},
	}
	verdict, err = parameters.EvaluateWindSpeed(windOverride)
	require.NoError(t, err)
	assert.True(t, verdict.Passed)
	assert.True(t, verdict.OverrideApplied)
	assert.Contains(t, verdict.AuditNote, "wind speed cutoff")
}
