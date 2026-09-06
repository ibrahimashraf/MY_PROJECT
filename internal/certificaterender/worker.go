package certificaterender

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/riverqueue/river"

	domainrender "integin/internal/domain/certificaterender"
	"integin/internal/domain/certificateauthority"
	"integin/internal/queue"
)

// CertificateRepository defines the query contract required by the render worker to hydrate
// the certificate snapshot and template cells into a sealed JobInput.
type CertificateRepository interface {
	GetRenderJobInput(ctx context.Context, actor certificateauthority.ActorContext, certificateID string) (domainrender.JobInput, error)
}

// CertificateRenderWorker processes queue.CertificateRenderJobArgs jobs asynchronously.
// It reconstructs the sealed JobInput and invokes RenderService.ExecuteRenderJob().
type CertificateRenderWorker struct {
	river.WorkerDefaults[queue.CertificateRenderJobArgs]
	Service    *RenderService
	Repository CertificateRepository
	Now        func() time.Time
}

func NewCertificateRenderWorker(svc *RenderService, repo CertificateRepository) *CertificateRenderWorker {
	return &CertificateRenderWorker{
		Service:    svc,
		Repository: repo,
		Now:        time.Now,
	}
}

// Work implements river.Worker[queue.CertificateRenderJobArgs].
func (w *CertificateRenderWorker) Work(ctx context.Context, job *river.Job[queue.CertificateRenderJobArgs]) error {
	if w.Service == nil {
		return errors.New("render service is required")
	}
	if w.Repository == nil {
		return errors.New("certificate repository is required")
	}
	args := job.Args
	if args.CertificateID == "" || args.TenantID == "" || args.OrganizationID == "" {
		return errors.New("job args missing required certificate identity")
	}

	actor := certificateauthority.ActorContext{
		TenantID:       args.TenantID,
		OrganizationID: args.OrganizationID,
		ActorID:        args.ActorUserID,
	}

	jobInput, err := w.Repository.GetRenderJobInput(ctx, actor, args.CertificateID)
	if err != nil {
		return fmt.Errorf("failed to load render job input: %w", err)
	}

	_, err = w.Service.ExecuteRenderJob(ctx, actor, jobInput)
	if err != nil {
		return fmt.Errorf("execute render job failed: %w", err)
	}

	return nil
}
