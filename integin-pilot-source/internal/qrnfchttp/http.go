package qrnfchttp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"integin/internal/domain/qrnfc"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/pkg/httputil"
)

// TokenValidator defines the contract for OIDC bearer token verification.
type TokenValidator interface {
	Validate(ctx context.Context, rawToken string) (oidcauth.Principal, error)
}

// EntryVerifier verifies QR/NFC entry tokens.
type EntryVerifier interface {
	VerifyByDigest(ctx context.Context, actor qrnfc.ActorContext, tokenDigest string) (qrnfc.Entry, error)
}

// AccessLogger logs access attempts.
type AccessLogger interface {
	LogAccess(ctx context.Context, actor qrnfc.ActorContext, log qrnfc.EntryLog) error
}

// Handler is the HTTP handler for authenticated QR/NFC entry.
type Handler struct {
	validator TokenValidator
	resolver  identity.Resolver
	verifier  EntryVerifier
	logger    AccessLogger
	log       *slog.Logger
}

// NewHandler creates a new authenticated entry handler wired to the shared
// OIDC validator and local identity resolver (same auth path as admin APIs).
func NewHandler(validator TokenValidator, resolver identity.Resolver, verifier EntryVerifier, logger AccessLogger, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{
		validator: validator,
		resolver:  resolver,
		verifier:  verifier,
		logger:    logger,
		log:       log,
	}
}

// ServeHTTP handles POST /qr-nfc/verify with Bearer token authentication.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteProblem(w, r, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed), "method_not_allowed")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")

	ip := extractClientIP(r)

	// Extract and validate Bearer token
	authHeader := r.Header.Get("Authorization")
	bearer, ok := extractBearer(authHeader)
	if !ok {
		h.logAccess(r, nil, "", ip, qrnfc.AccessOutcomeDenied, "missing or invalid authorization header")
		httputil.WriteProblem(w, r, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), "authentication_required")
		return
	}

	// Validate OIDC token (issuer/subject authenticity only)
	principal, err := h.validator.Validate(r.Context(), bearer)
	if err != nil {
		h.logAccess(r, nil, "", ip, qrnfc.AccessOutcomeDenied, "oidc validation failed")
		httputil.WriteProblem(w, r, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), "invalid_token")
		return
	}

	// Resolve local authorization context
	membership, err := h.resolver.Resolve(r.Context(), identity.PrincipalKey{
		Issuer:  principal.Issuer,
		Subject: principal.Subject,
	})
	if err != nil {
		h.logAccess(r, nil, "", ip, qrnfc.AccessOutcomeDenied, "actor resolution failed")
		httputil.WriteProblem(w, r, http.StatusForbidden, http.StatusText(http.StatusForbidden), "forbidden")
		return
	}
	actor := &qrnfc.ActorContext{
		TenantID:       membership.TenantID,
		OrganizationID: membership.OrganizationID,
		ActorID:        membership.ActorID,
	}

	// Parse request body
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		h.logAccess(r, actor, "", ip, qrnfc.AccessOutcomeDenied, "invalid request body")
		httputil.WriteProblem(w, r, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "invalid_request")
		return
	}
	if strings.TrimSpace(req.Token) == "" {
		h.logAccess(r, actor, "", ip, qrnfc.AccessOutcomeDenied, "token is required")
		httputil.WriteProblem(w, r, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "token_required")
		return
	}

	// Verify token digest
	entry, err := h.verifier.VerifyByDigest(r.Context(), *actor, computeDigest(req.Token))
	if err != nil {
		switch {
		case errors.Is(err, qrnfc.ErrNotFound):
			// Indistinguishable from revoked/expired for security
			h.logAccess(r, actor, "", ip, qrnfc.AccessOutcomeDenied, "token not found")
			httputil.WriteProblem(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound), "invalid_token")
			return
		case errors.Is(err, qrnfc.ErrExpired):
			h.logAccess(r, actor, entry.ID, ip, qrnfc.AccessOutcomeExpired, "token expired")
			httputil.WriteProblem(w, r, http.StatusGone, http.StatusText(http.StatusGone), "token_expired")
			return
		default:
			h.logAccess(r, actor, "", ip, qrnfc.AccessOutcomeDenied, "verification failed")
			httputil.WriteProblem(w, r, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), "verification_failed")
			return
		}
	}

	// Log successful access
	h.logAccess(r, actor, entry.ID, ip, qrnfc.AccessOutcomeSuccess, "")

	// Compose response (privacy-preserving: no token digest, no raw token)
	resp := map[string]interface{}{
		"entry_id":   entry.ID,
		"asset_id":   entry.AssetID,
		"entry_type": entry.EntryType,
		"status":     entry.Status,
		"issued_at":  entry.IssuedAt,
	}
	if entry.ExpiresAt != nil {
		resp["expires_at"] = entry.ExpiresAt
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.log.Error("failed to encode qr_nfc response", "error", err)
	}
}

func (h *Handler) logAccess(r *http.Request, actor *qrnfc.ActorContext, entryID, ip string, outcome qrnfc.AccessOutcome, detail string) {
	h.log.Info("qr_nfc_access",
		"entry_id", entryID,
		"outcome", outcome,
		"client_ip", ip,
		"detail", detail,
	)
	if h.logger == nil || actor == nil {
		return
	}
	record := qrnfc.EntryLog{
		ID:         fmt.Sprintf("log_%d", time.Now().UnixNano()),
		EntryID:    entryID,
		EntryType:  qrnfc.EntryTypeQR,
		AccessedAt: time.Now().UTC(),
		AccessorID: actor.ActorID,
		AccessorIP: ip,
		Outcome:    outcome,
	}
	if err := h.logger.LogAccess(r.Context(), *actor, record); err != nil {
		h.log.Error("failed to persist qr_nfc access log", "error", err)
	}
}

func extractBearer(authHeader string) (string, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
	if token == "" {
		return "", false
	}
	return token, true
}

func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func computeDigest(token string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return fmt.Sprintf("%x", h)
}
