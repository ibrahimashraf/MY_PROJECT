package compliance

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"
)

// DossierType dictates the regulatory format of the compliance pack.
type DossierType string

const (
	DossierOSHA        DossierType = "OSHA_1910_179"
	DossierLOLER       DossierType = "LOLER_1998"
	DossierDNV         DossierType = "DNV_MARINE_WARRANTY"
	DossierSaudiAramco DossierType = "ARAMCO_GI_7_027"
	DossierADNOC       DossierType = "ADNOC_COP_HSE_038"
	DossierSASO        DossierType = "SASO_SABER_PCOC"
)

// DossierExportArgs defines arguments for building a statutory compliance pack.
type DossierExportArgs struct {
	TenantID       string      `json:"tenant_id"`
	OrganizationID string      `json:"organization_id"`
	AssetID        string      `json:"asset_id"`
	CertificateID  string      `json:"certificate_id"`
	Dossier        DossierType `json:"dossier_type"`
}

func (DossierExportArgs) Kind() string { return "dossier_export" }

type StorageService interface {
	PutFile(ctx context.Context, key string, data []byte) error
}

type DossierRenderer interface {
	RenderDossier(ctx context.Context, args DossierExportArgs) ([]byte, error)
}

// DossierExportWorker compiles a signed, immutable regulatory dossier.
type DossierExportWorker struct {
	river.WorkerDefaults[DossierExportArgs]
	Renderer DossierRenderer
	Storage  StorageService
}

func (w *DossierExportWorker) Work(ctx context.Context, job *river.Job[DossierExportArgs]) error {
	// 1. Render Dossier PDF
	pdfBytes, err := w.Renderer.RenderDossier(ctx, job.Args)
	if err != nil {
		return fmt.Errorf("failed to render dossier %s: %w", job.Args.Dossier, err)
	}

	// 2. Upload to cold storage / evidence bucket
	key := fmt.Sprintf("dossiers/%s/%s/%s/%s.pdf",
		job.Args.TenantID,
		job.Args.OrganizationID,
		string(job.Args.Dossier),
		job.Args.CertificateID)

	if err := w.Storage.PutFile(ctx, key, pdfBytes); err != nil {
		return fmt.Errorf("failed to store dossier: %w", err)
	}

	return nil
}
