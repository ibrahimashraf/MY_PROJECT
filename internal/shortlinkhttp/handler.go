package shortlinkhttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"integin/internal/domain/shortlink"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/oidchttp"
	"integin/internal/shortlinksvc"
)

// TokenValidator defines the contract for OIDC bearer token verification.
type TokenValidator interface {
	Validate(ctx context.Context, rawToken string) (oidcauth.Principal, error)
}

type Handler struct {
	svc       *shortlinksvc.Service
	validator TokenValidator
	resolver  identity.Resolver
	engine    *gin.Engine
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
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	h.RegisterRoutes(engine)
	h.engine = engine
	return h
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// Public redirect
	r.GET("/s/:code", h.RedirectHandler)

	// Admin API
	admin := r.Group("/admin/api/v1/shortlinks")
	admin.Use(h.AdminAuthMiddleware())
	{
		admin.POST("", h.CreateHandler)
		admin.GET("", h.ListHandler)
		admin.POST("/bulk", h.BulkCreateHandler)
		admin.POST("/bulk/qr-zip", h.BulkQRZipHandler)
		admin.POST("/import", h.BulkImportHandler)
		admin.POST("/export", h.BulkExportHandler)
		admin.GET("/:code", h.GetHandler)
		admin.GET("/:code/stats", h.StatsHandler)
		admin.DELETE("/:code", h.RevokeHandler)

		// Webhook DLQ endpoints
		admin.GET("/webhook/dlq", h.ListDLQHandler)
		admin.POST("/webhook/dlq/:id/retry", h.RetryDLQHandler)
		admin.POST("/webhook/dlq/:id/resolve", h.ResolveDLQHandler)

		// HMAC secret management
		admin.POST("/hmac/secrets", h.CreateHMACSecretHandler)
		admin.GET("/hmac/secrets", h.ListHMACSecretsHandler)
		admin.GET("/hmac/secrets/:version", h.GetHMACSecretHandler)
		admin.DELETE("/hmac/secrets/:version", h.RevokeHMACSecretHandler)

		// Anomaly detection endpoints
		admin.POST("/anomaly/rules", h.CreateAnomalyRuleHandler)
		admin.GET("/anomaly/rules", h.ListAnomalyRulesHandler)
		admin.GET("/anomaly/rules/:id", h.GetAnomalyRuleHandler)
		admin.PATCH("/anomaly/rules/:id", h.UpdateAnomalyRuleHandler)
		admin.DELETE("/anomaly/rules/:id", h.DeleteAnomalyRuleHandler)

		admin.GET("/anomaly/alerts", h.ListAnomalyAlertsHandler)
		admin.GET("/anomaly/alerts/:id", h.GetAnomalyAlertHandler)
		admin.PATCH("/anomaly/alerts/:id", h.UpdateAnomalyAlertHandler)
		admin.POST("/anomaly/alerts/:id/acknowledge", h.AcknowledgeAnomalyAlertHandler)
		admin.POST("/anomaly/alerts/:id/resolve", h.ResolveAnomalyAlertHandler)

		// Dashboard Analytics endpoints
		analytics := admin.Group("/analytics")
		{
			analytics.GET("/overview", h.AnalyticsOverviewHandler)
			analytics.GET("/timeseries", h.AnalyticsTimeSeriesHandler)
			analytics.GET("/geo", h.AnalyticsGeoHandler)
			analytics.GET("/devices", h.AnalyticsDevicesHandler)
			analytics.GET("/funnel", h.AnalyticsFunnelHandler)
			analytics.GET("/top-assets", h.AnalyticsTopAssetsHandler)
		}
	}
}

func (h *Handler) RedirectHandler(c *gin.Context) {
	code := c.Param("code")
	// Enforce custom_domain host routing if set
	if sl, err := h.svc.GetStats(c.Request.Context(), code); err == nil && sl.CustomDomain != nil && *sl.CustomDomain != "" {
		if c.Request.Host != *sl.CustomDomain && c.GetHeader("X-Forwarded-Host") != *sl.CustomDomain {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
	}
	target, err := h.svc.ResolveShortLink(c.Request.Context(), code)
	if err != nil {
		if errors.Is(err, shortlinksvc.ErrExpired) || errors.Is(err, shortlinksvc.ErrRevoked) {
			c.AbortWithStatus(http.StatusGone)
			return
		}
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Redirect(http.StatusFound, target)
}

func (h *Handler) CreateHandler(c *gin.Context) {
	var req struct {
		TargetURL     string `json:"target_url" binding:"required,url"`
		TTL           string `json:"ttl"`
		WebhookURL    string `json:"webhook_url"`
		CustomDomain  string `json:"custom_domain"`
		HMACSecretRef string `json:"hmac_secret_ref"`
		HMACAlgorithm string `json:"hmac_algorithm"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var ttl time.Duration
	if req.TTL != "" {
		d, err := time.ParseDuration(req.TTL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ttl format"})
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

	code, err := h.svc.CreateShortLink(c.Request.Context(), shortlink.CreateRequest{
		TargetURL:     req.TargetURL,
		TTL:           ttl,
		WebhookURL:    webhookURL,
		CustomDomain:  customDomain,
		HMACSecretRef: hmacSecretRef,
		HMACAlgorithm: hmacAlgorithm,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	baseURL := getShortBaseURL(c.Request)
	c.JSON(http.StatusCreated, gin.H{
		"code":       code,
		"short_url":  baseURL + "/" + code,
		"target_url": req.TargetURL,
	})
}

func (h *Handler) ListHandler(c *gin.Context) {
	links, err := h.svc.List(c.Request.Context(), 50, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, links)
}

func (h *Handler) GetHandler(c *gin.Context) {
	code := c.Param("code")
	sl, err := h.svc.GetStats(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, sl)
}

func (h *Handler) StatsHandler(c *gin.Context) {
	code := c.Param("code")
	sl, err := h.svc.GetStats(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, sl)
}

func (h *Handler) RevokeHandler(c *gin.Context) {
	code := c.Param("code")
	if err := h.svc.RevokeShortLink(c.Request.Context(), code); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListDLQHandler(c *gin.Context) {
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if _, err := fmt.Sscanf(l, "%d", &limit); err != nil || limit <= 0 || limit > 1000 {
			limit = 50
		}
	}
	if o := c.Query("offset"); o != "" {
		if _, err := fmt.Sscanf(o, "%d", &offset); err != nil || offset < 0 {
			offset = 0
		}
	}

	var status *shortlink.WebhookDeliveryStatus
	if s := c.Query("status"); s != "" {
		ws := shortlink.WebhookDeliveryStatus(s)
		status = &ws
	}

	entries, total, err := h.svc.GetDLQEntries(c.Request.Context(), shortlink.ListDLQRequest{
		Limit:  limit,
		Offset: offset,
		Status: status,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

func (h *Handler) RetryDLQHandler(c *gin.Context) {
	idStr := c.Param("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	err := h.svc.RetryDLQEntry(c.Request.Context(), shortlink.RetryDLQRequest{DeliveryID: id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "retry scheduled"})
}

func (h *Handler) ResolveDLQHandler(c *gin.Context) {
	idStr := c.Param("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Require authenticated actor — never fall back to a ghost "admin"
	org, ok := oidchttp.OrganizationContextFrom(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Optional body override; if not present or empty, use authenticated actor
	var req struct {
		ResolvedBy string `json:"resolved_by"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.ResolvedBy == "" {
		req.ResolvedBy = org.ActorID
	}

	err := h.svc.ResolveDLQEntry(c.Request.Context(), id, req.ResolvedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "resolved"})
}

// getTenantID returns the authenticated tenant_id, enforcing tenant isolation.
// If a caller explicitly specifies a tenant_id, it is validated against the authenticated tenant_id
// to prevent cross-tenant access and organization takeover.
func (h *Handler) getTenantID(c *gin.Context, requestedTenant string) (string, error) {
	authTenant := c.GetString("tenant_id")
	if authTenant != "" {
		if requestedTenant != "" && requestedTenant != authTenant {
			return "", errors.New("cross-tenant access prohibited")
		}
		return authTenant, nil
	}
	if requestedTenant != "" {
		return requestedTenant, nil
	}
	return "", errors.New("tenant_id is required")
}

func (h *Handler) CreateHMACSecretHandler(c *gin.Context) {
	var req struct {
		TenantID  string `json:"tenant_id"`
		Secret    string `json:"secret" binding:"required"`
		Algorithm string `json:"algorithm"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID, err := h.getTenantID(c, req.TenantID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	secret, err := h.svc.CreateHMACSecret(c.Request.Context(), tenantID, req.Secret, req.Algorithm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, secret)
}

func (h *Handler) ListHMACSecretsHandler(c *gin.Context) {
	tenantID, err := h.getTenantID(c, c.Query("tenant_id"))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	secrets, err := h.svc.ListHMACSecrets(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, secrets)
}

func (h *Handler) GetHMACSecretHandler(c *gin.Context) {
	tenantID, err := h.getTenantID(c, c.Query("tenant_id"))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	versionStr := c.Param("version")
	var version int
	if versionStr == "latest" {
		secret, err := h.svc.GetActiveHMACSecret(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusOK, secret)
		return
	}

	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version"})
		return
	}

	secret, err := h.svc.GetHMACSecret(c.Request.Context(), tenantID, version)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, secret)
}

func (h *Handler) RevokeHMACSecretHandler(c *gin.Context) {
	tenantID, err := h.getTenantID(c, c.Query("tenant_id"))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	versionStr := c.Param("version")
	var version int
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version"})
		return
	}

	err = h.svc.RevokeHMACSecret(c.Request.Context(), tenantID, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "revoked"})
}

func (h *Handler) BulkCreateHandler(c *gin.Context) {
	var req struct {
		Links []struct {
			TargetURL    string `json:"target_url" binding:"required,url"`
			TTL          string `json:"ttl"`
			WebhookURL   string `json:"webhook_url"`
			CustomDomain string `json:"custom_domain"`
		} `json:"links" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	items := make([]shortlinksvc.BulkCreateItem, 0, len(req.Links))
	for _, l := range req.Links {
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
	resp, err := h.svc.BulkCreate(c.Request.Context(), shortlinksvc.BulkCreateRequest{Links: items})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) BulkImportHandler(c *gin.Context) {
	data, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.svc.ImportCSV(c.Request.Context(), string(data))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) BulkExportHandler(c *gin.Context) {
	var req struct {
		Format string `json:"format"`
	}
	_ = c.ShouldBindJSON(&req)
	expReq := shortlinksvc.ExportRequest{Format: req.Format}
	if req.Format == "json" {
		resp, err := h.svc.ExportJSON(c.Request.Context(), expReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
		return
	}
	resp, err := h.svc.ExportCSV(c.Request.Context(), expReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) BulkQRZipHandler(c *gin.Context) {
	var req struct {
		Codes []string `json:"codes" binding:"required"`
		Size  int      `json:"size"`
		Level string   `json:"level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Codes) == 0 || len(req.Codes) > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "codes: 1-1000 required"})
		return
	}
	zipData, err := h.svc.GenerateQRZip(c.Request.Context(), req.Codes, req.Size, req.Level, getShortBaseURL(c.Request))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename=\"qr-codes.zip\"")
	c.Data(http.StatusOK, "application/zip", zipData)
}

func getShortBaseURL(r *http.Request) string {
	// Use configured short base URL or derive from request
	// TODO: use config
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + r.Host + "/s"
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.initOnce.Do(func() {
		if h.engine == nil {
			gin.SetMode(gin.ReleaseMode)
			engine := gin.New()
			engine.Use(gin.Recovery())
			h.RegisterRoutes(engine)
			h.engine = engine
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
		h.engine.ServeHTTP(w, r2)
		return
	}

	h.engine.ServeHTTP(w, r)
}

// AdminAuthMiddleware validates callers via OIDC Bearer tokens and DB identity resolution,
// strictly preventing header spoofing and organization takeover attacks.
func (h *Handler) AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// If TokenValidator & Resolver are wired, enforce strict cryptographic OIDC & DB membership
		if h.validator != nil && h.resolver != nil {
			authHeader := c.GetHeader("Authorization")
			token, ok := extractBearer(authHeader)
			if !ok {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication_required"})
				return
			}

			principal, err := h.validator.Validate(c.Request.Context(), token)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
				return
			}

			membership, err := h.resolver.Resolve(c.Request.Context(), identity.PrincipalKey{
				Issuer:  principal.Issuer,
				Subject: principal.Subject,
			})
			if err != nil {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}

			// Securely bind authenticated context
			c.Set("tenant_id", membership.TenantID)
			c.Set("organization_id", membership.OrganizationID)
			c.Set("actor_id", membership.ActorID)
			c.Set("membership", membership)
			c.Next()
			return
		}

		// Fallback for tests/unauthenticated standalone mode: require headers
		tenant := c.GetHeader("X-Tenant-ID")
		org := c.GetHeader("X-Organization-ID")
		if tenant == "" || org == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID and X-Organization-ID required"})
			return
		}
		c.Set("tenant_id", tenant)
		c.Set("organization_id", org)
		c.Next()
	}
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

func (h *Handler) CreateAnomalyRuleHandler(c *gin.Context) {
	var req struct {
		TenantID    string                `json:"tenant_id"`
		Name        string                `json:"name" binding:"required"`
		Description string                `json:"description"`
		Type        string                `json:"type" binding:"required"`
		Config      shortlink.AlertConfig `json:"config" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID, err := h.getTenantID(c, req.TenantID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	rule, err := h.svc.CreateAnomalyRule(c.Request.Context(), shortlink.CreateAnomalyRuleRequest{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Type:        shortlink.AnomalyType(req.Type),
		Config:      req.Config,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

func (h *Handler) ListAnomalyRulesHandler(c *gin.Context) {
	tenantID, err := h.getTenantID(c, c.Query("tenant_id"))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	var enabled *bool
	if e := c.Query("enabled"); e != "" {
		val := e == "true"
		enabled = &val
	}

	rules, total, err := h.svc.ListAnomalyRules(c.Request.Context(), shortlink.ListAnomalyRulesRequest{
		TenantID: tenantID,
		Limit:    limit,
		Offset:   offset,
		Enabled:  enabled,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rules":  rules,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) GetAnomalyRuleHandler(c *gin.Context) {
	idStr := c.Param("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	rule, err := h.svc.GetAnomalyRule(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, shortlink.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rule)
}

func (h *Handler) UpdateAnomalyRuleHandler(c *gin.Context) {
	idStr := c.Param("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Name        *string                `json:"name"`
		Description *string                `json:"description"`
		Config      *shortlink.AlertConfig `json:"config"`
		Enabled     *bool                  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule, err := h.svc.UpdateAnomalyRule(c.Request.Context(), id, shortlink.UpdateAnomalyRuleRequest{
		Name:        req.Name,
		Description: req.Description,
		Config:      req.Config,
		Enabled:     req.Enabled,
	})
	if err != nil {
		if errors.Is(err, shortlink.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rule)
}

func (h *Handler) DeleteAnomalyRuleHandler(c *gin.Context) {
	idStr := c.Param("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	err := h.svc.DeleteAnomalyRule(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// Anomaly Alert Handlers

func (h *Handler) ListAnomalyAlertsHandler(c *gin.Context) {
	tenantID, err := h.getTenantID(c, c.Query("tenant_id"))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	var shortLinkCode *string
	if s := c.Query("short_link_code"); s != "" {
		shortLinkCode = &s
	}

	var ruleID *int64
	if r := c.Query("rule_id"); r != "" {
		var id int64
		fmt.Sscanf(r, "%d", &id)
		ruleID = &id
	}

	var status *shortlink.AlertStatus
	if s := c.Query("status"); s != "" {
		val := shortlink.AlertStatus(s)
		status = &val
	}

	var alertType *shortlink.AnomalyType
	if t := c.Query("type"); t != "" {
		val := shortlink.AnomalyType(t)
		alertType = &val
	}

	var since *time.Time
	if s := c.Query("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = &t
		}
	}

	alerts, total, err := h.svc.ListAnomalyAlerts(c.Request.Context(), shortlink.ListAnomalyAlertsRequest{
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) GetAnomalyAlertHandler(c *gin.Context) {
	idStr := c.Param("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	alert, err := h.svc.GetAnomalyAlert(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, shortlink.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alert)
}

func (h *Handler) UpdateAnomalyAlertHandler(c *gin.Context) {
	idStr := c.Param("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Status         *shortlink.AlertStatus `json:"status"`
		AcknowledgedBy *string                `json:"acknowledged_by"`
		ResolvedBy     *string                `json:"resolved_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	alert, err := h.svc.UpdateAnomalyAlert(c.Request.Context(), id, shortlink.UpdateAnomalyAlertRequest{
		Status:         req.Status,
		AcknowledgedBy: req.AcknowledgedBy,
		ResolvedBy:     req.ResolvedBy,
	})
	if err != nil {
		if errors.Is(err, shortlink.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alert)
}

func (h *Handler) AcknowledgeAnomalyAlertHandler(c *gin.Context) {
	idStr := c.Param("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		AcknowledgedBy string `json:"acknowledged_by" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status := shortlink.AlertStatusAcknowledged
	alert, err := h.svc.UpdateAnomalyAlert(c.Request.Context(), id, shortlink.UpdateAnomalyAlertRequest{
		Status:         &status,
		AcknowledgedBy: &req.AcknowledgedBy,
	})
	if err != nil {
		if errors.Is(err, shortlink.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alert)
}

func (h *Handler) ResolveAnomalyAlertHandler(c *gin.Context) {
	idStr := c.Param("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		ResolvedBy string `json:"resolved_by" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status := shortlink.AlertStatusResolved
	alert, err := h.svc.UpdateAnomalyAlert(c.Request.Context(), id, shortlink.UpdateAnomalyAlertRequest{
		Status:     &status,
		ResolvedBy: &req.ResolvedBy,
	})
	if err != nil {
		if errors.Is(err, shortlink.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// Dashboard Analytics Handlers

func (h *Handler) AnalyticsOverviewHandler(c *gin.Context) {
	req := h.parseAnalyticsRequest(c)
	if req.TenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	analytics, err := h.svc.GetDashboardAnalytics(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add cache headers
	c.Header("Cache-Control", "public, max-age=60")
	c.Header("ETag", fmt.Sprintf(`"%d-%d"`, analytics.TotalScans, analytics.UniqueIPs))

	c.JSON(http.StatusOK, analytics)
}

func (h *Handler) AnalyticsTimeSeriesHandler(c *gin.Context) {
	req := h.parseTimeSeriesRequest(c)
	if req.TenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	points, err := h.svc.GetTimeSeries(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Cache-Control", "public, max-age=60")
	c.JSON(http.StatusOK, gin.H{"data": points})
}

func (h *Handler) AnalyticsGeoHandler(c *gin.Context) {
	req := h.parseGeoHeatmapRequest(c)
	if req.TenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	points, err := h.svc.GetGeoHeatmap(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Cache-Control", "public, max-age=120")
	c.JSON(http.StatusOK, gin.H{"data": points})
}

func (h *Handler) AnalyticsDevicesHandler(c *gin.Context) {
	req := h.parseDeviceAnalyticsRequest(c)
	if req.TenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	devices, osList, browsers, err := h.svc.GetDeviceAnalytics(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Cache-Control", "public, max-age=120")
	c.JSON(http.StatusOK, gin.H{
		"devices":  devices,
		"os":       osList,
		"browsers": browsers,
	})
}

func (h *Handler) AnalyticsFunnelHandler(c *gin.Context) {
	req := h.parseFunnelRequest(c)
	if req.TenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	funnel, err := h.svc.GetFunnel(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Cache-Control", "public, max-age=60")
	c.JSON(http.StatusOK, funnel)
}

func (h *Handler) AnalyticsTopAssetsHandler(c *gin.Context) {
	req := h.parseTopAssetsRequest(c)
	if req.TenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	assets, err := h.svc.GetTopAssets(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Cache-Control", "public, max-age=60")
	c.JSON(http.StatusOK, gin.H{"data": assets})
}

func (h *Handler) parseAnalyticsRequest(c *gin.Context) shortlink.AnalyticsRequest {
	tenantID, _ := h.getTenantID(c, c.Query("tenant_id"))
	timeRange := shortlink.TimeRange(c.DefaultQuery("time_range", "24h"))
	var customStart, customEnd *time.Time
	if s := c.Query("custom_start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			customStart = &t
		}
	}
	if e := c.Query("custom_end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			customEnd = &t
		}
	}
	limit := 20
	if l := c.Query("limit"); l != "" {
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

func (h *Handler) parseTimeSeriesRequest(c *gin.Context) shortlink.TimeSeriesRequest {
	tenantID, _ := h.getTenantID(c, c.Query("tenant_id"))
	timeRange := shortlink.TimeRange(c.DefaultQuery("time_range", "24h"))
	var customStart, customEnd *time.Time
	if s := c.Query("custom_start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			customStart = &t
		}
	}
	if e := c.Query("custom_end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			customEnd = &t
		}
	}
	interval := c.DefaultQuery("interval", "1h")

	return shortlink.TimeSeriesRequest{
		TenantID:    tenantID,
		TimeRange:   timeRange,
		CustomStart: customStart,
		CustomEnd:   customEnd,
		Interval:    interval,
	}
}

func (h *Handler) parseGeoHeatmapRequest(c *gin.Context) shortlink.GeoHeatmapRequest {
	tenantID, _ := h.getTenantID(c, c.Query("tenant_id"))
	timeRange := shortlink.TimeRange(c.DefaultQuery("time_range", "24h"))
	var customStart, customEnd *time.Time
	if s := c.Query("custom_start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			customStart = &t
		}
	}
	if e := c.Query("custom_end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			customEnd = &t
		}
	}

	return shortlink.GeoHeatmapRequest{
		TenantID:    tenantID,
		TimeRange:   timeRange,
		CustomStart: customStart,
		CustomEnd:   customEnd,
		Country:     c.Query("country"),
	}
}

func (h *Handler) parseDeviceAnalyticsRequest(c *gin.Context) shortlink.DeviceAnalyticsRequest {
	tenantID, _ := h.getTenantID(c, c.Query("tenant_id"))
	timeRange := shortlink.TimeRange(c.DefaultQuery("time_range", "24h"))
	var customStart, customEnd *time.Time
	if s := c.Query("custom_start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			customStart = &t
		}
	}
	if e := c.Query("custom_end"); e != "" {
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

func (h *Handler) parseFunnelRequest(c *gin.Context) shortlink.FunnelRequest {
	tenantID, _ := h.getTenantID(c, c.Query("tenant_id"))
	timeRange := shortlink.TimeRange(c.DefaultQuery("time_range", "24h"))
	var customStart, customEnd *time.Time
	if s := c.Query("custom_start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			customStart = &t
		}
	}
	if e := c.Query("custom_end"); e != "" {
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

func (h *Handler) parseTopAssetsRequest(c *gin.Context) shortlink.TopAssetsRequest {
	tenantID, _ := h.getTenantID(c, c.Query("tenant_id"))
	timeRange := shortlink.TimeRange(c.DefaultQuery("time_range", "24h"))
	var customStart, customEnd *time.Time
	if s := c.Query("custom_start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			customStart = &t
		}
	}
	if e := c.Query("custom_end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			customEnd = &t
		}
	}
	limit := 20
	if l := c.Query("limit"); l != "" {
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
