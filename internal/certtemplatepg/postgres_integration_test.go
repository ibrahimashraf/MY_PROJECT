package certtemplatepg

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"integin/internal/domain/certificatetemplate"

	_ "github.com/lib/pq"
)

const certificateTemplateIntegrationDSNEnv = "INTEGIN_TEST_DATABASE_URL"

func TestPostgresCertificateTemplateVersionIntegration(t *testing.T) {
	dsn := os.Getenv(certificateTemplateIntegrationDSNEnv)
	if dsn == "" {
		t.Skipf("set %s to run certificate template PostgreSQL integration tests", certificateTemplateIntegrationDSNEnv)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}
	stamp := time.Now().UTC().UnixNano()
	actor := certificatetemplate.ActorContext{TenantID: fmt.Sprintf("it-certificate-template-%d", stamp), OrganizationID: "org-1", ActorID: "template-admin-1"}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		for _, statement := range []string{
			`DELETE FROM certificate_template_cell WHERE tenant_id = $1`,
			`DELETE FROM certificate_template WHERE tenant_id = $1`,
		} {
			if _, err := db.ExecContext(cleanupCtx, statement, actor.TenantID); err != nil {
				t.Errorf("cleanup certificate template fixtures: %v", err)
				return
			}
		}
	})
	repository, err := NewRepository(db, certificatetemplate.DefaultCatalog())
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	definition := certificatetemplate.Definition{ID: fmt.Sprintf("template-%d", stamp), TemplateCode: "lifting_certificate", Version: 1, Title: "Lifting certificate", AssetType: "lifting", Status: certificatetemplate.Draft, CatalogVersion: 1, PageCount: 1, PageWidth: 595, PageHeight: 842, Cells: []certificatetemplate.Cell{
		{ID: "inspection_id", PageNumber: 1, Rectangle: certificatetemplate.Rectangle{X: 20, Y: 20, Width: 180, Height: 30}, Kind: certificatetemplate.TextCell, BindingKey: certificatetemplate.InspectionID, FitPolicy: certificatetemplate.SingleLineRequired, MaxLines: 1, Required: true},
		{ID: "asset_id", PageNumber: 1, Rectangle: certificatetemplate.Rectangle{X: 20, Y: 60, Width: 180, Height: 30}, Kind: certificatetemplate.TextCell, BindingKey: certificatetemplate.InspectionAssetID, FitPolicy: certificatetemplate.SingleLineRequired, MaxLines: 1, Required: true},
	}}
	stored, inserted, err := repository.RegisterDraft(ctx, actor, definition)
	if err != nil || !inserted || stored.CreatedBy != actor.ActorID || stored.Status != certificatetemplate.Draft {
		t.Fatalf("register draft: stored=%#v inserted=%v err=%v", stored, inserted, err)
	}
	if _, inserted, err := repository.RegisterDraft(ctx, actor, definition); err != nil || inserted {
		t.Fatalf("immutable replay: inserted=%v err=%v", inserted, err)
	}
	conflicting := definition
	conflicting.Title = "Changed certificate"
	if _, _, err := repository.RegisterDraft(ctx, actor, conflicting); err != certificatetemplate.ErrImmutableConflict {
		t.Fatalf("expected immutable conflict, got %v", err)
	}
	if _, err := repository.Get(ctx, certificatetemplate.ActorContext{TenantID: actor.TenantID, OrganizationID: "org-2", ActorID: "template-admin-2"}, definition.TemplateCode, definition.Version); err != ErrNotFound {
		t.Fatalf("expected cross-organization not found, got %v", err)
	}
	approved, err := repository.Approve(ctx, actor, definition.TemplateCode, definition.Version, time.Date(2026, time.August, 21, 22, 0, 0, 0, time.UTC))
	if err != nil || approved.Status != certificatetemplate.Approved || approved.ApprovedBy != actor.ActorID {
		t.Fatalf("approve template: %#v err=%v", approved, err)
	}
}
