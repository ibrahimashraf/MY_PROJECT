package workpackageenforcement

import (
	"context"
	"errors"
	"testing"

	"integin/internal/domain/workpackage"
)

type memoryResolver struct {
	packageValue workpackage.Package
	err          error
}

func (r memoryResolver) GetApproved(_ context.Context, tenantID, organizationID, packageID string, version int) (workpackage.Package, error) {
	if r.err != nil {
		return workpackage.Package{}, r.err
	}
	if r.packageValue.TenantID != tenantID || r.packageValue.OrganizationID != organizationID || r.packageValue.ID != packageID || r.packageValue.PackageVersion != version {
		return workpackage.Package{}, errors.New("not found")
	}
	return r.packageValue, nil
}

func approvedPackage(t *testing.T) workpackage.Package {
	t.Helper()
	p := workpackage.Package{
		ID: "package-1", TenantID: "tenant-1", OrganizationID: "org-1",
		TemplateCode: "crane", TemplateVersion: 1, PackageVersion: 1, SchemaVersion: 1,
		State: workpackage.PublicationApproved,
		Sections: []workpackage.Section{{ID: "main", Title: "Main", Fields: []workpackage.FieldDefinition{
			{ID: "condition", Prompt: "Condition", Type: workpackage.FieldPassFailNA, Required: true},
			{ID: "capacity", Prompt: "Capacity", Type: workpackage.FieldNumber, Required: true},
		}}},
	}
	hashed, err := p.WithComputedHash()
	if err != nil {
		t.Fatalf("hash package: %v", err)
	}
	return hashed
}

func TestValidatorAcceptsExactApprovedPayload(t *testing.T) {
	p := approvedPackage(t)
	validator, err := NewValidator(memoryResolver{packageValue: p})
	if err != nil {
		t.Fatalf("new validator: %v", err)
	}
	payload := []byte(`{"inspection_id":"inspection-1","work_package_id":"package-1","work_package_version":1,"work_package_hash":"` + p.PackageHash + `","work_package_field_sequence":["condition","capacity"],"findings":[{"item_id":"condition","response":"pass"},{"item_id":"capacity","response":"12.5"}]}`)

	if err := validator.ValidateInspectionPayload(context.Background(), "tenant-1", "org-1", "inspection-1", payload); err != nil {
		t.Fatalf("validate payload: %v", err)
	}
}

func TestValidatorRejectsModifiedPackageHashAndInvalidTypedResponse(t *testing.T) {
	p := approvedPackage(t)
	validator, _ := NewValidator(memoryResolver{packageValue: p})
	badHash := []byte(`{"inspection_id":"inspection-1","work_package_id":"package-1","work_package_version":1,"work_package_hash":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","work_package_field_sequence":["condition","capacity"],"findings":[{"item_id":"condition","response":"pass"},{"item_id":"capacity","response":"12.5"}]}`)
	if err := validator.ValidateInspectionPayload(context.Background(), "tenant-1", "org-1", "inspection-1", badHash); !errors.Is(err, ErrPayloadBinding) {
		t.Fatalf("bad hash error = %v", err)
	}
	badNumber := []byte(`{"inspection_id":"inspection-1","work_package_id":"package-1","work_package_version":1,"work_package_hash":"` + p.PackageHash + `","work_package_field_sequence":["condition","capacity"],"findings":[{"item_id":"condition","response":"pass"},{"item_id":"capacity","response":"not-a-number"}]}`)
	if err := validator.ValidateInspectionPayload(context.Background(), "tenant-1", "org-1", "inspection-1", badNumber); !errors.Is(err, ErrPayloadBinding) {
		t.Fatalf("bad number error = %v", err)
	}
}
