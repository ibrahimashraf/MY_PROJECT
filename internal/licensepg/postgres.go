package licensepg

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"integin/internal/domain/license"
)

var (
	ErrNilDB           = errors.New("license postgres repository requires a database")
	ErrLicenseNotFound = errors.New("license not found")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

func (r *Repository) begin(ctx context.Context, actor license.ActorContext) (*sql.Tx, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx,
		"SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)",
		actor.TenantID, actor.OrganizationID,
	); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return tx, nil
}

type txRepository struct {
	tx *sql.Tx
}

func (r *txRepository) GetLicense(ctx context.Context, actor license.ActorContext, licenseID string) (license.License, error) {
	var lic license.License
	var featuresJSON []byte
	var expiresAt sql.NullTime
	err := r.tx.QueryRowContext(ctx,
		`SELECT id, tenant_id, organization_id, tier, status, issued_at, expires_at,
		        max_inspectors, max_inspections_per_month, features, created_by, created_at
		 FROM tenant_license
		 WHERE id = $1 AND tenant_id = current_setting('integin.tenant_id', true)
		   AND organization_id = current_setting('integin.organization_id', true)`,
		licenseID,
	).Scan(&lic.ID, &lic.TenantID, &lic.OrganizationID, &lic.Tier, &lic.Status,
		&lic.IssuedAt, &expiresAt, &lic.MaxInspectors, &lic.MaxInspectionsPerMonth,
		&featuresJSON, &lic.CreatedBy, &lic.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return license.License{}, ErrLicenseNotFound
	}
	if err != nil {
		return license.License{}, err
	}
	if expiresAt.Valid {
		lic.ExpiresAt = &expiresAt.Time
	}
	if featuresJSON != nil {
		_ = json.Unmarshal(featuresJSON, &lic.Features)
	}
	return lic, nil
}

func (r *txRepository) GetActiveLicense(ctx context.Context, actor license.ActorContext) (license.License, bool, error) {
	var lic license.License
	var featuresJSON []byte
	var expiresAt sql.NullTime
	err := r.tx.QueryRowContext(ctx,
		`SELECT id, tenant_id, organization_id, tier, status, issued_at, expires_at,
		        max_inspectors, max_inspections_per_month, features, created_by, created_at
		 FROM tenant_license
		 WHERE tenant_id = current_setting('integin.tenant_id', true)
		   AND organization_id = current_setting('integin.organization_id', true)
		   AND status IN ('trial', 'active')
		 ORDER BY created_at DESC
		 LIMIT 1`,
	).Scan(&lic.ID, &lic.TenantID, &lic.OrganizationID, &lic.Tier, &lic.Status,
		&lic.IssuedAt, &expiresAt, &lic.MaxInspectors, &lic.MaxInspectionsPerMonth,
		&featuresJSON, &lic.CreatedBy, &lic.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return license.License{}, false, nil
	}
	if err != nil {
		return license.License{}, false, err
	}
	if expiresAt.Valid {
		lic.ExpiresAt = &expiresAt.Time
	}
	if featuresJSON != nil {
		_ = json.Unmarshal(featuresJSON, &lic.Features)
	}
	return lic, true, nil
}

func (r *txRepository) CreateLicense(ctx context.Context, actor license.ActorContext, lic license.License) (license.License, error) {
	featuresJSON, _ := json.Marshal(lic.Features)
	now := time.Now()
	if lic.ID == "" {
		lic.ID = lic.TenantID + ":" + lic.OrganizationID + ":" + now.Format("20060102150405")
	}
	lic.CreatedAt = now
	lic.IssuedAt = now
	if lic.Status == "" {
		lic.Status = license.StatusTrial
	}

	keyInput := lic.TenantID + ":" + lic.OrganizationID + ":" + string(lic.Tier)
	hash := sha256.Sum256([]byte(keyInput))

	_, err := r.tx.ExecContext(ctx,
		`INSERT INTO tenant_license (id, tenant_id, organization_id, license_key_hash, tier, status,
		        issued_at, expires_at, max_inspectors, max_inspections_per_month, features, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		lic.ID, lic.TenantID, lic.OrganizationID, hash[:], lic.Tier, lic.Status,
		lic.IssuedAt, lic.ExpiresAt, lic.MaxInspectors, lic.MaxInspectionsPerMonth,
		featuresJSON, lic.CreatedBy, lic.CreatedAt,
	)
	if err != nil {
		return license.License{}, err
	}
	return lic, nil
}

func (r *txRepository) UpdateLicense(ctx context.Context, actor license.ActorContext, lic license.License) (license.License, error) {
	featuresJSON, _ := json.Marshal(lic.Features)
	_, err := r.tx.ExecContext(ctx,
		`UPDATE tenant_license
		 SET tier = $3, status = $4, expires_at = $5, max_inspectors = $6,
		     max_inspections_per_month = $7, features = $8
		 WHERE id = $1 AND tenant_id = current_setting('integin.tenant_id', true)
		   AND organization_id = current_setting('integin.organization_id', true)`,
		lic.ID, lic.TenantID, lic.Tier, lic.Status, lic.ExpiresAt,
		lic.MaxInspectors, lic.MaxInspectionsPerMonth, featuresJSON,
	)
	if err != nil {
		return license.License{}, err
	}
	return lic, nil
}

func (r *txRepository) RecordAudit(ctx context.Context, actor license.ActorContext, entry license.AuditEntry) error {
	metadataJSON, _ := json.Marshal(entry.Metadata)
	_, err := r.tx.ExecContext(ctx,
		`INSERT INTO license_audit (id, tenant_id, organization_id, license_id, action, actor_id, occurred_at, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		entry.ID, entry.TenantID, entry.OrganizationID, entry.LicenseID,
		entry.Action, entry.ActorID, entry.OccurredAt, metadataJSON,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) GetLicense(ctx context.Context, actor license.ActorContext, licenseID string) (license.License, error) {
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return license.License{}, err
	}
	defer tx.Rollback()

	txRepo := &txRepository{tx: tx}
	lic, err := txRepo.GetLicense(ctx, actor, licenseID)
	if err != nil {
		return license.License{}, err
	}
	return lic, tx.Commit()
}

func (r *Repository) GetActiveLicense(ctx context.Context, actor license.ActorContext) (license.License, bool, error) {
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return license.License{}, false, err
	}
	defer tx.Rollback()

	txRepo := &txRepository{tx: tx}
	lic, ok, err := txRepo.GetActiveLicense(ctx, actor)
	if err != nil {
		return license.License{}, false, err
	}
	return lic, ok, tx.Commit()
}

func (r *Repository) CreateLicense(ctx context.Context, actor license.ActorContext, lic license.License) (license.License, error) {
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return license.License{}, err
	}
	defer tx.Rollback()

	txRepo := &txRepository{tx: tx}
	lic, err = txRepo.CreateLicense(ctx, actor, lic)
	if err != nil {
		return license.License{}, err
	}
	return lic, tx.Commit()
}

func (r *Repository) UpdateLicense(ctx context.Context, actor license.ActorContext, lic license.License) (license.License, error) {
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return license.License{}, err
	}
	defer tx.Rollback()

	txRepo := &txRepository{tx: tx}
	lic, err = txRepo.UpdateLicense(ctx, actor, lic)
	if err != nil {
		return license.License{}, err
	}
	return lic, tx.Commit()
}

func (r *Repository) RecordAudit(ctx context.Context, actor license.ActorContext, entry license.AuditEntry) error {
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	txRepo := &txRepository{tx: tx}
	if err := txRepo.RecordAudit(ctx, actor, entry); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) WithinTransaction(ctx context.Context, actor license.ActorContext, fn func(context.Context, license.Repository) error) error {
	if r == nil {
		return ErrNilDB
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	txRepo := &txRepository{tx: tx}
	if err := fn(ctx, txRepo); err != nil {
		return err
	}
	return tx.Commit()
}
