package custodyhandoverpg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"integin/internal/domain/custodyhandover"
	"integin/internal/shared/pgtx"
)

var (
	ErrNilDB    = errors.New("custody handover repository requires a database")
	ErrNotFound = errors.New("custody or handover record not found")
)

// Repository provides PostgreSQL persistence for custody chain and site handover.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new custody and handover repository.
func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

func (r *Repository) begin(ctx context.Context, actor custodyhandover.ActorContext) (*sql.Tx, error) {
	return pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
}

// RecordCustody appends an immutable custody event into the custody_chain table.
func (r *Repository) RecordCustody(ctx context.Context, actor custodyhandover.ActorContext, record custodyhandover.CustodyRecord) (custodyhandover.CustodyRecord, error) {
	if err := actor.Validate(); err != nil {
		return custodyhandover.CustodyRecord{}, err
	}
	record.TenantID = actor.TenantID
	record.OrganizationID = actor.OrganizationID
	record.RecordedBy = actor.ActorID
	if record.RecordedAt.IsZero() {
		record.RecordedAt = time.Now().UTC()
	}
	if record.VerificationHash == "" {
		record.VerificationHash = custodyhandover.ComputeVerificationHash(
			record.TenantID,
			record.OrganizationID,
			record.AssetID,
			record.LocationID,
			record.CustodianID,
			record.Action,
			record.RecordedAt,
		)
	}
	if err := record.Validate(); err != nil {
		return custodyhandover.CustodyRecord{}, err
	}

	tx, err := r.begin(ctx, actor)
	if err != nil {
		return custodyhandover.CustodyRecord{}, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO custody_chain (
			id, tenant_id, organization_id, asset_id, work_order_id,
			location_id, site_name, custodian_id, action, verification_hash,
			notes, recorded_by, recorded_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13
		)
	`
	_, err = tx.ExecContext(ctx, query,
		record.ID,
		record.TenantID,
		record.OrganizationID,
		record.AssetID,
		record.WorkOrderID,
		record.LocationID,
		record.SiteName,
		record.CustodianID,
		string(record.Action),
		record.VerificationHash,
		record.Notes,
		record.RecordedBy,
		record.RecordedAt,
	)
	if err != nil {
		return custodyhandover.CustodyRecord{}, fmt.Errorf("failed to record custody: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return custodyhandover.CustodyRecord{}, err
	}
	return record, nil
}

// RequestHandover creates a new HandoverRequest in REQUESTED state.
func (r *Repository) RequestHandover(ctx context.Context, actor custodyhandover.ActorContext, req custodyhandover.HandoverRequest) (custodyhandover.HandoverRequest, error) {
	if err := actor.Validate(); err != nil {
		return custodyhandover.HandoverRequest{}, err
	}
	req.TenantID = actor.TenantID
	req.OrganizationID = actor.OrganizationID
	req.RequestedBy = actor.ActorID
	if req.RequestedAt.IsZero() {
		req.RequestedAt = time.Now().UTC()
	}
	req.State = custodyhandover.HandoverStateRequested
	req.Revision = 1
	req.CreatedAt = req.RequestedAt
	req.UpdatedAt = req.RequestedAt

	if err := req.Validate(); err != nil {
		return custodyhandover.HandoverRequest{}, err
	}

	tx, err := r.begin(ctx, actor)
	if err != nil {
		return custodyhandover.HandoverRequest{}, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO work_order_site_handover (
			id, tenant_id, organization_id, work_order_id, asset_id,
			from_assignment_id, to_assignment_id, from_site_id, to_site_id,
			state, reason, requested_by, requested_at, revision, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15, $16
		)
	`
	_, err = tx.ExecContext(ctx, query,
		req.ID,
		req.TenantID,
		req.OrganizationID,
		req.WorkOrderID,
		req.AssetID,
		req.FromAssignmentID,
		req.ToAssignmentID,
		req.FromSiteID,
		req.ToSiteID,
		string(req.State),
		req.Reason,
		req.RequestedBy,
		req.RequestedAt,
		req.Revision,
		req.CreatedAt,
		req.UpdatedAt,
	)
	if err != nil {
		return custodyhandover.HandoverRequest{}, fmt.Errorf("failed to create handover request: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return custodyhandover.HandoverRequest{}, err
	}
	return req, nil
}

// GetHandover retrieves a handover request by ID under forced RLS.
func (r *Repository) GetHandover(ctx context.Context, actor custodyhandover.ActorContext, id string) (custodyhandover.HandoverRequest, error) {
	if err := actor.Validate(); err != nil {
		return custodyhandover.HandoverRequest{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return custodyhandover.HandoverRequest{}, err
	}
	defer tx.Rollback()

	query := `
		SELECT id, tenant_id, organization_id, work_order_id, asset_id,
		       from_assignment_id, to_assignment_id, from_site_id, to_site_id,
		       state, reason, requested_by, requested_at,
		       acknowledged_by, acknowledged_at,
		       approved_by, approved_at,
		       transferred_by, transferred_at,
		       rejection_reason, revision, created_at, updated_at
		FROM work_order_site_handover
		WHERE id = $1 AND tenant_id = $2 AND organization_id = $3
	`
	var (
		req             custodyhandover.HandoverRequest
		stateStr        string
		ackBy           sql.NullString
		ackAt           sql.NullTime
		appBy           sql.NullString
		appAt           sql.NullTime
		transBy         sql.NullString
		transAt         sql.NullTime
		rejectionReason string
	)
	err = tx.QueryRowContext(ctx, query, id, actor.TenantID, actor.OrganizationID).Scan(
		&req.ID,
		&req.TenantID,
		&req.OrganizationID,
		&req.WorkOrderID,
		&req.AssetID,
		&req.FromAssignmentID,
		&req.ToAssignmentID,
		&req.FromSiteID,
		&req.ToSiteID,
		&stateStr,
		&req.Reason,
		&req.RequestedBy,
		&req.RequestedAt,
		&ackBy,
		&ackAt,
		&appBy,
		&appAt,
		&transBy,
		&transAt,
		&rejectionReason,
		&req.Revision,
		&req.CreatedAt,
		&req.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return custodyhandover.HandoverRequest{}, ErrNotFound
	}
	if err != nil {
		return custodyhandover.HandoverRequest{}, err
	}

	req.State = custodyhandover.HandoverState(stateStr)
	req.RejectionReason = rejectionReason
	if ackBy.Valid {
		req.AcknowledgedBy = ackBy.String
	}
	if ackAt.Valid {
		req.AcknowledgedAt = &ackAt.Time
	}
	if appBy.Valid {
		req.ApprovedBy = appBy.String
	}
	if appAt.Valid {
		req.ApprovedAt = &appAt.Time
	}
	if transBy.Valid {
		req.TransferredBy = transBy.String
	}
	if transAt.Valid {
		req.TransferredAt = &transAt.Time
	}

	if err := tx.Commit(); err != nil {
		return custodyhandover.HandoverRequest{}, err
	}
	return req, nil
}

// UpdateHandoverState advances the handover state machine with optimistic concurrency locking.
func (r *Repository) UpdateHandoverState(ctx context.Context, actor custodyhandover.ActorContext, id string, target custodyhandover.HandoverState, expectedRev int64, extra custodyhandover.HandoverTransitionMetadata) (custodyhandover.HandoverRequest, error) {
	if err := actor.Validate(); err != nil {
		return custodyhandover.HandoverRequest{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return custodyhandover.HandoverRequest{}, err
	}
	defer tx.Rollback()

	current, err := r.GetHandover(ctx, actor, id)
	if err != nil {
		return custodyhandover.HandoverRequest{}, err
	}
	if current.Revision != expectedRev {
		return custodyhandover.HandoverRequest{}, custodyhandover.ErrStaleRevision
	}
	if !custodyhandover.CanTransition(current.State, target) {
		return custodyhandover.HandoverRequest{}, custodyhandover.ErrInvalidTransition
	}

	current.State = target
	current.Revision = expectedRev + 1
	current.UpdatedAt = time.Now().UTC()

	switch target {
	case custodyhandover.HandoverStateAcknowledged:
		current.AcknowledgedBy = extra.AcknowledgedBy
		current.AcknowledgedAt = extra.AcknowledgedAt
	case custodyhandover.HandoverStateApproved:
		current.ApprovedBy = extra.ApprovedBy
		current.ApprovedAt = extra.ApprovedAt
	case custodyhandover.HandoverStateTransferred:
		current.TransferredBy = extra.TransferredBy
		current.TransferredAt = extra.TransferredAt
	case custodyhandover.HandoverStateRejected:
		current.RejectionReason = extra.RejectionReason
	}

	if err := current.Validate(); err != nil {
		return custodyhandover.HandoverRequest{}, err
	}

	query := `
		UPDATE work_order_site_handover
		SET state = $1, acknowledged_by = $2, acknowledged_at = $3,
		    approved_by = $4, approved_at = $5,
		    transferred_by = $6, transferred_at = $7,
		    rejection_reason = $8, revision = $9, updated_at = $10
		WHERE id = $11 AND tenant_id = $12 AND organization_id = $13 AND revision = $14
	`
	res, err := tx.ExecContext(ctx, query,
		string(current.State),
		current.AcknowledgedBy,
		current.AcknowledgedAt,
		current.ApprovedBy,
		current.ApprovedAt,
		current.TransferredBy,
		current.TransferredAt,
		current.RejectionReason,
		current.Revision,
		current.UpdatedAt,
		current.ID,
		actor.TenantID,
		actor.OrganizationID,
		expectedRev,
	)
	if err != nil {
		return custodyhandover.HandoverRequest{}, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return custodyhandover.HandoverRequest{}, err
	}
	if rows == 0 {
		return custodyhandover.HandoverRequest{}, custodyhandover.ErrStaleRevision
	}

	if err := tx.Commit(); err != nil {
		return custodyhandover.HandoverRequest{}, err
	}
	return current, nil
}

