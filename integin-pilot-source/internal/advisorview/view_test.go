package advisorview

import (
	"testing"
	"time"

	"integin/internal/advisory"
)

func TestAdvisorPanelOnlyExposesApprovedAdvisoryFields(t *testing.T) {
	insights := []advisory.Insight{{ID: "insight-1", TenantID: "tenant-1", Lens: "SAFETY_RISK", Title: "Risk", Summary: "summary", Severity: "ADVISORY", Confidence: .8, Rationale: "rationale", EvidenceRefs: []string{"event-1"}, Provider: "python", Model: "model", PromptVersion: "v1", Limitations: []string{"secondary only"}, CreatedAt: time.Now()}, {ID: "insight-2", TenantID: "tenant-2", Lens: "SAFETY_RISK", Rationale: "other"}}
	panel, err := BuildPanel("tenant-1", insights)
	if err != nil || !panel.SecondaryOnly || len(panel.Insights) != 1 {
		t.Fatalf("unexpected panel: %#v %v", panel, err)
	}
	if panel.Insights[0].Blocking {
		t.Fatal("advisor panel cannot expose blocking insight")
	}
	panel.Insights[0].EvidenceRefs[0] = "changed"
	if insights[0].EvidenceRefs[0] != "event-1" {
		t.Fatal("advisor panel aliased source evidence")
	}
}

func TestAdvisorViewRejectsBlockingInsightsAndValidatesDeployment(t *testing.T) {
	_, err := FromInsight(advisory.Insight{ID: "bad", TenantID: "tenant-1", Blocking: true, Rationale: "bad"})
	if err == nil {
		t.Fatal("blocking insight should not enter advisor view")
	}
	metadata := DeploymentMetadata{ServiceName: "ai-service", Endpoint: "http://ai-service:8000", ContractVersion: "v1", Baseline: "1.0.0", Candidate: "1.1.0", Health: "HEALTHY", TrafficPercent: 10, RollbackReady: true}
	if err := metadata.Validate(); err != nil {
		t.Fatal(err)
	}
	metadata.TrafficPercent = 101
	if err := metadata.Validate(); err == nil {
		t.Fatal("invalid traffic should fail")
	}
}
