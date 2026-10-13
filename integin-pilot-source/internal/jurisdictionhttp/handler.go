package jurisdictionhttp

import (
	"net/http"
	"strings"

	"integin/internal/shared/httpresponse"
	"integin/pkg/jurisdictions"
	"integin/pkg/jurisdictions/adapters"
)

// Handler serves read-only sovereign jurisdiction profile and compliance metadata.
type Handler struct {
	Registry *jurisdictions.Registry
}

// NewHandler constructs an HTTP handler around a validated jurisdictions registry.
func NewHandler(r *jurisdictions.Registry) *Handler {
	return &Handler{Registry: r}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.Registry == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "jurisdiction registry not configured")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/jurisdictions")
	trimmed := strings.Trim(path, "/")

	if trimmed == "" {
		profiles := h.Registry.All()
		httpresponse.JSON(w, http.StatusOK, profiles)
		return
	}

	parts := strings.Split(trimmed, "/")
	code := strings.ToUpper(strings.TrimSpace(parts[0]))

	profile, ok := h.Registry.ByISO2(code)
	if !ok {
		profile, ok = h.Registry.ByISO3(code)
	}
	if !ok {
		httpresponse.Error(w, http.StatusNotFound, "jurisdiction not found")
		return
	}

	switch len(parts) {
	case 1:
		httpresponse.JSON(w, http.StatusOK, profile)
	case 2:
		if parts[1] == "compliance" {
			results, err := adapters.CheckAll(profile)
			if err != nil {
				httpresponse.Error(w, http.StatusInternalServerError, err.Error())
				return
			}
			httpresponse.JSON(w, http.StatusOK, results)
			return
		}
		httpresponse.Error(w, http.StatusNotFound, "not found")
	default:
		httpresponse.Error(w, http.StatusNotFound, "not found")
	}
}
