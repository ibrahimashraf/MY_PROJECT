package certificate

import (
	"context"
	"errors"
	"time"
)

type DocxTemplate struct {
	ID               string
	TenantID         string
	OrganizationID   string
	TemplateCode     string
	Version          int
	Title            string
	AssetType        string
	DocxContent      []byte
	DocxSHA256       []byte
	Status           string
	PageCount        int
	PageWidthPoints  float64
	PageHeightPoints float64
	CreatedBy        string
	ApprovedBy       string
	ApprovedAt       time.Time
	CreatedAt        time.Time
}

type CertificatePack struct {
	ID              string
	TenantID        string
	OrganizationID  string
	WorkOrderID     string
	PackSHA256      []byte
	ObjectKey       string
	Status          string
	RecipientEmails string
	SentAt          time.Time
	CreatedBy       string
	CreatedAt       time.Time
}

type CertificatePackItem struct {
	PackID        string
	CertificateID string
	SequenceNum   int
}

type Repository interface {
	CreateDocxTemplate(ctx context.Context, actor ActorContext, t DocxTemplate) (DocxTemplate, error)
	GetDocxTemplate(ctx context.Context, actor ActorContext, id string) (DocxTemplate, error)
	ListDocxTemplates(ctx context.Context, actor ActorContext, templateCode string) ([]DocxTemplate, error)
	UpdateDocxTemplate(ctx context.Context, actor ActorContext, t DocxTemplate) error
	DeleteDocxTemplate(ctx context.Context, actor ActorContext, id string) error

	CreatePack(ctx context.Context, actor ActorContext, p CertificatePack) (CertificatePack, error)
	GetPack(ctx context.Context, actor ActorContext, id string) (CertificatePack, error)
	ListPacks(ctx context.Context, actor ActorContext, workOrderID string) ([]CertificatePack, error)
	UpdatePack(ctx context.Context, actor ActorContext, p CertificatePack) error

	AddPackItem(ctx context.Context, actor ActorContext, item CertificatePackItem) error
	ListPackItems(ctx context.Context, actor ActorContext, packID string) ([]CertificatePackItem, error)
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

var ErrInvalidActor = errors.New("certificate_docx actor context is invalid")
