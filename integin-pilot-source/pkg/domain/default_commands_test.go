package domain

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestRegisterDefaultCommands(t *testing.T) {
	dispatcher := NewDispatcher()
	RegisterDefaultCommands(dispatcher)

	ctx := context.Background()

	// 1. Test passport.custody.transfer
	transferPayload := json.RawMessage(`{
		"asset_did": "did:integin:asset:liebherr-crane-991",
		"recipient_tenant_id": "tenant-neom-01",
		"country_code_iso2": "SA",
		"authorized_by": "did:integin:user:inspector-ahmed"
	}`)

	res, err := dispatcher.Execute(ctx, Command{
		ID:             "passport.custody.transfer",
		TenantID:       "tenant-aramco-east",
		OrganizationID: "org-heavy-lift",
		ActorID:        "did:integin:user:inspector-ahmed",
		Payload:        transferPayload,
		Timestamp:      time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error executing custody transfer: %v", err)
	}
	if res.Status != "SUCCESS" {
		t.Fatalf("expected status SUCCESS, got %s", res.Status)
	}

	// 2. Test passport.status.update
	statusPayload := json.RawMessage(`{
		"asset_did": "did:integin:asset:liebherr-crane-991",
		"status": "OPERATIONAL",
		"reason": "Annual statutory inspection passed"
	}`)

	res, err = dispatcher.Execute(ctx, Command{
		ID:             "passport.status.update",
		TenantID:       "tenant-neom-01",
		OrganizationID: "org-neom-port",
		ActorID:        "did:integin:user:inspector-ahmed",
		Payload:        statusPayload,
		Timestamp:      time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error executing status update: %v", err)
	}
	if res.Status != "SUCCESS" {
		t.Fatalf("expected status SUCCESS, got %s", res.Status)
	}

	// 3. Test evidence.export.package
	exportPayload := json.RawMessage(`{
		"export_id": "exp-2026-audit-01",
		"evidence_ids": ["ev-1", "ev-2", "ev-3"]
	}`)

	res, err = dispatcher.Execute(ctx, Command{
		ID:             "evidence.export.package",
		TenantID:       "tenant-neom-01",
		OrganizationID: "org-neom-port",
		ActorID:        "did:integin:user:auditor-fatima",
		Payload:        exportPayload,
		Timestamp:      time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error executing evidence export: %v", err)
	}
	if res.Status != "SUCCESS" {
		t.Fatalf("expected status SUCCESS, got %s", res.Status)
	}
}
