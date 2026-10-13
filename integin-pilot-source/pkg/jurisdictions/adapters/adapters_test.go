package adapters

import (
	"testing"

	"integin/pkg/jurisdictions"
)

func TestSafetyAdapterPillar(t *testing.T) {
	a := SafetyRegulationAdapter{}
	if a.Pillar() != jurisdictions.PillarSafetyRegulation {
		t.Fatalf("expected SAFETY_REGULATION pillar, got %q", a.Pillar())
	}
}

func TestSafetyAdapterPass(t *testing.T) {
	profile := jurisdictions.CountryProfile{
		ISO2: "SA", ISO3: "SAU", CommonName: "Saudi Arabia",
		PrimaryLanguage: "ar", CurrencyCode: "SAR", Region: "Asia",
		Lifecycle:    jurisdictions.LifecycleActive,
		TaxAuthority: jurisdictions.TaxAuthority{Name: "ZATCA"},
		SafetyRegulators: []jurisdictions.SafetyRegulator{
			{Name: "SCE"},
			{Name: "SASO"},
		},
		ActivePillars: []jurisdictions.PillarType{jurisdictions.PillarSafetyRegulation},
	}
	a := SafetyRegulationAdapter{}
	result, err := a.Check(&profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != StatusPass {
		t.Fatalf("expected PASS, got %v", result.Status)
	}
	if result.HasBlocking() {
		t.Fatal("expected no blocking findings")
	}
	if len(result.Findings) != 2 {
		t.Fatalf("expected 2 regulator findings, got %d", len(result.Findings))
	}
}

func TestSafetyAdapterNoRegulatorsBlocking(t *testing.T) {
	profile := jurisdictions.CountryProfile{
		ISO2: "XX", ISO3: "XXX", CommonName: "Test",
		PrimaryLanguage: "en", CurrencyCode: "USD", Region: "Test",
		Lifecycle:     jurisdictions.LifecycleActive,
		TaxAuthority:  jurisdictions.TaxAuthority{Name: "Test"},
		ActivePillars: []jurisdictions.PillarType{jurisdictions.PillarSafetyRegulation},
	}
	a := SafetyRegulationAdapter{}
	result, err := a.Check(&profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasBlocking() {
		t.Fatal("expected blocking finding when no safety regulators exist")
	}
}

func TestSafetyAdapterEmptyRegulatorName(t *testing.T) {
	profile := jurisdictions.CountryProfile{
		ISO2: "XX", ISO3: "XXX", CommonName: "Test",
		PrimaryLanguage: "en", CurrencyCode: "USD", Region: "Test",
		Lifecycle:    jurisdictions.LifecycleActive,
		TaxAuthority: jurisdictions.TaxAuthority{Name: "Test"},
		SafetyRegulators: []jurisdictions.SafetyRegulator{
			{Name: ""},
		},
		ActivePillars: []jurisdictions.PillarType{jurisdictions.PillarSafetyRegulation},
	}
	a := SafetyRegulationAdapter{}
	result, err := a.Check(&profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasFailures() {
		t.Fatal("expected FAIL finding for empty regulator name")
	}
	if result.Status != StatusFail {
		t.Fatalf("expected FAIL status, got %v", result.Status)
	}
}

func TestSafetyAdapterUnsupportedPillar(t *testing.T) {
	// SAFETY adapter requires the profile to declare SAFETY_REGULATION.
	profile := jurisdictions.CountryProfile{
		ISO2: "XX", ISO3: "XXX", CommonName: "Test",
		PrimaryLanguage: "en", CurrencyCode: "USD", Region: "Test",
		Lifecycle:     jurisdictions.LifecycleActive,
		TaxAuthority:  jurisdictions.TaxAuthority{Name: "Test"},
		ActivePillars: []jurisdictions.PillarType{jurisdictions.PillarTaxFiscal},
	}
	a := SafetyRegulationAdapter{}
	_, err := a.Check(&profile)
	if err == nil {
		t.Fatal("expected error for unsupported pillar")
	}
}

func TestAdapterNilProfile(t *testing.T) {
	a := SafetyRegulationAdapter{}
	_, err := a.Check(nil)
	if err != ErrNilProfile {
		t.Fatalf("expected ErrNilProfile, got: %v", err)
	}
}

func TestTaxAdapterSAPass(t *testing.T) {
	profile := jurisdictions.CountryProfile{
		ISO2: "SA", ISO3: "SAU", CommonName: "Saudi Arabia",
		PrimaryLanguage: "ar", CurrencyCode: "SAR", Region: "Asia",
		Lifecycle: jurisdictions.LifecycleActive,
		TaxAuthority: jurisdictions.TaxAuthority{
			Name: "ZATCA", TaxIDLabel: "VAT Number",
			TaxIDRegex: `^3\d{14}$`, CommercialReg: "CR",
		},
		ActivePillars: []jurisdictions.PillarType{jurisdictions.PillarTaxFiscal},
	}
	a := TaxFiscalAdapter{}
	result, err := a.Check(&profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasFailures() {
		t.Fatal("expected no failures for valid SA profile")
	}
	for _, f := range result.Findings {
		if f.Status == StatusWarning {
			t.Fatalf("expected no warnings, got warning on field %q", f.Field)
		}
	}
}

func TestTaxAdapterNoAuthorityBlocking(t *testing.T) {
	profile := jurisdictions.CountryProfile{
		ISO2: "XX", ISO3: "XXX", CommonName: "Test",
		PrimaryLanguage: "en", CurrencyCode: "USD", Region: "Test",
		Lifecycle:     jurisdictions.LifecycleActive,
		TaxAuthority:  jurisdictions.TaxAuthority{},
		ActivePillars: []jurisdictions.PillarType{jurisdictions.PillarTaxFiscal},
	}
	a := TaxFiscalAdapter{}
	result, err := a.Check(&profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasBlocking() {
		t.Fatal("expected blocking finding for missing tax authority")
	}
}

func TestTaxAdapterInvalidRegex(t *testing.T) {
	profile := jurisdictions.CountryProfile{
		ISO2: "XX", ISO3: "XXX", CommonName: "Test",
		PrimaryLanguage: "en", CurrencyCode: "USD", Region: "Test",
		Lifecycle: jurisdictions.LifecycleActive,
		TaxAuthority: jurisdictions.TaxAuthority{
			Name: "IRS", TaxIDRegex: "[invalid",
		},
		ActivePillars: []jurisdictions.PillarType{jurisdictions.PillarTaxFiscal},
	}
	a := TaxFiscalAdapter{}
	result, err := a.Check(&profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != StatusFail {
		t.Fatalf("expected FAIL, got %v", result.Status)
	}
}

func TestTaxAdapterMismatchedTestIDWarns(t *testing.T) {
	profile := jurisdictions.CountryProfile{
		ISO2: "US", ISO3: "USA", CommonName: "United States",
		PrimaryLanguage: "en", CurrencyCode: "USD", Region: "Americas",
		Lifecycle: jurisdictions.LifecycleActive,
		TaxAuthority: jurisdictions.TaxAuthority{
			Name: "IRS", TaxIDLabel: "EIN", TaxIDRegex: `^\d{11}$`, CommercialReg: "State",
		},
		ActivePillars: []jurisdictions.PillarType{jurisdictions.PillarTaxFiscal},
	}
	a := TaxFiscalAdapter{}
	result, err := a.Check(&profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// US EIN "12-3456789" (10 chars incl dash) does not match a 11-digit pattern.
	sawWarning := false
	for _, f := range result.Findings {
		if f.Status == StatusWarning {
			sawWarning = true
			break
		}
	}
	if !sawWarning {
		t.Fatal("expected warning for test ID not matching pattern")
	}
}

func TestAccreditationAdapter(t *testing.T) {
	profile := jurisdictions.CountryProfile{
		ISO2: "SA", ISO3: "SAU", CommonName: "Saudi Arabia",
		PrimaryLanguage: "ar", CurrencyCode: "SAR", Region: "Asia",
		Lifecycle:        jurisdictions.LifecycleActive,
		TaxAuthority:     jurisdictions.TaxAuthority{Name: "ZATCA"},
		SafetyRegulators: []jurisdictions.SafetyRegulator{{Name: "SCE"}},
		ActivePillars:    []jurisdictions.PillarType{jurisdictions.PillarAccreditation},
	}
	a := AccreditationAdapter{}
	result, err := a.Check(&profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasFailures() {
		t.Fatal("expected no failures for profile with regulators")
	}
}

func TestCertificateMarkingAdapter(t *testing.T) {
	profile := jurisdictions.CountryProfile{
		ISO2: "GB", ISO3: "GBR", CommonName: "UK",
		PrimaryLanguage: "en", CurrencyCode: "GBP", Region: "Europe",
		Lifecycle:        jurisdictions.LifecycleActive,
		TaxAuthority:     jurisdictions.TaxAuthority{Name: "HMRC"},
		SafetyRegulators: []jurisdictions.SafetyRegulator{{Name: "HSE"}},
		ActivePillars:    []jurisdictions.PillarType{jurisdictions.PillarCertificateMarking},
	}
	a := CertificateMarkingAdapter{}
	result, err := a.Check(&profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasFailures() {
		t.Fatal("expected no failures for valid profile")
	}
}

func TestCurrencyTradeAdapterUppercase(t *testing.T) {
	profile := jurisdictions.CountryProfile{
		ISO2: "AU", ISO3: "AUS", CommonName: "Australia",
		PrimaryLanguage: "en", CurrencyCode: "AUD", Region: "Oceania",
		Lifecycle:     jurisdictions.LifecycleActive,
		TaxAuthority:  jurisdictions.TaxAuthority{Name: "ATO"},
		ActivePillars: []jurisdictions.PillarType{jurisdictions.PillarCurrencyTrade},
	}
	a := CurrencyTradeAdapter{}
	result, err := a.Check(&profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasFailures() {
		t.Fatal("expected no failures for AUD")
	}
}

func TestCurrencyTradeAdapterInvalidCode(t *testing.T) {
	profile := jurisdictions.CountryProfile{
		ISO2: "XX", ISO3: "XXX", CommonName: "Test",
		PrimaryLanguage: "en", CurrencyCode: "XX", Region: "Test",
		Lifecycle:     jurisdictions.LifecycleActive,
		TaxAuthority:  jurisdictions.TaxAuthority{Name: "Test"},
		ActivePillars: []jurisdictions.PillarType{jurisdictions.PillarCurrencyTrade},
	}
	a := CurrencyTradeAdapter{}
	result, err := a.Check(&profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != StatusFail {
		t.Fatalf("expected FAIL for 2-char currency, got %v", result.Status)
	}
}

func TestCheckAllSeedJurisdictions(t *testing.T) {
	registry, err := jurisdictions.NewRegistry(jurisdictions.TopJurisdictions())
	if err != nil {
		t.Fatalf("registry build failed: %v", err)
	}
	for _, profile := range registry.All() {
		results, err := CheckAll(profile)
		if err != nil {
			t.Fatalf("CheckAll failed for %s: %v", profile.ISO2, err)
		}
		if len(results) == 0 {
			t.Fatalf("expected results for %s", profile.ISO2)
		}
		for _, r := range results {
			if r.Status != StatusPass {
				t.Fatalf("expected PASS for %s pillar %s, got %v: %v",
					profile.ISO2, r.Pillar, r.Status, r.Findings)
			}
		}
	}
}

func TestCheckSpecificPillar(t *testing.T) {
	registry, _ := jurisdictions.NewRegistry(jurisdictions.TopJurisdictions())
	profile, _ := registry.ByISO2("DE")
	result, err := Check(profile, jurisdictions.PillarSafetyRegulation)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Pillar != jurisdictions.PillarSafetyRegulation {
		t.Fatalf("expected SAFETY_REGULATION, got %q", result.Pillar)
	}
}

func TestCheckUnknownPillar(t *testing.T) {
	registry, _ := jurisdictions.NewRegistry(jurisdictions.TopJurisdictions())
	profile, _ := registry.ByISO2("DE")
	_, err := Check(profile, "NONEXISTENT")
	if err != ErrUnsupportedPillar {
		t.Fatalf("expected ErrUnsupportedPillar, got: %v", err)
	}
}

func TestCheckNilProfile(t *testing.T) {
	_, err := Check(nil, jurisdictions.PillarSafetyRegulation)
	if err != ErrNilProfile {
		t.Fatalf("expected ErrNilProfile, got: %v", err)
	}
}

func TestUnsupportedPillarError(t *testing.T) {
	err := UnsupportedPillarError("FAKE")
	if err.Error() != "adapter not registered for pillar FAKE" {
		t.Fatalf("unexpected error message: %q", err.Error())
	}
}

func TestAllAdaptersRegisteredOncePerPillar(t *testing.T) {
	seen := map[jurisdictions.PillarType]bool{}
	for _, a := range allAdapters {
		p := a.Pillar()
		if seen[p] {
			t.Fatalf("pillar %s registered more than once", p)
		}
		seen[p] = true
	}
	if len(seen) != 7 {
		t.Fatalf("expected exactly 7 registered pillars, got %d", len(seen))
	}
}
