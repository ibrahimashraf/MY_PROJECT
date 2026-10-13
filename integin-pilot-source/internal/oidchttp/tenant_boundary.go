// INTEGIN tenant boundary: client-supplied scoping can never override the server-resolved identity projection.
package oidchttp

import (
	"net/http"
	"strings"
)

const (
	tenantScopeHeader       = "X-Tenant-ID"
	organizationScopeHeader = "X-Organization-ID"
	tenantScopeQuery        = "tenant_id"
	organizationScopeQuery  = "organization_id"
)

// tenantScopeConflict reports whether the request carries client-supplied tenant or
// organization scoping that does not match the server-resolved values. Matching or
// absent scoping is harmless and passes; any conflicting scoping is a spoofing
// attempt and fails closed.
func tenantScopeConflict(request *http.Request, tenantID, organizationID string) bool {
	for key, expected := range map[string]string{
		tenantScopeHeader:       tenantID,
		organizationScopeHeader: organizationID,
		tenantScopeQuery:        tenantID,
		organizationScopeQuery:  organizationID,
	} {
		var supplied string
		if strings.HasPrefix(key, "X-") {
			supplied = strings.TrimSpace(request.Header.Get(key))
		} else {
			supplied = strings.TrimSpace(request.URL.Query().Get(key))
		}
		if supplied != "" && supplied != expected {
			return true
		}
	}
	return false
}

// RejectTenantScopeConflict returns middleware that fails closed when an authenticated
// request carries tenant or organization scoping that conflicts with the organization
// context bound by SessionAuthenticator. The PostgreSQL session context
// (integin.tenant_id, integin.organization_id) is derived exclusively from the resolved
// issuer-subject membership, so competing client scoping is never honored.
func RejectTenantScopeConflict(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		org, ok := OrganizationContextFrom(request.Context())
		if ok && tenantScopeConflict(request, org.TenantID, org.OrganizationID) {
			writeFailure(writer, http.StatusForbidden, "tenant_scope_conflict")
			return
		}
		next.ServeHTTP(writer, request)
	})
}
