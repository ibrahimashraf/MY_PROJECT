package license

import (
	"testing"
	"time"
)

func TestLicenseValidate(t *testing.T) {
	lic := License{
		ID:             "lic-1",
		TenantID:       "t1",
		OrganizationID: "o1",
		Tier:           TierPro,
		CreatedBy:      "user-1",
	}
	if err := lic.Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestLicenseValidateMissingFields(t *testing.T) {
	lic := License{ID: "lic-1"}
	if err := lic.Validate(); err != ErrInvalidIdentity {
		t.Fatalf("expected ErrInvalidIdentity, got %v", err)
	}
}

func TestLicenseIsExpired(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	lic := License{ExpiresAt: &past}
	if !lic.IsExpired(time.Now()) {
		t.Fatal("expected expired")
	}

	future := time.Now().Add(time.Hour)
	lic2 := License{ExpiresAt: &future}
	if lic2.IsExpired(time.Now()) {
		t.Fatal("expected not expired")
	}

	lic3 := License{ExpiresAt: nil}
	if lic3.IsExpired(time.Now()) {
		t.Fatal("nil expiry should not be expired")
	}
}

func TestActorContextValidate(t *testing.T) {
	a := ActorContext{TenantID: "t1", OrganizationID: "o1", ActorID: "a1"}
	if err := a.Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}

	a2 := ActorContext{TenantID: "t1"}
	if err := a2.Validate(); err != ErrInvalidIdentity {
		t.Fatalf("expected ErrInvalidIdentity, got %v", err)
	}
}
