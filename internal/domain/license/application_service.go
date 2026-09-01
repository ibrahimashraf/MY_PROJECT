package license

import (
	"context"
	"fmt"
	"time"
)

type applicationService struct {
	deps ServiceDependencies
}

func NewService(deps ServiceDependencies) (Service, error) {
	if err := deps.Validate(); err != nil {
		return nil, err
	}
	return applicationService{deps: deps}, nil
}

func (s applicationService) ValidateLicense(ctx context.Context, actor ActorContext, at time.Time) (Validation, error) {
	var zero Validation
	if err := actor.Validate(); err != nil {
		return zero, err
	}
	lic, ok, err := s.deps.Repository.GetActiveLicense(ctx, actor)
	if err != nil {
		return zero, err
	}
	if !ok {
		return Validation{Valid: false}, nil
	}
	if lic.IsExpired(at) {
		return Validation{Valid: false, Tier: lic.Tier, Status: StatusExpired}, nil
	}
	return Validation{
		Valid:                  true,
		Tier:                   lic.Tier,
		Status:                 lic.Status,
		ExpiresAt:              lic.ExpiresAt,
		MaxInspectors:          lic.MaxInspectors,
		MaxInspectionsPerMonth: lic.MaxInspectionsPerMonth,
		Features:               lic.Features,
	}, nil
}

func (s applicationService) IssueLicense(ctx context.Context, actor ActorContext, lic License) (License, error) {
	var zero License
	if err := actor.Validate(); err != nil {
		return zero, err
	}
	if err := lic.Validate(); err != nil {
		return zero, err
	}
	var result License
	err := s.deps.Transactions.WithinTransaction(ctx, actor, func(txCtx context.Context, repo Repository) error {
		var err error
		result, err = repo.CreateLicense(txCtx, actor, lic)
		if err != nil {
			return err
		}
		return repo.RecordAudit(txCtx, actor, AuditEntry{
			ID:             fmt.Sprintf("audit:%s:%d", result.ID, time.Now().UnixNano()),
			TenantID:       actor.TenantID,
			OrganizationID: actor.OrganizationID,
			LicenseID:      result.ID,
			Action:         ActionIssued,
			ActorID:        actor.ActorID,
			OccurredAt:     time.Now(),
		})
	})
	return result, err
}

func (s applicationService) RenewLicense(ctx context.Context, actor ActorContext, licenseID string, newExpiresAt time.Time) (License, error) {
	var zero License
	if err := actor.Validate(); err != nil {
		return zero, err
	}
	var result License
	err := s.deps.Transactions.WithinTransaction(ctx, actor, func(txCtx context.Context, repo Repository) error {
		lic, err := repo.GetLicense(txCtx, actor, licenseID)
		if err != nil {
			return err
		}
		lic.ExpiresAt = &newExpiresAt
		lic.Status = StatusActive
		result, err = repo.UpdateLicense(txCtx, actor, lic)
		if err != nil {
			return err
		}
		return repo.RecordAudit(txCtx, actor, AuditEntry{
			ID:             fmt.Sprintf("audit:%s:%d", result.ID, time.Now().UnixNano()),
			TenantID:       actor.TenantID,
			OrganizationID: actor.OrganizationID,
			LicenseID:      result.ID,
			Action:         ActionRenewed,
			ActorID:        actor.ActorID,
			OccurredAt:     time.Now(),
		})
	})
	return result, err
}

func (s applicationService) RevokeLicense(ctx context.Context, actor ActorContext, licenseID string) error {
	if err := actor.Validate(); err != nil {
		return err
	}
	return s.deps.Transactions.WithinTransaction(ctx, actor, func(txCtx context.Context, repo Repository) error {
		lic, err := repo.GetLicense(txCtx, actor, licenseID)
		if err != nil {
			return err
		}
		lic.Status = StatusRevoked
		if _, err := repo.UpdateLicense(txCtx, actor, lic); err != nil {
			return err
		}
		return repo.RecordAudit(txCtx, actor, AuditEntry{
			ID:             fmt.Sprintf("audit:%s:%d", lic.ID, time.Now().UnixNano()),
			TenantID:       actor.TenantID,
			OrganizationID: actor.OrganizationID,
			LicenseID:      lic.ID,
			Action:         ActionRevoked,
			ActorID:        actor.ActorID,
			OccurredAt:     time.Now(),
		})
	})
}
