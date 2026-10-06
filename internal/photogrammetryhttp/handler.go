package photogrammetryhttp

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"integin/internal/shared/httpresponse"
	"integin/pkg/photogrammetry"
)

// Handler serves photogrammetry survey endpoints under /api/v1/photogrammetry/.
type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

type surveyRequest struct {
	AssetDID                 string                   `json:"asset_did"`
	Points                   []photogrammetry.Point3D `json:"points"`
	BaselinePlaneZ           float64                  `json:"baseline_plane_z"`
	MaxPermissibleDeflection float64                  `json:"max_permissible_deflection"`
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

	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10MB limit for point clouds
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "failed to read body")
		return
	}

	var req surveyRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "malformed request JSON")
		return
	}

	result, err := photogrammetry.ComputeVolumetricSurvey(
		req.AssetDID,
		req.Points,
		req.BaselinePlaneZ,
		req.MaxPermissibleDeflection,
	)
	if err != nil {
		httpresponse.JSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":  err.Error(),
			"result": result,
		})
		return
	}

	httpresponse.JSON(w, http.StatusOK, result)
}
