package standardsync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCodexLoadAndIndex(t *testing.T) {
	// Test against the live extracted_codex.json catalog file
	realPath := filepath.Join("catalog", "extracted_codex.json")
	if _, err := os.Stat(realPath); err != nil {
		t.Skipf("skipping live catalog test: %v", err)
	}

	codex, idx, err := LoadCodex(realPath)
	if err != nil {
		t.Fatalf("LoadCodex failed on real catalog: %v", err)
	}

	if codex.TotalStandardsScanned != 263 {
		t.Errorf("expected 263 standards scanned, got %d", codex.TotalStandardsScanned)
	}
	if codex.TotalParametersExtracted != 349 {
		t.Errorf("expected 349 parameters, got %d", codex.TotalParametersExtracted)
	}
	if len(codex.Parameters) != 349 {
		t.Errorf("expected 349 parameters slice, got %d", len(codex.Parameters))
	}

	// Test SearchByParameterType
	tippingParams := idx.SearchByParameterType("STABILITY_TIPPING_PERCENT_MAX")
	if len(tippingParams) == 0 {
		t.Fatal("expected at least one STABILITY_TIPPING_PERCENT_MAX parameter")
	}
	if tippingParams[0].NumericValue != 85.0 {
		t.Errorf("expected ASME B30.6 tipping limit to be 85.0, got %f", tippingParams[0].NumericValue)
	}

	// Verify citation card creation and strict schema validation
	for _, p := range codex.Parameters[:10] {
		card := p.ToMetadataCard()
		if err := card.Validate(); err != nil {
			t.Errorf("card validation failed for %s (%s): %v", p.Doc, card.StandardDID, err)
		}
	}
}
