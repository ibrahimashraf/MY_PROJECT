package analyticshttp

import (
	"net/http"
	"time"

	"integin/internal/domain/analytics"
	"integin/internal/shared/httpresponse"
)

type Handler struct {
	Repository analytics.Repository
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantID := r.Header.Get("X-Tenant-ID")
	orgID := r.Header.Get("X-Organization-ID")
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing required headers: X-Tenant-ID, X-Organization-ID")
		return
	}

	req := analytics.DashboardRequest{
		TenantID:       tenantID,
		OrganizationID: orgID,
	}

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "invalid 'from' parameter, use RFC3339 format")
			return
		}
		req.From = t
	}

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "invalid 'to' parameter, use RFC3339 format")
			return
		}
		req.To = t
	}

	req.InspectorID = r.URL.Query().Get("inspector_id")
	req.AssetType = r.URL.Query().Get("asset_type")

	dashboard, err := h.Repository.GetDashboard(r.Context(), req)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "failed to load dashboard")
		return
	}

	httpresponse.JSON(w, http.StatusOK, dashboard)
}
