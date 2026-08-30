package comment

import (
	"context"
	"errors"
	"time"
)

type CommentLibraryEntry struct {
	ID               string
	TenantID         string
	OrganizationID   string
	Category         string
	EquipmentTypeID  string
	Code             string
	Text             string
	TrafficLight     string
	IsActive         bool
	DisplayOrder     int
	CreatedBy        string
	CreatedAt        time.Time
}

type InspectionComment struct {
	ID                  string
	TenantID            string
	OrganizationID      string
	InspectionID        string
	QuestionCode        string
	CommentLibraryID    string
	CustomText          string
	TrafficLight        string
	CreatedBy           string
	CreatedAt           time.Time
}

type Repository interface {
	CreateLibraryEntry(ctx context.Context, actor ActorContext, c CommentLibraryEntry) (CommentLibraryEntry, error)
	GetLibraryEntry(ctx context.Context, actor ActorContext, id string) (CommentLibraryEntry, error)
	ListLibraryEntries(ctx context.Context, actor ActorContext, category, equipmentTypeID string) ([]CommentLibraryEntry, error)
	UpdateLibraryEntry(ctx context.Context, actor ActorContext, c CommentLibraryEntry) error
	DeleteLibraryEntry(ctx context.Context, actor ActorContext, id string) error

	CreateInspectionComment(ctx context.Context, actor ActorContext, c InspectionComment) (InspectionComment, error)
	GetInspectionComment(ctx context.Context, actor ActorContext, id string) (InspectionComment, error)
	ListInspectionComments(ctx context.Context, actor ActorContext, inspectionID, questionCode string) ([]InspectionComment, error)
	UpdateInspectionComment(ctx context.Context, actor ActorContext, c InspectionComment) error
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

var ErrInvalidActor = errors.New("comment actor context is invalid")