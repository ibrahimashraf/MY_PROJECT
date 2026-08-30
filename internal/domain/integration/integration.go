package integration

import (
	"context"
	"errors"
	"time"
)

type IntegrationConfig struct {
	ID              string
	TenantID        string
	OrganizationID  string
	Provider        string
	Config          string
	Status          string
	LastSyncAt      time.Time
	ErrorLog        string
	CreatedBy       string
	CreatedAt       time.Time
}

type DataAPIToken struct {
	ID              string
	TenantID        string
	OrganizationID  string
	TokenHash       []byte
	Scopes          string
	ExpiresAt       time.Time
	RevokedAt       time.Time
	CreatedBy       string
	CreatedAt       time.Time
}

type SyncJob struct {
	ID              string
	TenantID        string
	OrganizationID  string
	Provider        string
	JobType         string
	Status          string
	Stats           string
	StartedAt       time.Time
	CompletedAt     time.Time
	ErrorLog        string
}

type Repository interface {
	CreateIntegrationConfig(ctx context.Context, actor ActorContext, c IntegrationConfig) (IntegrationConfig, error)
	GetIntegrationConfig(ctx context.Context, actor ActorContext, id string) (IntegrationConfig, error)
	ListIntegrationConfigs(ctx context.Context, actor ActorContext) ([]IntegrationConfig, error)
	UpdateIntegrationConfig(ctx context.Context, actor ActorContext, c IntegrationConfig) error
	DeleteIntegrationConfig(ctx context.Context, actor ActorContext, id string) error

	CreateDataAPIToken(ctx context.Context, actor ActorContext, t DataAPIToken) (DataAPIToken, error)
	GetDataAPITokenByHash(ctx context.Context, actor ActorContext, tokenHash []byte) (DataAPIToken, bool, error)
	ListDataAPITokens(ctx context.Context, actor ActorContext) ([]DataAPIToken, error)
	RevokeDataAPIToken(ctx context.Context, actor ActorContext, id string) error

	CreateSyncJob(ctx context.Context, actor ActorContext, j SyncJob) (SyncJob, error)
	GetSyncJob(ctx context.Context, actor ActorContext, id string) (SyncJob, error)
	ListSyncJobs(ctx context.Context, actor ActorContext, provider string) ([]SyncJob, error)
	UpdateSyncJob(ctx context.Context, actor ActorContext, j SyncJob) error
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

var ErrInvalidActor = errors.New("integration actor context is invalid")