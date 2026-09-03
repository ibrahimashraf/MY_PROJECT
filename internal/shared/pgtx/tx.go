package pgtx

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

var (
	ErrNilDB        = errors.New("database handle is required")
	ErrInvalidScope = errors.New("tenant and organization scope are required")
)

// BeginScope begins a new transaction on db and configures the PostgreSQL session
// variables 'integin.tenant_id' and 'integin.organization_id' with local transaction scope (is_local = true).
// If any step fails, the transaction is safely rolled back and the error is returned.
func BeginScope(ctx context.Context, db *sql.DB, tenantID, organizationID string) (*sql.Tx, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(organizationID) == "" {
		return nil, ErrInvalidScope
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx,
		`SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)`,
		tenantID, organizationID,
	); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return tx, nil
}
