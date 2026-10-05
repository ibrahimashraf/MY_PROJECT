package piping

import (
	"errors"
	"math"
	"time"
)

var (
	ErrWallThicknessCritical = errors.New("piping: measured thickness is below minimum required wall thickness (quarantine triggered)")
	ErrNegativePressure      = errors.New("piping: internal design pressure must be positive")
	ErrZeroDiameter          = errors.New("piping: outer diameter must be positive")
	ErrZeroAllowableStress   = errors.New("piping: material allowable stress must be positive")
)

// CircuitParameters defines ASME B31.3 / API 570 piping circuit properties.
type CircuitParameters struct {
	CircuitDID            string    `json:"circuit_did"` // did:integin:circuit:<uuid>
	OuterDiameterMM       float64   `json:"outer_diameter_mm"`
	DesignPressureBar     float64   `json:"design_pressure_bar"`
	AllowableStressMPa    float64   `json:"allowable_stress_mpa"` // Material stress limit S
	JointEfficiencyE      float64   `json:"joint_efficiency_e"`    // e.g. 1.0 for seamless, 0.85 for ERW
	CoefficientY          float64   `json:"coefficient_y"`         // ASME table 304.1.1 (typically 0.4 for ferritic steels below 482C)
	CorrosionAllowanceMM  float64   `json:"corrosion_allowance_mm"`
	NominalThicknessMM    float64   `json:"nominal_thickness_mm"`
}

// MeasurementPoint represents an ultrasonic thickness gauging (UTG) monitoring location.
type MeasurementPoint struct {
	PointID               string    `json:"point_id"`
	CurrentThicknessMM    float64   `json:"current_thickness_mm"`
	PreviousThicknessMM   float64   `json:"previous_thickness_mm"`
	YearsBetweenReadings  float64   `json:"years_between_readings"`
}

// AssessmentResult encapsulates remaining life and retirement thickness calculations.
type AssessmentResult struct {
	CircuitDID             string    `json:"circuit_did"`
	MinimumRequiredTMM     float64   `json:"minimum_required_t_mm"` // t_min
	ShortTermCorrosionRate float64   `json:"short_term_corrosion_rate_mm_yr"`
	RemainingLifeYears     float64   `json:"remaining_life_years"`
	NextInspectionInterval float64   `json:"next_inspection_interval_years"` // Half-life rule per API 570
	IsSafeToOperate        bool      `json:"is_safe_to_operate"`
	QuarantineRequired     bool      `json:"quarantine_required"`
	EvaluatedAt            time.Time `json:"evaluated_at"`
}

// EvaluatePipingCircuit executes ASME B31.3 Eq. 3a pressure design wall thickness and API 570 life calculations.
// Formula: t_min = (P * D) / (2 * (S * E + P * Y))
func EvaluatePipingCircuit(cfg CircuitParameters, pt MeasurementPoint) (AssessmentResult, error) {
	if cfg.OuterDiameterMM <= 0 {
		return AssessmentResult{}, ErrZeroDiameter
	}
	if cfg.DesignPressureBar <= 0 {
		return AssessmentResult{}, ErrNegativePressure
	}
	if cfg.AllowableStressMPa <= 0 {
		return AssessmentResult{}, ErrZeroAllowableStress
	}

	// Convert pressure from Bar to MPa (1 Bar = 0.1 MPa)
	pMPa := cfg.DesignPressureBar * 0.1
	dMM := cfg.OuterDiameterMM
	sMPa := cfg.AllowableStressMPa
	e := cfg.JointEfficiencyE
	if e <= 0 {
		e = 1.0
	}
	y := cfg.CoefficientY
	if y <= 0 {
		y = 0.4
	}

	// ASME B31.3 Section 304.1.2 Eq. 3a: t = (P * D) / (2 * (S * E + P * Y))
	numerator := pMPa * dMM
	denominator := 2.0 * ((sMPa * e) + (pMPa * y))
	tDesignMM := numerator / denominator
	tMinMM := tDesignMM // Base structural minimum before corrosion allowance

	// Corrosion rate: CR = (t_previous - t_current) / time
	crMMYr := 0.0
	if pt.YearsBetweenReadings > 0 && pt.PreviousThicknessMM > pt.CurrentThicknessMM {
		crMMYr = (pt.PreviousThicknessMM - pt.CurrentThicknessMM) / pt.YearsBetweenReadings
	}

	// Remaining Life: RL = (t_actual - t_min) / CR
	remainingLifeYears := 99.0
	if crMMYr > 0 {
		remainingLifeYears = (pt.CurrentThicknessMM - tMinMM) / crMMYr
		if remainingLifeYears < 0 {
			remainingLifeYears = 0
		}
	}

	// API 570 Half-life rule: Next inspection interval = min(RL / 2, 5.0 years)
	nextInterval := remainingLifeYears / 2.0
	if nextInterval > 5.0 {
		nextInterval = 5.0
	}
	if nextInterval < 0.5 {
		nextInterval = 0.5
	}

	breached := pt.CurrentThicknessMM < tMinMM

	res := AssessmentResult{
		CircuitDID:             cfg.CircuitDID,
		MinimumRequiredTMM:     math.Round(tMinMM*1000) / 1000,
		ShortTermCorrosionRate: math.Round(crMMYr*1000) / 1000,
		RemainingLifeYears:     math.Round(remainingLifeYears*10) / 10,
		NextInspectionInterval: math.Round(nextInterval*10) / 10,
		IsSafeToOperate:        !breached,
		QuarantineRequired:     breached,
		EvaluatedAt:            time.Now().UTC(),
	}

	if breached {
		return res, ErrWallThicknessCritical
	}

	return res, nil
}
