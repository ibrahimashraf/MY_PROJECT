package flagpg

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"integin/internal/shared/featureflags"
	"integin/internal/shared/pgtx"
)

var ErrNilDB = errors.New("flag postgres repository requires a database")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

type OverrideRecord struct {
	ID             string             `json:"id"`
	TenantID       string             `json:"tenant_id"`
	OrganizationID string             `json:"organization_id"`
	FlagKey        featureflags.Key   `json:"flag_key"`
	Scope          featureflags.Scope `json:"scope"`
	ScopeID        string             `json:"scope_id"`
	State          featureflags.State `json:"state"`
	Reason         string             `json:"reason,omitempty"`
	ExpiresAt      *time.Time         `json:"expires_at,omitempty"`
	CreatedBy      string             `json:"created_by"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

func (r *Repository) begin(ctx context.Context, tenantID, orgID string) (*sql.Tx, error) {
	return pgtx.BeginScope(ctx, r.db, tenantID, orgID)
}

func (r *Repository) ListOverrides(ctx context.Context, tenantID, orgID string) ([]OverrideRecord, error) {
	tx, err := r.begin(ctx, tenantID, orgID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx,
		`SELECT id, tenant_id, organization_id, flag_key, scope, scope_id, state,
		        reason, expires_at, created_by, created_at, updated_at
		 FROM feature_flag_override
		 WHERE tenant_id = current_setting('integin.tenant_id', true)
		   AND organization_id = current_setting('integin.organization_id', true)
		 ORDER BY flag_key, scope, scope_id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []OverrideRecord
	for rows.Next() {
		var rec OverrideRecord
		var reason sql.NullString
		var expiresAt sql.NullTime
		if err := rows.Scan(&rec.ID, &rec.TenantID, &rec.OrganizationID, &rec.FlagKey,
			&rec.Scope, &rec.ScopeID, &rec.State, &reason, &expiresAt,
			&rec.CreatedBy, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		if reason.Valid {
			rec.Reason = reason.String
		}
		if expiresAt.Valid {
			rec.ExpiresAt = &expiresAt.Time
		}
		results = append(results, rec)
	}
	return results, tx.Commit()
}

func (r *Repository) GetOverride(ctx context.Context, tenantID, orgID string, key featureflags.Key, scope featureflags.Scope, scopeID string) (OverrideRecord, bool, error) {
	tx, err := r.begin(ctx, tenantID, orgID)
	if err != nil {
		return OverrideRecord{}, false, err
	}
	defer tx.Rollback()

	var rec OverrideRecord
	var reason sql.NullString
	var expiresAt sql.NullTime
	err = tx.QueryRowContext(ctx,
		`SELECT id, tenant_id, organization_id, flag_key, scope, scope_id, state,
		        reason, expires_at, created_by, created_at, updated_at
		 FROM feature_flag_override
		 WHERE tenant_id = current_setting('integin.tenant_id', true)
		   AND organization_id = current_setting('integin.organization_id', true)
		   AND flag_key = $1 AND scope = $2 AND scope_id = $3`,
		string(key), string(scope), scopeID,
	).Scan(&rec.ID, &rec.TenantID, &rec.OrganizationID, &rec.FlagKey,
		&rec.Scope, &rec.ScopeID, &rec.State, &reason, &expiresAt,
		&rec.CreatedBy, &rec.CreatedAt, &rec.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return OverrideRecord{}, false, nil
	}
	if err != nil {
		return OverrideRecord{}, false, err
	}
	if reason.Valid {
		rec.Reason = reason.String
	}
	if expiresAt.Valid {
		rec.ExpiresAt = &expiresAt.Time
	}
	return rec, true, tx.Commit()
}

func (r *Repository) UpsertOverride(ctx context.Context, tenantID, orgID string, rec OverrideRecord) (OverrideRecord, error) {
	tx, err := r.begin(ctx, tenantID, orgID)
	if err != nil {
		return OverrideRecord{}, err
	}
	defer tx.Rollback()

	now := time.Now()
	if rec.ID == "" {
		rec.ID = tenantID + ":" + orgID + ":" + string(rec.FlagKey) + ":" + string(rec.Scope) + ":" + rec.ScopeID
	}
	rec.UpdatedAt = now

	// Check if record exists to preserve CreatedAt
	var existingCreatedAt *time.Time
	err = tx.QueryRowContext(ctx,
		`SELECT created_at FROM feature_flag_override
		 WHERE tenant_id = $1 AND organization_id = $2 AND flag_key = $3 AND scope = $4 AND scope_id = $5`,
		tenantID, orgID, string(rec.FlagKey), string(rec.Scope), rec.ScopeID,
	).Scan(&existingCreatedAt)

	if err == nil && existingCreatedAt != nil {
		rec.CreatedAt = *existingCreatedAt
	} else {
		rec.CreatedAt = now
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO feature_flag_override (id, tenant_id, organization_id, flag_key, scope, scope_id, state, reason, expires_at, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 ON CONFLICT (tenant_id, organization_id, flag_key, scope, scope_id)
		 DO UPDATE SET state = EXCLUDED.state, reason = EXCLUDED.reason, expires_at = EXCLUDED.expires_at, updated_at = EXCLUDED.updated_at`,
		rec.ID, rec.TenantID, rec.OrganizationID, string(rec.FlagKey), string(rec.Scope), rec.ScopeID,
		string(rec.State), rec.Reason, rec.ExpiresAt, rec.CreatedBy, rec.CreatedAt, rec.UpdatedAt,
	)
	if err != nil {
		return OverrideRecord{}, err
	}
	return rec, tx.Commit()
}

func (r *Repository) DeleteOverride(ctx context.Context, tenantID, orgID string, key featureflags.Key, scope featureflags.Scope, scopeID string) error {
	tx, err := r.begin(ctx, tenantID, orgID)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`DELETE FROM feature_flag_override
		 WHERE tenant_id = current_setting('integin.tenant_id', true)
		   AND organization_id = current_setting('integin.organization_id', true)
		   AND flag_key = $1 AND scope = $2 AND scope_id = $3`,
		string(key), string(scope), scopeID,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}
