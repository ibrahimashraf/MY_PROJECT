package jurisdictions

import (
	"errors"
	"regexp"
	"strings"
)

// LifecycleState represents the operational status of a jurisdiction profile.
type LifecycleState string

const (
	LifecycleActive   LifecycleState = "ACTIVE"
	LifecycleDraft    LifecycleState = "DRAFT"
	LifecycleArchived LifecycleState = "ARCHIVED"
)

// PillarType enumerates the 7 Sovereign Compliance Pillars that each jurisdiction
// may enforce differently. Every adapter must map to exactly one pillar.
type PillarType string

const (
	PillarSafetyRegulation   PillarType = "SAFETY_REGULATION"
	PillarTaxFiscal          PillarType = "TAX_FISCAL"
	PillarAccreditation      PillarType = "ACCREDITATION"
	PillarCertificateMarking PillarType = "CERTIFICATE_MARKING"
	PillarCurrencyTrade      PillarType = "CURRENCY_TRADE"
	PillarEnvironmentalESG   PillarType = "ENVIRONMENTAL_ESG"
	PillarLaborWorkforce     PillarType = "LABOR_WORKFORCE"
)

// TaxAuthority describes a sovereign tax or fiscal authority with its identification rules.
type TaxAuthority struct {
	Name          string `json:"name"`
	TaxIDLabel    string `json:"tax_id_label"`
	TaxIDRegex    string `json:"tax_id_regex"`
	CommercialReg string `json:"commercial_reg_label"`
}

// Validate checks that the tax authority has a name and that any regex compiles.
func (ta TaxAuthority) Validate() error {
	if strings.TrimSpace(ta.Name) == "" {
		return ErrEmptyTaxAuthorityName
	}
	if ta.TaxIDRegex != "" {
		if _, err := regexp.Compile(ta.TaxIDRegex); err != nil {
			return ErrInvalidTaxIDRegex
		}
	}
	return nil
}

// SafetyRegulator describes a national safety or inspection regulatory body.
type SafetyRegulator struct {
	Name         string `json:"name"`
	Website      string `json:"website,omitempty"`
	Inspectorate string `json:"inspectorate,omitempty"`
}

// SubDivisionProfile describes a first-order administrative subdivision (state, province, emirate)
// that carries its own regulatory or tax profile.
type SubDivisionProfile struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Regulator string `json:"regulator,omitempty"`
	Notes     string `json:"notes,omitempty"`
}

// CountryProfile is the full metadata card for a sovereign jurisdiction.
// It is the primary entity in pkg/jurisdictions and carries all metadata needed
// by the 7-Pillar compliance adapters and the in-memory registry.
type CountryProfile struct {
	ISO2              string               `json:"iso2"`
	ISO3              string               `json:"iso3"`
	NumericCode       string               `json:"numeric_code"`
	CommonName        string               `json:"common_name"`
	OfficialName      string               `json:"official_name,omitempty"`
	PrimaryLanguage   string               `json:"primary_language"`
	SecondaryLanguage string               `json:"secondary_language,omitempty"`
	CurrencyCode      string               `json:"currency_code"`
	CurrencySymbol    string               `json:"currency_symbol,omitempty"`
	Region            string               `json:"region"`
	SubRegion         string               `json:"sub_region,omitempty"`
	Lifecycle         LifecycleState       `json:"lifecycle"`
	TaxAuthority      TaxAuthority         `json:"tax_authority"`
	SafetyRegulators  []SafetyRegulator    `json:"safety_regulators,omitempty"`
	SubDivisions      []SubDivisionProfile `json:"sub_divisions,omitempty"`
	ActivePillars     []PillarType         `json:"active_pillars"`
	Notes             string               `json:"notes,omitempty"`
}

var (
	ErrEmptyISO2             = errors.New("jurisdiction ISO2 code cannot be empty")
	ErrInvalidISO2Length     = errors.New("jurisdiction ISO2 code must be exactly 2 uppercase letters")
	ErrEmptyISO3             = errors.New("jurisdiction ISO3 code cannot be empty")
	ErrInvalidISO3Length     = errors.New("jurisdiction ISO3 code must be exactly 3 uppercase letters")
	ErrEmptyCommonName       = errors.New("jurisdiction common name cannot be empty")
	ErrEmptyCurrencyCode     = errors.New("jurisdiction currency code cannot be empty")
	ErrInvalidCurrencyLength = errors.New("jurisdiction currency code must be exactly 3 uppercase letters")
	ErrEmptyLifecycle        = errors.New("jurisdiction lifecycle state cannot be empty")
	ErrInvalidLifecycle      = errors.New("jurisdiction lifecycle state must be ACTIVE, DRAFT, or ARCHIVED")
	ErrEmptyPrimaryLanguage  = errors.New("jurisdiction primary language cannot be empty")
	ErrEmptyRegion           = errors.New("jurisdiction region cannot be empty")
	ErrEmptyActivePillars    = errors.New("jurisdiction must declare at least one active compliance pillar")
	ErrUnknownPillarType     = errors.New("jurisdiction references an unknown compliance pillar type")
	ErrEmptyTaxAuthorityName = errors.New("tax authority name cannot be empty")
	ErrInvalidTaxIDRegex     = errors.New("tax authority ID regex pattern is invalid")
	ErrEmptySubDivisionCode  = errors.New("sub-division code cannot be empty")
	ErrEmptySubDivisionName  = errors.New("sub-division name cannot be empty")
)

var iso2Pattern = regexp.MustCompile(`^[A-Z]{2}$`)
var iso3Pattern = regexp.MustCompile(`^[A-Z]{3}$`)
var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

// ValidLifecycleStates is the set of accepted lifecycle values.
var ValidLifecycleStates = map[LifecycleState]bool{
	LifecycleActive:   true,
	LifecycleDraft:    true,
	LifecycleArchived: true,
}

// ValidPillarTypes is the set of accepted 7-Pillar types.
var ValidPillarTypes = map[PillarType]bool{
	PillarSafetyRegulation:   true,
	PillarTaxFiscal:          true,
	PillarAccreditation:      true,
	PillarCertificateMarking: true,
	PillarCurrencyTrade:      true,
	PillarEnvironmentalESG:   true,
	PillarLaborWorkforce:     true,
}

// Validate performs exhaustive structural validation on a CountryProfile.
// It enforces ISO 3166-1 code formats, required metadata fields, valid lifecycle,
// valid pillar references, and sub-division integrity. It is deterministic and
// allocation-free for the hot path (regex pre-compiled at package init).
func (cp CountryProfile) Validate() error {
	cp.ISO2 = strings.TrimSpace(strings.ToUpper(cp.ISO2))
	cp.ISO3 = strings.TrimSpace(strings.ToUpper(cp.ISO3))
	cp.CurrencyCode = strings.TrimSpace(strings.ToUpper(cp.CurrencyCode))

	if cp.ISO2 == "" {
		return ErrEmptyISO2
	}
	if !iso2Pattern.MatchString(cp.ISO2) {
		return ErrInvalidISO2Length
	}
	if cp.ISO3 == "" {
		return ErrEmptyISO3
	}
	if !iso3Pattern.MatchString(cp.ISO3) {
		return ErrInvalidISO3Length
	}
	if strings.TrimSpace(cp.CommonName) == "" {
		return ErrEmptyCommonName
	}
	if cp.CurrencyCode == "" {
		return ErrEmptyCurrencyCode
	}
	if !currencyPattern.MatchString(cp.CurrencyCode) {
		return ErrInvalidCurrencyLength
	}
	if cp.PrimaryLanguage == "" {
		return ErrEmptyPrimaryLanguage
	}
	if strings.TrimSpace(cp.Region) == "" {
		return ErrEmptyRegion
	}
	if cp.Lifecycle == "" {
		return ErrEmptyLifecycle
	}
	if !ValidLifecycleStates[cp.Lifecycle] {
		return ErrInvalidLifecycle
	}
	if len(cp.ActivePillars) == 0 {
		return ErrEmptyActivePillars
	}
	for _, p := range cp.ActivePillars {
		if !ValidPillarTypes[p] {
			return ErrUnknownPillarType
		}
	}
	if err := cp.TaxAuthority.Validate(); err != nil {
		return err
	}
	for i, sub := range cp.SubDivisions {
		if strings.TrimSpace(sub.Code) == "" {
			return ErrEmptySubDivisionCode
		}
		if strings.TrimSpace(sub.Name) == "" {
			return ErrEmptySubDivisionName
		}
		cp.SubDivisions[i].Code = strings.TrimSpace(strings.ToUpper(sub.Code))
	}
	return nil
}

// DID returns the canonical W3C DID string for this jurisdiction.
// Format: did:integin:jurisdiction:<ISO2>
func (cp CountryProfile) DID() string {
	return "did:integin:jurisdiction:" + strings.ToUpper(cp.ISO2)
}

// HasPillar reports whether this jurisdiction enforces a given compliance pillar.
func (cp CountryProfile) HasPillar(p PillarType) bool {
	for _, ap := range cp.ActivePillars {
		if ap == p {
			return true
		}
	}
	return false
}

// LookupSubDivision finds a sub-division by its code (case-insensitive).
// Returns the subdivision and true if found, zero value and false otherwise.
func (cp CountryProfile) LookupSubDivision(code string) (SubDivisionProfile, bool) {
	upper := strings.ToUpper(strings.TrimSpace(code))
	for _, sub := range cp.SubDivisions {
		if strings.ToUpper(sub.Code) == upper {
			return sub, true
		}
	}
	return SubDivisionProfile{}, false
}
