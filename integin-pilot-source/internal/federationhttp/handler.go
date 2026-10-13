package federationhttp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"integin/internal/federation"
	"integin/internal/shared/httpresponse"
)

// Handler serves cross-cell federation replication ingress under /api/v1/federation/replicate.
type Handler struct {
	Gatekeeper *federation.FederationGatekeeper
}

func NewHandler(gatekeeper *federation.FederationGatekeeper) *Handler {
	return &Handler{Gatekeeper: gatekeeper}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.Gatekeeper == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "federation gatekeeper not configured")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10MB limit
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "failed to read body")
		return
	}

	var msg federation.FederationMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "malformed federation message JSON")
		return
	}

	payload, err := h.Gatekeeper.IngestCrossCellEnvelope(&msg)
	if err != nil {
		if errors.Is(err, federation.ErrSovereignBoundaryBreach) {
			httpresponse.Error(w, http.StatusForbidden, err.Error())
			return
		}
		if errors.Is(err, federation.ErrReplayDetected) {
			httpresponse.Error(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, federation.ErrTamperedSignature) {
			httpresponse.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		httpresponse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	httpresponse.JSON(w, http.StatusOK, map[string]interface{}{
		"status":         "replicated",
		"message_id":     msg.MessageID,
		"entity_did":     msg.EntityDID,
		"payload_length": len(payload),
	})
}
