package dpp

import (
	"context"
	"errors"
	"time"
)

type ProductPassportDPP struct {
	ID                string
	TenantID          string
	OrganizationID    string
	AssetID           string
	SerialNumber      string
	BatchNumber       string
	ManufacturerID    string
	DPPStatus         string
	DPPVersion        int
	DPPHash           []byte
	AssignmentPayload string
	UpdatePayload     string
	UsePayload        string
	DisposalPayload   string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type RegulatoryMonitor struct {
	ID               string
	TenantID         string
	OrganizationID   string
	Source           string
	RegulationCode   string
	Title            string
	EffectiveDate    time.Time
	Summary          string
	ImpactAssessment string
	Status           string
	LastCheckedAt    time.Time
	CheckedBy        string
}

type ComplianceAction struct {
	ID             string
	TenantID       string
	OrganizationID string
	MonitorID      string
	ActionType     string
	Status         string
	Assignee       string
	DueDate        time.Time
	CompletedAt    time.Time
	EvidenceRefs   string
}

type Repository interface {
	CreateProductPassportDPP(ctx context.Context, actor ActorContext, d ProductPassportDPP) (ProductPassportDPP, error)
	GetProductPassportDPP(ctx context.Context, actor ActorContext, id string) (ProductPassportDPP, error)
	GetProductPassportDPPByAsset(ctx context.Context, actor ActorContext, assetID string) (ProductPassportDPP, bool, error)
	UpdateProductPassportDPP(ctx context.Context, actor ActorContext, d ProductPassportDPP) error

	CreateRegulatoryMonitor(ctx context.Context, actor ActorContext, r RegulatoryMonitor) (RegulatoryMonitor, error)
	GetRegulatoryMonitor(ctx context.Context, actor ActorContext, id string) (RegulatoryMonitor, error)
	ListRegulatoryMonitors(ctx context.Context, actor ActorContext, source, status string) ([]RegulatoryMonitor, error)
	UpdateRegulatoryMonitor(ctx context.Context, actor ActorContext, r RegulatoryMonitor) error

	CreateComplianceAction(ctx context.Context, actor ActorContext, a ComplianceAction) (ComplianceAction, error)
	GetComplianceAction(ctx context.Context, actor ActorContext, id string) (ComplianceAction, error)
	ListComplianceActions(ctx context.Context, actor ActorContext, monitorID, status string) ([]ComplianceAction, error)
	UpdateComplianceAction(ctx context.Context, actor ActorContext, a ComplianceAction) error
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

var ErrInvalidActor = errors.New("dpp actor context is invalid")
