package workorderauth

import (
	"context"
	"errors"
	"testing"

	"integin/internal/domain/workorder"
)

func TestSubmitPartialRequiresDerivedInspectorAndCapability(t *testing.T) {
	authorizer := New()
	order := workorder.WorkOrder{TenantID: "tenant", OrganizationID: "org"}
	assignment := workorder.Assignment{InspectorID: "inspector-1"}
	actor := workorder.ActorContext{TenantID: "tenant", OrganizationID: "org", ActorID: "inspector-1", Role: "inspector", Capabilities: []string{CapabilitySubmitPartial}}
	if err := authorizer.CanSubmitPartial(context.Background(), actor, order, assignment); err != nil {
		t.Fatal(err)
	}
	actor.ActorID = "other-inspector"
	if err := authorizer.CanSubmitPartial(context.Background(), actor, order, assignment); !errors.Is(err, ErrDenied) {
		t.Fatalf("expected denied assignment ownership, got %v", err)
	}
	actor.ActorID = "inspector-1"
	actor.Capabilities = nil
	if err := authorizer.CanSubmitPartial(context.Background(), actor, order, assignment); !errors.Is(err, ErrDenied) {
		t.Fatalf("expected denied capability, got %v", err)
	}
}

func TestManagerCannotSubmitInspectorPartialWithoutAdministratorRole(t *testing.T) {
	authorizer := New()
	order := workorder.WorkOrder{TenantID: "tenant", OrganizationID: "org"}
	assignment := workorder.Assignment{InspectorID: "inspector-1"}
	actor := workorder.ActorContext{TenantID: "tenant", OrganizationID: "org", ActorID: "manager-1", Role: "manager", Capabilities: []string{CapabilitySubmitPartial}}
	if err := authorizer.CanSubmitPartial(context.Background(), actor, order, assignment); !errors.Is(err, ErrDenied) {
		t.Fatalf("expected denied role, got %v", err)
	}
}
