package assurance_test

import (
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"

	"integin/internal/domain/assurance"
)

func TestLedgerBuilder_BuildNextEntry(t *testing.T) {
	builder := &assurance.LedgerBuilder{}
	tenantID := uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001")
	assetID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	timestamp := time.Unix(1790278800, 0) // Fixed time for deterministic hash

	entry, err := builder.BuildNextEntry(
		tenantID,
		assetID,
		1,
		"sha256:dummyrecordhash",
		"sha256:prevcid",
		"sha256:epochcid",
		timestamp,
	)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if entry.CID == "" {
		t.Errorf("expected CID to be generated, got empty string")
	}

	// Because of the canonical JSON generation, this hash should be deterministic.
	expectedCID := "sha256:4dd90ae5a2873d4539d97aa60eda658d8f09b9f8e26d6690f94b6c36cc716511"
	if entry.CID != expectedCID {
		t.Errorf("expected CID %s, got %s", expectedCID, entry.CID)
	}
}
