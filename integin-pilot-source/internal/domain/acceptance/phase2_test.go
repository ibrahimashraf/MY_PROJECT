package acceptance

import (
	"testing"
	"time"

	"integin/internal/domain/certificate"
	"integin/internal/domain/completeness"
	"integin/internal/domain/inspection"
	"integin/internal/domain/template"
	"integin/internal/shared/types"
)

func TestPhase2InspectionToCertificateWorkflow(t *testing.T) {
	registry := template.NewRegistry()
	if err := registry.Register(template.Definition{Code: "CRANE", Version: 1, AssetType: "overhead_crane", Sections: []template.Section{{ID: "structure", Items: []template.Item{{ID: "beam", Prompt: "Inspect beam", ResponseType: "PASS_FAIL", Required: true}}}}}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := registry.Resolve("CRANE", 1, template.AssetNode{ID: "asset-1", AssetType: "overhead_crane", Name: "Crane 1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Items) != 1 {
		t.Fatal("template snapshot did not resolve")
	}

	completenessResult := completeness.Check([]completeness.Requirement{{Field: "asset_name", Mandatory: true}, {Field: "serial_number", Mandatory: true}}, map[string]any{"asset_name": "Crane 1", "serial_number": "SN-1"})
	if !completenessResult.Complete {
		t.Fatalf("unexpected incomplete data: %#v", completenessResult.Missing)
	}

	inspectionAggregate, err := inspection.New("inspection-1", "tenant-1", "org-1", types.EnvironmentLive, "asset-1", "annual", time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := inspectionAggregate.Assign("inspector-1"); err != nil {
		t.Fatal(err)
	}
	if err := inspectionAggregate.Start(); err != nil {
		t.Fatal(err)
	}
	if err := inspectionAggregate.RecordFinding(inspection.Finding{ID: "finding-1", AssetID: "asset-1", SectionID: "structure", ItemID: snapshot.Items[0].ID, ItemPrompt: snapshot.Items[0].Prompt, Response: "PASS", Severity: types.SeverityAdvisory, RecordedBy: "inspector-1"}); err != nil {
		t.Fatal(err)
	}
	if err := inspectionAggregate.Complete("passed"); err != nil {
		t.Fatal(err)
	}
	if err := inspectionAggregate.SubmitForReview(); err != nil {
		t.Fatal(err)
	}
	if err := inspectionAggregate.Approve("reviewer-1"); err != nil {
		t.Fatal(err)
	}
	if err := inspectionAggregate.Close(); err != nil {
		t.Fatal(err)
	}
	if inspectionAggregate.Status() != types.InspectionClosed {
		t.Fatal("inspection did not close")
	}

	certificateAggregate, err := certificate.New("certificate-1", "tenant-1", "org-1", types.EnvironmentLive, "CERT-001", inspectionAggregate.ID(), inspectionAggregate.Revision(), "asset-1", "inspector-1", "creator-1", time.Date(2027, 8, 13, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := certificateAggregate.SubmitForApproval("creator-1"); err != nil {
		t.Fatal(err)
	}
	if err := certificateAggregate.Approve("approver-1", nil); err != nil {
		t.Fatal(err)
	}
	if err := certificateAggregate.Sign("signer-1"); err != nil {
		t.Fatal(err)
	}
	if err := certificateAggregate.Issue("issuer-1", time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if certificateAggregate.Status() != certificate.Issued {
		t.Fatal("certificate did not issue")
	}
}
