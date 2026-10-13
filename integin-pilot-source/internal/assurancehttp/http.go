package assurancehttp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"integin/internal/domain/assurance"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/shared/httpresponse"
)

// TokenValidator defines the contract for OIDC bearer token verification.
type TokenValidator interface {
	Validate(ctx context.Context, rawToken string) (oidcauth.Principal, error)
}

// Handler provides HTTP endpoints for assurance projections and corrective work.
type Handler struct {
	validator      TokenValidator
	resolver       identity.Resolver
	projections    assurance.ProjectionRepository
	correctiveWork assurance.WorkRepository
	log            *slog.Logger
}

// NewHandler creates a new authenticated assurance HTTP handler.
func NewHandler(
	validator TokenValidator,
	resolver identity.Resolver,
	projections assurance.ProjectionRepository,
	correctiveWork assurance.WorkRepository,
	log *slog.Logger,
) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{
		validator:      validator,
		resolver:       resolver,
		projections:    projections,
		correctiveWork: correctiveWork,
		log:            log,
	}
}

func (h *Handler) authenticate(r *http.Request) (assurance.ActorContext, error) {
	if h.validator == nil || h.resolver == nil {
		// Backwards-compatible or local-dev mode with headers
		tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
		orgID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))
		actorID := strings.TrimSpace(r.Header.Get("X-Actor-ID"))
		if tenantID == "" || orgID == "" {
			return assurance.ActorContext{}, errors.New("missing tenant or organization header")
		}
		if actorID == "" {
			actorID = "system"
		}
		return assurance.ActorContext{
			TenantID:       tenantID,
			OrganizationID: orgID,
			ActorID:        actorID,
		}, nil
	}

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return assurance.ActorContext{}, errors.New("missing or invalid authorization header")
	}
	rawToken := strings.TrimPrefix(authHeader, "Bearer ")

	principal, err := h.validator.Validate(r.Context(), rawToken)
	if err != nil {
		return assurance.ActorContext{}, err
	}

	membership, err := h.resolver.Resolve(r.Context(), identity.PrincipalKey{
		Issuer:  principal.Issuer,
		Subject: principal.Subject,
	})
	if err != nil {
		return assurance.ActorContext{}, err
	}

	return assurance.ActorContext{
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

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/assurance")
	path = strings.TrimPrefix(path, "/")

	switch {
	case strings.HasPrefix(path, "projections/by-certificate/"):
		certID := strings.TrimPrefix(path, "projections/by-certificate/")
		h.handleGetProjection(w, r, actor, certID)
	case strings.HasPrefix(path, "projections/by-inspection/"):
		inspID := strings.TrimPrefix(path, "projections/by-inspection/")
		h.handleGetProjectionByInspection(w, r, actor, inspID)
	case strings.HasPrefix(path, "projections/by-asset/"):
		assetID := strings.TrimPrefix(path, "projections/by-asset/")
		h.handleListProjectionsByAsset(w, r, actor, assetID)
	case path == "corrective-work" && r.Method == http.MethodPost:
		h.handleCreateCorrectiveWork(w, r, actor)
	case strings.HasPrefix(path, "corrective-work/by-inspection/"):
		inspID := strings.TrimPrefix(path, "corrective-work/by-inspection/")
		h.handleListCorrectiveWorkByInspection(w, r, actor, inspID)
	case strings.HasPrefix(path, "corrective-work/by-asset/"):
		assetID := strings.TrimPrefix(path, "corrective-work/by-asset/")
		h.handleListCorrectiveWorkByAsset(w, r, actor, assetID)
	case strings.HasPrefix(path, "corrective-work/") && strings.HasSuffix(path, "/status") && r.Method == http.MethodPatch:
		workID := strings.TrimSuffix(strings.TrimPrefix(path, "corrective-work/"), "/status")
		h.handleUpdateWorkStatus(w, r, actor, workID)
	case strings.HasPrefix(path, "corrective-work/") && strings.HasSuffix(path, "/assign") && r.Method == http.MethodPatch:
		workID := strings.TrimSuffix(strings.TrimPrefix(path, "corrective-work/"), "/assign")
		h.handleAssignWork(w, r, actor, workID)
	case strings.HasPrefix(path, "corrective-work/") && r.Method == http.MethodGet:
		workID := strings.TrimPrefix(path, "corrective-work/")
		h.handleGetCorrectiveWork(w, r, actor, workID)
	default:
		httpresponse.Error(w, http.StatusNotFound, "endpoint not found")
	}
}

func (h *Handler) handleGetProjection(w http.ResponseWriter, r *http.Request, actor assurance.ActorContext, certID string) {
	if r.Method != http.MethodGet {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	p, err := h.projections.Get(r.Context(), actor, certID)
	if err != nil {
		if errors.Is(err, assurance.ErrNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "projection not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, p)
}

func (h *Handler) handleGetProjectionByInspection(w http.ResponseWriter, r *http.Request, actor assurance.ActorContext, inspID string) {
	if r.Method != http.MethodGet {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	p, err := h.projections.GetByInspection(r.Context(), actor, inspID)
	if err != nil {
		if errors.Is(err, assurance.ErrNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "projection not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, p)
}

func (h *Handler) handleListProjectionsByAsset(w http.ResponseWriter, r *http.Request, actor assurance.ActorContext, assetID string) {
	if r.Method != http.MethodGet {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projections, err := h.projections.ListByAsset(r.Context(), actor, assetID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, projections)
}

func (h *Handler) handleCreateCorrectiveWork(w http.ResponseWriter, r *http.Request, actor assurance.ActorContext) {
	var work assurance.CorrectiveWork
	if err := json.NewDecoder(r.Body).Decode(&work); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	created, err := h.correctiveWork.Create(r.Context(), actor, work)
	if err != nil {
		if errors.Is(err, assurance.ErrInvalidWork) {
			httpresponse.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, created)
}

func (h *Handler) handleGetCorrectiveWork(w http.ResponseWriter, r *http.Request, actor assurance.ActorContext, workID string) {
	work, err := h.correctiveWork.Get(r.Context(), actor, workID)
	if err != nil {
		if errors.Is(err, assurance.ErrNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "corrective work not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, work)
}

func (h *Handler) handleListCorrectiveWorkByInspection(w http.ResponseWriter, r *http.Request, actor assurance.ActorContext, inspID string) {
	if r.Method != http.MethodGet {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	items, err := h.correctiveWork.ListByInspection(r.Context(), actor, inspID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, items)
}

func (h *Handler) handleListCorrectiveWorkByAsset(w http.ResponseWriter, r *http.Request, actor assurance.ActorContext, assetID string) {
	if r.Method != http.MethodGet {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	items, err := h.correctiveWork.ListByAsset(r.Context(), actor, assetID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, items)
}

type updateStatusRequest struct {
	TargetStatus     assurance.WorkStatus `json:"target_status"`
	ExpectedRevision int64                `json:"expected_revision"`
}

func (h *Handler) handleUpdateWorkStatus(w http.ResponseWriter, r *http.Request, actor assurance.ActorContext, workID string) {
	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	updated, err := h.correctiveWork.UpdateStatus(r.Context(), actor, workID, req.TargetStatus, req.ExpectedRevision)
	if err != nil {
		if errors.Is(err, assurance.ErrInvalidTransition) || errors.Is(err, assurance.ErrStaleRevision) {
			httpresponse.Error(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, assurance.ErrNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "corrective work not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, updated)
}

type assignWorkRequest struct {
	InspectorID      string `json:"inspector_id"`
	ExpectedRevision int64  `json:"expected_revision"`
}

func (h *Handler) handleAssignWork(w http.ResponseWriter, r *http.Request, actor assurance.ActorContext, workID string) {
	var req assignWorkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	updated, err := h.correctiveWork.Assign(r.Context(), actor, workID, req.InspectorID, req.ExpectedRevision)
	if err != nil {
		if errors.Is(err, assurance.ErrStaleRevision) {
			httpresponse.Error(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, assurance.ErrNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "corrective work not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, updated)
}
