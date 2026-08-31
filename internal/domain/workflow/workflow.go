package workflow

import (
	"context"
	"errors"
	"time"
)

type JobLinkageConfig struct {
	ID                      string
	TenantID                string
	OrganizationID          string
	RequireJobForInspection bool
	CreatedBy               string
	CreatedAt               time.Time
}

type FailedInspectionQueue struct {
	ID                string
	TenantID          string
	OrganizationID    string
	InspectionID      string
	WorkOrderID       string
	AssignmentID      string
	FailureReason     string
	SuppressionReason string
	ReviewStatus      string
	ReviewedBy        string
	ReviewedAt        time.Time
	CreatedBy         string
	CreatedAt         time.Time
}

type Repository interface {
	GetJobLinkageConfig(ctx context.Context, actor ActorContext) (JobLinkageConfig, error)
	SetJobLinkageConfig(ctx context.Context, actor ActorContext, c JobLinkageConfig) error

	CreateFailedQueueEntry(ctx context.Context, actor ActorContext, f FailedInspectionQueue) (FailedInspectionQueue, error)
	GetFailedQueueEntry(ctx context.Context, actor ActorContext, id string) (FailedInspectionQueue, error)
	ListFailedQueueEntries(ctx context.Context, actor ActorContext, reviewStatus string) ([]FailedInspectionQueue, error)
	UpdateFailedQueueEntry(ctx context.Context, actor ActorContext, f FailedInspectionQueue) error
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

var ErrInvalidActor = errors.New("workflow actor context is invalid")
