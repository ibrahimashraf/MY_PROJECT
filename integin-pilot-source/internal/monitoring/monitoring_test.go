package monitoring

import (
	"context"
	"testing"
	"time"

	"integin/internal/advisory"
)

type monitoringProvider struct{}

func (monitoringProvider) Generate(context.Context, advisory.Request) (advisory.ProviderResult, error) {
	return advisory.ProviderResult{Title: "Advisory", Summary: "summary", Severity: "ADVISORY", Confidence: .7, Rationale: "Evidence indicates a secondary trend", Provider: "test-provider", Model: "model-1", PromptVersion: "prompt-1", Limitations: []string{"not a primary decision"}}, nil
}

func TestMonitoringSupportsSevenLensesAndReasoningTrace(t *testing.T) {
	repository := NewRepository()
	engine, err := NewEngine(monitoringProvider{}, repository, func() time.Time { return time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	if len(Lenses()) != 7 {
		t.Fatalf("expected seven lenses, got %d", len(Lenses()))
	}
	result, err := engine.Analyze(context.Background(), "tenant-1", LensTrendAnomaly, []string{"inspection_count", "finding_count"}, map[string]any{"inspection_count": 10}, []string{"event-1"}, "insight-1")
	if err != nil {
		t.Fatal(err)
	}
	if result.Insight.Blocking || result.Trace.Provider != "test-provider" || len(result.Trace.EvidenceRefs) != 1 {
		t.Fatalf("unexpected advisory trace: %#v", result)
	}
	if len(repository.List("tenant-1")) != 1 {
		t.Fatal("trace was not persisted")
	}
}

func TestMonitoringRepositoryIsTenantScopedAndRejectsBlockingInsight(t *testing.T) {
	repository := NewRepository()
	bad := Result{Insight: advisory.Insight{ID: "bad", TenantID: "tenant-1", Lens: string(LensSafetyRisk), Blocking: true, Rationale: "unsafe"}, Trace: Trace{TenantID: "tenant-1", Lens: LensSafetyRisk}}
	if err := repository.Save(bad); err == nil {
		t.Fatal("blocking advisory should be rejected")
	}
	good := Result{Insight: advisory.Insight{ID: "good", TenantID: "tenant-2", Lens: string(LensSafetyRisk), Rationale: "advisory"}, Trace: Trace{TenantID: "tenant-2", Lens: LensSafetyRisk}}
	if err := repository.Save(good); err != nil {
		t.Fatal(err)
	}
	if len(repository.List("tenant-1")) != 0 || len(repository.List("tenant-2")) != 1 {
		t.Fatal("monitoring result crossed tenant boundary")
	}
}
