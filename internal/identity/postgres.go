// INTEGIN identity foundation: PostgreSQL exposes only a narrow, local security-definer resolver to the runtime role.
package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

// PostgresResolver invokes the migration-owned resolver function; the runtime role receives no direct identity-table grants.
type PostgresResolver struct {
	db *sql.DB
	m  *pgtype.Map
}

func NewPostgresResolver(db *sql.DB) (*PostgresResolver, error) {
	if db == nil {
		return nil, errors.New("identity resolver database is required")
	}
	return &PostgresResolver{db: db, m: pgtype.NewMap()}, nil
}

func (r *PostgresResolver) Resolve(ctx context.Context, principal PrincipalKey) (Membership, error) {
	if err := ValidatePrincipalKey(principal); err != nil {
		return Membership{}, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT actor_id, tenant_id, organization_id, work_order_role, capabilities FROM integin_resolve_identity_membership($1, $2)`, principal.Issuer, principal.Subject)
	if err != nil {
		return Membership{}, fmt.Errorf("resolve local identity membership: %w", err)
	}
	defer rows.Close()
	result := make([]Membership, 0, 2)
	for rows.Next() {
		var membership Membership
		var capabilities []string
		if err := rows.Scan(&membership.ActorID, &membership.TenantID, &membership.OrganizationID, &membership.WorkOrderRole, r.m.SQLScanner(&capabilities)); err != nil {
			return Membership{}, fmt.Errorf("scan local identity membership: %w", err)
		}
		membership.Capabilities = append([]string(nil), capabilities...)
		result = append(result, membership)
	}
	if err := rows.Err(); err != nil {
		return Membership{}, err
	}
	switch len(result) {
	case 0:
		return Membership{}, ErrUnknownSubject
	case 1:
		return result[0], nil
	default:
		return Membership{}, ErrAmbiguousMembership
	}
}
