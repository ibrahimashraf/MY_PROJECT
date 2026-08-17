package vectors

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"integin/internal/domain/workpackage"
)

type manifestCanonicalVector struct {
	ManifestVersion    string                        `json:"manifest_version"`
	TenantID           string                        `json:"tenant_id"`
	OrganizationID     string                        `json:"organization_id"`
	InspectionID       string                        `json:"inspection_id"`
	DeviceID           string                        `json:"device_id"`
	PackageID          string                        `json:"package_id"`
	PackageVersion     int                           `json:"package_version"`
	PackageHash        string                        `json:"package_hash"`
	AssignmentContext  workpackage.AssignmentContext `json:"assignment_context"`
	SchemaVersion      int                           `json:"schema_version"`
	AuthorityEpoch     uint64                        `json:"authority_epoch"`
	IssuedAt           string                        `json:"issued_at"`
	ExpiresAt          string                        `json:"expires_at"`
	SignatureAlgorithm string                        `json:"signature_algorithm"`
	KeyID              string                        `json:"key_id"`
	CanonicalUTF8      string                        `json:"canonical_utf8"`
}

func TestManifestCanonicalV1Vector(t *testing.T) {
	raw, err := os.ReadFile("manifest_canonical_v1.json")
	if err != nil {
		t.Fatalf("read vector: %v", err)
	}
	var vector manifestCanonicalVector
	if err := json.Unmarshal(raw, &vector); err != nil {
		t.Fatalf("decode vector: %v", err)
	}
	issuedAt, err := time.Parse(time.RFC3339Nano, vector.IssuedAt)
	if err != nil {
		t.Fatalf("parse issued_at: %v", err)
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, vector.ExpiresAt)
	if err != nil {
		t.Fatalf("parse expires_at: %v", err)
	}
	got := workpackage.CanonicalManifest(workpackage.ManifestCanonicalInput{
		ManifestVersion:    vector.ManifestVersion,
		TenantID:           vector.TenantID,
		OrganizationID:     vector.OrganizationID,
		InspectionID:       vector.InspectionID,
		DeviceID:           vector.DeviceID,
		PackageID:          vector.PackageID,
		PackageVersion:     vector.PackageVersion,
		PackageHash:        vector.PackageHash,
		AssignmentContext:  vector.AssignmentContext,
		SchemaVersion:      vector.SchemaVersion,
		AuthorityEpoch:     vector.AuthorityEpoch,
		IssuedAt:           issuedAt,
		ExpiresAt:          expiresAt,
		SignatureAlgorithm: vector.SignatureAlgorithm,
		KeyID:              vector.KeyID,
	})
	if got != vector.CanonicalUTF8 {
		t.Fatalf("canonical manifest mismatch:\nwant %q\n got %q", vector.CanonicalUTF8, got)
	}
}
