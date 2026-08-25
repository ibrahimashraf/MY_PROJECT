package workorder

import (
	"errors"
	"testing"
	"time"
)

func validActor() ActorContext {
	return ActorContext{TenantID: "tenant-a", OrganizationID: "org-a", ActorID: "user-1", Role: "manager"}
}

func validOperation() OperationMeta {
	return OperationMeta{OperationID: "op-1", IdempotencyKey: "idem-1", ExpectedRevision: 1}
}

func TestCreateRequestCommandRequiresTenantMatch(t *testing.T) {
	order := validWorkOrder()
	command := CreateRequestCommand{Actor: validActor(), Operation: validOperation(), WorkOrder: order}
	if err := ValidateCreateRequestCommand(command); err != nil {
		t.Fatalf("valid create command rejected: %v", err)
	}

	command.Actor.TenantID = "tenant-b"
	if err := ValidateCreateRequestCommand(command); !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("expected tenant rejection, got %v", err)
	}
}

func TestAssignScopeCommandRejectsCrossTenantAndWrongOrder(t *testing.T) {
	order := validWorkOrder()
	assignment := Assignment{ID: "assignment-1", TenantID: order.TenantID, OrganizationID: order.OrganizationID, WorkOrderID: order.ID, InspectorID: "inspector-1", ScopeItemIDs: []string{"scope-1"}, State: AssignmentActive, Revision: 1, EffectiveFrom: testTime()}
	command := AssignScopeCommand{Actor: validActor(), Operation: validOperation(), WorkOrderID: order.ID, Assignment: assignment}
	if err := ValidateAssignScopeCommand(command, order); err != nil {
		t.Fatalf("valid assignment command rejected: %v", err)
	}

	command.WorkOrderID = "wo-other"
	if err := ValidateAssignScopeCommand(command, order); !errors.Is(err, ErrInvalidAssignment) {
		t.Fatalf("expected work-order rejection, got %v", err)
	}
}

func TestTransitionCommandRequiresCurrentRevisionState(t *testing.T) {
	order := validWorkOrder()
	command := TransitionExecutionCommand{Actor: validActor(), Operation: validOperation(), WorkOrderID: order.ID, From: ExecutionReady, To: ExecutionAssigned}
	if err := ValidateTransitionExecutionCommand(command, order); err != nil {
		t.Fatalf("valid transition rejected: %v", err)
	}

	command.From = ExecutionInProgress
	if err := ValidateTransitionExecutionCommand(command, order); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected stale-state rejection, got %v", err)
	}
}

func TestTransitionCommandRejectsOrganizationMismatch(t *testing.T) {
	order := validWorkOrder()
	command := TransitionExecutionCommand{Actor: validActor(), Operation: validOperation(), WorkOrderID: order.ID, From: ExecutionReady, To: ExecutionAssigned}
	command.Actor.OrganizationID = "org-other"
	if err := ValidateTransitionExecutionCommand(command, order); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected organization rejection, got %v", err)
	}
}

func TestPartialSubmissionRequiresScopedRecords(t *testing.T) {
	order := validWorkOrder()
	order.ExecutionState = ExecutionInProgress
	command := SubmitPartialCommand{Actor: validActor(), Operation: validOperation(), WorkOrderID: order.ID, AssignmentID: "assignment-1", InspectionIDs: []string{"inspection-1"}}
	if err := ValidateSubmitPartialCommand(command, order); err != nil {
		t.Fatalf("valid partial submission rejected: %v", err)
	}

	command.InspectionIDs = nil
	if err := ValidateSubmitPartialCommand(command, order); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("expected empty inspection rejection, got %v", err)
	}
}

func TestProvisionalReconciliationRequiresCanonicalIDOnlyForMatchOrCreate(t *testing.T) {
	command := ReconcileProvisionalCommand{
		Actor: validActor(), Operation: validOperation(),
		Record:  ProvisionalRecord{LocalID: "local-asset-1", Kind: RecordAsset, TenantID: "tenant-a", OrganizationID: "org-a", ReconcileState: ReconcilePending},
		Outcome: ReconcileMatched, CanonicalID: "asset-1",
	}
	if err := ValidateReconcileProvisionalCommand(command); err != nil {
		t.Fatalf("valid match rejected: %v", err)
	}

	command.CanonicalID = ""
	if err := ValidateReconcileProvisionalCommand(command); !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("expected canonical ID requirement, got %v", err)
	}
}

func TestCertificateValidationIsSeparateAndScoped(t *testing.T) {
	order := validWorkOrder()
	command := RequestCertificateValidationCommand{Actor: validActor(), Operation: validOperation(), WorkOrderID: order.ID, InspectionIDs: []string{"inspection-1"}}
	if err := ValidateCertificateValidationCommand(command, order); err != nil {
		t.Fatalf("valid certificate validation request rejected: %v", err)
	}

	command.Actor.Role = "inspector"
	if err := ValidateCertificateValidationCommand(command, order); err != nil {
		t.Fatalf("role authorization belongs to service policy, not shape validation: %v", err)
	}
}

func testTime() (t time.Time) {
	return time.Unix(1, 0)
}
