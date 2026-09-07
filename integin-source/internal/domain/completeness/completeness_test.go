package completeness

import "testing"

func TestCheckFindsMissingMandatoryFields(t *testing.T) {
	requirements := []Requirement{{Field: "asset_name", Mandatory: true}, {Field: "serial_number", Mandatory: true}, {Field: "notes", Mandatory: false}, {Field: "metadata.manufacturer", Mandatory: true}}
	data := map[string]any{"asset_name": "Crane 1", "serial_number": "", "metadata": map[string]any{"manufacturer": "Acme"}}
	result := Check(requirements, data)
	if result.Complete {
		t.Fatal("expected incomplete result")
	}
	if len(result.Missing) != 1 || result.Missing[0] != "serial_number" {
		t.Fatalf("unexpected missing fields: %#v", result.Missing)
	}
}

func TestAdvisorySuggestionsNeverBlockCompleteness(t *testing.T) {
	requirements := []Requirement{{Field: "asset_name", Mandatory: true}}
	result := Check(requirements, map[string]any{"asset_name": "Crane 1"})
	enriched := AddAdvisorySuggestions(result, []Suggestion{{Field: "manufacturer", Reason: "commonly recorded for this asset type", Blocking: true}})
	if !enriched.Complete || len(enriched.Missing) != 0 {
		t.Fatal("advisory suggestions changed deterministic completeness")
	}
	if len(enriched.Suggestions) != 1 || enriched.Suggestions[0].Blocking {
		t.Fatal("advisory suggestion must be non-blocking")
	}
}

func TestCheckDoesNotMutateInput(t *testing.T) {
	data := map[string]any{"asset_name": "Crane 1"}
	before := data["asset_name"]
	_ = Check([]Requirement{{Field: "asset_name", Mandatory: true}, {Field: "serial_number", Mandatory: true}}, data)
	if data["asset_name"] != before || len(data) != 1 {
		t.Fatal("completeness check mutated input")
	}
}
