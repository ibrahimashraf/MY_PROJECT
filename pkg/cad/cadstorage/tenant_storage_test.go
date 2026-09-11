package cadstorage

import (
	"context"
	"strings"
	"testing"
)

func TestTenantCADStorageGate(t *testing.T) {
	gate := NewTenantCADStorageGate(nil, "integin-pilot-evidence", "http://127.0.0.1:19000")

	ctx := context.Background()

	// 1. Test rejection on missing tenant ID
	_, err := gate.BuildPresignedDownloadPath(ctx, "", "blob-12345")
	if err != ErrUnauthorizedTenant {
		t.Fatalf("Expected ErrUnauthorizedTenant on empty tenant ID, got %v", err)
	}

	// 2. Test valid tenant S3 path generation
	path, err := gate.BuildPresignedDownloadPath(ctx, "tenant-sa-001", "blob-12345")
	if err != nil {
		t.Fatalf("BuildPresignedDownloadPath failed: %v", err)
	}

	expectedPrefix := "http://127.0.0.1:19000/integin-pilot-evidence/tenants/tenant-sa-001/cad/blob-12345"
	if !strings.HasPrefix(path, expectedPrefix) {
		t.Errorf("Expected path to start with %q, got %q", expectedPrefix, path)
	}

	// 3. Test SHA-256 computation
	digest := ComputeDigest([]byte("INTEGIN_CAD_BINARY_PAYLOAD"))
	if len(digest) != 64 {
		t.Errorf("Expected 64-char hex SHA256 digest, got %d chars: %s", len(digest), digest)
	}
}
