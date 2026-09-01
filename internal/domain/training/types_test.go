package training

import (
	"testing"
)

func TestActorContextValidate(t *testing.T) {
	a := ActorContext{TenantID: "t1", OrganizationID: "o1", ActorID: "a1"}
	if err := a.Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}

	a2 := ActorContext{TenantID: "t1"}
	if err := a2.Validate(); err != ErrInvalidIdentity {
		t.Fatalf("expected ErrInvalidIdentity, got %v", err)
	}

	a3 := ActorContext{}
	if err := a3.Validate(); err != ErrInvalidIdentity {
		t.Fatalf("expected ErrInvalidIdentity, got %v", err)
	}
}
