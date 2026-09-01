package license

import (
	"context"
	"time"
)

type ActorContext struct {
	TenantID       string
	OrganizationID string
	ActorID        string
}

func (a ActorContext) Validate() error {
	if a.TenantID == "" || a.OrganizationID == "" || a.ActorID == "" {
		return ErrInvalidIdentity
	}
	return nil
}

type Repository interface {
	GetLicense(ctx context.Context, actor ActorContext, licenseID string) (License, error)
	GetActiveLicense(ctx context.Context, actor ActorContext) (License, bool, error)
	CreateLicense(ctx context.Context, actor ActorContext, lic License) (License, error)
	UpdateLicense(ctx context.Context, actor ActorContext, lic License) (License, error)
	RecordAudit(ctx context.Context, actor ActorContext, entry AuditEntry) error
}

type TransactionRunner interface {
	WithinTransaction(ctx context.Context, actor ActorContext, fn func(context.Context, Repository) error) error
}

type ServiceDependencies struct {
	Repository   Repository
	Transactions TransactionRunner
}

func (d ServiceDependencies) Validate() error {
	if d.Repository == nil || d.Transactions == nil {
		return ErrInvalidIdentity
	}
	return nil
}

type Service interface {
	ValidateLicense(ctx context.Context, actor ActorContext, at time.Time) (Validation, error)
	IssueLicense(ctx context.Context, actor ActorContext, lic License) (License, error)
	RenewLicense(ctx context.Context, actor ActorContext, licenseID string, newExpiresAt time.Time) (License, error)
	RevokeLicense(ctx context.Context, actor ActorContext, licenseID string) error
}
