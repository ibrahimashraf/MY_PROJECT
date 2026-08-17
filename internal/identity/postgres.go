// INTEGIN identity foundation: PostgreSQL exposes only a narrow, local security-definer resolver to the runtime role.
package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

// PostgresResolver invokes the migration-owned resolver function; the runtime role receives no direct identity-table grants.
type PostgresResolver struct {
	db *sql.DB
}

func NewPostgresResolver(db *sql.DB) (*PostgresResolver, error) {
	if db == nil {
		return nil, errors.New("identity resolver database is required")
	}
	return &PostgresResolver{db: db}, nil
}

func (r *PostgresResolver) Resolve(ctx context.Context, principal PrincipalKey) (Membership, error) {
	if err := ValidatePrincipalKey(principal); err != nil {
		return Membership{}, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT tenant_id, organization_id, capabilities FROM integin_resolve_identity_membership($1, $2)`, principal.Issuer, principal.Subject)
	if err != nil {
		return Membership{}, fmt.Errorf("resolve local identity membership: %w", err)
	}
	defer rows.Close()
	result := make([]Membership, 0, 2)
	for rows.Next() {
		var membership Membership
		var capabilities pq.StringArray
		if err := rows.Scan(&membership.TenantID, &membership.OrganizationID, &capabilities); err != nil {
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
