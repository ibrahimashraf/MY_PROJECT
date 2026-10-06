package pipinghttp

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"integin/internal/shared/httpresponse"
	"integin/pkg/domain/piping"
)

// Handler serves ASME B31.3 / API 570 piping life assessment endpoints under /api/v1/piping/.
type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

type evaluateRequest struct {
	Parameters       piping.CircuitParameters `json:"parameters"`
	MeasurementPoint piping.MeasurementPoint  `json:"measurement_point"`
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

	var req evaluateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "malformed request JSON")
		return
	}

	result, err := piping.EvaluatePipingCircuit(req.Parameters, req.MeasurementPoint)
	if err != nil {
		httpresponse.JSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":  err.Error(),
			"result": result,
		})
		return
	}

	httpresponse.JSON(w, http.StatusOK, result)
}
