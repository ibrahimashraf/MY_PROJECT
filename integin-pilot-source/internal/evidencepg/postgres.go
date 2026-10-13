package evidencepg

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"integin/internal/domain/evidence"
	"integin/internal/domain/workorder"
	"integin/internal/shared/pgtx"
)

var (
	ErrNilDB                      = errors.New("evidence metadata postgres repository requires a database")
	ErrInspectionAssignmentDenied = errors.New("evidence inspection assignment is not active for actor")
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

func (r *Repository) Register(ctx context.Context, actor evidence.ActorContext, metadata evidence.Metadata) (evidence.Metadata, bool, error) {
	if err := metadata.ValidateForRegistration(actor); err != nil {
		return evidence.Metadata{}, false, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return evidence.Metadata{}, false, err
	}
	defer tx.Rollback()
	stored, inserted, err := register(ctx, tx, actor, metadata)
	if err != nil {
		return evidence.Metadata{}, false, err
	}
	return stored, inserted, tx.Commit()
}

// RegisterForActiveAssignment keeps persisted inspection assignment scope and
// immutable metadata registration inside one RLS-scoped transaction. Its caller
// must separately verify role and capability from server-derived membership.
func (r *Repository) RegisterForActiveAssignment(ctx context.Context, actor workorder.ActorContext, metadata evidence.Metadata) (evidence.Metadata, bool, error) {
	if err := actor.Validate(); err != nil {
		return evidence.Metadata{}, false, err
	}
	evidenceActor := evidence.ActorContext{TenantID: actor.TenantID, OrganizationID: actor.OrganizationID, ActorID: actor.ActorID}
	if err := metadata.ValidateForRegistration(evidenceActor); err != nil {
		return evidence.Metadata{}, false, err
	}
	tx, err := r.begin(ctx, evidenceActor)
	if err != nil {
		return evidence.Metadata{}, false, err
	}
	defer tx.Rollback()
	var permitted bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1
		FROM inspection_record inspection
		JOIN work_order_assignment assignment
		  ON assignment.id=inspection.assignment_id
		 AND assignment.tenant_id=inspection.tenant_id
		 AND assignment.organization_id=inspection.organization_id
		WHERE inspection.id=$1
		  AND inspection.tenant_id=$2
		  AND inspection.organization_id=$3
		  AND assignment.state='active'
		  AND ($4='administrator' OR ($4='inspector' AND assignment.inspector_id=$5))
	)`, metadata.InspectionID, actor.TenantID, actor.OrganizationID, actor.Role, actor.ActorID).Scan(&permitted)
	if err != nil {
		return evidence.Metadata{}, false, err
	}
	if !permitted {
		return evidence.Metadata{}, false, ErrInspectionAssignmentDenied
	}
	stored, inserted, err := register(ctx, tx, evidenceActor, metadata)
	if err != nil {
		return evidence.Metadata{}, false, err
	}
	return stored, inserted, tx.Commit()
}

func (r *Repository) ListByInspection(ctx context.Context, actor evidence.ActorContext, inspectionID string) ([]evidence.Metadata, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT id,tenant_id,organization_id,inspection_id,object_key,content_type,ciphertext_bytes,plaintext_sha256,ciphertext_sha256,captured_at,device_id,authority_id,authority_epoch,transaction_id,receipt_id,signature_algorithm,key_id,encryption_algorithm,encryption_key_reference,classification,retention_reference,hold_state,redaction_policy_reference,registered_by,registered_at FROM evidence_metadata WHERE inspection_id=$1 AND tenant_id=$2 AND organization_id=$3 ORDER BY captured_at,id`, inspectionID, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []evidence.Metadata
	for rows.Next() {
		metadata, err := scan(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, metadata)
	}
	return records, rows.Err()
}

func (r *Repository) ListByTenantOrganization(ctx context.Context, actor evidence.ActorContext) ([]evidence.Metadata, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT id,tenant_id,organization_id,inspection_id,object_key,content_type,ciphertext_bytes,plaintext_sha256,ciphertext_sha256,captured_at,device_id,authority_id,authority_epoch,transaction_id,receipt_id,signature_algorithm,key_id,encryption_algorithm,encryption_key_reference,classification,retention_reference,hold_state,redaction_policy_reference,registered_by,registered_at FROM evidence_metadata WHERE tenant_id=$1 AND organization_id=$2 ORDER BY captured_at,id`, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []evidence.Metadata
	for rows.Next() {
		metadata, err := scan(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, metadata)
	}
	return records, rows.Err()
}

func register(ctx context.Context, tx *sql.Tx, actor evidence.ActorContext, metadata evidence.Metadata) (evidence.Metadata, bool, error) {
	result, err := tx.ExecContext(ctx, `
		INSERT INTO evidence_metadata (
			id, tenant_id, organization_id, inspection_id, object_key, content_type, ciphertext_bytes,
			plaintext_sha256, ciphertext_sha256, captured_at, device_id, authority_id, authority_epoch,
			transaction_id, receipt_id, signature_algorithm, key_id, encryption_algorithm, encryption_key_reference,
			classification, retention_reference, hold_state, redaction_policy_reference, registered_by, registered_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,COALESCE($25,now()))
		ON CONFLICT (tenant_id, organization_id, id) DO NOTHING`,
		metadata.ID, metadata.TenantID, metadata.OrganizationID, metadata.InspectionID, metadata.ObjectKey,
		metadata.ContentType, metadata.CiphertextBytes, metadata.PlaintextSHA256, metadata.CiphertextSHA256,
		metadata.CapturedAt, metadata.DeviceID, metadata.AuthorityID, metadata.AuthorityEpoch, metadata.TransactionID,
		metadata.ReceiptID, metadata.SignatureAlgorithm, metadata.KeyID, metadata.EncryptionAlgorithm,
		metadata.EncryptionKeyRef, metadata.Classification, metadata.RetentionReference, metadata.HoldState,
		metadata.RedactionPolicyRef, metadata.RegisteredBy, nullableTime(metadata.RegisteredAt))
	if err != nil {
		return evidence.Metadata{}, false, err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return evidence.Metadata{}, false, err
	}
	stored, err := load(ctx, tx, actor, metadata.ID)
	if err != nil {
		return evidence.Metadata{}, false, err
	}
	if inserted == 0 && !stored.SameImmutableContent(metadata) {
		return evidence.Metadata{}, false, evidence.ErrImmutableConflict
	}
	return stored, inserted == 1, nil
}

func (r *Repository) begin(ctx context.Context, actor evidence.ActorContext) (*sql.Tx, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	return pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
}

type scanner interface{ Scan(...any) error }

func load(ctx context.Context, tx *sql.Tx, actor evidence.ActorContext, id string) (evidence.Metadata, error) {
	row := tx.QueryRowContext(ctx, `SELECT id,tenant_id,organization_id,inspection_id,object_key,content_type,ciphertext_bytes,plaintext_sha256,ciphertext_sha256,captured_at,device_id,authority_id,authority_epoch,transaction_id,receipt_id,signature_algorithm,key_id,encryption_algorithm,encryption_key_reference,classification,retention_reference,hold_state,redaction_policy_reference,registered_by,registered_at FROM evidence_metadata WHERE id=$1 AND tenant_id=$2 AND organization_id=$3`, id, actor.TenantID, actor.OrganizationID)
	return scan(row)
}

func scan(row scanner) (evidence.Metadata, error) {
	var metadata evidence.Metadata
	err := row.Scan(&metadata.ID, &metadata.TenantID, &metadata.OrganizationID, &metadata.InspectionID, &metadata.ObjectKey, &metadata.ContentType, &metadata.CiphertextBytes, &metadata.PlaintextSHA256, &metadata.CiphertextSHA256, &metadata.CapturedAt, &metadata.DeviceID, &metadata.AuthorityID, &metadata.AuthorityEpoch, &metadata.TransactionID, &metadata.ReceiptID, &metadata.SignatureAlgorithm, &metadata.KeyID, &metadata.EncryptionAlgorithm, &metadata.EncryptionKeyRef, &metadata.Classification, &metadata.RetentionReference, &metadata.HoldState, &metadata.RedactionPolicyRef, &metadata.RegisteredBy, &metadata.RegisteredAt)
	return metadata, err
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
