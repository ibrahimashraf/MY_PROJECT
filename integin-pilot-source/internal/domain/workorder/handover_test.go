package workorder

import (
	"errors"
	"testing"
	"time"
)

func validHandoverOrder() WorkOrder {
	return WorkOrder{
		ID: "wo-1", JobNumber: "JOB-1", TenantID: "tenant-a", OrganizationID: "org-a", ClientID: "client-a",
		RequestState: RequestAccepted, ExecutionState: ExecutionInProgress,
		CommercialState: CommercialNotReady, CertificateState: CertificateNotStarted,
		Revision: 2,
	}
}

func validHandoverAssignments(order WorkOrder) (Assignment, Assignment) {
	from := Assignment{
		ID: "assignment-from", TenantID: order.TenantID, OrganizationID: order.OrganizationID, WorkOrderID: order.ID, InspectorID: "inspector-1",
		ScopeItemIDs: []string{"scope-1"}, State: AssignmentActive, Revision: 1, EffectiveFrom: time.Unix(10, 0),
	}
	to := Assignment{
		ID: "assignment-to", TenantID: order.TenantID, OrganizationID: order.OrganizationID, WorkOrderID: order.ID, InspectorID: "inspector-2",
		ScopeItemIDs: []string{"scope-1"}, State: AssignmentActive, Revision: 1, EffectiveFrom: time.Unix(20, 0),
	}
	return from, to
}

func validHandover(order WorkOrder, from, to Assignment) Handover {
	return Handover{
		ID: "handover-1", TenantID: order.TenantID, OrganizationID: order.OrganizationID, WorkOrderID: order.ID,
		FromAssignmentID: from.ID, ToAssignmentID: to.ID,
		State: HandoverRequested, Revision: 1,
	}
}

func validHandoverCommand(order WorkOrder, handover Handover) HandoverCommand {
	return HandoverCommand{
		Actor: validActor(), Operation: validOperation(), WorkOrderID: order.ID, Handover: handover,
	}
}

func TestValidateHandoverCommandAcceptsValid(t *testing.T) {
	order := validHandoverOrder()
	from, to := validHandoverAssignments(order)
	handover := validHandover(order, from, to)
	cmd := validHandoverCommand(order, handover)
	if err := ValidateHandoverCommand(cmd, order, from, to); err != nil {
		t.Fatalf("valid handover command rejected: %v", err)
	}
}

func TestValidateHandoverCommandRejectsEmptyHandoverID(t *testing.T) {
	order := validHandoverOrder()
	from, to := validHandoverAssignments(order)
	handover := validHandover(order, from, to)
	handover.ID = ""
	cmd := validHandoverCommand(order, handover)
	if err := ValidateHandoverCommand(cmd, order, from, to); !errors.Is(err, ErrInvalidHandover) {
		t.Fatalf("expected handover identity rejection, got %v", err)
	}
}

func TestValidateHandoverCommandRejectsSameFromAndTo(t *testing.T) {
	order := validHandoverOrder()
	from, to := validHandoverAssignments(order)
	handover := validHandover(order, from, to)
	handover.ToAssignmentID = handover.FromAssignmentID
	to.ID = from.ID
	cmd := validHandoverCommand(order, handover)
	if err := ValidateHandoverCommand(cmd, order, from, to); !errors.Is(err, ErrInvalidHandover) {
		t.Fatalf("expected same assignment rejection, got %v", err)
	}
}

func TestValidateHandoverCommandRejectsTenantMismatch(t *testing.T) {
	order := validHandoverOrder()
	from, to := validHandoverAssignments(order)
	handover := validHandover(order, from, to)
	handover.TenantID = "tenant-b"
	cmd := validHandoverCommand(order, handover)
	if err := ValidateHandoverCommand(cmd, order, from, to); !errors.Is(err, ErrInvalidHandover) {
		t.Fatalf("expected tenant mismatch rejection, got %v", err)
	}
}

func TestValidateHandoverCommandRejectsInvalidState(t *testing.T) {
	order := validHandoverOrder()
	from, to := validHandoverAssignments(order)
	handover := validHandover(order, from, to)
	handover.State = HandoverState("invalid")
	cmd := validHandoverCommand(order, handover)
	if err := ValidateHandoverCommand(cmd, order, from, to); !errors.Is(err, ErrInvalidHandover) {
		t.Fatalf("expected invalid state rejection, got %v", err)
	}
}

func TestValidateHandoverCommandRejectsMissingRevision(t *testing.T) {
	order := validHandoverOrder()
	from, to := validHandoverAssignments(order)
	handover := validHandover(order, from, to)
	handover.Revision = 0
	cmd := validHandoverCommand(order, handover)
	if err := ValidateHandoverCommand(cmd, order, from, to); !errors.Is(err, ErrInvalidHandover) {
		t.Fatalf("expected revision rejection, got %v", err)
	}
}

func TestValidateHandoverCommandRejectsWorkOrderMismatch(t *testing.T) {
	order := validHandoverOrder()
	from, to := validHandoverAssignments(order)
	handover := validHandover(order, from, to)
	cmd := validHandoverCommand(order, handover)
	cmd.WorkOrderID = "wo-other"
	if err := ValidateHandoverCommand(cmd, order, from, to); !errors.Is(err, ErrInvalidHandover) {
		t.Fatalf("expected work-order mismatch rejection, got %v", err)
	}
}

func TestValidateHandoverCommandRejectsActorTenantMismatch(t *testing.T) {
	order := validHandoverOrder()
	from, to := validHandoverAssignments(order)
	handover := validHandover(order, from, to)
	cmd := validHandoverCommand(order, handover)
	cmd.Actor.TenantID = "tenant-b"
	if err := ValidateHandoverCommand(cmd, order, from, to); !errors.Is(err, ErrInvalidHandover) {
		t.Fatalf("expected actor tenant rejection, got %v", err)
	}
}

func TestValidateHandoverCommandRejectsFromAssignmentMismatch(t *testing.T) {
	order := validHandoverOrder()
	from, to := validHandoverAssignments(order)
	handover := validHandover(order, from, to)
	handover.FromAssignmentID = "assignment-unknown"
	cmd := validHandoverCommand(order, handover)
	if err := ValidateHandoverCommand(cmd, order, from, to); !errors.Is(err, ErrInvalidHandover) {
		t.Fatalf("expected from-assignment mismatch rejection, got %v", err)
	}
}

func TestCanTransitionHandover(t *testing.T) {
	if !CanTransitionHandover(HandoverRequested, HandoverApproved) {
		t.Fatal("requested -> approved must be allowed")
	}
	if !CanTransitionHandover(HandoverApproved, HandoverTransferred) {
		t.Fatal("approved -> transferred must be allowed")
	}
	if CanTransitionHandover(HandoverRequested, HandoverTransferred) {
		t.Fatal("requested -> transferred must not be allowed directly")
	}
	if CanTransitionHandover(HandoverApproved, HandoverRequested) {
		t.Fatal("approved -> requested backward must not be allowed")
	}
	if !CanTransitionHandover(HandoverRequested, HandoverRequested) {
		t.Fatal("same state must be allowed")
	}
}

func TestHandoverStateConstants(t *testing.T) {
	if HandoverRequested != "requested" || HandoverApproved != "approved" || HandoverTransferred != "transferred" {
		t.Fatalf("unexpected handover state values: %q %q %q", HandoverRequested, HandoverApproved, HandoverTransferred)
	}
}
