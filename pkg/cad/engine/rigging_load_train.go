package engine

import (
	"errors"
	"fmt"
	"math"

	"integin/pkg/engine/physics"
	"integin/pkg/engine/symbolic"
)

// PadeyeParams defines geometric and material properties per ASME BTH-1 / AISC 360.
type PadeyeParams struct {
	PinDiameterM     float64 `json:"pin_diameter_m"`
	HoleDiameterM    float64 `json:"hole_diameter_m"`
	MainPlateThickM  float64 `json:"main_plate_thick_m"`
	CheekPlateThickM float64 `json:"cheek_plate_thick_m"` // Sum of cheek plates
	EdgeDistanceM    float64 `json:"edge_distance_m"`     // From hole center to plate edge
	YieldStrengthPa  float64 `json:"yield_strength_pa"`
	WeldLengthM      float64 `json:"weld_length_m"`
	WeldLegSizeM     float64 `json:"weld_leg_size_m"`
}

// PadeyeEvaluationResult holds ASME BTH-1 stress verifications.
type PadeyeEvaluationResult struct {
	BearingStressPa    float64 `json:"bearing_stress_pa"`
	BearingUtilization float64 `json:"bearing_utilization"`
	TearOutStressPa    float64 `json:"tear_out_stress_pa"`
	TearOutUtilization float64 `json:"tear_out_utilization"`
	WeldShearStressPa  float64 `json:"weld_shear_stress_pa"`
	WeldUtilization    float64 `json:"weld_utilization"`
	PassAllChecks      bool    `json:"pass_all_checks"`
}

// EvaluatePadeyeStress checks ASME BTH-1 pin bearing, hole tear-out, and weld shear.
func EvaluatePadeyeStress(p PadeyeParams, appliedLoadN float64) (PadeyeEvaluationResult, error) {
	if appliedLoadN <= 0 {
		return PadeyeEvaluationResult{}, errors.New("applied load must be positive")
	}
	if p.PinDiameterM <= 0 || p.HoleDiameterM < p.PinDiameterM {
		return PadeyeEvaluationResult{}, errors.New("invalid pin or hole diameter")
	}
	tTotal := p.MainPlateThickM + p.CheekPlateThickM
	if tTotal <= 0 {
		return PadeyeEvaluationResult{}, errors.New("plate thickness must be positive")
	}

	// 1. Pin Bearing Stress: sigma_b = P / (d_pin * t_total)
	bearingArea := p.PinDiameterM * tTotal
	bearingStress := appliedLoadN / bearingArea
	allowableBearing := 0.90 * p.YieldStrengthPa
	bearingUtil := bearingStress / allowableBearing

	// 2. Tear-out Shear Stress: tau = P / (2 * (e - d_hole/2) * t_total)
	tearOutDist := p.EdgeDistanceM - (p.HoleDiameterM / 2.0)
	if tearOutDist <= 0 {
		return PadeyeEvaluationResult{}, errors.New("edge distance must be greater than hole radius")
	}
	tearOutArea := 2.0 * tearOutDist * tTotal
	tearOutStress := appliedLoadN / tearOutArea
	allowableShear := 0.40 * p.YieldStrengthPa
	tearOutUtil := tearOutStress / allowableShear

	// 3. Cheek plate / base weld shear
	weldUtil := 0.0
	weldStress := 0.0
	if p.WeldLengthM > 0 && p.WeldLegSizeM > 0 {
		throat := p.WeldLegSizeM * 0.7071
		weldArea := p.WeldLengthM * throat
		weldStress = appliedLoadN / weldArea
		allowableWeld := 0.30 * p.YieldStrengthPa
		weldUtil = weldStress / allowableWeld
	}

	pass := bearingUtil <= 1.0 && tearOutUtil <= 1.0 && weldUtil <= 1.0

	return PadeyeEvaluationResult{
		BearingStressPa:    bearingStress,
		BearingUtilization: bearingUtil,
		TearOutStressPa:    tearOutStress,
		TearOutUtilization: tearOutUtil,
		WeldShearStressPa:  weldStress,
		WeldUtilization:    weldUtil,
		PassAllChecks:      pass,
	}, nil
}

// SpreaderBeamParams defines geometry and cross-section per DIN EN 13155.
type SpreaderBeamParams struct {
	SpanLengthM      float64 `json:"span_length_m"`
	CrossSectionArea float64 `json:"cross_section_area_m2"`
	MomentOfInertia  float64 `json:"moment_of_inertia_m4"`
	ElasticModulusPa float64 `json:"elastic_modulus_pa"`
	YieldStrengthPa  float64 `json:"yield_strength_pa"`
	SelfWeightKgM    float64 `json:"self_weight_kg_m"`
}

// SpreaderBeamResult contains combined buckling and flexural capacity checks.
type SpreaderBeamResult struct {
	EulerBucklingLoadN  float64 `json:"euler_buckling_load_n"`
	AxialStressPa       float64 `json:"axial_stress_pa"`
	BendingStressPa     float64 `json:"bending_stress_pa"`
	CombinedUtilization float64 `json:"combined_utilization"`
	PassCheck           bool    `json:"pass_check"`
}

// EvaluateSpreaderBeam evaluates DIN EN 13155 combined compression + self-weight bending.
func EvaluateSpreaderBeam(p SpreaderBeamParams, axialCompressN float64) (SpreaderBeamResult, error) {
	if p.SpanLengthM <= 0 || p.CrossSectionArea <= 0 || p.MomentOfInertia <= 0 || p.ElasticModulusPa <= 0 {
		return SpreaderBeamResult{}, errors.New("invalid beam parameters")
	}

	// Euler buckling: P_e = pi^2 * E * I / L^2
	pEuler := (math.Pi * math.Pi * p.ElasticModulusPa * p.MomentOfInertia) / math.Pow(p.SpanLengthM, 2)
	axialStress := axialCompressN / p.CrossSectionArea

	// Self-weight bending: M_max = q * L^2 / 8
	q := p.SelfWeightKgM * physics.StandardGravity
	mMax := (q * math.Pow(p.SpanLengthM, 2)) / 8.0

	// Extreme fiber distance: assume symmetric section c = sqrt(I / A) * 1.5 approx
	c := math.Sqrt(p.MomentOfInertia/p.CrossSectionArea) * 1.732
	bendingStress := (mMax * c) / p.MomentOfInertia

	// Interaction equation: P / P_allow + M / M_allow
	bucklingUtil := axialCompressN / (pEuler * 0.5) // Safety factor 2.0
	bendingUtil := bendingStress / (0.66 * p.YieldStrengthPa)
	combinedUtil := bucklingUtil + bendingUtil

	return SpreaderBeamResult{
		EulerBucklingLoadN:  pEuler,
		AxialStressPa:       axialStress,
		BendingStressPa:     bendingStress,
		CombinedUtilization: combinedUtil,
		PassCheck:           combinedUtil <= 1.0,
	}, nil
}

// SlingGrommetParams models sling efficiency per IMCA M 179 / DNV-ST-N001.
type SlingGrommetParams struct {
	NominalMBLN     float64 `json:"nominal_mbl_n"`
	IsCableLaid     bool    `json:"is_cable_laid"`
	ChokeAngleDeg   float64 `json:"choke_angle_deg"` // 180 = straight, <120 = derated
	OperatingTempC  float64 `json:"operating_temp_c"`
	IsSyntheticHMPE bool    `json:"is_synthetic_hmpe"`
}

// SlingEvaluationResult contains derated capacity and safety factor.
type SlingEvaluationResult struct {
	DeratedWLLN     float64 `json:"derated_wll_n"`
	EffectiveFactor float64 `json:"effective_factor"`
	ThermalDerated  bool    `json:"thermal_derated"`
	PassCheck       bool    `json:"pass_check"`
}

// EvaluateSlingGrommet applies cable-laid, choke angle, and thermal cutoffs.
func EvaluateSlingGrommet(p SlingGrommetParams, workingTensionN float64) (SlingEvaluationResult, error) {
	if p.NominalMBLN <= 0 || workingTensionN <= 0 {
		return SlingEvaluationResult{}, errors.New("MBL and tension must be positive")
	}

	factor := 1.0
	if p.IsCableLaid {
		factor *= 0.85 // Cable-laid grommet efficiency derating
	}

	// Choke angle derating per DNV / ASME B30.9
	if p.ChokeAngleDeg < 120.0 {
		if p.ChokeAngleDeg >= 90.0 {
			factor *= 0.80
		} else if p.ChokeAngleDeg >= 60.0 {
			factor *= 0.70
		} else {
			factor *= 0.50
		}
	}

	thermalCutoff := false
	if p.IsSyntheticHMPE && p.OperatingTempC > 65.0 {
		factor *= 0.0 // Thermal cutoff for Dyneema/HMPE (>65°C)
		thermalCutoff = true
	}

	// Design factor 5:1 for rigging slings
	designWLL := (p.NominalMBLN / 5.0) * factor
	pass := !thermalCutoff && workingTensionN <= designWLL

	return SlingEvaluationResult{
		DeratedWLLN:     designWLL,
		EffectiveFactor: factor,
		ThermalDerated:  thermalCutoff,
		PassCheck:       pass,
	}, nil
}

// SheaveReevingParams models sheave and wire rope contact per DIN 15020.
type SheaveReevingParams struct {
	SheaveDiameterM float64 `json:"sheave_diameter_m"`
	RopeDiameterM   float64 `json:"rope_diameter_m"`
	FleetAngleDeg   float64 `json:"fleet_angle_deg"`
	IsGroovedDrum   bool    `json:"is_grooved_drum"`
}

// SheaveEvaluationResult holds contact pressure and fleet angle verifications.
type SheaveEvaluationResult struct {
	ContactPressurePa  float64 `json:"contact_pressure_pa"`
	FleetAngleLimitDeg float64 `json:"fleet_angle_limit_deg"`
	FleetAngleValid    bool    `json:"fleet_angle_valid"`
	PassAllChecks      bool    `json:"pass_all_checks"`
}

// EvaluateSheaveReeving evaluates DIN 15020 contact pressure and fleet angle limits.
func EvaluateSheaveReeving(p SheaveReevingParams, lineTensionN float64) (SheaveEvaluationResult, error) {
	if p.SheaveDiameterM <= 0 || p.RopeDiameterM <= 0 || lineTensionN <= 0 {
		return SheaveEvaluationResult{}, errors.New("invalid sheave parameters")
	}

	// Groove contact pressure: p = 2 * S / (D * d)
	contactP := (2.0 * lineTensionN) / (p.SheaveDiameterM * p.RopeDiameterM)

	fleetLimit := 1.5 // Smooth drum: <= 1.5 degrees
	if p.IsGroovedDrum {
		fleetLimit = 2.0 // Grooved drum: <= 2.0 degrees
	}

	fleetValid := p.FleetAngleDeg <= fleetLimit

	// Allowable contact pressure ~ 8 MPa for cast iron, ~15 MPa for steel sheave
	allowableP := 15.0e6
	pass := fleetValid && contactP <= allowableP

	return SheaveEvaluationResult{
		ContactPressurePa:  contactP,
		FleetAngleLimitDeg: fleetLimit,
		FleetAngleValid:    fleetValid,
		PassAllChecks:      pass,
	}, nil
}

// HookWinklerBachParams models curved beam flexural stresses per DIN 15401 / DIN 15402.
type HookWinklerBachParams struct {
	ThroatRadiusM   float64 `json:"throat_radius_m"` // Inner radius r_i
	DepthM          float64 `json:"depth_m"`         // Radial cross-section depth h
	WidthM          float64 `json:"width_m"`         // Cross-section width b
	YieldStrengthPa float64 `json:"yield_strength_pa"`
}

// HookStressResult holds inner/outer fiber flexural stresses via Winkler-Bach curved beam theory.
type HookStressResult struct {
	InnerFiberStressPa float64 `json:"inner_fiber_stress_pa"`
	OuterFiberStressPa float64 `json:"outer_fiber_stress_pa"`
	MaxStressPa        float64 `json:"max_stress_pa"`
	StressUtilization  float64 `json:"stress_utilization"`
	PassCheck          bool    `json:"pass_check"`
}

// EvaluateHookCrossSection evaluates Winkler-Bach curved beam stress distribution.
func EvaluateHookCrossSection(p HookWinklerBachParams, hookLoadN float64) (HookStressResult, error) {
	if p.ThroatRadiusM <= 0 || p.DepthM <= 0 || p.WidthM <= 0 || hookLoadN <= 0 {
		return HookStressResult{}, errors.New("invalid hook geometry")
	}

	ri := p.ThroatRadiusM
	ro := ri + p.DepthM
	area := p.WidthM * p.DepthM
	rCentroid := ri + (p.DepthM / 2.0)

	// Neutral axis radius for rectangular section: R_n = h / ln(r_o / r_i)
	rNeutral := p.DepthM / math.Log(ro/ri)
	eccentricity := rCentroid - rNeutral

	// Bending moment at critical hook section: M = P * rCentroid
	m := hookLoadN * rCentroid

	// Winkler-Bach extreme fiber stresses:
	// sigma_i = P/A + (M * (rNeutral - ri)) / (A * e * ri)
	// sigma_o = P/A - (M * (ro - rNeutral)) / (A * e * ro)
	directStress := hookLoadN / area
	innerStress := directStress + (m*(rNeutral-ri))/(area*eccentricity*ri)
	outerStress := directStress - (m*(ro-rNeutral))/(area*eccentricity*ro)

	maxStress := math.Max(math.Abs(innerStress), math.Abs(outerStress))
	allowable := 0.60 * p.YieldStrengthPa
	util := maxStress / allowable

	return HookStressResult{
		InnerFiberStressPa: innerStress,
		OuterFiberStressPa: outerStress,
		MaxStressPa:        maxStress,
		StressUtilization:  util,
		PassCheck:          util <= 1.0,
	}, nil
}

// CoupledEnvironmentalCraneConfig holds multi-physics parameters coupled into lift planning.
type CoupledEnvironmentalCraneConfig struct {
	WindSpeedAt10mMs   float64                     `json:"wind_speed_at_10m_m_s"`
	WindTerrain        physics.WindTerrainCategory `json:"wind_terrain"`
	BoomDragCoeff      float64                     `json:"boom_drag_coeff"`
	BoomProjectedAreaM float64                     `json:"boom_projected_area_m2"`
	LoadDragCoeff      float64                     `json:"load_drag_coeff"`
	LoadProjectedAreaM float64                     `json:"load_projected_area_m2"`
	OffshoreRelativeVM float64                     `json:"offshore_relative_v_m_s"`
	RiggingStiffnessNm float64                     `json:"rigging_stiffness_n_m"`
	IsOffshoreLift     bool                        `json:"is_offshore_lift"`
}

// CoupledCraneAnalysisResult holds multi-physics reaction, environmental loads, and formal proof.
type CoupledCraneAnalysisResult struct {
	EffectiveHookLoadTonne float64               `json:"effective_hook_load_tonne"`
	DAF                    float64               `json:"daf"`
	WindDragBoomN          float64               `json:"wind_drag_boom_n"`
	WindDragLoadN          float64               `json:"wind_drag_load_n"`
	OutriggerReactions     OutriggerPressures    `json:"outrigger_reactions"`
	CraneMatPunchShearPa   float64               `json:"crane_mat_punch_shear_pa"`
	LiftOffDetected        bool                  `json:"lift_off_detected"`
	Proof                  symbolic.ProofWitness `json:"proof_witness"`
}

// SolveCoupledEnvironmentalLift integrates aerodynamics, hydrodynamics, and indeterminate ground contact.
func SolveCoupledEnvironmentalLift(
	crane CraneKinematics,
	baseLoadTonne float64,
	envCfg CoupledEnvironmentalCraneConfig,
	craneMatAreaM2 float64,
	craneMatThickM float64,
) (*CoupledCraneAnalysisResult, error) {
	if baseLoadTonne <= 0 {
		return nil, errors.New("base load must be positive")
	}

	hook := crane.ComputeHookPosition()
	boomTipHeight := hook.HookTip.Y

	// 1. Aerodynamic Boundary Layer Wind Profile
	windSpeedAtTip, err := physics.EvaluateWindProfile(physics.WindProfileParams{
		RefVelocityMs: envCfg.WindSpeedAt10mMs,
		RefHeightM:    10.0,
		TargetHeightM: boomTipHeight,
		Terrain:       envCfg.WindTerrain,
	})
	if err != nil {
		windSpeedAtTip = envCfg.WindSpeedAt10mMs
	}

	// 2. Aerodynamic Drag Loads
	boomDrag, _ := physics.EvaluateAeroDrag(0, windSpeedAtTip, envCfg.BoomDragCoeff, envCfg.BoomProjectedAreaM)
	loadDrag, _ := physics.EvaluateAeroDrag(0, windSpeedAtTip, envCfg.LoadDragCoeff, envCfg.LoadProjectedAreaM)

	// 3. Hydrodynamic Offshore DAF if offshore
	effectiveLoadTonne := baseLoadTonne
	daf := 1.0
	if envCfg.IsOffshoreLift && envCfg.RiggingStiffnessNm > 0 {
		dafRes, err := physics.EvaluateOffshoreDAF(physics.OffshoreLiftParams{
			LiftedMassKg:       baseLoadTonne * 1000.0,
			RiggingStiffnessNm: envCfg.RiggingStiffnessNm,
			RelativeVelocityMs: envCfg.OffshoreRelativeVM,
		})
		if err == nil {
			daf = dafRes.DAF
			effectiveLoadTonne = baseLoadTonne * daf
		}
	}

	// 4. Overturning Moments with lateral wind drag
	windOverturningMoment := (boomDrag.DragForceN * (boomTipHeight * 0.5) / 9806.65) +
		(loadDrag.DragForceN * boomTipHeight / 9806.65) // Tonne-meters equivalent

	// 5. Indeterminate Ground Contact with Relaxation
	reactions, liftOff := crane.ComputeRelaxedOutriggerPressures(effectiveLoadTonne, windOverturningMoment)

	// 6. Crane Mat Punch Shear Stress
	matShear := 0.0
	if craneMatAreaM2 > 0 && craneMatThickM > 0 {
		// Punch shear perimeter = 4 * sqrt(A_mat) approx
		matPerimeter := 4.0 * math.Sqrt(craneMatAreaM2)
		shearForceN := reactions.MaxLoad * 1000.0 * physics.StandardGravity
		matShear = shearForceN / (matPerimeter * craneMatThickM)
	}

	// 7. Formal Verification Proof Witness
	// Equilibrium residual ||F_total - sum(P_i)||
	totalExpected := crane.ChassisWeightTonne + crane.CounterweightTonne + effectiveLoadTonne
	totalActual := reactions.FrontLeft + reactions.FrontRight + reactions.RearLeft + reactions.RearRight
	residual := math.Abs(totalExpected - totalActual)
	claim := fmt.Sprintf("CraneOutriggerEquilibrium(Total=%.2ft)", totalExpected)
	proof := symbolic.VerifyClaim(claim, "IterativeRelaxationBalance", residual, 1e-6)

	return &CoupledCraneAnalysisResult{
		EffectiveHookLoadTonne: effectiveLoadTonne,
		DAF:                    daf,
		WindDragBoomN:          boomDrag.DragForceN,
		WindDragLoadN:          loadDrag.DragForceN,
		OutriggerReactions:     reactions,
		CraneMatPunchShearPa:   matShear,
		LiftOffDetected:        liftOff,
		Proof:                  proof,
	}, nil
}
