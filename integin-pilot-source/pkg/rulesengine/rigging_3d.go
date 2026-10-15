package rulesengine

import (
	"fmt"
	"math"
)

const GravityMPS2 = 9.80665

// Point3D Vector Arithmetic Extensions
func (p Point3D) Add(o Point3D) Point3D      { return Point3D{X: p.X + o.X, Y: p.Y + o.Y, Z: p.Z + o.Z} }
func (p Point3D) Sub(o Point3D) Point3D      { return Point3D{X: p.X - o.X, Y: p.Y - o.Y, Z: p.Z - o.Z} }
func (p Point3D) Scale(s float64) Point3D   { return Point3D{X: p.X * s, Y: p.Y * s, Z: p.Z * s} }
func (p Point3D) Dot(o Point3D) float64     { return p.X*o.X + p.Y*o.Y + p.Z*o.Z }
func (p Point3D) Norm() float64             { return math.Sqrt(p.Dot(p)) }
func (p Point3D) Cross(o Point3D) Point3D {
	return Point3D{
		X: p.Y*o.Z - p.Z*o.Y,
		Y: p.Z*o.X - p.X*o.Z,
		Z: p.X*o.Y - p.Y*o.X,
	}
}
func (p Point3D) Unit() Point3D {
	n := p.Norm()
	if n < 1e-12 {
		return Point3D{0, 0, 0}
	}
	return p.Scale(1.0 / n)
}

// PadeyeSpec holds complete geometry and material properties for DNV-ST-N001 structural evaluation.
type PadeyeSpec struct {
	PadeyeID              string   `json:"padeye_id"`
	Position              Point3D  `json:"position_m"`
	InPlaneAzimuthDeg     float64  `json:"in_plane_azimuth_deg"`
	MainPlateThicknessMM  float64  `json:"main_plate_thickness_mm"`
	CheekPlateThicknessMM float64  `json:"cheek_plate_thickness_mm"`
	CheekOuterDiameterMM  float64  `json:"cheek_outer_diameter_mm"`
	CheekFilletLegMM      float64  `json:"cheek_fillet_weld_leg_mm"`
	OuterRadiusMM         float64  `json:"outer_radius_mm"`
	PinHoleDiameterMM     float64  `json:"pin_hole_diameter_mm"`
	BaseWidthMM           float64  `json:"base_width_mm"`
	HoleHeightFromBaseMM  float64  `json:"hole_height_from_base_mm"`
	BaseWeldThroatMM      float64  `json:"base_weld_throat_mm"`
	YieldStrengthMPa      float64  `json:"yield_strength_mpa"`
	UltimateStrengthMPa   float64  `json:"ultimate_strength_mpa"`
}

// ShackleSpec defines shackle dimensions and load capacity.
type ShackleSpec struct {
	ModelName        string  `json:"model_name"`
	WLLTonnes        float64 `json:"wll_tonnes"`
	PinDiameterMM    float64 `json:"pin_diameter_mm"`
	InsideJawWidthMM float64 `json:"inside_jaw_width_mm"`
	SpacerWashersMM  float64 `json:"spacer_washers_mm"`
}

// SlingLegInput defines an individual leg in a 3D rigging bridle.
type SlingLegInput struct {
	LegID                    string      `json:"leg_id"`
	Padeye                   PadeyeSpec  `json:"padeye"`
	Shackle                  ShackleSpec `json:"shackle"`
	SlingWLLTonnes           float64     `json:"sling_wll_tonnes"`
	AxialStiffnessEAkN       float64     `json:"axial_stiffness_ea_kn"`
	LengthToleranceShortenMM float64     `json:"length_tolerance_shorten_mm"`
}

// CoGEnvelopeInput models 3D Center of Gravity uncertainty boundaries per DNV-ST-N001.
type CoGEnvelopeInput struct {
	NominalCoG        Point3D `json:"nominal_cog_m"`
	PlusMinusDeltaXM  float64 `json:"plus_minus_delta_x_m"`
	PlusMinusDeltaYM  float64 `json:"plus_minus_delta_y_m"`
	PlusMinusDeltaZM  float64 `json:"plus_minus_delta_z_m"`
	WeightContingency float64 `json:"weight_contingency_factor"`
}

// Rigging3DInput encapsulates the complete parameters for 3D multi-leg rigging analysis.
type Rigging3DInput struct {
	NetLoadMassKg            float64          `json:"net_load_mass_kg"`
	DynamicAmplification     float64          `json:"dynamic_amplification_factor"`
	SkewLoadFactorOverride   float64          `json:"skew_load_factor_override"`
	MaterialResistanceGammaM float64          `json:"material_resistance_gamma_m"`
	LateralLoadFraction      float64          `json:"lateral_load_fraction"`
	HookElevationZM          float64          `json:"hook_elevation_z_m"`
	CoGEnvelope              CoGEnvelopeInput `json:"cog_envelope"`
	Legs                     []SlingLegInput  `json:"legs"`
}

// PadeyeCheckResult details all stress ratios and code-compliance metrics for a padeye.
type PadeyeCheckResult struct {
	PadeyeID                  string   `json:"padeye_id"`
	PassedAllChecks           bool     `json:"passed_all_checks"`
	GoverningUtilizationRatio float64  `json:"governing_utilization_ratio"`
	GoverningFailureMode      string   `json:"governing_failure_mode"`
	PinToHoleClearanceRatio   float64  `json:"pin_to_hole_clearance_ratio"`
	JawClearanceMM            float64  `json:"jaw_clearance_mm"`
	CheekWeldUtilization      float64  `json:"cheek_weld_utilization"`
	CheekPlatesCounted        bool     `json:"cheek_plates_counted"`
	EffectiveBearingThickMM   float64  `json:"effective_bearing_thickness_mm"`
	BearingStressMPa          float64  `json:"bearing_stress_mpa"`
	BearingUtilization        float64  `json:"bearing_utilization"`
	TearOutShearStressMPa     float64  `json:"tear_out_shear_stress_mpa"`
	TearOutUtilization        float64  `json:"tear_out_utilization"`
	NetSectionTensileMPa      float64  `json:"net_section_tensile_mpa"`
	NetSectionUtilization     float64  `json:"net_section_utilization"`
	OutOfPlaneMomentNm        float64  `json:"out_of_plane_moment_nm"`
	InPlaneMomentNm           float64  `json:"in_plane_moment_nm"`
	BaseVonMisesStressMPa     float64  `json:"base_von_mises_stress_mpa"`
	BaseVonMisesUtilization   float64  `json:"base_von_mises_utilization"`
	BaseWeldShearMPa          float64  `json:"base_weld_shear_mpa"`
	BaseWeldUtilization       float64  `json:"base_weld_utilization"`
	WarningsAndViolations     []string `json:"warnings_and_violations"`
}

// LegEvaluationResult captures tensions and utilization for a specific sling leg.
type LegEvaluationResult struct {
	LegID                      string            `json:"leg_id"`
	NominalSlingLengthM        float64           `json:"nominal_sling_length_m"`
	MinAngleToHorizontalDeg    float64           `json:"min_angle_to_horizontal_deg"`
	MaxOutOfPlaneAngleBetaDeg  float64           `json:"max_out_of_plane_angle_beta_deg"`
	DeterminateOr3LegTensionKN float64           `json:"determinate_or_3leg_envelope_tension_kn"`
	ElasticToleranceTensionKN  float64           `json:"elastic_tolerance_tension_kn"`
	DNVSkewFactoredTensionKN   float64           `json:"dnv_skew_factored_tension_kn"`
	GoverningDesignTensionKN   float64           `json:"governing_design_tension_kn"`
	GoverningDesignTensionTon  float64           `json:"governing_design_tension_tonnes"`
	GoverningCoGCase           string            `json:"governing_cog_case"`
	GoverningMethod            string            `json:"governing_method"`
	SlingUtilizationRatio      float64           `json:"sling_utilization_ratio"`
	ShackleUtilizationRatio    float64           `json:"shackle_utilization_ratio"`
	PadeyeEvaluation           PadeyeCheckResult `json:"padeye_evaluation"`
}

// Rigging3DEvaluationReport is the comprehensive multi-leg rigging and padeye verification report.
type Rigging3DEvaluationReport struct {
	IsSafeAndCompliant     bool                  `json:"is_safe_and_compliant"`
	FactoredTotalWeightKN  float64               `json:"factored_total_weight_kn"`
	FactoredTotalWeightTon float64               `json:"factored_total_weight_tonnes"`
	MaxTiltAngleDeg        float64               `json:"max_equilibrium_tilt_angle_deg"`
	LegResults             []LegEvaluationResult `json:"leg_results"`
	GlobalViolations       []string              `json:"global_violations"`
}

// Solve3DRiggingAndPadeyes evaluates 3D sling tensions across CoG shifts and validates padeyes.
func Solve3DRiggingAndPadeyes(in Rigging3DInput) (*Rigging3DEvaluationReport, error) {
	numLegs := len(in.Legs)
	if numLegs < 1 {
		return nil, fmt.Errorf("rigging: at least 1 sling leg is required")
	}

	if in.DynamicAmplification < 1.0 {
		in.DynamicAmplification = 1.15
	}
	if in.CoGEnvelope.WeightContingency < 1.0 {
		in.CoGEnvelope.WeightContingency = 1.0
	}
	if in.MaterialResistanceGammaM <= 0 {
		in.MaterialResistanceGammaM = 1.15
	}
	if in.LateralLoadFraction <= 0 {
		in.LateralLoadFraction = 0.05
	}
	if in.SkewLoadFactorOverride <= 0 {
		if numLegs >= 4 {
			in.SkewLoadFactorOverride = 1.25
		} else {
			in.SkewLoadFactorOverride = 1.00
		}
	}

	factoredMassKg := in.NetLoadMassKg * in.CoGEnvelope.WeightContingency * in.DynamicAmplification
	totalWkN := (factoredMassKg * GravityMPS2) / 1000.0

	report := &Rigging3DEvaluationReport{
		IsSafeAndCompliant:     true,
		FactoredTotalWeightKN:  math.Round(totalWkN*1000) / 1000,
		FactoredTotalWeightTon: math.Round((factoredMassKg/1000.0)*1000) / 1000,
		LegResults:             make([]LegEvaluationResult, numLegs),
		GlobalViolations:       make([]string, 0),
	}

	for i, leg := range in.Legs {
		nomHook := Point3D{X: in.CoGEnvelope.NominalCoG.X, Y: in.CoGEnvelope.NominalCoG.Y, Z: in.HookElevationZM}
		nomVec := nomHook.Sub(leg.Padeye.Position)
		report.LegResults[i] = LegEvaluationResult{
			LegID:                   leg.LegID,
			NominalSlingLengthM:     math.Round(nomVec.Norm()*1000) / 1000,
			MinAngleToHorizontalDeg: 90.0,
		}
	}

	cogCases := buildCoGEnvelopeCases(in.CoGEnvelope)

	for _, cogCase := range cogCases {
		hookApex := Point3D{X: cogCase.CoG.X, Y: cogCase.CoG.Y, Z: in.HookElevationZM}
		horizShift := math.Hypot(cogCase.CoG.X-in.CoGEnvelope.NominalCoG.X, cogCase.CoG.Y-in.CoGEnvelope.NominalCoG.Y)
		leverZ := math.Max(0.5, in.HookElevationZM-cogCase.CoG.Z)
		tiltDeg := math.Atan2(horizShift, leverZ) * (180.0 / math.Pi)
		if tiltDeg > report.MaxTiltAngleDeg {
			report.MaxTiltAngleDeg = math.Round(tiltDeg*100) / 100
		}

		unitVecs := make([]Point3D, numLegs)
		lengthsM := make([]float64, numLegs)
		for i, leg := range in.Legs {
			v := hookApex.Sub(leg.Padeye.Position)
			if v.Z <= 0 {
				return nil, fmt.Errorf("rigging: hook elevation Z (%.2fm) must be above padeye %s Z (%.2fm)",
					in.HookElevationZM, leg.Padeye.PadeyeID, leg.Padeye.Position.Z)
			}
			lengthsM[i] = v.Norm()
			unitVecs[i] = v.Unit()

			alphaDeg := math.Asin(math.Min(1.0, math.Max(0.0, unitVecs[i].Z))) * (180.0 / math.Pi)
			if alphaDeg < report.LegResults[i].MinAngleToHorizontalDeg {
				report.LegResults[i].MinAngleToHorizontalDeg = math.Round(alphaDeg*100) / 100
			}
		}

		var detTriadTensions, elasticTolTensions, dnvSkewTensions []float64
		if numLegs == 1 {
			detTriadTensions = []float64{totalWkN / unitVecs[0].Z}
			elasticTolTensions = detTriadTensions
			dnvSkewTensions = detTriadTensions
		} else if numLegs == 2 {
			detTriadTensions = solve2LegConcurrentStatics(unitVecs, totalWkN)
			elasticTolTensions = detTriadTensions
			dnvSkewTensions = detTriadTensions
		} else if numLegs == 3 {
			t3, ok := solve3x3ConcurrentStatics(unitVecs[0], unitVecs[1], unitVecs[2], totalWkN)
			if !ok {
				report.GlobalViolations = append(report.GlobalViolations,
					fmt.Sprintf("CoG case [%s] falls outside 3-leg support triangle.", cogCase.Name))
				continue
			}
			detTriadTensions = t3[:]
			elasticTolTensions = detTriadTensions
			dnvSkewTensions = detTriadTensions
		} else {
			detTriadTensions = solveWorstCase3ActiveLegTriads(unitVecs, totalWkN)
			elasticTolTensions = solveElasticStiffnessWithTolerance(in.Legs, unitVecs, lengthsM, totalWkN)
			dnvSkewTensions = make([]float64, numLegs)
			for i := 0; i < numLegs; i++ {
				dnvSkewTensions[i] = elasticTolTensions[i] * in.SkewLoadFactorOverride
			}
		}

		for i, leg := range in.Legs {
			if detTriadTensions[i] > report.LegResults[i].DeterminateOr3LegTensionKN {
				report.LegResults[i].DeterminateOr3LegTensionKN = math.Round(detTriadTensions[i]*100) / 100
			}
			if elasticTolTensions[i] > report.LegResults[i].ElasticToleranceTensionKN {
				report.LegResults[i].ElasticToleranceTensionKN = math.Round(elasticTolTensions[i]*100) / 100
			}
			if dnvSkewTensions[i] > report.LegResults[i].DNVSkewFactoredTensionKN {
				report.LegResults[i].DNVSkewFactoredTensionKN = math.Round(dnvSkewTensions[i]*100) / 100
			}

			candidates := []struct {
				val    float64
				method string
			}{
				{detTriadTensions[i], "3_active_leg_determinate_envelope"},
				{elasticTolTensions[i], "elastic_stiffness_tolerance"},
				{dnvSkewTensions[i], "dnv_skew_load_factor"},
			}
			for _, c := range candidates {
				if c.val > report.LegResults[i].GoverningDesignTensionKN {
					report.LegResults[i].GoverningDesignTensionKN = math.Round(c.val*100) / 100
					report.LegResults[i].GoverningDesignTensionTon = math.Round((c.val/GravityMPS2)*100) / 100
					report.LegResults[i].GoverningCoGCase = cogCase.Name
					report.LegResults[i].GoverningMethod = c.method
				}
			}

			betaDeg := computePadeyeOutOfPlaneAngleDeg(leg.Padeye, unitVecs[i])
			if betaDeg > report.LegResults[i].MaxOutOfPlaneAngleBetaDeg {
				report.LegResults[i].MaxOutOfPlaneAngleBetaDeg = math.Round(betaDeg*100) / 100
			}
		}
	}

	for i, leg := range in.Legs {
		lr := &report.LegResults[i]
		if lr.MinAngleToHorizontalDeg < CriticalSlingAngleDeg {
			report.IsSafeAndCompliant = false
			report.GlobalViolations = append(report.GlobalViolations,
				fmt.Sprintf("Leg %s: Sling angle %.1f deg is below critical limit %.1f deg.",
					leg.LegID, lr.MinAngleToHorizontalDeg, CriticalSlingAngleDeg))
		}

		if leg.SlingWLLTonnes > 0 {
			lr.SlingUtilizationRatio = math.Round((lr.GoverningDesignTensionTon/leg.SlingWLLTonnes)*1000) / 1000
			if lr.SlingUtilizationRatio > 1.0 {
				report.IsSafeAndCompliant = false
				report.GlobalViolations = append(report.GlobalViolations,
					fmt.Sprintf("Leg %s Sling Overload: Demand %.2ft > WLL %.2ft",
						leg.LegID, lr.GoverningDesignTensionTon, leg.SlingWLLTonnes))
			}
		}
		if leg.Shackle.WLLTonnes > 0 {
			lr.ShackleUtilizationRatio = math.Round((lr.GoverningDesignTensionTon/leg.Shackle.WLLTonnes)*1000) / 1000
			if lr.ShackleUtilizationRatio > 1.0 {
				report.IsSafeAndCompliant = false
				report.GlobalViolations = append(report.GlobalViolations,
					fmt.Sprintf("Leg %s Shackle Overload: Demand %.2ft > WLL %.2ft",
						leg.LegID, lr.GoverningDesignTensionTon, leg.Shackle.WLLTonnes))
			}
		}

		padeyeRes := evaluateDNVPadeye(
			leg.Padeye,
			leg.Shackle,
			lr.GoverningDesignTensionKN,
			lr.MinAngleToHorizontalDeg,
			lr.MaxOutOfPlaneAngleBetaDeg,
			in.LateralLoadFraction,
			in.MaterialResistanceGammaM,
		)
		lr.PadeyeEvaluation = padeyeRes
		if !padeyeRes.PassedAllChecks {
			report.IsSafeAndCompliant = false
		}
	}

	return report, nil
}

type cogState struct {
	Name string
	CoG  Point3D
}

func buildCoGEnvelopeCases(env CoGEnvelopeInput) []cogState {
	cases := []cogState{{Name: "Nominal_CoG", CoG: env.NominalCoG}}
	if env.PlusMinusDeltaXM == 0 && env.PlusMinusDeltaYM == 0 && env.PlusMinusDeltaZM == 0 {
		return cases
	}
	signs := []float64{-1.0, 1.0}
	idx := 1
	for _, sx := range signs {
		for _, sy := range signs {
			cases = append(cases, cogState{
				Name: fmt.Sprintf("Corner_%d", idx),
				CoG: Point3D{
					X: env.NominalCoG.X + sx*env.PlusMinusDeltaXM,
					Y: env.NominalCoG.Y + sy*env.PlusMinusDeltaYM,
					Z: env.NominalCoG.Z + env.PlusMinusDeltaZM,
				},
			})
			idx++
		}
	}
	return cases
}

func solve3x3ConcurrentStatics(e1, e2, e3 Point3D, WkN float64) ([3]float64, bool) {
	det := e1.Dot(e2.Cross(e3))
	if math.Abs(det) < 1e-9 {
		return [3]float64{}, false
	}
	t1 := (WkN * (e2.X*e3.Y - e2.Y*e3.X)) / det
	t2 := (WkN * (e3.X*e1.Y - e3.Y*e1.X)) / det
	t3 := (WkN * (e1.X*e2.Y - e1.Y*e2.X)) / det

	if t1 < -1e-4 || t2 < -1e-4 || t3 < -1e-4 {
		return [3]float64{}, false
	}
	return [3]float64{math.Max(0, t1), math.Max(0, t2), math.Max(0, t3)}, true
}

func solve2LegConcurrentStatics(unitVecs []Point3D, WkN float64) []float64 {
	e1h := math.Hypot(unitVecs[0].X, unitVecs[0].Y)
	e2h := math.Hypot(unitVecs[1].X, unitVecs[1].Y)
	denom := e1h*unitVecs[1].Z + e2h*unitVecs[0].Z
	if denom < 1e-9 {
		return []float64{WkN / 2.0, WkN / 2.0}
	}
	return []float64{(WkN * e2h) / denom, (WkN * e1h) / denom}
}

func solveWorstCase3ActiveLegTriads(unitVecs []Point3D, WkN float64) []float64 {
	n := len(unitVecs)
	maxT := make([]float64, n)
	validTriadFound := false

	for i := 0; i < n-2; i++ {
		for j := i + 1; j < n-1; j++ {
			for k := j + 1; k < n; k++ {
				t3, ok := solve3x3ConcurrentStatics(unitVecs[i], unitVecs[j], unitVecs[k], WkN)
				if ok {
					validTriadFound = true
					if t3[0] > maxT[i] {
						maxT[i] = t3[0]
					}
					if t3[1] > maxT[j] {
						maxT[j] = t3[1]
					}
					if t3[2] > maxT[k] {
						maxT[k] = t3[2]
					}
				}
			}
		}
	}
	if !validTriadFound {
		for i := 0; i < n; i++ {
			maxT[i] = WkN / (2.0 * math.Max(0.2, unitVecs[i].Z))
		}
	}
	return maxT
}

func solveElasticStiffnessWithTolerance(legs []SlingLegInput, unitVecs []Point3D, lengthsM []float64, WkN float64) []float64 {
	n := len(legs)
	stiffVertical := make([]float64, n)
	sumStiffZ2 := 0.0

	for i, leg := range legs {
		ea := leg.AxialStiffnessEAkN
		if ea <= 0 {
			ea = 65000.0
		}
		kAxial := ea / math.Max(0.5, lengthsM[i])
		stiffVertical[i] = kAxial * unitVecs[i].Z
		sumStiffZ2 += stiffVertical[i] * unitVecs[i].Z
	}

	tensions := make([]float64, n)
	if sumStiffZ2 < 1e-9 {
		return tensions
	}

	deltaZ := WkN / sumStiffZ2
	for i, leg := range legs {
		ea := leg.AxialStiffnessEAkN
		if ea <= 0 {
			ea = 65000.0
		}
		kAxial := ea / math.Max(0.5, lengthsM[i])
		shortenM := leg.LengthToleranceShortenMM / 1000.0
		tensions[i] = kAxial * (deltaZ*unitVecs[i].Z + 0.5*shortenM)
	}
	return tensions
}

func computePadeyeOutOfPlaneAngleDeg(p PadeyeSpec, slingUnitVec Point3D) float64 {
	azRad := p.InPlaneAzimuthDeg * (math.Pi / 180.0)
	normalOOP := Point3D{X: -math.Sin(azRad), Y: math.Cos(azRad), Z: 0}
	sinBeta := math.Abs(slingUnitVec.Dot(normalOOP))
	return math.Asin(math.Min(1.0, sinBeta)) * (180.0 / math.Pi)
}

func evaluateDNVPadeye(
	p PadeyeSpec,
	sh ShackleSpec,
	designTensionKN float64,
	slingAngleAlphaDeg float64,
	outOfPlaneBetaDeg float64,
	lateralLoadFraction float64,
	gammaM float64,
) PadeyeCheckResult {
	res := PadeyeCheckResult{
		PadeyeID:              p.PadeyeID,
		PassedAllChecks:       true,
		WarningsAndViolations: make([]string, 0),
	}

	pNewtons := designTensionKN * 1000.0
	allowableSigma := p.YieldStrengthMPa / gammaM
	allowableTau := p.YieldStrengthMPa / (math.Sqrt(3.0) * gammaM)

	if sh.PinDiameterMM > 0 && p.PinHoleDiameterMM > 0 {
		if sh.PinDiameterMM >= p.PinHoleDiameterMM {
			res.PassedAllChecks = false
			res.WarningsAndViolations = append(res.WarningsAndViolations,
				fmt.Sprintf("Shackle pin (%.1fmm) >= hole (%.1fmm).", sh.PinDiameterMM, p.PinHoleDiameterMM))
		}
		res.PinToHoleClearanceRatio = math.Round((p.PinHoleDiameterMM/sh.PinDiameterMM)*1000) / 1000
		maxAllowedHole := math.Max(sh.PinDiameterMM+3.0, sh.PinDiameterMM*1.06)
		if p.PinHoleDiameterMM > maxAllowedHole {
			res.WarningsAndViolations = append(res.WarningsAndViolations,
				fmt.Sprintf("Hole %.1fmm exceeds DNV max clearance limit %.1fmm.", p.PinHoleDiameterMM, maxAllowedHole))
		}
	}

	totalThickAtHole := p.MainPlateThicknessMM + 2.0*p.CheekPlateThicknessMM
	res.EffectiveBearingThickMM = p.MainPlateThicknessMM

	if p.CheekPlateThicknessMM > 0 {
		if p.CheekPlateThicknessMM > 0.5*p.MainPlateThicknessMM {
			res.WarningsAndViolations = append(res.WarningsAndViolations,
				fmt.Sprintf("Cheek plate thickness %.1fmm > 0.5*main plate %.1fmm.", p.CheekPlateThicknessMM, 0.5*p.MainPlateThicknessMM))
		}

		forceBothCheeksN := pNewtons * ((2.0 * p.CheekPlateThicknessMM) / totalThickAtHole)
		cheekWeldThroatMM := 0.707 * p.CheekFilletLegMM
		cheekWeldLengthEachMM := math.Pi * p.CheekOuterDiameterMM
		totalCheekWeldAreaMM2 := 2.0 * cheekWeldThroatMM * cheekWeldLengthEachMM

		if totalCheekWeldAreaMM2 > 0 {
			cheekWeldShearMPa := forceBothCheeksN / totalCheekWeldAreaMM2
			allowableWeldShearMPa := (p.UltimateStrengthMPa / math.Sqrt(3.0)) / (gammaM * 1.25)
			res.CheekWeldUtilization = math.Round((cheekWeldShearMPa/allowableWeldShearMPa)*1000) / 1000

			if res.CheekWeldUtilization <= 1.0 {
				res.CheekPlatesCounted = true
				res.EffectiveBearingThickMM = totalThickAtHole
			} else {
				res.PassedAllChecks = false
				res.WarningsAndViolations = append(res.WarningsAndViolations,
					fmt.Sprintf("Cheek fillet weld overloaded (Util %.2f); cheek plates excluded.", res.CheekWeldUtilization))
			}
		}
	}

	effectiveFittedWidth := totalThickAtHole + sh.SpacerWashersMM
	lateralEccentricityMM := 0.0
	if sh.InsideJawWidthMM > 0 {
		res.JawClearanceMM = math.Round((sh.InsideJawWidthMM-effectiveFittedWidth)*100) / 100
		if effectiveFittedWidth < 0.75*sh.InsideJawWidthMM {
			res.WarningsAndViolations = append(res.WarningsAndViolations,
				fmt.Sprintf("Padeye width %.1fmm < 75%% jaw width %.1fmm.", effectiveFittedWidth, sh.InsideJawWidthMM))
		}
		if res.JawClearanceMM > 0 {
			lateralEccentricityMM = 0.5 * res.JawClearanceMM
		}
	}

	if sh.PinDiameterMM > 0 && res.EffectiveBearingThickMM > 0 {
		res.BearingStressMPa = math.Round((pNewtons/(sh.PinDiameterMM*res.EffectiveBearingThickMM))*100) / 100
		allowableBearing := 0.90 * allowableSigma
		res.BearingUtilization = math.Round((res.BearingStressMPa/allowableBearing)*1000) / 1000
	}

	edgeLigamentMM := p.OuterRadiusMM - 0.5*p.PinHoleDiameterMM
	if edgeLigamentMM > 0 && res.EffectiveBearingThickMM > 0 {
		areaTearOutMM2 := 2.0 * edgeLigamentMM * res.EffectiveBearingThickMM
		res.TearOutShearStressMPa = math.Round((pNewtons/areaTearOutMM2)*100) / 100
		res.TearOutUtilization = math.Round((res.TearOutShearStressMPa/allowableTau)*1000) / 1000
	}

	if edgeLigamentMM > 0 && p.MainPlateThicknessMM > 0 {
		areaNetSectionMM2 := 2.0 * edgeLigamentMM * p.MainPlateThicknessMM
		res.NetSectionTensileMPa = math.Round(((1.25*pNewtons)/areaNetSectionMM2)*100) / 100
		res.NetSectionUtilization = math.Round((res.NetSectionTensileMPa/allowableSigma)*1000) / 1000
	}

	alphaRad := slingAngleAlphaDeg * (math.Pi / 180.0)
	betaRad := outOfPlaneBetaDeg * (math.Pi / 180.0)

	fVertical := pNewtons * math.Sin(alphaRad) * math.Cos(betaRad)
	fInplaneHoriz := pNewtons * math.Cos(alphaRad) * math.Cos(betaRad)
	fOopDesign := pNewtons*math.Sin(betaRad) + lateralLoadFraction*pNewtons

	mInplaneNmm := fInplaneHoriz * p.HoleHeightFromBaseMM
	mOopNmm := fOopDesign*p.HoleHeightFromBaseMM + pNewtons*lateralEccentricityMM

	res.InPlaneMomentNm = math.Round((mInplaneNmm/1000.0)*100) / 100
	res.OutOfPlaneMomentNm = math.Round((mOopNmm/1000.0)*100) / 100

	if p.BaseWidthMM > 0 && p.MainPlateThicknessMM > 0 {
		aBase := p.BaseWidthMM * p.MainPlateThicknessMM
		wIp := (p.MainPlateThicknessMM * p.BaseWidthMM * p.BaseWidthMM) / 6.0
		wOop := (p.BaseWidthMM * p.MainPlateThicknessMM * p.MainPlateThicknessMM) / 6.0

		sigmaAxial := fVertical / aBase
		sigmaBendIP := mInplaneNmm / wIp
		sigmaBendOOP := mOopNmm / wOop
		sigmaCombined := sigmaAxial + sigmaBendIP + sigmaBendOOP

		tauBase := math.Hypot(fInplaneHoriz, fOopDesign) / aBase
		res.BaseVonMisesStressMPa = math.Round(math.Sqrt(sigmaCombined*sigmaCombined+3.0*tauBase*tauBase)*100) / 100
		res.BaseVonMisesUtilization = math.Round((res.BaseVonMisesStressMPa/allowableSigma)*1000) / 1000
	}

	if p.BaseWeldThroatMM > 0 && p.BaseWidthMM > 0 {
		weldPerimMM := 2.0 * (p.BaseWidthMM + p.MainPlateThicknessMM)
		aWeld := weldPerimMM * p.BaseWeldThroatMM
		wWeldIP := (p.BaseWeldThroatMM * p.BaseWidthMM * p.BaseWidthMM) / 3.0
		wWeldOOP := (p.BaseWidthMM * p.MainPlateThicknessMM * p.BaseWeldThroatMM)

		sigmaWeld := (fVertical / aWeld) + (mInplaneNmm / wWeldIP) + (mOopNmm / wWeldOOP)
		tauWeld := math.Hypot(fInplaneHoriz, fOopDesign) / aWeld
		res.BaseWeldShearMPa = math.Round(math.Sqrt(sigmaWeld*sigmaWeld+3.0*tauWeld*tauWeld)*100) / 100
		allowableWeldMPa := (p.UltimateStrengthMPa / math.Sqrt(3.0)) / (gammaM * 1.25)
		res.BaseWeldUtilization = math.Round((res.BaseWeldShearMPa/allowableWeldMPa)*1000) / 1000
	}

	modes := []struct {
		name string
		util float64
	}{
		{"pin_bearing_stress", res.BearingUtilization},
		{"pin_tear_out_shear", res.TearOutUtilization},
		{"hole_net_section_tension", res.NetSectionUtilization},
		{"base_combined_von_mises", res.BaseVonMisesUtilization},
		{"base_weld_combined_stress", res.BaseWeldUtilization},
		{"cheek_plate_fillet_weld_shear", res.CheekWeldUtilization},
	}

	for _, m := range modes {
		if m.util > res.GoverningUtilizationRatio {
			res.GoverningUtilizationRatio = m.util
			res.GoverningFailureMode = m.name
		}
		if m.util > 1.0 {
			res.PassedAllChecks = false
			res.WarningsAndViolations = append(res.WarningsAndViolations,
				fmt.Sprintf("Padeye %s failed [%s]: Util = %.1f%% (>100%%).", p.PadeyeID, m.name, m.util*100))
		}
	}

	return res
}
