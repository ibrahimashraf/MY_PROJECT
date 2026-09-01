package flaghttp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"integin/internal/flagpg"
	"integin/internal/shared/featureflags"
	"integin/internal/shared/httpresponse"
)

type FlagRepository interface {
	ListOverrides(ctx context.Context, tenantID, orgID string) ([]flagpg.OverrideRecord, error)
	GetOverride(ctx context.Context, tenantID, orgID string, key featureflags.Key, scope featureflags.Scope, scopeID string) (flagpg.OverrideRecord, bool, error)
	UpsertOverride(ctx context.Context, tenantID, orgID string, rec flagpg.OverrideRecord) (flagpg.OverrideRecord, error)
	DeleteOverride(ctx context.Context, tenantID, orgID string, key featureflags.Key, scope featureflags.Scope, scopeID string) error
}

type Handler struct {
	Repository FlagRepository
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/feature-flags":
		h.handleList(w, r)
	case r.Method == http.MethodPut && r.URL.Path == "/api/v1/admin/feature-flags":
		h.handleUpsert(w, r)
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/admin/feature-flags/"):
		h.handleDelete(w, r)
	default:
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
	}
}

func (h Handler) handleList(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	orgID := r.Header.Get("X-Organization-ID")
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing tenant or organization header")
		return
	}
	overrides, err := h.Repository.ListOverrides(r.Context(), tenantID, orgID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if overrides == nil {
		overrides = []flagpg.OverrideRecord{}
	}
	httpresponse.JSON(w, http.StatusOK, overrides)
}

func (h Handler) handleUpsert(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	orgID := r.Header.Get("X-Organization-ID")
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing tenant or organization header")
		return
	}
	var req struct {
		FlagKey   string             `json:"flag_key"`
		Scope     featureflags.Scope `json:"scope"`
		ScopeID   string             `json:"scope_id"`
		State     featureflags.State `json:"state"`
		Reason    string             `json:"reason,omitempty"`
		ExpiresAt *string            `json:"expires_at,omitempty"`
		ActorID   string             `json:"actor_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if req.FlagKey == "" || req.Scope == "" || req.ScopeID == "" || req.State == "" || req.ActorID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "flag_key, scope, scope_id, state, and actor_id are required")
		return
	}

	rec := flagpg.OverrideRecord{
		TenantID:       tenantID,
		OrganizationID: orgID,
		FlagKey:        featureflags.Key(req.FlagKey),
		Scope:          req.Scope,
		ScopeID:        req.ScopeID,
		State:          req.State,
		Reason:         req.Reason,
		CreatedBy:      req.ActorID,
	}

	result, err := h.Repository.UpsertOverride(r.Context(), tenantID, orgID, rec)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	orgID := r.Header.Get("X-Organization-ID")
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing tenant or organization header")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/feature-flags/")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 3 {
		httpresponse.Error(w, http.StatusBadRequest, "path must be /{flag_key}/{scope}/{scope_id}")
		return
	}
	flagKey := parts[0]
	scope := parts[1]
	scopeID := parts[2]

	if err := h.Repository.DeleteOverride(r.Context(), tenantID, orgID,
		featureflags.Key(flagKey), featureflags.Scope(scope), scopeID); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
