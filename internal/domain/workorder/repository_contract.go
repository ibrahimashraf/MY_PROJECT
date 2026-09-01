package workorder

import "context"

// Conflict describes a server-authoritative rejection that must not be silently merged.
type Conflict struct {
	Code             string
	AggregateID      string
	ExpectedRevision int64
	ActualRevision   int64
	OperationID      string
	Message          string
}

const (
	ConflictStaleRevision       = "stale_revision"
	ConflictIdempotencyMismatch = "idempotency_key_reused_with_different_payload"
	ConflictOutOfScope          = "out_of_scope"
	ConflictTenantMismatch      = "tenant_mismatch"
	ConflictAlreadySubmitted    = "already_submitted"
	ConflictOwnershipChanged    = "ownership_changed"
	ConflictProvisionalMatch    = "provisional_match_required"
)

// Repository is the authoritative persistence boundary. Implementations must execute mutations
// transactionally under the authenticated organization context and must never trust local cache scope.
type Repository interface {
	GetWorkOrder(ctx context.Context, actor ActorContext, workOrderID string) (WorkOrder, error)
	GetAssignment(ctx context.Context, actor ActorContext, assignmentID string) (Assignment, error)
	FindOperationReceipt(ctx context.Context, actor ActorContext, idempotencyKey string) (MutationReceipt, bool, error)
	CreateRequest(ctx context.Context, command CreateRequestCommand) (MutationReceipt, error)
	AssignScope(ctx context.Context, command AssignScopeCommand) (MutationReceipt, error)
	TransitionExecution(ctx context.Context, command TransitionExecutionCommand) (MutationReceipt, error)
	SubmitPartial(ctx context.Context, command SubmitPartialCommand) (MutationReceipt, error)
	ReassignScope(ctx context.Context, command ReassignScopeCommand) (MutationReceipt, error)
	ReconcileProvisional(ctx context.Context, command ReconcileProvisionalCommand) (MutationReceipt, error)
	RequestCertificateValidation(ctx context.Context, command RequestCertificateValidationCommand) (MutationReceipt, error)
	AddEvidenceReference(ctx context.Context, command AddEvidenceReferenceCommand) (MutationReceipt, error)
}

// TransactionRunner makes transaction scope explicit without binding the domain to a SQL driver.
type TransactionRunner interface {
	WithinTransaction(ctx context.Context, actor ActorContext, fn func(context.Context, Repository) error) error
}

// Authorizer is evaluated by the service before repository mutation. It cannot grant cross-tenant access.
type Authorizer interface {
	CanCreateRequest(ctx context.Context, actor ActorContext) error
	CanAssignScope(ctx context.Context, actor ActorContext, order WorkOrder) error
	CanTransitionExecution(ctx context.Context, actor ActorContext, order WorkOrder, to ExecutionState) error
	CanSubmitPartial(ctx context.Context, actor ActorContext, order WorkOrder, assignment Assignment) error
	CanReassignScope(ctx context.Context, actor ActorContext, order WorkOrder) error
	CanReconcileProvisional(ctx context.Context, actor ActorContext, record ProvisionalRecord) error
	CanRequestCertificateValidation(ctx context.Context, actor ActorContext, order WorkOrder) error
	CanAddEvidenceReference(ctx context.Context, actor ActorContext, order WorkOrder, evidence EvidenceReference) error
}

// ServiceDependencies are injected into the application service, keeping transport and storage separate.
type ServiceDependencies struct {
	Repository   Repository
	Transactions TransactionRunner
	Authorizer   Authorizer
}

func (d ServiceDependencies) Validate() error {
	if d.Repository == nil || d.Transactions == nil || d.Authorizer == nil {
		return ErrInvalidIdentity
	}
	return nil
}
