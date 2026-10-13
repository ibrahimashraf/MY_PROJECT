package inspection

import (
	"context"
	"errors"
	"time"
)

type MultiInspectBatch struct {
	ID             string
	TenantID       string
	OrganizationID string
	WorkOrderID    string
	AssignmentID   string
	BatchCode      string
	Status         string
	TotalItems     int
	CompletedItems int
	CreatedBy      string
	CreatedAt      time.Time
}

type MultiInspectItem struct {
	ID             string
	TenantID       string
	OrganizationID string
	BatchID        string
	AssetID        string
	InspectionID   string
	SequenceNum    int
	Status         string
	DecisionAt     time.Time
	DecidedBy      string
	CreatedAt      time.Time
}

type InspectTemplatePreset struct {
	ID              string
	TenantID        string
	OrganizationID  string
	EquipmentTypeID string
	Name            string
	PassFailMode    bool
	ComponentChecks string
	CreatedBy       string
	CreatedAt       time.Time
}

type Repository interface {
	CreateBatch(ctx context.Context, actor ActorContext, b MultiInspectBatch) (MultiInspectBatch, error)
	GetBatch(ctx context.Context, actor ActorContext, id string) (MultiInspectBatch, error)
	ListBatches(ctx context.Context, actor ActorContext, workOrderID string) ([]MultiInspectBatch, error)
	UpdateBatch(ctx context.Context, actor ActorContext, b MultiInspectBatch) error
	DeleteBatch(ctx context.Context, actor ActorContext, id string) error

	CreateItem(ctx context.Context, actor ActorContext, i MultiInspectItem) (MultiInspectItem, error)
	GetItem(ctx context.Context, actor ActorContext, id string) (MultiInspectItem, error)
	ListItems(ctx context.Context, actor ActorContext, batchID string) ([]MultiInspectItem, error)
	UpdateItem(ctx context.Context, actor ActorContext, i MultiInspectItem) error

	CreateTemplatePreset(ctx context.Context, actor ActorContext, p InspectTemplatePreset) (InspectTemplatePreset, error)
	GetTemplatePreset(ctx context.Context, actor ActorContext, id string) (InspectTemplatePreset, error)
	ListTemplatePresets(ctx context.Context, actor ActorContext, equipmentTypeID string) ([]InspectTemplatePreset, error)
	UpdateTemplatePreset(ctx context.Context, actor ActorContext, p InspectTemplatePreset) error
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

var ErrInvalidActor = errors.New("multi_inspect actor context is invalid")
