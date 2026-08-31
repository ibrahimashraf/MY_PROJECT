package billing

import (
	"context"
	"errors"
	"time"
)

type ServiceCharge struct {
	ID             string
	TenantID       string
	OrganizationID string
	WorkOrderID    string
	InspectionID   string
	ChargeType     string
	Description    string
	Quantity       float64
	UnitPrice      float64
	TotalPrice     float64
	Currency       string
	TaxRate        float64
	TaxAmount      float64
	CreatedBy      string
	CreatedAt      time.Time
}

type TimesheetAutoCapture struct {
	ID                  string
	TenantID            string
	OrganizationID      string
	WorkOrderID         string
	AssignmentID        string
	TechnicianID        string
	CheckInAt           time.Time
	CheckOutAt          time.Time
	AutoCalculatedHours float64
	ManualOverrideHours float64
	Status              string
	ApprovedBy          string
	ApprovedAt          time.Time
	CreatedAt           time.Time
}

type PartsCatalog struct {
	ID             string
	TenantID       string
	OrganizationID string
	Code           string
	Name           string
	Description    string
	UnitPrice      float64
	Currency       string
	IsActive       bool
	CreatedBy      string
	CreatedAt      time.Time
}

type Repository interface {
	CreateServiceCharge(ctx context.Context, actor ActorContext, c ServiceCharge) (ServiceCharge, error)
	GetServiceCharge(ctx context.Context, actor ActorContext, id string) (ServiceCharge, error)
	ListServiceCharges(ctx context.Context, actor ActorContext, workOrderID string) ([]ServiceCharge, error)
	UpdateServiceCharge(ctx context.Context, actor ActorContext, c ServiceCharge) error

	CreateTimesheetAutoCapture(ctx context.Context, actor ActorContext, t TimesheetAutoCapture) (TimesheetAutoCapture, error)
	GetTimesheetAutoCapture(ctx context.Context, actor ActorContext, id string) (TimesheetAutoCapture, error)
	ListTimesheetAutoCaptures(ctx context.Context, actor ActorContext, workOrderID, technicianID string) ([]TimesheetAutoCapture, error)
	UpdateTimesheetAutoCapture(ctx context.Context, actor ActorContext, t TimesheetAutoCapture) error

	CreatePartsCatalog(ctx context.Context, actor ActorContext, p PartsCatalog) (PartsCatalog, error)
	GetPartsCatalog(ctx context.Context, actor ActorContext, id string) (PartsCatalog, error)
	ListPartsCatalog(ctx context.Context, actor ActorContext) ([]PartsCatalog, error)
	UpdatePartsCatalog(ctx context.Context, actor ActorContext, p PartsCatalog) error
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

var ErrInvalidActor = errors.New("billing actor context is invalid")
