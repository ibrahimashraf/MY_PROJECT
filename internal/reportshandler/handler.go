package reportshandler

import (
	"encoding/json"
	"net/http"
	"strings"

	domain "integin/internal/domain/reports"
	csvgen "integin/internal/reports"
	"integin/internal/shared/httpresponse"
)

type Handler struct {
	Repository domain.Repository
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case r.Method == http.MethodGet && path == "/api/v1/reports/configs":
		h.handleListConfigs(w, r)
	case r.Method == http.MethodPost && path == "/api/v1/reports/configs":
		h.handleCreateConfig(w, r)
	case r.Method == http.MethodPut && strings.HasPrefix(path, "/api/v1/reports/configs/"):
		h.handleUpdateConfig(w, r)
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/api/v1/reports/configs/"):
		h.handleDeleteConfig(w, r)
	case r.Method == http.MethodPost && path == "/api/v1/reports/generate":
		h.handleGenerate(w, r)
	case r.Method == http.MethodGet && path == "/api/v1/reports":
		h.handleListReports(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/api/v1/reports/") && strings.HasSuffix(path, "/download"):
		h.handleDownload(w, r)
	default:
		httpresponse.Error(w, http.StatusNotFound, "not_found")
	}
}

func (h Handler) extractIDFromPath(path, prefix string) string {
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/download")
	return strings.TrimRight(id, "/")
}

func extractTenantOrg(r *http.Request) (string, string) {
	return r.Header.Get("X-Tenant-ID"), r.Header.Get("X-Organization-ID")
}

func (h Handler) handleListConfigs(w http.ResponseWriter, r *http.Request) {
	tenantID, orgID := extractTenantOrg(r)
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing tenant or organization header")
		return
	}

	configs, err := h.Repository.ListConfigs(r.Context(), tenantID, orgID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "failed to list report configs")
		return
	}
	if configs == nil {
		configs = []domain.ReportConfig{}
	}
	httpresponse.JSON(w, http.StatusOK, configs)
}

func (h Handler) handleCreateConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, orgID := extractTenantOrg(r)
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing tenant or organization header")
		return
	}

	var config domain.ReportConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}

	config.TenantID = tenantID
	config.OrgID = orgID

	if err := config.Validate(); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := h.Repository.CreateConfig(r.Context(), config)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "failed to create report config")
		return
	}
	httpresponse.JSON(w, http.StatusCreated, created)
}

func (h Handler) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, orgID := extractTenantOrg(r)
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing tenant or organization header")
		return
	}

	id := h.extractIDFromPath(r.URL.Path, "/api/v1/reports/configs/")

	var config domain.ReportConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}

	config.ID = id
	config.TenantID = tenantID
	config.OrgID = orgID

	updated, err := h.Repository.UpdateConfig(r.Context(), config)
	if err != nil {
		if err == domain.ErrConfigNotFound {
			httpresponse.Error(w, http.StatusNotFound, "report config not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "failed to update report config")
		return
	}
	httpresponse.JSON(w, http.StatusOK, updated)
}

func (h Handler) handleDeleteConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, orgID := extractTenantOrg(r)
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing tenant or organization header")
		return
	}

	id := h.extractIDFromPath(r.URL.Path, "/api/v1/reports/configs/")

	if err := h.Repository.DeleteConfig(r.Context(), tenantID, orgID, id); err != nil {
		if err == domain.ErrConfigNotFound {
			httpresponse.Error(w, http.StatusNotFound, "report config not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "failed to delete report config")
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h Handler) handleGenerate(w http.ResponseWriter, r *http.Request) {
	tenantID, orgID := extractTenantOrg(r)
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing tenant or organization header")
		return
	}

	var req domain.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}

	if req.ConfigID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "config_id is required")
		return
	}

	report, err := h.Repository.GenerateReport(r.Context(), req, tenantID, orgID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "failed to generate report")
		return
	}
	httpresponse.JSON(w, http.StatusCreated, report)
}

func (h Handler) handleListReports(w http.ResponseWriter, r *http.Request) {
	tenantID, orgID := extractTenantOrg(r)
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing tenant or organization header")
		return
	}

	result, err := h.Repository.ListReports(r.Context(), tenantID, orgID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "failed to list reports")
		return
	}
	if result == nil {
		result = []domain.GeneratedReport{}
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h Handler) handleDownload(w http.ResponseWriter, r *http.Request) {
	tenantID, orgID := extractTenantOrg(r)
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing tenant or organization header")
		return
	}

	id := h.extractIDFromPath(r.URL.Path, "/api/v1/reports/")

	report, err := h.Repository.GetReport(r.Context(), tenantID, orgID, id)
	if err != nil {
		if err == domain.ErrReportNotFound {
			httpresponse.Error(w, http.StatusNotFound, "report not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "failed to get report")
		return
	}

	if report.Status != domain.ReportStatusReady {
		httpresponse.Error(w, http.StatusConflict, "report not ready")
		return
	}

	genReq := domain.GenerateRequest{
		ConfigID: report.ConfigID,
	}

	headers, rows, err := h.Repository.GenerateCSVData(r.Context(), genReq, tenantID, orgID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "failed to generate CSV data")
		return
	}

	csvData, err := csvgen.GenerateCSV(headers, rows)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "failed to generate CSV")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"report.csv\"")
	w.WriteHeader(http.StatusOK)
	w.Write(csvData) //nolint:G705 // CSV data written as text/csv; not HTML context
}
