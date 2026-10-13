package inspectionhttp

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"integin/internal/shared/httpresponse"
)

type Handler struct {
	DB *sql.DB
}

type InspectionPayload struct {
	ID             string                 `json:"id"`
	TenantID       string                 `json:"tenant_id"`
	OrganizationID string                 `json:"organization_id"`
	WorkOrderID    string                 `json:"work_order_id"`
	AssetID        string                 `json:"asset_id"`
	EquipmentType  string                 `json:"equipment_type"`
	Status         string                 `json:"status"`
	Answers        map[string]interface{} `json:"answers"`
	Photos         []string               `json:"photos"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	tenantID := r.Header.Get("X-Tenant-ID")
	orgID := r.Header.Get("X-Organization-ID")
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing tenant or organization header")
		return
	}

	var payload InspectionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}

	payload.TenantID = tenantID
	payload.OrganizationID = orgID

	answersJSON, _ := json.Marshal(payload.Answers)
	photosJSON, _ := json.Marshal(payload.Photos)

	if payload.ID == "" {
		payload.ID = tenantID + ":inspection:" + time.Now().Format("20060102150405")
	}

	_, err := h.DB.ExecContext(r.Context(),
		`INSERT INTO inspection_record (id, tenant_id, organization_id, work_order_id, asset_id, equipment_type, status, answers, photos, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, answers = EXCLUDED.answers, photos = EXCLUDED.photos, updated_at = EXCLUDED.updated_at`,
		payload.ID, payload.TenantID, payload.OrganizationID, payload.WorkOrderID,
		payload.AssetID, payload.EquipmentType, payload.Status, answersJSON, photosJSON,
		payload.CreatedAt, payload.UpdatedAt,
	)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "synced", "id": payload.ID})
}
