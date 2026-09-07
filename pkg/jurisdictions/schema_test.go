package jurisdictions

import (
	"testing"
)

func validProfile() CountryProfile {
	return CountryProfile{
		ISO2:              "SA",
		ISO3:              "SAU",
		NumericCode:       "682",
		CommonName:        "Saudi Arabia",
		OfficialName:      "Kingdom of Saudi Arabia",
		PrimaryLanguage:   "ar",
		SecondaryLanguage: "en",
		CurrencyCode:      "SAR",
		CurrencySymbol:    "﷼",
		Region:            "Asia",
		SubRegion:         "Western Asia",
		Lifecycle:         LifecycleActive,
		TaxAuthority: TaxAuthority{
			Name:          "ZATCA",
			TaxIDLabel:    "VAT Number",
			TaxIDRegex:    `^3\d{13}$`,
			CommercialReg: "Commercial Registration",
		},
		SafetyRegulators: []SafetyRegulator{
			{Name: "Saudi Council of Engineers"},
			{Name: "SASO"},
		},
		ActivePillars: []PillarType{
			PillarSafetyRegulation,
			PillarTaxFiscal,
			PillarAccreditation,
			PillarCertificateMarking,
			PillarCurrencyTrade,
			PillarEnvironmentalESG,
			PillarLaborWorkforce,
		},
	}
}

func TestValidProfilePassesValidation(t *testing.T) {
	p := validProfile()
	if err := p.Validate(); err != nil {
		t.Fatalf("valid profile should pass validation, got: %v", err)
	}
}

func TestValidateISO2Empty(t *testing.T) {
	p := validProfile()
	p.ISO2 = ""
	if err := p.Validate(); err != ErrEmptyISO2 {
		t.Fatalf("expected ErrEmptyISO2, got: %v", err)
	}
}

func TestValidateISO2InvalidLength(t *testing.T) {
	p := validProfile()
	p.ISO2 = "USA"
	if err := p.Validate(); err != ErrInvalidISO2Length {
		t.Fatalf("expected ErrInvalidISO2Length, got: %v", err)
	}
}

func TestValidateISO2LowercaseNormalized(t *testing.T) {
	p := validProfile()
	p.ISO2 = "sa"
	if err := p.Validate(); err != nil {
		t.Fatalf("lowercase ISO2 should be normalized and pass, got: %v", err)
	}
}

func TestValidateISO3Empty(t *testing.T) {
	p := validProfile()
	p.ISO3 = ""
	if err := p.Validate(); err != ErrEmptyISO3 {
		t.Fatalf("expected ErrEmptyISO3, got: %v", err)
	}
}

func TestValidateISO3InvalidLength(t *testing.T) {
	p := validProfile()
	p.ISO3 = "US"
	if err := p.Validate(); err != ErrInvalidISO3Length {
		t.Fatalf("expected ErrInvalidISO3Length, got: %v", err)
	}
}

func TestValidateCommonNameEmpty(t *testing.T) {
	p := validProfile()
	p.CommonName = ""
	if err := p.Validate(); err != ErrEmptyCommonName {
		t.Fatalf("expected ErrEmptyCommonName, got: %v", err)
	}
}

func TestValidateCurrencyCodeEmpty(t *testing.T) {
	p := validProfile()
	p.CurrencyCode = ""
	if err := p.Validate(); err != ErrEmptyCurrencyCode {
		t.Fatalf("expected ErrEmptyCurrencyCode, got: %v", err)
	}
}

func TestValidateCurrencyCodeInvalid(t *testing.T) {
	p := validProfile()
	p.CurrencyCode = "US"
	if err := p.Validate(); err != ErrInvalidCurrencyLength {
		t.Fatalf("expected ErrInvalidCurrencyLength, got: %v", err)
	}
}

func TestValidatePrimaryLanguageEmpty(t *testing.T) {
	p := validProfile()
	p.PrimaryLanguage = ""
	if err := p.Validate(); err != ErrEmptyPrimaryLanguage {
		t.Fatalf("expected ErrEmptyPrimaryLanguage, got: %v", err)
	}
}

func TestValidateRegionEmpty(t *testing.T) {
	p := validProfile()
	p.Region = ""
	if err := p.Validate(); err != ErrEmptyRegion {
		t.Fatalf("expected ErrEmptyRegion, got: %v", err)
	}
}

func TestValidateLifecycleEmpty(t *testing.T) {
	p := validProfile()
	p.Lifecycle = ""
	if err := p.Validate(); err != ErrEmptyLifecycle {
		t.Fatalf("expected ErrEmptyLifecycle, got: %v", err)
	}
}

func TestValidateLifecycleInvalid(t *testing.T) {
	p := validProfile()
	p.Lifecycle = "INVALID"
	if err := p.Validate(); err != ErrInvalidLifecycle {
		t.Fatalf("expected ErrInvalidLifecycle, got: %v", err)
	}
}

func TestValidateActivePillarsEmpty(t *testing.T) {
	p := validProfile()
	p.ActivePillars = nil
	if err := p.Validate(); err != ErrEmptyActivePillars {
		t.Fatalf("expected ErrEmptyActivePillars, got: %v", err)
	}
}

func TestValidateUnknownPillarType(t *testing.T) {
	p := validProfile()
	p.ActivePillars = []PillarType{"UNKNOWN_PILLAR"}
	if err := p.Validate(); err != ErrUnknownPillarType {
		t.Fatalf("expected ErrUnknownPillarType, got: %v", err)
	}
}

func TestValidateTaxAuthorityNameEmpty(t *testing.T) {
	p := validProfile()
	p.TaxAuthority.Name = ""
	if err := p.Validate(); err != ErrEmptyTaxAuthorityName {
		t.Fatalf("expected ErrEmptyTaxAuthorityName, got: %v", err)
	}
}

func TestValidateTaxIDRegexInvalid(t *testing.T) {
	p := validProfile()
	p.TaxAuthority.TaxIDRegex = "[invalid"
	if err := p.Validate(); err != ErrInvalidTaxIDRegex {
		t.Fatalf("expected ErrInvalidTaxIDRegex, got: %v", err)
	}
}

func TestValidateTaxIDRegexEmptyIsValid(t *testing.T) {
	p := validProfile()
	p.TaxAuthority.TaxIDRegex = ""
	if err := p.Validate(); err != nil {
		t.Fatalf("empty tax ID regex should be valid, got: %v", err)
	}
}

func TestValidateSubDivisionCodeEmpty(t *testing.T) {
	p := validProfile()
	p.SubDivisions = []SubDivisionProfile{{Code: "", Name: "Riyadh Region"}}
	if err := p.Validate(); err != ErrEmptySubDivisionCode {
		t.Fatalf("expected ErrEmptySubDivisionCode, got: %v", err)
	}
}

func TestValidateSubDivisionNameEmpty(t *testing.T) {
	p := validProfile()
	p.SubDivisions = []SubDivisionProfile{{Code: "RD", Name: ""}}
	if err := p.Validate(); err != ErrEmptySubDivisionName {
		t.Fatalf("expected ErrEmptySubDivisionName, got: %v", err)
	}
}

func TestValidateSubDivisionCodeNormalized(t *testing.T) {
	p := validProfile()
	p.SubDivisions = []SubDivisionProfile{{Code: "rd", Name: "Riyadh"}}
	if err := p.Validate(); err != nil {
		t.Fatalf("subdivision code should be normalized to uppercase, got: %v", err)
	}
	if p.SubDivisions[0].Code != "RD" {
		t.Fatalf("expected sub-division code 'RD', got: %q", p.SubDivisions[0].Code)
	}
}

func TestValidateAllLifecycleStates(t *testing.T) {
	states := []LifecycleState{LifecycleActive, LifecycleDraft, LifecycleArchived}
	for _, state := range states {
		p := validProfile()
		p.Lifecycle = state
		if err := p.Validate(); err != nil {
			t.Fatalf("lifecycle %q should be valid, got: %v", state, err)
		}
	}
}

func TestValidateAllPillarTypes(t *testing.T) {
	pillars := []PillarType{
		PillarSafetyRegulation, PillarTaxFiscal, PillarAccreditation,
		PillarCertificateMarking, PillarCurrencyTrade, PillarEnvironmentalESG,
		PillarLaborWorkforce,
	}
	for _, pillar := range pillars {
		p := validProfile()
		p.ActivePillars = []PillarType{pillar}
		if err := p.Validate(); err != nil {
			t.Fatalf("pillar %q should be valid, got: %v", pillar, err)
		}
	}
}

func TestCountryProfileDID(t *testing.T) {
	p := validProfile()
	expected := "did:integin:jurisdiction:SA"
	if p.DID() != expected {
		t.Fatalf("expected DID %q, got %q", expected, p.DID())
	}
}

func TestCountryProfileHasPillar(t *testing.T) {
	p := validProfile()
	if !p.HasPillar(PillarSafetyRegulation) {
		t.Fatal("expected HasPillar(SafetyRegulation) to be true")
	}
	if p.HasPillar("NONEXISTENT") {
		t.Fatal("expected HasPillar(NONEXISTENT) to be false")
	}
}

func TestLookupSubDivisionFound(t *testing.T) {
	p := validProfile()
	p.SubDivisions = []SubDivisionProfile{
		{Code: "RD", Name: "Riyadh"},
		{Code: "MC", Name: "Makkah"},
	}
	sub, ok := p.LookupSubDivision("rd")
	if !ok || sub.Name != "Riyadh" {
		t.Fatalf("expected to find Riyadh subdivision, got ok=%v name=%q", ok, sub.Name)
	}
}

func TestLookupSubDivisionNotFound(t *testing.T) {
	p := validProfile()
	_, ok := p.LookupSubDivision("XX")
	if ok {
		t.Fatal("expected LookupSubDivision to return false for unknown code")
	}
}

func TestLifecycleStateConstants(t *testing.T) {
	if LifecycleActive != "ACTIVE" {
		t.Fatalf("LifecycleActive should be 'ACTIVE', got %q", LifecycleActive)
	}
	if LifecycleDraft != "DRAFT" {
		t.Fatalf("LifecycleDraft should be 'DRAFT', got %q", LifecycleDraft)
	}
	if LifecycleArchived != "ARCHIVED" {
		t.Fatalf("LifecycleArchived should be 'ARCHIVED', got %q", LifecycleArchived)
	}
}

func TestPillarTypeConstants(t *testing.T) {
	expected := map[PillarType]string{
		PillarSafetyRegulation:   "SAFETY_REGULATION",
		PillarTaxFiscal:          "TAX_FISCAL",
		PillarAccreditation:      "ACCREDITATION",
		PillarCertificateMarking: "CERTIFICATE_MARKING",
		PillarCurrencyTrade:      "CURRENCY_TRADE",
		PillarEnvironmentalESG:   "ENVIRONMENTAL_ESG",
		PillarLaborWorkforce:     "LABOR_WORKFORCE",
	}
	for pillar, exp := range expected {
		if string(pillar) != exp {
			t.Fatalf("pillar constant mismatch: expected %q, got %q", exp, string(pillar))
		}
	}
}
