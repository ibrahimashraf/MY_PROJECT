// INTEGIN organization-aware authorization: authenticates principal type and org MFA posture before projecting tenant-scoped context.
package identity

import (
	"errors"
	"strings"

	"integin/internal/oidcauth"
)

// PrincipalType discriminates human users from programmatic service accounts.
type PrincipalType uint8

const (
	PrincipalTypeHuman PrincipalType = iota
	PrincipalTypeServiceAccount
)

var (
	ErrMFARequired                = errors.New("identity principal did not authenticate with a permitted MFA method")
	ErrMissingOrganizationBinding = errors.New("identity membership lacks an organization binding")
	ErrCrossTenantScope           = errors.New("identity membership escapes the tenant scope")
)

// MFAPolicy is the organization authentication requirement applied after OIDC signature validation.
type MFAPolicy struct {
	RequireMFA bool
	AllowedAMR []string
}

// Enforce fails closed: when RequireMFA is true, the principal must carry at least one permitted
// AMR value. An empty AllowedAMR list under RequireMFA admits no authentication method.
func (p MFAPolicy) Enforce(principal oidcauth.Principal) error {
	if !p.RequireMFA {
		return nil
	}
	for _, method := range principal.AMR {
		for _, allowed := range p.AllowedAMR {
			if method == allowed {
				return nil
			}
		}
	}
	return ErrMFARequired
}

// OrganizationContext is the validated, tenant-scoped authorization projection bound into request context.
type OrganizationContext struct {
	TenantID       string
	OrganizationID string
	ActorID        string
	PrincipalType  PrincipalType
	Roles          []string
	Capabilities   []string
}

// IsServiceAccountSubject reports whether an OIDC subject or authorized-party identifier follows
// the INTEGIN service account convention.
func IsServiceAccountSubject(identifier string) bool {
	value := strings.ToLower(strings.TrimSpace(identifier))
	return strings.HasPrefix(value, "sa:") || strings.HasPrefix(value, "service-account:")
}

// PrincipalTypeOf classifies an authenticated principal. Both the subject and the authorized party
// are inspected so client-credentials tokens (subject and azp carry the client identifier) map to
// service accounts instead of humans.
func PrincipalTypeOf(principal oidcauth.Principal) PrincipalType {
	if IsServiceAccountSubject(principal.Subject) || IsServiceAccountSubject(principal.AuthorizedParty) {
		return PrincipalTypeServiceAccount
	}
	return PrincipalTypeHuman
}

// ProjectMembership enforces the organization MFA policy and maps one resolved local membership plus
// its authenticated principal into a validated OrganizationContext. It fails closed when the local
// binding is missing or escapes tenant scope, and copies slices so callers cannot mutate the projection.
func ProjectMembership(membership Membership, principal oidcauth.Principal, policy MFAPolicy) (OrganizationContext, error) {
	if err := ValidatePrincipalKey(PrincipalKey{Issuer: principal.Issuer, Subject: principal.Subject}); err != nil {
		return OrganizationContext{}, err
	}
	if err := policy.Enforce(principal); err != nil {
		return OrganizationContext{}, err
	}
	if strings.TrimSpace(membership.OrganizationID) == "" || strings.TrimSpace(membership.ActorID) == "" {
		return OrganizationContext{}, ErrMissingOrganizationBinding
	}
	if strings.TrimSpace(membership.TenantID) == "" {
		return OrganizationContext{}, ErrCrossTenantScope
	}
	org := OrganizationContext{
		TenantID:       membership.TenantID,
		OrganizationID: membership.OrganizationID,
		ActorID:        membership.ActorID,
		PrincipalType:  PrincipalTypeOf(principal),
		Capabilities:   append([]string{}, membership.Capabilities...),
	}
	if role := strings.TrimSpace(membership.WorkOrderRole); role != "" {
		org.Roles = []string{role}
	}
	return org, nil
}
