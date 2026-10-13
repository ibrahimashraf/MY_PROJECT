package riskhttp

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"integin/internal/shared/httpresponse"
	"integin/pkg/riskengine"
)

// Handler serves parametric risk evaluation endpoints under /api/v1/risk/.
type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

type evaluateRequest struct {
	Inputs  riskengine.OperationalRiskInputs `json:"inputs"`
	Weights *riskengine.Weights              `json:"weights,omitempty"`
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

	if req.Inputs.TenantID == "" {
		req.Inputs.TenantID = tenantID
	}
	if req.Inputs.OrganizationID == "" {
		req.Inputs.OrganizationID = orgID
	}

	weights := riskengine.DefaultWeights()
	if req.Weights != nil {
		weights = *req.Weights
	}

	result, err := riskengine.ComputeOperationalRisk(req.Inputs, weights)
	if err != nil {
		// Even if error is returned, result may contain quarantine status
		httpresponse.JSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":  err.Error(),
			"result": result,
		})
		return
	}

	httpresponse.JSON(w, http.StatusOK, result)
}
