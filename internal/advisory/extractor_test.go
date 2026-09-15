package advisory

import (
	"strings"
	"sync"
	"testing"
	"time"
)

const sampleRegulatoryText = `# INTEGIN Crane Inspection Regulation 2026

TITLE: INTEGIN Crane Inspection Regulation 2026
AUTHORITY: INTEGIN Standards Board
EFFECTIVE DATE: 2026-03-01

Section 1 — Scope
Section 2 — Definitions
Checklist A(1): lift point visible damage
Checklist A(2): rope fleet angle
Annex III — Load test records
`

func TestDocumentExtractorExtractsStructuralMetadata(t *testing.T) {
	extractor := DocumentExtractor{}
	extract, err := extractor.Extract("tenant-1", sampleRegulatoryText, time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if extract.Title != "INTEGIN Crane Inspection Regulation 2026" {
		t.Fatalf("unexpected title: %q", extract.Title)
	}
	if extract.Authority != "INTEGIN Standards Board" {
		t.Fatalf("unexpected authority: %q", extract.Authority)
	}
	if extract.EffectiveDate != "2026-03-01" {
		t.Fatalf("unexpected effective date: %q", extract.EffectiveDate)
	}
	if extract.Blocking {
		t.Fatal("document extraction must never be blocking")
	}
	if got, want := len(extract.Clauses), 3; got != want {
		t.Fatalf("expected %d clauses, got %d: %#v", want, got, extract.Clauses)
	}
	if got, want := len(extract.ChecklistRefs), 2; got != want {
		t.Fatalf("expected %d checklist refs, got %d: %#v", want, got, extract.ChecklistRefs)
	}
	if !strings.Contains(extract.Clauses[0], "Section 1") || !strings.Contains(extract.Clauses[2], "Annex III") {
		t.Fatalf("unexpected clause extraction: %#v", extract.Clauses)
	}
}

func TestDocumentExtractorRejectsEmptyInputs(t *testing.T) {
	extractor := DocumentExtractor{}
	if _, err := extractor.Extract("", sampleRegulatoryText, time.Now()); err == nil {
		t.Fatal("empty tenant id should fail")
	}
	if _, err := extractor.Extract("tenant-1", "   ", time.Now()); err == nil {
		t.Fatal("empty regulatory text should fail")
	}
}

func sampleObservations() []DefectObservation {
	return []DefectObservation{
		{ID: "obs-1", Zone: ZoneNDTDefect, Discipline: "NDT_MPI", Severity: "MINOR", ReportedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "obs-2", Zone: ZoneNDTDefect, Discipline: "NDT_MPI", Severity: "MINOR", ReportedAt: time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)},
		{ID: "obs-3", Zone: ZoneNDTDefect, Discipline: "NDT_MPI", Severity: "MAJOR", ReportedAt: time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)},
		{ID: "obs-4", Zone: ZoneNDTDefect, Discipline: "LIFTING_WIRE_ROPE", Severity: "MINOR", ReportedAt: time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)},
	}
}

func TestTrendSummarizerProducesNonBlockingInsight(t *testing.T) {
	summarizer := TrendSummarizer{}
	insight, err := summarizer.Summarize("tenant-1", "NDT_MPI", ZoneNDTDefect, sampleObservations(), []string{"event-9"}, time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if insight.Blocking {
		t.Fatal("trend insight must never be blocking")
	}
	if insight.Severity != "ADVISORY" || insight.Confidence < 0 || insight.Confidence > 1 {
		t.Fatalf("unexpected severity/confidence: %#v", insight)
	}
	if insight.Confidence < 0.45 {
		t.Fatalf("expected bounded confidence floor, got %f", insight.Confidence)
	}
	if !strings.Contains(insight.Title, "NDT_MPI") || !strings.Contains(insight.Summary, "3 of 4") {
		t.Fatalf("unexpected trend content: title=%q summary=%q", insight.Title, insight.Summary)
	}
	if len(insight.EvidenceRefs) != 5 {
		t.Fatalf("expected 5 evidence refs (1 input + 4 observations), got %d", len(insight.EvidenceRefs))
	}
	if err := EnsureAdvisoryStrict(ZoneNDTDefect, insight); err != nil {
		t.Fatal(err)
	}
}

func TestTrendSummarizerRejectsForbiddenState(t *testing.T) {
	summarizer := TrendSummarizer{}
	if _, err := summarizer.Summarize("tenant-1", "lens", ZoneVerdict, sampleObservations(), nil, time.Now()); err == nil {
		t.Fatal("verdict zone should reject trend summarization")
	}
	if _, err := summarizer.Summarize("", "lens", ZoneNDTDefect, sampleObservations(), nil, time.Now()); err == nil {
		t.Fatal("empty tenant id should fail")
	}
	if _, err := summarizer.Summarize("tenant-1", "lens", ZoneNDTDefect, sampleObservations()[:1], nil, time.Now()); err == nil {
		t.Fatal("too few observations should fail")
	}
	distinct := sampleObservations()
	distinct[1].Discipline = "NDT_UT"
	distinct[2].Discipline = "PAUT"
	if _, err := summarizer.Summarize("tenant-1", "lens", ZoneNDTDefect, distinct, nil, time.Now()); err == nil {
		t.Fatal("no recurring pattern should fail")
	}
}

func TestEnsureAdvisoryStrictRejectsBlockingAndNonAIZones(t *testing.T) {
	valid := Insight{TenantID: "tenant-1", Rationale: "advisory", Blocking: false}
	if err := EnsureAdvisoryStrict(ZoneRegulation, valid); err != nil {
		t.Fatal(err)
	}
	blocking := valid
	blocking.Blocking = true
	if err := EnsureAdvisoryStrict(ZoneRegulation, blocking); err == nil {
		t.Fatal("blocking insight must be rejected")
	}
	if err := EnsureAdvisoryStrict(ZoneVerdict, valid); err == nil {
		t.Fatal("non-AI zone must be rejected")
	}
}

func TestExtractorsAreConcurrencySafe(t *testing.T) {
	extractor := DocumentExtractor{}
	summarizer := TrendSummarizer{}
	observations := sampleObservations()
	const goroutines = 32
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			extract, err := extractor.Extract("tenant-1", sampleRegulatoryText, time.Now())
			if err != nil {
				t.Error(err)
				return
			}
			if extract.Blocking || len(extract.Clauses) != 3 {
				t.Errorf("unexpected extract: %#v", extract)
			}
			insight, err := summarizer.Summarize("tenant-1", "NDT_MPI", ZoneNDTDefect, observations, nil, time.Now())
			if err != nil {
				t.Error(err)
				return
			}
			if insight.Blocking || insight.Confidence < 0 || insight.Confidence > 1 {
				t.Errorf("unexpected trend insight: %#v", insight)
			}
			if err := EnsureAdvisoryStrict(ZoneNDTDefect, insight); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}
