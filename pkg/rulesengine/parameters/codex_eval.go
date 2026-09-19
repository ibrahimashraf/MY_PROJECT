package parameters

import (
	"fmt"
	"math"
	"sync"

	"cel.dev/cel-go/cel"
)

// CEL Constants for codex parameters
const (
	StabilityTippingExpression = `load_percent <= tipping_percent_max`
	WindSpeedExpression        = `wind_speed_mph <= wind_speed_max`
	TemperatureExpression      = `temperature_f >= temp_min && temperature_f <= temp_max`
	SafetyFactorExpression     = `safety_factor >= safety_factor_min`
)

var (
	stabilityProgram    cel.Program
	windSpeedProgram    cel.Program
	temperatureProgram  cel.Program
	safetyFactorProgram cel.Program
	initCELOnce         sync.Once
	initCELErr          error
)

func initCodexCEL() error {
	initCELOnce.Do(func() {
		// 1. Stability
		env1, err := cel.NewEnv(
			cel.Variable("load_percent", cel.DoubleType),
			cel.Variable("tipping_percent_max", cel.DoubleType),
		)
		if err != nil {
			initCELErr = err
			return
		}
		ast1, iss1 := env1.Compile(StabilityTippingExpression)
		if iss1.Err() != nil {
			initCELErr = iss1.Err()
			return
		}
		if stabilityProgram, err = env1.Program(ast1); err != nil {
			initCELErr = err
			return
		}

		// 2. Wind Speed
		env2, err := cel.NewEnv(
			cel.Variable("wind_speed_mph", cel.DoubleType),
			cel.Variable("wind_speed_max", cel.DoubleType),
		)
		if err != nil {
			initCELErr = err
			return
		}
		ast2, iss2 := env2.Compile(WindSpeedExpression)
		if iss2.Err() != nil {
			initCELErr = iss2.Err()
			return
		}
		if windSpeedProgram, err = env2.Program(ast2); err != nil {
			initCELErr = err
			return
		}

		// 3. Temperature
		env3, err := cel.NewEnv(
			cel.Variable("temperature_f", cel.DoubleType),
			cel.Variable("temp_min", cel.DoubleType),
			cel.Variable("temp_max", cel.DoubleType),
		)
		if err != nil {
			initCELErr = err
			return
		}
		ast3, iss3 := env3.Compile(TemperatureExpression)
		if iss3.Err() != nil {
			initCELErr = iss3.Err()
			return
		}
		if temperatureProgram, err = env3.Program(ast3); err != nil {
			initCELErr = err
			return
		}

		// 4. Safety Factor
		env4, err := cel.NewEnv(
			cel.Variable("safety_factor", cel.DoubleType),
			cel.Variable("safety_factor_min", cel.DoubleType),
		)
		if err != nil {
			initCELErr = err
			return
		}
		ast4, iss4 := env4.Compile(SafetyFactorExpression)
		if iss4.Err() != nil {
			initCELErr = iss4.Err()
			return
		}
		if safetyFactorProgram, err = env4.Program(ast4); err != nil {
			initCELErr = err
			return
		}
	})
	return initCELErr
}

type StabilityInput struct {
	LoadPercent       float64 `json:"load_percent"`
	TippingPercentMax float64 `json:"tipping_percent_max"`
}

type WindSpeedInput struct {
	WindSpeedMPH float64 `json:"wind_speed_mph"`
	WindSpeedMax float64 `json:"wind_speed_max"`
}

type TemperatureInput struct {
	TemperatureF float64 `json:"temperature_f"`
	TempMin      float64 `json:"temp_min"`
	TempMax      float64 `json:"temp_max"`
}

type SafetyFactorInput struct {
	SafetyFactor    float64 `json:"safety_factor"`
	SafetyFactorMin float64 `json:"safety_factor_min"`
}

type CodexVerdict struct {
	Passed     bool
	Violations []string
}

func checkNaNs(name string, vals ...float64) error {
	for _, v := range vals {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("parameters: NaN/Inf %s rejected", name)
		}
	}
	return nil
}

func evaluateProgram(prg cel.Program, vars map[string]interface{}) (bool, error) {
	out, _, err := prg.Eval(vars)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate CEL expression: %w", err)
	}

	passed, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("CEL expression did not return boolean")
	}

	return passed, nil
}

func EvaluateStability(input StabilityInput) (CodexVerdict, error) {
	if err := checkNaNs("stability", input.LoadPercent, input.TippingPercentMax); err != nil {
		return CodexVerdict{}, err
	}
	if err := initCodexCEL(); err != nil {
		return CodexVerdict{}, err
	}

	passed, err := evaluateProgram(stabilityProgram, map[string]interface{}{
		"load_percent":        input.LoadPercent,
		"tipping_percent_max": input.TippingPercentMax,
	})

	verdict := CodexVerdict{Passed: passed}
	if !passed {
		verdict.Violations = append(verdict.Violations, "Load percent exceeds maximum tipping percent threshold")
	}
	return verdict, err
}

func EvaluateWindSpeed(input WindSpeedInput) (CodexVerdict, error) {
	if err := checkNaNs("wind_speed", input.WindSpeedMPH, input.WindSpeedMax); err != nil {
		return CodexVerdict{}, err
	}
	if err := initCodexCEL(); err != nil {
		return CodexVerdict{}, err
	}

	passed, err := evaluateProgram(windSpeedProgram, map[string]interface{}{
		"wind_speed_mph": input.WindSpeedMPH,
		"wind_speed_max": input.WindSpeedMax,
	})

	verdict := CodexVerdict{Passed: passed}
	if !passed {
		verdict.Violations = append(verdict.Violations, "Wind speed exceeds maximum cutoff threshold")
	}
	return verdict, err
}

func EvaluateTemperature(input TemperatureInput) (CodexVerdict, error) {
	if err := checkNaNs("temperature", input.TemperatureF, input.TempMin, input.TempMax); err != nil {
		return CodexVerdict{}, err
	}
	if err := initCodexCEL(); err != nil {
		return CodexVerdict{}, err
	}

	passed, err := evaluateProgram(temperatureProgram, map[string]interface{}{
		"temperature_f": input.TemperatureF,
		"temp_min":      input.TempMin,
		"temp_max":      input.TempMax,
	})

	verdict := CodexVerdict{Passed: passed}
	if !passed {
		verdict.Violations = append(verdict.Violations, "Operating temperature out of bounds")
	}
	return verdict, err
}

func EvaluateSafetyFactor(input SafetyFactorInput) (CodexVerdict, error) {
	if err := checkNaNs("safety_factor", input.SafetyFactor, input.SafetyFactorMin); err != nil {
		return CodexVerdict{}, err
	}
	if err := initCodexCEL(); err != nil {
		return CodexVerdict{}, err
	}

	passed, err := evaluateProgram(safetyFactorProgram, map[string]interface{}{
		"safety_factor":     input.SafetyFactor,
		"safety_factor_min": input.SafetyFactorMin,
	})

	verdict := CodexVerdict{Passed: passed}
	if !passed {
		verdict.Violations = append(verdict.Violations, "Safety factor is below minimum required")
	}
	return verdict, err
}

// CodexStabilityGateVars declares CEL typed variable mappings for stability tipping rules.
func CodexStabilityGateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"load_percent":        cel.DoubleType,
		"tipping_percent_max": cel.DoubleType,
	}
}

// CodexWindSpeedGateVars declares CEL typed variable mappings for wind cutoff rules.
func CodexWindSpeedGateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"wind_speed_mph": cel.DoubleType,
		"wind_speed_max": cel.DoubleType,
	}
}

// CodexTemperatureGateVars declares CEL typed variable mappings for temperature rules.
func CodexTemperatureGateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"temperature_f": cel.DoubleType,
		"temp_min":      cel.DoubleType,
		"temp_max":      cel.DoubleType,
	}
}

// CodexSafetyFactorGateVars declares CEL typed variable mappings for safety factor rules.
func CodexSafetyFactorGateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"safety_factor":     cel.DoubleType,
		"safety_factor_min": cel.DoubleType,
	}
}
