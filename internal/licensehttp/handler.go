package licensehttp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"integin/internal/domain/license"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/shared/httpresponse"
)

type TokenValidator interface {
	Validate(context.Context, string) (oidcauth.Principal, error)
}

type ValidatorWrapper struct {
	*oidcauth.Validator
}

type Handler struct {
	Validator      TokenValidator
	Resolver       identity.Resolver
	LicenseService license.Service
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.Validator == nil || h.Resolver == nil || h.LicenseService == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "service_unavailable")
		return
	}

	raw, ok := bearer(r.Header.Get("Authorization"))
	if !ok {
		httpresponse.Error(w, http.StatusUnauthorized, "authentication_failed")
		return
	}
	principal, err := h.Validator.Validate(r.Context(), raw)
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "authentication_failed")
		return
	}
	membership, err := h.Resolver.Resolve(r.Context(), identity.PrincipalKey{Issuer: principal.Issuer, Subject: principal.Subject})
	if err != nil {
		httpresponse.Error(w, http.StatusForbidden, "authorization_failed")
		return
	}
	actor := license.ActorContext{
		TenantID:       membership.TenantID,
		OrganizationID: membership.OrganizationID,
		ActorID:        membership.ActorID,
	}

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/licenses":
		h.handleList(w, r, actor)
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/licenses/validate":
		h.handleValidate(w, r, actor)
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/licenses":
		h.handleIssue(w, r, actor)
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/licenses/renew":
		h.handleRenew(w, r, actor)
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/licenses/revoke":
		h.handleRevoke(w, r, actor)
	default:
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
	}
}

func bearer(value string) (string, bool) {
	parts := strings.Fields(value)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func (h Handler) handleValidate(w http.ResponseWriter, r *http.Request, actor license.ActorContext) {
	result, err := h.LicenseService.ValidateLicense(r.Context(), actor, time.Now())
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h Handler) handleIssue(w http.ResponseWriter, r *http.Request, actor license.ActorContext) {
	var req struct {
		Tier                   license.Tier    `json:"tier"`
		MaxInspectors          int             `json:"max_inspectors"`
		MaxInspectionsPerMonth int             `json:"max_inspections_per_month"`
		Features               map[string]bool `json:"features"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}
	lic := license.License{
		TenantID:               actor.TenantID,
		OrganizationID:         actor.OrganizationID,
		Tier:                   req.Tier,
		MaxInspectors:          req.MaxInspectors,
		MaxInspectionsPerMonth: req.MaxInspectionsPerMonth,
		Features:               req.Features,
		CreatedBy:              actor.ActorID,
	}
	result, err := h.LicenseService.IssueLicense(r.Context(), actor, lic)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, result)
}

func (h Handler) handleRenew(w http.ResponseWriter, r *http.Request, actor license.ActorContext) {
	var req struct {
		LicenseID    string    `json:"license_id"`
		NewExpiresAt time.Time `json:"new_expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}
	result, err := h.LicenseService.RenewLicense(r.Context(), actor, req.LicenseID, req.NewExpiresAt)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h Handler) handleRevoke(w http.ResponseWriter, r *http.Request, actor license.ActorContext) {
	var req struct {
		LicenseID string `json:"license_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := h.LicenseService.RevokeLicense(r.Context(), actor, req.LicenseID); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h Handler) handleList(w http.ResponseWriter, r *http.Request, actor license.ActorContext) {
	licenses, err := h.LicenseService.ListLicenses(r.Context(), actor)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, licenses)
}
