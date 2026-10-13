package physics

import (
	"errors"
	"fmt"
	"math"
)

// WindTerrainCategory classifies surrounding boundary surface roughness per ASCE 7 / Eurocode 1.
type WindTerrainCategory string

const (
	TerrainOffshore    WindTerrainCategory = "OFFSHORE"     // Open sea / coastal water (alpha ~ 0.11)
	TerrainOpenCountry WindTerrainCategory = "OPEN_TERRAIN" // Flat open ground, grasslands (alpha ~ 0.14)
	TerrainSuburban    WindTerrainCategory = "SUBURBAN"     // Suburban, industrial estates (alpha ~ 0.22)
	TerrainUrban       WindTerrainCategory = "URBAN"        // Dense city centers, tall buildings (alpha ~ 0.33)
)

// WindProfileParams defines inputs for altitude-dependent wind speed profile.
type WindProfileParams struct {
	RefVelocityMs float64             `json:"ref_velocity_m_s"` // Speed at reference height (e.g. 10m)
	RefHeightM    float64             `json:"ref_height_m"`     // Reference anemometer height (usually 10.0m)
	TargetHeightM float64             `json:"target_height_m"`  // Boom tip / load height z
	Terrain       WindTerrainCategory `json:"terrain_category"` // Terrain roughness category
}

// EvaluateWindProfile evaluates atmospheric boundary layer wind velocity at elevation z:
// v(z) = v_ref * (z / z_ref)^alpha
func EvaluateWindProfile(params WindProfileParams) (float64, error) {
	if params.RefVelocityMs < 0 {
		return 0, errors.New("reference wind velocity must be non-negative")
	}
	if params.RefHeightM <= 0 {
		return 0, errors.New("reference height must be positive")
	}
	if params.TargetHeightM <= 0 {
		return 0, errors.New("target height must be positive")
	}

	var alpha float64
	switch params.Terrain {
	case TerrainOffshore:
		alpha = 0.11
	case TerrainOpenCountry:
		alpha = 0.14
	case TerrainSuburban:
		alpha = 0.22
	case TerrainUrban:
		alpha = 0.33
	default:
		alpha = 0.14 // Default to open terrain
	}

	// Clamp ground layer boundary
	effectiveZ := math.Max(params.TargetHeightM, 2.0)
	vz := params.RefVelocityMs * math.Pow(effectiveZ/params.RefHeightM, alpha)
	return vz, nil
}

// AeroDragResult holds aerodynamic pressure and resultant drag force.
type AeroDragResult struct {
	DynamicPressurePa float64 `json:"dynamic_pressure_pa"` // q = 0.5 * rho * v^2
	DragForceN        float64 `json:"drag_force_n"`        // F_d = q * C_d * A
}

// EvaluateAeroDrag computes aerodynamic drag force on structural members or crane cargo:
// F_d = 0.5 * rho * v^2 * C_d * A
func EvaluateAeroDrag(airDensityKgM3, velocityMs, dragCoeff, projectedAreaM2 float64) (AeroDragResult, error) {
	if projectedAreaM2 <= 0 {
		return AeroDragResult{}, errors.New("projected area must be positive")
	}
	if dragCoeff <= 0 {
		return AeroDragResult{}, errors.New("drag coefficient must be positive")
	}
	rho := airDensityKgM3
	if rho <= 0 {
		rho = SeaLevelAirDensity
	}

	dynP := 0.5 * rho * math.Pow(velocityMs, 2)
	dragF := dynP * dragCoeff * projectedAreaM2

	return AeroDragResult{
		DynamicPressurePa: dynP,
		DragForceN:        dragF,
	}, nil
}

// VIVParams specifies aerodynamic vortex shedding parameters for cylindrical/tubular booms.
type VIVParams struct {
	DiameterM          float64 `json:"diameter_m"`           // Outer diameter D
	WindVelocityMs     float64 `json:"wind_velocity_m_s"`    // Cross-wind speed v
	NaturalFrequencyHz float64 `json:"natural_frequency_hz"` // Boom structural natural frequency f_n
	StrouhalNumber     float64 `json:"strouhal_number"`      // St (typical 0.20 for circular cylinders)
}

// VIVResult contains vortex shedding frequency, lock-in vulnerability, and critical speeds.
type VIVResult struct {
	SheddingFrequencyHz      float64 `json:"shedding_frequency_hz"`
	FrequencyRatio           float64 `json:"frequency_ratio"` // f_s / f_n
	LockInRisk               bool    `json:"lock_in_risk"`    // true if 0.8 <= ratio <= 1.2
	CriticalLockInVelocityMs float64 `json:"critical_lock_in_velocity_m_s"`
}

// EvaluateVIVLockIn checks for vortex shedding lock-in resonance:
// f_s = St * v / D
// Critical velocity: v_crit = f_n * D / St
// Lock-in occurs when f_s matches natural frequency (0.8 <= f_s / f_n <= 1.2).
func EvaluateVIVLockIn(p VIVParams) (VIVResult, error) {
	if p.DiameterM <= 0 {
		return VIVResult{}, errors.New("member diameter must be positive")
	}
	if p.NaturalFrequencyHz <= 0 {
		return VIVResult{}, errors.New("natural frequency must be positive")
	}
	if p.WindVelocityMs < 0 {
		return VIVResult{}, errors.New("wind velocity must be non-negative")
	}
	st := p.StrouhalNumber
	if st <= 0 {
		st = 0.20 // Standard Strouhal number for circular cross-sections
	}

	fs := (st * p.WindVelocityMs) / p.DiameterM
	ratio := fs / p.NaturalFrequencyHz
	vCrit := (p.NaturalFrequencyHz * p.DiameterM) / st

	lockIn := ratio >= 0.8 && ratio <= 1.2

	return VIVResult{
		SheddingFrequencyHz:      fs,
		FrequencyRatio:           ratio,
		LockInRisk:               lockIn,
		CriticalLockInVelocityMs: vCrit,
	}, nil
}

// PDeltaParams defines inputs for second-order aeroelastic P-Delta boom deflection coupling.
type PDeltaParams struct {
	BoomLengthM               float64 `json:"boom_length_m"`
	ElasticModulusPa          float64 `json:"elastic_modulus_pa"`           // E (e.g. 2.1e11 Pa for steel)
	MomentOfInertiaM4         float64 `json:"moment_of_inertia_m4"`         // I
	EffectiveLengthFactor     float64 `json:"effective_length_factor"`      // K (e.g. 1.0)
	AxialCompressiveLoadN     float64 `json:"axial_compressive_load_n"`     // P
	InitialLateralDeflectionM float64 `json:"initial_lateral_deflection_m"` // Delta_0 from lateral wind drag
}

// PDeltaResult holds elastic buckling capacity, deflection amplification, and total lateral offset.
type PDeltaResult struct {
	EulerBucklingLoadN  float64 `json:"euler_buckling_load_n"`
	AmplificationFactor float64 `json:"amplification_factor"`
	TotalDeflectionM    float64 `json:"total_deflection_m"`
	IsBucklingUnstable  bool    `json:"is_buckling_unstable"`
	BucklingUtilization float64 `json:"buckling_utilization"` // P / P_euler
}

// EvaluatePDeltaAmplification calculates second-order P-Delta aeroelastic deflection amplification:
// P_Euler = (pi^2 * E * I) / (K * L)^2
// Amplification = 1 / (1 - P / P_Euler)
// Delta_total = Delta_0 * Amplification
func EvaluatePDeltaAmplification(p PDeltaParams) (PDeltaResult, error) {
	if p.BoomLengthM <= 0 {
		return PDeltaResult{}, errors.New("boom length must be positive")
	}
	if p.ElasticModulusPa <= 0 || p.MomentOfInertiaM4 <= 0 {
		return PDeltaResult{}, errors.New("elastic modulus and moment of inertia must be positive")
	}
	if p.AxialCompressiveLoadN < 0 {
		return PDeltaResult{}, errors.New("axial compressive load must be non-negative")
	}
	k := p.EffectiveLengthFactor
	if k <= 0 {
		k = 1.0
	}

	effectiveL := k * p.BoomLengthM
	pEuler := (math.Pi * math.Pi * p.ElasticModulusPa * p.MomentOfInertiaM4) / (effectiveL * effectiveL)

	utilization := p.AxialCompressiveLoadN / pEuler

	if utilization >= 1.0 {
		return PDeltaResult{
			EulerBucklingLoadN:  pEuler,
			AmplificationFactor: math.Inf(1),
			TotalDeflectionM:    math.Inf(1),
			IsBucklingUnstable:  true,
			BucklingUtilization: utilization,
		}, fmt.Errorf("axial load %v N exceeds Euler buckling load %v N", p.AxialCompressiveLoadN, pEuler)
	}

	amp := 1.0 / (1.0 - utilization)
	totalDelta := p.InitialLateralDeflectionM * amp

	return PDeltaResult{
		EulerBucklingLoadN:  pEuler,
		AmplificationFactor: amp,
		TotalDeflectionM:    totalDelta,
		IsBucklingUnstable:  false,
		BucklingUtilization: utilization,
	}, nil
}
