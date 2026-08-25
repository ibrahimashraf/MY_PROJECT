package workorder

import (
	"errors"
	"testing"
)

func validReassignCommand(order WorkOrder) ReassignScopeCommand {
	return ReassignScopeCommand{
		Actor: validActor(), Operation: validOperation(), WorkOrderID: order.ID, ScopeItemIDs: []string{"scope-one"},
		Previous:    Assignment{ID: "previous", TenantID: order.TenantID, OrganizationID: order.OrganizationID, WorkOrderID: order.ID, InspectorID: "inspector-one", ScopeItemIDs: []string{"scope-one"}, State: AssignmentActive, Revision: 1, EffectiveFrom: testTime()},
		Replacement: Assignment{ID: "replacement", TenantID: order.TenantID, OrganizationID: order.OrganizationID, WorkOrderID: order.ID, InspectorID: "inspector-two", ScopeItemIDs: []string{"scope-one"}, State: AssignmentActive, Revision: 1, EffectiveFrom: testTime()},
	}
}

func TestReassignScopeCommandRejectsOrganizationMismatch(t *testing.T) {
	order := validWorkOrder()
	command := validReassignCommand(order)
	command.Actor.OrganizationID = "org-other"

	if err := ValidateReassignScopeCommand(command, order); !errors.Is(err, ErrInvalidAssignment) {
		t.Fatalf("expected organization rejection, got %v", err)
	}
}

func TestReassignScopeCommandRejectsDivergentReplacementScope(t *testing.T) {
	order := validWorkOrder()
	command := validReassignCommand(order)
	command.Replacement.ScopeItemIDs = []string{"scope-two"}

	if err := ValidateReassignScopeCommand(command, order); !errors.Is(err, ErrInvalidAssignment) {
		t.Fatalf("expected divergent scope rejection, got %v", err)
	}
}
