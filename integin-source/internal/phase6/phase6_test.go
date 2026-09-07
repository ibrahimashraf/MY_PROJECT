package phase6

import (
	"testing"
	"time"
)

func TestRegulationAnalysisFindsAffectedTemplatesWithoutBlocking(t *testing.T) {
	change := RegulationChange{ID: "change-1", TenantID: "tenant-1", StandardCode: "ISO-1", OldVersion: "2020", NewVersion: "2026", ChangedClauses: []string{"7.1"}, Summary: "inspection interval changed"}
	impacts, err := AnalyzeRegulation(change, []TemplateReference{{ID: "template-1", TenantID: "tenant-1", StandardCodes: []string{"ISO-1"}, ClauseRefs: []string{"7.1", "8.2"}}, {ID: "template-2", TenantID: "tenant-2", StandardCodes: []string{"ISO-1"}}}, time.Now())
	if err != nil || len(impacts) != 1 {
		t.Fatalf("unexpected impacts: %#v %v", impacts, err)
	}
	if impacts[0].Advisory.Blocking || len(impacts[0].AffectedClauses) != 1 {
		t.Fatalf("regulation result must be advisory: %#v", impacts[0])
	}
}

func TestFieldMergePreservesNonConflictingChangesAndConflicts(t *testing.T) {
	base := map[string]any{"serial": "SN-1", "notes": "base", "status": "OPEN"}
	local := map[string]any{"serial": "SN-LOCAL", "notes": "local note", "status": "IN_REVIEW"}
	remote := map[string]any{"serial": "SN-2", "notes": "base", "status": "CLOSED"}
	merged := MergeFields(base, local, remote)
	if merged.Merged["notes"] != "local note" {
		t.Fatalf("local non-conflict was not merged: %#v", merged)
	}
	if len(merged.Conflicts) != 2 {
		t.Fatalf("expected two human conflicts: %#v", merged.Conflicts)
	}
}

func TestClientRequestLifecycleEnforcesTenant(t *testing.T) {
	request, err := NewClientRequest("request-1", "tenant-1", "client-1", "asset-1", "request annual inspection")
	if err != nil {
		t.Fatal(err)
	}
	if err := request.Review("tenant-2", "reviewer-1"); err == nil {
		t.Fatal("cross-tenant review should fail")
	}
	if err := request.Review("tenant-1", "reviewer-1"); err != nil {
		t.Fatal(err)
	}
	if err := request.Approve("tenant-1"); err != nil {
		t.Fatal(err)
	}
	if err := request.Schedule("tenant-1", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := request.Complete("tenant-1"); err != nil {
		t.Fatal(err)
	}
	if request.Status != Completed {
		t.Fatalf("unexpected request status: %s", request.Status)
	}
}
