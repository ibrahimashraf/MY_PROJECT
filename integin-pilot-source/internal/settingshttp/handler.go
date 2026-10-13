package settingshttp

import (
	"encoding/json"
	"net/http"
	"strings"

	"integin/internal/settingspg"
	"integin/internal/shared/httpresponse"
	"integin/pkg/httputil"
)

type Handler struct {
	Repository *settingspg.Repository
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tenantID := r.Header.Get("X-Tenant-ID")
	orgID := r.Header.Get("X-Organization-ID")
	if tenantID == "" || orgID == "" {
		httputil.WriteProblem(w, r, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "missing tenant or organization header")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/settings")

	switch {
	case r.Method == http.MethodGet && (path == "" || path == "/"):
		h.handleList(w, r, tenantID, orgID)
	case r.Method == http.MethodGet && path != "":
		key := strings.TrimPrefix(path, "/")
		scope := r.URL.Query().Get("scope")
		if scope == "" {
			scope = "ORGANIZATION"
		}
		h.handleGet(w, r, tenantID, orgID, key, scope)
	case r.Method == http.MethodPut && (path == "" || path == "/"):
		h.handleUpsert(w, r, tenantID, orgID)
	default:
		httputil.WriteProblem(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound), "not_found")
	}
}

func (h Handler) handleList(w http.ResponseWriter, r *http.Request, tenantID, orgID string) {
	settings, err := h.Repository.ListSettings(r.Context(), tenantID, orgID)
	if err != nil {
		httputil.WriteProblem(w, r, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), err.Error())
		return
	}
	if settings == nil {
		settings = []settingspg.Setting{}
	}
	httpresponse.JSON(w, http.StatusOK, settings)
}

func (h Handler) handleGet(w http.ResponseWriter, r *http.Request, tenantID, orgID, key, scope string) {
	setting, ok, err := h.Repository.GetSetting(r.Context(), tenantID, orgID, key, scope)
	if err != nil {
		httputil.WriteProblem(w, r, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), err.Error())
		return
	}
	if !ok {
		httputil.WriteProblem(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound), "setting not found")
		return
	}
	httpresponse.JSON(w, http.StatusOK, setting)
}

func (h Handler) handleUpsert(w http.ResponseWriter, r *http.Request, tenantID, orgID string) {
	var s settingspg.Setting
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		httputil.WriteProblem(w, r, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "invalid_request")
		return
	}
	s.TenantID = tenantID
	s.OrganizationID = orgID
	result, err := h.Repository.UpsertSetting(r.Context(), tenantID, orgID, s)
	if err != nil {
		httputil.WriteProblem(w, r, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}
