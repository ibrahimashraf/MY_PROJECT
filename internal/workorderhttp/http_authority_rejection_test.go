package workorderhttp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"integin/internal/identity"
	"integin/internal/oidcauth"
)

func TestHandlerRejectsClientSuppliedAuthorityFields(t *testing.T) {
	service := &testService{}
	handler := Handler{
		Validator: testValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}},
		Resolver:  testResolver{membership: identity.Membership{ActorID: "derived", TenantID: "tenant", OrganizationID: "organization", WorkOrderRole: "inspector", Capabilities: []string{"workorder.submit_partial"}}},
		Service:   service,
	}
	request := httptest.NewRequest(http.MethodPost, "/work-orders/partial-submissions", strings.NewReader(`{"operation_id":"op","idempotency_key":"idem","expected_revision":1,"work_order_id":"wo","assignment_id":"assignment","inspection_ids":["inspection"],"tenant_id":"attacker"}`))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("authority-shaped field status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if service.command.WorkOrderID != "" {
		t.Fatalf("service received rejected authority-shaped request: %+v", service.command)
	}
}
