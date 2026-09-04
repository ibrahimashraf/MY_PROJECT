package workorderauth

import (
	"context"
	"errors"

	"integin/internal/domain/workorder"
)

var ErrDenied = errors.New("work-order authorization denied")

const (
	CapabilityCreateRequest        = "workorder.create_request"
	CapabilityAssignScope          = "workorder.assign_scope"
	CapabilityTransitionExecution  = "workorder.transition_execution"
	CapabilitySubmitPartial        = "workorder.submit_partial"
	CapabilityReassignScope        = "workorder.reassign_scope"
	CapabilityReconcileProvisional = "workorder.reconcile_provisional"
	CapabilityRequestValidation    = "workorder.request_certificate_validation"
	CapabilityAddEvidenceReference = "workorder.add_evidence_reference"
	CapabilityHandover             = "workorder.handover"
)

type Authorizer struct{}

func New() Authorizer { return Authorizer{} }

func (Authorizer) CanCreateRequest(_ context.Context, actor workorder.ActorContext) error {
	return authorize(actor, CapabilityCreateRequest, "manager", "administrator")
}

func (Authorizer) CanAssignScope(_ context.Context, actor workorder.ActorContext, order workorder.WorkOrder) error {
	return authorizeOrder(actor, order, CapabilityAssignScope, "manager", "administrator")
}

func (Authorizer) CanTransitionExecution(_ context.Context, actor workorder.ActorContext, order workorder.WorkOrder, _ workorder.ExecutionState) error {
	return authorizeOrder(actor, order, CapabilityTransitionExecution, "manager", "administrator")
}

func (Authorizer) CanSubmitPartial(_ context.Context, actor workorder.ActorContext, order workorder.WorkOrder, assignment workorder.Assignment) error {
	if err := authorizeOrder(actor, order, CapabilitySubmitPartial, "inspector", "administrator"); err != nil {
		return err
	}
	if actor.Role == "inspector" && assignment.InspectorID != actor.ActorID {
		return ErrDenied
	}
	return nil
}

func (Authorizer) CanReassignScope(_ context.Context, actor workorder.ActorContext, order workorder.WorkOrder) error {
	return authorizeOrder(actor, order, CapabilityReassignScope, "manager", "administrator")
}

func (Authorizer) CanReconcileProvisional(_ context.Context, actor workorder.ActorContext, record workorder.ProvisionalRecord) error {
	if actor.TenantID != record.TenantID || actor.OrganizationID != record.OrganizationID {
		return ErrDenied
	}
	return authorize(actor, CapabilityReconcileProvisional, "manager", "administrator")
}

func (Authorizer) CanRequestCertificateValidation(_ context.Context, actor workorder.ActorContext, order workorder.WorkOrder) error {
	return authorizeOrder(actor, order, CapabilityRequestValidation, "reviewer", "manager", "administrator")
}

func (Authorizer) CanAddEvidenceReference(_ context.Context, actor workorder.ActorContext, order workorder.WorkOrder, _ workorder.EvidenceReference) error {
	return authorizeOrder(actor, order, CapabilityAddEvidenceReference, "inspector", "manager", "administrator", "reviewer")
}

func (Authorizer) CanHandover(_ context.Context, actor workorder.ActorContext, order workorder.WorkOrder) error {
	return authorizeOrder(actor, order, CapabilityHandover, "inspector", "manager", "administrator")
}

func authorizeOrder(actor workorder.ActorContext, order workorder.WorkOrder, capability string, roles ...string) error {
	if actor.TenantID != order.TenantID || actor.OrganizationID != order.OrganizationID {
		return ErrDenied
	}
	return authorize(actor, capability, roles...)
}

func authorize(actor workorder.ActorContext, capability string, roles ...string) error {
	if !has(roles, actor.Role) || !has(actor.Capabilities, capability) {
		return ErrDenied
	}
	return nil
}

func has(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
