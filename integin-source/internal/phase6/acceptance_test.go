package phase6

import (
	"context"
	"testing"
	"time"

	"integin/internal/advisory"
	"integin/internal/canary"
	"integin/internal/monitoring"
)

type acceptanceProvider struct{}

func (acceptanceProvider) Generate(context.Context, advisory.Request) (advisory.ProviderResult, error) {
	return advisory.ProviderResult{Title: "advisory", Summary: "secondary", Severity: "ADVISORY", Confidence: .9, Rationale: "evidence-linked rationale", Provider: "test", Model: "test-model", PromptVersion: "v1"}, nil
}

func TestPhase6AdvisoryWorkflow(t *testing.T) {
	repository := monitoring.NewRepository()
	engine, err := monitoring.NewEngine(acceptanceProvider{}, repository, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	result, err := engine.Analyze(context.Background(), "tenant-1", monitoring.LensComplianceGap, []string{"standard_version"}, map[string]any{"standard_version": "2026"}, []string{"standard-1"}, "insight-1")
	if err != nil || result.Insight.Blocking {
		t.Fatalf("advisory workflow failed: %#v %v", result, err)
	}
	impacts, err := AnalyzeRegulation(RegulationChange{ID: "change-1", TenantID: "tenant-1", StandardCode: "ISO-1", NewVersion: "2026", ChangedClauses: []string{"7.1"}, Summary: "changed"}, []TemplateReference{{ID: "template-1", TenantID: "tenant-1", StandardCodes: []string{"ISO-1"}, ClauseRefs: []string{"7.1"}}}, time.Now())
	if err != nil || len(impacts) != 1 || impacts[0].Advisory.Blocking {
		t.Fatalf("regulation workflow failed: %#v %v", impacts, err)
	}
	merge := MergeFields(map[string]any{"a": 1}, map[string]any{"a": 2}, map[string]any{"a": 3})
	if len(merge.Conflicts) != 1 {
		t.Fatal("conflicting field was not preserved")
	}
	request, err := NewClientRequest("request-1", "tenant-1", "client-1", "asset-1", "inspection")
	if err != nil {
		t.Fatal(err)
	}
	if err := request.Review("tenant-1", "reviewer"); err != nil {
		t.Fatal(err)
	}
	if err := request.Approve("tenant-1"); err != nil {
		t.Fatal(err)
	}
	deployment, err := canary.New("deploy-1", "candidate", "baseline")
	if err != nil {
		t.Fatal(err)
	}
	deployment.SetHealth(canary.HealthHealthy)
	deployment.MarkRollbackReady(true)
	if err := deployment.SetTraffic(10); err != nil || !deployment.RollbackReady {
		t.Fatalf("canary workflow failed: %#v %v", deployment, err)
	}
	if _, err := advisory.Generate(context.Background(), acceptanceProvider{}, advisory.Request{TenantID: "tenant-1", Zone: advisory.ZoneCertificate}, "bad", time.Now()); err == nil {
		t.Fatal("certificate AI-free zone was bypassed")
	}
}
