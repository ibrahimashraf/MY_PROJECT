package jurisdictions

import (
	"testing"

	"integin/pkg/rulesengine"
)

// TestDesignBasisForISO2 locks the seeded ASD/LRFD switch against the
// registered jurisdictions plus unknown-code fallback.
func TestDesignBasisForISO2(t *testing.T) {
	cases := []struct {
		iso2 string
		want rulesengine.DesignBasis
	}{
		// US-code practice: Allowable Stress Design.
		{"US", rulesengine.DesignBasisASD},
		{"SA", rulesengine.DesignBasisASD},
		{"AE", rulesengine.DesignBasisASD},
		// Eurocode / limit-state practice: Load & Resistance Factor Design.
		{"GB", rulesengine.DesignBasisLRFD},
		{"DE", rulesengine.DesignBasisLRFD},
		{"SG", rulesengine.DesignBasisLRFD},
		{"AU", rulesengine.DesignBasisLRFD},
		{"NO", rulesengine.DesignBasisLRFD},
		// Case-insensitive.
		{"us", rulesengine.DesignBasisASD},
		{"gb", rulesengine.DesignBasisLRFD},
	}
	for _, c := range cases {
		if got := DesignBasisForISO2(c.iso2); got != c.want {
			t.Errorf("DesignBasisForISO2(%q) = %q, want %q", c.iso2, got, c.want)
		}
	}

	// Unknown or empty codes resolve to the conservative default (ASD).
	for _, unknown := range []string{"ZZ", "CN", "", "  "} {
		if got := DesignBasisForISO2(unknown); got != rulesengine.DesignBasisASD {
			t.Errorf("DesignBasisForISO2(%q) = %q, want default ASD", unknown, got)
		}
	}
}

// TestDesignBasisForISO2SwitchCards proves the returned bases resolve to
// distinct compilable CEL rule cards in the rules engine.
func TestDesignBasisForISO2SwitchCards(t *testing.T) {
	e, err := rulesengine.NewEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	asd := DesignBasisForISO2("US")
	lrfd := DesignBasisForISO2("DE")
	if asd == lrfd {
		t.Fatal("expected distinct design bases across the switch")
	}
	for _, basis := range []rulesengine.DesignBasis{asd, lrfd} {
		ruleID, expr, vars := rulesengine.DesignGate(basis)
		if _, err := e.Compile(ruleID, expr, vars); err != nil {
			t.Fatalf("design card %q must compile: %v", ruleID, err)
		}
	}
}
