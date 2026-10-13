package certtemplatepg

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"integin/internal/domain/certificatetemplate"
)

func (r *Repository) ResolveInspectionTemplate(ctx context.Context, actor certificatetemplate.ActorContext, templateCode string, version int, inspectionID string) (certificatetemplate.ResolvedTemplate, error) {
	if err := actor.Validate(); err != nil {
		return certificatetemplate.ResolvedTemplate{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return certificatetemplate.ResolvedTemplate{}, err
	}
	defer tx.Rollback()
	record, err := load(ctx, tx, actor, templateCode, version)
	if err != nil {
		return certificatetemplate.ResolvedTemplate{}, err
	}
	var id, workOrderID, assetID, inspectorID, lifecycleState, finalizationState string
	var revision int
	var createdAt, updatedAt time.Time
	err = tx.QueryRowContext(ctx, `SELECT id, work_order_id, asset_id, inspector_id, lifecycle_state, revision, finalization_state, created_at, updated_at FROM inspection_record WHERE id=$1 AND tenant_id=$2 AND organization_id=$3`, inspectionID, actor.TenantID, actor.OrganizationID).Scan(&id, &workOrderID, &assetID, &inspectorID, &lifecycleState, &revision, &finalizationState, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return certificatetemplate.ResolvedTemplate{}, ErrNotFound
		}
		return certificatetemplate.ResolvedTemplate{}, err
	}
	resolved, err := certificatetemplate.Resolve(record, certificatetemplate.InspectionValues(id, workOrderID, assetID, inspectorID, lifecycleState, revision, finalizationState, createdAt, updatedAt))
	if err != nil {
		return certificatetemplate.ResolvedTemplate{}, err
	}
	return resolved, tx.Commit()
}
