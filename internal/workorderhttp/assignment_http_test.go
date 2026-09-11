package workorderhttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/workorder"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/workorderauth"
)

type assignTestService struct {
	assignCommand workorder.AssignScopeCommand
}

func (s *assignTestService) AssignScope(_ context.Context, cmd workorder.AssignScopeCommand) (workorder.MutationReceipt, error) {
	s.assignCommand = cmd
	return workorder.MutationReceipt{
		OperationID:    cmd.Operation.OperationID,
		IdempotencyKey: cmd.Operation.IdempotencyKey,
		TenantID:       cmd.Actor.TenantID,
		WorkOrderID:    cmd.WorkOrderID,
		Revision:       1,
		Status:         workorder.ReceiptAccepted,
	}, nil
}
func (s *assignTestService) CreateRequest(context.Context, workorder.CreateRequestCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (s *assignTestService) TransitionExecution(context.Context, workorder.TransitionExecutionCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (s *assignTestService) SubmitPartial(context.Context, workorder.SubmitPartialCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (s *assignTestService) ReassignScope(context.Context, workorder.ReassignScopeCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (s *assignTestService) ReconcileProvisional(context.Context, workorder.ReconcileProvisionalCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (s *assignTestService) RequestCertificateValidation(context.Context, workorder.RequestCertificateValidationCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (s *assignTestService) AddEvidenceReference(context.Context, workorder.AddEvidenceReferenceCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}

func TestAssignmentHandlerRejectsUnverifiedCompetency(t *testing.T) {
	service := &assignTestService{}
	handler := AssignmentHandler{
		Validator: testValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}},
		Resolver: testResolver{membership: identity.Membership{
			ActorID:        "dispatcher-1",
			TenantID:       "tenant-1",
			OrganizationID: "org-1",
			WorkOrderRole:  "manager",
			Capabilities:   []string{workorderauth.CapabilityAssignScope},
		}},
		Service: service,
		Now: func() time.Time {
			return time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
		},
	}

	// Payload with unapproved credential
	body := `{
		"operation_id":"op-1",
		"idempotency_key":"key-1",
		"expected_revision":1,
		"assignment_id":"asgn-1",
		"inspector_id":"insp-1",
		"scope_item_ids":["scope-1"],
		"required_skills":["lifting.inspection"],
		"credential":{
			"credential_id":"cred-1",
			"inspector_id":"insp-1",
			"verification_status":"PENDING_QA_REVIEW"
		},
		"competencies":[]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/work-orders/wo-100/assign", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if service.assignCommand.WorkOrderID != "" {
		t.Fatalf("service received assignment despite failed competency check: %+v", service.assignCommand)
	}
}

func TestAssignmentHandlerAllowsApprovedAndCompetentInspector(t *testing.T) {
	service := &assignTestService{}
	fixedNow := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	handler := AssignmentHandler{
		Validator: testValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}},
		Resolver: testResolver{membership: identity.Membership{
			ActorID:        "dispatcher-1",
			TenantID:       "tenant-1",
			OrganizationID: "org-1",
			WorkOrderRole:  "manager",
			Capabilities:   []string{workorderauth.CapabilityAssignScope},
		}},
		Service: service,
		Now:     func() time.Time { return fixedNow },
	}

	body := `{
		"operation_id":"op-1",
		"idempotency_key":"key-1",
		"expected_revision":1,
		"assignment_id":"asgn-1",
		"inspector_id":"insp-1",
		"scope_item_ids":["scope-1"],
		"required_skills":["lifting.inspection"],
		"credential":{
			"credential_id":"cred-1",
			"inspector_id":"insp-1",
			"verification_status":"APPROVED_BY_TECHNICAL_DIRECTOR",
			"valid_from":"2026-01-01T00:00:00Z",
			"expires_at":"2027-01-01T00:00:00Z"
		},
		"competencies":[
			{
				"id":"comp-1",
				"tenant_id":"tenant-1",
				"organization_id":"org-1",
				"technician_id":"insp-1",
				"equipment_type_id":"lifting.inspection",
				"status":"CURRENT",
				"expires_at":"2027-01-01T00:00:00Z"
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/work-orders/wo-100/assign", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if service.assignCommand.WorkOrderID != "wo-100" {
		t.Fatalf("expected WorkOrderID 'wo-100', got %q", service.assignCommand.WorkOrderID)
	}
	if service.assignCommand.Assignment.InspectorID != "insp-1" {
		t.Fatalf("expected InspectorID 'insp-1', got %q", service.assignCommand.Assignment.InspectorID)
	}
}
