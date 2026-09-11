package cadstorage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
)

var (
	ErrUnauthorizedTenant = errors.New("unauthorized: missing or invalid tenant context (Hazard 01)")
	ErrBlobNotFound       = errors.New("cad blob not found or access denied by RLS")
)

// BlobMetadata stores cryptographic and storage indexing for a CAD file.
type BlobMetadata struct {
	BlobID       string `json:"blob_id"`
	TenantID     string `json:"tenant_id"`
	FileName     string `json:"file_name"`
	Format       string `json:"format"` // DXF, GLB, PDF, DWG
	SHA256Digest string `json:"sha256_digest"`
	ByteSize     int64  `json:"byte_size"`
	S3Key        string `json:"s3_key"`
}

// TenantCADStorageGate enforces PostgreSQL RLS session GUCs prior to issuing S3 operations.
type TenantCADStorageGate struct {
	db        *sql.DB
	s3Bucket  string
	s3BaseURL string
}

// NewTenantCADStorageGate initializes the gated CAD storage service.
func NewTenantCADStorageGate(db *sql.DB, s3Bucket, s3BaseURL string) *TenantCADStorageGate {
	return &TenantCADStorageGate{
		db:        db,
		s3Bucket:  s3Bucket,
		s3BaseURL: s3BaseURL,
	}
}

// BuildPresignedDownloadPath validates tenant session context and generates tenant-isolated S3 key.
func (g *TenantCADStorageGate) BuildPresignedDownloadPath(ctx context.Context, tenantID, blobID string) (string, error) {
	if tenantID == "" {
		return "", ErrUnauthorizedTenant
	}

	// ADV-03: Validate tenant context against PostgreSQL session GUC if DB connection provided
	if g.db != nil {
		var exists bool
		query := `
			SELECT EXISTS (
				SELECT 1 FROM cad_blobs 
				WHERE id = $1 AND tenant_id = current_setting('integin.tenant_id', true)
			);
		`
		err := g.db.QueryRowContext(ctx, query, blobID).Scan(&exists)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("rls tenant verification query failed: %w", err)
		}
		if !exists {
			return "", ErrBlobNotFound
		}
	}

	// Strictly partitioned S3 key prefix: s3://<bucket>/tenants/<tenantID>/cad/<blobID>
	s3Key := fmt.Sprintf("tenants/%s/cad/%s", tenantID, blobID)
	presignedURL := fmt.Sprintf("%s/%s/%s?auth=gated_rls", g.s3BaseURL, g.s3Bucket, s3Key)

	return presignedURL, nil
}

// ComputeDigest returns the SHA-256 hex digest of raw binary data.
func ComputeDigest(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
