package workorder

import (
	"context"
	"testing"
)

type noopRepository struct{}
type noopTransactions struct{}
type noopAuthorizer struct{}

func (noopRepository) GetWorkOrder(context.Context, ActorContext, string) (WorkOrder, error) {
	return WorkOrder{}, nil
}
func (noopRepository) GetAssignment(context.Context, ActorContext, string) (Assignment, error) {
	return Assignment{}, nil
}
func (noopRepository) FindOperationReceipt(context.Context, ActorContext, string) (MutationReceipt, bool, error) {
	return MutationReceipt{}, false, nil
}
func (noopRepository) CreateRequest(context.Context, CreateRequestCommand) (MutationReceipt, error) {
	return MutationReceipt{}, nil
}
func (noopRepository) AssignScope(context.Context, AssignScopeCommand) (MutationReceipt, error) {
	return MutationReceipt{}, nil
}
func (noopRepository) TransitionExecution(context.Context, TransitionExecutionCommand) (MutationReceipt, error) {
	return MutationReceipt{}, nil
}
func (noopRepository) SubmitPartial(context.Context, SubmitPartialCommand) (MutationReceipt, error) {
	return MutationReceipt{}, nil
}
func (noopRepository) ReassignScope(context.Context, ReassignScopeCommand) (MutationReceipt, error) {
	return MutationReceipt{}, nil
}
func (noopRepository) ReconcileProvisional(context.Context, ReconcileProvisionalCommand) (MutationReceipt, error) {
	return MutationReceipt{}, nil
}
func (noopRepository) RequestCertificateValidation(context.Context, RequestCertificateValidationCommand) (MutationReceipt, error) {
	return MutationReceipt{}, nil
}
func (noopRepository) AddEvidenceReference(context.Context, AddEvidenceReferenceCommand) (MutationReceipt, error) {
	return MutationReceipt{}, nil
}

func (noopTransactions) WithinTransaction(context.Context, ActorContext, func(context.Context, Repository) error) error {
	return nil
}
func (noopAuthorizer) CanCreateRequest(context.Context, ActorContext) error          { return nil }
func (noopAuthorizer) CanAssignScope(context.Context, ActorContext, WorkOrder) error { return nil }
func (noopAuthorizer) CanTransitionExecution(context.Context, ActorContext, WorkOrder, ExecutionState) error {
	return nil
}
func (noopAuthorizer) CanSubmitPartial(context.Context, ActorContext, WorkOrder, Assignment) error {
	return nil
}
func (noopAuthorizer) CanReassignScope(context.Context, ActorContext, WorkOrder) error { return nil }
func (noopAuthorizer) CanReconcileProvisional(context.Context, ActorContext, ProvisionalRecord) error {
	return nil
}
func (noopAuthorizer) CanRequestCertificateValidation(context.Context, ActorContext, WorkOrder) error {
	return nil
}
func (noopAuthorizer) CanAddEvidenceReference(context.Context, ActorContext, WorkOrder, EvidenceReference) error {
	return nil
}

func TestServiceDependenciesRequireAllAuthorityBoundaries(t *testing.T) {
	if err := (ServiceDependencies{}).Validate(); err == nil {
		t.Fatal("expected missing dependency rejection")
	}
	deps := ServiceDependencies{Repository: noopRepository{}, Transactions: noopTransactions{}, Authorizer: noopAuthorizer{}}
	if err := deps.Validate(); err != nil {
		t.Fatalf("valid dependencies rejected: %v", err)
	}
}

func TestConflictCategoriesRemainExplicit(t *testing.T) {
	values := []string{ConflictStaleRevision, ConflictIdempotencyMismatch, ConflictOutOfScope, ConflictTenantMismatch, ConflictAlreadySubmitted, ConflictOwnershipChanged, ConflictProvisionalMatch}
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" {
			t.Fatal("conflict code must not be empty")
		}
		if seen[value] {
			t.Fatalf("duplicate conflict code %q", value)
		}
		seen[value] = true
	}
}
