package integration

import (
	"context"
	"fmt"
	"io"

	"github.com/riverqueue/river"
	
	"integin/pkg/cfihos"
	"integin/internal/domain/equipment"
)

// CFIHOSExportArgs defines the arguments for a River job to generate a CFIHOS export.
type CFIHOSExportArgs struct {
	TenantID       string `json:"tenant_id"`
	OrganizationID string `json:"organization_id"`
	FacilityID     string `json:"facility_id"`
	ExportID       string `json:"export_id"`
	RequestedBy    string `json:"requested_by"`
}

// Kind implements river.JobArgs
func (CFIHOSExportArgs) Kind() string { return "cfihos_export" }

// StorageService defines where the generated CSV will be persisted.
type StorageService interface {
	PutFile(ctx context.Context, key string, data []byte) error
	PutStream(ctx context.Context, key string, data io.Reader) error
}

// CFIHOSExportWorker implements the ISO 18101 export engine via a River worker.
type CFIHOSExportWorker struct {
	river.WorkerDefaults[CFIHOSExportArgs]
	EquipmentRepo equipment.Repository
	Storage       StorageService
}

// Work extracts equipment taxonomy, maps it to CFIHOS tags, and serializes it.
func (w *CFIHOSExportWorker) Work(ctx context.Context, job *river.Job[CFIHOSExportArgs]) error {
	actor := equipment.ActorContext{
		TenantID:       job.Args.TenantID,
		OrganizationID: job.Args.OrganizationID,
		ActorID:        job.Args.RequestedBy,
	}

	// 1. Fetch Taxonomy from Equipment Repo (Paginated/Filtered)
	filter := equipment.AssetFilter{
		BranchID: job.Args.FacilityID,
	}
	assets, err := w.EquipmentRepo.ListAssets(ctx, actor, filter)
	if err != nil {
		return fmt.Errorf("failed to list equipment assets: %w", err)
	}

	var tags []cfihos.Tag
	var equip []cfihos.Equipment

	// 2. Map Assets to CFIHOS 
	for _, a := range assets {
		tags = append(tags, cfihos.Tag{
			ID:             a.AssetID,
			Facility:       job.Args.FacilityID,
			System:         a.AreaID,
			EquipmentClass: a.AssetType,
			Status:         a.LifecycleState,
		})

		equip = append(equip, cfihos.Equipment{
			ID:           a.ID,
			TagID:        a.AssetID,
			SerialNumber: a.SerialNumber,
			Manufacturer: "", 
			Model:        a.Description,
			Status:       a.LifecycleState,
		})
	}

	// 3. Serialize & Store via Stream (preventing OOM on enterprise datasets)
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		if err := cfihos.WriteCSV(pw, tags, equip); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to serialize CFIHOS dataset: %w", err))
		}
	}()

	key := fmt.Sprintf("cfihos/%s/%s/export_%s.csv", job.Args.TenantID, job.Args.OrganizationID, job.Args.ExportID)
	if err := w.Storage.PutStream(ctx, key, pr); err != nil {
		return fmt.Errorf("failed to upload CFIHOS export to storage: %w", err)
	}

	return nil
}
