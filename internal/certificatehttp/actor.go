package certificatehttp

import (
	"context"
	"fmt"
	"strings"

	"integin/internal/domain/certificateauthority"
	"integin/internal/identity"
	"integin/internal/oidcauth"
)

type MembershipResolver interface {
	Resolve(context.Context, identity.PrincipalKey) (identity.Membership, error)
}
type LocalActorResolver struct{ Memberships MembershipResolver }

func (r LocalActorResolver) Resolve(ctx context.Context, principal oidcauth.Principal) (certificateauthority.ActorContext, error) {
	if r.Memberships == nil {
		return certificateauthority.ActorContext{}, fmt.Errorf("membership resolver is required")
	}
	membership, err := r.Memberships.Resolve(ctx, identity.PrincipalKey{Issuer: principal.Issuer, Subject: principal.Subject})
	if err != nil {
		return certificateauthority.ActorContext{}, err
	}
	capabilities := map[string]bool{}
	for _, capability := range membership.Capabilities {
		if value := strings.TrimSpace(capability); value != "" {
			capabilities[value] = true
		}
	}
	actor := certificateauthority.ActorContext{TenantID: strings.TrimSpace(membership.TenantID), OrganizationID: strings.TrimSpace(membership.OrganizationID), ActorID: strings.TrimSpace(membership.ActorID), Capabilities: capabilities}
	if actor.TenantID == "" || actor.OrganizationID == "" || actor.ActorID == "" || len(actor.Capabilities) == 0 {
		return certificateauthority.ActorContext{}, fmt.Errorf("invalid derived certificate actor")
	}
	return actor, nil
}
