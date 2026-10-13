package workorder

import (
	"context"
	"testing"
)

type serviceTestRepo struct {
	receipt     MutationReceipt
	found       bool
	createCalls int
	lookupCalls int
}

func (r *serviceTestRepo) GetWorkOrder(context.Context, ActorContext, string) (WorkOrder, error) {
	return validWorkOrder(), nil
}
func (r *serviceTestRepo) GetAssignment(context.Context, ActorContext, string) (Assignment, error) {
	return Assignment{ID: "assignment-1", TenantID: "tenant-a", OrganizationID: "org-a", WorkOrderID: "wo-1", InspectorID: "inspector-1", ScopeItemIDs: []string{"scope-1"}, State: AssignmentActive, Revision: 1, EffectiveFrom: testTime()}, nil
}
func (r *serviceTestRepo) FindOperationReceipt(context.Context, ActorContext, string) (MutationReceipt, bool, error) {
	r.lookupCalls++
	return r.receipt, r.found, nil
}
func (r *serviceTestRepo) CreateRequest(context.Context, CreateRequestCommand) (MutationReceipt, error) {
	if r.found {
		return r.receipt, nil
	}
	r.createCalls++
	return MutationReceipt{OperationID: "op-1", IdempotencyKey: "idem-1", TenantID: "tenant-a", WorkOrderID: "wo-1", Revision: 1, Status: ReceiptAccepted}, nil
}
func (r *serviceTestRepo) AssignScope(context.Context, AssignScopeCommand) (MutationReceipt, error) {
	return MutationReceipt{}, ErrInvalidScope
}
func (r *serviceTestRepo) TransitionExecution(context.Context, TransitionExecutionCommand) (MutationReceipt, error) {
	return MutationReceipt{}, ErrInvalidTransition
}
func (r *serviceTestRepo) SubmitPartial(context.Context, SubmitPartialCommand) (MutationReceipt, error) {
	return MutationReceipt{}, ErrInvalidScope
}
func (r *serviceTestRepo) ReassignScope(context.Context, ReassignScopeCommand) (MutationReceipt, error) {
	return MutationReceipt{}, ErrInvalidAssignment
}
func (r *serviceTestRepo) ReconcileProvisional(context.Context, ReconcileProvisionalCommand) (MutationReceipt, error) {
	return MutationReceipt{}, ErrInvalidIdentity
}
func (r *serviceTestRepo) RequestCertificateValidation(context.Context, RequestCertificateValidationCommand) (MutationReceipt, error) {
	return MutationReceipt{}, ErrInvalidScope
}
func (r *serviceTestRepo) AddEvidenceReference(context.Context, AddEvidenceReferenceCommand) (MutationReceipt, error) {
	return MutationReceipt{}, ErrInvalidEvidence
}

type serviceTestTx struct{ repo Repository }

func (t serviceTestTx) WithinTransaction(ctx context.Context, _ ActorContext, fn func(context.Context, Repository) error) error {
	return fn(ctx, t.repo)
}

type serviceTestAuth struct{}

func (serviceTestAuth) CanCreateRequest(context.Context, ActorContext) error          { return nil }
func (serviceTestAuth) CanAssignScope(context.Context, ActorContext, WorkOrder) error { return nil }
func (serviceTestAuth) CanTransitionExecution(context.Context, ActorContext, WorkOrder, ExecutionState) error {
	return nil
}
func (serviceTestAuth) CanSubmitPartial(context.Context, ActorContext, WorkOrder, Assignment) error {
	return nil
}
func (serviceTestAuth) CanReassignScope(context.Context, ActorContext, WorkOrder) error { return nil }
func (serviceTestAuth) CanReconcileProvisional(context.Context, ActorContext, ProvisionalRecord) error {
	return nil
}
func (serviceTestAuth) CanRequestCertificateValidation(context.Context, ActorContext, WorkOrder) error {
	return nil
}
func (serviceTestAuth) CanAddEvidenceReference(context.Context, ActorContext, WorkOrder, EvidenceReference) error {
	return nil
}

func TestApplicationServiceCreateRequestUsesTransactionalRepository(t *testing.T) {
	repo := &serviceTestRepo{}
	service, err := NewService(ServiceDependencies{Repository: repo, Transactions: serviceTestTx{repo: repo}, Authorizer: serviceTestAuth{}})
	if err != nil {
		t.Fatal(err)
	}
	command := CreateRequestCommand{Actor: validActor(), Operation: validOperation(), WorkOrder: validWorkOrder()}
	receipt, err := service.CreateRequest(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != ReceiptAccepted || repo.createCalls != 1 || repo.lookupCalls != 1 {
		t.Fatalf("unexpected dispatch: receipt=%+v create=%d lookup=%d", receipt, repo.createCalls, repo.lookupCalls)
	}
}

func TestApplicationServiceCreateRequestReturnsIdempotentReceiptWithoutMutation(t *testing.T) {
	repo := &serviceTestRepo{found: true, receipt: MutationReceipt{OperationID: "op-1", IdempotencyKey: "idem-1", TenantID: "tenant-a", WorkOrderID: "wo-1", Revision: 1, Status: ReceiptAccepted}}
	service, err := NewService(ServiceDependencies{Repository: repo, Transactions: serviceTestTx{repo: repo}, Authorizer: serviceTestAuth{}})
	if err != nil {
		t.Fatal(err)
	}
	command := CreateRequestCommand{Actor: validActor(), Operation: validOperation(), WorkOrder: validWorkOrder()}
	receipt, err := service.CreateRequest(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if receipt != repo.receipt || repo.createCalls != 0 {
		t.Fatalf("idempotent receipt was not returned without mutation: receipt=%+v create=%d", receipt, repo.createCalls)
	}
}
