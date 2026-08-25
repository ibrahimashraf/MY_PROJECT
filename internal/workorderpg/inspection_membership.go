package workorderpg

import (
	"context"
	"database/sql"
	"errors"

	"integin/internal/domain/workorder"
)

var ErrInspectionMembershipInvalid = errors.New("work-order inspection membership is invalid or ineligible")

type inspectionMembership struct {
	InspectionID string
	ScopeItemID  string
}

// transactionalInspectionMembershipValidator keeps authoritative inspection
// validation and normalized submission writes inside the Work-Order mutation
// transaction. The legacy validator contract remains available for isolated
// tests only; a production composition must supply this stronger contract.
type transactionalInspectionMembershipValidator interface {
	ValidatePartialSubmissionTx(context.Context, *sql.Tx, workorder.ActorContext, string, string, []string) ([]inspectionMembership, error)
	RecordPartialSubmissionTx(context.Context, *sql.Tx, workorder.ActorContext, string, string, string, []inspectionMembership) error
}

type PostgresInspectionMembershipValidator struct{}

func NewPostgresInspectionMembershipValidator() *PostgresInspectionMembershipValidator {
	return &PostgresInspectionMembershipValidator{}
}

func (v *PostgresInspectionMembershipValidator) ValidatePartialSubmission(
	context.Context,
	workorder.ActorContext,
	string,
	string,
	[]string,
) error {
	return ErrInspectionMembershipUnavailable
}

func (v *PostgresInspectionMembershipValidator) ValidatePartialSubmissionTx(
	ctx context.Context,
	tx *sql.Tx,
	actor workorder.ActorContext,
	workOrderID, assignmentID string,
	inspectionIDs []string,
) ([]inspectionMembership, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT i.id, i.scope_item_id
		FROM inspection_record i
		JOIN work_order_assignment a
		  ON a.tenant_id=i.tenant_id AND a.organization_id=i.organization_id
		 AND a.work_order_id=i.work_order_id AND a.id=i.assignment_id
		JOIN work_order_assignment_scope ass
		  ON ass.tenant_id=i.tenant_id AND ass.organization_id=i.organization_id
		 AND ass.assignment_id=i.assignment_id AND ass.scope_item_id=i.scope_item_id
		JOIN work_order_scope_item s
		  ON s.tenant_id=i.tenant_id AND s.organization_id=i.organization_id
		 AND s.work_order_id=i.work_order_id AND s.id=i.scope_item_id
		WHERE i.tenant_id=$1 AND i.organization_id=$2
		  AND i.work_order_id=$3 AND i.assignment_id=$4
		  AND i.id = ANY($5::text[])
		  AND a.state='active'
		  AND i.inspector_id=a.inspector_id
		  AND i.asset_id=s.asset_id
		  AND i.lifecycle_state IN ('COMPLETED','PENDING_REVIEW')
		  AND i.finalization_state='OPEN'
		FOR UPDATE OF i`,
		actor.TenantID, actor.OrganizationID, workOrderID, assignmentID, pqStringArray(inspectionIDs),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	memberships := make([]inspectionMembership, 0, len(inspectionIDs))
	for rows.Next() {
		var membership inspectionMembership
		if err := rows.Scan(&membership.InspectionID, &membership.ScopeItemID); err != nil {
			return nil, err
		}
		memberships = append(memberships, membership)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(memberships) != len(inspectionIDs) {
		return nil, ErrInspectionMembershipInvalid
	}
	return memberships, nil
}

func (v *PostgresInspectionMembershipValidator) RecordPartialSubmissionTx(
	ctx context.Context,
	tx *sql.Tx,
	actor workorder.ActorContext,
	segmentID, workOrderID, assignmentID string,
	memberships []inspectionMembership,
) error {
	for _, membership := range memberships {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO work_order_submission_item
				(tenant_id,organization_id,submission_segment_id,work_order_id,assignment_id,scope_item_id,inspection_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			actor.TenantID, actor.OrganizationID, segmentID, workOrderID, assignmentID, membership.ScopeItemID, membership.InspectionID,
		); err != nil {
			return err
		}
	}
	ids := make([]string, 0, len(memberships))
	for _, membership := range memberships {
		ids = append(ids, membership.InspectionID)
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE inspection_record
		SET finalization_state='SUBMITTED', updated_by=$1, updated_at=now()
		WHERE tenant_id=$2 AND organization_id=$3 AND id = ANY($4::text[]) AND finalization_state='OPEN'`,
		actor.ActorID, actor.TenantID, actor.OrganizationID, pqStringArray(ids),
	)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil || updated != int64(len(ids)) {
		return ErrInspectionMembershipInvalid
	}
	return nil
}
