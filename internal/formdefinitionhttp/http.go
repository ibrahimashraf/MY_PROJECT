package formdefinitionhttp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"integin/internal/domain/formdefinition"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/shared/httpresponse"
)

// TokenValidator defines the contract for OIDC bearer token verification.
type TokenValidator interface {
	Validate(ctx context.Context, rawToken string) (oidcauth.Principal, error)
}

// Handler provides HTTP endpoints for versioned form definitions and evidence policy.
type Handler struct {
	validator TokenValidator
	resolver  identity.Resolver
	repo      formdefinition.Repository
	log       *slog.Logger
}

// NewHandler creates a new authenticated form definition HTTP handler.
func NewHandler(
	validator TokenValidator,
	resolver identity.Resolver,
	repo formdefinition.Repository,
	log *slog.Logger,
) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{
		validator: validator,
		resolver:  resolver,
		repo:      repo,
		log:       log,
	}
}

func (h *Handler) authenticate(r *http.Request) (formdefinition.ActorContext, error) {
	if h.validator == nil || h.resolver == nil {
		// Backwards-compatible or local-dev mode with headers
		tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
		orgID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))
		actorID := strings.TrimSpace(r.Header.Get("X-Actor-ID"))
		if tenantID == "" || orgID == "" {
			return formdefinition.ActorContext{}, errors.New("missing tenant or organization header")
		}
		if actorID == "" {
			actorID = "system"
		}
		return formdefinition.ActorContext{
			TenantID:       tenantID,
			OrganizationID: orgID,
			ActorID:        actorID,
		}, nil
	}

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return formdefinition.ActorContext{}, errors.New("missing or invalid authorization header")
	}
	rawToken := strings.TrimPrefix(authHeader, "Bearer ")

	principal, err := h.validator.Validate(r.Context(), rawToken)
	if err != nil {
		return formdefinition.ActorContext{}, err
	}

	membership, err := h.resolver.Resolve(r.Context(), identity.PrincipalKey{
		Issuer:  principal.Issuer,
		Subject: principal.Subject,
	})
	if err != nil {
		return formdefinition.ActorContext{}, err
	}

	return formdefinition.ActorContext{
		TenantID:       membership.TenantID,
		OrganizationID: membership.OrganizationID,
		ActorID:        membership.ActorID,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authenticate(r)
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "unauthorized: "+err.Error())
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/form-definitions")
	path = strings.TrimPrefix(path, "/")

	switch {
	case path == "" && r.Method == http.MethodPost:
		h.handleRegisterDraft(w, r, actor)
	case strings.HasPrefix(path, "by-code/"):
		formCode := strings.TrimPrefix(path, "by-code/")
		h.handleGetByCode(w, r, actor, formCode)
	case strings.HasPrefix(path, "by-asset-type/"):
		assetType := strings.TrimPrefix(path, "by-asset-type/")
		h.handleListByAssetType(w, r, actor, assetType)
	case strings.HasPrefix(path, "by-status/"):
		status := strings.TrimPrefix(path, "by-status/")
		h.handleListByStatus(w, r, actor, formdefinition.FormStatus(status))
	case strings.HasSuffix(path, "/approve") && r.Method == http.MethodPost:
		formID := strings.TrimSuffix(path, "/approve")
		h.handleApprove(w, r, actor, formID)
	case strings.HasSuffix(path, "/retire") && r.Method == http.MethodPost:
		formID := strings.TrimSuffix(path, "/retire")
		h.handleRetire(w, r, actor, formID)
	case path != "" && r.Method == http.MethodGet:
		h.handleGet(w, r, actor, path)
	default:
		httpresponse.Error(w, http.StatusNotFound, "endpoint not found")
	}
}

func (h *Handler) handleRegisterDraft(w http.ResponseWriter, r *http.Request, actor formdefinition.ActorContext) {
	var form formdefinition.FormVersion
	if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	registered, created, err := h.repo.RegisterDraft(r.Context(), actor, form)
	if err != nil {
		if errors.Is(err, formdefinition.ErrInvalidForm) || errors.Is(err, formdefinition.ErrInvalidField) || errors.Is(err, formdefinition.ErrDuplicateField) {
			httpresponse.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, formdefinition.ErrImmutable) {
			httpresponse.Error(w, http.StatusConflict, err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if created {
		httpresponse.JSON(w, http.StatusCreated, registered)
	} else {
		httpresponse.JSON(w, http.StatusOK, registered)
	}
}

type transitionRequest struct {
	ExpectedVersion int `json:"expected_version"`
}

func (h *Handler) handleApprove(w http.ResponseWriter, r *http.Request, actor formdefinition.ActorContext, formID string) {
	var req transitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	approved, err := h.repo.Approve(r.Context(), actor, formID, req.ExpectedVersion)
	if err != nil {
		if errors.Is(err, formdefinition.ErrInvalidStatus) {
			httpresponse.Error(w, http.StatusConflict, err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, approved)
}

func (h *Handler) handleRetire(w http.ResponseWriter, r *http.Request, actor formdefinition.ActorContext, formID string) {
	var req transitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	retired, err := h.repo.Retire(r.Context(), actor, formID, req.ExpectedVersion)
	if err != nil {
		if errors.Is(err, formdefinition.ErrInvalidStatus) {
			httpresponse.Error(w, http.StatusConflict, err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, retired)
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request, actor formdefinition.ActorContext, formID string) {
	form, err := h.repo.Get(r.Context(), actor, formID)
	if err != nil {
		httpresponse.Error(w, http.StatusNotFound, "form definition not found")
		return
	}
	httpresponse.JSON(w, http.StatusOK, form)
}

func (h *Handler) handleGetByCode(w http.ResponseWriter, r *http.Request, actor formdefinition.ActorContext, formCode string) {
	if r.Method != http.MethodGet {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	form, err := h.repo.GetByCode(r.Context(), actor, formCode)
	if err != nil {
		httpresponse.Error(w, http.StatusNotFound, "form definition not found")
		return
	}
	httpresponse.JSON(w, http.StatusOK, form)
}

func (h *Handler) handleListByAssetType(w http.ResponseWriter, r *http.Request, actor formdefinition.ActorContext, assetType string) {
	if r.Method != http.MethodGet {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	forms, err := h.repo.ListByAssetType(r.Context(), actor, assetType)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, forms)
}

func (h *Handler) handleListByStatus(w http.ResponseWriter, r *http.Request, actor formdefinition.ActorContext, status formdefinition.FormStatus) {
	if r.Method != http.MethodGet {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	forms, err := h.repo.ListByStatus(r.Context(), actor, status)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, forms)
}
