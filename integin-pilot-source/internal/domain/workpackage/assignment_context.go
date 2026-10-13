package workpackage

import (
	"errors"
	"strings"
	"time"
)

// AssignmentContext carries the inspection-specific values needed to render an
// approved package as a Field work pack. It is immutable once assigned and is
// intended to be persisted and included in a future signed manifest.
type AssignmentContext struct {
	RootAssetID      string            `json:"root_asset_id"`
	InspectionType   string            `json:"inspection_type"`
	ProcedureVersion string            `json:"procedure_version"`
	ScheduledAt      time.Time         `json:"scheduled_at"`
	FieldAssetIDs    map[string]string `json:"field_asset_ids"`
}

// Validate rejects incomplete or ambiguous context so a Field client never
// manufactures inspection attributes while building a cached work pack.
func (c AssignmentContext) Validate() error {
	if strings.TrimSpace(c.RootAssetID) == "" || strings.TrimSpace(c.InspectionType) == "" || strings.TrimSpace(c.ProcedureVersion) == "" || c.ScheduledAt.IsZero() {
		return errors.New("assignment context is incomplete")
	}
	if strings.Contains(c.RootAssetID, "|") || strings.Contains(c.InspectionType, "|") || strings.Contains(c.ProcedureVersion, "|") {
		return errors.New("assignment context fields cannot contain pipe delimiter '|'")
	}
	for fieldID, assetID := range c.FieldAssetIDs {
		if strings.TrimSpace(fieldID) == "" || strings.TrimSpace(assetID) == "" {
			return errors.New("assignment context field asset mapping is invalid")
		}
		if strings.Contains(fieldID, "|") || strings.Contains(fieldID, "=") || strings.Contains(fieldID, ",") ||
			strings.Contains(assetID, "|") || strings.Contains(assetID, "=") || strings.Contains(assetID, ",") {
			return errors.New("assignment context field asset mapping contains reserved delimiters")
		}
	}
	return nil
}
