package workorderauth

import (
	"context"
	"errors"
	"testing"

	"integin/internal/domain/workorder"
)

func TestAuthorizerAllowsCurrentRoleCapabilityMatrix(t *testing.T) {
	ctx := context.Background()
	authorizer := New()
	order := workorder.WorkOrder{TenantID: "tenant", OrganizationID: "org"}
	assignment := workorder.Assignment{InspectorID: "inspector-1"}
	record := workorder.ProvisionalRecord{TenantID: "tenant", OrganizationID: "org"}

	tests := []struct {
		name  string
		actor workorder.ActorContext
		allow func(workorder.ActorContext) error
	}{
		{"create request", matrixActor("manager", CapabilityCreateRequest, "manager-1"), func(actor workorder.ActorContext) error {
			return authorizer.CanCreateRequest(ctx, actor)
		}},
		{"assign scope", matrixActor("manager", CapabilityAssignScope, "manager-1"), func(actor workorder.ActorContext) error {
			return authorizer.CanAssignScope(ctx, actor, order)
		}},
		{"transition execution", matrixActor("manager", CapabilityTransitionExecution, "manager-1"), func(actor workorder.ActorContext) error {
			return authorizer.CanTransitionExecution(ctx, actor, order, workorder.ExecutionInProgress)
		}},
		{"submit partial as inspector", matrixActor("inspector", CapabilitySubmitPartial, "inspector-1"), func(actor workorder.ActorContext) error {
			return authorizer.CanSubmitPartial(ctx, actor, order, assignment)
		}},
		{"reassign scope", matrixActor("manager", CapabilityReassignScope, "manager-1"), func(actor workorder.ActorContext) error {
			return authorizer.CanReassignScope(ctx, actor, order)
		}},
		{"reconcile provisional", matrixActor("manager", CapabilityReconcileProvisional, "manager-1"), func(actor workorder.ActorContext) error {
			return authorizer.CanReconcileProvisional(ctx, actor, record)
		}},
		{"request certificate validation as reviewer", matrixActor("reviewer", CapabilityRequestValidation, "reviewer-1"), func(actor workorder.ActorContext) error {
			return authorizer.CanRequestCertificateValidation(ctx, actor, order)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.allow(test.actor); err != nil {
				t.Fatalf("current matrix allowed case denied: %v", err)
			}
		})
	}
}

func TestAuthorizerDeniesWrongRoleAndMissingCapability(t *testing.T) {
	ctx := context.Background()
	authorizer := New()
	order := workorder.WorkOrder{TenantID: "tenant", OrganizationID: "org"}
	assignment := workorder.Assignment{InspectorID: "inspector-1"}
	record := workorder.ProvisionalRecord{TenantID: "tenant", OrganizationID: "org"}

	tests := []struct {
		name        string
		allowedRole string
		capability  string
		allow       func(workorder.ActorContext) error
	}{
		{"create request", "manager", CapabilityCreateRequest, func(actor workorder.ActorContext) error {
			return authorizer.CanCreateRequest(ctx, actor)
		}},
		{"assign scope", "manager", CapabilityAssignScope, func(actor workorder.ActorContext) error {
			return authorizer.CanAssignScope(ctx, actor, order)
		}},
		{"transition execution", "manager", CapabilityTransitionExecution, func(actor workorder.ActorContext) error {
			return authorizer.CanTransitionExecution(ctx, actor, order, workorder.ExecutionInProgress)
		}},
		{"submit partial", "inspector", CapabilitySubmitPartial, func(actor workorder.ActorContext) error {
			return authorizer.CanSubmitPartial(ctx, actor, order, assignment)
		}},
		{"reassign scope", "manager", CapabilityReassignScope, func(actor workorder.ActorContext) error {
			return authorizer.CanReassignScope(ctx, actor, order)
		}},
		{"reconcile provisional", "manager", CapabilityReconcileProvisional, func(actor workorder.ActorContext) error {
			return authorizer.CanReconcileProvisional(ctx, actor, record)
		}},
		{"request certificate validation", "reviewer", CapabilityRequestValidation, func(actor workorder.ActorContext) error {
			return authorizer.CanRequestCertificateValidation(ctx, actor, order)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wrongRole := matrixActor("inspector", test.capability, "actor-1")
			if test.allowedRole == "inspector" {
				wrongRole = matrixActor("manager", test.capability, "actor-1")
			}
			if err := test.allow(wrongRole); !errors.Is(err, ErrDenied) {
				t.Fatalf("expected wrong role to be denied, got %v", err)
			}
			missingCapability := matrixActor(test.allowedRole, "", "actor-1")
			if test.allowedRole == "inspector" {
				missingCapability.ActorID = assignment.InspectorID
			}
			if err := test.allow(missingCapability); !errors.Is(err, ErrDenied) {
				t.Fatalf("expected missing capability to be denied, got %v", err)
			}
		})
	}
}

func TestAuthorizerDeniesOrderScopeMismatch(t *testing.T) {
	ctx := context.Background()
	authorizer := New()
	order := workorder.WorkOrder{TenantID: "tenant", OrganizationID: "org"}
	assignment := workorder.Assignment{InspectorID: "inspector-1"}

	tests := []struct {
		name  string
		actor workorder.ActorContext
		allow func(workorder.ActorContext) error
	}{
		{"assign scope", matrixActor("manager", CapabilityAssignScope, "manager-1"), func(actor workorder.ActorContext) error {
			return authorizer.CanAssignScope(ctx, actor, order)
		}},
		{"transition execution", matrixActor("manager", CapabilityTransitionExecution, "manager-1"), func(actor workorder.ActorContext) error {
			return authorizer.CanTransitionExecution(ctx, actor, order, workorder.ExecutionInProgress)
		}},
		{"submit partial", matrixActor("inspector", CapabilitySubmitPartial, assignment.InspectorID), func(actor workorder.ActorContext) error {
			return authorizer.CanSubmitPartial(ctx, actor, order, assignment)
		}},
		{"reassign scope", matrixActor("manager", CapabilityReassignScope, "manager-1"), func(actor workorder.ActorContext) error {
			return authorizer.CanReassignScope(ctx, actor, order)
		}},
		{"request certificate validation", matrixActor("reviewer", CapabilityRequestValidation, "reviewer-1"), func(actor workorder.ActorContext) error {
			return authorizer.CanRequestCertificateValidation(ctx, actor, order)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wrongTenant := test.actor
			wrongTenant.TenantID = "other-tenant"
			if err := test.allow(wrongTenant); !errors.Is(err, ErrDenied) {
				t.Fatalf("expected cross-tenant denial, got %v", err)
			}
			wrongOrganization := test.actor
			wrongOrganization.OrganizationID = "other-org"
			if err := test.allow(wrongOrganization); !errors.Is(err, ErrDenied) {
				t.Fatalf("expected cross-organization denial, got %v", err)
			}
		})
	}
}

func TestAdministratorCanSubmitPartialWithoutAssignmentOwnership(t *testing.T) {
	actor := matrixActor("administrator", CapabilitySubmitPartial, "administrator-1")
	order := workorder.WorkOrder{TenantID: actor.TenantID, OrganizationID: actor.OrganizationID}
	assignment := workorder.Assignment{InspectorID: "different-inspector"}
	if err := New().CanSubmitPartial(context.Background(), actor, order, assignment); err != nil {
		t.Fatalf("current administrator bypass characterization was denied: %v", err)
	}
}

func matrixActor(role, capability, actorID string) workorder.ActorContext {
	capabilities := []string(nil)
	if capability != "" {
		capabilities = []string{capability}
	}
	return workorder.ActorContext{
		TenantID:       "tenant",
		OrganizationID: "org",
		ActorID:        actorID,
		Role:           role,
		Capabilities:   capabilities,
	}
}

func TestAuthorizerAllowsAllCurrentRoleCombinations(t *testing.T) {
	ctx := context.Background()
	authorizer := New()
	order := workorder.WorkOrder{TenantID: "tenant", OrganizationID: "org"}
	assignment := workorder.Assignment{InspectorID: "inspector-1"}
	record := workorder.ProvisionalRecord{TenantID: "tenant", OrganizationID: "org"}

	tests := []struct {
		name   string
		actors []workorder.ActorContext
		allow  func(workorder.ActorContext) error
	}{
		{"create request", []workorder.ActorContext{
			matrixActor("manager", CapabilityCreateRequest, "manager-1"),
			matrixActor("administrator", CapabilityCreateRequest, "administrator-1"),
		}, func(actor workorder.ActorContext) error { return authorizer.CanCreateRequest(ctx, actor) }},
		{"assign scope", []workorder.ActorContext{
			matrixActor("manager", CapabilityAssignScope, "manager-1"),
			matrixActor("administrator", CapabilityAssignScope, "administrator-1"),
		}, func(actor workorder.ActorContext) error { return authorizer.CanAssignScope(ctx, actor, order) }},
		{"transition execution", []workorder.ActorContext{
			matrixActor("manager", CapabilityTransitionExecution, "manager-1"),
			matrixActor("administrator", CapabilityTransitionExecution, "administrator-1"),
		}, func(actor workorder.ActorContext) error {
			return authorizer.CanTransitionExecution(ctx, actor, order, workorder.ExecutionInProgress)
		}},
		{"submit partial", []workorder.ActorContext{
			matrixActor("inspector", CapabilitySubmitPartial, assignment.InspectorID),
			matrixActor("administrator", CapabilitySubmitPartial, "administrator-1"),
		}, func(actor workorder.ActorContext) error {
			return authorizer.CanSubmitPartial(ctx, actor, order, assignment)
		}},
		{"reassign scope", []workorder.ActorContext{
			matrixActor("manager", CapabilityReassignScope, "manager-1"),
			matrixActor("administrator", CapabilityReassignScope, "administrator-1"),
		}, func(actor workorder.ActorContext) error { return authorizer.CanReassignScope(ctx, actor, order) }},
		{"reconcile provisional", []workorder.ActorContext{
			matrixActor("manager", CapabilityReconcileProvisional, "manager-1"),
			matrixActor("administrator", CapabilityReconcileProvisional, "administrator-1"),
		}, func(actor workorder.ActorContext) error {
			return authorizer.CanReconcileProvisional(ctx, actor, record)
		}},
		{"request certificate validation", []workorder.ActorContext{
			matrixActor("reviewer", CapabilityRequestValidation, "reviewer-1"),
			matrixActor("manager", CapabilityRequestValidation, "manager-1"),
			matrixActor("administrator", CapabilityRequestValidation, "administrator-1"),
		}, func(actor workorder.ActorContext) error {
			return authorizer.CanRequestCertificateValidation(ctx, actor, order)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, actor := range test.actors {
				if err := test.allow(actor); err != nil {
					t.Fatalf("current matrix allowed case denied for role %q: %v", actor.Role, err)
				}
			}
		})
	}
}

func TestAuthorizerDeniesProvisionalScopeMismatch(t *testing.T) {
	ctx := context.Background()
	authorizer := New()
	record := workorder.ProvisionalRecord{TenantID: "tenant", OrganizationID: "org"}
	actor := matrixActor("manager", CapabilityReconcileProvisional, "manager-1")
	wrongTenant := actor
	wrongTenant.TenantID = "other-tenant"
	if err := authorizer.CanReconcileProvisional(ctx, wrongTenant, record); !errors.Is(err, ErrDenied) {
		t.Fatalf("expected cross-tenant provisional denial, got %v", err)
	}
	wrongOrganization := actor
	wrongOrganization.OrganizationID = "other-org"
	if err := authorizer.CanReconcileProvisional(ctx, wrongOrganization, record); !errors.Is(err, ErrDenied) {
		t.Fatalf("expected cross-organization provisional denial, got %v", err)
	}
}
