package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gofrs/uuid/v5"
	"integin/internal/application"
	"integin/internal/domain/assurance"
	"integin/internal/domain/authorization"
)

// AssuranceHandler provides the REST API for the Assurance subsystem.
type AssuranceHandler struct {
	service  *application.AssuranceService
	query    assurance.QueryService
}

func NewAssuranceHandler(service *application.AssuranceService, query assurance.QueryService) *AssuranceHandler {
	return &AssuranceHandler{
		service:  service,
		query:    query,
	}
}

// SyncHandler receives offline edge evaluations and routes them to the orchestrator.
// POST /api/v1/assurance/sync
func (h *AssuranceHandler) SyncHandler(w http.ResponseWriter, r *http.Request) {
	// Extract TenantID from authenticated context (simulated here)
	tenantIDRaw := r.Context().Value("tenant_id")
	if tenantIDRaw == nil {
		http.Error(w, "unauthorized: missing tenant context", http.StatusUnauthorized)
		return
	}
	tenantID, err := uuid.FromString(tenantIDRaw.(string))
	if err != nil {
		http.Error(w, "invalid tenant id", http.StatusBadRequest)
		return
	}

	var cmd application.IngestSyncCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, "malformed payload", http.StatusBadRequest)
		return
	}
	
	// Force tenant binding to prevent cross-tenant ingestion forgery
	cmd.TenantID = tenantID

	// Reject requests from the future to protect Lamport timeline
	if cmd.EffectiveAt.After(time.Now().Add(5 * time.Minute)) {
		http.Error(w, "effective_at cannot be in the future", http.StatusBadRequest)
		return
	}

	if err := h.service.ProcessEdgeSync(r.Context(), cmd); err != nil {
		// Differentiate between cryptographically rejected vs system error
		if err.Error() == "sync rejected by security policy: "+assurance.ErrSignatureTampered.Error() {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

// GetCurrentStateHandler fetches the O(1) materialized state for dashboard UI.
// GET /api/v1/assurance/assets/{asset_id}/state
func (h *AssuranceHandler) GetCurrentStateHandler(w http.ResponseWriter, r *http.Request) {
	tenantIDRaw := r.Context().Value("tenant_id")
	if tenantIDRaw == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tenantID, err := uuid.FromString(tenantIDRaw.(string))
	if err != nil {
		http.Error(w, "invalid tenant context", http.StatusForbidden)
		return
	}

	assetIDStr := r.URL.Query().Get("asset_id")
	assetID, err := uuid.FromString(assetIDStr)
	if err != nil {
		http.Error(w, "invalid asset_id", http.StatusBadRequest)
		return
	}

	state, err := h.query.GetCurrentState(r.Context(), tenantID, assetID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

// GetLedgerHistoryHandler fetches the immutable cryptographic history.
// GET /api/v1/assurance/assets/{asset_id}/ledger
func (h *AssuranceHandler) GetLedgerHistoryHandler(w http.ResponseWriter, r *http.Request) {
	tenantIDRaw := r.Context().Value("tenant_id")
	if tenantIDRaw == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tenantID, err := uuid.FromString(tenantIDRaw.(string))
	if err != nil {
		http.Error(w, "invalid tenant context", http.StatusForbidden)
		return
	}

	assetIDStr := r.URL.Query().Get("asset_id")
	assetID, err := uuid.FromString(assetIDStr)
	if err != nil {
		http.Error(w, "invalid asset_id", http.StatusBadRequest)
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	history, err := h.query.GetLedgerHistory(r.Context(), tenantID, assetID, limit)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// GetRuleBundleHandler serves compiled static rule bundles to the offline Edge app.
// GET /api/v1/assurance/rules/{bundle_id}
func (h *AssuranceHandler) GetRuleBundleHandler(w http.ResponseWriter, r *http.Request) {
	// Require tenant context for authorization
	tenantIDRaw := r.Context().Value("tenant_id")
	if tenantIDRaw == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	tenantID, err := uuid.FromString(tenantIDRaw.(string))
	if err != nil {
		http.Error(w, "invalid tenant context", http.StatusForbidden)
		return
	}

	bundleID := r.URL.Query().Get("bundle_id")
	if bundleID == "" {
		http.Error(w, "missing bundle_id", http.StatusBadRequest)
		return
	}

	// Fetch from the dynamic database rule registry
	bundle, err := h.query.GetRuleBundle(r.Context(), tenantID, bundleID)
	if err != nil {
		http.Error(w, "rule bundle not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bundle)
}

type CreateRuleBundleRequest struct {
	BundleID string          `json:"bundle_id"`
	Version  string          `json:"version"`
	IsGlobal bool            `json:"is_global"`
	Rules    json.RawMessage `json:"rules"` // The raw JSON of the rule bundle
}

// CreateRuleBundleHandler allows authorized GUI users to write new rule bundles.
// POST /api/v1/assurance/rules
func (h *AssuranceHandler) CreateRuleBundleHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Build Authorization Subject from JWT/Context
	tenantIDRaw := r.Context().Value("tenant_id")
	userIDRaw := r.Context().Value("user_id")
	if tenantIDRaw == nil || userIDRaw == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Assuming roles/capabilities are parsed into context by middleware in a real app
	subject := authorization.Subject{
		UserID:         userIDRaw.(string),
		TenantID:       tenantIDRaw.(string),
		OrganizationID: tenantIDRaw.(string), // Simplified for prototype
		Capabilities: map[authorization.Capability]bool{
			authorization.CapabilityPlatformAdmin:     true, // Simulated: user has platform admin rights
			authorization.CapabilityOrganizationAdmin: true, // Simulated: user has org admin rights
		},
	}

	// 2. Parse Request
	var req CreateRuleBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.BundleID == "" || req.Version == "" || len(req.Rules) == 0 {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	// 3. Delegate to Service (which enforces the auth matrix)
	err := h.service.SaveRuleBundle(r.Context(), subject, req.BundleID, req.Version, req.Rules, req.IsGlobal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
