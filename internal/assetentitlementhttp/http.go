package assetentitlementhttp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"integin/internal/domain/assetentitlement"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/shared/httpresponse"
)

// TokenValidator defines the contract for OIDC bearer token verification.
type TokenValidator interface {
	Validate(ctx context.Context, rawToken string) (oidcauth.Principal, error)
}

// Handler provides authenticated HTTP endpoints for asset entitlements, physical tags, and offline field packages.
type Handler struct {
	validator   TokenValidator
	resolver    identity.Resolver
	repo        assetentitlement.Repository
	packageRepo assetentitlement.PackageRepository
	log         *slog.Logger
}

// NewHandler creates a new asset entitlement HTTP handler.
func NewHandler(
	validator TokenValidator,
	resolver identity.Resolver,
	repo assetentitlement.Repository,
	packageRepo assetentitlement.PackageRepository,
	log *slog.Logger,
) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{
		validator:   validator,
		resolver:    resolver,
		repo:        repo,
		packageRepo: packageRepo,
		log:         log,
	}
}

func (h *Handler) authenticate(r *http.Request) (assetentitlement.ActorContext, error) {
	if h.validator == nil || h.resolver == nil {
		tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
		orgID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))
		actorID := strings.TrimSpace(r.Header.Get("X-Actor-ID"))
		if tenantID == "" || orgID == "" {
			return assetentitlement.ActorContext{}, errors.New("missing tenant or organization header")
		}
		if actorID == "" {
			actorID = "system"
		}
		return assetentitlement.ActorContext{
			TenantID:       tenantID,
			OrganizationID: orgID,
			ActorID:        actorID,
		}, nil
	}

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return assetentitlement.ActorContext{}, errors.New("missing or invalid authorization header")
	}
	rawToken := strings.TrimPrefix(authHeader, "Bearer ")

	principal, err := h.validator.Validate(r.Context(), rawToken)
	if err != nil {
		return assetentitlement.ActorContext{}, err
	}

	membership, err := h.resolver.Resolve(r.Context(), identity.PrincipalKey{
		Issuer:  principal.Issuer,
		Subject: principal.Subject,
	})
	if err != nil {
		return assetentitlement.ActorContext{}, err
	}

	return assetentitlement.ActorContext{
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

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/asset-entitlements")
	path = strings.TrimPrefix(path, "/")

	switch {
	// POST /api/v1/asset-entitlements -> register entitlement
	case path == "" && r.Method == http.MethodPost:
		h.handleRegister(w, r, actor)

	// GET /api/v1/asset-entitlements/by-work-order/{id}
	case strings.HasPrefix(path, "by-work-order/") && r.Method == http.MethodGet:
		woID := strings.TrimPrefix(path, "by-work-order/")
		h.handleGetByWorkOrder(w, r, actor, woID)

	// GET /api/v1/asset-entitlements/by-asset/{id}
	case strings.HasPrefix(path, "by-asset/") && r.Method == http.MethodGet:
		assetID := strings.TrimPrefix(path, "by-asset/")
		h.handleGetByAsset(w, r, actor, assetID)

	// POST /api/v1/asset-entitlements/{id}/complete
	case strings.HasSuffix(path, "/complete") && r.Method == http.MethodPost:
		entID := strings.TrimSuffix(path, "/complete")
		h.handleComplete(w, r, actor, entID)

	// POST /api/v1/asset-entitlements/{id}/cancel
	case strings.HasSuffix(path, "/cancel") && r.Method == http.MethodPost:
		entID := strings.TrimSuffix(path, "/cancel")
		h.handleCancel(w, r, actor, entID)

	// POST /api/v1/asset-entitlements/{id}/assign-inspector
	case strings.HasSuffix(path, "/assign-inspector") && r.Method == http.MethodPost:
		entID := strings.TrimSuffix(path, "/assign-inspector")
		h.handleAssignInspector(w, r, actor, entID)

	// POST /api/v1/asset-entitlements/tags -> add physical tag
	case path == "tags" && r.Method == http.MethodPost:
		h.handleAddTag(w, r, actor)

	// GET /api/v1/asset-entitlements/tags/{asset_id} -> list tags
	case strings.HasPrefix(path, "tags/") && r.Method == http.MethodGet:
		assetID := strings.TrimPrefix(path, "tags/")
		h.handleListTags(w, r, actor, assetID)

	// POST /api/v1/asset-entitlements/packages -> generate offline package
	case path == "packages" && r.Method == http.MethodPost:
		h.handleGeneratePackage(w, r, actor)

	// GET /api/v1/asset-entitlements/packages/{id}
	case strings.HasPrefix(path, "packages/") && r.Method == http.MethodGet:
		pkgID := strings.TrimPrefix(path, "packages/")
		h.handleGetPackage(w, r, actor, pkgID)

	// GET /api/v1/asset-entitlements/{id}
	case path != "" && r.Method == http.MethodGet:
		h.handleGet(w, r, actor, path)

	default:
		httpresponse.Error(w, http.StatusNotFound, "endpoint not found")
	}
}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request, actor assetentitlement.ActorContext) {
	var ent assetentitlement.AssetEntitlement
	if err := json.NewDecoder(r.Body).Decode(&ent); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	registered, err := h.repo.Register(r.Context(), actor, ent)
	if err != nil {
		if errors.Is(err, assetentitlement.ErrInvalidEntitlement) {
			httpresponse.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, assetentitlement.ErrImmutable) {
			httpresponse.Error(w, http.StatusConflict, err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, registered)
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request, actor assetentitlement.ActorContext, id string) {
	ent, err := h.repo.Get(r.Context(), actor, id)
	if err != nil {
		httpresponse.Error(w, http.StatusNotFound, "asset entitlement not found")
		return
	}
	httpresponse.JSON(w, http.StatusOK, ent)
}

func (h *Handler) handleGetByWorkOrder(w http.ResponseWriter, r *http.Request, actor assetentitlement.ActorContext, woID string) {
	ents, err := h.repo.GetByWorkOrder(r.Context(), actor, woID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, ents)
}

func (h *Handler) handleGetByAsset(w http.ResponseWriter, r *http.Request, actor assetentitlement.ActorContext, assetID string) {
	ents, err := h.repo.GetByAsset(r.Context(), actor, assetID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, ents)
}

type statusRequest struct {
	ExpectedRevision int64 `json:"expected_revision"`
}

func (h *Handler) handleComplete(w http.ResponseWriter, r *http.Request, actor assetentitlement.ActorContext, id string) {
	var req statusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	completed, err := h.repo.Complete(r.Context(), actor, id, req.ExpectedRevision)
	if err != nil {
		if errors.Is(err, assetentitlement.ErrInvalidTransition) || errors.Is(err, assetentitlement.ErrStaleRevision) {
			httpresponse.Error(w, http.StatusConflict, err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, completed)
}

func (h *Handler) handleCancel(w http.ResponseWriter, r *http.Request, actor assetentitlement.ActorContext, id string) {
	var req statusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	cancelled, err := h.repo.Cancel(r.Context(), actor, id, req.ExpectedRevision)
	if err != nil {
		if errors.Is(err, assetentitlement.ErrInvalidTransition) || errors.Is(err, assetentitlement.ErrStaleRevision) {
			httpresponse.Error(w, http.StatusConflict, err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, cancelled)
}

type assignInspectorRequest struct {
	InspectorID      string `json:"inspector_id"`
	ExpectedRevision int64  `json:"expected_revision"`
}

func (h *Handler) handleAssignInspector(w http.ResponseWriter, r *http.Request, actor assetentitlement.ActorContext, id string) {
	var req assignInspectorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	assigned, err := h.repo.AssignInspector(r.Context(), actor, id, req.InspectorID, req.ExpectedRevision)
	if err != nil {
		if errors.Is(err, assetentitlement.ErrStaleRevision) {
			httpresponse.Error(w, http.StatusConflict, err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, assigned)
}

func (h *Handler) handleAddTag(w http.ResponseWriter, r *http.Request, actor assetentitlement.ActorContext) {
	var tag assetentitlement.AssetTag
	if err := json.NewDecoder(r.Body).Decode(&tag); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	added, err := h.repo.AddTag(r.Context(), actor, tag)
	if err != nil {
		if errors.Is(err, assetentitlement.ErrInvalidTag) {
			httpresponse.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, added)
}

func (h *Handler) handleListTags(w http.ResponseWriter, r *http.Request, actor assetentitlement.ActorContext, assetID string) {
	tags, err := h.repo.ListTags(r.Context(), actor, assetID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, tags)
}

func (h *Handler) handleGeneratePackage(w http.ResponseWriter, r *http.Request, actor assetentitlement.ActorContext) {
	if h.packageRepo == nil {
		httpresponse.Error(w, http.StatusNotImplemented, "package repository not configured")
		return
	}
	var pkg assetentitlement.OfflinePackage
	if err := json.NewDecoder(r.Body).Decode(&pkg); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	generated, err := h.packageRepo.Generate(r.Context(), actor, pkg)
	if err != nil {
		if errors.Is(err, assetentitlement.ErrInvalidPackage) {
			httpresponse.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, generated)
}

func (h *Handler) handleGetPackage(w http.ResponseWriter, r *http.Request, actor assetentitlement.ActorContext, pkgID string) {
	if h.packageRepo == nil {
		httpresponse.Error(w, http.StatusNotImplemented, "package repository not configured")
		return
	}
	pkg, err := h.packageRepo.Get(r.Context(), actor, pkgID)
	if err != nil {
		httpresponse.Error(w, http.StatusNotFound, "offline package not found")
		return
	}
	httpresponse.JSON(w, http.StatusOK, pkg)
}
