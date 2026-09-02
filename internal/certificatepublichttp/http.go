package certificatepublichttp

import (
	"context"
	"encoding/json"
	"integin/internal/certificatepg"
	"integin/internal/certificaterender"
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
	Verifier        Verifier
	Limit           int
	Window          time.Duration
	Now             func() time.Time
	VerifierBaseURL string
	Renderer        *certificaterender.Request
	mu              sync.Mutex
	rates           map[string]rateWindow
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
	token, pathSuffix, ok := extractTokenAndPath(r.URL.Path)
	if !ok {
		reply(w, http.StatusNotFound, nil)
		return
	}
	// Validate token length before any verifier call (matches original behavior)
	if len(token) < 32 || len(token) > 128 {
		reply(w, http.StatusNotFound, nil)
		return
	}
	now := time.Now
	if h.Now != nil {
		now = h.Now
	}
	if !h.allow(remoteKey(r), now().UTC()) {
		reply(w, http.StatusTooManyRequests, nil)
		return
	}

	// Handle /qr suffix
	if pathSuffix == "/qr" {
		h.handleQR(w, r, token)
		return
	}
	// Handle /pdf suffix
	if pathSuffix == "/pdf" {
		h.handlePDF(w, r, token)
		return
	}

	// Default: JSON projection
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
	reply(w, http.StatusOK, response)
}

func (h *Handler) handleQR(w http.ResponseWriter, r *http.Request, token string) {
	_, found, err := h.Verifier.VerifyPublic(r.Context(), token)
	if err != nil {
		reply(w, http.StatusServiceUnavailable, nil)
		return
	}
	if !found {
		reply(w, http.StatusNotFound, nil)
		return
	}
	if h.VerifierBaseURL == "" {
		reply(w, http.StatusServiceUnavailable, nil)
		return
	}
	qrURL, err := certificaterender.VerifierURL(h.VerifierBaseURL, token)
	if err != nil {
		reply(w, http.StatusServiceUnavailable, nil)
		return
	}
	qrPNG, err := certificaterender.GenerateQR(qrURL)
	if err != nil {
		reply(w, http.StatusServiceUnavailable, nil)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(qrPNG)
}

func (h *Handler) handlePDF(w http.ResponseWriter, r *http.Request, token string) {
	if h.Renderer == nil {
		reply(w, http.StatusServiceUnavailable, nil)
		return
	}
	renderData, found, err := h.getRenderData(token)
	if err != nil {
		reply(w, http.StatusServiceUnavailable, nil)
		return
	}
	if !found {
		reply(w, http.StatusNotFound, nil)
		return
	}
	result, err := certificaterender.Render(*renderData)
	if err != nil {
		reply(w, http.StatusServiceUnavailable, nil)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+result.Metadata["content_disposition_name"]+"\"")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(result.PDF)
}

func (h *Handler) getRenderData(token string) (*certificaterender.Request, bool, error) {
	// This would need a Repository with GetRenderData method
	// For now return not found - requires extending the Verifier interface
	return nil, false, nil
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
