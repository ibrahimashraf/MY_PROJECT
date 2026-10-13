package template

import "testing"

func TestRecursiveTemplateCompositionCreatesFlatChecklist(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{Code: "HOOK", Version: 1, AssetType: "hook", Sections: []Section{{ID: "hook-checks", Items: []Item{{ID: "hook-visual", Prompt: "Inspect hook", ResponseType: "PASS_FAIL", Required: true}}}}}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(Definition{Code: "HOIST", Version: 1, AssetType: "hoist", Sections: []Section{{ID: "hoist-checks", Items: []Item{{ID: "hoist-brake", Prompt: "Test brake", ResponseType: "PASS_FAIL", Required: true}}, SubTemplate: "HOOK@1"}}}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(Definition{Code: "CRANE", Version: 1, AssetType: "overhead_crane", Sections: []Section{{ID: "crane-structure", Items: []Item{{ID: "crane-beam", Prompt: "Inspect beam", ResponseType: "PASS_FAIL", Required: true}}, SubTemplate: "HOIST@1"}}}); err != nil {
		t.Fatal(err)
	}
	root := AssetNode{ID: "crane-1", AssetType: "overhead_crane", Name: "Main crane", Children: []AssetNode{{ID: "hoist-1", AssetType: "hoist", Name: "Main hoist", Children: []AssetNode{{ID: "hook-1", AssetType: "hook", Name: "Main hook"}}}}}
	snapshot, err := registry.Resolve("CRANE", 1, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Items) != 3 {
		t.Fatalf("expected three flattened checklist items, got %d", len(snapshot.Items))
	}
	if snapshot.Items[0].AssetID != "crane-1" || snapshot.Items[1].AssetID != "hoist-1" || snapshot.Items[2].AssetID != "hook-1" {
		t.Fatalf("unexpected asset composition: %#v", snapshot.Items)
	}
	if snapshot.Items[2].SectionID != "hook-checks" {
		t.Fatalf("expected hook section, got %s", snapshot.Items[2].SectionID)
	}
}

func TestSnapshotRetainsHistoricalTemplateVersion(t *testing.T) {
	registry := NewRegistry()
	v1 := Definition{Code: "CRANE", Version: 1, AssetType: "overhead_crane", Sections: []Section{{ID: "checks", Items: []Item{{ID: "beam-v1", Prompt: "Inspect beam", ResponseType: "PASS_FAIL"}}}}}
	if err := registry.Register(v1); err != nil {
		t.Fatal(err)
	}
	root := AssetNode{ID: "crane-1", AssetType: "overhead_crane", Name: "Main crane"}
	snapshotV1, err := registry.Resolve("CRANE", 1, root)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(Definition{Code: "CRANE", Version: 2, AssetType: "overhead_crane", Sections: []Section{{ID: "checks", Items: []Item{{ID: "beam-v2", Prompt: "Inspect revised beam", ResponseType: "PASS_FAIL"}}}}}); err != nil {
		t.Fatal(err)
	}
	snapshotV2, err := registry.Resolve("CRANE", 2, root)
	if err != nil {
		t.Fatal(err)
	}
	if snapshotV1.Version != 1 || snapshotV1.Items[0].ID != "beam-v1" {
		t.Fatalf("v1 snapshot was rewritten: %#v", snapshotV1)
	}
	if snapshotV2.Version != 2 || snapshotV2.Items[0].ID != "beam-v2" {
		t.Fatalf("v2 snapshot incorrect: %#v", snapshotV2)
	}
	copied := snapshotV1.ItemsCopy()
	copied[0].ID = "mutated"
	if snapshotV1.Items[0].ID == "mutated" {
		t.Fatal("snapshot items were mutable through accessor")
	}
}

func TestExpressionEvaluation(t *testing.T) {
	fields := map[string]any{"pressure": 125.0, "result": "PASS", "enabled": true}
	cases := map[string]bool{
		"pressure > 100 && result == 'PASS'": true,
		"pressure < 100 || enabled == false": false,
		"(pressure >= 125) && enabled":       true,
		"result != 'FAIL'":                   true,
	}
	for expression, expected := range cases {
		actual, err := Evaluate(expression, fields)
		if err != nil {
			t.Fatalf("%s: %v", expression, err)
		}
		if actual != expected {
			t.Fatalf("%s: expected %v, got %v", expression, expected, actual)
		}
	}
}

func TestExpressionRejectsUnknownFieldAndInvalidSyntax(t *testing.T) {
	if _, err := Evaluate("missing == true", map[string]any{}); err == nil {
		t.Fatal("expected unknown field error")
	}
	if _, err := Evaluate("pressure >", map[string]any{"pressure": 1}); err == nil {
		t.Fatal("expected incomplete expression error")
	}
	if _, err := Evaluate("pressure", map[string]any{"pressure": 1}); err == nil {
		t.Fatal("expected non-boolean expression error")
	}
}
