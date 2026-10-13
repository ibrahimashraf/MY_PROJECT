package searchhttp

import (
	"log/slog"
	"net/http"
	"strings"

	"integin/internal/domain/search"
	"integin/internal/shared/httpresponse"
)

type Handler struct {
	Repository search.Repository
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantID := r.Header.Get("X-Tenant-ID")
	orgID := r.Header.Get("X-Organization-ID")
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing tenant or organization header")
		return
	}

	q := r.URL.Query().Get("q")
	if q == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing query parameter q")
		return
	}

	typesParam := r.URL.Query().Get("types")
	var types []search.EntityType
	if typesParam != "" {
		for _, t := range strings.Split(typesParam, ",") {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			et := search.EntityType(t)
			if !isValidEntityType(et) {
				httpresponse.Error(w, http.StatusBadRequest, "invalid entity type: "+t)
				return
			}
			types = append(types, et)
		}
	}

	req := search.SearchRequest{
		Query:          q,
		TenantID:       tenantID,
		OrganizationID: orgID,
		Types:          types,
	}

	resp, err := h.Repository.Search(r.Context(), req)
	if err != nil {
		slog.Error("search failed", "error", err, "query", q, "tenant", tenantID, "org", orgID)
		httpresponse.Error(w, http.StatusInternalServerError, "search failed")
		return
	}

	httpresponse.JSON(w, http.StatusOK, resp)
}

func isValidEntityType(et search.EntityType) bool {
	switch et {
	case search.EntityAsset, search.EntityWorkOrder, search.EntityInspection:
		return true
	default:
		return false
	}
}
