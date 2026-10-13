package openapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"integin/internal/platform/featureflag"
	"integin/internal/shared/featureflags"
	"integin/pkg/httputil"
)

// TokenStore maps opaque token -> tenant scope. No GRANT change; every handler
// still calls set_config('integin.tenant_id') and checks assignment.state='active'.
type TokenStore interface {
	Resolve(token string) (tenantID, organizationID string, ok bool)
}

type Handler struct {
	Store TokenStore
	Flags *featureflag.Service
	Now   func() time.Time
}

// ServeHTTP is read-only, tenant-scoped. Enforces FlagOpenAPI + RLS via caller.
// Caller must still call evidencepg.RegisterForActiveAssignment / certificatepg.VerifyPublic
// which enforce assignment.state and FORCE RLS. This layer only gates the API.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Now == nil {
		h.Now = time.Now
	}
	if h.Flags != nil && h.Flags.Evaluate(featureflags.FlagOpenAPI, featureflag.Request{}, h.Now().UTC()) != featureflags.Enabled {
		httputil.WriteProblem(w, r, http.StatusForbidden, "Forbidden", "Open API feature is disabled.")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		httputil.WriteProblem(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "Allowed method: GET")
		return
	}
	token := strings.TrimSpace(r.Header.Get("X-API-Token"))
	if token == "" {
		httputil.WriteProblem(w, r, http.StatusUnauthorized, "Unauthorized", "Missing required X-API-Token header.")
		return
	}
	if h.Store == nil {
		httputil.WriteProblem(w, r, http.StatusServiceUnavailable, "Service Unavailable", "Token store service is unavailable.")
		return
	}
	tenantID, orgID, ok := h.Store.Resolve(token)
	if !ok || strings.TrimSpace(tenantID) == "" || strings.TrimSpace(orgID) == "" {
		httputil.WriteProblem(w, r, http.StatusUnauthorized, "Unauthorized", "Invalid API token.")
		return
	}
	// Echo scope; caller can set_config from here. No cross-tenant read.
	w.Header().Set("X-Tenant-ID", tenantID)
	w.Header().Set("X-Organization-ID", orgID)
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]string{"tenant_id": tenantID, "organization_id": orgID, "mode": "readonly"})
}
