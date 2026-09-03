package assurancepg

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"integin/internal/domain/assurance"
	"integin/internal/shared/pgtx"
)

var (
	ErrNilDB    = errors.New("assurance postgres repository requires a database")
	ErrNotFound = errors.New("assurance projection or corrective work not found")
)

// ProjectionRepository provides read-only access to assurance projections.
type ProjectionRepository struct {
	db *sql.DB
}

// NewProjectionRepository creates a projection repository.
func NewProjectionRepository(db *sql.DB) (*ProjectionRepository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &ProjectionRepository{db: db}, nil
}

func (r *ProjectionRepository) Get(ctx context.Context, actor assurance.ActorContext, certificateID string) (assurance.Projection, error) {
	if err := actor.Validate(); err != nil {
		return assurance.Projection{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return assurance.Projection{}, err
	}
	defer tx.Rollback()
	p, err := loadProjection(ctx, tx, actor, certificateID)
	if err != nil {
		return assurance.Projection{}, err
	}
	return p, tx.Commit()
}

func (r *ProjectionRepository) GetByInspection(ctx context.Context, actor assurance.ActorContext, inspectionID string) (assurance.Projection, error) {
	if err := actor.Validate(); err != nil {
		return assurance.Projection{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return assurance.Projection{}, err
	}
	defer tx.Rollback()
	var p assurance.Projection
	err = tx.QueryRowContext(ctx,
		`SELECT id, certificate_id, inspection_id, asset_id, asset_type, certificate_number, certificate_status, inspection_status, inspection_verdict, inspector_id, issued_at, expires_at, revoked_at, superseded_at, projection_hash, projected_at, created_at
		 FROM assurance_projection WHERE tenant_id=$1 AND organization_id=$2 AND inspection_id=$3`,
		actor.TenantID, actor.OrganizationID, inspectionID).Scan(
		&p.ID, &p.CertificateID, &p.InspectionID, &p.AssetID, &p.AssetType,
		&p.CertificateNumber, &p.CertificateStatus, &p.InspectionStatus, &p.InspectionVerdict,
		&p.InspectorID, &p.IssuedAt, &p.ExpiresAt, &p.RevokedAt, &p.SupersededAt,
		&p.ProjectionHash, &p.ProjectedAt, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return assurance.Projection{}, assurance.ErrNotFound
	}
	if err != nil {
		return assurance.Projection{}, err
	}
	p.TenantID = actor.TenantID
	p.OrganizationID = actor.OrganizationID
	return p, tx.Commit()
}

func (r *ProjectionRepository) ListByAsset(ctx context.Context, actor assurance.ActorContext, assetID string) ([]assurance.Projection, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx,
		`SELECT id, certificate_id, inspection_id, asset_id, asset_type, certificate_number, certificate_status, inspection_status, inspection_verdict, inspector_id, issued_at, expires_at, revoked_at, superseded_at, projection_hash, projected_at, created_at
		 FROM assurance_projection WHERE tenant_id=$1 AND organization_id=$2 AND asset_id=$3
		 ORDER BY certificate_status, projected_at DESC`,
		actor.TenantID, actor.OrganizationID, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var projections []assurance.Projection
	for rows.Next() {
		var p assurance.Projection
		if err := rows.Scan(
			&p.ID, &p.CertificateID, &p.InspectionID, &p.AssetID, &p.AssetType,
			&p.CertificateNumber, &p.CertificateStatus, &p.InspectionStatus, &p.InspectionVerdict,
			&p.InspectorID, &p.IssuedAt, &p.ExpiresAt, &p.RevokedAt, &p.SupersededAt,
			&p.ProjectionHash, &p.ProjectedAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.TenantID = actor.TenantID
		p.OrganizationID = actor.OrganizationID
		projections = append(projections, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return projections, tx.Commit()
}

func (r *ProjectionRepository) ListByStatus(ctx context.Context, actor assurance.ActorContext, status string) ([]assurance.Projection, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx,
		`SELECT id, certificate_id, inspection_id, asset_id, asset_type, certificate_number, certificate_status, inspection_status, inspection_verdict, inspector_id, issued_at, expires_at, revoked_at, superseded_at, projection_hash, projected_at, created_at
		 FROM assurance_projection WHERE tenant_id=$1 AND organization_id=$2 AND certificate_status=$3
		 ORDER BY projected_at DESC`,
		actor.TenantID, actor.OrganizationID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var projections []assurance.Projection
	for rows.Next() {
		var p assurance.Projection
		if err := rows.Scan(
			&p.ID, &p.CertificateID, &p.InspectionID, &p.AssetID, &p.AssetType,
			&p.CertificateNumber, &p.CertificateStatus, &p.InspectionStatus, &p.InspectionVerdict,
			&p.InspectorID, &p.IssuedAt, &p.ExpiresAt, &p.RevokedAt, &p.SupersededAt,
			&p.ProjectionHash, &p.ProjectedAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.TenantID = actor.TenantID
		p.OrganizationID = actor.OrganizationID
		projections = append(projections, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return projections, tx.Commit()
}

// WorkRepository provides persistence for corrective work.
type WorkRepository struct {
	db *sql.DB
}

// NewWorkRepository creates a corrective work repository.
func NewWorkRepository(db *sql.DB) (*WorkRepository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &WorkRepository{db: db}, nil
}

func (r *WorkRepository) Create(ctx context.Context, actor assurance.ActorContext, work assurance.CorrectiveWork) (assurance.CorrectiveWork, error) {
	if err := actor.Validate(); err != nil {
		return assurance.CorrectiveWork{}, err
	}
	work.TenantID = actor.TenantID
	work.OrganizationID = actor.OrganizationID
	work.Status = assurance.WorkStatusOpen
	work.CreatedBy = actor.ActorID
	work.UpdatedBy = actor.ActorID
	if work.Revision <= 0 {
		work.Revision = 1
	}
	if err := work.Validate(); err != nil {
		return assurance.CorrectiveWork{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return assurance.CorrectiveWork{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx,
		`INSERT INTO corrective_work (id, tenant_id, organization_id, inspection_id, finding_id, asset_id, severity, description, required_by, assigned_to, status, created_by, created_at, updated_by, updated_at, revision)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,''),$11,$12,$13,$14,$15,$16)`,
		work.ID, actor.TenantID, actor.OrganizationID, work.InspectionID, work.FindingID,
		work.AssetID, string(work.Severity), work.Description, work.RequiredBy, work.AssignedTo,
		string(work.Status), work.CreatedBy, work.CreatedAt, work.UpdatedBy, work.UpdatedAt, work.Revision)
	if err != nil {
		return assurance.CorrectiveWork{}, err
	}
	stored, err := loadWork(ctx, tx, actor, work.ID)
	if err != nil {
		return assurance.CorrectiveWork{}, err
	}
	return stored, tx.Commit()
}

func (r *WorkRepository) Get(ctx context.Context, actor assurance.ActorContext, workID string) (assurance.CorrectiveWork, error) {
	if err := actor.Validate(); err != nil {
		return assurance.CorrectiveWork{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return assurance.CorrectiveWork{}, err
	}
	defer tx.Rollback()
	work, err := loadWork(ctx, tx, actor, workID)
	if err != nil {
		return assurance.CorrectiveWork{}, err
	}
	return work, tx.Commit()
}

func (r *WorkRepository) ListByInspection(ctx context.Context, actor assurance.ActorContext, inspectionID string) ([]assurance.CorrectiveWork, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx,
		`SELECT id, inspection_id, finding_id, asset_id, severity, description, required_by, COALESCE(assigned_to,''), status, completed_at, verified_at, COALESCE(verified_by,''), created_by, created_at, updated_by, updated_at, revision
		 FROM corrective_work WHERE tenant_id=$1 AND organization_id=$2 AND inspection_id=$3
		 ORDER BY severity DESC, created_at ASC`,
		actor.TenantID, actor.OrganizationID, inspectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var works []assurance.CorrectiveWork
	for rows.Next() {
		var w assurance.CorrectiveWork
		if err := rows.Scan(
			&w.ID, &w.InspectionID, &w.FindingID, &w.AssetID, &w.Severity, &w.Description,
			&w.RequiredBy, &w.AssignedTo, &w.Status, &w.CompletedAt, &w.VerifiedAt,
			&w.VerifiedBy, &w.CreatedBy, &w.CreatedAt, &w.UpdatedBy, &w.UpdatedAt, &w.Revision); err != nil {
			return nil, err
		}
		w.TenantID = actor.TenantID
		w.OrganizationID = actor.OrganizationID
		works = append(works, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return works, tx.Commit()
}

func (r *WorkRepository) ListByAsset(ctx context.Context, actor assurance.ActorContext, assetID string) ([]assurance.CorrectiveWork, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx,
		`SELECT id, inspection_id, finding_id, asset_id, severity, description, required_by, COALESCE(assigned_to,''), status, completed_at, verified_at, COALESCE(verified_by,''), created_by, created_at, updated_by, updated_at, revision
		 FROM corrective_work WHERE tenant_id=$1 AND organization_id=$2 AND asset_id=$3
		 ORDER BY severity DESC, created_at ASC`,
		actor.TenantID, actor.OrganizationID, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var works []assurance.CorrectiveWork
	for rows.Next() {
		var w assurance.CorrectiveWork
		if err := rows.Scan(
			&w.ID, &w.InspectionID, &w.FindingID, &w.AssetID, &w.Severity, &w.Description,
			&w.RequiredBy, &w.AssignedTo, &w.Status, &w.CompletedAt, &w.VerifiedAt,
			&w.VerifiedBy, &w.CreatedBy, &w.CreatedAt, &w.UpdatedBy, &w.UpdatedAt, &w.Revision); err != nil {
			return nil, err
		}
		w.TenantID = actor.TenantID
		w.OrganizationID = actor.OrganizationID
		works = append(works, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return works, tx.Commit()
}

func (r *WorkRepository) UpdateStatus(ctx context.Context, actor assurance.ActorContext, workID string, target assurance.WorkStatus, expectedRevision int64) (assurance.CorrectiveWork, error) {
	if err := actor.Validate(); err != nil {
		return assurance.CorrectiveWork{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return assurance.CorrectiveWork{}, err
	}
	defer tx.Rollback()
	current, err := loadWork(ctx, tx, actor, workID)
	if errors.Is(err, assurance.ErrNotFound) {
		return assurance.CorrectiveWork{}, assurance.ErrNotFound
	}
	if err != nil {
		return assurance.CorrectiveWork{}, err
	}
	if current.Revision != expectedRevision {
		return assurance.CorrectiveWork{}, assurance.ErrStaleRevision
	}
	if err := current.CanTransitionTo(target); err != nil {
		return assurance.CorrectiveWork{}, err
	}
	current.Status = target
	current.Revision++
	current.UpdatedBy = actor.ActorID
	current.UpdatedAt = now()
	switch target {
	case assurance.WorkStatusCompleted:
		current.CompletedAt = timePtr(now())
	case assurance.WorkStatusVerified:
		current.VerifiedAt = timePtr(now())
		current.VerifiedBy = actor.ActorID
	case assurance.WorkStatusInProgress:
		current.CompletedAt = nil
	case assurance.WorkStatusOpen:
		current.CompletedAt = nil
		current.VerifiedAt = nil
		current.VerifiedBy = ""
	}
	if err := current.Validate(); err != nil {
		return assurance.CorrectiveWork{}, err
	}
	if err := saveWork(ctx, tx, actor, current); err != nil {
		return assurance.CorrectiveWork{}, err
	}
	stored, err := loadWork(ctx, tx, actor, workID)
	if err != nil {
		return assurance.CorrectiveWork{}, err
	}
	return stored, tx.Commit()
}

func (r *WorkRepository) Assign(ctx context.Context, actor assurance.ActorContext, workID string, inspectorID string, expectedRevision int64) (assurance.CorrectiveWork, error) {
	if err := actor.Validate(); err != nil {
		return assurance.CorrectiveWork{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return assurance.CorrectiveWork{}, err
	}
	defer tx.Rollback()
	current, err := loadWork(ctx, tx, actor, workID)
	if errors.Is(err, assurance.ErrNotFound) {
		return assurance.CorrectiveWork{}, assurance.ErrNotFound
	}
	if err != nil {
		return assurance.CorrectiveWork{}, err
	}
	if current.Revision != expectedRevision {
		return assurance.CorrectiveWork{}, assurance.ErrStaleRevision
	}
	current.AssignedTo = inspectorID
	current.Revision++
	current.UpdatedBy = actor.ActorID
	current.UpdatedAt = now()
	if err := current.Validate(); err != nil {
		return assurance.CorrectiveWork{}, err
	}
	if err := saveWork(ctx, tx, actor, current); err != nil {
		return assurance.CorrectiveWork{}, err
	}
	stored, err := loadWork(ctx, tx, actor, workID)
	if err != nil {
		return assurance.CorrectiveWork{}, err
	}
	return stored, tx.Commit()
}

func loadProjection(ctx context.Context, tx *sql.Tx, actor assurance.ActorContext, certificateID string) (assurance.Projection, error) {
	var p assurance.Projection
	err := tx.QueryRowContext(ctx,
		`SELECT id, certificate_id, inspection_id, asset_id, asset_type, certificate_number, certificate_status, inspection_status, inspection_verdict, inspector_id, issued_at, expires_at, revoked_at, superseded_at, projection_hash, projected_at, created_at
		 FROM assurance_projection WHERE tenant_id=$1 AND organization_id=$2 AND certificate_id=$3`,
		actor.TenantID, actor.OrganizationID, certificateID).Scan(
		&p.ID, &p.CertificateID, &p.InspectionID, &p.AssetID, &p.AssetType,
		&p.CertificateNumber, &p.CertificateStatus, &p.InspectionStatus, &p.InspectionVerdict,
		&p.InspectorID, &p.IssuedAt, &p.ExpiresAt, &p.RevokedAt, &p.SupersededAt,
		&p.ProjectionHash, &p.ProjectedAt, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return assurance.Projection{}, assurance.ErrNotFound
	}
	if err != nil {
		return assurance.Projection{}, err
	}
	p.TenantID = actor.TenantID
	p.OrganizationID = actor.OrganizationID
	return p, nil
}

func loadWork(ctx context.Context, tx *sql.Tx, actor assurance.ActorContext, workID string) (assurance.CorrectiveWork, error) {
	var w assurance.CorrectiveWork
	err := tx.QueryRowContext(ctx,
		`SELECT id, inspection_id, finding_id, asset_id, severity, description, required_by, COALESCE(assigned_to,''), status, completed_at, verified_at, COALESCE(verified_by,''), created_by, created_at, updated_by, updated_at, revision
		 FROM corrective_work WHERE tenant_id=$1 AND organization_id=$2 AND id=$3`,
		actor.TenantID, actor.OrganizationID, workID).Scan(
		&w.ID, &w.InspectionID, &w.FindingID, &w.AssetID, &w.Severity, &w.Description,
		&w.RequiredBy, &w.AssignedTo, &w.Status, &w.CompletedAt, &w.VerifiedAt,
		&w.VerifiedBy, &w.CreatedBy, &w.CreatedAt, &w.UpdatedBy, &w.UpdatedAt, &w.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return assurance.CorrectiveWork{}, assurance.ErrNotFound
	}
	if err != nil {
		return assurance.CorrectiveWork{}, err
	}
	w.TenantID = actor.TenantID
	w.OrganizationID = actor.OrganizationID
	return w, nil
}

func saveWork(ctx context.Context, tx *sql.Tx, actor assurance.ActorContext, w assurance.CorrectiveWork) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE corrective_work SET severity=$1, description=$2, required_by=$3, assigned_to=NULLIF($4,''), status=$5, completed_at=$6, verified_at=$7, verified_by=NULLIF($8,''), updated_by=$9, updated_at=$10, revision=$11
		 WHERE tenant_id=$12 AND organization_id=$13 AND id=$14`,
		string(w.Severity), w.Description, w.RequiredBy, w.AssignedTo, string(w.Status),
		w.CompletedAt, w.VerifiedAt, w.VerifiedBy, w.UpdatedBy, w.UpdatedAt, w.Revision,
		actor.TenantID, actor.OrganizationID, w.ID)
	return err
}

func (r *ProjectionRepository) begin(ctx context.Context, actor assurance.ActorContext) (*sql.Tx, error) {
	return pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
}

func (r *WorkRepository) begin(ctx context.Context, actor assurance.ActorContext) (*sql.Tx, error) {
	return pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
}

func timePtr(t time.Time) *time.Time { return &t }

func now() time.Time { return time.Now().UTC() }
