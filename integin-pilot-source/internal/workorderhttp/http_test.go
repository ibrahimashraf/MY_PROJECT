package workorderhttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"integin/internal/domain/workorder"
	"integin/internal/identity"
	"integin/internal/oidcauth"
)

type testValidator struct {
	principal oidcauth.Principal
	err       error
}

func (v testValidator) Validate(context.Context, string) (oidcauth.Principal, error) {
	return v.principal, v.err
}

type testResolver struct {
	membership identity.Membership
	err        error
}

func (r testResolver) Resolve(context.Context, identity.PrincipalKey) (identity.Membership, error) {
	return r.membership, r.err
}

type testService struct {
	partialCommand  workorder.SubmitPartialCommand
	evidenceCommand workorder.AddEvidenceReferenceCommand
}

func (s *testService) SubmitPartial(_ context.Context, c workorder.SubmitPartialCommand) (workorder.MutationReceipt, error) {
	s.partialCommand = c
	return workorder.MutationReceipt{Status: workorder.ReceiptAccepted}, nil
}
func (s *testService) CreateRequest(context.Context, workorder.CreateRequestCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (s *testService) AssignScope(context.Context, workorder.AssignScopeCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (s *testService) TransitionExecution(context.Context, workorder.TransitionExecutionCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (s *testService) ReassignScope(context.Context, workorder.ReassignScopeCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (s *testService) ReconcileProvisional(context.Context, workorder.ReconcileProvisionalCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (s *testService) RequestCertificateValidation(context.Context, workorder.RequestCertificateValidationCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (s *testService) AddEvidenceReference(_ context.Context, c workorder.AddEvidenceReferenceCommand) (workorder.MutationReceipt, error) {
	s.evidenceCommand = c
	return workorder.MutationReceipt{Status: workorder.ReceiptAccepted}, nil
}
func TestHandlerDerivesActorAndIgnoresBodyAuthority(t *testing.T) {
	service := &testService{}
	handler := Handler{Validator: testValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}}, Resolver: testResolver{membership: identity.Membership{ActorID: "derived", TenantID: "tenant", OrganizationID: "org", WorkOrderRole: "inspector", Capabilities: []string{"workorder.submit_partial"}}}, Service: service}
	request := httptest.NewRequest(http.MethodPost, "/work-orders/partial-submissions", strings.NewReader(`{"operation_id":"op","idempotency_key":"key","expected_revision":3,"work_order_id":"wo","assignment_id":"a","inspection_ids":["i"]}`))
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
	if service.partialCommand.Actor.ActorID != "derived" || service.partialCommand.Actor.Role != "inspector" {
		t.Fatalf("actor was not derived: %+v", service.partialCommand.Actor)
	}
}
func TestHandlerRejectsMissingBearer(t *testing.T) {
	response := httptest.NewRecorder()
	Handler{}.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", response.Code)
	}
}

func TestEvidenceHandlerDerivesActorAndIgnoresBodyAuthority(t *testing.T) {
	service := &testService{}
	handler := EvidenceHandler{Validator: testValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}}, Resolver: testResolver{membership: identity.Membership{ActorID: "derived", TenantID: "tenant", OrganizationID: "org", WorkOrderRole: "inspector", Capabilities: []string{"workorder.add_evidence_reference"}}}, Service: service}
	attackerBody := `{"operation_id":"op","idempotency_key":"key","expected_revision":3,"evidence_id":"ev","content_hash":"` + strings.Repeat("a", 64) + `","reference_url":"https://example.com/ev","tenant_id":"attacker","organization_id":"attacker","actor_id":"attacker"}`
	request := httptest.NewRequest(http.MethodPost, "/work-orders/wo/evidence", strings.NewReader(attackerBody))
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("authority-shaped request status = %d body=%s, want %d", response.Code, response.Body.String(), http.StatusBadRequest)
	}
	validBody := `{"operation_id":"op","idempotency_key":"key","expected_revision":3,"evidence_id":"ev","content_hash":"` + strings.Repeat("a", 64) + `","reference_url":"https://example.com/ev"}`
	request = httptest.NewRequest(http.MethodPost, "/work-orders/wo/evidence", strings.NewReader(validBody))
	request.Header.Set("Authorization", "Bearer valid")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if service.evidenceCommand.Actor.ActorID != "derived" || service.evidenceCommand.Actor.Role != "inspector" {
		t.Fatalf("actor was not derived: %+v", service.evidenceCommand.Actor)
	}
	if service.evidenceCommand.Evidence.TenantID != "tenant" || service.evidenceCommand.Evidence.OrganizationID != "org" || service.evidenceCommand.Evidence.WorkOrderID != "wo" || service.evidenceCommand.Evidence.CreatedBy != "derived" {
		t.Fatalf("evidence was not derived: %+v", service.evidenceCommand.Evidence)
	}
}

func TestEvidenceHandlerRejectsMissingBearer(t *testing.T) {
	response := httptest.NewRecorder()
	EvidenceHandler{}.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/work-orders/wo/evidence", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", response.Code)
	}
}
