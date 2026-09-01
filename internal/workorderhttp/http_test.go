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
	command workorder.SubmitPartialCommand
}

func (s *testService) SubmitPartial(_ context.Context, c workorder.SubmitPartialCommand) (workorder.MutationReceipt, error) {
	s.command = c
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
func (s *testService) AddEvidenceReference(context.Context, workorder.AddEvidenceReferenceCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
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
	if service.command.Actor.ActorID != "derived" || service.command.Actor.Role != "inspector" {
		t.Fatalf("actor was not derived: %+v", service.command.Actor)
	}
}
func TestHandlerRejectsMissingBearer(t *testing.T) {
	response := httptest.NewRecorder()
	Handler{}.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", response.Code)
	}
}
