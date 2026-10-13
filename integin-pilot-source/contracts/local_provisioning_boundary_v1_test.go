package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalProvisioningBoundaryV1FreezesProductionExclusion(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("local_provisioning_boundary_v1.json"))
	if err != nil {
		t.Fatalf("read local provisioning boundary contract: %v", err)
	}
	var contract struct {
		ContractVersion   string `json:"contract_version"`
		Status            string `json:"status"`
		LocalProvisioning struct {
			RequiredEnablementVariable  string `json:"required_enablement_variable"`
			DefaultEnabled              bool   `json:"default_enabled"`
			Scope                       string `json:"scope"`
			LoopbackRequiredWhenEnabled bool   `json:"loopback_required_when_enabled"`
		} `json:"local_provisioning"`
		ProductionDeviceEnrollment struct {
			ImplementationStatus             string `json:"implementation_status"`
			Authority                        string `json:"authority"`
			LocalProvisioningMutualExclusion bool   `json:"local_provisioning_mutual_exclusion"`
		} `json:"production_device_enrollment"`
		ProductionImageDefaults map[string]bool `json:"production_image_defaults"`
	}
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatalf("local provisioning boundary contract must be valid JSON: %v", err)
	}
	if contract.ContractVersion != "local-provisioning-boundary/v1" || contract.Status != "test-only-boundary-freeze" {
		t.Fatalf("unexpected boundary contract identity: %q / %q", contract.ContractVersion, contract.Status)
	}
	if contract.LocalProvisioning.RequiredEnablementVariable != "INTEGIN_LOCAL_PROVISIONING_ENABLED" || contract.LocalProvisioning.DefaultEnabled || !contract.LocalProvisioning.LoopbackRequiredWhenEnabled || contract.LocalProvisioning.Scope != "development-and-pilot-loopback-only" {
		t.Fatal("local provisioning contract must remain explicit-opt-in, disabled-by-default, and loopback-only")
	}
	if contract.ProductionDeviceEnrollment.ImplementationStatus != "NOT_IMPLEMENTED" || contract.ProductionDeviceEnrollment.Authority != "EXPLICIT_HUMAN_APPROVAL_REQUIRED" || !contract.ProductionDeviceEnrollment.LocalProvisioningMutualExclusion {
		t.Fatal("production enrollment must remain unimplemented, approval-gated, and mutually exclusive with local provisioning")
	}
	if enabled, ok := contract.ProductionImageDefaults["INTEGIN_LOCAL_PROVISIONING_ENABLED"]; !ok || enabled {
		t.Fatal("production image default must disable local provisioning")
	}
}
