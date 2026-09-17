package certificaterender

import (
	"context"
	"fmt"
	
	"integin/internal/domain/compliance"
	domainrender "integin/internal/domain/certificaterender"
)

type EquipmentDataSource interface {
	GetAssetContext(ctx context.Context, tenantID, assetID string) (map[string]any, error)
}

type DossierRenderer struct {
	PDFRenderer domainrender.Renderer
	Data        EquipmentDataSource
}

func (r *DossierRenderer) RenderDossier(ctx context.Context, args compliance.DossierExportArgs) ([]byte, error) {
	// Query the actual asset details, inspection records, and fatigue limits
	var assetData map[string]any
	if r.Data != nil {
		data, err := r.Data.GetAssetContext(ctx, args.TenantID, args.AssetID)
		if err != nil {
			return nil, fmt.Errorf("failed to load asset context: %w", err)
		}
		assetData = data
	} else {
		assetData = map[string]any{"warning": "no data source bound"}
	}

	input := domainrender.JobInput{
		TenantID:       args.TenantID,
		OrganizationID: args.OrganizationID,
		CertificateID:  args.CertificateID,
		PublicBindings: map[string]any{
			"dossier_type": args.Dossier,
			"asset_data":   assetData,
		},
	}

	output, err := r.PDFRenderer.Render(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to render %s dossier for asset %s: %w", args.Dossier, args.AssetID, err)
	}

	return output.PDFData, nil
}
