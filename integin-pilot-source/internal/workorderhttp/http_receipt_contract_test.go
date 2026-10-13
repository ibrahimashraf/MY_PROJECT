package workorderhttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"integin/internal/identity"
	"integin/internal/oidcauth"
)

func TestHandlerUsesSnakeCaseReceiptJSON(t *testing.T) {
	handler := Handler{Validator: testValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}}, Resolver: testResolver{membership: identity.Membership{ActorID: "actor", TenantID: "tenant", OrganizationID: "org", WorkOrderRole: "inspector", Capabilities: []string{"workorder.submit_partial"}}}, Service: &testService{}}
	request := httptest.NewRequest(http.MethodPost, "/work-orders/partial-submissions", strings.NewReader(`{"operation_id":"op","idempotency_key":"key","expected_revision":1,"work_order_id":"wo","assignment_id":"assignment","inspection_ids":["inspection"]}`))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := fields["Status"]; ok {
		t.Fatal("response uses Go field names instead of wire-format names")
	}
	if _, ok := fields["status"]; !ok {
		t.Fatal("response does not expose the stable status field")
	}
}
