package evidencepackhttp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"integin/internal/domain/evidencepack"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/shared/httpresponse"
)

// TokenValidator defines the contract for OIDC bearer token verification.
type TokenValidator interface {
	Validate(ctx context.Context, rawToken string) (oidcauth.Principal, error)
}

// Handler provides authenticated HTTP endpoints for evidence packs and release ledger entries.
type Handler struct {
	validator   TokenValidator
	resolver    identity.Resolver
	packRepo    evidencepack.PackRepository
	releaseRepo evidencepack.ReleaseRepository
	log         *slog.Logger
}

// NewHandler creates a new evidence pack HTTP handler.
func NewHandler(
	validator TokenValidator,
	resolver identity.Resolver,
	packRepo evidencepack.PackRepository,
	releaseRepo evidencepack.ReleaseRepository,
	log *slog.Logger,
) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{
		validator:   validator,
		resolver:    resolver,
		packRepo:    packRepo,
		releaseRepo: releaseRepo,
		log:         log,
	}
}

func (h *Handler) authenticate(r *http.Request) (evidencepack.ActorContext, error) {
	if h.validator == nil || h.resolver == nil {
		tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
		orgID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))
		actorID := strings.TrimSpace(r.Header.Get("X-Actor-ID"))
		if tenantID == "" || orgID == "" {
			return evidencepack.ActorContext{}, errors.New("missing tenant or organization header")
		}
		if actorID == "" {
			actorID = "system"
		}
		return evidencepack.ActorContext{
			TenantID:       tenantID,
			OrganizationID: orgID,
			ActorID:        actorID,
		}, nil
	}

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return evidencepack.ActorContext{}, errors.New("missing or invalid authorization header")
	}
	rawToken := strings.TrimPrefix(authHeader, "Bearer ")

	principal, err := h.validator.Validate(r.Context(), rawToken)
	if err != nil {
		return evidencepack.ActorContext{}, err
	}

	membership, err := h.resolver.Resolve(r.Context(), identity.PrincipalKey{
		Issuer:  principal.Issuer,
		Subject: principal.Subject,
	})
	if err != nil {
		return evidencepack.ActorContext{}, err
	}

	return evidencepack.ActorContext{
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

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/evidence-packs")
	path = strings.TrimPrefix(path, "/")

	switch {
	// POST /api/v1/evidence-packs -> create pack
	case path == "" && r.Method == http.MethodPost:
		h.handleCreatePack(w, r, actor)

	// GET /api/v1/evidence-packs/by-inspection/{id}
	case strings.HasPrefix(path, "by-inspection/") && r.Method == http.MethodGet:
		inspectionID := strings.TrimPrefix(path, "by-inspection/")
		h.handleListPacksByInspection(w, r, actor, inspectionID)

	// POST /api/v1/evidence-packs/{id}/seal
	case strings.HasSuffix(path, "/seal") && r.Method == http.MethodPost:
		packID := strings.TrimSuffix(path, "/seal")
		h.handleUpdatePackStatus(w, r, actor, packID, evidencepack.PackStatusSealed)

	// POST /api/v1/evidence-packs/releases -> create release record
	case path == "releases" && r.Method == http.MethodPost:
		h.handleCreateRelease(w, r, actor)

	// GET /api/v1/evidence-packs/releases/{id}
	case strings.HasPrefix(path, "releases/") && r.Method == http.MethodGet:
		sub := strings.TrimPrefix(path, "releases/")
		if strings.HasPrefix(sub, "by-pack/") {
			packID := strings.TrimPrefix(sub, "by-pack/")
			h.handleListReleasesByPack(w, r, actor, packID)
		} else {
			h.handleGetRelease(w, r, actor, sub)
		}

	// POST /api/v1/evidence-packs/releases/{id}/approve
	case strings.HasPrefix(path, "releases/") && strings.HasSuffix(path, "/approve") && r.Method == http.MethodPost:
		relID := strings.TrimPrefix(path, "releases/")
		relID = strings.TrimSuffix(relID, "/approve")
		h.handleUpdateReleaseStatus(w, r, actor, relID, evidencepack.ReleaseStatusApproved)

	// POST /api/v1/evidence-packs/releases/{id}/release
	case strings.HasPrefix(path, "releases/") && strings.HasSuffix(path, "/release") && r.Method == http.MethodPost:
		relID := strings.TrimPrefix(path, "releases/")
		relID = strings.TrimSuffix(relID, "/release")
		h.handleUpdateReleaseStatus(w, r, actor, relID, evidencepack.ReleaseStatusReleased)

	// GET /api/v1/evidence-packs/{id}
	case path != "" && r.Method == http.MethodGet:
		h.handleGetPack(w, r, actor, path)

	default:
		httpresponse.Error(w, http.StatusNotFound, "endpoint not found")
	}
}

func (h *Handler) handleCreatePack(w http.ResponseWriter, r *http.Request, actor evidencepack.ActorContext) {
	var pack evidencepack.EvidencePack
	if err := json.NewDecoder(r.Body).Decode(&pack); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	created, err := h.packRepo.Create(r.Context(), actor, pack)
	if err != nil {
		if errors.Is(err, evidencepack.ErrInvalidPack) || errors.Is(err, evidencepack.ErrPolicyViolation) || errors.Is(err, evidencepack.ErrMixedPrivacy) {
			httpresponse.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, created)
}

func (h *Handler) handleGetPack(w http.ResponseWriter, r *http.Request, actor evidencepack.ActorContext, packID string) {
	pack, err := h.packRepo.Get(r.Context(), actor, packID)
	if err != nil {
		httpresponse.Error(w, http.StatusNotFound, "evidence pack not found")
		return
	}
	httpresponse.JSON(w, http.StatusOK, pack)
}

func (h *Handler) handleListPacksByInspection(w http.ResponseWriter, r *http.Request, actor evidencepack.ActorContext, inspectionID string) {
	packs, err := h.packRepo.ListByInspection(r.Context(), actor, inspectionID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, packs)
}

type statusUpdateRequest struct {
	ExpectedRevision int64 `json:"expected_revision"`
}

func (h *Handler) handleUpdatePackStatus(w http.ResponseWriter, r *http.Request, actor evidencepack.ActorContext, packID string, target evidencepack.PackStatus) {
	var req statusUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	updated, err := h.packRepo.UpdateStatus(r.Context(), actor, packID, target, req.ExpectedRevision)
	if err != nil {
		if errors.Is(err, evidencepack.ErrInvalidTransition) || errors.Is(err, evidencepack.ErrSealConflict) || errors.Is(err, evidencepack.ErrStaleRevision) {
			httpresponse.Error(w, http.StatusConflict, err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, updated)
}

func (h *Handler) handleCreateRelease(w http.ResponseWriter, r *http.Request, actor evidencepack.ActorContext) {
	var release evidencepack.ReleasePack
	if err := json.NewDecoder(r.Body).Decode(&release); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	created, err := h.releaseRepo.Create(r.Context(), actor, release)
	if err != nil {
		if errors.Is(err, evidencepack.ErrInvalidRelease) {
			httpresponse.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, created)
}

func (h *Handler) handleGetRelease(w http.ResponseWriter, r *http.Request, actor evidencepack.ActorContext, releaseID string) {
	release, err := h.releaseRepo.Get(r.Context(), actor, releaseID)
	if err != nil {
		httpresponse.Error(w, http.StatusNotFound, "release pack not found")
		return
	}
	httpresponse.JSON(w, http.StatusOK, release)
}

func (h *Handler) handleListReleasesByPack(w http.ResponseWriter, r *http.Request, actor evidencepack.ActorContext, packID string) {
	releases, err := h.releaseRepo.ListByPack(r.Context(), actor, packID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, releases)
}

func (h *Handler) handleUpdateReleaseStatus(w http.ResponseWriter, r *http.Request, actor evidencepack.ActorContext, releaseID string, target evidencepack.ReleaseStatus) {
	var req statusUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	updated, err := h.releaseRepo.UpdateStatus(r.Context(), actor, releaseID, target, req.ExpectedRevision)
	if err != nil {
		if errors.Is(err, evidencepack.ErrInvalidReleaseState) || errors.Is(err, evidencepack.ErrStaleRevision) {
			httpresponse.Error(w, http.StatusConflict, err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, updated)
}

// Ensure strconv is used
var _ = strconv.Itoa
