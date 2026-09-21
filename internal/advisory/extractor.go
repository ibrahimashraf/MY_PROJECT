package advisory

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	documentTitleRE     = regexp.MustCompile(`(?i)^\s*title\s*[:=]\s*(.+?)\s*$`)
	headingRE           = regexp.MustCompile(`^\s*#{1,6}\s+(.+?)\s*$`)
	documentAuthorityRE = regexp.MustCompile(`(?i)^\s*(?:issuing\s+)?authority\s*[:=]\s*(.+?)\s*$`)
	documentEffectiveRE = regexp.MustCompile(`(?i)^\s*effective(?:\s+date)?\s*[:=]\s*(.+?)\s*$`)
	documentClauseRE    = regexp.MustCompile(`(?i)^\s*(?:article|section|clause|annex|part)\s+[a-z0-9ivxlcdm.()]+.*$`)
	documentChecklistRE = regexp.MustCompile(`(?i)^\s*checklist\b.*$`)
)

const (
	defaultTrendMinObservations = 2
	trendConfidenceFloor        = 0.45
	trendConfidenceSpread       = 0.45
)

type DocumentExtract struct {
	TenantID      string
	Title         string
	Authority     string
	EffectiveDate string
	Clauses       []string
	ChecklistRefs []string
	Blocking      bool
	ExtractedAt   time.Time
}

type DocumentExtractor struct{}

func (d DocumentExtractor) Extract(tenantID, text string, now time.Time) (DocumentExtract, error) {
	if strings.TrimSpace(tenantID) == "" {
		return DocumentExtract{}, errors.New("advisory: extractor requires tenant id")
	}
	if strings.TrimSpace(text) == "" {
		return DocumentExtract{}, errors.New("advisory: extractor requires regulatory text")
	}
	if now.IsZero() {
		now = time.Now()
	}
	out := DocumentExtract{TenantID: tenantID, Blocking: false, ExtractedAt: now.UTC()}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if out.Title == "" {
			if m := documentTitleRE.FindStringSubmatch(trimmed); m != nil {
				out.Title = strings.TrimSpace(m[1])
			} else if m := headingRE.FindStringSubmatch(trimmed); m != nil {
				out.Title = strings.TrimSpace(m[1])
			}
		}
		if out.Authority == "" {
			if m := documentAuthorityRE.FindStringSubmatch(trimmed); m != nil {
				out.Authority = strings.TrimSpace(m[1])
			}
		}
		if out.EffectiveDate == "" {
			if m := documentEffectiveRE.FindStringSubmatch(trimmed); m != nil {
				out.EffectiveDate = strings.TrimSpace(m[1])
			}
		}
		if documentClauseRE.MatchString(trimmed) {
			out.Clauses = append(out.Clauses, trimmed)
		}
		if documentChecklistRE.MatchString(trimmed) {
			out.ChecklistRefs = append(out.ChecklistRefs, trimmed)
		}
	}
	return out, nil
}

type DefectObservation struct {
	ID         string
	Zone       Zone
	Discipline string
	Location   string
	Severity   string
	ReportedAt time.Time
	Findings   []string
}

type TrendSummarizer struct {
	MinObservations int
}

func (s TrendSummarizer) minObservations() int {
	if s.MinObservations < defaultTrendMinObservations {
		return defaultTrendMinObservations
	}
	return s.MinObservations
}

func (s TrendSummarizer) Summarize(tenantID, lens string, zone Zone, observations []DefectObservation, evidenceRefs []string, now time.Time) (Insight, error) {
	if strings.TrimSpace(tenantID) == "" {
		return Insight{}, errors.New("advisory: trend summary requires tenant id")
	}
	if !AIAllowed(zone) {
		return Insight{}, errors.New("advisory: trend summary zone is not among approved AI zones")
	}
	if len(observations) < s.minObservations() {
		return Insight{}, fmt.Errorf("advisory: trend summary requires at least %d observations", s.minObservations())
	}
	if now.IsZero() {
		now = time.Now()
	}
	counts := map[string]int{}
	total := len(observations)
	var recurring string
	var maxCount int
	for _, obs := range observations {
		discipline := strings.TrimSpace(obs.Discipline)
		if discipline == "" {
			continue
		}
		counts[discipline]++
		if counts[discipline] > maxCount {
			maxCount = counts[discipline]
			recurring = discipline
		}
	}
	if recurring == "" || maxCount < defaultTrendMinObservations {
		return Insight{}, errors.New("advisory: no recurring defect pattern found in observations")
	}
	ratio := float64(maxCount) / float64(total)
	confidence := trendConfidenceFloor + trendConfidenceSpread*ratio
	title := fmt.Sprintf("Recurring finding trend — %s", recurring)
	summary := fmt.Sprintf("%d of %d observations recur in discipline %q; advisory review recommended.", maxCount, total, recurring)
	rationale := fmt.Sprintf("Recurrence of %q appeared in %d of %d observations. This summary is non-binding and limited to approved advisory evidence references.", recurring, maxCount, total)
	evidence := append([]string(nil), evidenceRefs...)
	for _, obs := range observations {
		if strings.TrimSpace(obs.ID) != "" {
			evidence = append(evidence, obs.ID)
		}
	}
	return Insight{
		TenantID:      tenantID,
		Lens:          lens,
		Title:         title,
		Summary:       summary,
		Severity:      "ADVISORY",
		Confidence:    confidence,
		Rationale:     rationale,
		EvidenceRefs:  evidence,
		Provider:      "integin-trend-deterministic",
		Model:         "none",
		PromptVersion: "trend/v1",
		Blocking:      false,
		CreatedAt:     now.UTC(),
		Limitations: []string{
			"Advisory only and read-oriented; cannot alter a primary workflow or decision.",
		},
	}, nil
}

func EnsureAdvisoryStrict(zone Zone, insight Insight) error {
	if err := EnsureAdvisory(insight); err != nil {
		return err
	}
	if !AIAllowed(zone) {
		return errors.New("advisory: insight zone is not among approved AI zones")
	}
	return nil
}
