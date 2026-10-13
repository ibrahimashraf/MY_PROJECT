// Package workpackagepg persists approved INTEGIN work packages under tenant RLS.
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

var (
	ErrNotFound = errors.New("work package not found")
	ErrConflict = errors.New("work package conflicts with existing immutable version")
)

// Repository owns scoped persistence only; HTTP, device distribution, and sync handlers
// must independently authorize their callers before invoking it.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, errors.New("work package repository requires database")
	}
	return &Repository{db: db}, nil
}

// Assignment is the current device-scoped distribution record. It never rewrites
// an already captured inspection; every submitted draft remains package-bound.
// Assignment remains a compatibility alias for repository callers.
// The stable manifest contract is owned by the domain package.
type Assignment = workpackage.Assignment

func (r *Repository) SaveApproved(ctx context.Context, p workpackage.Package, approvedAt time.Time) error {
	if p.State != workpackage.PublicationApproved {
		return errors.New("only approved work packages may be persisted for field distribution")
	}
	if err := p.Validate(); err != nil {
		return fmt.Errorf("validate work package: %w", err)
	}
	if approvedAt.IsZero() {
		return errors.New("approved timestamp is required")
	}
	definition, err := json.Marshal(p.Sections)
	if err != nil {
		return fmt.Errorf("encode work package definition: %w", err)
	}
	tx, err := r.scopedTx(ctx, p.TenantID, p.OrganizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `
        INSERT INTO work_package (
            tenant_id, organization_id, package_id, package_version,
            template_code, template_version, schema_version, publication_state,
            package_hash, definition, approved_at
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
        ON CONFLICT (tenant_id, organization_id, package_id, package_version) DO NOTHING`,
		p.TenantID, p.OrganizationID, p.ID, p.PackageVersion,
		p.TemplateCode, p.TemplateVersion, p.SchemaVersion, string(p.State),
		p.PackageHash, definition, approvedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert approved work package: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check work package insert: %w", err)
	}
	if inserted != 1 {
		return ErrConflict
	}
	return tx.Commit()
}

func (r *Repository) GetApproved(ctx context.Context, tenantID, organizationID, packageID string, version int) (workpackage.Package, error) {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(organizationID) == "" || strings.TrimSpace(packageID) == "" || version <= 0 {
		return workpackage.Package{}, errors.New("tenant, organization, package id, and positive version are required")
	}
	tx, err := r.scopedTx(ctx, tenantID, organizationID)
	if err != nil {
		return workpackage.Package{}, err
	}
	defer tx.Rollback()
	var p workpackage.Package
	var state string
	var definition []byte
	err = tx.QueryRowContext(ctx, `
        SELECT package_id, tenant_id, organization_id, template_code, template_version,
               package_version, schema_version, publication_state, package_hash, definition
        FROM work_package
        WHERE tenant_id = $1 AND organization_id = $2 AND package_id = $3
              AND package_version = $4 AND publication_state = 'approved'`,
		tenantID, organizationID, packageID, version,
	).Scan(
		&p.ID, &p.TenantID, &p.OrganizationID, &p.TemplateCode, &p.TemplateVersion,
		&p.PackageVersion, &p.SchemaVersion, &state, &p.PackageHash, &definition,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return workpackage.Package{}, ErrNotFound
	}
	if err != nil {
		return workpackage.Package{}, fmt.Errorf("get approved work package: %w", err)
	}
	p.State = workpackage.PublicationState(state)
	if err := json.Unmarshal(definition, &p.Sections); err != nil {
		return workpackage.Package{}, fmt.Errorf("decode work package definition: %w", err)
	}
	if err := validateStoredPackage(p); err != nil {
		return workpackage.Package{}, err
	}
	if err := tx.Commit(); err != nil {
		return workpackage.Package{}, err
	}
	return p, nil
}

func validateStoredPackage(p workpackage.Package) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("%w: validate stored work package: %v", workpackage.ErrPackageIntegrity, err)
	}
	return nil
}

func (r *Repository) AssignApproved(ctx context.Context, assignment Assignment, now time.Time) error {
	if strings.TrimSpace(assignment.TenantID) == "" || strings.TrimSpace(assignment.OrganizationID) == "" ||
		strings.TrimSpace(assignment.InspectionID) == "" || strings.TrimSpace(assignment.DeviceID) == "" ||
		strings.TrimSpace(assignment.PackageID) == "" || assignment.PackageVersion <= 0 || assignment.AuthorityEpoch <= 0 {
		return errors.New("assignment tenant, organization, inspection, device, package, version, and authority epoch are required")
	}
	if !assignment.ExpiresAt.After(now.UTC()) {
		return errors.New("assignment expiry must be in the future")
	}
	tx, err := r.scopedTx(ctx, assignment.TenantID, assignment.OrganizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var approved bool
	err = tx.QueryRowContext(ctx, `
        SELECT EXISTS(
            SELECT 1 FROM work_package
            WHERE tenant_id = $1 AND organization_id = $2 AND package_id = $3
                  AND package_version = $4 AND publication_state = 'approved'
        )`, assignment.TenantID, assignment.OrganizationID, assignment.PackageID, assignment.PackageVersion,
	).Scan(&approved)
	if err != nil {
		return fmt.Errorf("check approved work package for assignment: %w", err)
	}
	if !approved {
		return ErrNotFound
	}
	_, err = tx.ExecContext(ctx, `
        INSERT INTO work_package_assignment (
            tenant_id, organization_id, inspection_id, device_id,
            package_id, package_version, authority_epoch, expires_at, assigned_at
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
        ON CONFLICT (tenant_id, organization_id, inspection_id) DO UPDATE
            SET device_id = EXCLUDED.device_id,
                package_id = EXCLUDED.package_id,
                package_version = EXCLUDED.package_version,
                authority_epoch = EXCLUDED.authority_epoch,
                expires_at = EXCLUDED.expires_at,
                assigned_at = EXCLUDED.assigned_at`,
		assignment.TenantID, assignment.OrganizationID, assignment.InspectionID, assignment.DeviceID,
		assignment.PackageID, assignment.PackageVersion, assignment.AuthorityEpoch,
		assignment.ExpiresAt.UTC(), now.UTC(),
	)
	if err != nil {
		return fmt.Errorf("save work package assignment: %w", err)
	}
	return tx.Commit()
}

func (r *Repository) scopedTx(ctx context.Context, tenantID, organizationID string) (*sql.Tx, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id', $1, true)`, tenantID); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("set work package tenant scope: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('integin.organization_id', $1, true)`, organizationID); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("set work package organization scope: %w", err)
	}
	return tx, nil
}
