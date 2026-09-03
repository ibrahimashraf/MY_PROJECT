package evidencepackpg

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"integin/internal/domain/evidencepack"
	"integin/internal/shared/pgtx"
)

var (
	ErrNilDB    = errors.New("evidence pack postgres repository requires a database")
	ErrNotFound = errors.New("evidence pack or release pack not found")
)

// PackRepository provides persistence for evidence packs.
type PackRepository struct {
	db *sql.DB
}

// NewPackRepository creates an evidence pack repository.
func NewPackRepository(db *sql.DB) (*PackRepository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &PackRepository{db: db}, nil
}

func (r *PackRepository) Create(ctx context.Context, actor evidencepack.ActorContext, pack evidencepack.EvidencePack) (evidencepack.EvidencePack, error) {
	if err := actor.Validate(); err != nil {
		return evidencepack.EvidencePack{}, err
	}
	pack.TenantID = actor.TenantID
	pack.OrganizationID = actor.OrganizationID
	pack.CreatedBy = actor.ActorID
	pack.UpdatedBy = actor.ActorID
	if pack.CreatedAt.IsZero() {
		pack.CreatedAt = time.Now().UTC()
	}
	pack.UpdatedAt = pack.CreatedAt
	if pack.Revision <= 0 {
		pack.Revision = 1
	}
	if pack.Status == "" {
		pack.Status = evidencepack.PackStatusDraft
	}
	if err := pack.Validate(); err != nil {
		return evidencepack.EvidencePack{}, err
	}
	manifest, err := json.Marshal(pack.Records)
	if err != nil {
		return evidencepack.EvidencePack{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return evidencepack.EvidencePack{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx,
		`INSERT INTO evidence_pack (id, tenant_id, organization_id, inspection_id, pack_name, status, evidence_manifest, evidence_count, classification, retention_reference, hold_state, redaction_policy_ref, pack_checksum, created_by, created_at, updated_by, updated_at, revision)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		pack.ID, actor.TenantID, actor.OrganizationID, pack.InspectionID, pack.PackName,
		string(pack.Status), string(manifest), len(pack.Records), pack.Classification,
		pack.RetentionReference, pack.HoldState, pack.RedactionPolicyRef, pack.PackChecksum,
		pack.CreatedBy, pack.CreatedAt, pack.UpdatedBy, pack.UpdatedAt, pack.Revision)
	if err != nil {
		return evidencepack.EvidencePack{}, err
	}
	stored, err := loadPack(ctx, tx, actor, pack.ID)
	if err != nil {
		return evidencepack.EvidencePack{}, err
	}
	return stored, tx.Commit()
}

func (r *PackRepository) Get(ctx context.Context, actor evidencepack.ActorContext, packID string) (evidencepack.EvidencePack, error) {
	if err := actor.Validate(); err != nil {
		return evidencepack.EvidencePack{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return evidencepack.EvidencePack{}, err
	}
	defer tx.Rollback()
	pack, err := loadPack(ctx, tx, actor, packID)
	if err != nil {
		return evidencepack.EvidencePack{}, err
	}
	return pack, tx.Commit()
}

func (r *PackRepository) GetByName(ctx context.Context, actor evidencepack.ActorContext, packName string) (evidencepack.EvidencePack, error) {
	if err := actor.Validate(); err != nil {
		return evidencepack.EvidencePack{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return evidencepack.EvidencePack{}, err
	}
	defer tx.Rollback()
	var pack evidencepack.EvidencePack
	pack, err = loadPackByName(ctx, tx, actor, packName)
	if err != nil {
		return evidencepack.EvidencePack{}, err
	}
	return pack, tx.Commit()
}

func (r *PackRepository) ListByInspection(ctx context.Context, actor evidencepack.ActorContext, inspectionID string) ([]evidencepack.EvidencePack, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx,
		`SELECT id, inspection_id, pack_name, status, evidence_manifest, evidence_count, classification, retention_reference, hold_state, redaction_policy_ref, pack_checksum, created_by, created_at, updated_by, updated_at, revision
		 FROM evidence_pack WHERE tenant_id=$1 AND organization_id=$2 AND inspection_id=$3
		 ORDER BY created_at DESC`,
		actor.TenantID, actor.OrganizationID, inspectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var packs []evidencepack.EvidencePack
	for rows.Next() {
		pack, err := scanPackRow(rows)
		if err != nil {
			return nil, err
		}
		pack.TenantID = actor.TenantID
		pack.OrganizationID = actor.OrganizationID
		packs = append(packs, pack)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return packs, tx.Commit()
}

func (r *PackRepository) UpdateStatus(ctx context.Context, actor evidencepack.ActorContext, packID string, target evidencepack.PackStatus, expectedRevision int64) (evidencepack.EvidencePack, error) {
	if err := actor.Validate(); err != nil {
		return evidencepack.EvidencePack{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return evidencepack.EvidencePack{}, err
	}
	defer tx.Rollback()
	current, err := loadPack(ctx, tx, actor, packID)
	if errors.Is(err, evidencepack.ErrNotFound) {
		return evidencepack.EvidencePack{}, evidencepack.ErrNotFound
	}
	if err != nil {
		return evidencepack.EvidencePack{}, err
	}
	if current.Revision != expectedRevision {
		return evidencepack.EvidencePack{}, evidencepack.ErrStaleRevision
	}
	if err := current.CanTransitionTo(target); err != nil {
		return evidencepack.EvidencePack{}, err
	}
	if target == evidencepack.PackStatusSealed {
		if err := current.Seal(time.Now().UTC()); err != nil {
			return evidencepack.EvidencePack{}, err
		}
	} else {
		current.Status = target
	}
	current.Revision++
	current.UpdatedBy = actor.ActorID
	current.UpdatedAt = time.Now().UTC()
	if err := current.Validate(); err != nil {
		return evidencepack.EvidencePack{}, err
	}
	if err := savePack(ctx, tx, actor, current); err != nil {
		return evidencepack.EvidencePack{}, err
	}
	stored, err := loadPack(ctx, tx, actor, packID)
	if err != nil {
		return evidencepack.EvidencePack{}, err
	}
	return stored, tx.Commit()
}

// ReleaseRepository provides persistence for release ledger entries.
type ReleaseRepository struct {
	db *sql.DB
}

// NewReleaseRepository creates a release repository.
func NewReleaseRepository(db *sql.DB) (*ReleaseRepository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &ReleaseRepository{db: db}, nil
}

func (r *ReleaseRepository) Create(ctx context.Context, actor evidencepack.ActorContext, release evidencepack.ReleasePack) (evidencepack.ReleasePack, error) {
	if err := actor.Validate(); err != nil {
		return evidencepack.ReleasePack{}, err
	}
	release.TenantID = actor.TenantID
	release.OrganizationID = actor.OrganizationID
	release.CreatedBy = actor.ActorID
	release.UpdatedBy = actor.ActorID
	if release.CreatedAt.IsZero() {
		release.CreatedAt = time.Now().UTC()
	}
	release.UpdatedAt = release.CreatedAt
	if release.Revision <= 0 {
		release.Revision = 1
	}
	if release.Status == "" {
		release.Status = evidencepack.ReleaseStatusPending
	}
	if err := release.Validate(); err != nil {
		return evidencepack.ReleasePack{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return evidencepack.ReleasePack{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx,
		`INSERT INTO release_pack (id, tenant_id, organization_id, pack_id, recipient, purpose, status, classification, retention_reference, hold_state, redaction_policy_ref, created_by, created_at, updated_by, updated_at, revision)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		release.ID, actor.TenantID, actor.OrganizationID, release.PackID, release.Recipient,
		release.Purpose, string(release.Status), release.Classification, release.RetentionReference,
		release.HoldState, release.RedactionPolicyRef, release.CreatedBy, release.CreatedAt,
		release.UpdatedBy, release.UpdatedAt, release.Revision)
	if err != nil {
		return evidencepack.ReleasePack{}, err
	}
	stored, err := loadRelease(ctx, tx, actor, release.ID)
	if err != nil {
		return evidencepack.ReleasePack{}, err
	}
	return stored, tx.Commit()
}

func (r *ReleaseRepository) Get(ctx context.Context, actor evidencepack.ActorContext, releaseID string) (evidencepack.ReleasePack, error) {
	if err := actor.Validate(); err != nil {
		return evidencepack.ReleasePack{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return evidencepack.ReleasePack{}, err
	}
	defer tx.Rollback()
	release, err := loadRelease(ctx, tx, actor, releaseID)
	if err != nil {
		return evidencepack.ReleasePack{}, err
	}
	return release, tx.Commit()
}

func (r *ReleaseRepository) ListByPack(ctx context.Context, actor evidencepack.ActorContext, packID string) ([]evidencepack.ReleasePack, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx,
		`SELECT id, pack_id, recipient, purpose, status, classification, retention_reference, hold_state, redaction_policy_ref, COALESCE(approved_by,''), approved_at, COALESCE(released_by,''), released_at, created_by, created_at, updated_by, updated_at, revision
		 FROM release_pack WHERE tenant_id=$1 AND organization_id=$2 AND pack_id=$3
		 ORDER BY created_at DESC`,
		actor.TenantID, actor.OrganizationID, packID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var releases []evidencepack.ReleasePack
	for rows.Next() {
		var rel evidencepack.ReleasePack
		if err := rows.Scan(
			&rel.ID, &rel.PackID, &rel.Recipient, &rel.Purpose, &rel.Status, &rel.Classification,
			&rel.RetentionReference, &rel.HoldState, &rel.RedactionPolicyRef, &rel.ApprovedBy,
			&rel.ApprovedAt, &rel.ReleasedBy, &rel.ReleasedAt, &rel.CreatedBy, &rel.CreatedAt,
			&rel.UpdatedBy, &rel.UpdatedAt, &rel.Revision); err != nil {
			return nil, err
		}
		rel.TenantID = actor.TenantID
		rel.OrganizationID = actor.OrganizationID
		releases = append(releases, rel)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return releases, tx.Commit()
}

func (r *ReleaseRepository) UpdateStatus(ctx context.Context, actor evidencepack.ActorContext, releaseID string, target evidencepack.ReleaseStatus, expectedRevision int64) (evidencepack.ReleasePack, error) {
	if err := actor.Validate(); err != nil {
		return evidencepack.ReleasePack{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return evidencepack.ReleasePack{}, err
	}
	defer tx.Rollback()
	current, err := loadRelease(ctx, tx, actor, releaseID)
	if errors.Is(err, evidencepack.ErrNotFound) {
		return evidencepack.ReleasePack{}, evidencepack.ErrNotFound
	}
	if err != nil {
		return evidencepack.ReleasePack{}, err
	}
	if current.Revision != expectedRevision {
		return evidencepack.ReleasePack{}, evidencepack.ErrStaleRevision
	}
	if err := current.CanTransitionTo(target); err != nil {
		return evidencepack.ReleasePack{}, err
	}
	current.Status = target
	switch target {
	case evidencepack.ReleaseStatusApproved:
		now := time.Now().UTC()
		current.ApprovedBy = actor.ActorID
		current.ApprovedAt = &now
	case evidencepack.ReleaseStatusReleased:
		now := time.Now().UTC()
		current.ReleasedBy = actor.ActorID
		current.ReleasedAt = &now
	case evidencepack.ReleaseStatusPending:
		current.ApprovedBy = ""
		current.ApprovedAt = nil
		current.ReleasedBy = ""
		current.ReleasedAt = nil
	}
	current.Revision++
	current.UpdatedBy = actor.ActorID
	current.UpdatedAt = time.Now().UTC()
	if err := current.Validate(); err != nil {
		return evidencepack.ReleasePack{}, err
	}
	if err := saveRelease(ctx, tx, actor, current); err != nil {
		return evidencepack.ReleasePack{}, err
	}
	stored, err := loadRelease(ctx, tx, actor, releaseID)
	if err != nil {
		return evidencepack.ReleasePack{}, err
	}
	return stored, tx.Commit()
}

func loadPack(ctx context.Context, tx *sql.Tx, actor evidencepack.ActorContext, packID string) (evidencepack.EvidencePack, error) {
	var pack evidencepack.EvidencePack
	var manifest []byte
	var count int64
	err := tx.QueryRowContext(ctx,
		`SELECT id, inspection_id, pack_name, status, evidence_manifest, evidence_count, classification, retention_reference, hold_state, redaction_policy_ref, pack_checksum, created_by, created_at, updated_by, updated_at, revision
		 FROM evidence_pack WHERE tenant_id=$1 AND organization_id=$2 AND id=$3`,
		actor.TenantID, actor.OrganizationID, packID).Scan(
		&pack.ID, &pack.InspectionID, &pack.PackName, &pack.Status, &manifest, &count,
		&pack.Classification, &pack.RetentionReference, &pack.HoldState, &pack.RedactionPolicyRef,
		&pack.PackChecksum, &pack.CreatedBy, &pack.CreatedAt, &pack.UpdatedBy, &pack.UpdatedAt,
		&pack.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return evidencepack.EvidencePack{}, ErrNotFound
	}
	if err != nil {
		return evidencepack.EvidencePack{}, err
	}
	if len(manifest) > 0 {
		if err := json.Unmarshal(manifest, &pack.Records); err != nil {
			return evidencepack.EvidencePack{}, err
		}
	}
	pack.TenantID = actor.TenantID
	pack.OrganizationID = actor.OrganizationID
	return pack, nil
}

func loadPackByName(ctx context.Context, tx *sql.Tx, actor evidencepack.ActorContext, packName string) (evidencepack.EvidencePack, error) {
	var pack evidencepack.EvidencePack
	var manifest []byte
	var count int64
	err := tx.QueryRowContext(ctx,
		`SELECT id, inspection_id, pack_name, status, evidence_manifest, evidence_count, classification, retention_reference, hold_state, redaction_policy_ref, pack_checksum, created_by, created_at, updated_by, updated_at, revision
		 FROM evidence_pack WHERE tenant_id=$1 AND organization_id=$2 AND pack_name=$3`,
		actor.TenantID, actor.OrganizationID, packName).Scan(
		&pack.ID, &pack.InspectionID, &pack.PackName, &pack.Status, &manifest, &count,
		&pack.Classification, &pack.RetentionReference, &pack.HoldState, &pack.RedactionPolicyRef,
		&pack.PackChecksum, &pack.CreatedBy, &pack.CreatedAt, &pack.UpdatedBy, &pack.UpdatedAt,
		&pack.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return evidencepack.EvidencePack{}, ErrNotFound
	}
	if err != nil {
		return evidencepack.EvidencePack{}, err
	}
	if len(manifest) > 0 {
		if err := json.Unmarshal(manifest, &pack.Records); err != nil {
			return evidencepack.EvidencePack{}, err
		}
	}
	pack.TenantID = actor.TenantID
	pack.OrganizationID = actor.OrganizationID
	return pack, nil
}

func scanPackRow(rows *sql.Rows) (evidencepack.EvidencePack, error) {
	var pack evidencepack.EvidencePack
	var manifest []byte
	var count int64
	if err := rows.Scan(
		&pack.ID, &pack.InspectionID, &pack.PackName, &pack.Status, &manifest, &count,
		&pack.Classification, &pack.RetentionReference, &pack.HoldState, &pack.RedactionPolicyRef,
		&pack.PackChecksum, &pack.CreatedBy, &pack.CreatedAt, &pack.UpdatedBy, &pack.UpdatedAt,
		&pack.Revision); err != nil {
		return evidencepack.EvidencePack{}, err
	}
	if len(manifest) > 0 {
		if err := json.Unmarshal(manifest, &pack.Records); err != nil {
			return evidencepack.EvidencePack{}, err
		}
	}
	return pack, nil
}

func loadRelease(ctx context.Context, tx *sql.Tx, actor evidencepack.ActorContext, releaseID string) (evidencepack.ReleasePack, error) {
	var rel evidencepack.ReleasePack
	err := tx.QueryRowContext(ctx,
		`SELECT id, pack_id, recipient, purpose, status, classification, retention_reference, hold_state, redaction_policy_ref, COALESCE(approved_by,''), approved_at, COALESCE(released_by,''), released_at, created_by, created_at, updated_by, updated_at, revision
		 FROM release_pack WHERE tenant_id=$1 AND organization_id=$2 AND id=$3`,
		actor.TenantID, actor.OrganizationID, releaseID).Scan(
		&rel.ID, &rel.PackID, &rel.Recipient, &rel.Purpose, &rel.Status, &rel.Classification,
		&rel.RetentionReference, &rel.HoldState, &rel.RedactionPolicyRef, &rel.ApprovedBy,
		&rel.ApprovedAt, &rel.ReleasedBy, &rel.ReleasedAt, &rel.CreatedBy, &rel.CreatedAt,
		&rel.UpdatedBy, &rel.UpdatedAt, &rel.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return evidencepack.ReleasePack{}, ErrNotFound
	}
	if err != nil {
		return evidencepack.ReleasePack{}, err
	}
	rel.TenantID = actor.TenantID
	rel.OrganizationID = actor.OrganizationID
	return rel, nil
}

func savePack(ctx context.Context, tx *sql.Tx, actor evidencepack.ActorContext, pack evidencepack.EvidencePack) error {
	manifest, err := json.Marshal(pack.Records)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`UPDATE evidence_pack SET status=$1, evidence_manifest=$2, evidence_count=$3, pack_checksum=$4, updated_by=$5, updated_at=$6, revision=$7
		 WHERE tenant_id=$8 AND organization_id=$9 AND id=$10`,
		string(pack.Status), string(manifest), len(pack.Records), pack.PackChecksum,
		pack.UpdatedBy, pack.UpdatedAt, pack.Revision, actor.TenantID, actor.OrganizationID, pack.ID)
	return err
}

func saveRelease(ctx context.Context, tx *sql.Tx, actor evidencepack.ActorContext, rel evidencepack.ReleasePack) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE release_pack SET status=$1, approved_by=NULLIF($2,''), approved_at=$3, released_by=NULLIF($4,''), released_at=$5, updated_by=$6, updated_at=$7, revision=$8
		 WHERE tenant_id=$9 AND organization_id=$10 AND id=$11`,
		string(rel.Status), rel.ApprovedBy, rel.ApprovedAt, rel.ReleasedBy, rel.ReleasedAt,
		rel.UpdatedBy, rel.UpdatedAt, rel.Revision, actor.TenantID, actor.OrganizationID, rel.ID)
	return err
}

func (r *PackRepository) begin(ctx context.Context, actor evidencepack.ActorContext) (*sql.Tx, error) {
	return pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
}

func (r *ReleaseRepository) begin(ctx context.Context, actor evidencepack.ActorContext) (*sql.Tx, error) {
	return pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
}
