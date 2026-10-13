package formdefinitionpg

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"integin/internal/domain/formdefinition"
	"integin/internal/shared/pgtx"
)

var (
	ErrNilDB       = errors.New("form definition postgres repository requires a database")
	ErrNotFound    = errors.New("form version not found")
	ErrNotDraft    = errors.New("form version is not a draft")
	ErrNotApproved = errors.New("form version is not approved")
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

func (r *Repository) RegisterDraft(ctx context.Context, actor formdefinition.ActorContext, form formdefinition.FormVersion) (formdefinition.FormVersion, bool, error) {
	if err := actor.Validate(); err != nil {
		return formdefinition.FormVersion{}, false, err
	}
	form.Status = formdefinition.FormStatusDraft
	if err := form.Validate(); err != nil {
		return formdefinition.FormVersion{}, false, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return formdefinition.FormVersion{}, false, err
	}
	defer tx.Rollback()
	existing, err := loadForm(ctx, tx, actor, form.FormCode, form.Version)
	if err == nil {
		if !sameForm(existing, form) {
			return formdefinition.FormVersion{}, false, formdefinition.ErrImmutable
		}
		return existing, false, tx.Commit()
	}
	if !errors.Is(err, ErrNotFound) {
		return formdefinition.FormVersion{}, false, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO form_version (id, tenant_id, organization_id, form_code, version, title, description, asset_type, status, catalog_version, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''),NULLIF($8,''),'DRAFT',$9,$10)`,
		form.ID, actor.TenantID, actor.OrganizationID, form.FormCode, form.Version, form.Title, form.Description, form.AssetType, form.CatalogVersion, actor.ActorID); err != nil {
		return formdefinition.FormVersion{}, false, err
	}
	for _, field := range form.Fields {
		if err := insertField(ctx, tx, actor, form.ID, field); err != nil {
			return formdefinition.FormVersion{}, false, err
		}
		for _, policy := range field.EvidencePolicies {
			if err := insertPolicy(ctx, tx, actor, form.ID, field.FieldID, policy); err != nil {
				return formdefinition.FormVersion{}, false, err
			}
		}
	}
	stored, err := loadForm(ctx, tx, actor, form.FormCode, form.Version)
	if err != nil {
		return formdefinition.FormVersion{}, false, err
	}
	return stored, true, tx.Commit()
}

func (r *Repository) Approve(ctx context.Context, actor formdefinition.ActorContext, formID string, expectedVersion int) (formdefinition.FormVersion, error) {
	if err := actor.Validate(); err != nil {
		return formdefinition.FormVersion{}, err
	}
	if expectedVersion <= 0 {
		return formdefinition.FormVersion{}, errors.New("positive form version is required")
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return formdefinition.FormVersion{}, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx,
		`UPDATE form_version SET status = 'APPROVED', approved_by = $1, approved_at = $2
		 WHERE tenant_id = $3 AND organization_id = $4 AND id = $5 AND version = $6 AND status = 'DRAFT'`,
		actor.ActorID, now, actor.TenantID, actor.OrganizationID, formID, expectedVersion)
	if err != nil {
		return formdefinition.FormVersion{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return formdefinition.FormVersion{}, err
	}
	if count == 0 {
		if _, err := loadFormByID(ctx, tx, actor, formID); errors.Is(err, ErrNotFound) {
			return formdefinition.FormVersion{}, ErrNotFound
		}
		return formdefinition.FormVersion{}, ErrNotDraft
	}
	stored, err := loadFormByID(ctx, tx, actor, formID)
	if err != nil {
		return formdefinition.FormVersion{}, err
	}
	return stored, tx.Commit()
}

func (r *Repository) Retire(ctx context.Context, actor formdefinition.ActorContext, formID string, expectedVersion int) (formdefinition.FormVersion, error) {
	if err := actor.Validate(); err != nil {
		return formdefinition.FormVersion{}, err
	}
	if expectedVersion <= 0 {
		return formdefinition.FormVersion{}, errors.New("positive form version is required")
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return formdefinition.FormVersion{}, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx,
		`UPDATE form_version SET status = 'RETIRED', retired_by = $1, retired_at = $2
		 WHERE tenant_id = $3 AND organization_id = $4 AND id = $5 AND version = $6 AND status = 'APPROVED'`,
		actor.ActorID, now, actor.TenantID, actor.OrganizationID, formID, expectedVersion)
	if err != nil {
		return formdefinition.FormVersion{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return formdefinition.FormVersion{}, err
	}
	if count == 0 {
		existing, err := loadFormByID(ctx, tx, actor, formID)
		if errors.Is(err, ErrNotFound) {
			return formdefinition.FormVersion{}, ErrNotFound
		}
		if err != nil {
			return formdefinition.FormVersion{}, err
		}
		if existing.Status != formdefinition.FormStatusApproved {
			return formdefinition.FormVersion{}, formdefinition.ErrInvalidStatus
		}
		return formdefinition.FormVersion{}, formdefinition.ErrInvalidStatus
	}
	stored, err := loadFormByID(ctx, tx, actor, formID)
	if err != nil {
		return formdefinition.FormVersion{}, err
	}
	return stored, tx.Commit()
}

func (r *Repository) Get(ctx context.Context, actor formdefinition.ActorContext, formID string) (formdefinition.FormVersion, error) {
	if err := actor.Validate(); err != nil {
		return formdefinition.FormVersion{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return formdefinition.FormVersion{}, err
	}
	defer tx.Rollback()
	stored, err := loadFormByID(ctx, tx, actor, formID)
	if err != nil {
		return formdefinition.FormVersion{}, err
	}
	return stored, tx.Commit()
}

func (r *Repository) GetByCode(ctx context.Context, actor formdefinition.ActorContext, formCode string) (formdefinition.FormVersion, error) {
	if err := actor.Validate(); err != nil {
		return formdefinition.FormVersion{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return formdefinition.FormVersion{}, err
	}
	defer tx.Rollback()
	var form formdefinition.FormVersion
	err = tx.QueryRowContext(ctx,
		`SELECT id, form_code, version, title, COALESCE(description,''), COALESCE(asset_type,''), status, catalog_version, created_by, created_at, COALESCE(approved_by,''), approved_at, COALESCE(retired_by,''), retired_at
		 FROM form_version WHERE tenant_id=$1 AND organization_id=$2 AND form_code=$3
		 ORDER BY version DESC LIMIT 1`,
		actor.TenantID, actor.OrganizationID, formCode).Scan(
		&form.ID, &form.FormCode, &form.Version, &form.Title, &form.Description, &form.AssetType,
		&form.Status, &form.CatalogVersion, &form.CreatedBy, &form.CreatedAt,
		&form.ApprovedBy, &form.ApprovedAt, &form.RetiredBy, &form.RetiredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return formdefinition.FormVersion{}, ErrNotFound
	}
	if err != nil {
		return formdefinition.FormVersion{}, err
	}
	form.TenantID = actor.TenantID
	form.OrganizationID = actor.OrganizationID
	if err := loadFields(ctx, tx, actor, &form); err != nil {
		return formdefinition.FormVersion{}, err
	}
	return form, tx.Commit()
}

func (r *Repository) ListByAssetType(ctx context.Context, actor formdefinition.ActorContext, assetType string) ([]formdefinition.FormVersion, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx,
		`SELECT id, form_code, version, title, COALESCE(description,''), COALESCE(asset_type,''), status, catalog_version, created_by, created_at, COALESCE(approved_by,''), approved_at, COALESCE(retired_by,''), retired_at
		 FROM form_version WHERE tenant_id=$1 AND organization_id=$2 AND asset_type=$3
		 ORDER BY form_code, version DESC`,
		actor.TenantID, actor.OrganizationID, assetType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var forms []formdefinition.FormVersion
	for rows.Next() {
		var form formdefinition.FormVersion
		if err := rows.Scan(
			&form.ID, &form.FormCode, &form.Version, &form.Title, &form.Description, &form.AssetType,
			&form.Status, &form.CatalogVersion, &form.CreatedBy, &form.CreatedAt,
			&form.ApprovedBy, &form.ApprovedAt, &form.RetiredBy, &form.RetiredAt); err != nil {
			return nil, err
		}
		form.TenantID = actor.TenantID
		form.OrganizationID = actor.OrganizationID
		forms = append(forms, form)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return forms, tx.Commit()
}

func (r *Repository) ListByStatus(ctx context.Context, actor formdefinition.ActorContext, status formdefinition.FormStatus) ([]formdefinition.FormVersion, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx,
		`SELECT id, form_code, version, title, COALESCE(description,''), COALESCE(asset_type,''), status, catalog_version, created_by, created_at, COALESCE(approved_by,''), approved_at, COALESCE(retired_by,''), retired_at
		 FROM form_version WHERE tenant_id=$1 AND organization_id=$2 AND status=$3
		 ORDER BY form_code, version DESC`,
		actor.TenantID, actor.OrganizationID, string(status))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var forms []formdefinition.FormVersion
	for rows.Next() {
		var form formdefinition.FormVersion
		if err := rows.Scan(
			&form.ID, &form.FormCode, &form.Version, &form.Title, &form.Description, &form.AssetType,
			&form.Status, &form.CatalogVersion, &form.CreatedBy, &form.CreatedAt,
			&form.ApprovedBy, &form.ApprovedAt, &form.RetiredBy, &form.RetiredAt); err != nil {
			return nil, err
		}
		form.TenantID = actor.TenantID
		form.OrganizationID = actor.OrganizationID
		forms = append(forms, form)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return forms, tx.Commit()
}

func (r *Repository) begin(ctx context.Context, actor formdefinition.ActorContext) (*sql.Tx, error) {
	return pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
}

func loadFormByID(ctx context.Context, tx *sql.Tx, actor formdefinition.ActorContext, formID string) (formdefinition.FormVersion, error) {
	var form formdefinition.FormVersion
	err := tx.QueryRowContext(ctx,
		`SELECT id, form_code, version, title, COALESCE(description,''), COALESCE(asset_type,''), status, catalog_version, created_by, created_at, COALESCE(approved_by,''), approved_at, COALESCE(retired_by,''), retired_at
		 FROM form_version WHERE tenant_id=$1 AND organization_id=$2 AND id=$3`,
		actor.TenantID, actor.OrganizationID, formID).Scan(
		&form.ID, &form.FormCode, &form.Version, &form.Title, &form.Description, &form.AssetType,
		&form.Status, &form.CatalogVersion, &form.CreatedBy, &form.CreatedAt,
		&form.ApprovedBy, &form.ApprovedAt, &form.RetiredBy, &form.RetiredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return formdefinition.FormVersion{}, ErrNotFound
	}
	if err != nil {
		return formdefinition.FormVersion{}, err
	}
	form.TenantID = actor.TenantID
	form.OrganizationID = actor.OrganizationID
	if err := loadFields(ctx, tx, actor, &form); err != nil {
		return formdefinition.FormVersion{}, err
	}
	return form, nil
}

func loadForm(ctx context.Context, tx *sql.Tx, actor formdefinition.ActorContext, formCode string, version int) (formdefinition.FormVersion, error) {
	var form formdefinition.FormVersion
	err := tx.QueryRowContext(ctx,
		`SELECT id, form_code, version, title, COALESCE(description,''), COALESCE(asset_type,''), status, catalog_version, created_by, created_at, COALESCE(approved_by,''), approved_at, COALESCE(retired_by,''), retired_at
		 FROM form_version WHERE tenant_id=$1 AND organization_id=$2 AND form_code=$3 AND version=$4`,
		actor.TenantID, actor.OrganizationID, formCode, version).Scan(
		&form.ID, &form.FormCode, &form.Version, &form.Title, &form.Description, &form.AssetType,
		&form.Status, &form.CatalogVersion, &form.CreatedBy, &form.CreatedAt,
		&form.ApprovedBy, &form.ApprovedAt, &form.RetiredBy, &form.RetiredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return formdefinition.FormVersion{}, ErrNotFound
	}
	if err != nil {
		return formdefinition.FormVersion{}, err
	}
	form.TenantID = actor.TenantID
	form.OrganizationID = actor.OrganizationID
	if err := loadFields(ctx, tx, actor, &form); err != nil {
		return formdefinition.FormVersion{}, err
	}
	return form, nil
}

func loadFields(ctx context.Context, tx *sql.Tx, actor formdefinition.ActorContext, form *formdefinition.FormVersion) error {
	rows, err := tx.QueryContext(ctx,
		`SELECT field_id, field_code, field_type, label, COALESCE(description,''), required, critical, sort_order,
		        validation_rules, options, COALESCE(default_value,''), COALESCE(section_id,''), COALESCE(section_label,'')
		 FROM form_field WHERE tenant_id=$1 AND organization_id=$2 AND form_id=$3
		 ORDER BY sort_order, field_id`,
		actor.TenantID, actor.OrganizationID, form.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var field formdefinition.FormField
		var validationRulesJSON, optionsJSON []byte
		if err := rows.Scan(
			&field.FieldID, &field.FieldCode, &field.FieldType, &field.Label, &field.Description,
			&field.Required, &field.Critical, &field.SortOrder,
			&validationRulesJSON, &optionsJSON, &field.DefaultValue,
			&field.SectionID, &field.SectionLabel); err != nil {
			return err
		}
		field.FormID = form.ID
		field.TenantID = actor.TenantID
		field.OrganizationID = actor.OrganizationID
		if len(validationRulesJSON) > 0 && string(validationRulesJSON) != "null" {
			if err := json.Unmarshal(validationRulesJSON, &field.ValidationRules); err != nil {
				return fmt.Errorf("failed to unmarshal validation_rules: %w", err)
			}
		}
		if len(optionsJSON) > 0 && string(optionsJSON) != "null" {
			if err := json.Unmarshal(optionsJSON, &field.Options); err != nil {
				return fmt.Errorf("failed to unmarshal options: %w", err)
			}
		}
		if err := loadPolicies(ctx, tx, actor, form.ID, &field); err != nil {
			return err
		}
		form.Fields = append(form.Fields, field)
	}
	return rows.Err()
}

func loadPolicies(ctx context.Context, tx *sql.Tx, actor formdefinition.ActorContext, formID string, field *formdefinition.FormField) error {
	rows, err := tx.QueryContext(ctx,
		`SELECT field_id, evidence_type, required, max_count, max_size_bytes, allowed_content_types, retention_days, classification
		 FROM evidence_policy WHERE tenant_id=$1 AND organization_id=$2 AND form_id=$3 AND field_id=$4
		 ORDER BY evidence_type`,
		actor.TenantID, actor.OrganizationID, formID, field.FieldID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var policy formdefinition.EvidencePolicy
		var allowedContentTypesJSON []byte
		var maxCount, maxSizeBytes, retentionDays sql.NullInt64
		if err := rows.Scan(
			&policy.FieldID, &policy.EvidenceType, &policy.Required,
			&maxCount, &maxSizeBytes, &allowedContentTypesJSON, &retentionDays, &policy.Classification); err != nil {
			return err
		}
		policy.FormID = formID
		policy.TenantID = actor.TenantID
		policy.OrganizationID = actor.OrganizationID
		if maxCount.Valid {
			v := maxCount.Int64
			policy.MaxCount = &v
		}
		if maxSizeBytes.Valid {
			v := maxSizeBytes.Int64
			policy.MaxSizeBytes = &v
		}
		if retentionDays.Valid {
			v := retentionDays.Int64
			policy.RetentionDays = &v
		}
		if len(allowedContentTypesJSON) > 0 && string(allowedContentTypesJSON) != "null" {
			if err := json.Unmarshal(allowedContentTypesJSON, &policy.AllowedContentTypes); err != nil {
				return fmt.Errorf("failed to unmarshal allowed_content_types: %w", err)
			}
		}
		field.EvidencePolicies = append(field.EvidencePolicies, policy)
	}
	return rows.Err()
}

func insertField(ctx context.Context, tx *sql.Tx, actor formdefinition.ActorContext, formID string, field formdefinition.FormField) error {
	validationRulesJSON, err := json.Marshal(field.ValidationRules)
	if err != nil {
		return err
	}
	optionsJSON, err := json.Marshal(field.Options)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO form_field (tenant_id, organization_id, form_id, field_id, field_code, field_type, label, description, required, critical, sort_order, validation_rules, options, default_value, section_id, section_label)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),$9,$10,$11,NULLIF($12,'{}'::jsonb),NULLIF($13,'{}'::jsonb),NULLIF($14,''),NULLIF($15,''),NULLIF($16,''))`,
		actor.TenantID, actor.OrganizationID, formID, field.FieldID, field.FieldCode,
		string(field.FieldType), field.Label, field.Description,
		field.Required, field.Critical, field.SortOrder,
		validationRulesJSON, optionsJSON, field.DefaultValue,
		field.SectionID, field.SectionLabel); err != nil {
		return err
	}
	return nil
}

func insertPolicy(ctx context.Context, tx *sql.Tx, actor formdefinition.ActorContext, formID, fieldID string, policy formdefinition.EvidencePolicy) error {
	allowedContentTypesJSON, err := json.Marshal(policy.AllowedContentTypes)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO evidence_policy (tenant_id, organization_id, form_id, field_id, evidence_type, required, max_count, max_size_bytes, allowed_content_types, retention_days, classification)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,'{}'::jsonb),$10,$11)`,
		actor.TenantID, actor.OrganizationID, formID, fieldID,
		string(policy.EvidenceType), policy.Required,
		policy.MaxCount, policy.MaxSizeBytes,
		allowedContentTypesJSON, policy.RetentionDays,
		string(policy.Classification)); err != nil {
		return err
	}
	return nil
}

func sameForm(left, right formdefinition.FormVersion) bool {
	leftFields := left.Fields
	rightFields := right.Fields
	if left.ID != right.ID || left.FormCode != right.FormCode || left.Version != right.Version || left.Title != right.Title || left.Description != right.Description || left.AssetType != right.AssetType || left.CatalogVersion != right.CatalogVersion || len(leftFields) != len(rightFields) {
		return false
	}
	for i := range leftFields {
		leftJSON, leftErr := json.Marshal(leftFields[i])
		rightJSON, rightErr := json.Marshal(rightFields[i])
		if leftErr != nil || rightErr != nil || string(leftJSON) != string(rightJSON) {
			return false
		}
	}
	return true
}

func normalizeString(s string) string {
	return strings.TrimSpace(s)
}
