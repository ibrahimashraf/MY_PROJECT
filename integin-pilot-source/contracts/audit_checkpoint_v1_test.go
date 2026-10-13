package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAuditCheckpointV1SchemaRequiresScopedHashedEntries(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("audit_checkpoint_v1.schema.json"))
	if err != nil {
		t.Fatalf("read audit checkpoint schema: %v", err)
	}
	var schema struct {
		Schema     string                     `json:"$schema"`
		Required   []string                   `json:"required"`
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("audit checkpoint schema must be valid JSON: %v", err)
	}
	if schema.Schema != "https://json-schema.org/draft/2020-12/schema" {
		t.Fatalf("schema version = %q", schema.Schema)
	}
	required := map[string]bool{}
	for _, field := range schema.Required {
		required[field] = true
	}
	for _, field := range []string{"checkpoint_version", "tenant_id", "environment", "sequence_start", "sequence_end", "previous_root_sha256", "entries", "root_sha256"} {
		if !required[field] {
			t.Fatalf("audit checkpoint schema omits required field %q", field)
		}
	}
	if len(schema.Properties) == 0 {
		t.Fatal("audit checkpoint schema has no properties")
	}
}
