package atexhttp

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"integin/internal/shared/httpresponse"
	"integin/pkg/atex"
)

// Handler serves ATEX hazardous area evaluation endpoints under /api/v1/atex/.
type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	orgID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing required headers: X-Tenant-ID, X-Organization-ID")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "failed to read body")
		return
	}

	var loop atex.IntrinsicSafetyLoop
	if err := json.Unmarshal(body, &loop); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "malformed request JSON")
		return
	}

	result, err := atex.ValidateIntrinsicSafetyLoop(loop)
	if err != nil {
		httpresponse.JSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":  err.Error(),
			"result": result,
		})
		return
	}

	httpresponse.JSON(w, http.StatusOK, result)
}
