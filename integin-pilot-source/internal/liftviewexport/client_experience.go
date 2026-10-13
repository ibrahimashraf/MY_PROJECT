package liftviewexport

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"integin/pkg/httputil"
)

var defaultShareHMACKey = []byte("integin-liftview-client-share-secret-key-2026")

var (
	ErrShareTokenInvalid = errors.New("share token is invalid or corrupted")
	ErrShareTokenExpired = errors.New("share token has expired")
	ErrTenantMismatch    = errors.New("tenant mismatch: client token not authorized for this tenant")
)

// ClientBranding configures tenant and client white-label styling for lift dossiers.
// INVARIANT: Branding properties decorate visual presentation only and CANNOT
// alter solver physics, rated capacities, or statutory disclaimers.
type ClientBranding struct {
	OrganizationName string `json:"organization_name,omitempty"`
	ClientName       string `json:"client_name,omitempty"`
	LogoURL          string `json:"logo_url,omitempty"`
	PrimaryColorHex  string `json:"primary_color_hex,omitempty"`
	CustomHeader     string `json:"custom_header,omitempty"`
	WatermarkText    string `json:"watermark_text,omitempty"`
}

// RevisionRecord tracks auditable plan revisions linked by cryptographic Merkle hash chains.
type RevisionRecord struct {
	RevisionCode       string    `json:"revision_code"` // e.g. "REV-A", "REV-01"
	CreatedAt          time.Time `json:"created_at"`
	AuthorDID          string    `json:"author_did"`
	ChangeSummary      string    `json:"change_summary"`
	ParentHashSHA256   string    `json:"parent_hash_sha256"`             // Previous manifest hash
	ManifestHashSHA256 string    `json:"manifest_hash_sha256,omitempty"` // Current manifest hash
}

// ShareTokenClaims contains tenant-scoped authorization claims for client portal access.
type ShareTokenClaims struct {
	TokenID                     string    `json:"token_id"`
	TenantID                    string    `json:"tenant_id"`
	PlanID                      string    `json:"plan_id"`
	IssuedAt                    time.Time `json:"issued_at"`
	ExpiresAt                   time.Time `json:"expires_at"`
	AllowDownloadCAD            bool      `json:"allow_download_cad"`
	RedactSensitiveSiteGeometry bool      `json:"redact_sensitive_site_geometry"`
}

// CreateShareRequest payload to generate a client share token.
type CreateShareRequest struct {
	PlanID                      string `json:"plan_id"`
	ExpiresInHours              int    `json:"expires_in_hours"`
	AllowDownloadCAD            bool   `json:"allow_download_cad"`
	RedactSensitiveSiteGeometry bool   `json:"redact_sensitive_site_geometry"`
}

// RenewalStatus evaluates plan validity against statutory inspection/re-evaluation cycles (ASME B30.5 / LOLER 1998).
type RenewalStatus struct {
	PlanExpiresAt    time.Time `json:"plan_expires_at"`
	EvaluatedAt      time.Time `json:"evaluated_at"`
	IsExpired        bool      `json:"is_expired"`
	RenewalRequired  bool      `json:"renewal_required"`
	DaysRemaining    int       `json:"days_remaining"`
	AlertSeverity    string    `json:"alert_severity"` // "EXPIRED", "RENEWAL_DUE", "VALID"
	ActionableNotice string    `json:"actionable_notice"`
}

// ApplyBranding decorates metadata with client branding without modifying authoritative fields.
func ApplyBranding(meta map[string]any, branding *ClientBranding) {
	if branding == nil {
		return
	}
	// Copy of branding attributes only
	meta["branding"] = map[string]any{
		"organization_name": branding.OrganizationName,
		"client_name":       branding.ClientName,
		"logo_url":          branding.LogoURL,
		"primary_color_hex": branding.PrimaryColorHex,
		"custom_header":     branding.CustomHeader,
		"watermark_text":    branding.WatermarkText,
	}
}

// RedactPlanMetadata removes sensitive proprietary operational geometry for client-shared views.
func RedactPlanMetadata(meta map[string]any) map[string]any {
	redacted := make(map[string]any, len(meta)+2)
	for k, v := range meta {
		redacted[k] = v
	}
	redacted["redacted"] = true
	redacted["confidential_site_geometry_redacted"] = true
	// Sensitive internal operational properties are stripped
	delete(redacted, "chassis_tare_weight")
	delete(redacted, "internal_operator_notes")
	delete(redacted, "outrigger_pad_details")
	return redacted
}

// GenerateShareToken constructs an HMAC-SHA256 signed bearer token for client portal access.
func GenerateShareToken(claims ShareTokenClaims, key []byte) (string, error) {
	if len(key) == 0 {
		key = defaultShareHMACKey
	}
	raw, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(payloadB64))
	sigHex := hex.EncodeToString(mac.Sum(nil))
	return payloadB64 + "." + sigHex, nil
}

// ValidateShareToken verifies token integrity, signature, and expiration.
func ValidateShareToken(tokenStr string, key []byte, now time.Time) (*ShareTokenClaims, error) {
	if len(key) == 0 {
		key = defaultShareHMACKey
	}
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 2 {
		return nil, ErrShareTokenInvalid
	}
	payloadB64, sigHex := parts[0], parts[1]

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(payloadB64))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sigHex), []byte(expectedSig)) {
		return nil, ErrShareTokenInvalid
	}

	raw, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, ErrShareTokenInvalid
	}

	var claims ShareTokenClaims
	if err := json.Unmarshal(raw, &claims); err != nil {
		return nil, ErrShareTokenInvalid
	}

	if now.After(claims.ExpiresAt) {
		return &claims, ErrShareTokenExpired
	}

	return &claims, nil
}

// EvaluatePlanRenewal assesses lift plan statutory expiry and renewal lead time.
func EvaluatePlanRenewal(expiresAt, now time.Time, leadDays int) RenewalStatus {
	if leadDays <= 0 {
		leadDays = 14
	}
	diff := expiresAt.Sub(now)
	daysRemaining := int(diff.Hours() / 24.0)

	if diff <= 0 {
		return RenewalStatus{
			PlanExpiresAt:    expiresAt,
			EvaluatedAt:      now,
			IsExpired:        true,
			RenewalRequired:  true,
			DaysRemaining:    daysRemaining,
			AlertSeverity:    "EXPIRED",
			ActionableNotice: "Lift plan has expired under statutory review requirements (LOLER / ASME B30.5). Re-engineering and re-certification mandatory before lift execution.",
		}
	}

	if daysRemaining <= leadDays {
		return RenewalStatus{
			PlanExpiresAt:    expiresAt,
			EvaluatedAt:      now,
			IsExpired:        false,
			RenewalRequired:  true,
			DaysRemaining:    daysRemaining,
			AlertSeverity:    "RENEWAL_DUE",
			ActionableNotice: fmt.Sprintf("Lift plan expires in %d days. Initiate appointed person renewal assessment.", daysRemaining),
		}
	}

	return RenewalStatus{
		PlanExpiresAt:    expiresAt,
		EvaluatedAt:      now,
		IsExpired:        false,
		RenewalRequired:  false,
		DaysRemaining:    daysRemaining,
		AlertSeverity:    "VALID",
		ActionableNotice: "Lift plan is within active certification and engineering validity window.",
	}
}

// ServeShare handles creation and client portal consumption of time-bounded share links.
func (Handler) ServeShare(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
		if tenantID == "" {
			httputil.WriteProblem(w, r, http.StatusUnauthorized, "Unauthorized", "tenant identification required to issue client share")
			return
		}
		var req CreateShareRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid share request payload")
			return
		}
		if req.PlanID == "" {
			httputil.WriteProblem(w, r, http.StatusBadRequest, "Bad Request", "plan_id is required")
			return
		}
		hours := req.ExpiresInHours
		if hours <= 0 {
			hours = 72 // default 3 days
		}
		now := time.Now().UTC()
		claims := ShareTokenClaims{
			TokenID:                     fmt.Sprintf("sh-%d", now.UnixNano()),
			TenantID:                    tenantID,
			PlanID:                      req.PlanID,
			IssuedAt:                    now,
			ExpiresAt:                   now.Add(time.Duration(hours) * time.Hour),
			AllowDownloadCAD:            req.AllowDownloadCAD,
			RedactSensitiveSiteGeometry: req.RedactSensitiveSiteGeometry,
		}
		token, err := GenerateShareToken(claims, defaultShareHMACKey)
		if err != nil {
			httputil.WriteProblem(w, r, http.StatusInternalServerError, "Internal Server Error", "failed to generate share token")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"share_token":       token,
			"share_url":         fmt.Sprintf("/api/v1/liftviews/share?token=%s", token),
			"tenant_did":        fmt.Sprintf("did:integin:tenant:%s", tenantID),
			"plan_id":           req.PlanID,
			"expires_at":        claims.ExpiresAt.Format(time.RFC3339),
			"allow_cad":         claims.AllowDownloadCAD,
			"geometry_redacted": claims.RedactSensitiveSiteGeometry,
		})

	case http.MethodGet:
		tokenStr := r.URL.Query().Get("token")
		if tokenStr == "" {
			httputil.WriteProblem(w, r, http.StatusBadRequest, "Bad Request", "token query parameter is required")
			return
		}
		now := time.Now().UTC()
		claims, err := ValidateShareToken(tokenStr, defaultShareHMACKey, now)
		if err != nil {
			if errors.Is(err, ErrShareTokenExpired) {
				httputil.WriteProblem(w, r, http.StatusGone, "Gone", "client share token has expired")
				return
			}
			httputil.WriteProblem(w, r, http.StatusForbidden, "Forbidden", "invalid or tampered share token")
			return
		}

		// Enforce tenant scoping if caller provides tenant context (cross-tenant lockout)
		callerTenant := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
		if callerTenant != "" && callerTenant != claims.TenantID {
			httputil.WriteProblem(w, r, http.StatusForbidden, "Forbidden", "cross-tenant access forbidden: token belongs to different tenant")
			return
		}

		// Build client portal view
		clientPortalView := map[string]any{
			"status":                "AUTHORIZED",
			"tenant_did":            fmt.Sprintf("did:integin:tenant:%s", claims.TenantID),
			"plan_id":               claims.PlanID,
			"expires_at":            claims.ExpiresAt.Format(time.RFC3339),
			"allow_download_cad":    claims.AllowDownloadCAD,
			"confidential_redacted": claims.RedactSensitiveSiteGeometry,
			"statutory_scope":       "SIMULATION ONLY: NOT A CERTIFIED LIFT PLAN",
			"portal_disclaimer":     "Client portal view. Authoritative calculations verified server-side under ASME B30.5 / BS 7121.",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(clientPortalView)

	default:
		w.Header().Set("Allow", "GET, POST")
		httputil.WriteProblem(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "Allowed methods: GET, POST")
	}
}

// ServeRenewal handles statutory validity checks and renewal notifications.
func (Handler) ServeRenewal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		httputil.WriteProblem(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "Allowed methods: GET")
		return
	}
	expiresAtStr := r.URL.Query().Get("expires_at")
	if expiresAtStr == "" {
		httputil.WriteProblem(w, r, http.StatusBadRequest, "Bad Request", "expires_at parameter required (RFC3339)")
		return
	}
	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		httputil.WriteProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid expires_at date format")
		return
	}
	leadDays := 14
	if leadStr := r.URL.Query().Get("lead_days"); leadStr != "" {
		if val, err := strconv.Atoi(leadStr); err == nil && val > 0 {
			leadDays = val
		}
	}
	now := time.Now().UTC()
	status := EvaluatePlanRenewal(expiresAt, now, leadDays)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(status)
}
