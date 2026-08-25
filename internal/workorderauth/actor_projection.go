package workorderauth

import (
	"errors"
	"strings"

	"integin/internal/domain/workorder"
	"integin/internal/identity"
)

var ErrInvalidDerivedActor = errors.New("work-order derived actor is incomplete")

func ActorFromMembership(membership identity.Membership) (workorder.ActorContext, error) {
	actor := workorder.ActorContext{
		ActorID:        strings.TrimSpace(membership.ActorID),
		TenantID:       strings.TrimSpace(membership.TenantID),
		OrganizationID: strings.TrimSpace(membership.OrganizationID),
		Role:           strings.TrimSpace(membership.WorkOrderRole),
		Capabilities:   append([]string(nil), membership.Capabilities...),
	}
	if actor.Validate() != nil || len(actor.Capabilities) == 0 {
		return workorder.ActorContext{}, ErrInvalidDerivedActor
	}
	return actor, nil
}
