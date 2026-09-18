package rulesengine

import (
	"fmt"
	"math"

	"cel.dev/cel-go/cel"
)

// Lift-plan physics core: the deterministic gates behind every number a lift
// plan asserts. Sources: DNV Rules for Marine Operations 1996 Pt.2 Ch.5 and
// DNV RP H103 Ch.9 (Braemar "Offshore Lifting for Subsea Equipment" deck),
// ASME B30.5 proof-load tiers. Jurisdiction-free: pure mathematics.
//
// Each gate ships as a guarded concrete function plus the mirror CEL
// expression, following the house pattern (geotech.go, rigging.go).
//
// Deliberate exclusions (corpus tokens without extractable definitions,
// refused rather than guessed):
//   - "SPL" (Braemar deck §Capacity, adjacent to "DHL = W+Wrig"): no
//     definition in any extractable corpus text; unmodeled pending the
//     cited DNV source, never defaulted.
//   - Fsling closed form (SKL/kCoG/DW combination): the combination
//     formula is raster-locked in the deck; SKL alone is already enforced
//     by DNVSKLGateExpression, kCoG/DW stay caller-side inputs elsewhere.

const (
	// LiftPlanDHLRuleID identifies the dynamic-hook-load ledger gate.
	LiftPlanDHLRuleID = "liftplan.dynamic-hook-load"
	// LiftPlanCapacityRuleID identifies the four-gate capacity verdict.
	LiftPlanCapacityRuleID = "liftplan.four-gate-capacity"
	// LiftPlanLateralRuleID identifies the shackle-bow lateral load rule.
	LiftPlanLateralRuleID = "liftplan.shackle-lateral-load"
	// LiftPlanDAFRuleID identifies the DAF table-applicability discipline.
	LiftPlanDAFRuleID = "liftplan.daf-table-applicability"
)

// LiftPlanDHLExpression: the hook load is exactly structure (inaccuracy
// already folded in) plus rigging, and must be positive.
const LiftPlanDHLExpression = `dhl_t == structure_t + rigging_t && dhl_t > 0.0`

// LiftPlanCapacityExpression: all four utilisation gates at or below 1.0.
const LiftPlanCapacityExpression = `crane_util <= 1.0 && rigging_util <= 1.0 && steel_util <= 1.0 && object_util <= 1.0`

// LiftPlanLateralExpression: lateral load is at least 3% of design load.
const LiftPlanLateralExpression = `lateral_t >= 0.03 * design_t`

// LiftPlanDAFExpression: tabulated in-air DAF applies only in minor sea
// states (Hs <= 2.5 m); anything above needs separate estimation.
const LiftPlanDAFExpression = `hs_m >= 0.0 && hs_m <= 2.5 && daf >= 1.0`

// MaxTabulatedDAFSeaStateM is the DNV minor-sea-state ceiling for tabulated
// in-air DAF values.
const MaxTabulatedDAFSeaStateM = 2.5

// MinShackleLateralFactor is the minimum lateral load fraction acting in the
// shackle bow.
const MinShackleLateralFactor = 0.03

func liftPlanRejectNaN(name string, vals ...float64) error {
	for _, v := range vals {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("liftplan: NaN/Inf %s rejected", name)
		}
	}
	return nil
}

// ApplyWeightInaccuracy folds the weight-inaccuracy factor into the base
// structural weight: W = base * (1 + factor). Factor must be >= 0.
func ApplyWeightInaccuracy(baseT, factor float64) (float64, error) {
	if err := liftPlanRejectNaN("weight", baseT, factor); err != nil {
		return 0, err
	}
	if baseT <= 0 {
		return 0, fmt.Errorf("liftplan: base weight must be > 0, got %g", baseT)
	}
	if factor < 0 {
		return 0, fmt.Errorf("liftplan: inaccuracy factor must be >= 0, got %g", factor)
	}
	return baseT * (1 + factor), nil
}

// DynamicHookLoad is the DHL ledger: DHL = W + Wrig, where W already includes
// the weight-inaccuracy factor. Both inputs must be > 0.
func DynamicHookLoad(structureT, riggingT float64) (float64, error) {
	if err := liftPlanRejectNaN("hook load", structureT, riggingT); err != nil {
		return 0, err
	}
	if structureT <= 0 {
		return 0, fmt.Errorf("liftplan: structure weight must be > 0, got %g", structureT)
	}
	if riggingT <= 0 {
		return 0, fmt.Errorf("liftplan: rigging weight must be > 0, got %g", riggingT)
	}
	return structureT + riggingT, nil
}

// CapacityGates carries the four DNV capacity checks: crane vs DHL, rigging
// tension vs MBL, structural steel, lifted object. Loads and capacities pair
// by position.
type CapacityGates struct {
	CraneLoadT   float64
	CraneCapT    float64
	RiggingLoadT float64
	RiggingCapT  float64
	SteelLoadT   float64
	SteelCapT    float64
	ObjectLoadT  float64
	ObjectCapT   float64
}

// CapacityVerdict is the per-gate and overall pass/fail record.
type CapacityVerdict struct {
	CraneUtil   float64 `json:"crane_util"`
	RiggingUtil float64 `json:"rigging_util"`
	SteelUtil   float64 `json:"steel_util"`
	ObjectUtil  float64 `json:"object_util"`
	Passed      bool    `json:"passed"`
}

func gateUtil(loadT, capT float64, name string) (float64, error) {
	if err := liftPlanRejectNaN(name, loadT, capT); err != nil {
		return 0, err
	}
	if capT <= 0 {
		return 0, fmt.Errorf("liftplan: %s capacity must be > 0, got %g", name, capT)
	}
	if loadT < 0 {
		return 0, fmt.Errorf("liftplan: %s load must be >= 0, got %g", name, loadT)
	}
	return loadT / capT, nil
}

// EvaluateCapacityGates runs the four DNV capacity gates. Every utilisation
// must be <= 1.0; any refusal is fail-closed.
func EvaluateCapacityGates(g CapacityGates) (CapacityVerdict, error) {
	crane, err := gateUtil(g.CraneLoadT, g.CraneCapT, "crane")
	if err != nil {
		return CapacityVerdict{}, err
	}
	rigging, err := gateUtil(g.RiggingLoadT, g.RiggingCapT, "rigging")
	if err != nil {
		return CapacityVerdict{}, err
	}
	steel, err := gateUtil(g.SteelLoadT, g.SteelCapT, "steel")
	if err != nil {
		return CapacityVerdict{}, err
	}
	object, err := gateUtil(g.ObjectLoadT, g.ObjectCapT, "object")
	if err != nil {
		return CapacityVerdict{}, err
	}
	v := CapacityVerdict{CraneUtil: crane, RiggingUtil: rigging, SteelUtil: steel, ObjectUtil: object}
	v.Passed = crane <= 1.0 && rigging <= 1.0 && steel <= 1.0 && object <= 1.0
	return v, nil
}

// ShackleLateralLoad is the minimum lateral load in the shackle bow: 3% of
// the design load acting laterally.
func ShackleLateralLoad(designT float64) (float64, error) {
	if err := liftPlanRejectNaN("design load", designT); err != nil {
		return 0, err
	}
	if designT <= 0 {
		return 0, fmt.Errorf("liftplan: design load must be > 0, got %g", designT)
	}
	return MinShackleLateralFactor * designT, nil
}

// GateDAFTableApplicability enforces the DAF discipline: tabulated in-air DAF
// values apply only at Hs <= 2.5 m. Higher sea states and all subsea lifts
// must estimate DAF separately — this gate refuses them fail-closed.
func GateDAFTableApplicability(hsM float64) (bool, error) {
	if err := liftPlanRejectNaN("sea state", hsM); err != nil {
		return false, err
	}
	if hsM < 0 {
		return false, fmt.Errorf("liftplan: significant wave height must be >= 0, got %g", hsM)
	}
	if hsM > MaxTabulatedDAFSeaStateM {
		return false, fmt.Errorf("liftplan: Hs %g m exceeds tabulated-DAF ceiling %g m (estimate DAF separately)", hsM, MaxTabulatedDAFSeaStateM)
	}
	return true, nil
}

// DAFBand is one Hs-indexed row of a cited in-air DAF table: it covers sea
// states up to MaxHsM with the tabulated factor DAF.
type DAFBand struct {
	MaxHsM float64 `json:"max_hs_m"`
	DAF    float64 `json:"daf"`
}

// DAFTable is owner-supplied cited data (e.g. the DNV in-air table
// referenced by the Braemar deck §Capacity). Tables are bound data, never
// computed: Source must name the cited table so the provenance survives.
type DAFTable struct {
	Source string    `json:"source"`
	Bands  []DAFBand `json:"bands"`
}

// LookupDAF resolves the tabulated DAF for hsM against a cited table. Bands
// must be ascending by MaxHsM with every DAF >= 1.0; Hs above the table or
// above the 2.5 m discipline ceiling refuses fail-closed.
func LookupDAF(table DAFTable, hsM float64) (float64, error) {
	if err := liftPlanRejectNaN("sea state", hsM); err != nil {
		return 0, err
	}
	if hsM < 0 {
		return 0, fmt.Errorf("liftplan: significant wave height must be >= 0, got %g", hsM)
	}
	if table.Source == "" {
		return 0, fmt.Errorf("liftplan: DAF table source citation required (tables are bound data)")
	}
	if len(table.Bands) == 0 {
		return 0, fmt.Errorf("liftplan: DAF table %q has no bands", table.Source)
	}
	prev := -1.0
	for i, band := range table.Bands {
		if err := liftPlanRejectNaN("DAF band", band.MaxHsM, band.DAF); err != nil {
			return 0, err
		}
		if band.MaxHsM <= prev {
			return 0, fmt.Errorf("liftplan: DAF table %q band %d not ascending (max Hs %g)", table.Source, i, band.MaxHsM)
		}
		if band.DAF < 1.0 {
			return 0, fmt.Errorf("liftplan: DAF table %q band %d factor %g below 1.0", table.Source, i, band.DAF)
		}
		prev = band.MaxHsM
	}
	for _, band := range table.Bands {
		if hsM <= band.MaxHsM {
			if ok, err := GateDAFTableApplicability(hsM); err != nil || !ok {
				return 0, err
			}
			return band.DAF, nil
		}
	}
	return 0, fmt.Errorf("liftplan: Hs %g m above DAF table %q top band (estimate DAF separately)", hsM, table.Source)
}

// ApplyDAF amplifies a static load by the resolved dynamic factor: dynamic =
// static * daf. DAF must be >= 1.0; dynamics never reduce the load.
func ApplyDAF(staticT, daf float64) (float64, error) {
	if err := liftPlanRejectNaN("dynamic load", staticT, daf); err != nil {
		return 0, err
	}
	if staticT <= 0 {
		return 0, fmt.Errorf("liftplan: static load must be > 0, got %g", staticT)
	}
	if daf < 1.0 {
		return 0, fmt.Errorf("liftplan: DAF must be >= 1.0, got %g", daf)
	}
	return staticT * daf, nil
}

// LiftPlanDHLGateVars declares the typed variables for the DHL ledger gate.
func LiftPlanDHLGateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"dhl_t":       cel.DoubleType,
		"structure_t": cel.DoubleType,
		"rigging_t":   cel.DoubleType,
	}
}

// LiftPlanCapacityGateVars declares the typed variables for the four-gate
// capacity verdict.
func LiftPlanCapacityGateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"crane_util":   cel.DoubleType,
		"rigging_util": cel.DoubleType,
		"steel_util":   cel.DoubleType,
		"object_util":  cel.DoubleType,
	}
}

// LiftPlanLateralGateVars declares the typed variables for the lateral-load
// rule.
func LiftPlanLateralGateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"lateral_t": cel.DoubleType,
		"design_t":  cel.DoubleType,
	}
}

// LiftPlanDAFGateVars declares the typed variables for the DAF discipline.
func LiftPlanDAFGateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"hs_m": cel.DoubleType,
		"daf":  cel.DoubleType,
	}
}
