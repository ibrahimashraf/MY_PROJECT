package certtemplatepg

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"integin/internal/domain/certificatetemplate"
)

var (
	ErrNilDB    = errors.New("certificate template postgres repository requires a database")
	ErrNotFound = errors.New("certificate template version not found")
	ErrNotDraft = errors.New("certificate template version is not a draft")
)

type Repository struct {
	db      *sql.DB
	catalog certificatetemplate.Catalog
}

func NewRepository(db *sql.DB, catalog certificatetemplate.Catalog) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	if catalog.Version <= 0 {
		return nil, errors.New("certificate binding catalog is required")
	}
	return &Repository{db: db, catalog: catalog}, nil
}

func (r *Repository) RegisterDraft(ctx context.Context, actor certificatetemplate.ActorContext, definition certificatetemplate.Definition) (certificatetemplate.DefinitionRecord, bool, error) {
	if err := actor.Validate(); err != nil {
		return certificatetemplate.DefinitionRecord{}, false, err
	}
	definition.Status = certificatetemplate.Draft
	if err := definition.Validate(r.catalog); err != nil {
		return certificatetemplate.DefinitionRecord{}, false, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return certificatetemplate.DefinitionRecord{}, false, err
	}
	defer tx.Rollback()
	stored, err := load(ctx, tx, actor, definition.TemplateCode, definition.Version)
	if err == nil {
		if !sameDefinition(stored.Definition, definition) {
			return certificatetemplate.DefinitionRecord{}, false, certificatetemplate.ErrImmutableConflict
		}
		return stored, false, tx.Commit()
	}
	if !errors.Is(err, ErrNotFound) {
		return certificatetemplate.DefinitionRecord{}, false, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO certificate_template (id, tenant_id, organization_id, template_code, version, title, asset_type, status, catalog_version, page_count, page_width_points, page_height_points, created_by) VALUES ($1,$2,$3,$4,$5,$6,$7,'DRAFT',$8,$9,$10,$11,$12)`, definition.ID, actor.TenantID, actor.OrganizationID, definition.TemplateCode, definition.Version, definition.Title, definition.AssetType, definition.CatalogVersion, definition.PageCount, definition.PageWidth, definition.PageHeight, actor.ActorID); err != nil {
		return certificatetemplate.DefinitionRecord{}, false, err
	}
	for _, cell := range definition.SortedCells() {
		checkboxValues, err := json.Marshal(cell.CheckboxValues)
		if err != nil {
			return certificatetemplate.DefinitionRecord{}, false, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO certificate_template_cell (tenant_id, organization_id, template_id, cell_id, page_number, x_points, y_points, width_points, height_points, kind, binding_key, static_text, label, fit_policy, max_lines, required, checkbox_values, condition_binding_key, condition_operator, condition_literal, repeat_source, max_items) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,''),NULLIF($12,''),NULLIF($13,''),$14,NULLIF($15,0),$16,NULLIF($17,'{}'::jsonb),NULLIF($18,''),NULLIF($19,''),NULLIF($20,''),NULLIF($21,''),NULLIF($22,0))`, actor.TenantID, actor.OrganizationID, definition.ID, cell.ID, cell.PageNumber, cell.Rectangle.X, cell.Rectangle.Y, cell.Rectangle.Width, cell.Rectangle.Height, string(cell.Kind), string(cell.BindingKey), cell.StaticText, cell.Label, string(cell.FitPolicy), cell.MaxLines, cell.Required, checkboxValues, nullableBinding(cell.Applicability, func(a *certificatetemplate.Applicability) string { return string(a.BindingKey) }), nullableConditionOperator(cell.Applicability), nullableBinding(cell.Applicability, func(a *certificatetemplate.Applicability) string { return a.Literal }), string(cell.RepeatSource), cell.MaxItems); err != nil {
			return certificatetemplate.DefinitionRecord{}, false, err
		}
	}
	stored, err = load(ctx, tx, actor, definition.TemplateCode, definition.Version)
	if err != nil {
		return certificatetemplate.DefinitionRecord{}, false, err
	}
	return stored, true, tx.Commit()
}

func (r *Repository) Approve(ctx context.Context, actor certificatetemplate.ActorContext, templateCode string, version int, approvedAt time.Time) (certificatetemplate.DefinitionRecord, error) {
	if err := actor.Validate(); err != nil {
		return certificatetemplate.DefinitionRecord{}, err
	}
	if version <= 0 || approvedAt.IsZero() {
		return certificatetemplate.DefinitionRecord{}, errors.New("positive template version and approval time are required")
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return certificatetemplate.DefinitionRecord{}, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE certificate_template SET status = 'APPROVED', approved_by = $1, approved_at = $2 WHERE tenant_id = $3 AND organization_id = $4 AND template_code = $5 AND version = $6 AND status = 'DRAFT'`, actor.ActorID, approvedAt.UTC(), actor.TenantID, actor.OrganizationID, templateCode, version)
	if err != nil {
		return certificatetemplate.DefinitionRecord{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return certificatetemplate.DefinitionRecord{}, err
	}
	if count == 0 {
		if _, err := load(ctx, tx, actor, templateCode, version); errors.Is(err, ErrNotFound) {
			return certificatetemplate.DefinitionRecord{}, ErrNotFound
		}
		return certificatetemplate.DefinitionRecord{}, ErrNotDraft
	}
	stored, err := load(ctx, tx, actor, templateCode, version)
	if err != nil {
		return certificatetemplate.DefinitionRecord{}, err
	}
	return stored, tx.Commit()
}

func (r *Repository) Get(ctx context.Context, actor certificatetemplate.ActorContext, templateCode string, version int) (certificatetemplate.DefinitionRecord, error) {
	if err := actor.Validate(); err != nil {
		return certificatetemplate.DefinitionRecord{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return certificatetemplate.DefinitionRecord{}, err
	}
	defer tx.Rollback()
	stored, err := load(ctx, tx, actor, templateCode, version)
	if err != nil {
		return certificatetemplate.DefinitionRecord{}, err
	}
	return stored, tx.Commit()
}

func (r *Repository) begin(ctx context.Context, actor certificatetemplate.ActorContext) (*sql.Tx, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id',$1,true), set_config('integin.organization_id',$2,true)`, actor.TenantID, actor.OrganizationID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func load(ctx context.Context, tx *sql.Tx, actor certificatetemplate.ActorContext, templateCode string, version int) (certificatetemplate.DefinitionRecord, error) {
	var record certificatetemplate.DefinitionRecord
	var status string
	var approvedBy sql.NullString
	var approvedAt sql.NullTime
	err := tx.QueryRowContext(ctx, `SELECT id, template_code, version, title, asset_type, status, catalog_version, page_count, page_width_points, page_height_points, created_by, created_at, approved_by, approved_at FROM certificate_template WHERE tenant_id=$1 AND organization_id=$2 AND template_code=$3 AND version=$4`, actor.TenantID, actor.OrganizationID, templateCode, version).Scan(&record.ID, &record.TemplateCode, &record.Version, &record.Title, &record.AssetType, &status, &record.CatalogVersion, &record.PageCount, &record.PageWidth, &record.PageHeight, &record.CreatedBy, &record.CreatedAt, &approvedBy, &approvedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return certificatetemplate.DefinitionRecord{}, ErrNotFound
	}
	if err != nil {
		return certificatetemplate.DefinitionRecord{}, err
	}
	record.Status = certificatetemplate.Status(status)
	record.ApprovedBy = approvedBy.String
	if approvedAt.Valid {
		record.ApprovedAt = approvedAt.Time.UTC()
	}
	rows, err := tx.QueryContext(ctx, `SELECT cell_id, page_number, x_points, y_points, width_points, height_points, kind, COALESCE(binding_key,''), COALESCE(static_text,''), COALESCE(label,''), fit_policy, COALESCE(max_lines,0), required, checkbox_values, COALESCE(condition_binding_key,''), COALESCE(condition_operator,''), COALESCE(condition_literal,''), COALESCE(repeat_source,''), COALESCE(max_items,0) FROM certificate_template_cell WHERE tenant_id=$1 AND organization_id=$2 AND template_id=$3 ORDER BY page_number, cell_id`, actor.TenantID, actor.OrganizationID, record.ID)
	if err != nil {
		return certificatetemplate.DefinitionRecord{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var cell certificatetemplate.Cell
		var kind, fitPolicy, operator string
		var checkboxValues []byte
		var conditionBinding, conditionLiteral, repeatSource string
		if err := rows.Scan(&cell.ID, &cell.PageNumber, &cell.Rectangle.X, &cell.Rectangle.Y, &cell.Rectangle.Width, &cell.Rectangle.Height, &kind, &cell.BindingKey, &cell.StaticText, &cell.Label, &fitPolicy, &cell.MaxLines, &cell.Required, &checkboxValues, &conditionBinding, &operator, &conditionLiteral, &repeatSource, &cell.MaxItems); err != nil {
			return certificatetemplate.DefinitionRecord{}, err
		}
		cell.Kind, cell.FitPolicy, cell.RepeatSource = certificatetemplate.PresentationKind(kind), certificatetemplate.FitPolicy(fitPolicy), certificatetemplate.BindingKey(repeatSource)
		if len(checkboxValues) > 0 && string(checkboxValues) != "null" {
			if err := json.Unmarshal(checkboxValues, &cell.CheckboxValues); err != nil {
				return certificatetemplate.DefinitionRecord{}, err
			}
		}
		if conditionBinding != "" {
			cell.Applicability = &certificatetemplate.Applicability{BindingKey: certificatetemplate.BindingKey(conditionBinding), Operator: certificatetemplate.ConditionOperator(operator), Literal: conditionLiteral}
		}
		record.Cells = append(record.Cells, cell)
	}
	if err := rows.Err(); err != nil {
		return certificatetemplate.DefinitionRecord{}, err
	}
	return record, nil
}

func nullableBinding(applicability *certificatetemplate.Applicability, project func(*certificatetemplate.Applicability) string) string {
	if applicability == nil {
		return ""
	}
	return project(applicability)
}

func nullableConditionOperator(applicability *certificatetemplate.Applicability) string {
	if applicability == nil {
		return ""
	}
	return string(applicability.Operator)
}

func sameDefinition(left, right certificatetemplate.Definition) bool {
	leftCells, rightCells := left.SortedCells(), right.SortedCells()
	if left.ID != right.ID || left.TemplateCode != right.TemplateCode || left.Version != right.Version || left.Title != right.Title || left.AssetType != right.AssetType || left.CatalogVersion != right.CatalogVersion || left.PageCount != right.PageCount || left.PageWidth != right.PageWidth || left.PageHeight != right.PageHeight || len(leftCells) != len(rightCells) {
		return false
	}
	for index := range leftCells {
		leftJSON, leftErr := json.Marshal(leftCells[index])
		rightJSON, rightErr := json.Marshal(rightCells[index])
		if leftErr != nil || rightErr != nil || string(leftJSON) != string(rightJSON) {
			return false
		}
	}
	return true
}
