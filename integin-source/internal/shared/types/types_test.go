package types

import "testing"

func TestTenantContextValidation(t *testing.T) {
	valid := TenantContext{TenantID: "tenant-1", OrganizationID: "org-1", Environment: EnvironmentLive}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	invalid := []TenantContext{{OrganizationID: "org-1", Environment: EnvironmentLive}, {TenantID: "tenant-1", Environment: EnvironmentLive}, {TenantID: "tenant-1", OrganizationID: "org-1", Environment: "DEV"}}
	for _, context := range invalid {
		if err := context.Validate(); err == nil {
			t.Fatalf("expected invalid context: %#v", context)
		}
	}
}
