package workpackagepg

import (
	"database/sql"
	"errors"
	"testing"

	"integin/internal/domain/workpackage"
)

func TestNewRepositoryRejectsNilDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("expected nil database rejection")
	}
}

func TestNewRepositoryAcceptsDatabaseHandle(t *testing.T) {
	repository, err := NewRepository(&sql.DB{})
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	if repository == nil {
		t.Fatal("expected repository")
	}
}

func TestValidateStoredPackageMarksMismatchedHashAsIntegrityFailure(t *testing.T) {
	p := workpackage.Package{
		ID:              "package",
		TenantID:        "tenant",
		OrganizationID:  "organization",
		TemplateCode:    "template",
		TemplateVersion: 1,
		PackageVersion:  1,
		SchemaVersion:   1,
		State:           workpackage.PublicationApproved,
		Sections: []workpackage.Section{{
			ID: "section", Title: "Section", Fields: []workpackage.FieldDefinition{{ID: "field", Prompt: "Field", Type: workpackage.FieldText, Required: true}},
		}},
	}
	computed, err := p.WithComputedHash()
	if err != nil {
		t.Fatal(err)
	}
	computed.PackageHash = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	if err := validateStoredPackage(computed); !errors.Is(err, workpackage.ErrPackageIntegrity) {
		t.Fatalf("stored package mismatch did not carry integrity sentinel: %v", err)
	}
}
