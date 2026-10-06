package robotichttp

import (
	"io"
	"net/http"
	"strings"

	"integin/internal/robotics"
	"integin/internal/shared/httpresponse"
)

// Handler handles autonomous robotic inspection ingress requests.
type Handler struct {
	Service *robotics.IngressService
}

func NewHandler(service *robotics.IngressService) *Handler {
	if service == nil {
		service = robotics.NewIngressService()
	}
	return &Handler{Service: service}
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

	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20)) // 4MB ceiling
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	receipt, err := robotics.ConvertReceiptJSON(body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	coreReceipt, err := h.Service.IngestRoboticInspection(r.Context(), tenantID, orgID, receipt)
	if err != nil {
		httpresponse.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	httpresponse.JSON(w, http.StatusOK, coreReceipt)
}
