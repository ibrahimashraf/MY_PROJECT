package certificaterender

import (
	"context"
	"fmt"
	
	"integin/internal/domain/compliance"
	domainrender "integin/internal/domain/certificaterender"
)

type DossierRenderer struct {
	PDFRenderer domainrender.Renderer
}

func (r *DossierRenderer) RenderDossier(ctx context.Context, args compliance.DossierExportArgs) ([]byte, error) {
	// A real implementation would query the asset details, inspection records,
	// and fatigue limits, then map them into the domainrender.JobInput payload.
	input := domainrender.JobInput{
		TenantID:       args.TenantID,
		OrganizationID: args.OrganizationID,
		CertificateID:  args.CertificateID,
		// Map dossier type into CertificateTemplateID or similar field
	}

	output, err := r.PDFRenderer.Render(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to render %s dossier for asset %s: %w", args.Dossier, args.AssetID, err)
	}

	return output.PDFData, nil
}
