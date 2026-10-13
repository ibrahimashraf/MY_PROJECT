package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"integin/internal/shared/pgtx"
)

var (
	ErrInvalidGrant = errors.New("identity grant is invalid")
	ErrGrantFailed  = errors.New("identity grant failed")
)

// GrantInput binds an external issuer-subject to local tenant context.
// Role must satisfy the identity_membership_work_order_role_ck check;
// capabilities are format-validated (dotted namespace) but intentionally not
// matched against a hardcoded list, which would drift from authorizers.
type GrantInput struct {
	Issuer         string
	Subject        string
	TenantID       string
	OrganizationID string
	ActorID        string
	Role           string
	Capabilities   []string
}

var (
	grantRoles = map[string]bool{
		"inspector": true, "reviewer": true, "manager": true, "administrator": true,
	}
	capabilityPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)
)

func validateGrant(in GrantInput) error {
	if strings.TrimSpace(in.Issuer) == "" || strings.TrimSpace(in.Subject) == "" {
		return fmt.Errorf("%w: issuer and subject are required", ErrInvalidGrant)
	}
	if strings.TrimSpace(in.TenantID) == "" || strings.TrimSpace(in.OrganizationID) == "" {
		return fmt.Errorf("%w: tenant and organization are required", ErrInvalidGrant)
	}
	if strings.TrimSpace(in.ActorID) == "" {
		return fmt.Errorf("%w: actor is required", ErrInvalidGrant)
	}
	if !grantRoles[strings.TrimSpace(in.Role)] {
		return fmt.Errorf("%w: unknown role %q", ErrInvalidGrant, in.Role)
	}
	if len(in.Capabilities) == 0 {
		return fmt.Errorf("%w: at least one capability is required", ErrInvalidGrant)
	}
	for _, capability := range in.Capabilities {
		if len(capability) > 128 || !capabilityPattern.MatchString(capability) {
			return fmt.Errorf("%w: malformed capability %q", ErrInvalidGrant, capability)
		}
	}
	return nil
}

// GrantMembership upserts the actor, subject, membership, and capability rows
// for one issuer-subject binding inside a tenant-scoped transaction. It is
// idempotent: reruns reconcile status and add missing capabilities without
// disturbing existing rows. The actor ID must already be tenant-scoped by the
// caller: actor_id is globally unique while the membership FK binds the
// (actor_id, tenant_id, organization_id) triple.
func GrantMembership(ctx context.Context, db *sql.DB, in GrantInput) (membershipID int64, err error) {
	if err := validateGrant(in); err != nil {
		return 0, err
	}
	tx, err := pgtx.BeginScope(ctx, db, in.TenantID, in.OrganizationID)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO identity_actor (actor_id, tenant_id, organization_id, status) VALUES ($1,$2,$3,'ACTIVE') ON CONFLICT (actor_id) DO UPDATE SET status='ACTIVE', tenant_id=EXCLUDED.tenant_id, organization_id=EXCLUDED.organization_id`, in.ActorID, in.TenantID, in.OrganizationID); err != nil {
		return 0, fmt.Errorf("%w: actor: %v", ErrGrantFailed, err)
	}
	var subjectID int64
	if err := tx.QueryRowContext(ctx, `INSERT INTO identity_subject (issuer, subject, status) VALUES ($1,$2,'ACTIVE') ON CONFLICT (issuer, subject) DO UPDATE SET status='ACTIVE' RETURNING subject_id`, in.Issuer, in.Subject).Scan(&subjectID); err != nil {
		return 0, fmt.Errorf("%w: subject: %v", ErrGrantFailed, err)
	}
	if err := tx.QueryRowContext(ctx, `INSERT INTO identity_membership (subject_id, tenant_id, organization_id, status, actor_id, work_order_role) VALUES ($1,$2,$3,'ACTIVE',$4,$5) ON CONFLICT DO NOTHING RETURNING membership_id`, subjectID, in.TenantID, in.OrganizationID, in.ActorID, in.Role).Scan(&membershipID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("%w: membership: %v", ErrGrantFailed, err)
		}
		if err := tx.QueryRowContext(ctx, `SELECT membership_id FROM identity_membership WHERE subject_id=$1 AND tenant_id=$2 AND organization_id=$3 AND status='ACTIVE'`, subjectID, in.TenantID, in.OrganizationID).Scan(&membershipID); err != nil {
			return 0, fmt.Errorf("%w: membership lookup: %v", ErrGrantFailed, err)
		}
	}
	for _, capability := range in.Capabilities {
		if _, err := tx.ExecContext(ctx, `INSERT INTO identity_membership_capability (membership_id, capability, status) VALUES ($1,$2,'ACTIVE') ON CONFLICT DO NOTHING`, membershipID, capability); err != nil {
			return 0, fmt.Errorf("%w: capability: %v", ErrGrantFailed, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return membershipID, nil
}
