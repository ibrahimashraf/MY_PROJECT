package metallurgy

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

// CertificateStandard defines the metallurgical compliance specification standard.
type CertificateStandard string

const (
	CertEN10204_2_1 CertificateStandard = "EN_10204_2.1" // Declaration of compliance with the order
	CertEN10204_2_2 CertificateStandard = "EN_10204_2.2" // Test report based on non-specific inspection
	CertEN10204_3_1 CertificateStandard = "EN_10204_3.1" // Inspection certificate validated by manufacturer's authorized inspection representative
	CertEN10204_3_2 CertificateStandard = "EN_10204_3.2" // Inspection certificate validated by manufacturer + independent third-party inspector
)

var (
	ErrInvalidHeatNumber      = errors.New("metallurgy: heat number must be non-empty and alphanumeric")
	ErrInvalidSteelGrade       = errors.New("metallurgy: steel grade specification cannot be empty")
	ErrEmptyMillName          = errors.New("metallurgy: steel mill name cannot be empty")
	ErrInvalidCertStandard    = errors.New("metallurgy: invalid EN 10204 certificate standard")
	ErrInvalidChemicalRange   = errors.New("metallurgy: chemical composition element percentage out of physical range (0-100%)")
	ErrInvalidMechanicalValue = errors.New("metallurgy: yield or tensile strength must be strictly positive")
	ErrInvalidStressCycle     = errors.New("metallurgy: stress cycle amplitude or allowable cycle count must be positive")
	ErrFatigueLifeExceeded    = errors.New("metallurgy: cumulative fatigue damage index D >= 1.0 (critical structural fatigue failure)")
)

var heatNumberRegex = regexp.MustCompile(`^[A-Za-z0-9\-_./]+$`)

// ChemicalComposition stores the certified chemical ladle analysis of the heat melt.
type ChemicalComposition struct {
	CarbonPercent     float64 `json:"carbon_percent"`     // C %
	SiliconPercent    float64 `json:"silicon_percent"`    // Si %
	ManganesePercent  float64 `json:"manganese_percent"`  // Mn %
	PhosphorusPercent float64 `json:"phosphorus_percent"` // P %
	SulfurPercent     float64 `json:"sulfur_percent"`     // S %
	ChromiumPercent   float64 `json:"chromium_percent"`   // Cr %
	NickelPercent     float64 `json:"nickel_percent"`     // Ni %
	MolybdenumPercent float64 `json:"molybdenum_percent"` // Mo %
	VanadiumPercent   float64 `json:"vanadium_percent"`   // V %
}

// CarbonEquivalent (CEV) calculates the IIW carbon equivalent formula for weldability:
// CEV = C + Mn/6 + (Cr + Mo + V)/5 + (Ni + Cu)/15
func (c ChemicalComposition) CarbonEquivalent() float64 {
	return c.CarbonPercent + (c.ManganesePercent / 6.0) +
		((c.ChromiumPercent + c.MolybdenumPercent + c.VanadiumPercent) / 5.0) +
		(c.NickelPercent / 15.0)
}

func (c ChemicalComposition) Validate() error {
	elements := []float64{
		c.CarbonPercent, c.SiliconPercent, c.ManganesePercent,
		c.PhosphorusPercent, c.SulfurPercent, c.ChromiumPercent,
		c.NickelPercent, c.MolybdenumPercent, c.VanadiumPercent,
	}
	for _, val := range elements {
		if val < 0.0 || val > 100.0 {
			return ErrInvalidChemicalRange
		}
	}
	return nil
}

// MechanicalProperties stores certified tensile, yield, elongation, and Charpy impact values.
type MechanicalProperties struct {
	YieldStrengthMPa   float64 `json:"yield_strength_mpa"`   // ReH or Rp0.2 (MPa)
	TensileStrengthMPa float64 `json:"tensile_strength_mpa"` // Rm (MPa)
	ElongationPercent  float64 `json:"elongation_percent"`   // A5 (%)
	CharpyImpactJoules float64 `json:"charpy_impact_joules"` // Absorbed energy KV (J) at test temperature
	TestTemperatureC   float64 `json:"test_temperature_c"`   // e.g. -20°C, -40°C for arctic/offshore grades
}

func (m MechanicalProperties) Validate() error {
	if m.YieldStrengthMPa <= 0.0 || m.TensileStrengthMPa <= 0.0 {
		return ErrInvalidMechanicalValue
	}
	if m.TensileStrengthMPa < m.YieldStrengthMPa {
		return fmt.Errorf("%w: tensile strength (%.1f MPa) cannot be less than yield strength (%.1f MPa)",
			ErrInvalidMechanicalValue, m.TensileStrengthMPa, m.YieldStrengthMPa)
	}
	if m.ElongationPercent <= 0.0 || m.ElongationPercent > 100.0 {
		return fmt.Errorf("%w: elongation must be in (0, 100]", ErrInvalidMechanicalValue)
	}
	return nil
}

// MillHeatCertificate represents the verified EN 10204 3.1 / 3.2 inspection certificate
// directly linked to the steel melt.
type MillHeatCertificate struct {
	HeatDID              string               `json:"heat_did"`               // e.g. "did:integin:heat:AR-2026-X8911"
	HeatNumber           string               `json:"heat_number"`            // Furnace melt identification
	SteelMillName        string               `json:"steel_mill_name"`        // e.g. "HADEED / SABIC", "Voestalpine", "SSAB"
	MillCountryISO2      string               `json:"mill_country_iso2"`      // "SA", "AT", "SE"
	Standard             CertificateStandard  `json:"standard"`               // EN 10204 3.1 or 3.2
	SteelGrade           string               `json:"steel_grade"`            // e.g. "S355ML", "S690QL", "ASTM A514"
	MeltDate             time.Time            `json:"melt_date"`
	Chemistry            ChemicalComposition  `json:"chemistry"`
	Mechanicals          MechanicalProperties `json:"mechanicals"`
	MillInspectorSigner  string               `json:"mill_inspector_signer"`  // Authorized QA agent
	ThirdPartyInspector  string               `json:"third_party_inspector,omitempty"` // Required for 3.2 (e.g. DNV, TÜV)
	CertificateDocSHA256 string               `json:"certificate_doc_sha256"` // Digest of PDF mill cert
}

func (c MillHeatCertificate) Validate() error {
	if !heatNumberRegex.MatchString(c.HeatNumber) {
		return ErrInvalidHeatNumber
	}
	if strings.TrimSpace(c.SteelMillName) == "" {
		return ErrEmptyMillName
	}
	if strings.TrimSpace(c.SteelGrade) == "" {
		return ErrInvalidSteelGrade
	}
	switch c.Standard {
	case CertEN10204_2_1, CertEN10204_2_2, CertEN10204_3_1, CertEN10204_3_2:
	default:
		return ErrInvalidCertStandard
	}
	if c.Standard == CertEN10204_3_2 && strings.TrimSpace(c.ThirdPartyInspector) == "" {
		return fmt.Errorf("%w: EN 10204 3.2 requires third party inspector identification", ErrInvalidCertStandard)
	}
	if err := c.Chemistry.Validate(); err != nil {
		return err
	}
	if err := c.Mechanicals.Validate(); err != nil {
		return err
	}
	return nil
}

// StressCycle represents a recorded operational load reversal cycle under crane/vessel service.
type StressCycle struct {
	CycleCount        uint64  `json:"cycle_count"`         // ni: Number of applied stress cycles
	StressRangeMPa    float64 `json:"stress_range_mpa"`    // Δσ: Cyclic stress amplitude
	AllowableCyclesNi float64 `json:"allowable_cycles_ni"` // Ni: Fatigue endurance limit from design S-N curve
}

// CumulativeFatigueState models structural fatigue accumulation using Palmgren-Miner's Rule:
// Damage Index D = \sum (ni / Ni)
// Failure occurs when D >= 1.0.
type CumulativeFatigueState struct {
	AssetDID            string        `json:"asset_did"`
	TotalCyclesRecorded uint64        `json:"total_cycles_recorded"`
	CyclesHistory       []StressCycle `json:"cycles_history"`
	LastUpdated         time.Time     `json:"last_updated"`
}

// CalculateDamageIndex computes Miner's cumulative fatigue damage fraction D.
func (f *CumulativeFatigueState) CalculateDamageIndex() (float64, error) {
	var totalDamage float64
	for _, cycle := range f.CyclesHistory {
		if cycle.AllowableCyclesNi <= 0.0 || cycle.CycleCount <= 0 {
			return 0.0, ErrInvalidStressCycle
		}
		damageFraction := float64(cycle.CycleCount) / cycle.AllowableCyclesNi
		totalDamage += damageFraction
	}
	return totalDamage, nil
}

// RemainingSafeWorkingLifeFraction returns the structural margin (1.0 - D).
// If D >= 1.0, an error is returned triggering statutory equipment quarantine.
func (f *CumulativeFatigueState) RemainingSafeWorkingLifeFraction() (float64, error) {
	d, err := f.CalculateDamageIndex()
	if err != nil {
		return 0.0, err
	}
	if d >= 1.0 {
		return 0.0, fmt.Errorf("%w: damage index %.4f exceeds 1.0 limit", ErrFatigueLifeExceeded, d)
	}
	return math.Max(0.0, 1.0-d), nil
}
