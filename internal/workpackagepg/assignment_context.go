package workpackagepg

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"integin/internal/domain/workpackage"
)

// GetAssignmentContext resolves immutable inspection context only for the
// current unexpired device assignment within the caller's tenant RLS scope.
func (r *Repository) GetAssignmentContext(
	ctx context.Context,
	tenantID, organizationID, inspectionID, deviceID string,
	now time.Time,
) (workpackage.AssignmentContext, error) {
	if r == nil || r.db == nil {
		return workpackage.AssignmentContext{}, errors.New("work package repository is unavailable")
	}
	tenantID, organizationID = strings.TrimSpace(tenantID), strings.TrimSpace(organizationID)
	inspectionID, deviceID = strings.TrimSpace(inspectionID), strings.TrimSpace(deviceID)
	if tenantID == "" || organizationID == "" || inspectionID == "" || deviceID == "" || now.IsZero() {
		return workpackage.AssignmentContext{}, errors.New("assignment context scope and lookup time are required")
	}
	tx, err := r.scopedTx(ctx, tenantID, organizationID)
	if err != nil {
		return workpackage.AssignmentContext{}, err
	}
	defer tx.Rollback()

	var result workpackage.AssignmentContext
	var fieldAssetIDs []byte
	err = tx.QueryRowContext(ctx, `
		SELECT c.root_asset_id, c.inspection_type, c.procedure_version, c.scheduled_at, c.field_asset_ids
		FROM work_package_assignment_context c
		JOIN work_package_assignment a
		  ON a.tenant_id = c.tenant_id
		 AND a.organization_id = c.organization_id
		 AND a.inspection_id = c.inspection_id
		WHERE c.tenant_id = $1 AND c.organization_id = $2 AND c.inspection_id = $3
		  AND a.device_id = $4 AND a.expires_at > $5
	`, tenantID, organizationID, inspectionID, deviceID, now.UTC()).Scan(
		&result.RootAssetID, &result.InspectionType, &result.ProcedureVersion, &result.ScheduledAt, &fieldAssetIDs,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return workpackage.AssignmentContext{}, ErrNotFound
	}
	if err != nil {
		return workpackage.AssignmentContext{}, fmt.Errorf("get assignment context: %w", err)
	}
	if err := json.Unmarshal(fieldAssetIDs, &result.FieldAssetIDs); err != nil {
		return workpackage.AssignmentContext{}, fmt.Errorf("decode assignment context field assets: %w", err)
	}
	if err := result.Validate(); err != nil {
		return workpackage.AssignmentContext{}, fmt.Errorf("validate assignment context: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return workpackage.AssignmentContext{}, fmt.Errorf("commit assignment context lookup: %w", err)
	}
	return result, nil
}
