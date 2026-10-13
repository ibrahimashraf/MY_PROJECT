package workpackage

import (
	"testing"
	"time"
)

func TestCanonicalManifestSortsAssignmentContextFieldAssets(t *testing.T) {
	input := ManifestCanonicalInput{
		ManifestVersion: ManifestProtocolVersion,
		TenantID:        "tenant-vector-a",
		OrganizationID:  "org-vector-a",
		InspectionID:    "inspection-vector-a",
		DeviceID:        "device-vector-a",
		PackageID:       "package-vector-a",
		PackageVersion:  2,
		PackageHash:     "sha256:package-vector-hash",
		AssignmentContext: AssignmentContext{
			RootAssetID:      "asset-root-a",
			InspectionType:   "crane-inspection",
			ProcedureVersion: "procedure-3",
			ScheduledAt:      time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC),
			FieldAssetIDs: map[string]string{
				"field-z": "asset-z",
				"field-a": "asset-a",
			},
		},
		SchemaVersion:      3,
		AuthorityEpoch:     7,
		IssuedAt:           time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC),
		ExpiresAt:          time.Date(2026, time.August, 17, 12, 30, 0, 0, time.UTC),
		SignatureAlgorithm: ManifestProofSignatureAlgorithm,
		KeyID:              "manifest-key-1",
	}
	const want = "work-package-manifest/v1|tenant-vector-a|org-vector-a|inspection-vector-a|device-vector-a|package-vector-a|2|sha256:package-vector-hash|asset-root-a|crane-inspection|procedure-3|2026-08-17T12:00:00Z|field-a=asset-a,field-z=asset-z|3|7|2026-08-17T12:00:00Z|2026-08-17T12:30:00Z|Ed25519|manifest-key-1"
	if got := CanonicalManifest(input); got != want {
		t.Fatalf("canonical manifest mismatch:\nwant %q\n got %q", want, got)
	}
}
