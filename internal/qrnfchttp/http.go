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
	"sync"
	"time"

	"integin/internal/domain/qrnfc"
)

// TokenValidator validates OIDC tokens and resolves local membership.
type TokenValidator interface {
	Validate(ctx context.Context, token string) (string, error)
}

// ActorResolver resolves a validated OIDC subject to a local actor context.
type ActorResolver interface {
	Resolve(ctx context.Context, subject string) (qrnfc.ActorContext, error)
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
	Validator    TokenValidator
	Actors       ActorResolver
	Verifier     EntryVerifier
	Logger       AccessLogger
	Log          *slog.Logger
	rateWindow   map[string]rateWindow
	rateMu       sync.Mutex
	limit        int
	window       time.Duration
	now          func() time.Time
}

type rateWindow struct {
	started time.Time
	count   int
}

// NewHandler creates a new authenticated entry handler.
func NewHandler(validator TokenValidator, actors ActorResolver, verifier EntryVerifier, logger AccessLogger, log *slog.Logger) *Handler {
	return &Handler{
		Validator:  validator,
		Actors:     actors,
		Verifier:   verifier,
		Logger:     logger,
		Log:        log,
		rateWindow: make(map[string]rateWindow),
		limit:      60,
		window:     time.Minute,
		now:        time.Now,
	}
}

// ServeHTTP handles POST /qr-nfc/verify with Bearer token authentication.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")

	// Extract and validate Bearer token
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, `{"error":"missing or invalid authorization header"}`, http.StatusUnauthorized)
		return
	}
	bearerToken := strings.TrimPrefix(authHeader, "Bearer ")
	if strings.TrimSpace(bearerToken) == "" {
		http.Error(w, `{"error":"empty bearer token"}`, http.StatusUnauthorized)
		return
	}

	// Rate limit check
	clientIP := extractClientIP(r)
	if !h.allow(clientIP, h.now()) {
		h.logAccess(r, "", "DENIED", "rate limit exceeded")
		http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
		return
	}

	// Validate OIDC token
	subject, err := h.Validator.Validate(r.Context(), bearerToken)
	if err != nil {
		h.logAccess(r, "", "DENIED", "oidc validation failed")
		http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
		return
	}

	// Resolve local actor
	actor, err := h.Actors.Resolve(r.Context(), subject)
	if err != nil {
		h.logAccess(r, "", "DENIED", "actor resolution failed")
		http.Error(w, `{"error":"access denied"}`, http.StatusForbidden)
		return
	}

	// Parse request body
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Token) == "" {
		http.Error(w, `{"error":"token is required"}`, http.StatusBadRequest)
		return
	}

	// Verify token digest
	tokenDigest := computeDigest(req.Token)
	entry, err := h.Verifier.VerifyByDigest(r.Context(), actor, tokenDigest)
	if err != nil {
		if errors.Is(err, qrnfc.ErrNotFound) {
			h.logAccess(r, "", "DENIED", "token not found")
			// Indistinguishable from revoked/expired for security
			http.Error(w, `{"error":"invalid token"}`, http.StatusNotFound)
			return
		}
		if errors.Is(err, qrnfc.ErrExpired) {
			h.logAccess(r, entry.ID, "EXPIRED", "")
			http.Error(w, `{"error":"token expired"}`, http.StatusGone)
			return
		}
		h.logAccess(r, "", "DENIED", "verification failed")
		http.Error(w, `{"error":"verification failed"}`, http.StatusInternalServerError)
		return
	}

	// Log successful access
	h.logAccess(r, entry.ID, "SUCCESS", "")

	// Compose response (privacy-preserving: no token digest, no raw token)
	resp := map[string]interface{}{
		"entry_id":    entry.ID,
		"asset_id":    entry.AssetID,
		"entry_type":  entry.EntryType,
		"status":      entry.Status,
		"issued_at":   entry.IssuedAt,
	}
	if entry.ExpiresAt != nil {
		resp["expires_at"] = entry.ExpiresAt
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) logAccess(r *http.Request, entryID, outcome, detail string) {
	if h.Logger == nil {
		return
	}
	clientIP := extractClientIP(r)
	logID := fmt.Sprintf("log_%d", time.Now().UnixNano())
	entryType := qrnfc.EntryTypeQR
	if entryID != "" {
		// Try to determine entry type from context
		entryType = qrnfc.EntryTypeQR
	}
	h.Logger.LogAccess(r.Context(), qrnfc.ActorContext{
		TenantID:       "",
		OrganizationID: "",
		ActorID:        "",
	}, qrnfc.EntryLog{
		ID:           logID,
		EntryID:      entryID,
		EntryType:    entryType,
		AccessedAt:   time.Now().UTC(),
		AccessorID:   "",
		AccessorIP:   clientIP,
		Outcome:      qrnfc.AccessOutcome(outcome),
	})
	if h.Log != nil {
		h.Log.Info("qr_nfc_access",
			"entry_id", entryID,
			"outcome", outcome,
			"client_ip", clientIP,
			"detail", detail,
		)
	}
}

func (h *Handler) allow(key string, now time.Time) bool {
	h.rateMu.Lock()
	defer h.rateMu.Unlock()
	rw, ok := h.rateWindow[key]
	if !ok || rw.started.IsZero() || now.Sub(rw.started) >= h.window {
		h.rateWindow[key] = rateWindow{started: now, count: 1}
		return true
	}
	if rw.count >= h.limit {
		return false
	}
	rw.count++
	h.rateWindow[key] = rw
	return true
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
