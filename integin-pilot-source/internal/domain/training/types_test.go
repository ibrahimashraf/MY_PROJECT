package training

import (
	"testing"
	"time"
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

func TestISO9712CompetenceGating(t *testing.T) {
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	validExp := now.Add(365 * 24 * time.Hour)
	pastExp := now.Add(-24 * time.Hour)

	l1 := NDTQualification{
		TechnicianID:     "tech-1",
		Method:           "UT",
		Level:            NDTLevel1,
		CertificationRef: "PCN-123456",
		AccreditingBody:  "BINDT",
		ExpiresAt:        validExp,
	}

	l2 := NDTQualification{
		TechnicianID:     "tech-2",
		Method:           "UT",
		Level:            NDTLevel2,
		CertificationRef: "PCN-654321",
		AccreditingBody:  "BINDT",
		ExpiresAt:        validExp,
	}

	l3 := NDTQualification{
		TechnicianID:     "tech-3",
		Method:           "UT",
		Level:            NDTLevel3,
		CertificationRef: "PCN-999999",
		AccreditingBody:  "BINDT",
		ExpiresAt:        validExp,
	}

	// 1. Level 1 can perform test and record raw data
	if err := l1.ValidateNDTSignOff("PERFORM_TEST", now); err != nil {
		t.Errorf("Level 1 should be allowed to perform test: %v", err)
	}
	if err := l1.ValidateNDTSignOff("RECORD_RAW_DATA", now); err != nil {
		t.Errorf("Level 1 should be allowed to record data: %v", err)
	}

	// 2. Level 1 must be strictly rejected from interpreting results or signing certificates
	if err := l1.ValidateNDTSignOff("INTERPRET_RESULTS", now); err != ErrInsufficientNDTLevel {
		t.Errorf("Level 1 must not interpret results, got %v", err)
	}
	if err := l1.ValidateNDTSignOff("SIGN_CERTIFICATE", now); err != ErrInsufficientNDTLevel {
		t.Errorf("Level 1 must not sign certificate, got %v", err)
	}

	// 3. Level 2 can interpret results and sign certificate
	if err := l2.ValidateNDTSignOff("INTERPRET_RESULTS", now); err != nil {
		t.Errorf("Level 2 should interpret results: %v", err)
	}
	if err := l2.ValidateNDTSignOff("SIGN_CERTIFICATE", now); err != nil {
		t.Errorf("Level 2 should sign certificate: %v", err)
	}

	// 4. Level 2 cannot approve procedure (Level 3 only)
	if err := l2.ValidateNDTSignOff("APPROVE_PROCEDURE", now); err != ErrLevel3Required {
		t.Errorf("Level 2 must not approve procedure, got %v", err)
	}

	// 5. Level 3 can approve procedure
	if err := l3.ValidateNDTSignOff("APPROVE_PROCEDURE", now); err != nil {
		t.Errorf("Level 3 should approve procedure: %v", err)
	}

	// 6. Expired qualification must be rejected
	l2Expired := l2
	l2Expired.ExpiresAt = pastExp
	if err := l2Expired.ValidateNDTSignOff("SIGN_CERTIFICATE", now); err != ErrQualificationExpired {
		t.Errorf("Expired qualification must refuse, got %v", err)
	}
}
