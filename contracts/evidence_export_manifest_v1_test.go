package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEvidenceExportManifestV1SchemaHasRequiredIntegrityAndPrivacyFields(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("evidence_export_manifest_v1.schema.json"))
	if err != nil {
		t.Fatalf("read evidence export manifest schema: %v", err)
	}

	var schema struct {
		Schema     string                     `json:"$schema"`
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("manifest schema must be valid JSON: %v", err)
	}
	if schema.Schema != "https://json-schema.org/draft/2020-12/schema" {
		t.Fatalf("schema version = %q", schema.Schema)
	}
	if len(schema.Properties) == 0 {
		t.Fatal("manifest schema has no properties")
	}

	required := make(map[string]struct{}, len(schema.Required))
	for _, field := range schema.Required {
		required[field] = struct{}{}
	}
	for _, field := range []string{"manifest_version", "tenant_id", "organization_id", "evidence", "object_count", "recovery", "privacy", "approval", "manifest_checksum"} {
		if _, ok := required[field]; !ok {
			t.Fatalf("manifest schema omits required field %q", field)
		}
	}
}
