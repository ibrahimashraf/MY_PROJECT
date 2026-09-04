package certificaterender

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"integin/internal/certificatepg"
	domainrender "integin/internal/domain/certificaterender"
	"integin/internal/domain/certificateauthority"
	"integin/internal/storage"
)

// ArtifactPersistence defines the database methods needed to record and query artifact metadata.
type ArtifactPersistence interface {
	RecordArtifact(ctx context.Context, actor certificateauthority.ActorContext, record certificatepg.ArtifactRecord) error
	GetArtifact(ctx context.Context, actor certificateauthority.ActorContext, certificateID, artifactType string) (certificatepg.ArtifactRecord, error)
}

// RenderService coordinates the sealed render-job pipeline and tenant-isolated storage handoff.
type RenderService struct {
	Renderer    domainrender.Renderer
	Storage     storage.Store
	Persistence ArtifactPersistence
}

func NewRenderService(renderer domainrender.Renderer, store storage.Store, persistence ArtifactPersistence) (*RenderService, error) {
	if renderer == nil {
		return nil, errors.New("renderer is required")
	}
	if store == nil {
		return nil, errors.New("storage is required")
	}
	if persistence == nil {
		return nil, errors.New("artifact persistence is required")
	}
	return &RenderService{
		Renderer:    renderer,
		Storage:     store,
		Persistence: persistence,
	}, nil
}

// ExecuteRenderJob executes non-authoritative rendering, securely saves the PDF to S3/RustFS under tenant namespace,
// and records the metadata in the database under tenant RLS.
func (s *RenderService) ExecuteRenderJob(ctx context.Context, actor certificateauthority.ActorContext, input domainrender.JobInput) (certificatepg.ArtifactRecord, error) {
	if actor.TenantID != input.TenantID || actor.OrganizationID != input.OrganizationID {
		return certificatepg.ArtifactRecord{}, errors.New("actor tenant/organization does not match render job input")
	}

	// 1. Run non-authoritative renderer
	output, err := s.Renderer.Render(ctx, input)
	if err != nil {
		return certificatepg.ArtifactRecord{}, fmt.Errorf("rendering failed: %w", err)
	}

	// 2. Put object to private tenant-isolated storage
	meta := map[string]string{
		"certificate-id":   input.CertificateID,
		"snapshot-sha256":  hex.EncodeToString(input.SnapshotSHA256),
		"artifact-sha256":  hex.EncodeToString(output.ArtifactSHA256),
		"renderer-version": output.RendererVersion,
	}

	obj := storage.Object{
		Key:         output.ObjectKey,
		ContentType: output.ContentType,
		Data:        output.PDFData,
		Metadata:    meta,
	}

	if err := s.Storage.Put(ctx, obj); err != nil {
		return certificatepg.ArtifactRecord{}, fmt.Errorf("storage put failed: %w", err)
	}

	// 3. Persist artifact record in DB
	artifactID, err := randomHex(16)
	if err != nil {
		return certificatepg.ArtifactRecord{}, err
	}

	record := certificatepg.ArtifactRecord{
		ID:              "art-" + artifactID,
		TenantID:        actor.TenantID,
		OrganizationID:  actor.OrganizationID,
		CertificateID:   input.CertificateID,
		ArtifactType:    output.ArtifactType,
		ObjectKey:       output.ObjectKey,
		ContentType:     output.ContentType,
		ByteSize:        output.ByteSize,
		ArtifactSHA256:  output.ArtifactSHA256,
		SnapshotSHA256:  input.SnapshotSHA256,
		RendererVersion: output.RendererVersion,
		CreatedBy:       actor.ActorID,
		CreatedAt:       time.Now().UTC(),
	}

	if err := s.Persistence.RecordArtifact(ctx, actor, record); err != nil {
		return certificatepg.ArtifactRecord{}, fmt.Errorf("recording artifact metadata failed: %w", err)
	}

	return record, nil
}

// RetrieveArtifact fetches artifact metadata and streams the stored PDF payload.
func (s *RenderService) RetrieveArtifact(ctx context.Context, actor certificateauthority.ActorContext, certificateID string) (certificatepg.ArtifactRecord, storage.Object, error) {
	record, err := s.Persistence.GetArtifact(ctx, actor, certificateID, domainrender.ArtifactTypePDF)
	if err != nil {
		return certificatepg.ArtifactRecord{}, storage.Object{}, err
	}

	object, err := s.Storage.Get(ctx, record.ObjectKey)
	if err != nil {
		return certificatepg.ArtifactRecord{}, storage.Object{}, fmt.Errorf("storage get failed: %w", err)
	}

	return record, object, nil
}

func randomHex(bytesLen int) (string, error) {
	b := make([]byte, bytesLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
