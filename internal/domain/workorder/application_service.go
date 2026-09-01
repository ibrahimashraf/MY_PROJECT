package workorder

import "context"

type applicationService struct {
	deps ServiceDependencies
}

func NewService(deps ServiceDependencies) (Service, error) {
	if err := deps.Validate(); err != nil {
		return nil, err
	}
	return applicationService{deps: deps}, nil
}

type mutation func(context.Context, Repository) (MutationReceipt, error)

func (s applicationService) execute(ctx context.Context, actor ActorContext, operation OperationMeta, apply mutation) (MutationReceipt, error) {
	var zero MutationReceipt
	if err := validateActorOperation(actor, operation); err != nil {
		return zero, err
	}
    if receipt, found, err := s.deps.Repository.FindOperationReceipt(ctx, actor, operation.IdempotencyKey); err != nil {
        return zero, err
    } else if found {
        return receipt, nil
    }

	var receipt MutationReceipt
	err := s.deps.Transactions.WithinTransaction(ctx, actor, func(txCtx context.Context, repo Repository) error {
		var err error
		receipt, err = apply(txCtx, repo)
		return err
	})
	return receipt, err
}

func (s applicationService) CreateRequest(ctx context.Context, command CreateRequestCommand) (MutationReceipt, error) {
	if err := ValidateCreateRequestCommand(command); err != nil {
		return MutationReceipt{}, err
	}
	if err := s.deps.Authorizer.CanCreateRequest(ctx, command.Actor); err != nil {
		return MutationReceipt{}, err
	}
	return s.execute(ctx, command.Actor, command.Operation, func(txCtx context.Context, repo Repository) (MutationReceipt, error) {
		return repo.CreateRequest(txCtx, command)
	})
}

func (s applicationService) AssignScope(ctx context.Context, command AssignScopeCommand) (MutationReceipt, error) {
	order, err := s.deps.Repository.GetWorkOrder(ctx, command.Actor, command.WorkOrderID)
	if err != nil {
		return MutationReceipt{}, err
	}
	if err := ValidateAssignScopeCommand(command, order); err != nil {
		return MutationReceipt{}, err
	}
	if err := s.deps.Authorizer.CanAssignScope(ctx, command.Actor, order); err != nil {
		return MutationReceipt{}, err
	}
	return s.execute(ctx, command.Actor, command.Operation, func(txCtx context.Context, repo Repository) (MutationReceipt, error) {
		return repo.AssignScope(txCtx, command)
	})
}

func (s applicationService) TransitionExecution(ctx context.Context, command TransitionExecutionCommand) (MutationReceipt, error) {
	order, err := s.deps.Repository.GetWorkOrder(ctx, command.Actor, command.WorkOrderID)
	if err != nil {
		return MutationReceipt{}, err
	}
	if err := ValidateTransitionExecutionCommand(command, order); err != nil {
		return MutationReceipt{}, err
	}
	if err := s.deps.Authorizer.CanTransitionExecution(ctx, command.Actor, order, command.To); err != nil {
		return MutationReceipt{}, err
	}
	return s.execute(ctx, command.Actor, command.Operation, func(txCtx context.Context, repo Repository) (MutationReceipt, error) {
		return repo.TransitionExecution(txCtx, command)
	})
}

func (s applicationService) SubmitPartial(ctx context.Context, command SubmitPartialCommand) (MutationReceipt, error) {
	order, err := s.deps.Repository.GetWorkOrder(ctx, command.Actor, command.WorkOrderID)
	if err != nil {
		return MutationReceipt{}, err
	}
	assignment, err := s.deps.Repository.GetAssignment(ctx, command.Actor, command.AssignmentID)
	if err != nil {
		return MutationReceipt{}, err
	}
	if err := ValidateSubmitPartialCommand(command, order); err != nil {
		return MutationReceipt{}, err
	}
	if assignment.WorkOrderID != order.ID || assignment.TenantID != order.TenantID || assignment.OrganizationID != order.OrganizationID {
		return MutationReceipt{}, ErrInvalidScope
	}
	if err := s.deps.Authorizer.CanSubmitPartial(ctx, command.Actor, order, assignment); err != nil {
		return MutationReceipt{}, err
	}
	return s.execute(ctx, command.Actor, command.Operation, func(txCtx context.Context, repo Repository) (MutationReceipt, error) {
		return repo.SubmitPartial(txCtx, command)
	})
}

func (s applicationService) ReassignScope(ctx context.Context, command ReassignScopeCommand) (MutationReceipt, error) {
	order, err := s.deps.Repository.GetWorkOrder(ctx, command.Actor, command.WorkOrderID)
	if err != nil {
		return MutationReceipt{}, err
	}
	if err := ValidateReassignScopeCommand(command, order); err != nil {
		return MutationReceipt{}, err
	}
	if err := s.deps.Authorizer.CanReassignScope(ctx, command.Actor, order); err != nil {
		return MutationReceipt{}, err
	}
	return s.execute(ctx, command.Actor, command.Operation, func(txCtx context.Context, repo Repository) (MutationReceipt, error) {
		return repo.ReassignScope(txCtx, command)
	})
}

func (s applicationService) ReconcileProvisional(ctx context.Context, command ReconcileProvisionalCommand) (MutationReceipt, error) {
	if err := ValidateReconcileProvisionalCommand(command); err != nil {
		return MutationReceipt{}, err
	}
	if err := s.deps.Authorizer.CanReconcileProvisional(ctx, command.Actor, command.Record); err != nil {
		return MutationReceipt{}, err
	}
	return s.execute(ctx, command.Actor, command.Operation, func(txCtx context.Context, repo Repository) (MutationReceipt, error) {
		return repo.ReconcileProvisional(txCtx, command)
	})
}

func (s applicationService) RequestCertificateValidation(ctx context.Context, command RequestCertificateValidationCommand) (MutationReceipt, error) {
	order, err := s.deps.Repository.GetWorkOrder(ctx, command.Actor, command.WorkOrderID)
	if err != nil {
		return MutationReceipt{}, err
	}
	if err := ValidateCertificateValidationCommand(command, order); err != nil {
		return MutationReceipt{}, err
	}
	if err := s.deps.Authorizer.CanRequestCertificateValidation(ctx, command.Actor, order); err != nil {
		return MutationReceipt{}, err
	}
	return s.execute(ctx, command.Actor, command.Operation, func(txCtx context.Context, repo Repository) (MutationReceipt, error) {
		return repo.RequestCertificateValidation(txCtx, command)
	})
}

func (s applicationService) AddEvidenceReference(ctx context.Context, command AddEvidenceReferenceCommand) (MutationReceipt, error) {
	order, err := s.deps.Repository.GetWorkOrder(ctx, command.Actor, command.Evidence.WorkOrderID)
	if err != nil {
		return MutationReceipt{}, err
	}
	if err := ValidateAddEvidenceReferenceCommand(command, order); err != nil {
		return MutationReceipt{}, err
	}
	if err := s.deps.Authorizer.CanAddEvidenceReference(ctx, command.Actor, order, command.Evidence); err != nil {
		return MutationReceipt{}, err
	}
	return s.execute(ctx, command.Actor, command.Operation, func(txCtx context.Context, repo Repository) (MutationReceipt, error) {
		return repo.AddEvidenceReference(txCtx, command)
	})
}
