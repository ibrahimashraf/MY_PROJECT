package calibration

import (
	"errors"
	"strings"
	"time"

	"integin/internal/shared/types"
)

type Record struct {
	ID                string                  `json:"id"`
	TenantID          string                  `json:"tenant_id"`
	EquipmentID       string                  `json:"equipment_id"`
	StandardReference string                  `json:"standard_reference"`
	CalibrationDate   time.Time               `json:"calibration_date"`
	NextDueDate       time.Time               `json:"next_due_date"`
	TechnicianID      string                  `json:"technician_id"`
	Result            string                  `json:"result"`
	Status            types.CalibrationStatus `json:"status"`
}

func (r Record) Validate() error {
	required := map[string]string{"id": r.ID, "tenant_id": r.TenantID, "equipment_id": r.EquipmentID, "standard_reference": r.StandardReference, "technician_id": r.TechnicianID}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return errors.New(field + " is required")
		}
	}
	if r.CalibrationDate.IsZero() || r.NextDueDate.IsZero() {
		return errors.New("calibration dates are required")
	}
	if r.NextDueDate.Before(r.CalibrationDate) {
		return errors.New("next due date cannot precede calibration date")
	}
	if r.Status != types.CalibrationActive && r.Status != types.CalibrationExpired && r.Status != types.CalibrationSuperseded {
		return errors.New("invalid calibration status")
	}
	return nil
}

func (r Record) IsExpired(at time.Time) bool { return at.After(r.NextDueDate) }
