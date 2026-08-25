package workorder

import (
	"errors"
	"testing"
	"time"
)

func validWorkOrder() WorkOrder {
	return WorkOrder{
		ID: "wo-1", JobNumber: "JOB-1", TenantID: "tenant-a", OrganizationID: "org-a", ClientID: "client-a",
		RequestState: RequestAccepted, ExecutionState: ExecutionReady,
		CommercialState: CommercialNotReady, CertificateState: CertificateNotStarted,
		Revision: 1,
	}
}

func TestWorkOrderIdentityAndScopeAreTenantBound(t *testing.T) {
	order := validWorkOrder()
	if err := order.ValidateIdentity(); err != nil {
		t.Fatalf("valid order rejected: %v", err)
	}

	valid := ScopeItem{ID: "scope-1", TenantID: "tenant-a", OrganizationID: "org-a", ClientID: "client-a", LocationID: "loc-1", AssetID: "asset-1", AssetType: "wire_rope"}
	if err := valid.ValidateFor(order); err != nil {
		t.Fatalf("valid scope rejected: %v", err)
	}

	crossTenant := valid
	crossTenant.TenantID = "tenant-b"
	if err := crossTenant.ValidateFor(order); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("expected tenant scope rejection, got %v", err)
	}

	crossClient := valid
	crossClient.ClientID = "client-b"
	if err := crossClient.ValidateFor(order); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("expected client scope rejection, got %v", err)
	}
}

func TestAssignmentRequiresExplicitScopeAndMatchesOrder(t *testing.T) {
	order := validWorkOrder()
	assignment := Assignment{
		ID: "assignment-1", TenantID: order.TenantID, OrganizationID: order.OrganizationID, WorkOrderID: order.ID, InspectorID: "inspector-1",
		ScopeItemIDs: []string{"scope-1"}, State: AssignmentActive, Revision: 1, EffectiveFrom: time.Unix(1, 0),
	}
	if err := assignment.ValidateFor(order); err != nil {
		t.Fatalf("valid assignment rejected: %v", err)
	}

	wrongOrder := assignment
	wrongOrder.WorkOrderID = "wo-other"
	if err := wrongOrder.ValidateFor(order); !errors.Is(err, ErrInvalidAssignment) {
		t.Fatalf("expected work-order rejection, got %v", err)
	}

	withoutScope := assignment
	withoutScope.ScopeItemIDs = nil
	if err := withoutScope.ValidateFor(order); !errors.Is(err, ErrInvalidAssignment) {
		t.Fatalf("expected empty scope rejection, got %v", err)
	}
}

func TestOperationMetadataRequiresRetryIdentityAndRevision(t *testing.T) {
	valid := OperationMeta{OperationID: "op-1", IdempotencyKey: "idem-1", ExpectedRevision: 1}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid operation rejected: %v", err)
	}

	missingRevision := valid
	missingRevision.ExpectedRevision = 0
	if err := missingRevision.Validate(); !errors.Is(err, ErrInvalidOperation) {
		t.Fatalf("expected revision rejection, got %v", err)
	}
}

func TestProvisionalRecordCannotClaimCanonicalIdentityWhilePending(t *testing.T) {
	pending := ProvisionalRecord{LocalID: "local-asset-1", Kind: RecordAsset, TenantID: "tenant-a", OrganizationID: "org-a", ReconcileState: ReconcilePending}
	if err := pending.Validate(); err != nil {
		t.Fatalf("valid pending record rejected: %v", err)
	}

	invalid := pending
	invalid.CanonicalID = "asset-1"
	if err := invalid.Validate(); !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("expected pending canonical identity rejection, got %v", err)
	}
}

func TestExecutionTransitionsPreservePartialWork(t *testing.T) {
	allowed := [][2]ExecutionState{
		{ExecutionReady, ExecutionAssigned},
		{ExecutionAssigned, ExecutionInProgress},
		{ExecutionInProgress, ExecutionPartiallySubmitted},
		{ExecutionPartiallySubmitted, ExecutionInProgress},
		{ExecutionPartiallySubmitted, ExecutionCompleted},
	}
	for _, transition := range allowed {
		if !CanTransition(transition[0], transition[1]) {
			t.Fatalf("expected transition %q -> %q", transition[0], transition[1])
		}
	}
	if CanTransition(ExecutionReady, ExecutionCompleted) {
		t.Fatal("ready must not jump directly to completed")
	}
	if CanTransition(ExecutionInProgress, ExecutionAssigned) {
		t.Fatal("in-progress must not move backward to assigned")
	}
}

func TestValidateProvisionalFingerprintRejectsChangedCandidate(t *testing.T) {
	if err := ValidateProvisionalFingerprint("fingerprint-a", "fingerprint-b"); !errors.Is(err, ErrProvisionalConflict) {
		t.Fatalf("expected provisional conflict, got %v", err)
	}
}

func TestValidateProvisionalFingerprintAcceptsIdenticalCandidate(t *testing.T) {
	if err := ValidateProvisionalFingerprint("fingerprint-a", "fingerprint-a"); err != nil {
		t.Fatalf("expected identical fingerprint to be accepted, got %v", err)
	}
}
