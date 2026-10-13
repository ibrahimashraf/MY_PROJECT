package settings

import (
	"context"
	"errors"
	"time"
)

type TenantSetting struct {
	ID             string
	TenantID       string
	OrganizationID string
	SettingKey     string
	SettingValue   string
	Scope          string
	IsEditable     bool
	Description    string
	CreatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AuditTrailExport struct {
	ID             string
	TenantID       string
	OrganizationID string
	ExportType     string
	DateFrom       time.Time
	DateTo         time.Time
	Filters        string
	Status         string
	FileKey        string
	RowCount       int64
	ChecksumSHA256 []byte
	RequestedBy    string
	RequestedAt    time.Time
	CompletedAt    time.Time
}

type Repository interface {
	GetTenantSetting(ctx context.Context, actor ActorContext, key, scope string) (TenantSetting, error)
	ListTenantSettings(ctx context.Context, actor ActorContext, scope string) ([]TenantSetting, error)
	UpdateTenantSetting(ctx context.Context, actor ActorContext, s TenantSetting) error

	CreateAuditTrailExport(ctx context.Context, actor ActorContext, e AuditTrailExport) (AuditTrailExport, error)
	GetAuditTrailExport(ctx context.Context, actor ActorContext, id string) (AuditTrailExport, error)
	ListAuditTrailExports(ctx context.Context, actor ActorContext, status string) ([]AuditTrailExport, error)
	UpdateAuditTrailExport(ctx context.Context, actor ActorContext, e AuditTrailExport) error
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

var ErrInvalidActor = errors.New("settings actor context is invalid")
