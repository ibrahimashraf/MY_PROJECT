package settingspg

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

var ErrNilDB = errors.New("settings postgres repository requires a database")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

type Setting struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	OrganizationID string          `json:"organization_id"`
	SettingKey     string          `json:"setting_key"`
	SettingValue   json.RawMessage `json:"setting_value"`
	Scope          string          `json:"scope"`
	IsEditable     bool            `json:"is_editable"`
	Description    string          `json:"description,omitempty"`
	CreatedBy      string          `json:"created_by"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (r *Repository) begin(ctx context.Context, tenantID, orgID string) (*sql.Tx, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx,
		"SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)",
		tenantID, orgID,
	); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func (r *Repository) ListSettings(ctx context.Context, tenantID, orgID string) ([]Setting, error) {
	tx, err := r.begin(ctx, tenantID, orgID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx,
		`SELECT id, tenant_id, organization_id, setting_key, setting_value, scope, is_editable, description, created_by, created_at, updated_at
		 FROM tenant_setting
		 WHERE tenant_id = current_setting('integin.tenant_id', true)
		   AND organization_id = current_setting('integin.organization_id', true)
		 ORDER BY setting_key, scope`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []Setting
	for rows.Next() {
		var s Setting
		var description sql.NullString
		if err := rows.Scan(&s.ID, &s.TenantID, &s.OrganizationID, &s.SettingKey, &s.SettingValue,
			&s.Scope, &s.IsEditable, &description, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		if description.Valid {
			s.Description = description.String
		}
		settings = append(settings, s)
	}
	return settings, tx.Commit()
}

func (r *Repository) GetSetting(ctx context.Context, tenantID, orgID, key, scope string) (Setting, bool, error) {
	tx, err := r.begin(ctx, tenantID, orgID)
	if err != nil {
		return Setting{}, false, err
	}
	defer tx.Rollback()

	var s Setting
	var description sql.NullString
	err = tx.QueryRowContext(ctx,
		`SELECT id, tenant_id, organization_id, setting_key, setting_value, scope, is_editable, description, created_by, created_at, updated_at
		 FROM tenant_setting
		 WHERE tenant_id = current_setting('integin.tenant_id', true)
		   AND organization_id = current_setting('integin.organization_id', true)
		   AND setting_key = $1 AND scope = $2`,
		key, scope,
	).Scan(&s.ID, &s.TenantID, &s.OrganizationID, &s.SettingKey, &s.SettingValue,
		&s.Scope, &s.IsEditable, &description, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Setting{}, false, nil
	}
	if err != nil {
		return Setting{}, false, err
	}
	if description.Valid {
		s.Description = description.String
	}
	return s, true, tx.Commit()
}

func (r *Repository) UpsertSetting(ctx context.Context, tenantID, orgID string, s Setting) (Setting, error) {
	tx, err := r.begin(ctx, tenantID, orgID)
	if err != nil {
		return Setting{}, err
	}
	defer tx.Rollback()

	now := time.Now()
	if s.ID == "" {
		s.ID = tenantID + ":" + orgID + ":" + s.SettingKey + ":" + s.Scope
	}
	s.UpdatedAt = now

	_, err = tx.ExecContext(ctx,
		`INSERT INTO tenant_setting (id, tenant_id, organization_id, setting_key, setting_value, scope, is_editable, description, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (tenant_id, organization_id, setting_key, scope)
		 DO UPDATE SET setting_value = EXCLUDED.setting_value, description = EXCLUDED.description, updated_at = EXCLUDED.updated_at`,
		s.ID, s.TenantID, s.OrganizationID, s.SettingKey, s.SettingValue, s.Scope,
		s.IsEditable, s.Description, s.CreatedBy, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return Setting{}, err
	}
	return s, tx.Commit()
}
