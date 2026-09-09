package certificaterender

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/riverqueue/river"

	"integin/internal/domain/certificateauthority"
	domainrender "integin/internal/domain/certificaterender"
	"integin/internal/queue"
	"integin/internal/storage"
)

type mockRepo struct {
	jobInput domainrender.JobInput
	err      error
}

func (m *mockRepo) GetRenderJobInput(ctx context.Context, actor certificateauthority.ActorContext, certificateID string) (domainrender.JobInput, error) {
	if m.err != nil {
		return domainrender.JobInput{}, m.err
	}
	return m.jobInput, nil
}

func TestCertificateRenderWorker(t *testing.T) {
	store := storage.NewInMemoryStore()
	renderer := domainrender.NewDeterministicPDFRenderer()
	persistence := newMockPersistence()

	svc, err := NewRenderService(renderer, store, persistence)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	snapDigest := sha256.Sum256([]byte("snapshot-data-worker"))
	jobInput := domainrender.JobInput{
		TenantID:             "tenant-alpha",
		OrganizationID:       "org-1",
		CertificateID:        "cert-worker-1",
		CertificateNumber:    "CERT-2026-W1",
		Status:               "ISSUED",
		IssuedAt:             time.Now().UTC(),
		ExpiresAt:            time.Now().UTC().AddDate(1, 0, 0),
		PublicTokenDigestHex: "worker-digest-hex",
		SnapshotSHA256:       snapDigest[:],
	}

	repo := &mockRepo{jobInput: jobInput}
	worker := NewCertificateRenderWorker(svc, repo)

	riverJob := &river.Job[queue.CertificateRenderJobArgs]{
		Args: queue.CertificateRenderJobArgs{
			TenantID:       "tenant-alpha",
			OrganizationID: "org-1",
			CertificateID:  "cert-worker-1",
			InspectionID:   "insp-1",
			ActorUserID:    "user-w1",
		},
	}

	ctx := context.Background()
	if err := worker.Work(ctx, riverJob); err != nil {
		t.Fatalf("worker.Work failed: %v", err)
	}

	// Verify artifact stored in mock persistence and store
	actor := certificateauthority.ActorContext{
		TenantID:       "tenant-alpha",
		OrganizationID: "org-1",
		ActorID:        "user-w1",
	}
	rec, obj, err := svc.RetrieveArtifact(ctx, actor, "cert-worker-1")
	if err != nil {
		t.Fatalf("failed to retrieve rendered artifact: %v", err)
	}

	if rec.CertificateID != "cert-worker-1" {
		t.Fatalf("expected certificate id cert-worker-1, got %s", rec.CertificateID)
	}
	if len(obj.Data) == 0 {
		t.Fatal("expected non-empty rendered PDF data")
	}
}
