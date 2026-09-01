package workorder

import (
	"context"
	"strings"
)

// ActorContext is supplied by the server transport and never trusted from a Field cache.
type ActorContext struct {
	TenantID       string
	OrganizationID string
	ActorID        string
	Role           string
	Capabilities   []string
}

func (a ActorContext) Validate() error {
	if a.TenantID == "" || a.OrganizationID == "" || a.ActorID == "" || a.Role == "" {
		return ErrInvalidIdentity
	}
	return nil
}

type CreateRequestCommand struct {
	Actor     ActorContext
	Operation OperationMeta
	WorkOrder WorkOrder
}

type AssignScopeCommand struct {
	Actor       ActorContext
	Operation   OperationMeta
	WorkOrderID string
	Assignment  Assignment
}

type TransitionExecutionCommand struct {
	Actor       ActorContext
	Operation   OperationMeta
	WorkOrderID string
	From        ExecutionState
	To          ExecutionState
}

type SubmitPartialCommand struct {
	Actor         ActorContext
	Operation     OperationMeta
	WorkOrderID   string
	AssignmentID  string
	InspectionIDs []string
}

type ReassignScopeCommand struct {
	Actor        ActorContext
	Operation    OperationMeta
	WorkOrderID  string
	Previous     Assignment
	Replacement  Assignment
	ScopeItemIDs []string
}

type ReconcileProvisionalCommand struct {
	Actor       ActorContext
	Operation   OperationMeta
	Record      ProvisionalRecord
	Outcome     string
	CanonicalID string
}

type RequestCertificateValidationCommand struct {
	Actor         ActorContext
	Operation     OperationMeta
	WorkOrderID   string
	InspectionIDs []string
}

type AddEvidenceReferenceCommand struct {
	Actor     ActorContext
	Operation OperationMeta
	Evidence  EvidenceReference
}

type MutationReceipt struct {
	OperationID    string `json:"operation_id"`
	IdempotencyKey string `json:"idempotency_key"`
	TenantID       string `json:"tenant_id"`
	WorkOrderID    string `json:"work_order_id"`
	Revision       int64  `json:"revision"`
	Status         string `json:"status"`
}

const (
	ReceiptAccepted = "accepted"
	ReceiptConflict = "conflict"
	ReceiptRejected = "rejected"
)

// Service is the application boundary for work-order mutations. Implementations must enforce
// tenant scope, authorization, idempotency, revision checks, and transactional persistence.
type Service interface {
	CreateRequest(context.Context, CreateRequestCommand) (MutationReceipt, error)
	AssignScope(context.Context, AssignScopeCommand) (MutationReceipt, error)
	TransitionExecution(context.Context, TransitionExecutionCommand) (MutationReceipt, error)
	SubmitPartial(context.Context, SubmitPartialCommand) (MutationReceipt, error)
	ReassignScope(context.Context, ReassignScopeCommand) (MutationReceipt, error)
	ReconcileProvisional(context.Context, ReconcileProvisionalCommand) (MutationReceipt, error)
	RequestCertificateValidation(context.Context, RequestCertificateValidationCommand) (MutationReceipt, error)
	AddEvidenceReference(context.Context, AddEvidenceReferenceCommand) (MutationReceipt, error)
}

func validateActorOperation(actor ActorContext, operation OperationMeta) error {
	if err := actor.Validate(); err != nil {
		return err
	}
	return operation.Validate()
}

func ValidateCreateRequestCommand(command CreateRequestCommand) error {
	if err := validateActorOperation(command.Actor, command.Operation); err != nil {
		return err
	}
	if command.Actor.TenantID != command.WorkOrder.TenantID || command.Actor.OrganizationID != command.WorkOrder.OrganizationID {
		return ErrInvalidIdentity
	}
	return command.WorkOrder.ValidateIdentity()
}

func ValidateAssignScopeCommand(command AssignScopeCommand, order WorkOrder) error {
	if err := validateActorOperation(command.Actor, command.Operation); err != nil {
		return err
	}
	if command.Actor.TenantID != order.TenantID || command.Actor.OrganizationID != order.OrganizationID || command.WorkOrderID != order.ID {
		return ErrInvalidAssignment
	}
	if order.ExecutionState != ExecutionReady {
		return ErrInvalidTransition
	}
	return command.Assignment.ValidateFor(order)
}

func ValidateTransitionExecutionCommand(command TransitionExecutionCommand, order WorkOrder) error {
	if err := validateActorOperation(command.Actor, command.Operation); err != nil {
		return err
	}
	if command.Actor.TenantID != order.TenantID || command.Actor.OrganizationID != order.OrganizationID || command.WorkOrderID != order.ID || command.From != order.ExecutionState {
		return ErrInvalidTransition
	}
	if !CanTransition(command.From, command.To) {
		return ErrInvalidTransition
	}
	return nil
}

func ValidateSubmitPartialCommand(command SubmitPartialCommand, order WorkOrder) error {
	if err := validateActorOperation(command.Actor, command.Operation); err != nil {
		return err
	}
	if command.Actor.TenantID != order.TenantID || command.Actor.OrganizationID != order.OrganizationID || command.WorkOrderID != order.ID || command.AssignmentID == "" || len(command.InspectionIDs) == 0 {
		return ErrInvalidScope
	}
	if !CanTransition(order.ExecutionState, ExecutionPartiallySubmitted) {
		return ErrInvalidTransition
	}
	seen := make(map[string]struct{}, len(command.InspectionIDs))
	for _, inspectionID := range command.InspectionIDs {
		inspectionID = strings.TrimSpace(inspectionID)
		if inspectionID == "" {
			return ErrInvalidScope
		}
		if _, exists := seen[inspectionID]; exists {
			return ErrInvalidScope
		}
		seen[inspectionID] = struct{}{}
	}
	return nil
}

func ValidateReassignScopeCommand(command ReassignScopeCommand, order WorkOrder) error {
	if err := validateActorOperation(command.Actor, command.Operation); err != nil {
		return err
	}
	if command.Actor.TenantID != order.TenantID || command.Actor.OrganizationID != order.OrganizationID || command.WorkOrderID != order.ID || len(command.ScopeItemIDs) == 0 {
		return ErrInvalidAssignment
	}
	if err := command.Previous.ValidateFor(order); err != nil {
		return err
	}
	if err := command.Replacement.ValidateFor(order); err != nil {
		return err
	}
	if len(command.ScopeItemIDs) != len(command.Replacement.ScopeItemIDs) {
		return ErrInvalidAssignment
	}
	replacementScope := make(map[string]struct{}, len(command.Replacement.ScopeItemIDs))
	for _, id := range command.Replacement.ScopeItemIDs {
		replacementScope[id] = struct{}{}
	}
	for _, id := range command.ScopeItemIDs {
		if _, ok := replacementScope[id]; !ok {
			return ErrInvalidAssignment
		}
	}
	return nil
}

func ValidateReconcileProvisionalCommand(command ReconcileProvisionalCommand) error {
	if err := validateActorOperation(command.Actor, command.Operation); err != nil {
		return err
	}
	if command.Actor.TenantID != command.Record.TenantID || command.Actor.OrganizationID != command.Record.OrganizationID {
		return ErrInvalidIdentity
	}
	if err := command.Record.Validate(); err != nil {
		return err
	}
	switch command.Outcome {
	case ReconcileMatched, ReconcileCreated:
		if command.CanonicalID == "" {
			return ErrInvalidIdentity
		}
	case ReconcileConflict, ReconcileRejected:
		if command.CanonicalID != "" {
			return ErrInvalidIdentity
		}
	default:
		return ErrInvalidIdentity
	}
	return nil
}

func ValidateCertificateInspectionState(lifecycleState, finalizationState string) error {
	if lifecycleState != "COMPLETED" || finalizationState != "SUBMITTED" {
		return ErrInvalidScope
	}
	return nil
}

func ValidateCertificateValidationCommand(command RequestCertificateValidationCommand, order WorkOrder) error {
	if err := validateActorOperation(command.Actor, command.Operation); err != nil {
		return err
	}
	if command.Actor.TenantID != order.TenantID || command.Actor.OrganizationID != order.OrganizationID || command.WorkOrderID != order.ID || len(command.InspectionIDs) == 0 {
		return ErrInvalidScope
	}
	seen := make(map[string]struct{}, len(command.InspectionIDs))
	for _, inspectionID := range command.InspectionIDs {
		if _, exists := seen[inspectionID]; exists {
			return ErrInvalidScope
		}
		seen[inspectionID] = struct{}{}
	}
	return nil
}

func ValidateAddEvidenceReferenceCommand(command AddEvidenceReferenceCommand, order WorkOrder) error {
	if err := validateActorOperation(command.Actor, command.Operation); err != nil {
		return err
	}
	if command.Actor.TenantID != order.TenantID || command.Actor.OrganizationID != order.OrganizationID {
		return ErrInvalidEvidence
	}
	return command.Evidence.ValidateFor(order)
}
