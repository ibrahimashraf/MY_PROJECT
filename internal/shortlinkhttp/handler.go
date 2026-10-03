package shortlinkhttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"integin/internal/domain/shortlink"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/oidchttp"
	"integin/internal/shortlinksvc"
)

type contextKey string

const (
	ctxTenantID   contextKey = "shortlink_tenant_id"
	ctxOrgID      contextKey = "shortlink_organization_id"
	ctxActorID    contextKey = "shortlink_actor_id"
	ctxMembership contextKey = "shortlink_membership"
)

// TokenValidator defines the contract for OIDC bearer token verification.
type TokenValidator interface {
	Validate(ctx context.Context, rawToken string) (oidcauth.Principal, error)
}

type Handler struct {
	svc       *shortlinksvc.Service
	validator TokenValidator
	resolver  identity.Resolver
	router    chi.Router
	initOnce  sync.Once
}

// New creates a short link HTTP handler for backwards-compatibility and testing.
func New(svc *shortlinksvc.Service) *Handler {
	return NewWithAuth(svc, nil, nil)
}

// NewWithAuth creates a short link HTTP handler fully wired with OIDC token validation
// and local identity resolution to prevent tenant impersonation and organization takeover.
func NewWithAuth(svc *shortlinksvc.Service, validator TokenValidator, resolver identity.Resolver) *Handler {
	h := &Handler{
		svc:       svc,
		validator: validator,
		resolver:  resolver,
	}
	router := chi.NewRouter()
	h.RegisterRoutes(router)
	h.router = router
	return h
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Use(chimiddleware.Recoverer)

	// Public redirect
	r.Get("/s/{code}", h.RedirectHandler)

	// Admin API
	r.Route("/admin/api/v1/shortlinks", func(admin chi.Router) {
		admin.Use(h.AdminAuthMiddleware)

		admin.Post("/", h.CreateHandler)
		admin.Get("/", h.ListHandler)
		admin.Post("/bulk", h.BulkCreateHandler)
		admin.Post("/bulk/qr-zip", h.BulkQRZipHandler)
		admin.Post("/import", h.BulkImportHandler)
		admin.Post("/export", h.BulkExportHandler)
		admin.Get("/{code}", h.GetHandler)
		admin.Get("/{code}/stats", h.StatsHandler)
		admin.Delete("/{code}", h.RevokeHandler)

		// Webhook DLQ endpoints
		admin.Get("/webhook/dlq", h.ListDLQHandler)
		admin.Post("/webhook/dlq/{id}/retry", h.RetryDLQHandler)
		admin.Post("/webhook/dlq/{id}/resolve", h.ResolveDLQHandler)

		// HMAC secret management
		admin.Post("/hmac/secrets", h.CreateHMACSecretHandler)
		admin.Get("/hmac/secrets", h.ListHMACSecretsHandler)
		admin.Get("/hmac/secrets/{version}", h.GetHMACSecretHandler)
		admin.Delete("/hmac/secrets/{version}", h.RevokeHMACSecretHandler)

		// Anomaly detection endpoints
		admin.Post("/anomaly/rules", h.CreateAnomalyRuleHandler)
		admin.Get("/anomaly/rules", h.ListAnomalyRulesHandler)
		admin.Get("/anomaly/rules/{id}", h.GetAnomalyRuleHandler)
		admin.Patch("/anomaly/rules/{id}", h.UpdateAnomalyRuleHandler)
		admin.Delete("/anomaly/rules/{id}", h.DeleteAnomalyRuleHandler)

		admin.Get("/anomaly/alerts", h.ListAnomalyAlertsHandler)
		admin.Get("/anomaly/alerts/{id}", h.GetAnomalyAlertHandler)
		admin.Patch("/anomaly/alerts/{id}", h.UpdateAnomalyAlertHandler)
		admin.Post("/anomaly/alerts/{id}/acknowledge", h.AcknowledgeAnomalyAlertHandler)
		admin.Post("/anomaly/alerts/{id}/resolve", h.ResolveAnomalyAlertHandler)

		// Dashboard Analytics endpoints
		admin.Route("/analytics", func(analytics chi.Router) {
			analytics.Get("/overview", h.AnalyticsOverviewHandler)
			analytics.Get("/timeseries", h.AnalyticsTimeSeriesHandler)
			analytics.Get("/geo", h.AnalyticsGeoHandler)
			analytics.Get("/devices", h.AnalyticsDevicesHandler)
			analytics.Get("/funnel", h.AnalyticsFunnelHandler)
			analytics.Get("/top-assets", h.AnalyticsTopAssetsHandler)
		})
	})
}

func (h *Handler) RedirectHandler(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	// Enforce custom_domain host routing if set
	if sl, err := h.svc.GetStats(r.Context(), code); err == nil && sl.CustomDomain != nil && *sl.CustomDomain != "" {
		if r.Host != *sl.CustomDomain && r.Header.Get("X-Forwarded-Host") != *sl.CustomDomain {
			http.NotFound(w, r)
			return
		}
	}
	target, err := h.svc.ResolveShortLink(r.Context(), code)
	if err != nil {
		if errors.Is(err, shortlinksvc.ErrExpired) || errors.Is(err, shortlinksvc.ErrRevoked) {
			w.WriteHeader(http.StatusGone)
			return
		}
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func isValidURL(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return false
	}
	u, err := url.ParseRequestURI(raw)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func (h *Handler) CreateHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TargetURL     string `json:"target_url"`
		TTL           string `json:"ttl"`
		WebhookURL    string `json:"webhook_url"`
		CustomDomain  string `json:"custom_domain"`
		HMACSecretRef string `json:"hmac_secret_ref"`
		HMACAlgorithm string `json:"hmac_algorithm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !isValidURL(req.TargetURL) {
		writeError(w, http.StatusBadRequest, "target_url: valid URL required")
		return
	}

	var ttl time.Duration
	if req.TTL != "" {
		d, err := time.ParseDuration(req.TTL)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid ttl format")
			return
		}
		ttl = d
	}

	var webhookURL *string
	if req.WebhookURL != "" {
		webhookURL = &req.WebhookURL
	}
	var customDomain *string
	if req.CustomDomain != "" {
		customDomain = &req.CustomDomain
	}
	var hmacSecretRef *string
	if req.HMACSecretRef != "" {
		hmacSecretRef = &req.HMACSecretRef
	}
	var hmacAlgorithm *string
	if req.HMACAlgorithm != "" {
		hmacAlgorithm = &req.HMACAlgorithm
	}

	code, err := h.svc.CreateShortLink(r.Context(), shortlink.CreateRequest{
		TargetURL:     req.TargetURL,
		TTL:           ttl,
		WebhookURL:    webhookURL,
		CustomDomain:  customDomain,
		HMACSecretRef: hmacSecretRef,
		HMACAlgorithm: hmacAlgorithm,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	baseURL := getShortBaseURL(r)
	writeJSON(w, http.StatusCreated, map[string]any{
		"code":       code,
		"short_url":  baseURL + "/" + code,
		"target_url": req.TargetURL,
	})
}

func (h *Handler) ListHandler(w http.ResponseWriter, r *http.Request) {
	links, err := h.svc.List(r.Context(), 50, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, links)
}

func (h *Handler) GetHandler(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	sl, err := h.svc.GetStats(r.Context(), code)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, sl)
}

func (h *Handler) StatsHandler(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	sl, err := h.svc.GetStats(r.Context(), code)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, sl)
}

func (h *Handler) RevokeHandler(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if err := h.svc.RevokeShortLink(r.Context(), code); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListDLQHandler(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if _, err := fmt.Sscanf(l, "%d", &limit); err != nil || limit <= 0 || limit > 1000 {
			limit = 50
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if _, err := fmt.Sscanf(o, "%d", &offset); err != nil || offset < 0 {
			offset = 0
		}
	}

	var status *shortlink.WebhookDeliveryStatus
	if s := r.URL.Query().Get("status"); s != "" {
		ws := shortlink.WebhookDeliveryStatus(s)
		status = &ws
	}

	entries, total, err := h.svc.GetDLQEntries(r.Context(), shortlink.ListDLQRequest{
		Limit:  limit,
		Offset: offset,
		Status: status,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"entries": entries,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

func (h *Handler) RetryDLQHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	err := h.svc.RetryDLQEntry(r.Context(), shortlink.RetryDLQRequest{DeliveryID: id})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "retry scheduled"})
}

func (h *Handler) ResolveDLQHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Require authenticated actor — never fall back to a ghost "admin"
	org, ok := oidchttp.OrganizationContextFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Optional body override; if not present or empty, use authenticated actor
	var req struct {
		ResolvedBy string `json:"resolved_by"`
	}
	if err := decodeOptionalJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json payload: "+err.Error())
		return
	}
	if req.ResolvedBy == "" {
		req.ResolvedBy = org.ActorID
	}

	err := h.svc.ResolveDLQEntry(r.Context(), id, req.ResolvedBy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "resolved"})
}

// getTenantID returns the authenticated tenant_id, enforcing tenant isolation.
// If a caller explicitly specifies a tenant_id, it is validated against the authenticated tenant_id
// to prevent cross-tenant access and organization takeover.
func (h *Handler) getTenantID(r *http.Request, requestedTenant string) (string, error) {
	if val := r.Context().Value(ctxTenantID); val != nil {
		if authTenant, ok := val.(string); ok && authTenant != "" {
			if requestedTenant != "" && requestedTenant != authTenant {
				return "", errors.New("cross-tenant access prohibited")
			}
			return authTenant, nil
		}
	}
	if org, ok := oidchttp.OrganizationContextFrom(r.Context()); ok && org.TenantID != "" {
		if requestedTenant != "" && requestedTenant != org.TenantID {
			return "", errors.New("cross-tenant access prohibited")
		}
		return org.TenantID, nil
	}
	if requestedTenant != "" {
		return requestedTenant, nil
	}
	return "", errors.New("tenant_id is required")
}

func (h *Handler) CreateHMACSecretHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID  string `json:"tenant_id"`
		Secret    string `json:"secret"`
		Algorithm string `json:"algorithm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(req.Secret) == "" {
		writeError(w, http.StatusBadRequest, "secret is required")
		return
	}

	tenantID, err := h.getTenantID(r, req.TenantID)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	secret, err := h.svc.CreateHMACSecret(r.Context(), tenantID, req.Secret, req.Algorithm)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, secret)
}

func (h *Handler) ListHMACSecretsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r, r.URL.Query().Get("tenant_id"))
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	secrets, err := h.svc.ListHMACSecrets(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, secrets)
}

func (h *Handler) GetHMACSecretHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r, r.URL.Query().Get("tenant_id"))
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	versionStr := chi.URLParam(r, "version")
	var version int
	if versionStr == "latest" {
		secret, err := h.svc.GetActiveHMACSecret(r.Context(), tenantID)
		if err != nil {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSON(w, http.StatusOK, secret)
		return
	}

	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
		writeError(w, http.StatusBadRequest, "invalid version")
		return
	}

	secret, err := h.svc.GetHMACSecret(r.Context(), tenantID, version)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	writeJSON(w, http.StatusOK, secret)
}

func (h *Handler) RevokeHMACSecretHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r, r.URL.Query().Get("tenant_id"))
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	versionStr := chi.URLParam(r, "version")
	var version int
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
		writeError(w, http.StatusBadRequest, "invalid version")
		return
	}

	err = h.svc.RevokeHMACSecret(r.Context(), tenantID, version)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "revoked"})
}

func (h *Handler) BulkCreateHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Links []struct {
			TargetURL    string `json:"target_url"`
			TTL          string `json:"ttl"`
			WebhookURL   string `json:"webhook_url"`
			CustomDomain string `json:"custom_domain"`
		} `json:"links"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Links) == 0 {
		writeError(w, http.StatusBadRequest, "links: required and non-empty")
		return
	}

	items := make([]shortlinksvc.BulkCreateItem, 0, len(req.Links))
	for _, l := range req.Links {
		if !isValidURL(l.TargetURL) {
			writeError(w, http.StatusBadRequest, "target_url: valid URL required")
			return
		}
		var ttl time.Duration
		if l.TTL != "" {
			ttl, _ = time.ParseDuration(l.TTL)
		}
		item := shortlinksvc.BulkCreateItem{TargetURL: l.TargetURL, TTL: ttl}
		if l.WebhookURL != "" {
			item.WebhookURL = &l.WebhookURL
		}
		if l.CustomDomain != "" {
			item.CustomDomain = &l.CustomDomain
		}
		items = append(items, item)
	}
	resp, err := h.svc.BulkCreate(r.Context(), shortlinksvc.BulkCreateRequest{Links: items})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) BulkImportHandler(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.svc.ImportCSV(r.Context(), string(data))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) BulkExportHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Format string `json:"format"`
	}
	if err := decodeOptionalJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json payload: "+err.Error())
		return
	}
	expReq := shortlinksvc.ExportRequest{Format: req.Format}
	if req.Format == "json" {
		resp, err := h.svc.ExportJSON(r.Context(), expReq)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}
	resp, err := h.svc.ExportCSV(r.Context(), expReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) BulkQRZipHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Codes []string `json:"codes"`
		Size  int      `json:"size"`
		Level string   `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Codes) == 0 || len(req.Codes) > 1000 {
		writeError(w, http.StatusBadRequest, "codes: 1-1000 required")
		return
	}
	zipData, err := h.svc.GenerateQRZip(r.Context(), req.Codes, req.Size, req.Level, getShortBaseURL(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"qr-codes.zip\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(zipData)
}

func getShortBaseURL(r *http.Request) string {
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + r.Host + "/s"
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.initOnce.Do(func() {
		if h.router == nil {
			router := chi.NewRouter()
			h.RegisterRoutes(router)
			h.router = router
		}
	})

	// Align request path: if server router mounts /api/v1/admin/shortlinks,
	// normalize the path prefix to /admin/api/v1/shortlinks expected by the internal router.
	path := r.URL.Path
	if strings.HasPrefix(path, "/api/v1/admin/shortlinks") {
		trimmed := strings.TrimPrefix(path, "/api/v1/admin/shortlinks")
		r2 := new(http.Request)
		*r2 = *r
		u2 := new(url.URL)
		*u2 = *r.URL
		u2.Path = "/admin/api/v1/shortlinks" + trimmed
		r2.URL = u2
		h.router.ServeHTTP(w, r2)
		return
	}

	h.router.ServeHTTP(w, r)
}

// AdminAuthMiddleware validates callers via OIDC Bearer tokens and DB identity resolution,
// strictly preventing header spoofing and organization takeover attacks.
func (h *Handler) AdminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If TokenValidator & Resolver are wired, enforce strict cryptographic OIDC & DB membership
		if h.validator != nil && h.resolver != nil {
			authHeader := r.Header.Get("Authorization")
			token, ok := extractBearer(authHeader)
			if !ok {
				writeError(w, http.StatusUnauthorized, "authentication_required")
				return
			}

			principal, err := h.validator.Validate(r.Context(), token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid_token")
				return
			}

			membership, err := h.resolver.Resolve(r.Context(), identity.PrincipalKey{
				Issuer:  principal.Issuer,
				Subject: principal.Subject,
			})
			if err != nil {
				writeError(w, http.StatusForbidden, "forbidden")
				return
			}

			// Securely bind authenticated context
			ctx := r.Context()
			ctx = context.WithValue(ctx, ctxTenantID, membership.TenantID)
			ctx = context.WithValue(ctx, ctxOrgID, membership.OrganizationID)
			ctx = context.WithValue(ctx, ctxActorID, membership.ActorID)
			ctx = context.WithValue(ctx, ctxMembership, membership)
			ctx = oidchttp.WithOrganizationContext(ctx, identity.OrganizationContext{
				TenantID:       membership.TenantID,
				OrganizationID: membership.OrganizationID,
				ActorID:        membership.ActorID,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Fallback for tests/unauthenticated standalone mode: require headers
		tenant := r.Header.Get("X-Tenant-ID")
		org := r.Header.Get("X-Organization-ID")
		if tenant == "" || org == "" {
			writeError(w, http.StatusBadRequest, "X-Tenant-ID and X-Organization-ID required")
			return
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, ctxTenantID, tenant)
		ctx = context.WithValue(ctx, ctxOrgID, org)
		ctx = oidchttp.WithOrganizationContext(ctx, identity.OrganizationContext{
			TenantID:       tenant,
			OrganizationID: org,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractBearer(header string) (string, bool) {
	if header == "" {
		return "", false
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	t := strings.TrimSpace(parts[1])
	if t == "" {
		return "", false
	}
	return t, true
}

// Anomaly Rule Handlers

func (h *Handler) CreateAnomalyRuleHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID    string                `json:"tenant_id"`
		Name        string                `json:"name"`
		Description string                `json:"description"`
		Type        string                `json:"type"`
		Config      shortlink.AlertConfig `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Type) == "" {
		writeError(w, http.StatusBadRequest, "name and type are required")
		return
	}

	tenantID, err := h.getTenantID(r, req.TenantID)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	rule, err := h.svc.CreateAnomalyRule(r.Context(), shortlink.CreateAnomalyRuleRequest{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Type:        shortlink.AnomalyType(req.Type),
		Config:      req.Config,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, rule)
}

func (h *Handler) ListAnomalyRulesHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r, r.URL.Query().Get("tenant_id"))
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	var enabled *bool
	if e := r.URL.Query().Get("enabled"); e != "" {
		val := e == "true"
		enabled = &val
	}

	rules, total, err := h.svc.ListAnomalyRules(r.Context(), shortlink.ListAnomalyRulesRequest{
		TenantID: tenantID,
		Limit:    limit,
		Offset:   offset,
		Enabled:  enabled,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"rules":  rules,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) GetAnomalyRuleHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	rule, err := h.svc.GetAnomalyRule(r.Context(), id)
	if err != nil {
		if errors.Is(err, shortlink.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, rule)
}

func (h *Handler) UpdateAnomalyRuleHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		Name        *string                `json:"name"`
		Description *string                `json:"description"`
		Config      *shortlink.AlertConfig `json:"config"`
		Enabled     *bool                  `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	rule, err := h.svc.UpdateAnomalyRule(r.Context(), id, shortlink.UpdateAnomalyRuleRequest{
		Name:        req.Name,
		Description: req.Description,
		Config:      req.Config,
		Enabled:     req.Enabled,
	})
	if err != nil {
		if errors.Is(err, shortlink.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, rule)
}

func (h *Handler) DeleteAnomalyRuleHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	err := h.svc.DeleteAnomalyRule(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Anomaly Alert Handlers

func (h *Handler) ListAnomalyAlertsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r, r.URL.Query().Get("tenant_id"))
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	var shortLinkCode *string
	if s := r.URL.Query().Get("short_link_code"); s != "" {
		shortLinkCode = &s
	}

	var ruleID *int64
	if s := r.URL.Query().Get("rule_id"); s != "" {
		var id int64
		if _, err := fmt.Sscanf(s, "%d", &id); err == nil {
			ruleID = &id
		}
	}

	var status *shortlink.AlertStatus
	if s := r.URL.Query().Get("status"); s != "" {
		val := shortlink.AlertStatus(s)
		status = &val
	}

	var alertType *shortlink.AnomalyType
	if t := r.URL.Query().Get("type"); t != "" {
		val := shortlink.AnomalyType(t)
		alertType = &val
	}

	var since *time.Time
	if s := r.URL.Query().Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = &t
		}
	}

	alerts, total, err := h.svc.ListAnomalyAlerts(r.Context(), shortlink.ListAnomalyAlertsRequest{
		TenantID:      tenantID,
		ShortLinkCode: shortLinkCode,
		RuleID:        ruleID,
		Status:        status,
		Type:          alertType,
		Since:         since,
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"alerts": alerts,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) GetAnomalyAlertHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	alert, err := h.svc.GetAnomalyAlert(r.Context(), id)
	if err != nil {
		if errors.Is(err, shortlink.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, alert)
}

func (h *Handler) UpdateAnomalyAlertHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		Status         *shortlink.AlertStatus `json:"status"`
		AcknowledgedBy *string                `json:"acknowledged_by"`
		ResolvedBy     *string                `json:"resolved_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	alert, err := h.svc.UpdateAnomalyAlert(r.Context(), id, shortlink.UpdateAnomalyAlertRequest{
		Status:         req.Status,
		AcknowledgedBy: req.AcknowledgedBy,
		ResolvedBy:     req.ResolvedBy,
	})
	if err != nil {
		if errors.Is(err, shortlink.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, alert)
}

func (h *Handler) AcknowledgeAnomalyAlertHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		AcknowledgedBy string `json:"acknowledged_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(req.AcknowledgedBy) == "" {
		writeError(w, http.StatusBadRequest, "acknowledged_by is required")
		return
	}

	status := shortlink.AlertStatusAcknowledged
	alert, err := h.svc.UpdateAnomalyAlert(r.Context(), id, shortlink.UpdateAnomalyAlertRequest{
		Status:         &status,
		AcknowledgedBy: &req.AcknowledgedBy,
	})
	if err != nil {
		if errors.Is(err, shortlink.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, alert)
}

func (h *Handler) ResolveAnomalyAlertHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		ResolvedBy string `json:"resolved_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(req.ResolvedBy) == "" {
		writeError(w, http.StatusBadRequest, "resolved_by is required")
		return
	}

	status := shortlink.AlertStatusResolved
	alert, err := h.svc.UpdateAnomalyAlert(r.Context(), id, shortlink.UpdateAnomalyAlertRequest{
		Status:     &status,
		ResolvedBy: &req.ResolvedBy,
	})
	if err != nil {
		if errors.Is(err, shortlink.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, alert)
}

// Dashboard Analytics Handlers

func (h *Handler) AnalyticsOverviewHandler(w http.ResponseWriter, r *http.Request) {
	req := h.parseAnalyticsRequest(r)
	if req.TenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	analytics, err := h.svc.GetDashboardAnalytics(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Add cache headers
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Header().Set("ETag", fmt.Sprintf(`"%d-%d"`, analytics.TotalScans, analytics.UniqueIPs))

	writeJSON(w, http.StatusOK, analytics)
}

func (h *Handler) AnalyticsTimeSeriesHandler(w http.ResponseWriter, r *http.Request) {
	req := h.parseTimeSeriesRequest(r)
	if req.TenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	points, err := h.svc.GetTimeSeries(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, map[string]any{"data": points})
}

func (h *Handler) AnalyticsGeoHandler(w http.ResponseWriter, r *http.Request) {
	req := h.parseGeoHeatmapRequest(r)
	if req.TenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	points, err := h.svc.GetGeoHeatmap(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=120")
	writeJSON(w, http.StatusOK, map[string]any{"data": points})
}

func (h *Handler) AnalyticsDevicesHandler(w http.ResponseWriter, r *http.Request) {
	req := h.parseDeviceAnalyticsRequest(r)
	if req.TenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	devices, osList, browsers, err := h.svc.GetDeviceAnalytics(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=120")
	writeJSON(w, http.StatusOK, map[string]any{
		"devices":  devices,
		"os":       osList,
		"browsers": browsers,
	})
}

func (h *Handler) AnalyticsFunnelHandler(w http.ResponseWriter, r *http.Request) {
	req := h.parseFunnelRequest(r)
	if req.TenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	funnel, err := h.svc.GetFunnel(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, funnel)
}

func (h *Handler) AnalyticsTopAssetsHandler(w http.ResponseWriter, r *http.Request) {
	req := h.parseTopAssetsRequest(r)
	if req.TenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	assets, err := h.svc.GetTopAssets(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, map[string]any{"data": assets})
}

func (h *Handler) parseAnalyticsRequest(r *http.Request) shortlink.AnalyticsRequest {
	tenantID, _ := h.getTenantID(r, r.URL.Query().Get("tenant_id"))
	timeRange := shortlink.TimeRange(defaultQuery(r.URL.Query(), "time_range", "24h"))
	var customStart, customEnd *time.Time
	if s := r.URL.Query().Get("custom_start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			customStart = &t
		}
	}
	if e := r.URL.Query().Get("custom_end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			customEnd = &t
		}
	}
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	return shortlink.AnalyticsRequest{
		TenantID:    tenantID,
		TimeRange:   timeRange,
		CustomStart: customStart,
		CustomEnd:   customEnd,
		Limit:       limit,
	}
}

func (h *Handler) parseTimeSeriesRequest(r *http.Request) shortlink.TimeSeriesRequest {
	tenantID, _ := h.getTenantID(r, r.URL.Query().Get("tenant_id"))
	timeRange := shortlink.TimeRange(defaultQuery(r.URL.Query(), "time_range", "24h"))
	var customStart, customEnd *time.Time
	if s := r.URL.Query().Get("custom_start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			customStart = &t
		}
	}
	if e := r.URL.Query().Get("custom_end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			customEnd = &t
		}
	}
	interval := defaultQuery(r.URL.Query(), "interval", "1h")

	return shortlink.TimeSeriesRequest{
		TenantID:    tenantID,
		TimeRange:   timeRange,
		CustomStart: customStart,
		CustomEnd:   customEnd,
		Interval:    interval,
	}
}

func (h *Handler) parseGeoHeatmapRequest(r *http.Request) shortlink.GeoHeatmapRequest {
	tenantID, _ := h.getTenantID(r, r.URL.Query().Get("tenant_id"))
	timeRange := shortlink.TimeRange(defaultQuery(r.URL.Query(), "time_range", "24h"))
	var customStart, customEnd *time.Time
	if s := r.URL.Query().Get("custom_start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			customStart = &t
		}
	}
	if e := r.URL.Query().Get("custom_end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			customEnd = &t
		}
	}

	return shortlink.GeoHeatmapRequest{
		TenantID:    tenantID,
		TimeRange:   timeRange,
		CustomStart: customStart,
		CustomEnd:   customEnd,
		Country:     r.URL.Query().Get("country"),
	}
}

func (h *Handler) parseDeviceAnalyticsRequest(r *http.Request) shortlink.DeviceAnalyticsRequest {
	tenantID, _ := h.getTenantID(r, r.URL.Query().Get("tenant_id"))
	timeRange := shortlink.TimeRange(defaultQuery(r.URL.Query(), "time_range", "24h"))
	var customStart, customEnd *time.Time
	if s := r.URL.Query().Get("custom_start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			customStart = &t
		}
	}
	if e := r.URL.Query().Get("custom_end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			customEnd = &t
		}
	}

	return shortlink.DeviceAnalyticsRequest{
		TenantID:    tenantID,
		TimeRange:   timeRange,
		CustomStart: customStart,
		CustomEnd:   customEnd,
	}
}

func (h *Handler) parseFunnelRequest(r *http.Request) shortlink.FunnelRequest {
	tenantID, _ := h.getTenantID(r, r.URL.Query().Get("tenant_id"))
	timeRange := shortlink.TimeRange(defaultQuery(r.URL.Query(), "time_range", "24h"))
	var customStart, customEnd *time.Time
	if s := r.URL.Query().Get("custom_start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			customStart = &t
		}
	}
	if e := r.URL.Query().Get("custom_end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			customEnd = &t
		}
	}

	return shortlink.FunnelRequest{
		TenantID:    tenantID,
		TimeRange:   timeRange,
		CustomStart: customStart,
		CustomEnd:   customEnd,
	}
}

func (h *Handler) parseTopAssetsRequest(r *http.Request) shortlink.TopAssetsRequest {
	tenantID, _ := h.getTenantID(r, r.URL.Query().Get("tenant_id"))
	timeRange := shortlink.TimeRange(defaultQuery(r.URL.Query(), "time_range", "24h"))
	var customStart, customEnd *time.Time
	if s := r.URL.Query().Get("custom_start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			customStart = &t
		}
	}
	if e := r.URL.Query().Get("custom_end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			customEnd = &t
		}
	}
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	return shortlink.TopAssetsRequest{
		TenantID:    tenantID,
		TimeRange:   timeRange,
		CustomStart: customStart,
		CustomEnd:   customEnd,
		Limit:       limit,
	}
}

func defaultQuery(q url.Values, key, fallback string) string {
	if v := q.Get(key); v != "" {
		return v
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeOptionalJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return nil
	}
	err := json.NewDecoder(r.Body).Decode(dst)
	if err == nil || errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

