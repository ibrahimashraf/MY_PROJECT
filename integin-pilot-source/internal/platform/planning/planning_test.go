package planning

import "testing"

func TestWorkOrderAssignmentAndLifecycle(t *testing.T) {
	workOrder, err := NewWorkOrder("wo-1", "tenant-1", "org-1", "asset-1", "site-1", []string{"lifting.inspection"})
	if err != nil {
		t.Fatal(err)
	}
	if err := workOrder.Plan(); err != nil {
		t.Fatal(err)
	}
	user := User{ID: "user-1", TenantID: "tenant-1", OrganizationID: "org-1", Active: true, Competencies: map[string]bool{"lifting.inspection": true}, Scopes: map[string]bool{"site-1": true}}
	if err := workOrder.Assign(user); err != nil {
		t.Fatal(err)
	}
	if err := workOrder.Start(); err != nil {
		t.Fatal(err)
	}
	if err := workOrder.Complete(); err != nil {
		t.Fatal(err)
	}
	if workOrder.State != Completed || workOrder.AssignedUserID != "user-1" {
		t.Fatalf("unexpected work order: %#v", workOrder)
	}
}

func TestWorkOrderRejectsInvalidAssignment(t *testing.T) {
	workOrder, err := NewWorkOrder("wo-1", "tenant-1", "org-1", "asset-1", "site-1", []string{"lifting.inspection"})
	if err != nil {
		t.Fatal(err)
	}
	if err := workOrder.Plan(); err != nil {
		t.Fatal(err)
	}
	inactive := User{ID: "user-1", TenantID: "tenant-1", OrganizationID: "org-1", Active: false, Competencies: map[string]bool{"lifting.inspection": true}, Scopes: map[string]bool{"site-1": true}}
	if err := workOrder.Assign(inactive); err == nil {
		t.Fatal("inactive user should be rejected")
	}
	wrongScope := inactive
	wrongScope.Active = true
	wrongScope.Scopes = map[string]bool{"site-2": true}
	if err := workOrder.Assign(wrongScope); err == nil {
		t.Fatal("wrong scope should be rejected")
	}
	missingCompetency := wrongScope
	missingCompetency.Scopes = map[string]bool{"site-1": true}
	missingCompetency.Competencies = map[string]bool{}
	if err := workOrder.Assign(missingCompetency); err == nil {
		t.Fatal("missing competency should be rejected")
	}
}
