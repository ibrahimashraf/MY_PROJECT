package formdefinitionpg

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"integin/internal/domain/formdefinition"
)

func TestRegisterDraftAndLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testDB(t)
	repo, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	actor := formdefinition.ActorContext{
		TenantID:       "test-tenant",
		OrganizationID: "test-org",
		ActorID:        "test-user",
	}
	form := formdefinition.FormVersion{
		ID:             "form_test_001",
		FormCode:       "test_form",
		Version:        1,
		Title:          "Test Form",
		Description:    "A test form",
		AssetType:      "EQUIPMENT",
		CatalogVersion: 1,
		Fields: []formdefinition.FormField{
			{
				FieldID:   "field_001",
				FieldCode: "question_1",
				FieldType: formdefinition.FieldTypeText,
				Label:     "First Question",
				Required:  true,
				SortOrder: 0,
			},
			{
				FieldID:   "field_002",
				FieldCode: "question_2",
				FieldType: formdefinition.FieldTypeSingleSelect,
				Label:     "Severity",
				Required:  true,
				SortOrder: 1,
				Options: []formdefinition.FieldOption{
					{Code: "high", Label: "High"},
					{Code: "low", Label: "Low"},
				},
				EvidencePolicies: []formdefinition.EvidencePolicy{
					{
						EvidenceType:   formdefinition.EvidenceTypePhoto,
						Required:       true,
						MaxCount:       int64Ptr(3),
						Classification: formdefinition.ClassificationConfidential,
					},
				},
			},
		},
	}
	stored, created, err := repo.RegisterDraft(context.Background(), actor, form)
	if err != nil {
		t.Fatalf("RegisterDraft: %v", err)
	}
	if !created {
		t.Fatal("expected created=true")
	}
	if stored.FormCode != "test_form" {
		t.Errorf("FormCode = %q, want %q", stored.FormCode, "test_form")
	}
	if stored.Status != formdefinition.FormStatusDraft {
		t.Errorf("Status = %q, want %q", stored.Status, formdefinition.FormStatusDraft)
	}
	if len(stored.Fields) != 2 {
		t.Fatalf("Fields = %d, want 2", len(stored.Fields))
	}
	if stored.Fields[0].FieldCode != "question_1" {
		t.Errorf("Fields[0].FieldCode = %q, want %q", stored.Fields[0].FieldCode, "question_1")
	}
	if len(stored.Fields[1].EvidencePolicies) != 1 {
		t.Fatalf("Fields[1].EvidencePolicies = %d, want 1", len(stored.Fields[1].EvidencePolicies))
	}
	// Idempotent re-register
	stored2, created2, err := repo.RegisterDraft(context.Background(), actor, form)
	if err != nil {
		t.Fatalf("RegisterDraft (idempotent): %v", err)
	}
	if created2 {
		t.Fatal("expected created=false on idempotent re-register")
	}
	if stored2.ID != stored.ID {
		t.Errorf("idempotent stored ID = %q, want %q", stored2.ID, stored.ID)
	}
}

func TestApproveAndRetire(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testDB(t)
	repo, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	actor := formdefinition.ActorContext{
		TenantID:       "test-tenant-approve",
		OrganizationID: "test-org-approve",
		ActorID:        "test-user",
	}
	form := formdefinition.FormVersion{
		ID:             "form_approve_001",
		FormCode:       "approve_form",
		Version:        1,
		Title:          "Approve Form",
		CatalogVersion: 1,
		Fields: []formdefinition.FormField{
			{
				FieldID:   "field_001",
				FieldCode: "q1",
				FieldType: formdefinition.FieldTypeText,
				Label:     "Q1",
				SortOrder: 0,
			},
		},
	}
	_, _, err = repo.RegisterDraft(context.Background(), actor, form)
	if err != nil {
		t.Fatalf("RegisterDraft: %v", err)
	}
	approved, err := repo.Approve(context.Background(), actor, form.ID, 1)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if approved.Status != formdefinition.FormStatusApproved {
		t.Errorf("Status = %q, want %q", approved.Status, formdefinition.FormStatusApproved)
	}
	if approved.ApprovedBy != "test-user" {
		t.Errorf("ApprovedBy = %q, want %q", approved.ApprovedBy, "test-user")
	}
	// Approve again should fail
	_, err = repo.Approve(context.Background(), actor, form.ID, 1)
	if err != ErrNotDraft {
		t.Errorf("Approve again: err = %v, want ErrNotDraft", err)
	}
	retired, err := repo.Retire(context.Background(), actor, form.ID, 1)
	if err != nil {
		t.Fatalf("Retire: %v", err)
	}
	if retired.Status != formdefinition.FormStatusRetired {
		t.Errorf("Status = %q, want %q", retired.Status, formdefinition.FormStatusRetired)
	}
}

func TestRLSTenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testDB(t)
	repo, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	actorA := formdefinition.ActorContext{
		TenantID:       "tenant-a",
		OrganizationID: "org-a",
		ActorID:        "user-a",
	}
	actorB := formdefinition.ActorContext{
		TenantID:       "tenant-b",
		OrganizationID: "org-b",
		ActorID:        "user-b",
	}
	form := formdefinition.FormVersion{
		ID:             "form_isolation_001",
		FormCode:       "isolation_form",
		Version:        1,
		Title:          "Isolation Form",
		CatalogVersion: 1,
		Fields: []formdefinition.FormField{
			{
				FieldID:   "field_001",
				FieldCode: "q1",
				FieldType: formdefinition.FieldTypeText,
				Label:     "Q1",
				SortOrder: 0,
			},
		},
	}
	// Register as tenant A
	_, _, err = repo.RegisterDraft(context.Background(), actorA, form)
	if err != nil {
		t.Fatalf("RegisterDraft (tenant A): %v", err)
	}
	// Try to load as tenant B - should not find
	_, err = repo.Get(context.Background(), actorB, form.ID)
	if err != ErrNotFound {
		t.Errorf("Get as tenant B: err = %v, want ErrNotFound", err)
	}
	// Approve as tenant A
	_, err = repo.Approve(context.Background(), actorA, form.ID, 1)
	if err != nil {
		t.Fatalf("Approve (tenant A): %v", err)
	}
	// Try to retire as tenant B - should not find
	_, err = repo.Retire(context.Background(), actorB, form.ID, 1)
	if err != ErrNotFound {
		t.Errorf("Retire as tenant B: err = %v, want ErrNotFound", err)
	}
}

func TestRLSOrganizationIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testDB(t)
	repo, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	actorA := formdefinition.ActorContext{
		TenantID:       "tenant-org",
		OrganizationID: "org-a",
		ActorID:        "user-a",
	}
	actorB := formdefinition.ActorContext{
		TenantID:       "tenant-org",
		OrganizationID: "org-b",
		ActorID:        "user-b",
	}
	form := formdefinition.FormVersion{
		ID:             "form_org_iso_001",
		FormCode:       "org_iso_form",
		Version:        1,
		Title:          "Org Isolation Form",
		CatalogVersion: 1,
		Fields: []formdefinition.FormField{
			{
				FieldID:   "field_001",
				FieldCode: "q1",
				FieldType: formdefinition.FieldTypeText,
				Label:     "Q1",
				SortOrder: 0,
			},
		},
	}
	// Register as org A
	_, _, err = repo.RegisterDraft(context.Background(), actorA, form)
	if err != nil {
		t.Fatalf("RegisterDraft (org A): %v", err)
	}
	// Try to load as org B - should not find
	_, err = repo.Get(context.Background(), actorB, form.ID)
	if err != ErrNotFound {
		t.Errorf("Get as org B: err = %v, want ErrNotFound", err)
	}
}

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := "postgres://integin:integin@localhost:5432/integin_test?sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("skipping: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("skipping: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func int64Ptr(v int64) *int64 {
	return &v
}

func TestMain(m *testing.M) {
	// Run tests
	m.Run()
}

// Verify that the domain model validates correctly.
func TestDomainValidation(t *testing.T) {
	actor := formdefinition.ActorContext{
		TenantID:       "tenant",
		OrganizationID: "org",
		ActorID:        "user",
	}
	if err := actor.Validate(); err != nil {
		t.Errorf("ActorContext.Validate() = %v", err)
	}
	form := formdefinition.FormVersion{
		ID:             "form_1",
		TenantID:       "tenant",
		OrganizationID: "org",
		FormCode:       "test",
		Version:        1,
		Title:          "Test",
		Status:         formdefinition.FormStatusDraft,
		CatalogVersion: 1,
		CreatedBy:      "user",
		CreatedAt:      time.Now(),
		Fields: []formdefinition.FormField{
			{
				FieldID:   "f1",
				FieldCode: "q1",
				FieldType: formdefinition.FieldTypeText,
				Label:     "Q1",
				SortOrder: 0,
			},
		},
	}
	if err := form.Validate(); err != nil {
		t.Errorf("FormVersion.Validate() = %v", err)
	}
}
