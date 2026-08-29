package certificatetemplate

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrImmutableConflict = errors.New("certificate template conflicts with immutable version")

type ActorContext struct {
	TenantID       string
	OrganizationID string
	ActorID        string
}

func (a ActorContext) Validate() error {
	if strings.TrimSpace(a.TenantID) == "" || strings.TrimSpace(a.OrganizationID) == "" || strings.TrimSpace(a.ActorID) == "" {
		return errors.New("tenant, organization, and actor are required")
	}
	return nil
}

type DefinitionRecord struct {
	Definition
	CreatedBy  string
	CreatedAt  time.Time
	ApprovedBy string
	ApprovedAt time.Time
}

type Repository interface {
	RegisterDraft(context.Context, ActorContext, Definition) (DefinitionRecord, bool, error)
	Approve(context.Context, ActorContext, string, int, time.Time) (DefinitionRecord, error)
	Get(context.Context, ActorContext, string, int) (DefinitionRecord, error)
}
