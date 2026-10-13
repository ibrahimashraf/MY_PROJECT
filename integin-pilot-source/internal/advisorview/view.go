package advisorview

import (
	"errors"
	"strings"
	"time"

	"integin/internal/advisory"
)

type InsightView struct {
	ID            string
	TenantID      string
	Lens          string
	Title         string
	Summary       string
	Severity      string
	Confidence    float64
	Rationale     string
	EvidenceRefs  []string
	Provider      string
	Model         string
	PromptVersion string
	Limitations   []string
	CreatedAt     time.Time
	Blocking      bool
}
type Panel struct {
	TenantID      string
	Insights      []InsightView
	SecondaryOnly bool
}

func FromInsight(insight advisory.Insight) (InsightView, error) {
	if err := advisory.EnsureAdvisory(insight); err != nil {
		return InsightView{}, err
	}
	if strings.TrimSpace(insight.TenantID) == "" || strings.TrimSpace(insight.ID) == "" {
		return InsightView{}, errors.New("insight identity is required")
	}
	return InsightView{ID: insight.ID, TenantID: insight.TenantID, Lens: insight.Lens, Title: insight.Title, Summary: insight.Summary, Severity: insight.Severity, Confidence: insight.Confidence, Rationale: insight.Rationale, EvidenceRefs: append([]string(nil), insight.EvidenceRefs...), Provider: insight.Provider, Model: insight.Model, PromptVersion: insight.PromptVersion, Limitations: append([]string(nil), insight.Limitations...), CreatedAt: insight.CreatedAt, Blocking: false}, nil
}
func BuildPanel(tenantID string, insights []advisory.Insight) (Panel, error) {
	if strings.TrimSpace(tenantID) == "" {
		return Panel{}, errors.New("tenant id is required")
	}
	panel := Panel{TenantID: tenantID, SecondaryOnly: true, Insights: make([]InsightView, 0)}
	for _, insight := range insights {
		if insight.TenantID != tenantID {
			continue
		}
		view, err := FromInsight(insight)
		if err != nil {
			return Panel{}, err
		}
		panel.Insights = append(panel.Insights, view)
	}
	return panel, nil
}

type DeploymentMetadata struct {
	ServiceName     string
	Endpoint        string
	ContractVersion string
	Baseline        string
	Candidate       string
	Health          string
	TrafficPercent  int
	RollbackReady   bool
}

func (m DeploymentMetadata) Validate() error {
	if strings.TrimSpace(m.ServiceName) == "" || strings.TrimSpace(m.Endpoint) == "" || strings.TrimSpace(m.ContractVersion) == "" || strings.TrimSpace(m.Baseline) == "" || strings.TrimSpace(m.Candidate) == "" {
		return errors.New("deployment identity is incomplete")
	}
	if m.TrafficPercent < 0 || m.TrafficPercent > 100 {
		return errors.New("traffic percent must be between 0 and 100")
	}
	if m.Health != "HEALTHY" && m.Health != "FAILED" && m.Health != "UNKNOWN" {
		return errors.New("invalid deployment health")
	}
	return nil
}
