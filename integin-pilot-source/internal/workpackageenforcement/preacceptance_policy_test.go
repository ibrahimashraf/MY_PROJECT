package workpackageenforcement

import (
	"context"
	"errors"
	"testing"
	"time"

	"integin/internal/domain/sync"
	"integin/internal/domain/workpackage"
)

type policyAssignmentResolver struct {
	assignment workpackage.Assignment
	err        error
}

func (r policyAssignmentResolver) GetCurrentAssignment(_ context.Context, _, _, _, _ string, _ time.Time) (workpackage.Assignment, error) {
	return r.assignment, r.err
}

type policyObserver struct{ observations []PolicyObservation }

func (o *policyObserver) ObserveWorkPackagePolicy(observation PolicyObservation) {
	o.observations = append(o.observations, observation)
}

func TestPreAcceptancePolicyRequiresCurrentAssignmentMatch(t *testing.T) {
	pkg := approvedPackage(t)
	validator, err := NewValidator(memoryResolver{packageValue: pkg})
	if err != nil {
		t.Fatalf("new validator: %v", err)
	}
	observer := &policyObserver{}
	policy, err := NewPreAcceptancePolicy(validator, policyAssignmentResolver{assignment: workpackage.Assignment{PackageID: pkg.ID, PackageVersion: pkg.PackageVersion, AuthorityEpoch: 7, ExpiresAt: time.Date(2026, 8, 17, 13, 0, 0, 0, time.UTC)}}, func() time.Time { return time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC) }, observer)
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}
	transaction := sync.Transaction{TenantID: "tenant-1", OrganizationID: "org-1", DeviceID: "device-1", EntityID: "inspection-1", Operation: inspectionSubmissionOperation, AuthorityEpoch: 7, Payload: []byte(`{"inspection_id":"inspection-1","work_package_id":"package-1","work_package_version":1,"work_package_hash":"` + pkg.PackageHash + `","work_package_field_sequence":["condition","capacity"],"findings":[{"item_id":"condition","response":"pass"},{"item_id":"capacity","response":"12.5"}]}`)}
	if err := policy.ValidatePreAcceptance(context.Background(), transaction); err != nil {
		t.Fatalf("validate matching transaction: %v", err)
	}
	if len(observer.observations) != 3 || observer.observations[2].State != PolicyObservationAllowed {
		t.Fatalf("observations = %#v", observer.observations)
	}
	transaction.Payload = []byte(`{"inspection_id":"inspection-1","work_package_id":"other-package","work_package_version":1}`)
	if err := policy.ValidatePreAcceptance(context.Background(), transaction); !errors.Is(err, ErrAssignmentBinding) {
		t.Fatalf("mismatch error = %v", err)
	}
	if got := observer.observations[len(observer.observations)-1]; got.State != PolicyObservationRejected || got.Category != "assignment_mismatch" {
		t.Fatalf("last observation = %#v", got)
	}
}

func TestPreAcceptancePolicySkipsOtherVerifiedOperations(t *testing.T) {
	pkg := approvedPackage(t)
	validator, _ := NewValidator(memoryResolver{packageValue: pkg})
	observer := &policyObserver{}
	policy, err := NewPreAcceptancePolicy(validator, policyAssignmentResolver{}, time.Now, observer)
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}
	if err := policy.ValidatePreAcceptance(context.Background(), sync.Transaction{Operation: "EvidenceUploaded"}); err != nil {
		t.Fatalf("non-inspection operation: %v", err)
	}
	if len(observer.observations) != 1 || observer.observations[0].State != PolicyObservationPrepared {
		t.Fatalf("observations = %#v", observer.observations)
	}
}
