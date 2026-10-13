package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseRecordV1RequiresApprovalRecoveryRollbackAndNoSecrets(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("release_record_v1.schema.json"))
	if err != nil {
		t.Fatalf("read release record schema: %v", err)
	}
	var schema struct {
		Schema   string   `json:"$schema"`
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("release record schema must be valid JSON: %v", err)
	}
	if schema.Schema != "https://json-schema.org/draft/2020-12/schema" {
		t.Fatalf("schema version = %q", schema.Schema)
	}
	required := make(map[string]bool, len(schema.Required))
	for _, field := range schema.Required {
		required[field] = true
	}
	for _, field := range []string{"release", "migrations", "dependency_lock", "verification", "recovery", "approval", "rollback", "post_release_reconciliation", "no_secret_material_embedded"} {
		if !required[field] {
			t.Fatalf("release record schema omits required field %q", field)
		}
	}
}
