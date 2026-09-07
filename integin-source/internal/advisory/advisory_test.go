package advisory

import (
	"context"
	"testing"
	"time"
)

type testProvider struct {
	calls  int
	result ProviderResult
}

func (p *testProvider) Generate(context.Context, Request) (ProviderResult, error) {
	p.calls++
	return p.result, nil
}

func TestAdvisoryGenerationForcesNonBlockingAndPreservesTraceMetadata(t *testing.T) {
	provider := &testProvider{result: ProviderResult{Title: "Trend", Summary: "trend", Severity: "ADVISORY", Confidence: .8, Rationale: "Evidence indicates a trend", Provider: "test", Model: "model-1", PromptVersion: "prompt-1"}}
	insight, err := Generate(context.Background(), provider, Request{TenantID: "tenant-1", Zone: ZoneMonitoring, Lens: "TREND_ANOMALY", EvidenceRefs: []string{"event-1"}}, "insight-1", time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if insight.Blocking || insight.TenantID != "tenant-1" || len(insight.EvidenceRefs) != 1 || insight.Provider != "test" {
		t.Fatalf("unexpected insight: %#v", insight)
	}
	if err := EnsureAdvisory(insight); err != nil {
		t.Fatal(err)
	}
}

func TestAIFreeZonesRejectInvocation(t *testing.T) {
	provider := &testProvider{result: ProviderResult{Rationale: "should not run", Confidence: .5}}
	_, err := Generate(context.Background(), provider, Request{TenantID: "tenant-1", Zone: ZoneVerdict}, "insight-1", time.Now())
	if err == nil {
		t.Fatal("verdict zone should reject AI")
	}
	if provider.calls != 0 {
		t.Fatal("AI provider was called in an AI-free zone")
	}
	if AIAllowed(ZoneAuthorization) || !AIAllowed(ZoneMonitoring) {
		t.Fatal("AI-free-zone policy incorrect")
	}
}

func TestAdvisoryValidationRejectsUnsafeMetadata(t *testing.T) {
	provider := &testProvider{result: ProviderResult{Confidence: 2, Rationale: "invalid"}}
	if _, err := Generate(context.Background(), provider, Request{TenantID: "tenant-1", Zone: ZoneRegulation}, "insight-1", time.Now()); err == nil {
		t.Fatal("invalid confidence should fail")
	}
	provider.result = ProviderResult{Confidence: .5}
	if _, err := Generate(context.Background(), provider, Request{TenantID: "tenant-1", Zone: ZoneRegulation}, "insight-1", time.Now()); err == nil {
		t.Fatal("missing rationale should fail")
	}
}
