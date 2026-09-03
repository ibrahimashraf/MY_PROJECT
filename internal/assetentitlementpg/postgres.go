package assetentitlementpg

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"integin/internal/domain/assetentitlement"
	"integin/internal/shared/pgtx"
)

var (
	ErrNilDB         = errors.New("asset entitlement postgres repository requires a database")
	ErrNotFound      = errors.New("asset entitlement not found")
	ErrPackageNotFound = errors.New("offline package not found")
	ErrNotActive     = errors.New("asset entitlement is not active")
	ErrStaleRevision = errors.New("stale revision")
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

func (r *Repository) Register(ctx context.Context, actor assetentitlement.ActorContext, ent assetentitlement.AssetEntitlement) (assetentitlement.AssetEntitlement, error) {
	if err := actor.Validate(); err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	ent.Status = assetentitlement.EntitlementStatusActive
	ent.TenantID = actor.TenantID
	ent.OrganizationID = actor.OrganizationID
	ent.CreatedBy = actor.ActorID
	ent.CreatedAt = time.Now().UTC()
	ent.UpdatedBy = actor.ActorID
	ent.UpdatedAt = ent.CreatedAt
	ent.Revision = 1
	if ent.EntitledAt.IsZero() {
		ent.EntitledAt = ent.CreatedAt
	}
	if err := ent.Validate(); err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO asset_entitlement (id, tenant_id, organization_id, work_order_id, scope_item_id, asset_id, asset_type, entitlement_type, status, form_version_id, assigned_inspector_id, entitled_at, expires_at, created_by, created_at, updated_by, updated_at, revision)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,''),NULLIF($11,''),$12,$13,$14,$15,$16,$17,$18)`,
		ent.ID, actor.TenantID, actor.OrganizationID, ent.WorkOrderID, ent.ScopeItemID,
		ent.AssetID, ent.AssetType, string(ent.EntitlementType), string(ent.Status),
		ent.FormVersionID, ent.AssignedInspectorID,
		ent.EntitledAt, ent.ExpiresAt,
		ent.CreatedBy, ent.CreatedAt, ent.UpdatedBy, ent.UpdatedAt, ent.Revision); err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	for _, tag := range ent.Tags {
		if err := insertTag(ctx, tx, actor, ent.AssetID, tag); err != nil {
			return assetentitlement.AssetEntitlement{}, err
		}
	}
	stored, err := loadEntitlement(ctx, tx, actor, ent.ID)
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	return stored, tx.Commit()
}

func (r *Repository) Get(ctx context.Context, actor assetentitlement.ActorContext, entitlementID string) (assetentitlement.AssetEntitlement, error) {
	if err := actor.Validate(); err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	defer tx.Rollback()
	stored, err := loadEntitlement(ctx, tx, actor, entitlementID)
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	return stored, tx.Commit()
}

func (r *Repository) GetByWorkOrder(ctx context.Context, actor assetentitlement.ActorContext, workOrderID string) ([]assetentitlement.AssetEntitlement, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx,
		`SELECT id, work_order_id, scope_item_id, asset_id, asset_type, entitlement_type, status,
		        COALESCE(form_version_id,''), COALESCE(assigned_inspector_id,''),
		        entitled_at, expires_at, completed_at, created_by, created_at, updated_by, updated_at, revision
		 FROM asset_entitlement WHERE tenant_id=$1 AND organization_id=$2 AND work_order_id=$3
		 ORDER BY created_at, id`,
		actor.TenantID, actor.OrganizationID, workOrderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entitlements []assetentitlement.AssetEntitlement
	for rows.Next() {
		var ent assetentitlement.AssetEntitlement
		if err := rows.Scan(
			&ent.ID, &ent.WorkOrderID, &ent.ScopeItemID, &ent.AssetID, &ent.AssetType,
			&ent.EntitlementType, &ent.Status, &ent.FormVersionID, &ent.AssignedInspectorID,
			&ent.EntitledAt, &ent.ExpiresAt, &ent.CompletedAt,
			&ent.CreatedBy, &ent.CreatedAt, &ent.UpdatedBy, &ent.UpdatedAt, &ent.Revision); err != nil {
			return nil, err
		}
		ent.TenantID = actor.TenantID
		ent.OrganizationID = actor.OrganizationID
		entitlements = append(entitlements, ent)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entitlements, tx.Commit()
}

func (r *Repository) GetByAsset(ctx context.Context, actor assetentitlement.ActorContext, assetID string) ([]assetentitlement.AssetEntitlement, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx,
		`SELECT id, work_order_id, scope_item_id, asset_id, asset_type, entitlement_type, status,
		        COALESCE(form_version_id,''), COALESCE(assigned_inspector_id,''),
		        entitled_at, expires_at, completed_at, created_by, created_at, updated_by, updated_at, revision
		 FROM asset_entitlement WHERE tenant_id=$1 AND organization_id=$2 AND asset_id=$3
		 ORDER BY created_at DESC, id`,
		actor.TenantID, actor.OrganizationID, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entitlements []assetentitlement.AssetEntitlement
	for rows.Next() {
		var ent assetentitlement.AssetEntitlement
		if err := rows.Scan(
			&ent.ID, &ent.WorkOrderID, &ent.ScopeItemID, &ent.AssetID, &ent.AssetType,
			&ent.EntitlementType, &ent.Status, &ent.FormVersionID, &ent.AssignedInspectorID,
			&ent.EntitledAt, &ent.ExpiresAt, &ent.CompletedAt,
			&ent.CreatedBy, &ent.CreatedAt, &ent.UpdatedBy, &ent.UpdatedAt, &ent.Revision); err != nil {
			return nil, err
		}
		ent.TenantID = actor.TenantID
		ent.OrganizationID = actor.OrganizationID
		entitlements = append(entitlements, ent)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entitlements, tx.Commit()
}

func (r *Repository) Complete(ctx context.Context, actor assetentitlement.ActorContext, entitlementID string, expectedRevision int64) (assetentitlement.AssetEntitlement, error) {
	if err := actor.Validate(); err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx,
		`UPDATE asset_entitlement SET status = 'COMPLETED', completed_at = $1, updated_by = $2, updated_at = $3, revision = revision + 1
		 WHERE tenant_id = $4 AND organization_id = $5 AND id = $6 AND revision = $7 AND status = 'ACTIVE'`,
		now, actor.ActorID, now, actor.TenantID, actor.OrganizationID, entitlementID, expectedRevision)
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	if count == 0 {
		ent, err := loadEntitlement(ctx, tx, actor, entitlementID)
		if errors.Is(err, ErrNotFound) {
			return assetentitlement.AssetEntitlement{}, ErrNotFound
		}
		if err != nil {
			return assetentitlement.AssetEntitlement{}, err
		}
		if ent.Revision != expectedRevision {
			return assetentitlement.AssetEntitlement{}, ErrStaleRevision
		}
		return assetentitlement.AssetEntitlement{}, ErrNotActive
	}
	stored, err := loadEntitlement(ctx, tx, actor, entitlementID)
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	return stored, tx.Commit()
}

func (r *Repository) Cancel(ctx context.Context, actor assetentitlement.ActorContext, entitlementID string, expectedRevision int64) (assetentitlement.AssetEntitlement, error) {
	if err := actor.Validate(); err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx,
		`UPDATE asset_entitlement SET status = 'CANCELLED', updated_by = $1, updated_at = $2, revision = revision + 1
		 WHERE tenant_id = $3 AND organization_id = $4 AND id = $5 AND revision = $6 AND status = 'ACTIVE'`,
		actor.ActorID, now, actor.TenantID, actor.OrganizationID, entitlementID, expectedRevision)
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	if count == 0 {
		ent, err := loadEntitlement(ctx, tx, actor, entitlementID)
		if errors.Is(err, ErrNotFound) {
			return assetentitlement.AssetEntitlement{}, ErrNotFound
		}
		if err != nil {
			return assetentitlement.AssetEntitlement{}, err
		}
		if ent.Revision != expectedRevision {
			return assetentitlement.AssetEntitlement{}, ErrStaleRevision
		}
		return assetentitlement.AssetEntitlement{}, ErrNotActive
	}
	stored, err := loadEntitlement(ctx, tx, actor, entitlementID)
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	return stored, tx.Commit()
}

func (r *Repository) AssignInspector(ctx context.Context, actor assetentitlement.ActorContext, entitlementID string, inspectorID string, expectedRevision int64) (assetentitlement.AssetEntitlement, error) {
	if err := actor.Validate(); err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx,
		`UPDATE asset_entitlement SET assigned_inspector_id = $1, updated_by = $2, updated_at = $3, revision = revision + 1
		 WHERE tenant_id = $4 AND organization_id = $5 AND id = $6 AND revision = $7 AND status = 'ACTIVE'`,
		inspectorID, actor.ActorID, now, actor.TenantID, actor.OrganizationID, entitlementID, expectedRevision)
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	if count == 0 {
		ent, err := loadEntitlement(ctx, tx, actor, entitlementID)
		if errors.Is(err, ErrNotFound) {
			return assetentitlement.AssetEntitlement{}, ErrNotFound
		}
		if err != nil {
			return assetentitlement.AssetEntitlement{}, err
		}
		if ent.Revision != expectedRevision {
			return assetentitlement.AssetEntitlement{}, ErrStaleRevision
		}
		return assetentitlement.AssetEntitlement{}, ErrNotActive
	}
	stored, err := loadEntitlement(ctx, tx, actor, entitlementID)
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	return stored, tx.Commit()
}

func (r *Repository) AddTag(ctx context.Context, actor assetentitlement.ActorContext, tag assetentitlement.AssetTag) (assetentitlement.AssetTag, error) {
	if err := actor.Validate(); err != nil {
		return assetentitlement.AssetTag{}, err
	}
	tag.TenantID = actor.TenantID
	tag.OrganizationID = actor.OrganizationID
	if tag.AssignedAt.IsZero() {
		tag.AssignedAt = time.Now().UTC()
	}
	tag.AssignedBy = actor.ActorID
	if tag.TagDigest == "" {
		tag.TagDigest = assetentitlement.ComputeTagDigest(tag.TagType, tag.TagValue)
	}
	if err := tag.Validate(); err != nil {
		return assetentitlement.AssetTag{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return assetentitlement.AssetTag{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO asset_tag (tenant_id, organization_id, asset_id, tag_type, tag_value, tag_digest, assigned_at, assigned_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 ON CONFLICT (tenant_id, organization_id, asset_id, tag_type, tag_value) DO UPDATE SET tag_digest = $6, assigned_at = $7, assigned_by = $8`,
		actor.TenantID, actor.OrganizationID, tag.AssetID, string(tag.TagType), tag.TagValue, tag.TagDigest, tag.AssignedAt, tag.AssignedBy); err != nil {
		return assetentitlement.AssetTag{}, err
	}
	return tag, tx.Commit()
}

func (r *Repository) ListTags(ctx context.Context, actor assetentitlement.ActorContext, assetID string) ([]assetentitlement.AssetTag, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx,
		`SELECT asset_id, tag_type, tag_value, tag_digest, assigned_at, assigned_by
		 FROM asset_tag WHERE tenant_id=$1 AND organization_id=$2 AND asset_id=$3
		 ORDER BY tag_type, tag_value`,
		actor.TenantID, actor.OrganizationID, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []assetentitlement.AssetTag
	for rows.Next() {
		var tag assetentitlement.AssetTag
		if err := rows.Scan(&tag.AssetID, &tag.TagType, &tag.TagValue, &tag.TagDigest, &tag.AssignedAt, &tag.AssignedBy); err != nil {
			return nil, err
		}
		tag.TenantID = actor.TenantID
		tag.OrganizationID = actor.OrganizationID
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, tx.Commit()
}

func (r *Repository) RemoveTag(ctx context.Context, actor assetentitlement.ActorContext, assetID string, tagType assetentitlement.TagType, tagValue string) error {
	if err := actor.Validate(); err != nil {
		return err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx,
		`DELETE FROM asset_tag WHERE tenant_id=$1 AND organization_id=$2 AND asset_id=$3 AND tag_type=$4 AND tag_value=$5`,
		actor.TenantID, actor.OrganizationID, assetID, string(tagType), tagValue)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("tag not found")
	}
	return tx.Commit()
}

func loadEntitlement(ctx context.Context, tx *sql.Tx, actor assetentitlement.ActorContext, entitlementID string) (assetentitlement.AssetEntitlement, error) {
	var ent assetentitlement.AssetEntitlement
	err := tx.QueryRowContext(ctx,
		`SELECT id, work_order_id, scope_item_id, asset_id, asset_type, entitlement_type, status,
		        COALESCE(form_version_id,''), COALESCE(assigned_inspector_id,''),
		        entitled_at, expires_at, completed_at, created_by, created_at, updated_by, updated_at, revision
		 FROM asset_entitlement WHERE tenant_id=$1 AND organization_id=$2 AND id=$3`,
		actor.TenantID, actor.OrganizationID, entitlementID).Scan(
		&ent.ID, &ent.WorkOrderID, &ent.ScopeItemID, &ent.AssetID, &ent.AssetType,
		&ent.EntitlementType, &ent.Status, &ent.FormVersionID, &ent.AssignedInspectorID,
		&ent.EntitledAt, &ent.ExpiresAt, &ent.CompletedAt,
		&ent.CreatedBy, &ent.CreatedAt, &ent.UpdatedBy, &ent.UpdatedAt, &ent.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return assetentitlement.AssetEntitlement{}, ErrNotFound
	}
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	ent.TenantID = actor.TenantID
	ent.OrganizationID = actor.OrganizationID
	tags, err := loadTags(ctx, tx, actor, ent.AssetID)
	if err != nil {
		return assetentitlement.AssetEntitlement{}, err
	}
	ent.Tags = tags
	return ent, nil
}

func loadTags(ctx context.Context, tx *sql.Tx, actor assetentitlement.ActorContext, assetID string) ([]assetentitlement.AssetTag, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT asset_id, tag_type, tag_value, tag_digest, assigned_at, assigned_by
		 FROM asset_tag WHERE tenant_id=$1 AND organization_id=$2 AND asset_id=$3
		 ORDER BY tag_type, tag_value`,
		actor.TenantID, actor.OrganizationID, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []assetentitlement.AssetTag
	for rows.Next() {
		var tag assetentitlement.AssetTag
		if err := rows.Scan(&tag.AssetID, &tag.TagType, &tag.TagValue, &tag.TagDigest, &tag.AssignedAt, &tag.AssignedBy); err != nil {
			return nil, err
		}
		tag.TenantID = actor.TenantID
		tag.OrganizationID = actor.OrganizationID
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

func insertTag(ctx context.Context, tx *sql.Tx, actor assetentitlement.ActorContext, assetID string, tag assetentitlement.AssetTag) error {
	if tag.TagDigest == "" {
		tag.TagDigest = assetentitlement.ComputeTagDigest(tag.TagType, tag.TagValue)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO asset_tag (tenant_id, organization_id, asset_id, tag_type, tag_value, tag_digest, assigned_at, assigned_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 ON CONFLICT (tenant_id, organization_id, asset_id, tag_type, tag_value) DO UPDATE SET tag_digest = $6, assigned_at = $7, assigned_by = $8`,
		actor.TenantID, actor.OrganizationID, assetID, string(tag.TagType), tag.TagValue, tag.TagDigest, tag.AssignedAt, tag.AssignedBy); err != nil {
		return err
	}
	return nil
}

func (r *Repository) begin(ctx context.Context, actor assetentitlement.ActorContext) (*sql.Tx, error) {
	return pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
}

// PackageRepository implements offline package persistence.
type PackageRepository struct {
	db *sql.DB
}

func NewPackageRepository(db *sql.DB) (*PackageRepository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &PackageRepository{db: db}, nil
}

func (r *PackageRepository) Generate(ctx context.Context, actor assetentitlement.ActorContext, pkg assetentitlement.OfflinePackage) (assetentitlement.OfflinePackage, error) {
	if err := actor.Validate(); err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	pkg.TenantID = actor.TenantID
	pkg.OrganizationID = actor.OrganizationID
	pkg.Status = assetentitlement.PackageStatusPending
	pkg.CreatedBy = actor.ActorID
	pkg.CreatedAt = time.Now().UTC()
	if err := pkg.Validate(); err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO offline_package (id, tenant_id, organization_id, entitlement_id, package_version, form_snapshot, evidence_policy_snapshot, asset_context, package_hash, status, created_by, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'GENERATED',$10,$11)`,
		pkg.ID, actor.TenantID, actor.OrganizationID, pkg.EntitlementID, pkg.PackageVersion,
		pkg.FormSnapshot, pkg.EvidencePolicySnapshot, pkg.AssetContext, pkg.PackageHash,
		actor.ActorID, now); err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	pkg.Status = assetentitlement.PackageStatusGenerated
	pkg.GeneratedAt = &now
	return pkg, tx.Commit()
}

func (r *PackageRepository) Get(ctx context.Context, actor assetentitlement.ActorContext, packageID string) (assetentitlement.OfflinePackage, error) {
	if err := actor.Validate(); err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	defer tx.Rollback()
	pkg, err := loadPackage(ctx, tx, actor, packageID)
	if err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	return pkg, tx.Commit()
}

func (r *PackageRepository) GetByEntitlement(ctx context.Context, actor assetentitlement.ActorContext, entitlementID string) ([]assetentitlement.OfflinePackage, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx,
		`SELECT id, entitlement_id, package_version, form_snapshot, evidence_policy_snapshot, asset_context, package_hash, status, generated_at, delivered_at, COALESCE(delivered_to_device_id,''), COALESCE(delivered_to_inspector_id,''), expires_at, revoked_at, COALESCE(revoked_by,''), created_by, created_at
		 FROM offline_package WHERE tenant_id=$1 AND organization_id=$2 AND entitlement_id=$3
		 ORDER BY package_version DESC`,
		actor.TenantID, actor.OrganizationID, entitlementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var packages []assetentitlement.OfflinePackage
	for rows.Next() {
		var pkg assetentitlement.OfflinePackage
		if err := rows.Scan(
			&pkg.ID, &pkg.EntitlementID, &pkg.PackageVersion, &pkg.FormSnapshot, &pkg.EvidencePolicySnapshot, &pkg.AssetContext,
			&pkg.PackageHash, &pkg.Status, &pkg.GeneratedAt, &pkg.DeliveredAt, &pkg.DeliveredToDeviceID, &pkg.DeliveredToInspectorID,
			&pkg.ExpiresAt, &pkg.RevokedAt, &pkg.RevokedBy, &pkg.CreatedBy, &pkg.CreatedAt); err != nil {
			return nil, err
		}
		pkg.TenantID = actor.TenantID
		pkg.OrganizationID = actor.OrganizationID
		packages = append(packages, pkg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return packages, tx.Commit()
}

func (r *PackageRepository) Deliver(ctx context.Context, actor assetentitlement.ActorContext, packageID string, deviceID string, inspectorID string) (assetentitlement.OfflinePackage, error) {
	if err := actor.Validate(); err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx,
		`UPDATE offline_package SET status = 'DELIVERED', delivered_at = $1, delivered_to_device_id = $2, delivered_to_inspector_id = $3
		 WHERE tenant_id = $4 AND organization_id = $5 AND id = $6 AND status = 'GENERATED'`,
		now, deviceID, inspectorID, actor.TenantID, actor.OrganizationID, packageID)
	if err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	if count == 0 {
		pkg, err := loadPackage(ctx, tx, actor, packageID)
		if errors.Is(err, ErrPackageNotFound) {
			return assetentitlement.OfflinePackage{}, ErrPackageNotFound
		}
		if err != nil {
			return assetentitlement.OfflinePackage{}, err
		}
		return assetentitlement.OfflinePackage{}, fmt.Errorf("package is not in GENERATED status (current: %s)", pkg.Status)
	}
	pkg, err := loadPackage(ctx, tx, actor, packageID)
	if err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	return pkg, tx.Commit()
}

func (r *PackageRepository) Revoke(ctx context.Context, actor assetentitlement.ActorContext, packageID string, reason string) (assetentitlement.OfflinePackage, error) {
	if err := actor.Validate(); err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx,
		`UPDATE offline_package SET status = 'REVOKED', revoked_at = $1, revoked_by = $2
		 WHERE tenant_id = $3 AND organization_id = $4 AND id = $5 AND status NOT IN ('EXPIRED', 'REVOKED')`,
		now, actor.ActorID, actor.TenantID, actor.OrganizationID, packageID)
	if err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	if count == 0 {
		pkg, err := loadPackage(ctx, tx, actor, packageID)
		if errors.Is(err, ErrPackageNotFound) {
			return assetentitlement.OfflinePackage{}, ErrPackageNotFound
		}
		if err != nil {
			return assetentitlement.OfflinePackage{}, err
		}
		return assetentitlement.OfflinePackage{}, fmt.Errorf("package is terminal (current: %s)", pkg.Status)
	}
	pkg, err := loadPackage(ctx, tx, actor, packageID)
	if err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	return pkg, tx.Commit()
}

func (r *PackageRepository) ListByDevice(ctx context.Context, actor assetentitlement.ActorContext, deviceID string) ([]assetentitlement.OfflinePackage, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx,
		`SELECT id, entitlement_id, package_version, form_snapshot, evidence_policy_snapshot, asset_context, package_hash, status, generated_at, delivered_at, delivered_to_device_id, delivered_to_inspector_id, expires_at, revoked_at, COALESCE(revoked_by,''), created_by, created_at
		 FROM offline_package WHERE tenant_id=$1 AND organization_id=$2 AND delivered_to_device_id=$3
		 ORDER BY delivered_at DESC`,
		actor.TenantID, actor.OrganizationID, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var packages []assetentitlement.OfflinePackage
	for rows.Next() {
		var pkg assetentitlement.OfflinePackage
		if err := rows.Scan(
			&pkg.ID, &pkg.EntitlementID, &pkg.PackageVersion, &pkg.FormSnapshot, &pkg.EvidencePolicySnapshot, &pkg.AssetContext,
			&pkg.PackageHash, &pkg.Status, &pkg.GeneratedAt, &pkg.DeliveredAt, &pkg.DeliveredToDeviceID, &pkg.DeliveredToInspectorID,
			&pkg.ExpiresAt, &pkg.RevokedAt, &pkg.RevokedBy, &pkg.CreatedBy, &pkg.CreatedAt); err != nil {
			return nil, err
		}
		pkg.TenantID = actor.TenantID
		pkg.OrganizationID = actor.OrganizationID
		packages = append(packages, pkg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return packages, tx.Commit()
}

func loadPackage(ctx context.Context, tx *sql.Tx, actor assetentitlement.ActorContext, packageID string) (assetentitlement.OfflinePackage, error) {
	var pkg assetentitlement.OfflinePackage
	err := tx.QueryRowContext(ctx,
		`SELECT id, entitlement_id, package_version, form_snapshot, evidence_policy_snapshot, asset_context, package_hash, status, generated_at, delivered_at, COALESCE(delivered_to_device_id,''), COALESCE(delivered_to_inspector_id,''), expires_at, revoked_at, COALESCE(revoked_by,''), created_by, created_at
		 FROM offline_package WHERE tenant_id=$1 AND organization_id=$2 AND id=$3`,
		actor.TenantID, actor.OrganizationID, packageID).Scan(
		&pkg.ID, &pkg.EntitlementID, &pkg.PackageVersion, &pkg.FormSnapshot, &pkg.EvidencePolicySnapshot, &pkg.AssetContext,
		&pkg.PackageHash, &pkg.Status, &pkg.GeneratedAt, &pkg.DeliveredAt, &pkg.DeliveredToDeviceID, &pkg.DeliveredToInspectorID,
		&pkg.ExpiresAt, &pkg.RevokedAt, &pkg.RevokedBy, &pkg.CreatedBy, &pkg.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return assetentitlement.OfflinePackage{}, ErrPackageNotFound
	}
	if err != nil {
		return assetentitlement.OfflinePackage{}, err
	}
	pkg.TenantID = actor.TenantID
	pkg.OrganizationID = actor.OrganizationID
	return pkg, nil
}

func (r *PackageRepository) begin(ctx context.Context, actor assetentitlement.ActorContext) (*sql.Tx, error) {
	return pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
}

// Ensure json import is used
var _ = json.Marshal
