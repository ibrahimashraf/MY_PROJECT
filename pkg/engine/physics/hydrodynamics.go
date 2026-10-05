package physics

import (
	"errors"
	"math"
)

// OffshoreLiftParams encapsulates inputs for offshore dynamic lift analysis (API 2C & DNV-ST-N001).
type OffshoreLiftParams struct {
	LiftedMassKg       float64 `json:"lifted_mass_kg"`
	RiggingStiffnessNm float64 `json:"rigging_stiffness_n_m"` // Equivalent vertical spring stiffness K_rig
	RelativeVelocityMs float64 `json:"relative_velocity_m_s"` // Sum of crane hook speed + vessel heave velocity
}

// OffshoreDAFResult holds dynamic amplification outputs.
type OffshoreDAFResult struct {
	DAF                float64 `json:"daf"`                  // Dynamic Amplification Factor >= 1.0
	StaticHookLoadN    float64 `json:"static_hook_load_n"`   // m * g
	DynamicHookLoadN   float64 `json:"dynamic_hook_load_n"`  // m * g * DAF
	NaturalFrequencyHz float64 `json:"natural_frequency_hz"` // (1 / 2pi) * sqrt(K / m)
}

// EvaluateOffshoreDAF computes the API 2C / DNV-ST-N001 Dynamic Amplification Factor:
// DAF = 1.0 + v_rel * sqrt(K_rig / (m * g^2))
func EvaluateOffshoreDAF(params OffshoreLiftParams) (OffshoreDAFResult, error) {
	if params.LiftedMassKg <= 0 {
		return OffshoreDAFResult{}, errors.New("lifted mass must be positive")
	}
	if params.RiggingStiffnessNm <= 0 {
		return OffshoreDAFResult{}, errors.New("rigging stiffness must be positive")
	}
	if params.RelativeVelocityMs < 0 {
		return OffshoreDAFResult{}, errors.New("relative velocity must be non-negative")
	}

	m := params.LiftedMassKg
	k := params.RiggingStiffnessNm
	g := StandardGravity
	vRel := params.RelativeVelocityMs

	staticLoadN := m * g

	// DAF = 1.0 + v_rel * sqrt(K / (m * g^2))
	term := math.Sqrt(k / (m * g * g))
	daf := 1.0 + vRel*term

	dynamicLoadN := staticLoadN * daf
	natFreqHz := (1.0 / (2.0 * math.Pi)) * math.Sqrt(k/m)

	return OffshoreDAFResult{
		DAF:                daf,
		StaticHookLoadN:    staticLoadN,
		DynamicHookLoadN:   dynamicLoadN,
		NaturalFrequencyHz: natFreqHz,
	}, nil
}

// SplashZoneParams defines parameters for marine splash zone trajectory and wave slamming.
type SplashZoneParams struct {
	EntryVelocityMs   float64 `json:"entry_velocity_m_s"`
	ProjectedAreaM2   float64 `json:"projected_area_m2"`
	DisplacedVolumeM3 float64 `json:"displaced_volume_m3"`
	WaterDensityKgM3  float64 `json:"water_density_kg_m3"` // Default 1025 kg/m^3
	SlammingCoeff     float64 `json:"slamming_coeff"`      // C_s (typical ~3.14 to 5.0)
	AddedMassCoeff    float64 `json:"added_mass_coeff"`    // C_m (typical ~1.0 to 2.0)
	VerticalAccelMs2  float64 `json:"vertical_accel_m_s2"`
}

// SplashZoneResult holds water-entry hydrodynamic force components.
type SplashZoneResult struct {
	SlammingForceN        float64 `json:"slamming_force_n"`
	AddedMassForceN       float64 `json:"added_mass_force_n"`
	HydrostaticBuoyancyN  float64 `json:"hydrostatic_buoyancy_n"`
	PeakHydrodynamicForce float64 `json:"peak_hydrodynamic_force_n"`
}

// EvaluateSplashZoneTransition computes hydrodynamic wave slamming, added mass entrapment, and buoyancy.
func EvaluateSplashZoneTransition(p SplashZoneParams) (SplashZoneResult, error) {
	if p.ProjectedAreaM2 <= 0 || p.DisplacedVolumeM3 <= 0 {
		return SplashZoneResult{}, errors.New("projected area and displaced volume must be positive")
	}
	rho := p.WaterDensityKgM3
	if rho <= 0 {
		rho = StandardSeawaterDensity
	}
	cs := p.SlammingCoeff
	if cs <= 0 {
		cs = math.Pi // Standard baseline slamming coefficient
	}
	cm := p.AddedMassCoeff
	if cm <= 0 {
		cm = 1.0 // Standard spherical/cylindrical added mass coefficient
	}

	// Hydrodynamic slamming force: F_slam = 0.5 * rho * C_s * A_proj * v_rel^2
	slammingForce := 0.5 * rho * cs * p.ProjectedAreaM2 * math.Pow(p.EntryVelocityMs, 2)

	// Entrapped added mass force: F_add = rho * C_m * V_disp * a
	addedMassForce := rho * cm * p.DisplacedVolumeM3 * math.Abs(p.VerticalAccelMs2)

	// Hydrostatic buoyancy: F_b = rho * g * V_disp
	buoyancy := rho * StandardGravity * p.DisplacedVolumeM3

	peakForce := slammingForce + addedMassForce

	return SplashZoneResult{
		SlammingForceN:        slammingForce,
		AddedMassForceN:       addedMassForce,
		HydrostaticBuoyancyN:  buoyancy,
		PeakHydrodynamicForce: peakForce,
	}, nil
}

// MorisonForceResult decomposes wave forces into drag and inertia.
type MorisonForceResult struct {
	TotalForceN   float64 `json:"total_force_n"`
	DragForceN    float64 `json:"drag_force_n"`
	InertiaForceN float64 `json:"inertia_force_n"`
}

// EvaluateMorisonForce computes wave hydrodynamic inline force on submerged structural members:
// F = F_D + F_M = 0.5 * rho * C_d * A * |u| * u + rho * C_m * V * du/dt
func EvaluateMorisonForce(waterDensityKgM3, dragCoeff, projAreaM2, inertiaCoeff, volumeM3, waterVelocityMs, waterAccelMs2 float64) (MorisonForceResult, error) {
	if projAreaM2 <= 0 || volumeM3 <= 0 {
		return MorisonForceResult{}, errors.New("projected area and volume must be positive")
	}
	rho := waterDensityKgM3
	if rho <= 0 {
		rho = StandardSeawaterDensity
	}

	drag := 0.5 * rho * dragCoeff * projAreaM2 * math.Abs(waterVelocityMs) * waterVelocityMs
	inertia := rho * inertiaCoeff * volumeM3 * waterAccelMs2
	total := drag + inertia

	return MorisonForceResult{
		TotalForceN:   total,
		DragForceN:    drag,
		InertiaForceN: inertia,
	}, nil
}

// SlackSlingSnapRisk checks whether wave heave + buoyancy causes sling slackness (T <= 0),
// creating catastrophic re-engagement shock loads.
type SlackSlingSnapRisk struct {
	MinDynamicTensionN float64 `json:"min_dynamic_tension_n"`
	IsAtRisk           bool    `json:"is_at_risk"`
	SlackMarginN       float64 `json:"slack_margin_n"`
}

// EvaluateSlackSlingSnapRisk evaluates slack sling re-entry risk:
// T_min = m * (g - a_wave_max) - F_buoyancy
// Must satisfy T_min > 0.
func EvaluateSlackSlingSnapRisk(liftedMassKg, displacedVolumeM3, waterDensityKgM3, maxWaveUpwardAccelMs2 float64) (SlackSlingSnapRisk, error) {
	if liftedMassKg <= 0 {
		return SlackSlingSnapRisk{}, errors.New("lifted mass must be positive")
	}
	rho := waterDensityKgM3
	if rho <= 0 {
		rho = StandardSeawaterDensity
	}

	gravityDownward := liftedMassKg * StandardGravity
	inertialUpward := liftedMassKg * maxWaveUpwardAccelMs2
	buoyancyUpward := rho * StandardGravity * displacedVolumeM3

	minTension := gravityDownward - inertialUpward - buoyancyUpward
	isAtRisk := minTension <= 0

	return SlackSlingSnapRisk{
		MinDynamicTensionN: minTension,
		IsAtRisk:           isAtRisk,
		SlackMarginN:       minTension,
	}, nil
}
