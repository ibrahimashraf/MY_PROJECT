// INTEGIN work-package domain tests: prove package identity and schema rules are authoritative.
package workpackage

import (
	"strings"
	"testing"
)

func TestPackageComputedHashAndBoundSubmission(t *testing.T) {
	pkg := validPackage(t)
	binding := SubmissionBinding{
		PackageID:        pkg.ID,
		PackageVersion:   pkg.PackageVersion,
		PackageHash:      pkg.PackageHash,
		CapturedFieldIDs: pkg.OrderedFieldIDs(),
	}
	values := map[string]any{
		"asset_serial":     "CRANE-100",
		"condition":        "pass",
		"load_radius":      float64(12.5),
		"is_operable":      true,
		"inspection_class": "thorough",
	}

	if err := pkg.ValidateSubmission(binding, values); err != nil {
		t.Fatalf("expected approved package submission to validate: %v", err)
	}
}

func TestPackageHashChangesWhenRenderedDefinitionChanges(t *testing.T) {
	pkg := validPackage(t)
	changed := pkg
	changed.Sections = append([]Section(nil), pkg.Sections...)
	changed.Sections[0].Fields = append([]FieldDefinition(nil), pkg.Sections[0].Fields...)
	changed.Sections[0].Fields[0].Prompt = "Updated asset serial"
	changedHash, err := changed.WithComputedHash()
	if err != nil {
		t.Fatalf("compute changed package hash: %v", err)
	}
	if changedHash.PackageHash == pkg.PackageHash {
		t.Fatal("expected rendered package change to produce a new package hash")
	}
}

func TestValidateSubmissionRejectsMismatchedPackageBinding(t *testing.T) {
	pkg := validPackage(t)
	binding := validBinding(pkg)
	binding.PackageHash = "sha256:" + strings.Repeat("0", 64)

	err := pkg.ValidateSubmission(binding, validValues())
	if err == nil || !strings.Contains(err.Error(), "binding") {
		t.Fatalf("expected binding mismatch, got %v", err)
	}
}

func TestValidateSubmissionRejectsUnapprovedPackage(t *testing.T) {
	pkg := validPackage(t)
	pkg.State = PublicationDraft
	updated, err := pkg.WithComputedHash()
	if err != nil {
		t.Fatalf("compute draft package hash: %v", err)
	}

	err = updated.ValidateSubmission(validBinding(updated), validValues())
	if err == nil || !strings.Contains(err.Error(), "approved work package") {
		t.Fatalf("expected draft package rejection, got %v", err)
	}
}

func TestValidateSubmissionRejectsInvalidFieldState(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(SubmissionBinding, map[string]any) (SubmissionBinding, map[string]any)
		message string
	}{
		{
			name: "missing required field",
			mutate: func(binding SubmissionBinding, values map[string]any) (SubmissionBinding, map[string]any) {
				delete(values, "asset_serial")
				return binding, values
			},
			message: "required field",
		},
		{
			name: "unknown field",
			mutate: func(binding SubmissionBinding, values map[string]any) (SubmissionBinding, map[string]any) {
				values["unapproved_field"] = "value"
				return binding, values
			},
			message: "unknown field",
		},
		{
			name: "wrong status value",
			mutate: func(binding SubmissionBinding, values map[string]any) (SubmissionBinding, map[string]any) {
				values["condition"] = "acceptable"
				return binding, values
			},
			message: "requires pass, fail, or not_applicable",
		},
		{
			name: "captured sequence changed",
			mutate: func(binding SubmissionBinding, values map[string]any) (SubmissionBinding, map[string]any) {
				binding.CapturedFieldIDs[0], binding.CapturedFieldIDs[1] = binding.CapturedFieldIDs[1], binding.CapturedFieldIDs[0]
				return binding, values
			},
			message: "captured-field sequence",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pkg := validPackage(t)
			binding, values := test.mutate(validBinding(pkg), validValues())
			err := pkg.ValidateSubmission(binding, values)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("expected %q error, got %v", test.message, err)
			}
		})
	}
}

func validPackage(t *testing.T) Package {
	t.Helper()
	pkg, err := (Package{
		ID:              "pkg-crane-thorough-v1",
		TenantID:        "tenant-a",
		OrganizationID:  "organization-a",
		TemplateCode:    "mobile-crane-thorough",
		TemplateVersion: 1,
		PackageVersion:  1,
		SchemaVersion:   1,
		State:           PublicationApproved,
		Sections: []Section{
			{
				ID:    "equipment",
				Title: "Equipment profile",
				Fields: []FieldDefinition{
					{ID: "asset_serial", Prompt: "Asset serial", Type: FieldText, Required: true},
					{ID: "inspection_class", Prompt: "Inspection class", Type: FieldChoice, Required: true, Options: []string{"thorough", "periodic"}},
				},
			},
			{
				ID:    "condition",
				Title: "Condition",
				Fields: []FieldDefinition{
					{ID: "condition", Prompt: "Overall condition", Type: FieldPassFailNA, Required: true},
					{ID: "load_radius", Prompt: "Load radius", Type: FieldNumber},
					{ID: "is_operable", Prompt: "Equipment operable", Type: FieldBoolean, Required: true},
				},
			},
		},
	}).WithComputedHash()
	if err != nil {
		t.Fatalf("create package hash: %v", err)
	}
	return pkg
}

func validBinding(pkg Package) SubmissionBinding {
	return SubmissionBinding{
		PackageID:        pkg.ID,
		PackageVersion:   pkg.PackageVersion,
		PackageHash:      pkg.PackageHash,
		CapturedFieldIDs: pkg.OrderedFieldIDs(),
	}
}

func validValues() map[string]any {
	return map[string]any{
		"asset_serial":     "CRANE-100",
		"inspection_class": "thorough",
		"condition":        "pass",
		"load_radius":      float64(12.5),
		"is_operable":      true,
	}
}
