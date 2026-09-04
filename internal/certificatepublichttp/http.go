package certificatepublichttp

import (
	"context"
	"encoding/json"
	"integin/internal/certificatepg"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Verifier interface {
	VerifyPublic(context.Context, string) (certificatepg.PublicProjection, bool, error)
}
type rateWindow struct {
	started time.Time
	count   int
}
type Handler struct {
	Verifier         Verifier
	Authenticated    *AuthenticatedTransport
	Limit            int
	Window           time.Duration
	Now              func() time.Time
	VerifierBaseURL  string
	TrustedProxies   []string
	mu               sync.Mutex
	rates            map[string]rateWindow
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		reply(w, http.StatusMethodNotAllowed, nil)
		return
	}
	if h.Verifier == nil {
		reply(w, http.StatusServiceUnavailable, nil)
		return
	}
	token, _, ok := extractTokenAndPath(r.URL.Path)
	if !ok {
		reply(w, http.StatusNotFound, nil)
		return
	}
	// Validate token length before any verifier call (matches original behavior)
	if len(token) < 32 || len(token) > 128 {
		reply(w, http.StatusNotFound, nil)
		return
	}
	now := time.Now()
	if h.Now != nil {
		now = h.Now()
	}
	if !h.allow(h.remoteKey(r), now.UTC()) {
		reply(w, http.StatusTooManyRequests, nil)
		return
	}

	// Suffixes removed, fall back to authenticated transport if configured
	if h.Authenticated != nil {
		h.handleAuthenticated(w, r, token)
		return
	}

	// Fall back to regular verifier (original behavior)
	view, found, err := h.Verifier.VerifyPublic(r.Context(), token)
	if err != nil {
		reply(w, http.StatusServiceUnavailable, nil)
		return
	}
	if !found {
		reply(w, http.StatusNotFound, nil)
		return
	}
	response := map[string]any{"certificate_number": view.CertificateNumber, "status": view.Status, "issued_at": view.IssuedAt, "expires_at": view.ExpiresAt, "asset_id": view.AssetID}
	if view.AssetSerialNumber != "" {
		response["asset_serial_number"] = view.AssetSerialNumber
	}
	if view.AssetDescription != "" {
		response["asset_description"] = view.AssetDescription
	}
	if view.AssetType != "" {
		response["asset_type"] = view.AssetType
	}
	if view.InspectionType != "" {
		response["inspection_type"] = view.InspectionType
	}
	if len(view.TestScope) > 0 {
		response["test_scope"] = json.RawMessage(view.TestScope)
	}
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		replyHTML(w, http.StatusOK, view)
		return
	}
	reply(w, http.StatusOK, response)
}



func (h *Handler) handleAuthenticated(w http.ResponseWriter, r *http.Request, token string) {
	view, found, err := h.Authenticated.Verifier.VerifyPublic(r.Context(), token)
	if err != nil {
		reply(w, http.StatusServiceUnavailable, nil)
		return
	}
	if !found {
		reply(w, http.StatusNotFound, nil)
		return
	}
	response := h.Authenticated.composeAuthenticatedResponse(view)
	reply(w, http.StatusOK, response)
}

func extractTokenAndPath(path string) (string, string, bool) {
	p := strings.Trim(path, "/")
	if !strings.HasPrefix(p, "verify/certificates/") {
		return "", "", false
	}
	rest := strings.TrimPrefix(p, "verify/certificates/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", "", false
	}
	token := parts[0]
	suffix := ""
	if len(parts) > 1 {
		suffix = "/" + strings.Join(parts[1:], "/")
	}
	return token, suffix, true
}
