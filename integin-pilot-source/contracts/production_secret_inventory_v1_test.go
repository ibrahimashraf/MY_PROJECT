package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionSecretInventoryV1ContainsReferencesNotValues(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("production_secret_inventory_v1.json"))
	if err != nil {
		t.Fatalf("read production secret inventory: %v", err)
	}

	var inventory struct {
		InventoryVersion     string `json:"inventory_version"`
		Status               string `json:"status"`
		PrivateLocalBoundary string `json:"private_local_boundary"`
		SecretClasses        []struct {
			SecretID           string `json:"secret_id"`
			Owner              string `json:"owner"`
			Classification     string `json:"classification"`
			PlannedOpenBaoPath string `json:"planned_openbao_path"`
			RotationTrigger    string `json:"rotation_trigger"`
			RecoveryReference  string `json:"recovery_reference"`
			AuditRequirement   string `json:"audit_requirement"`
			AvailabilityRule   string `json:"availability_rule"`
		} `json:"secret_classes"`
	}
	if err := json.Unmarshal(raw, &inventory); err != nil {
		t.Fatalf("inventory must be valid JSON: %v", err)
	}
	if inventory.InventoryVersion != "production-secret-inventory/v1" {
		t.Fatalf("inventory_version = %q", inventory.InventoryVersion)
	}
	if inventory.Status != "design-baseline-not-wired" {
		t.Fatalf("inventory status = %q", inventory.Status)
	}
	if inventory.PrivateLocalBoundary == "" {
		t.Fatal("private local boundary is required")
	}
	if len(inventory.SecretClasses) < 7 {
		t.Fatalf("secret class count = %d, want at least 7", len(inventory.SecretClasses))
	}
	for _, class := range inventory.SecretClasses {
		for field, value := range map[string]string{
			"secret_id":            class.SecretID,
			"owner":                class.Owner,
			"classification":       class.Classification,
			"planned_openbao_path": class.PlannedOpenBaoPath,
			"rotation_trigger":     class.RotationTrigger,
			"recovery_reference":   class.RecoveryReference,
			"audit_requirement":    class.AuditRequirement,
			"availability_rule":    class.AvailabilityRule,
		} {
			if strings.TrimSpace(value) == "" {
				t.Fatalf("%s missing for secret class %q", field, class.SecretID)
			}
		}
		if !strings.HasPrefix(class.PlannedOpenBaoPath, "kv/production/") {
			t.Fatalf("planned OpenBao path %q is outside production namespace", class.PlannedOpenBaoPath)
		}
	}

	for _, forbidden := range []string{"\"token\":", "\"password\":", "\"private_key\":", "\"secret_value\":"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("inventory contains forbidden value-bearing field %q", forbidden)
		}
	}
}
