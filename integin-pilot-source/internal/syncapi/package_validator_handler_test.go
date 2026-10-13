package syncapi

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/domain/workpackage"
	"integin/internal/workpackageenforcement"
)

type rejectingWorkPackageResolver struct{}

func (rejectingWorkPackageResolver) GetApproved(context.Context, string, string, string, int) (workpackage.Package, error) {
	return workpackage.Package{}, errors.New("approved work package is unavailable")
}

func TestHandlerDefaultsToNilPackageValidator(t *testing.T) {
	processor, _ := testProcessor(t)
	handler := NewHandler(processor)
	if handler.PackageValidator != nil {
		t.Fatal("default handler unexpectedly enables package validation")
	}
}

func TestHandlerRejectsInspectionPayloadWhenPackageValidatorRejects(t *testing.T) {
	processor, authority := testProcessor(t)
	validator, err := workpackageenforcement.NewValidator(rejectingWorkPackageResolver{})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(processor)
	handler.PackageValidator = validator
	handler.RegisterAuthority(authority)

	body := []byte(`{
		"protocol_version":"v1",
		"transaction_id":"validator-rejected",
		"tenant_id":"tenant-1",
		"organization_id":"org-1",
		"environment":"pilot",
		"device_id":"device-1",
		"user_id":"user-1",
		"sequence_number":1,
		"operation":"InspectionSubmitted",
		"entity_id":"inspection-1",
		"payload":{"findings":[]},
		"authority_id":"` + authority.ID + `"
	}`)
	record := httptest.NewRecorder()
	handler.ServeHTTP(record, httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body)))
	if record.Code != http.StatusUnprocessableEntity {
		t.Fatalf("validator rejection status = %d, want %d", record.Code, http.StatusUnprocessableEntity)
	}
}
