package certificatepg

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"integin/internal/domain/certificateauthority"
)

// ArtifactRecord models the immutable metadata row in certificate_artifact.
type ArtifactRecord struct {
	ID              string
	TenantID        string
	OrganizationID  string
	CertificateID   string
	ArtifactType    string
	ObjectKey       string
	ContentType     string
	ByteSize        int64
	ArtifactSHA256  []byte
	SnapshotSHA256  []byte
	RendererVersion string
	CreatedBy       string
	CreatedAt       time.Time
}

// RecordArtifact safely inserts certificate artifact metadata under tenant RLS.
// It enforces that the snapshot_sha256 matches the certificate snapshot and the certificate belongs to the actor.
func (r *Repository) RecordArtifact(ctx context.Context, actor certificateauthority.ActorContext, record ArtifactRecord) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	if strings.TrimSpace(record.ID) == "" {
		return errors.New("artifact id is required")
	}
	if strings.TrimSpace(record.CertificateID) == "" {
		return errors.New("certificate id is required")
	}
	if record.ArtifactType != "CERTIFICATE_PDF" {
		return fmt.Errorf("unsupported artifact type %q", record.ArtifactType)
	}
	if record.ContentType != "application/pdf" {
		return fmt.Errorf("unsupported content type %q", record.ContentType)
	}
	if record.ByteSize <= 0 {
		return errors.New("byte size must be positive")
	}
	if len(record.ArtifactSHA256) != 32 {
		return errors.New("artifact sha256 must be 32 bytes")
	}
	if len(record.SnapshotSHA256) != 32 {
		return errors.New("snapshot sha256 must be 32 bytes")
	}
	if strings.TrimSpace(record.RendererVersion) == "" {
		return errors.New("renderer version is required")
	}
	if strings.TrimSpace(record.ObjectKey) == "" || strings.HasPrefix(record.ObjectKey, "/") || strings.Contains(record.ObjectKey, "..") {
		return errors.New("invalid object key")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := setScope(ctx, tx, actor); err != nil {
		return err
	}

	// Verify certificate exists, belongs to tenant/org, and verify snapshot_sha256 matches
	var dbSnapshotSHA []byte
	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT c.status, s.snapshot_sha256
		FROM certificate_record c
		JOIN certificate_snapshot s ON s.certificate_id = c.id AND s.tenant_id = c.tenant_id AND s.organization_id = c.organization_id
		WHERE c.id = $1 AND c.tenant_id = $2 AND c.organization_id = $3
	`, record.CertificateID, actor.TenantID, actor.OrganizationID).Scan(&status, &dbSnapshotSHA)
	if err == sql.ErrNoRows {
		return fmt.Errorf("certificate or snapshot not found")
	}
	if err != nil {
		return err
	}

	if status != "ISSUED" && status != "EXPIRED" && status != "REVOKED" && status != "SUPERSEDED" {
		return fmt.Errorf("cannot attach artifact to certificate in status %s", status)
	}

	if sha256.Sum256(record.SnapshotSHA256) != sha256.Sum256(dbSnapshotSHA) {
		return fmt.Errorf("snapshot sha256 mismatch")
	}

	query := `
		INSERT INTO certificate_artifact (
			id, tenant_id, organization_id, certificate_id, artifact_type,
			object_key, content_type, byte_size, artifact_sha256, snapshot_sha256,
			renderer_version, created_by, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13
		)
		ON CONFLICT (tenant_id, organization_id, certificate_id, artifact_type) DO UPDATE
		SET object_key = EXCLUDED.object_key,
		    byte_size = EXCLUDED.byte_size,
		    artifact_sha256 = EXCLUDED.artifact_sha256,
		    renderer_version = EXCLUDED.renderer_version,
		    created_by = EXCLUDED.created_by,
		    created_at = EXCLUDED.created_at
	`
	now := time.Now().UTC()
	if !record.CreatedAt.IsZero() {
		now = record.CreatedAt.UTC()
	}

	_, err = tx.ExecContext(ctx, query,
		record.ID, actor.TenantID, actor.OrganizationID, record.CertificateID, record.ArtifactType,
		record.ObjectKey, record.ContentType, record.ByteSize, record.ArtifactSHA256, record.SnapshotSHA256,
		record.RendererVersion, actor.ActorID, now,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetArtifact retrieves the artifact metadata for a certificate scoped to the actor's tenant and organization.
func (r *Repository) GetArtifact(ctx context.Context, actor certificateauthority.ActorContext, certificateID, artifactType string) (ArtifactRecord, error) {
	if err := validateActor(actor); err != nil {
		return ArtifactRecord{}, err
	}
	if strings.TrimSpace(certificateID) == "" {
		return ArtifactRecord{}, errors.New("certificate id is required")
	}
	if artifactType == "" {
		artifactType = "CERTIFICATE_PDF"
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ArtifactRecord{}, err
	}
	defer tx.Rollback()

	if err := setScope(ctx, tx, actor); err != nil {
		return ArtifactRecord{}, err
	}

	var rec ArtifactRecord
	query := `
		SELECT id, tenant_id, organization_id, certificate_id, artifact_type,
		       object_key, content_type, byte_size, artifact_sha256, snapshot_sha256,
		       renderer_version, created_by, created_at
		FROM certificate_artifact
		WHERE certificate_id = $1 AND artifact_type = $2 AND tenant_id = $3 AND organization_id = $4
	`
	err = tx.QueryRowContext(ctx, query, certificateID, artifactType, actor.TenantID, actor.OrganizationID).Scan(
		&rec.ID, &rec.TenantID, &rec.OrganizationID, &rec.CertificateID, &rec.ArtifactType,
		&rec.ObjectKey, &rec.ContentType, &rec.ByteSize, &rec.ArtifactSHA256, &rec.SnapshotSHA256,
		&rec.RendererVersion, &rec.CreatedBy, &rec.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return ArtifactRecord{}, errors.New("artifact not found")
	}
	if err != nil {
		return ArtifactRecord{}, err
	}

	return rec, nil
}
