package workorder

import (
	"errors"
	"testing"
)

func TestValidateCertificateInspectionStateRejectsUnsubmittedInspection(t *testing.T) {
	if err := ValidateCertificateInspectionState("COMPLETED", "OPEN"); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("expected invalid inspection scope, got %v", err)
	}
}

func TestValidateCertificateInspectionStateAcceptsCompletedSubmittedInspection(t *testing.T) {
	if err := ValidateCertificateInspectionState("COMPLETED", "SUBMITTED"); err != nil {
		t.Fatalf("expected completed submitted inspection to be accepted, got %v", err)
	}
}
