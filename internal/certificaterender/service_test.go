package certificaterender

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"integin/internal/certificatepg"
	domainrender "integin/internal/domain/certificaterender"
	"integin/internal/domain/certificateauthority"
	"integin/internal/storage"
)

type mockPersistence struct {
	records map[string]certificatepg.ArtifactRecord
}

func newMockPersistence() *mockPersistence {
	return &mockPersistence{records: make(map[string]certificatepg.ArtifactRecord)}
}

func (m *mockPersistence) RecordArtifact(ctx context.Context, actor certificateauthority.ActorContext, record certificatepg.ArtifactRecord) error {
	key := record.TenantID + ":" + record.OrganizationID + ":" + record.CertificateID + ":" + record.ArtifactType
	m.records[key] = record
	return nil
}

func (m *mockPersistence) GetArtifact(ctx context.Context, actor certificateauthority.ActorContext, certificateID, artifactType string) (certificatepg.ArtifactRecord, error) {
	key := actor.TenantID + ":" + actor.OrganizationID + ":" + certificateID + ":" + artifactType
	rec, ok := m.records[key]
	if !ok {
		return certificatepg.ArtifactRecord{}, errors.New("not found")
	}
	return rec, nil
}

func TestRenderServicePipeline(t *testing.T) {
	store := storage.NewInMemoryStore()
	renderer := domainrender.NewDeterministicPDFRenderer()
	persistence := newMockPersistence()

	svc, err := NewRenderService(renderer, store, persistence)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	snapDigest := sha256.Sum256([]byte("snapshot-data"))
	actor := certificateauthority.ActorContext{
		TenantID:       "tenant-alpha",
		OrganizationID: "org-1",
		ActorID:        "user-100",
		Capabilities:   map[string]bool{"certificate.view": true},
	}

	jobInput := domainrender.JobInput{
		TenantID:             "tenant-alpha",
		OrganizationID:       "org-1",
		CertificateID:        "cert-999",
		CertificateNumber:    "CERT-2026-999",
		Status:               "ISSUED",
		IssuedAt:             time.Now().UTC(),
		ExpiresAt:            time.Now().UTC().AddDate(1, 0, 0),
		PublicTokenDigestHex: "token-digest",
		SnapshotSHA256:       snapDigest[:],
	}

	// 1. Execute render job
	record, err := svc.ExecuteRenderJob(context.Background(), actor, jobInput)
	if err != nil {
		t.Fatalf("execute render job failed: %v", err)
	}

	if record.CertificateID != "cert-999" {
		t.Fatalf("unexpected certificate id in record: %s", record.CertificateID)
	}
	if record.ObjectKey != "tenants/tenant-alpha/certs/cert-999/certificate.pdf" {
		t.Fatalf("unexpected object key: %s", record.ObjectKey)
	}

	// 2. Retrieve artifact
	retrievedRecord, retrievedObj, err := svc.RetrieveArtifact(context.Background(), actor, "cert-999")
	if err != nil {
		t.Fatalf("retrieve artifact failed: %v", err)
	}

	if retrievedRecord.ID != record.ID {
		t.Fatalf("record mismatch: %s vs %s", retrievedRecord.ID, record.ID)
	}
	if retrievedObj.ContentType != "application/pdf" {
		t.Fatalf("unexpected retrieved content type: %s", retrievedObj.ContentType)
	}
	if len(retrievedObj.Data) == 0 {
		t.Fatal("retrieved object data is empty")
	}

	// 3. Unauthorized actor (different tenant) should fail
	foreignActor := certificateauthority.ActorContext{
		TenantID:       "tenant-beta",
		OrganizationID: "org-1",
		ActorID:        "user-200",
	}
	_, _, err = svc.RetrieveArtifact(context.Background(), foreignActor, "cert-999")
	if err == nil {
		t.Fatal("expected error when retrieving artifact across tenants")
	}
}
