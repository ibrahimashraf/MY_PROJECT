package workorder

import (
	"errors"
	"testing"
)

func TestValidateCertificateValidationCommandRejectsDuplicateInspectionIDs(t *testing.T) {
	actor := ActorContext{TenantID: "tenant-a", OrganizationID: "org-a", ActorID: "actor-a", Role: "manager"}
	order := WorkOrder{ID: "order-a", TenantID: actor.TenantID, OrganizationID: actor.OrganizationID}
	command := RequestCertificateValidationCommand{
		Actor:         actor,
		Operation:     OperationMeta{OperationID: "operation-a", IdempotencyKey: "idempotency-a", ExpectedRevision: 1},
		WorkOrderID:   order.ID,
		InspectionIDs: []string{"inspection-a", "inspection-a"},
	}
	if err := ValidateCertificateValidationCommand(command, order); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("expected duplicate inspection IDs to be rejected with ErrInvalidScope, got %v", err)
	}
}
