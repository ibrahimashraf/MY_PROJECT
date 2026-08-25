package workorderauth

import (
	"errors"
	"testing"

	"integin/internal/identity"
)

func TestActorFromMembershipUsesOnlyResolvedLocalFields(t *testing.T) {
	membership := identity.Membership{ActorID: " inspector-1 ", TenantID: "tenant", OrganizationID: "org", WorkOrderRole: " inspector ", Capabilities: []string{"workorder.submit_partial"}}
	actor, err := ActorFromMembership(membership)
	if err != nil {
		t.Fatal(err)
	}
	if actor.ActorID != "inspector-1" || actor.Role != "inspector" || len(actor.Capabilities) != 1 {
		t.Fatalf("unexpected derived actor: %+v", actor)
	}
}

func TestActorFromMembershipFailsClosedOnMissingAuthority(t *testing.T) {
	_, err := ActorFromMembership(identity.Membership{ActorID: "actor", TenantID: "tenant", OrganizationID: "org", WorkOrderRole: "inspector"})
	if !errors.Is(err, ErrInvalidDerivedActor) {
		t.Fatalf("expected invalid derived actor, got %v", err)
	}
}
