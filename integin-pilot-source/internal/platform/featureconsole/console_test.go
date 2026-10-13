package featureconsole

import "testing"

func TestFeatureConsoleModuleMetadataAndUpdatePath(t *testing.T) {
	console := New()
	if err := console.Register(Module{Name: "inspection", Version: "1.0.0", EntryPoint: "internal/domain/inspection", Subscriptions: []string{"InspectionStarted"}, Emissions: []string{"InspectionClosed"}, ConfigSchema: map[string]string{"review_timeout": "duration"}, Environment: "LIVE"}); err != nil {
		t.Fatal(err)
	}
	installed := console.Installed()
	if len(installed) != 1 || installed[0].Version != "1.0.0" {
		t.Fatalf("unexpected installed modules: %#v", installed)
	}
	installed[0].ConfigSchema["review_timeout"] = "changed"
	if console.Installed()[0].ConfigSchema["review_timeout"] != "duration" {
		t.Fatal("module metadata was aliased")
	}
	update, err := console.UpdatePath("inspection", "1.0.0", "1.1.0")
	if err != nil || len(update.Path) != 4 {
		t.Fatalf("unexpected update path: %#v %v", update, err)
	}
}
