package standardsync

import (
	"errors"
	"testing"
	"time"
)

func TestDIDRoundTrip(t *testing.T) {
	did := DID("ASME", "B30.5", 2021)
	if want := "did:integin:standard:asme-b30-5-2021"; did != want {
		t.Fatalf("DID = %q, want %q", did, want)
	}
	body, code, year, ok := ParseDID(did)
	if !ok || body != "asme" || code != "b30-5" || year != 2021 {
		t.Fatalf("ParseDID(%q) = %q %q %d %v", did, body, code, year, ok)
	}
	// Slug round trip must reconstruct the same DID.
	rebuilt := DID(body, code, year)
	if rebuilt != did {
		t.Fatalf("rebuilt DID = %q, want %q", rebuilt, did)
	}
}

func TestDIDSlugCollapsesPunctuation(t *testing.T) {
	got := DID("ISO", "4309", 2010)
	if want := "did:integin:standard:iso-4309-2010"; got != want {
		t.Fatalf("DID = %q, want %q", got, want)
	}
}

func TestParseDIDRejectsMalformed(t *testing.T) {
	for _, value := range []string{
		"",
		"did:integin:standard:",
		"did:integin:standard:asme-2019",
		"did:integin:standard:-b30-5-2021",
		"did:integin:standard:asmeB30.5-2021",
		"urn:integin:standard:iso-4309-2010",
	} {
		if _, _, _, ok := ParseDID(value); ok {
			t.Fatalf("ParseDID(%q) unexpectedly succeeded", value)
		}
	}
}

func validCard() StandardMetadataCard {
	return StandardMetadataCard{
		StandardDID:      DID("ASME", "B30.5", 2021),
		StandardBody:     "ASME",
		Code:             "B30.5",
		RevisionYear:     2021,
		Title:            "Mobile and Locomotive Cranes",
		ScopeAbstract:    "Applies to mobile and locomotive cranes in material handling operations.",
		LifecycleState:   LifecycleActive,
		OfficialStoreURL: "https://www.asme.org/codes-standards",
		PublishedDate:    time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestValidateCitationContract(t *testing.T) {
	base := validCard()
	if err := base.Validate(); err != nil {
		t.Fatalf("valid card rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*StandardMetadataCard)
	}{
		{"empty DID", func(c *StandardMetadataCard) { c.StandardDID = "" }},
		{"mismatched DID", func(c *StandardMetadataCard) { c.StandardDID = "did:integin:standard:iso-4309-2010" }},
		{"published zero", func(c *StandardMetadataCard) { c.PublishedDate = time.Time{} }},
		{"bad lifecycle", func(c *StandardMetadataCard) { c.LifecycleState = "RETIRED" }},
		{"http url", func(c *StandardMetadataCard) { c.OfficialStoreURL = "http://www.asme.org/codes-standards" }},
		{"pdf url", func(c *StandardMetadataCard) { c.OfficialStoreURL = "https://www.asme.org/documents/std.pdf" }},
		{"document path", func(c *StandardMetadataCard) { c.OfficialStoreURL = "https://www.asme.org/files/b30.5" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := base
			tt.mutate(&card)
			if err := card.Validate(); !errors.Is(err, ErrMalformedCard) {
				t.Fatalf("Validate = %v, want ErrMalformedCard", err)
			}
		})
	}
}

func TestLifecycleStateBinding(t *testing.T) {
	if !LifecycleActive.Binding() {
		t.Fatal("ACTIVE must bind")
	}
	for _, state := range []LifecycleState{LifecycleSuperseded, LifecycleWithdrawn, LifecycleDraft} {
		if state.Binding() {
			t.Fatalf("%s must not bind", state)
		}
		if !state.Valid() {
			t.Fatalf("%s must be a valid state", state)
		}
	}
}

func TestArticlesNeverIncludeScope(t *testing.T) {
	card := validCard()
	for _, article := range card.Articles() {
		if article == card.ScopeAbstract || contains(article, "Applies to") {
			t.Fatalf("Articles() leaked scope abstract token %q", article)
		}
	}
}
