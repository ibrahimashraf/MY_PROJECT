package certificatepublichttp

import (
	"context"
	"encoding/json"
	"integin/internal/certificatepg"
	"net/http"
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
	Verifier Verifier
	Limit    int
	Window   time.Duration
	Now      func() time.Time
	mu       sync.Mutex
	rates    map[string]rateWindow
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
	token, ok := extractToken(r.URL.Path)
	if !ok {
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
