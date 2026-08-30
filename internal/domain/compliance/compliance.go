package compliance

import (
	"context"
	"errors"
	"time"
)

type HSENotification struct {
	ID               string
	TenantID         string
	OrganizationID   string
	InspectionID     string
	DefectCode       string
	DefectSeverity   string
	HSEReference     string
	ReportPayload    string
	Status           string
	SubmittedBy      string
	SubmittedAt      time.Time
	AcknowledgedAt   time.Time
	AcknowledgedBy   string
	CreatedBy        string
	CreatedAt        time.Time
}

type CSVExportJob struct {
	ID              string
	TenantID        string
	OrganizationID  string
	ExportType      string
	Filters         string
	Status          string
	FileKey         string
	RowCount        int64
	ExpiresAt       time.Time
	RequestedBy     string
	RequestedAt     time.Time
	CompletedAt     time.Time
}

type ExportTemplate struct {
	ID              string
	TenantID        string
	OrganizationID  string
	ExportType      string
	Name            string
	Columns         string
	CreatedBy       string
	CreatedAt       time.Time
}

type Repository interface {
	CreateHSENotification(ctx context.Context, actor ActorContext, n HSENotification) (HSENotification, error)
	GetHSENotification(ctx context.Context, actor ActorContext, id string) (HSENotification, error)
	ListHSENotifications(ctx context.Context, actor ActorContext, inspectionID string) ([]HSENotification, error)
	UpdateHSENotification(ctx context.Context, actor ActorContext, n HSENotification) error

	CreateCSVExportJob(ctx context.Context, actor ActorContext, j CSVExportJob) (CSVExportJob, error)
	GetCSVExportJob(ctx context.Context, actor ActorContext, id string) (CSVExportJob, error)
	ListCSVExportJobs(ctx context.Context, actor ActorContext, status string) ([]CSVExportJob, error)
	UpdateCSVExportJob(ctx context.Context, actor ActorContext, j CSVExportJob) error

	CreateExportTemplate(ctx context.Context, actor ActorContext, t ExportTemplate) (ExportTemplate, error)
	GetExportTemplate(ctx context.Context, actor ActorContext, id string) (ExportTemplate, error)
	ListExportTemplates(ctx context.Context, actor ActorContext, exportType string) ([]ExportTemplate, error)
	UpdateExportTemplate(ctx context.Context, actor ActorContext, t ExportTemplate) error
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

var ErrInvalidActor = errors.New("compliance actor context is invalid")