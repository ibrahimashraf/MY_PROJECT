package jurisdictions

import (
	"errors"
	"reflect"
	"testing"
)

func TestResolvePreemptionASMEvsOSHA(t *testing.T) {
	res, err := ResolvePreemption([]string{"ASME B30.5-2021", "OSHA 29 CFR 1926"}, "US")
	if err != nil {
		t.Fatalf("ResolvePreemption: %v", err)
	}
	if res.Winner != "OSHA 29 CFR 1926" {
		t.Errorf("national statutory OSHA must preempt advisory ASME, got winner %q", res.Winner)
	}
	if !reflect.DeepEqual(res.EffectiveOrder, []string{"OSHA 29 CFR 1926", "ASME B30.5-2021"}) {
		t.Errorf("unexpected effective order: %v", res.EffectiveOrder)
	}
	if res.JurisdictionISO2 != "US" {
		t.Errorf("jurisdiction not echoed: %q", res.JurisdictionISO2)
	}
	found := false
	for _, r := range res.AppliedRules {
		if r == RuleStatutoryOverridesAdvisory {
			found = true
		}
	}
	if !found {
		t.Errorf("expected STATUTORY_OVERRIDES_ADVISORY rule, got %v", res.AppliedRules)
	}
}

func TestResolvePreemptionISOvsAramcoAndAdnoc(t *testing.T) {
	res, err := ResolvePreemption([]string{"ISO 4309:2018", "ARAMCO SAE-J-011"}, "SA")
	if err != nil {
		t.Fatalf("ResolvePreemption: %v", err)
	}
	if res.Winner != "ARAMCO SAE-J-011" {
		t.Errorf("site jurisdiction ARAMCO must preempt international ISO, got %q", res.Winner)
	}
	if res.EffectiveOrder[0] != "ARAMCO SAE-J-011" || res.EffectiveOrder[1] != "ISO 4309:2018" {
		t.Errorf("unexpected order: %v", res.EffectiveOrder)
	}

	res, err = ResolvePreemption([]string{"ISO 4309:2018", "ADNOC CoP V3.0"}, "AE")
	if err != nil {
		t.Fatalf("ResolvePreemption: %v", err)
	}
	if res.Winner != "ADNOC COP V3.0" {
		t.Errorf("national statutory ADNOC must preempt advisory ISO, got %q", res.Winner)
	}
}

func TestResolvePreemptionFullPriority(t *testing.T) {
	res, err := ResolvePreemption([]string{"ISO 4309:2018", "IEC 61508", "ADNOC CoP", "OSHA 29 CFR 1926", "ARAMCO SAE-J-011"}, "SA")
	if err != nil {
		t.Fatalf("ResolvePreemption: %v", err)
	}
	if res.EffectiveOrder[0] != "ARAMCO SAE-J-011" {
		t.Errorf("winner must be site ARAMCO, got %q", res.EffectiveOrder[0])
	}
	if res.EffectiveOrder[1] != "ADNOC COP" {
		t.Errorf("ADNOC must sit above OSHA, got %v", res.EffectiveOrder)
	}
	if res.EffectiveOrder[len(res.EffectiveOrder)-1] != "ISO 4309:2018" {
		t.Errorf("advisory ISO must rank last, got %v", res.EffectiveOrder)
	}
}

func TestResolvePreemptionDeterministicAcrossInputOrder(t *testing.T) {
	orderA, err := ResolvePreemption([]string{"ISO 4309", "OSHA 29 CFR 1926", "ADNOC CoP"}, "AE")
	if err != nil {
		t.Fatalf("order A: %v", err)
	}
	orderB, err := ResolvePreemption([]string{"ADNOC CoP", "OSHA 29 CFR 1926", "ISO 4309"}, "AE")
	if err != nil {
		t.Fatalf("order B: %v", err)
	}
	if !reflect.DeepEqual(orderA.EffectiveOrder, orderB.EffectiveOrder) {
		t.Errorf("resolution must be order-independent: %v vs %v", orderA.EffectiveOrder, orderB.EffectiveOrder)
	}
	if !reflect.DeepEqual(orderA.AppliedRules, orderB.AppliedRules) {
		t.Errorf("rules must be order-independent: %v vs %v", orderA.AppliedRules, orderB.AppliedRules)
	}
}

func TestResolvePreemptionEdgeCases(t *testing.T) {
	res, err := ResolvePreemption([]string{"ISO 4309:2018"}, "DE")
	if err != nil {
		t.Fatalf("single standard: %v", err)
	}
	if res.Winner != "ISO 4309:2018" || len(res.EffectiveOrder) != 1 {
		t.Errorf("single standard must resolve to itself: %+v", res)
	}

	if _, err := ResolvePreemption(nil, "SA"); !errors.Is(err, ErrEmptyStandardSet) {
		t.Errorf("empty set must fail, got %v", err)
	}
	if _, err := ResolvePreemption([]string{"MYSTERY-STD-1"}, "SA"); !errors.Is(err, ErrUnknownStandardBody) {
		t.Errorf("unknown body must fail, got %v", err)
	}
	if _, err := ResolvePreemption([]string{"ISO 4309"}, "SAU"); err == nil {
		t.Errorf("3-letter jurisdiction must fail")
	}
}

func TestClassifyStandard(t *testing.T) {
	for std, wantLevel := range map[string]LegalHierarchyLevel{
		"ISO 4309:2018":    LevelInternational,
		"IEC 61508":        LevelInternational,
		"OSHA 29 CFR 1926": LevelNational,
		"ASME B30.5":       LevelNational,
		"SASO 2807":        LevelNational,
		"ADNOC CoP":        LevelNational,
		"ARAMCO SAE-J-011": LevelSiteJurisdiction,
	} {
		cls, err := ClassifyStandard(std)
		if err != nil {
			t.Fatalf("ClassifyStandard(%q): %v", std, err)
		}
		if cls.Level != wantLevel {
			t.Errorf("ClassifyStandard(%q) level = %v, want %v", std, cls.Level, wantLevel)
		}
	}
	if _, err := ClassifyStandard(""); !errors.Is(err, ErrUnknownStandardBody) {
		t.Errorf("empty standard must fail, got %v", err)
	}
}
