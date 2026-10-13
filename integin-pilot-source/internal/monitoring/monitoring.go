package monitoring

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"integin/internal/advisory"
)

type Lens string

const (
	LensAssetIntegrity       Lens = "ASSET_INTEGRITY"
	LensSafetyRisk           Lens = "SAFETY_RISK"
	LensComplianceGap        Lens = "COMPLIANCE_GAP"
	LensProcessDeviation     Lens = "PROCESS_DEVIATION"
	LensDocumentationQuality Lens = "DOCUMENTATION_QUALITY"
	LensTrendAnomaly         Lens = "TREND_ANOMALY"
	LensDataCompleteness     Lens = "DATA_COMPLETENESS"
)

func Lenses() []Lens {
	return []Lens{LensAssetIntegrity, LensSafetyRisk, LensComplianceGap, LensProcessDeviation, LensDocumentationQuality, LensTrendAnomaly, LensDataCompleteness}
}
func validLens(lens Lens) bool {
	for _, candidate := range Lenses() {
		if candidate == lens {
			return true
		}
	}
	return false
}

type Trace struct {
	ID            string
	TenantID      string
	Lens          Lens
	InputKeys     []string
	EvidenceRefs  []string
	Provider      string
	Model         string
	PromptVersion string
	Rationale     string
	Limitations   []string
	CreatedAt     time.Time
}
type Result struct {
	Insight advisory.Insight
	Trace   Trace
}
type Repository struct {
	mu      sync.RWMutex
	results []Result
}

func NewRepository() *Repository { return &Repository{} }
func (r *Repository) Save(result Result) error {
	if err := advisory.EnsureAdvisory(result.Insight); err != nil {
		return err
	}
	if result.Trace.TenantID != result.Insight.TenantID || result.Trace.Lens != Lens(result.Insight.Lens) {
		return errors.New("trace and insight identity mismatch")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	result.Insight.EvidenceRefs = append([]string(nil), result.Insight.EvidenceRefs...)
	result.Trace.EvidenceRefs = append([]string(nil), result.Trace.EvidenceRefs...)
	result.Trace.InputKeys = append([]string(nil), result.Trace.InputKeys...)
	result.Trace.Limitations = append([]string(nil), result.Trace.Limitations...)
	r.results = append(r.results, result)
	return nil
}
func (r *Repository) List(tenantID string) []Result {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Result, 0)
	for _, item := range r.results {
		if tenantID == "" || item.Insight.TenantID == tenantID {
			item.Insight.EvidenceRefs = append([]string(nil), item.Insight.EvidenceRefs...)
			item.Trace.EvidenceRefs = append([]string(nil), item.Trace.EvidenceRefs...)
			item.Trace.InputKeys = append([]string(nil), item.Trace.InputKeys...)
			item.Trace.Limitations = append([]string(nil), item.Trace.Limitations...)
			result = append(result, item)
		}
	}
	return result
}

type Engine struct {
	provider   advisory.Provider
	repository *Repository
	now        func() time.Time
}

func NewEngine(provider advisory.Provider, repository *Repository, now func() time.Time) (*Engine, error) {
	if provider == nil || repository == nil {
		return nil, errors.New("provider and repository are required")
	}
	if now == nil {
		now = time.Now
	}
	return &Engine{provider: provider, repository: repository, now: now}, nil
}
func (e *Engine) Analyze(ctx context.Context, tenantID string, lens Lens, inputKeys []string, inputs map[string]any, evidenceRefs []string, id string) (Result, error) {
	if strings.TrimSpace(tenantID) == "" || !validLens(lens) {
		return Result{}, errors.New("tenant and valid lens are required")
	}
	insight, err := advisory.Generate(ctx, e.provider, advisory.Request{TenantID: tenantID, Zone: advisory.ZoneMonitoring, Lens: string(lens), Inputs: inputs, EvidenceRefs: evidenceRefs}, id, e.now())
	if err != nil {
		return Result{}, err
	}
	trace := Trace{ID: id + "-trace", TenantID: tenantID, Lens: lens, InputKeys: append([]string(nil), inputKeys...), EvidenceRefs: append([]string(nil), evidenceRefs...), Provider: insight.Provider, Model: insight.Model, PromptVersion: insight.PromptVersion, Rationale: insight.Rationale, Limitations: append([]string(nil), insight.Limitations...), CreatedAt: insight.CreatedAt}
	result := Result{Insight: insight, Trace: trace}
	if err := e.repository.Save(result); err != nil {
		return Result{}, err
	}
	return result, nil
}
