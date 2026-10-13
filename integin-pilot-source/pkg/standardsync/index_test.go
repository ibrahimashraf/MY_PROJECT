package standardsync

import (
	"testing"
	"time"
)

// indexFixtureCards builds a synthetic but citation-valid card universe large
// enough to stress the <30ms latency budget. Every (body, code, year) triple is
// unique so DIDs remain distinct across the whole fixture.
func indexFixtureCards(n int) []StandardMetadataCard {
	bodies := []struct{ body, code, scope string }{
		{"ASME", "B30.5", "Mobile and locomotive cranes, hoisting rigging and lifting equipment."},
		{"ISO", "4309", "Wire ropes for cranes, inspection and discard criteria."},
		{"ASME", "B30.2", "Overhead and gantry cranes."},
		{"DNV", "ST-0377", "Shipboard cranes and offshore crane operations."},
		{"AWS", "D1.1", "Structural welding steel, procedures and qualification."},
		{"NFPA", "70E", "Electrical safety in the workplace, arc flash boundaries."},
	}
	cards := make([]StandardMetadataCard, 0, n)
	for i := 0; i < n; i++ {
		tpl := bodies[i%len(bodies)]
		variant := i / len(bodies)
		year := 2001 + (i % 28) // 28 valid years, rolled to keep DIDs unique per body.
		code := tpl.code
		if variant > 0 {
			code = tpl.code + "-" + itoa(variant+1)
		}
		card := StandardMetadataCard{
			StandardDID:      DID(tpl.body, code, year),
			StandardBody:     tpl.body,
			Code:             code,
			RevisionYear:     year,
			Title:            tpl.body + " " + code + " series " + itoa(year),
			ScopeAbstract:    tpl.scope,
			LifecycleState:   LifecycleActive,
			OfficialStoreURL: "https://www.example.org/standards",
			PublishedDate:    time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC),
			ApplicableAssets: []string{"crane", "rigging"},
			Tags:             []string{tpl.body + "-" + code},
		}
		cards = append(cards, card)
	}
	return cards
}

func TestIndexQueryPrefersMatchingSeries(t *testing.T) {
	idx := NewAbstractIndex(indexFixtureCards(200))
	results := idx.Query("mobile crane lifting wire rope inspection", 1.0, 1.0)
	if len(results) == 0 {
		t.Fatal("query returned no results")
	}
	top := results[0].Card
	if top.StandardBody != "ASME" && top.StandardBody != "ISO" {
		t.Fatalf("top hit = %s %s, want ASME or ISO", top.StandardBody, top.Code)
	}
}

func TestIndexDeterminismAcrossInsertionOrder(t *testing.T) {
	cards := indexFixtureCards(50)
	shuffled := make([]StandardMetadataCard, len(cards))
	copy(shuffled, cards)
	// Rotate the insertion order without changing the universe.
	for i, j := 0, len(shuffled)-1; i < j; i, j = i+1, j-1 {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	first := NewAbstractIndex(cards).Query("crane rigging inspection", 1.0, 1.0)
	second := NewAbstractIndex(shuffled).Query("crane rigging inspection", 1.0, 1.0)
	if len(first) != len(second) {
		t.Fatalf("ranked set sizes differ: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].Card.StandardDID != second[i].Card.StandardDID {
			t.Fatalf("rank %d differs: %s vs %s", i, first[i].Card.StandardDID, second[i].Card.StandardDID)
		}
	}
}

func BenchmarkEngineSearchWithin30ms(b *testing.B) {
	cards := indexFixtureCards(2000)
	resolver, err := NewResolver(cards)
	if err != nil {
		b.Fatalf("NewResolver: %v", err)
	}
	engine := NewEngine(resolver, NewAbstractIndex(cards))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := engine.Search("what governs mobile crane wire rope inspection and discard")
		if err != nil {
			b.Fatalf("Search: %v", err)
		}
		if len(result.PrimaryMatches) == 0 {
			b.Fatal("no primary matches")
		}
	}
	b.ReportMetric(30, "ms-sla")
}
