package rulesengine

import (
	"fmt"
	"math"
)

// crane_3d.go — 3D crane capacity, boom kinematics, and utilization solver.
//
// Governing equations:
//   Geometric planimetric radius:
//     R_geom = sqrt((X_L - X_c)^2 + (Y_L - Y_c)^2)
//   Effective lookup radius (adds boom deflection allowance):
//     R_eff = R_geom + ΔR_defl
//   Boom angle from foot pin:
//     θ = arccos((R_eff - r_pin) / L)
//   Boom tip elevation:
//     Z_tip = Z_c + z_pin + L·sin(θ)
//   Anti-two-block check:
//     Z_tip >= Z_load + H_rigging + H_block + C_min_drift
//   Gross hook load with DAF:
//     W_gross = (W_net + W_spreader + W_slings) × DAF + W_block + W_ball + W_rope
//   Capacity interpolation follows INTEGIN step-down-strict policy by default
//   (same philosophy as loadchart.LookupConservativeCapacity): always use the
//   next larger radius / lower capacity; NEVER interpolate across a Structural↔Tipping
//   failure mode boundary.
//
// Standard references: ASME B30.5, DNV-ST-N001, EN 13000.

// FailureMode tags how a capacity limit is governed.
type FailureMode string

const (
	FailureModeStructural   FailureMode = "structural_strength"
	FailureModeTipping      FailureMode = "tipping_stability"
	FailureModeReeving      FailureMode = "hoist_reeving_line_pull"
	FailureModeUnclassified FailureMode = "unclassified"
)

// InterpolationPolicy governs how radii between table entries are handled.
type InterpolationPolicy string

const (
	// InterpolationStepDownStrict: always use the next larger radius / lower capacity.
	// This is the default and the only policy permitted across Structural↔Tipping boundaries.
	InterpolationStepDownStrict InterpolationPolicy = "step_down_strict"
	// InterpolationLinearRadiusOnly: linear interpolation between radii is permitted ONLY
	// when both bounding cells share the same FailureMode.
	InterpolationLinearRadiusOnly InterpolationPolicy = "linear_radius_only"
)

// RatedCell holds one discrete entry in a load chart matrix.
type RatedCell struct {
	RadiusM            float64     `json:"radius_m"`
	BoomLengthM        float64     `json:"boom_length_m"`
	TelePinningCode    string      `json:"tele_pinning_code"` // e.g. "STANDARD" or "0-50-50-50"
	CapacityKg         float64     `json:"capacity_kg"`
	LoadedBoomAngleDeg *float64    `json:"loaded_boom_angle_deg,omitempty"`
	LimitGovernedBy    FailureMode `json:"limit_governed_by"`
	MinReevingParts    *int        `json:"min_reeving_parts,omitempty"`
}

// CraneConfig holds crane-level metadata for a single load chart configuration.
type CraneConfig struct {
	CraneModel          string              `json:"crane_model"`
	CapacityType        string              `json:"capacity_type"` // "gross" or "net"
	InterpolationPolicy InterpolationPolicy `json:"interpolation_policy"`
	CounterweightKg     float64             `json:"counterweight_kg"`
	OutriggerLengthM    float64             `json:"outrigger_length_m"`
	OutriggerWidthM     float64             `json:"outrigger_width_m"`
	SlewRangeDeg        float64             `json:"slew_range_deg"`
	MaxWindSpeedMps     float64             `json:"max_wind_speed_mps"`
	// Cells MUST be sorted: primary boom_length_m ASC, secondary radius_m ASC.
	Cells []RatedCell `json:"cells"`
}

// LiftPlan3DInput is the complete input to Evaluate3DLift.
type LiftPlan3DInput struct {
	Config CraneConfig `json:"config"`

	BoomLengthM          float64 `json:"boom_length_m"`
	TelePinningCode      string  `json:"tele_pinning_code"`
	CraneSlewCenter      Point3D `json:"crane_slew_center"`
	BoomFootPinOffsetR   float64 `json:"boom_foot_pin_offset_r_m"` // horizontal offset from slew axis to boom foot pin
	BoomFootPinOffsetZ   float64 `json:"boom_foot_pin_offset_z_m"` // height of boom foot pin above Z_c
	TargetLoadHookPos    Point3D `json:"target_load_hook_pos"`
	BoomDeflectionMargin float64 `json:"boom_deflection_margin_m"` // added to R_geom (e.g. 0.5–1.5 m)

	// Vertical envelope heights for anti-two-block / clearance check.
	RiggingHeightM       float64 `json:"rigging_height_m"`
	HookBlockHeightM     float64 `json:"hook_block_height_m"`
	MinTwoBlockClearance float64 `json:"min_two_block_clearance_m"`

	// Masses (kg).
	NetLoadMassKg         float64 `json:"net_load_mass_kg"`
	SpreaderBeamMassKg    float64 `json:"spreader_beam_mass_kg"`
	SlingsAndShacklesKg   float64 `json:"slings_and_shackles_kg"`
	MainHookBlockMassKg   float64 `json:"main_hook_block_mass_kg"`
	AuxBallMassKg         float64 `json:"aux_ball_mass_kg"`
	HoistRopeMassPerMeter float64 `json:"hoist_rope_mass_per_m_kg"` // per single part, per metre
	ReevingPartsOfLine    int     `json:"reeving_parts_of_line"`
	SingleLinePullLimitKg float64 `json:"single_line_pull_limit_kg"`
	DynamicAmplification  float64 `json:"dynamic_amplification_factor"`  // >= 1.0; 1.0 onshore, 1.15–1.30 offshore
	MaxAllowedUtilization float64 `json:"max_allowed_utilization_ratio"` // 0.85 ASME 85%, 1.0 ASME 100%
}

// LiftEvalResult is the complete deterministic output from Evaluate3DLift.
type LiftEvalResult struct {
	IsSafeToLift           bool    `json:"is_safe_to_lift"`
	GeometricRadiusM       float64 `json:"geometric_radius_m"`
	EffectiveLoadedRadiusM float64 `json:"effective_loaded_radius_m"`
	BoomAngleDeg           float64 `json:"boom_angle_deg"`
	BoomTipElevationZM     float64 `json:"boom_tip_elevation_z_m"`
	AvailableHeadroomM     float64 `json:"available_headroom_m"`

	ChartGrossCapacityKg    float64 `json:"chart_gross_capacity_kg"`
	ReevingCapacityKg       float64 `json:"reeving_capacity_kg"`
	GoverningCapacityKg     float64 `json:"governing_capacity_kg"`
	TotalRiggingDeductionKg float64 `json:"total_rigging_deduction_kg"`
	NetAvailableCapacityKg  float64 `json:"net_available_capacity_kg"`
	TotalGrossHookLoadKg    float64 `json:"total_gross_hook_load_kg"`
	UtilizationRatio        float64 `json:"utilization_ratio"`
	UtilizationPercent      float64 `json:"utilization_percent"`

	GoverningLimitType   FailureMode `json:"governing_limit_type"`
	InterpolationApplied string      `json:"interpolation_applied"`
	AuditTrace           []string    `json:"audit_trace"`
	SafetyViolations     []string    `json:"safety_violations"`
}

func crane3DRejectNaN(name string, vals ...float64) error {
	for _, v := range vals {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("crane_3d: NaN/Inf in %s rejected", name)
		}
	}
	return nil
}

// lookupBoundingCells finds the lower and upper bounding cells in the chart
// for the given boom length and effective radius, filtered by tele pinning code.
// Both cells must share the same boom_length_m (step-down on boom is the caller's job).
func lookupBoundingCells(cells []RatedCell, boomM float64, tele string, radiusM float64) (lower, upper *RatedCell, err error) {
	const eps = 1e-6
	// Filter to the target boom+tele subset.
	var subset []RatedCell
	for i := range cells {
		c := &cells[i]
		if math.Abs(c.BoomLengthM-boomM) < eps && c.TelePinningCode == tele {
			subset = append(subset, *c)
		}
	}
	if len(subset) == 0 {
		return nil, nil, fmt.Errorf("crane_3d: no cells for boom %.2fm tele %q", boomM, tele)
	}
	// Find bounding pair (cells assumed sorted radius_m ASC).
	var lo, hi *RatedCell
	for i := range subset {
		cell := subset[i]
		if cell.RadiusM <= radiusM {
			if lo == nil || cell.RadiusM > lo.RadiusM {
				c := cell
				lo = &c
			}
		}
		if cell.RadiusM >= radiusM {
			if hi == nil || cell.RadiusM < hi.RadiusM {
				c := cell
				hi = &c
			}
		}
	}
	return lo, hi, nil
}

// nearestLargerBoomLength returns the smallest boom length in the cells >= reqBoomM
// for the given tele pinning code. Implements INTEGIN step-down-strict on boom.
func nearestLargerBoomLength(cells []RatedCell, reqBoomM float64, tele string) (float64, bool) {
	const eps = 1e-6
	best := math.MaxFloat64
	found := false
	for _, c := range cells {
		if c.TelePinningCode == tele && c.BoomLengthM >= reqBoomM-eps {
			if c.BoomLengthM < best {
				best = c.BoomLengthM
				found = true
			}
		}
	}
	return best, found
}

// Evaluate3DLift executes a deterministic 3D capacity, clearance, and rigging
// utilization check against an in-memory rated cell matrix.
// All DB lookups (Evaluate3DLift with pgx) follow the same algorithm and must
// use pgtx.BeginScope before querying — see crane_3d_store.go (repository layer).
func Evaluate3DLift(in LiftPlan3DInput) (*LiftEvalResult, error) {
	res := &LiftEvalResult{
		AuditTrace:       make([]string, 0, 16),
		SafetyViolations: make([]string, 0),
	}

	// Guards.
	if err := crane3DRejectNaN("positions",
		in.CraneSlewCenter.X, in.CraneSlewCenter.Y, in.CraneSlewCenter.Z,
		in.TargetLoadHookPos.X, in.TargetLoadHookPos.Y, in.TargetLoadHookPos.Z,
	); err != nil {
		return nil, err
	}
	if in.BoomLengthM <= 0 {
		return nil, fmt.Errorf("crane_3d: boom_length_m must be > 0")
	}
	if in.NetLoadMassKg <= 0 {
		return nil, fmt.Errorf("crane_3d: net_load_mass_kg must be > 0")
	}
	if in.DynamicAmplification < 1.0 {
		in.DynamicAmplification = 1.0
	}
	if in.MaxAllowedUtilization <= 0 {
		in.MaxAllowedUtilization = 1.0
	}
	if in.TelePinningCode == "" {
		in.TelePinningCode = "STANDARD"
	}
	if in.Config.InterpolationPolicy == "" {
		in.Config.InterpolationPolicy = InterpolationStepDownStrict
	}

	// 1. Planimetric radius + deflection-adjusted lookup radius.
	dx := in.TargetLoadHookPos.X - in.CraneSlewCenter.X
	dy := in.TargetLoadHookPos.Y - in.CraneSlewCenter.Y
	res.GeometricRadiusM = math.Hypot(dx, dy)
	res.EffectiveLoadedRadiusM = res.GeometricRadiusM + in.BoomDeflectionMargin
	res.AuditTrace = append(res.AuditTrace,
		fmt.Sprintf("R_geom = √(%.3f²+%.3f²) = %.4f m; +%.3f m deflection → R_eff = %.4f m",
			dx, dy, res.GeometricRadiusM, in.BoomDeflectionMargin, res.EffectiveLoadedRadiusM))

	// 2. Boom kinematics: angle and tip elevation.
	boomM, ok := nearestLargerBoomLength(in.Config.Cells, in.BoomLengthM, in.TelePinningCode)
	if !ok {
		return nil, fmt.Errorf("crane_3d: no chart cells for tele %q at boom >= %.2fm",
			in.TelePinningCode, in.BoomLengthM)
	}
	if math.Abs(boomM-in.BoomLengthM) > 1e-3 {
		res.AuditTrace = append(res.AuditTrace,
			fmt.Sprintf("Boom step-up: requested %.2fm → using conservative chart entry %.2fm", in.BoomLengthM, boomM))
	}

	hReach := res.EffectiveLoadedRadiusM - in.BoomFootPinOffsetR
	if hReach <= 0 || hReach >= boomM {
		res.SafetyViolations = append(res.SafetyViolations,
			fmt.Sprintf("Geometric infeasibility: horizontal reach from boom pin (%.2fm) vs boom length (%.2fm)", hReach, boomM))
		res.IsSafeToLift = false
		return res, nil
	}
	boomAngleRad := math.Acos(hReach / boomM)
	res.BoomAngleDeg = boomAngleRad * (180.0 / math.Pi)
	res.BoomTipElevationZM = in.CraneSlewCenter.Z + in.BoomFootPinOffsetZ + boomM*math.Sin(boomAngleRad)

	reqClearanceZ := in.TargetLoadHookPos.Z + in.RiggingHeightM + in.HookBlockHeightM + in.MinTwoBlockClearance
	res.AvailableHeadroomM = res.BoomTipElevationZM - reqClearanceZ
	res.AuditTrace = append(res.AuditTrace,
		fmt.Sprintf("θ = arccos(%.3f/%.2f) = %.2f°; Z_tip = %.3f m; headroom = %.3f m",
			hReach, boomM, res.BoomAngleDeg, res.BoomTipElevationZM, res.AvailableHeadroomM))
	if res.AvailableHeadroomM < 0 {
		res.SafetyViolations = append(res.SafetyViolations,
			fmt.Sprintf("Anti-two-block: boom tip Z (%.2fm) is %.2fm below required clearance Z (%.2fm)",
				res.BoomTipElevationZM, -res.AvailableHeadroomM, reqClearanceZ))
	}

	// 3. Chart lookup: bounding cells for effective radius.
	lowerCell, upperCell, err := lookupBoundingCells(in.Config.Cells, boomM, in.TelePinningCode, res.EffectiveLoadedRadiusM)
	if err != nil {
		return nil, err
	}
	if upperCell == nil {
		res.SafetyViolations = append(res.SafetyViolations,
			fmt.Sprintf("Effective radius %.2fm exceeds max tabulated radius for boom %.2fm", res.EffectiveLoadedRadiusM, boomM))
		res.IsSafeToLift = false
		return res, nil
	}
	if lowerCell == nil {
		// Radius is smaller than the minimum table entry: use minimum entry (most conservative).
		res.ChartGrossCapacityKg = upperCell.CapacityKg
		res.GoverningLimitType = upperCell.LimitGovernedBy
		res.InterpolationApplied = fmt.Sprintf("radius_below_min_table_entry (using %.2fm cell)", upperCell.RadiusM)
		res.AuditTrace = append(res.AuditTrace, res.InterpolationApplied)
	} else {
		// 4. Boundary-safe interpolation / step-down.
		const eps = 1e-4
		if math.Abs(upperCell.RadiusM-lowerCell.RadiusM) < eps {
			// Exact match.
			res.ChartGrossCapacityKg = upperCell.CapacityKg
			res.GoverningLimitType = upperCell.LimitGovernedBy
			res.InterpolationApplied = fmt.Sprintf("exact_match at %.2fm", upperCell.RadiusM)
		} else if in.Config.InterpolationPolicy == InterpolationLinearRadiusOnly &&
			lowerCell.LimitGovernedBy == upperCell.LimitGovernedBy {
			// Safe linear interpolation: same failure mode on both sides.
			ratio := (res.EffectiveLoadedRadiusM - lowerCell.RadiusM) / (upperCell.RadiusM - lowerCell.RadiusM)
			res.ChartGrossCapacityKg = lowerCell.CapacityKg - ratio*(lowerCell.CapacityKg-upperCell.CapacityKg)
			res.GoverningLimitType = upperCell.LimitGovernedBy
			res.InterpolationApplied = fmt.Sprintf("linear_radius (%.2fm→%.2fm, ratio=%.4f)", lowerCell.RadiusM, upperCell.RadiusM, ratio)
		} else {
			// Step-down strict (mandatory when modes differ or policy requires it).
			res.ChartGrossCapacityKg = upperCell.CapacityKg
			res.GoverningLimitType = upperCell.LimitGovernedBy
			if lowerCell.LimitGovernedBy != upperCell.LimitGovernedBy {
				res.InterpolationApplied = fmt.Sprintf("forced_step_down_across_physics_boundary (%s→%s at %.2fm)",
					lowerCell.LimitGovernedBy, upperCell.LimitGovernedBy, upperCell.RadiusM)
			} else {
				res.InterpolationApplied = fmt.Sprintf("step_down_strict to %.2fm cell", upperCell.RadiusM)
			}
		}
		res.AuditTrace = append(res.AuditTrace,
			fmt.Sprintf("Chart capacity: %.0f kg [%s]; interpolation: %s",
				res.ChartGrossCapacityKg, res.GoverningLimitType, res.InterpolationApplied))
	}

	// 5. Hoist reeving capacity and single-line pull limit.
	res.ReevingCapacityKg = float64(in.ReevingPartsOfLine) * in.SingleLinePullLimitKg
	if upperCell != nil && upperCell.MinReevingParts != nil && in.ReevingPartsOfLine < *upperCell.MinReevingParts {
		res.SafetyViolations = append(res.SafetyViolations,
			fmt.Sprintf("Insufficient reeving: %d parts rigged, chart requires min %d parts",
				in.ReevingPartsOfLine, *upperCell.MinReevingParts))
	}

	res.GoverningCapacityKg = res.ChartGrossCapacityKg
	if res.ReevingCapacityKg > 0 && res.ReevingCapacityKg < res.GoverningCapacityKg {
		res.GoverningCapacityKg = res.ReevingCapacityKg
		res.GoverningLimitType = FailureModeReeving
		res.AuditTrace = append(res.AuditTrace,
			fmt.Sprintf("Reeving governs: %.0f kg < chart %.0f kg", res.ReevingCapacityKg, res.ChartGrossCapacityKg))
	}

	// 6. Hook load = (net + spreader + slings)×DAF + block + ball + rope.
	hoistDropM := math.Max(0, res.BoomTipElevationZM-in.TargetLoadHookPos.Z)
	ropeKg := hoistDropM * float64(in.ReevingPartsOfLine) * in.HoistRopeMassPerMeter
	dynamicRiggedKg := (in.NetLoadMassKg + in.SpreaderBeamMassKg + in.SlingsAndShacklesKg) * in.DynamicAmplification
	fixedDeductKg := in.MainHookBlockMassKg + in.AuxBallMassKg + ropeKg
	res.TotalGrossHookLoadKg = dynamicRiggedKg + fixedDeductKg

	if in.Config.CapacityType == "net" {
		// Net chart: manufacturer already deducted main block; deduct rigging+DAF below the hook.
		res.TotalRiggingDeductionKg = (in.SpreaderBeamMassKg + in.SlingsAndShacklesKg) * in.DynamicAmplification
	} else {
		// Gross chart (default): deduct spreader+slings+DAF + block+ball+rope.
		res.TotalRiggingDeductionKg = (in.SpreaderBeamMassKg+in.SlingsAndShacklesKg)*in.DynamicAmplification + fixedDeductKg
	}
	res.NetAvailableCapacityKg = res.GoverningCapacityKg - res.TotalRiggingDeductionKg

	res.AuditTrace = append(res.AuditTrace,
		fmt.Sprintf("Gross hook load = (%.0f+%.0f+%.0f)×%.3f + %.0f + %.0f + %.0f(rope) = %.0f kg",
			in.NetLoadMassKg, in.SpreaderBeamMassKg, in.SlingsAndShacklesKg,
			in.DynamicAmplification, in.MainHookBlockMassKg, in.AuxBallMassKg, ropeKg,
			res.TotalGrossHookLoadKg))

	// 7. Utilization and final verdict.
	res.UtilizationRatio = res.TotalGrossHookLoadKg / res.GoverningCapacityKg
	res.UtilizationPercent = math.Round(res.UtilizationRatio*10000) / 100.0

	if res.UtilizationRatio > in.MaxAllowedUtilization {
		res.SafetyViolations = append(res.SafetyViolations,
			fmt.Sprintf("Overload: utilization %.2f%% exceeds permitted max %.2f%%",
				res.UtilizationPercent, in.MaxAllowedUtilization*100.0))
	}
	if res.NetAvailableCapacityKg < in.NetLoadMassKg*in.DynamicAmplification {
		res.SafetyViolations = append(res.SafetyViolations,
			fmt.Sprintf("Net capacity %.0f kg insufficient for net dynamic load %.0f kg",
				res.NetAvailableCapacityKg, in.NetLoadMassKg*in.DynamicAmplification))
	}

	res.IsSafeToLift = len(res.SafetyViolations) == 0
	return res, nil
}
