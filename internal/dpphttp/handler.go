package dpphttp

import (
	"encoding/json"
	"net/http"
	"strings"

	"integin/internal/domain/dpp"
	"integin/internal/oidcauth"
)

type TokenValidator interface {
	Validate(r *http.Request) (oidcauth.Principal, error)
}

type ActorResolver interface {
	Resolve(r *http.Request, principal oidcauth.Principal) (dpp.ActorContext, error)
}

type Handler struct {
	repo      dpp.Repository
	validator *oidcauth.Validator
	resolver  func(r *http.Request) (dpp.ActorContext, error)
}

func NewHandler(repo dpp.Repository, validator *oidcauth.Validator, resolver func(r *http.Request) (dpp.ActorContext, error)) *Handler {
	return &Handler{
		repo:      repo,
		validator: validator,
		resolver:  resolver,
	}
}

type passportRequest struct {
	ID                string `json:"id"`
	AssetID           string `json:"asset_id"`
	SerialNumber      string `json:"serial_number"`
	BatchNumber       string `json:"batch_number"`
	ManufacturerID    string `json:"manufacturer_id"`
	DPPStatus         string `json:"dpp_status"`
	AssignmentPayload string `json:"assignment_payload"`
	UpdatePayload     string `json:"update_payload"`
	UsePayload        string `json:"use_payload"`
	DisposalPayload   string `json:"disposal_payload"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")

	actor, err := h.resolver(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/dpp")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 0 || parts[0] != "passports" {
		writeError(w, http.StatusNotFound, "not_found", "route not found")
		return
	}

	switch r.Method {
	case http.MethodPost:
		if len(parts) == 1 {
			// POST /dpp/passports (Pillar 1: Assignment)
			var req passportRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "bad_request", "invalid json payload")
				return
			}
			passport := dpp.ProductPassportDPP{
				ID:                req.ID,
				TenantID:          actor.TenantID,
				OrganizationID:    actor.OrganizationID,
				AssetID:           req.AssetID,
				SerialNumber:      req.SerialNumber,
				BatchNumber:       req.BatchNumber,
				ManufacturerID:    req.ManufacturerID,
				DPPStatus:         req.DPPStatus,
				AssignmentPayload: req.AssignmentPayload,
			}
			created, err := h.repo.CreateProductPassportDPP(r.Context(), actor, passport)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "error", err.Error())
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(created)
			return
		}

	case http.MethodGet:
		if len(parts) == 2 {
			// GET /dpp/passports/{id}
			passportID := parts[1]
			got, err := h.repo.GetProductPassportDPP(r.Context(), actor, passportID)
			if err != nil {
				writeError(w, http.StatusNotFound, "not_found", err.Error())
				return
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(got)
			return
		}
		if len(parts) == 3 && parts[1] == "asset" {
			// GET /dpp/passports/asset/{assetId}
			assetID := parts[2]
			got, found, err := h.repo.GetProductPassportDPPByAsset(r.Context(), actor, assetID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "error", err.Error())
				return
			}
			if !found {
				writeError(w, http.StatusNotFound, "not_found", "asset dpp not found")
				return
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(got)
			return
		}

	case http.MethodPut:
		if len(parts) == 2 {
			// PUT /dpp/passports/{id} (Pillar 2, 3, 4 update & sealing)
			passportID := parts[1]
			var req passportRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "bad_request", "invalid json payload")
				return
			}
			passport := dpp.ProductPassportDPP{
				ID:                passportID,
				TenantID:          actor.TenantID,
				OrganizationID:    actor.OrganizationID,
				SerialNumber:      req.SerialNumber,
				BatchNumber:       req.BatchNumber,
				ManufacturerID:    req.ManufacturerID,
				DPPStatus:         req.DPPStatus,
				AssignmentPayload: req.AssignmentPayload,
				UpdatePayload:     req.UpdatePayload,
				UsePayload:        req.UsePayload,
				DisposalPayload:   req.DisposalPayload,
			}
			if err := h.repo.UpdateProductPassportDPP(r.Context(), actor, passport); err != nil {
				writeError(w, http.StatusConflict, "conflict", err.Error())
				return
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "updated", "id": passportID})
			return
		}
	}

	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not supported")
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    code,
		"message": message,
	})
}
