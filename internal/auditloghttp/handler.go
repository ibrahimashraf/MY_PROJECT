package auditloghttp

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"integin/internal/domain/auditlog"
	"integin/internal/shared/httpresponse"
)

type Handler struct {
	AuditLogRepository auditlog.Repository
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/audit-log":
		h.handleAppend(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/audit-log/verify":
		h.handleVerify(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/audit-log":
		h.handleQuery(w, r)
	default:
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
	}
}

func (h Handler) handleAppend(w http.ResponseWriter, r *http.Request) {
	var req auditlog.CreateEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := req.Validate(); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	tenantID := r.Header.Get("X-Tenant-ID")
	orgID := r.Header.Get("X-Organization-ID")
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "tenant_id and organization_id are required")
		return
	}

	ipAddress := req.IPAddress
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}
	userAgent := req.UserAgent
	if userAgent == "" {
		userAgent = r.UserAgent()
	}

	entry := auditlog.Entry{
		TenantID:       tenantID,
		OrganizationID: orgID,
		EventType:      req.EventType,
		EntityType:     req.EntityType,
		EntityID:       req.EntityID,
		ActorID:        req.ActorID,
		ActorName:      req.ActorName,
		Action:         req.Action,
		OldValue:       req.OldValue,
		NewValue:       req.NewValue,
		Metadata:       req.Metadata,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
	}

	created, err := h.AuditLogRepository.Append(r.Context(), entry)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "failed to append audit entry")
		return
	}

	httpresponse.JSON(w, http.StatusCreated, created)
}

func (h Handler) handleQuery(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	orgID := r.Header.Get("X-Organization-ID")
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "tenant_id and organization_id are required")
		return
	}

	req := auditlog.QueryRequest{
		TenantID:       tenantID,
		OrganizationID: orgID,
		EntityType:     auditlog.EntityType(r.URL.Query().Get("entity_type")),
		EntityID:       r.URL.Query().Get("entity_id"),
		ActorID:        r.URL.Query().Get("actor_id"),
		EventType:      r.URL.Query().Get("event_type"),
	}

	if v := r.URL.Query().Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "invalid from date")
			return
		}
		req.From = &t
	}
	if v := r.URL.Query().Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "invalid to date")
			return
		}
		req.To = &t
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			httpresponse.Error(w, http.StatusBadRequest, "invalid limit")
			return
		}
		req.Limit = n
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			httpresponse.Error(w, http.StatusBadRequest, "invalid offset")
			return
		}
		req.Offset = n
	}

	result, err := h.AuditLogRepository.Query(r.Context(), req)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "failed to query audit log")
		return
	}

	httpresponse.JSON(w, http.StatusOK, result)
}

func (h Handler) handleVerify(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	orgID := r.Header.Get("X-Organization-ID")
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "tenant_id and organization_id are required")
		return
	}

	var from, to *time.Time
	if v := r.URL.Query().Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "invalid from date")
			return
		}
		from = &t
	}
	if v := r.URL.Query().Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "invalid to date")
			return
		}
		to = &t
	}

	result, err := h.AuditLogRepository.VerifyChain(r.Context(), tenantID, orgID, from, to)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "failed to verify audit chain")
		return
	}

	httpresponse.JSON(w, http.StatusOK, result)
}
