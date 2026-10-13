package workorderhttp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"integin/internal/identity"
	"integin/internal/oidcauth"
)

func TestHandoverHandlerDerivesActorAndExecutes(t *testing.T) {
	service := &testService{}
	handler := HandoverHandler{
		Validator: testValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}},
		Resolver: testResolver{membership: identity.Membership{
			ActorID:        "inspector-1",
			TenantID:       "tenant-a",
			OrganizationID: "org-a",
			WorkOrderRole:  "inspector",
			Capabilities:   []string{"workorder.handover"},
		}},
		Service: service,
	}

	body := `{"operation_id":"op-1","idempotency_key":"key-1","expected_revision":2,"handover_id":"ho-1","from_assignment_id":"as-1","to_assignment_id":"as-2","state":"requested","revision":1}`
	req := httptest.NewRequest(http.MethodPost, "/work-orders/wo-1/handovers", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"accepted"`) {
		t.Fatalf("expected accepted receipt, got %s", rec.Body.String())
	}

	// Also verify mounted prefix /work-orders/handovers/{id}
	req2 := httptest.NewRequest(http.MethodPost, "/work-orders/handovers/wo-1", strings.NewReader(body))
	req2.Header.Set("Authorization", "Bearer valid-token")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on prefix route, got %d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestHandoverHandlerRejectsInvalidPayloadAndMissingBearer(t *testing.T) {
	service := &testService{}
	handler := HandoverHandler{
		Validator: testValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}},
		Resolver: testResolver{membership: identity.Membership{
			ActorID:        "inspector-1",
			TenantID:       "tenant-a",
			OrganizationID: "org-a",
			WorkOrderRole:  "inspector",
			Capabilities:   []string{"workorder.handover"},
		}},
		Service: service,
	}

	// Missing bearer
	req := httptest.NewRequest(http.MethodPost, "/work-orders/wo-1/handovers", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for missing bearer, got %d", rec.Code)
	}

	// Unknown field in body
	req = httptest.NewRequest(http.MethodPost, "/work-orders/wo-1/handovers", strings.NewReader(`{"unknown_field":"value"}`))
	req.Header.Set("Authorization", "Bearer valid-token")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for unknown fields, got %d", rec.Code)
	}

	// Invalid path (not /work-orders/{id}/handovers)
	req = httptest.NewRequest(http.MethodPost, "/work-orders//handovers", strings.NewReader(`{"operation_id":"op-1","idempotency_key":"key-1","expected_revision":2,"handover_id":"ho-1","from_assignment_id":"as-1","to_assignment_id":"as-2","state":"requested","revision":1}`))
	req.Header.Set("Authorization", "Bearer valid-token")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for missing work order ID, got %d", rec.Code)
	}

	// Same from and to assignment
	sameAssignmentBody := fmt.Sprintf(`{"operation_id":"op-1","idempotency_key":"key-1","expected_revision":2,"handover_id":"ho-1","from_assignment_id":"as-1","to_assignment_id":"as-1","state":"requested","revision":1}`)
	req = httptest.NewRequest(http.MethodPost, "/work-orders/wo-1/handovers", strings.NewReader(sameAssignmentBody))
	req.Header.Set("Authorization", "Bearer valid-token")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request when from == to assignment, got %d", rec.Code)
	}
}
