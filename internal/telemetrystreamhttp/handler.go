package telemetrystreamhttp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"integin/internal/shared/httpresponse"
	"integin/internal/telemetrystream"
)

// Handler handles high-frequency sensor telemetry stream ingest.
type Handler struct {
	Buffer *telemetrystream.StreamBuffer
}

func NewHandler(buffer *telemetrystream.StreamBuffer) *Handler {
	return &Handler{Buffer: buffer}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.Buffer == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "telemetry stream buffer not configured")
		return
	}

	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	orgID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing required headers: X-Tenant-ID, X-Organization-ID")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20)) // 2MB limit
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "failed to read body")
		return
	}

	var reading telemetrystream.TelemetryReading
	if err := json.Unmarshal(body, &reading); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "malformed telemetry reading JSON")
		return
	}

	// Tenant boundary enforcement
	if reading.TenantID == "" {
		reading.TenantID = tenantID
	} else if reading.TenantID != tenantID {
		httpresponse.Error(w, http.StatusForbidden, "tenant mismatch")
		return
	}
	if reading.OrganizationID == "" {
		reading.OrganizationID = orgID
	} else if reading.OrganizationID != orgID {
		httpresponse.Error(w, http.StatusForbidden, "organization mismatch")
		return
	}

	if err := h.Buffer.Ingest(reading); err != nil {
		if errors.Is(err, telemetrystream.ErrCriticalOverload) {
			httpresponse.Error(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		if errors.Is(err, telemetrystream.ErrBufferOverflow) {
			httpresponse.Error(w, http.StatusServiceUnavailable, "telemetry buffer full")
			return
		}
		httpresponse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	httpresponse.JSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}
