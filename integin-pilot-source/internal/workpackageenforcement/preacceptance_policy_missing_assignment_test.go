package workpackageenforcement

import (
	"context"
	"errors"
	"testing"
	"time"

	domainsync "integin/internal/domain/sync"
)

func TestPreAcceptancePolicyRejectsMissingAssignment(t *testing.T) {
	pkg := approvedPackage(t)
	validator, err := NewValidator(memoryResolver{packageValue: pkg})
	if err != nil {
		t.Fatalf("new validator: %v", err)
	}

	observer := &policyObserver{}
	policy, err := NewPreAcceptancePolicy(
		validator,
		policyAssignmentResolver{err: errors.New("assignment unavailable")},
		func() time.Time { return time.Date(2026, 8, 18, 10, 30, 0, 0, time.UTC) },
		observer,
	)
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}

	transaction := domainsync.Transaction{
		TenantID:       "tenant-1",
		OrganizationID: "org-1",
		DeviceID:       "device-1",
		EntityID:       "inspection-1",
		Operation:      inspectionSubmissionOperation,
		AuthorityEpoch: 7,
		Payload:        []byte(`{"inspection_id":"inspection-1","work_package_id":"package-1","work_package_version":1,"work_package_hash":"` + pkg.PackageHash + `","work_package_field_sequence":["condition","capacity"],"findings":[{"item_id":"condition","response":"pass"},{"item_id":"capacity","response":"12.5"}]}`),
	}

	if err := policy.ValidatePreAcceptance(context.Background(), transaction); err == nil {
		t.Fatal("missing assignment was accepted")
	}
	if len(observer.observations) == 0 {
		t.Fatal("missing assignment did not produce a policy observation")
	}
	got := observer.observations[len(observer.observations)-1]
	if got.State != PolicyObservationRejected || got.Category != "assignment_unavailable" {
		t.Fatalf("last observation = %#v", got)
	}
}
