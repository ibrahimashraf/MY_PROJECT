// INTEGIN identity foundation: local mappings resolve authorization context only after external issuer-subject authentication succeeds.
package identity

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrUnknownSubject      = errors.New("identity subject is not locally mapped")
	ErrAmbiguousMembership = errors.New("identity subject has ambiguous active memberships")
)

// PrincipalKey is the immutable local lookup key established after OIDC signature and claim validation.
type PrincipalKey struct {
	Issuer  string
	Subject string
}

// Membership is local INTEGIN context. No Keycloak claim populates these fields.
type Membership struct {
	ActorID        string
	TenantID       string
	OrganizationID string
	WorkOrderRole  string
	Capabilities   []string
}

// Resolver returns exactly one local active membership or a local denial reason.
type Resolver interface {
	Resolve(ctx context.Context, principal PrincipalKey) (Membership, error)
}

func ValidatePrincipalKey(principal PrincipalKey) error {
	if strings.TrimSpace(principal.Issuer) == "" || strings.TrimSpace(principal.Subject) == "" {
		return ErrUnknownSubject
	}
	return nil
}
