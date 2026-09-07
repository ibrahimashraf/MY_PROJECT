package jurisdictions

import (
	"sync"
	"testing"
)

func TestNewRegistryValid(t *testing.T) {
	r, err := NewRegistry(TopJurisdictions())
	if err != nil {
		t.Fatalf("NewRegistry should succeed, got: %v", err)
	}
	if r.Count() != 8 {
		t.Fatalf("expected 8 jurisdictions, got %d", r.Count())
	}
}

func TestRegistryByISO2Found(t *testing.T) {
	r, _ := NewRegistry(TopJurisdictions())
	p, ok := r.ByISO2("SA")
	if !ok || p.CommonName != "Saudi Arabia" {
		t.Fatalf("expected Saudi Arabia by ISO2 SA, got ok=%v name=%q", ok, p.CommonName)
	}
}

func TestRegistryByISO2CaseInsensitive(t *testing.T) {
	r, _ := NewRegistry(TopJurisdictions())
	p, ok := r.ByISO2("us")
	if !ok || p.CommonName != "United States" {
		t.Fatalf("expected United States by lowercase 'us', got ok=%v name=%q", ok, p.CommonName)
	}
}

func TestRegistryByISO2NotFound(t *testing.T) {
	r, _ := NewRegistry(TopJurisdictions())
	_, ok := r.ByISO2("ZZ")
	if ok {
		t.Fatal("expected ByISO2 to return false for unknown code")
	}
}

func TestRegistryByISO3Found(t *testing.T) {
	r, _ := NewRegistry(TopJurisdictions())
	p, ok := r.ByISO3("GBR")
	if !ok || p.CommonName != "United Kingdom" {
		t.Fatalf("expected United Kingdom by ISO3 GBR, got ok=%v name=%q", ok, p.CommonName)
	}
}

func TestRegistryByISO3NotFound(t *testing.T) {
	r, _ := NewRegistry(TopJurisdictions())
	_, ok := r.ByISO3("ZZZ")
	if ok {
		t.Fatal("expected ByISO3 to return false for unknown code")
	}
}

func TestRegistryByCurrencyUSD(t *testing.T) {
	r, _ := NewRegistry(TopJurisdictions())
	profiles := r.ByCurrency("USD")
	if len(profiles) != 1 {
		t.Fatalf("expected 1 USD jurisdiction, got %d", len(profiles))
	}
	if profiles[0].ISO2 != "US" {
		t.Fatalf("expected US for USD currency, got %q", profiles[0].ISO2)
	}
}

func TestRegistryByCurrencyEUR(t *testing.T) {
	r, _ := NewRegistry(TopJurisdictions())
	profiles := r.ByCurrency("EUR")
	if len(profiles) != 1 {
		t.Fatalf("expected 1 EUR jurisdiction, got %d", len(profiles))
	}
	if profiles[0].ISO2 != "DE" {
		t.Fatalf("expected DE for EUR currency, got %q", profiles[0].ISO2)
	}
}

func TestRegistryByPillarSafetyRegulation(t *testing.T) {
	r, _ := NewRegistry(TopJurisdictions())
	profiles := r.ByPillar(PillarSafetyRegulation)
	if len(profiles) != 8 {
		t.Fatalf("expected 8 jurisdictions with SAFETY_REGULATION pillar, got %d", len(profiles))
	}
}

func TestRegistryByPillarTaxFiscal(t *testing.T) {
	r, _ := NewRegistry(TopJurisdictions())
	profiles := r.ByPillar(PillarTaxFiscal)
	if len(profiles) != 7 {
		t.Fatalf("expected 7 jurisdictions with TAX_FISCAL pillar (US has no VAT), got %d", len(profiles))
	}
}

func TestRegistryByPillarUnknown(t *testing.T) {
	r, _ := NewRegistry(TopJurisdictions())
	profiles := r.ByPillar("UNKNOWN")
	if len(profiles) != 0 {
		t.Fatalf("expected 0 for unknown pillar, got %d", len(profiles))
	}
}

func TestRegistryAllSorted(t *testing.T) {
	r, _ := NewRegistry(TopJurisdictions())
	all := r.All()
	if len(all) != 8 {
		t.Fatalf("expected 8, got %d", len(all))
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].ISO2 >= all[i].ISO2 {
			t.Fatalf("All() not sorted: %s >= %s at index %d", all[i-1].ISO2, all[i].ISO2, i)
		}
	}
}

func TestRegistryAllReturnsCopy(t *testing.T) {
	r, _ := NewRegistry(TopJurisdictions())
	all1 := r.All()
	all1[0].ISO2 = "HACKED"
	all2 := r.All()
	if all2[0].ISO2 == "HACKED" {
		t.Fatal("All() should return a copy, not a reference to internal state")
	}
}

func TestRegistryByCurrencyReturnsCopy(t *testing.T) {
	r, _ := NewRegistry(TopJurisdictions())
	profiles := r.ByCurrency("USD")
	profiles[0].ISO2 = "HACKED"
	original, _ := r.ByISO2("US")
	if original.ISO2 == "HACKED" {
		t.Fatal("ByCurrency() should return a copy, not a reference to internal state")
	}
}

func TestNewRegistryDuplicateRejected(t *testing.T) {
	dupes := []CountryProfile{
		{
			ISO2: "SA", ISO3: "SAU", NumericCode: "682", CommonName: "Saudi Arabia",
			PrimaryLanguage: "ar", CurrencyCode: "SAR", Region: "Asia",
			Lifecycle:     LifecycleActive,
			TaxAuthority:  TaxAuthority{Name: "ZATCA"},
			ActivePillars: []PillarType{PillarSafetyRegulation},
		},
		{
			ISO2: "SA", ISO3: "XXX", NumericCode: "000", CommonName: "Duplicate",
			PrimaryLanguage: "en", CurrencyCode: "USD", Region: "Test",
			Lifecycle:     LifecycleActive,
			TaxAuthority:  TaxAuthority{Name: "Test"},
			ActivePillars: []PillarType{PillarSafetyRegulation},
		},
	}
	_, err := NewRegistry(dupes)
	if err != ErrDuplicateJurisdiction {
		t.Fatalf("expected ErrDuplicateJurisdiction, got: %v", err)
	}
}

func TestNewRegistryRejectsInactiveLifecycle(t *testing.T) {
	profiles := []CountryProfile{
		{
			ISO2: "XX", ISO3: "XXX", NumericCode: "000", CommonName: "Test",
			PrimaryLanguage: "en", CurrencyCode: "USD", Region: "Test",
			Lifecycle:     LifecycleDraft,
			TaxAuthority:  TaxAuthority{Name: "Test"},
			ActivePillars: []PillarType{PillarSafetyRegulation},
		},
	}
	_, err := NewRegistry(profiles)
	if err != ErrJurisdictionNotActive {
		t.Fatalf("expected ErrJurisdictionNotActive, got: %v", err)
	}
}

func TestNewRegistryRejectsInvalidProfile(t *testing.T) {
	profiles := []CountryProfile{
		{
			ISO2: "", ISO3: "XXX", NumericCode: "000", CommonName: "Test",
			PrimaryLanguage: "en", CurrencyCode: "USD", Region: "Test",
			Lifecycle:     LifecycleActive,
			TaxAuthority:  TaxAuthority{Name: "Test"},
			ActivePillars: []PillarType{PillarSafetyRegulation},
		},
	}
	_, err := NewRegistry(profiles)
	if err != ErrEmptyISO2 {
		t.Fatalf("expected ErrEmptyISO2, got: %v", err)
	}
}

func TestRegistryConcurrency(t *testing.T) {
	r, _ := NewRegistry(TopJurisdictions())
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.ByISO2("SA")
			r.ByISO3("USA")
			r.ByCurrency("GBP")
			r.ByPillar(PillarAccreditation)
			r.All()
			r.Count()
		}()
	}
	wg.Wait()
}

func TestRegistryLookupAllCountries(t *testing.T) {
	r, _ := NewRegistry(TopJurisdictions())
	codes := []string{"SA", "AE", "US", "GB", "DE", "SG", "AU", "NO"}
	for _, code := range codes {
		p, ok := r.ByISO2(code)
		if !ok {
			t.Fatalf("expected to find jurisdiction %q", code)
		}
		if err := p.Validate(); err != nil {
			t.Fatalf("jurisdiction %q failed validation: %v", code, err)
		}
	}
}
