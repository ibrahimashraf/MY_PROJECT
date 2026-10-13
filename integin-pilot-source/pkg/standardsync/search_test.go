package standardsync

import (
	"strings"
	"testing"
)

func searchFixture(t *testing.T) *Engine {
	t.Helper()
	cards := []StandardMetadataCard{
		mustCard(t, "did:integin:standard:asme-b30-5-2021", "ASME", "B30.5", 2021, LifecycleActive, "2021-05-01", ""),
		mustCard(t, "did:integin:standard:iso-4309-2010", "ISO", "4309", 2010, LifecycleActive, "2010-02-15", "did:integin:standard:iso-4309-1990"),
		mustCard(t, "did:integin:standard:iso-4309-1990", "ISO", "4309", 1990, LifecycleSuperseded, "1990-12-01", "did:integin:standard:iso-4309-2010"),
	}
	resolver, err := NewResolver(cards)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	return NewEngine(resolver, NewAbstractIndex(cards))
}

func TestSearchAnchorsPinCurrentRevision(t *testing.T) {
	e := searchFixture(t)
	result, err := e.Search("what is the current ISO 4309 revision for wire rope inspection")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	foundCurrent := false
	for _, match := range result.PrimaryMatches {
		if match.DID == "did:integin:standard:iso-4309-2010" {
			foundCurrent = true
		}
	}
	if !foundCurrent {
		t.Fatalf("primary matches did not pin the current revision: %+v", result.PrimaryMatches)
	}
	if len(result.Anchors) == 0 {
		t.Fatal("query with ISO 4309 produced no anchors")
	}
}

func TestSearchSupersededRevisionInDeprecated(t *testing.T) {
	e := searchFixture(t)
	result, err := e.Search("ISO 4309 wire rope inspection criteria")
	if err != nil {
		t.Fatal(err)
	}
	historical := false
	for _, match := range result.DeprecatedMatches {
		if match.DID == "did:integin:standard:iso-4309-1990" {
			historical = true
		}
	}
	if !historical {
		t.Fatal("superseded revision missing from DeprecatedMatches")
	}
}

func TestSearchEmptyQueryErrors(t *testing.T) {
	e := searchFixture(t)
	if _, err := e.Search("   "); err == nil {
		t.Fatal("empty query accepted")
	}
}

func TestSearchSummaryNeverLongerThanBudget(t *testing.T) {
	e := searchFixture(t)
	result, err := e.Search("mobile crane boom load chart")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.QuerySummary) > 1200 {
		t.Fatalf("synthesized summary too long: %d", len(result.QuerySummary))
	}
}

func TestExtractAnchorsNormalization(t *testing.T) {
	var got []CodeAnchor
	for _, query := range []string{
		"ASME B30.5 load charts",
		"b30.5 crane boom",
		"B30.5-2021 mobile crane",
		"ISO 4309 discard wire rope",
	} {
		got = extractAnchors(query)
		if len(got) == 0 {
			t.Fatalf("no anchors extracted from %q", query)
		}
	}
	// Code normalization is stable across body-less reformulations: any
	// mention of the B30.5 code normalizes to a code ending in "B30.5".
	for _, query := range []string{"b30.5", "B30.5", "b30.5-2021", "asme b30.5"} {
		anchors := extractAnchors(query)
		found := false
		for _, anchor := range anchors {
			if anchor.Code == "B30.5" || anchor.Code == "B30.5-2021" {
				found = true
			}
		}
		if !found {
			t.Fatalf("query %q produced no B30.5 code anchor: %+v", query, anchors)
		}
	}
}

func TestBodyLessAnchorResolvesOwningBody(t *testing.T) {
	e := searchFixture(t)
	result, err := e.Search("b30.5 mobile crane load charts")
	if err != nil {
		t.Fatal(err)
	}
	pinned := false
	for _, anchor := range result.Anchors {
		if anchor.Body == "ASME" && anchor.Code == "B30.5" {
			pinned = true
		}
	}
	if !pinned {
		t.Fatalf("body-less b30.5 query did not resolve to ASME: %+v", result.Anchors)
	}
	if len(result.PrimaryMatches) == 0 {
		t.Fatal("no primary match for b30.5 query")
	}
}

func TestSummariesOnlyUseCitationMetadata(t *testing.T) {
	e := searchFixture(t)
	result, err := e.Search("wire rope inspection discard criteria")
	if err != nil {
		t.Fatal(err)
	}
	for _, word := range []string{".pdf", "/documents/", "/files/"} {
		if strings.Contains(result.QuerySummary, word) {
			t.Fatalf("summary leaked download path marker %q", word)
		}
	}
}
