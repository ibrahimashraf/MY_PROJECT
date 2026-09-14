package oidchttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/identity"
)

func TestTenantScopeConflictDetection(t *testing.T) {
	base := httptest.NewRequest(http.MethodGet, "/identity/session", nil)
	if tenantScopeConflict(base, "tenant-a", "org-a") {
		t.Fatal("absent scoping must not conflict")
	}
	if tenantScopeConflict(base, "tenant-a", "org-a") {
		t.Fatal("empty scoping must not conflict")
	}

	byHeader := httptest.NewRequest(http.MethodGet, "/identity/session", nil)
	byHeader.Header.Set("X-Tenant-ID", "tenant-a")
	byHeader.Header.Set("X-Organization-ID", "org-a")
	if tenantScopeConflict(byHeader, "tenant-a", "org-a") {
		t.Fatal("matching header scoping must not conflict")
	}

	conflictingHeader := httptest.NewRequest(http.MethodGet, "/identity/session", nil)
	conflictingHeader.Header.Set("X-Tenant-ID", "tenant-b")
	conflictingHeader.Header.Set("X-Organization-ID", "org-a")
	if !tenantScopeConflict(conflictingHeader, "tenant-a", "org-a") {
		t.Fatal("conflicting tenant header must conflict")
	}

	caseInsensitive := httptest.NewRequest(http.MethodGet, "/identity/session", nil)
	caseInsensitive.Header.Set("x-tenant-id", "tenant-b")
	if !tenantScopeConflict(caseInsensitive, "tenant-a", "org-a") {
		t.Fatal("tenant header must be matched case-insensitively")
	}

	conflictingOrg := httptest.NewRequest(http.MethodGet, "/identity/session", nil)
	conflictingOrg.Header.Set("X-Organization-ID", "org-b")
	if !tenantScopeConflict(conflictingOrg, "tenant-a", "org-a") {
		t.Fatal("conflicting organization header must conflict")
	}

	conflictingQuery := httptest.NewRequest(http.MethodGet, "/identity/session?tenant_id=tenant-c", nil)
	if !tenantScopeConflict(conflictingQuery, "tenant-a", "org-a") {
		t.Fatal("conflicting tenant query parameter must conflict")
	}

	matchingQuery := httptest.NewRequest(http.MethodGet, "/identity/session?tenant_id=tenant-a&organization_id=org-a", nil)
	if tenantScopeConflict(matchingQuery, "tenant-a", "org-a") {
		t.Fatal("matching query scoping must not conflict")
	}

	conflictingOrgQuery := httptest.NewRequest(http.MethodGet, "/identity/session?organization_id=org-c", nil)
	if !tenantScopeConflict(conflictingOrgQuery, "tenant-a", "org-a") {
		t.Fatal("conflicting organization query parameter must conflict")
	}
}

func TestRejectTenantScopeConflictFailsClosedOnSpoofedScope(t *testing.T) {
	var calls int
	next := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		calls++
		writer.WriteHeader(http.StatusOK)
	})
	org := identity.OrganizationContext{TenantID: "tenant-a", OrganizationID: "org-a"}
	request := httptest.NewRequest(http.MethodGet, "/secure", nil)
	request = request.WithContext(WithOrganizationContext(context.Background(), org))
	request.Header.Set("X-Tenant-ID", "tenant-attacker")
	response := httptest.NewRecorder()
	RejectTenantScopeConflict(next).ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("spoofed status=%d", response.Code)
	}
	if calls != 0 {
		t.Fatalf("spoofed scope must not reach the handler, calls=%d", calls)
	}
	var body map[string]string
	_ = json.Unmarshal(response.Body.Bytes(), &body)
	if body["error"] != "tenant_scope_conflict" {
		t.Fatalf("body=%v", body)
	}
}

func TestRejectTenantScopeConflictAllowsMatchingOrAbsentScope(t *testing.T) {
	for name, scoped := range map[string]func(*http.Request){
		"absent":    func(_ *http.Request) {},
		"matching":  func(r *http.Request) { r.Header.Set("X-Tenant-ID", "tenant-a") },
		"matchingq": func(r *http.Request) { r.URL.RawQuery = "tenant_id=tenant-a&organization_id=org-a" },
	} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			next := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				calls++
				writer.WriteHeader(http.StatusOK)
			})
			org := identity.OrganizationContext{TenantID: "tenant-a", OrganizationID: "org-a"}
			request := httptest.NewRequest(http.MethodGet, "/secure", nil)
			request = request.WithContext(WithOrganizationContext(context.Background(), org))
			scoped(request)
			response := httptest.NewRecorder()
			RejectTenantScopeConflict(next).ServeHTTP(response, request)
			if response.Code != http.StatusOK || calls != 1 {
				t.Fatalf("status=%d calls=%d", response.Code, calls)
			}
		})
	}
}

func TestRejectTenantScopeConflictPassesThroughWithoutBoundContext(t *testing.T) {
	calls := 0
	next := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		calls++
		writer.WriteHeader(http.StatusOK)
	})
	request := httptest.NewRequest(http.MethodGet, "/unauthenticated", nil)
	request.Header.Set("X-Tenant-ID", "tenant-b")
	response := httptest.NewRecorder()
	RejectTenantScopeConflict(next).ServeHTTP(response, request)
	if response.Code != http.StatusOK || calls != 1 {
		t.Fatalf("unauthenticated scoping must pass through for comparison, status=%d calls=%d", response.Code, calls)
	}
}
