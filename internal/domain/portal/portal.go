package portal

import (
	"context"
	"errors"
	"time"
)

type ClientPortalDomain struct {
	ID             string
	TenantID       string
	OrganizationID string
	Domain         string
	CertARN        string
	Status         string
	VerifiedAt     time.Time
	CreatedBy      string
	CreatedAt      time.Time
}

type ClientPortalACL struct {
	ID             string
	TenantID       string
	OrganizationID string
	ClientID       string
	UserEmail      string
	Permissions    string
	GrantedBy      string
	CreatedAt      time.Time
}

type QuickLink struct {
	ID             string
	TenantID       string
	OrganizationID string
	Token          string
	TargetType     string
	TargetID       string
	ExpiresAt      time.Time
	AccessCount    int
	CreatedBy      string
	CreatedAt      time.Time
}

type ClientAuditEntry struct {
	ID             string
	TenantID       string
	OrganizationID string
	ClientID       string
	ActorEmail     string
	Action         string
	TargetType     string
	TargetID       string
	Metadata       string
	OccurredAt     time.Time
}

type Repository interface {
	CreateDomain(ctx context.Context, actor ActorContext, d ClientPortalDomain) (ClientPortalDomain, error)
	GetDomain(ctx context.Context, actor ActorContext, id string) (ClientPortalDomain, error)
	ListDomains(ctx context.Context, actor ActorContext) ([]ClientPortalDomain, error)
	UpdateDomain(ctx context.Context, actor ActorContext, d ClientPortalDomain) error
	DeleteDomain(ctx context.Context, actor ActorContext, id string) error

	CreateACL(ctx context.Context, actor ActorContext, a ClientPortalACL) (ClientPortalACL, error)
	GetACL(ctx context.Context, actor ActorContext, id string) (ClientPortalACL, error)
	ListACLs(ctx context.Context, actor ActorContext, clientID string) ([]ClientPortalACL, error)
	UpdateACL(ctx context.Context, actor ActorContext, a ClientPortalACL) error
	DeleteACL(ctx context.Context, actor ActorContext, id string) error

	CreateQuickLink(ctx context.Context, actor ActorContext, q QuickLink) (QuickLink, error)
	GetQuickLink(ctx context.Context, actor ActorContext, token string) (QuickLink, error)
	UpdateQuickLink(ctx context.Context, actor ActorContext, q QuickLink) error

	CreateAuditEntry(ctx context.Context, actor ActorContext, e ClientAuditEntry) (ClientAuditEntry, error)
	ListAuditEntries(ctx context.Context, actor ActorContext, clientID string, from, to time.Time) ([]ClientAuditEntry, error)
}

type ActorContext struct {
	TenantID       string
	OrganizationID string
	ActorID        string
}

func (a ActorContext) Validate() error {
	if a.TenantID == "" || a.OrganizationID == "" || a.ActorID == "" {
		return ErrInvalidActor
	}
	return nil
}

var ErrInvalidActor = errors.New("portal actor context is invalid")
