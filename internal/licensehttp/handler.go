package licensehttp

import (
	"encoding/json"
	"net/http"
	"time"

	"integin/internal/domain/license"
	"integin/internal/shared/httpresponse"
)

type Handler struct {
	LicenseService license.Service
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/licenses/validate":
		h.handleValidate(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/licenses":
		h.handleIssue(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/licenses/renew":
		h.handleRenew(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/licenses/revoke":
		h.handleRevoke(w, r)
	default:
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
	}
}

func (h Handler) handleValidate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID       string `json:"tenant_id"`
		OrganizationID string `json:"organization_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}
	actor := license.ActorContext{
		TenantID:       req.TenantID,
		OrganizationID: req.OrganizationID,
		ActorID:        "system",
	}
	result, err := h.LicenseService.ValidateLicense(r.Context(), actor, time.Now())
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h Handler) handleIssue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID               string          `json:"tenant_id"`
		OrganizationID         string          `json:"organization_id"`
		ActorID                string          `json:"actor_id"`
		Tier                   license.Tier    `json:"tier"`
		MaxInspectors          int             `json:"max_inspectors"`
		MaxInspectionsPerMonth int             `json:"max_inspections_per_month"`
		Features               map[string]bool `json:"features"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}
	actor := license.ActorContext{
		TenantID:       req.TenantID,
		OrganizationID: req.OrganizationID,
		ActorID:        req.ActorID,
	}
	lic := license.License{
		TenantID:               req.TenantID,
		OrganizationID:         req.OrganizationID,
		Tier:                   req.Tier,
		MaxInspectors:          req.MaxInspectors,
		MaxInspectionsPerMonth: req.MaxInspectionsPerMonth,
		Features:               req.Features,
		CreatedBy:              req.ActorID,
	}
	result, err := h.LicenseService.IssueLicense(r.Context(), actor, lic)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, result)
}

func (h Handler) handleRenew(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID       string    `json:"tenant_id"`
		OrganizationID string    `json:"organization_id"`
		ActorID        string    `json:"actor_id"`
		LicenseID      string    `json:"license_id"`
		NewExpiresAt   time.Time `json:"new_expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}
	actor := license.ActorContext{
		TenantID:       req.TenantID,
		OrganizationID: req.OrganizationID,
		ActorID:        req.ActorID,
	}
	result, err := h.LicenseService.RenewLicense(r.Context(), actor, req.LicenseID, req.NewExpiresAt)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h Handler) handleRevoke(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID  string `json:"tenant_id"`
		OrgID     string `json:"organization_id"`
		ActorID   string `json:"actor_id"`
		LicenseID string `json:"license_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}
	actor := license.ActorContext{
		TenantID:       req.TenantID,
		OrganizationID: req.OrgID,
		ActorID:        req.ActorID,
	}
	if err := h.LicenseService.RevokeLicense(r.Context(), actor, req.LicenseID); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}
