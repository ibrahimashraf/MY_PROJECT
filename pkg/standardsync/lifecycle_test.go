package standardsync

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }

func mustCard(t *testing.T, did, body, code string, year int, state LifecycleState, published string, replaces string) StandardMetadataCard {
	t.Helper()
	return StandardMetadataCard{
		StandardDID:      did,
		StandardBody:     body,
		Code:             code,
		RevisionYear:     year,
		Title:            body + " " + code + " (" + itoa(year) + ")",
		ScopeAbstract:    "Applies to " + code + " equipment and operations.",
		LifecycleState:   state,
		ReplacesStandard: replaces,
		OfficialStoreURL: "https://www.example.org/standards",
		PublishedDate:    mustParse(t, published),
		ApplicableAssets: []string{"mobile-crane", "overhead-crane", "port-equipment"},
		Tags:             []string{body + "-" + code},
	}
}

func mustParse(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}
	return parsed
}

func sampleResolver(t *testing.T) *Resolver {
	t.Helper()
	cards := []StandardMetadataCard{
		mustCard(t, "did:integin:standard:iso-4309-1990", "ISO", "4309", 1990, LifecycleSuperseded, "1990-12-01", "did:integin:standard:iso-4309-2010"),
		mustCard(t, "did:integin:standard:iso-4309-2010", "ISO", "4309", 2010, LifecycleActive, "2010-02-15", "did:integin:standard:iso-4309-1990"),
		mustCard(t, "did:integin:standard:asme-b30-5-2021", "ASME", "B30.5", 2021, LifecycleActive, "2021-05-01", ""),
		mustCard(t, "did:integin:standard:asme-b30-2-2016", "ASME", "B30.2", 2016, LifecycleSuperseded, "2016-01-01", "did:integin:standard:asme-b30-2-2021"),
		mustCard(t, "did:integin:standard:asme-b30-2-2021", "ASME", "B30.2", 2021, LifecycleActive, "2021-05-01", "did:integin:standard:asme-b30-2-2016"),
	}
	resolver, err := NewResolver(cards)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	return resolver
}

func TestResolverCurrentAndEffectiveAt(t *testing.T) {
	r := sampleResolver(t)

	current, err := r.Current("ISO", "4309")
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if current.RevisionYear != 2010 {
		t.Fatalf("Current year = %d, want 2010", current.RevisionYear)
	}

	eff, err := r.EffectiveAt("ISO", "4309", mustParse(t, "2009-06-01"))
	if err != nil {
		t.Fatalf("EffectiveAt early: %v", err)
	}
	if eff.RevisionYear != 1990 {
		t.Fatalf("EffectiveAt(2009) = %d, want 1990", eff.RevisionYear)
	}
	eff, err = r.EffectiveAt("ISO", "4309", mustParse(t, "2011-01-01"))
	if err != nil {
		t.Fatalf("EffectiveAt late: %v", err)
	}
	if eff.RevisionYear != 2010 {
		t.Fatalf("EffectiveAt(2011) = %d, want 2010", eff.RevisionYear)
	}

	if _, err := r.Current("ISO", "404"); !errors.Is(err, ErrUnknownStandard) {
		t.Fatalf("Current unknown = %v, want ErrUnknownStandard", err)
	}
}

func TestResolverBindingAndVerification(t *testing.T) {
	r := sampleResolver(t)

	if !r.VerifyBool("did:integin:standard:iso-4309-2010") {
		t.Fatal("VerifyBool returned false for the governing revision")
	}
	if r.VerifyBool("did:integin:standard:iso-4309-1990") {
		t.Fatal("VerifyBool accepted a superseded revision as binding")
	}
	if r.VerifyBool("did:integin:standard:does-not-exist") {
		t.Fatal("VerifyBool accepted an unknown DID")
	}
}

func TestResolverReplacedBy(t *testing.T) {
	r := sampleResolver(t)

	chain, err := r.ReplacedBy("did:integin:standard:iso-4309-1990")
	if err != nil {
		t.Fatalf("ReplacedBy: %v", err)
	}
	if len(chain) != 1 || chain[0].RevisionYear != 2010 {
		t.Fatalf("chain = %v, want [2010]", chain)
	}

	predecessors, err := r.Predecessors("did:integin:standard:iso-4309-2010")
	if err != nil {
		t.Fatalf("Predecessors: %v", err)
	}
	if len(predecessors) != 1 || predecessors[0].RevisionYear != 1990 {
		t.Fatalf("predecessors = %v, want [1990]", predecessors)
	}
	if got := r.FormatReplaces(chain[0]); !strings.Contains(got, "ISO 4309:2010 replaces ISO 4309:1990") {
		t.Fatalf("FormatReplaces = %q", got)
	}
}

func TestValidateLifecycleRejectsTwoActive(t *testing.T) {
	cards := []StandardMetadataCard{
		mustCard(t, "a", "ISO", "9001", 2015, LifecycleActive, "2015-09-15", ""),
		mustCard(t, "b", "ISO", "9001", 2018, LifecycleActive, "2018-12-01", ""),
	}
	if _, err := NewResolver(cards); err == nil {
		t.Fatal("expected rejection for two ACTIVE revisions of one series")
	}
}

func TestValidateLifecycleRejectsBrokenAncestry(t *testing.T) {
	cards := []StandardMetadataCard{
		mustCard(t, "c", "ASME", "B30.2", 2021, LifecycleActive, "2021-05-01", ""),
		mustCard(t, "d", "ASME", "B30.2", 2016, LifecycleSuperseded, "2016-01-01", ""),
	}
	cards[1].ReplacesStandard = "did:integin:standard:asme-b30-5-2021"
	cards[0].StandardDID = "did:integin:standard:asme-b30-2-2021"
	if _, err := NewResolver(cards); err == nil {
		t.Fatal("expected rejection for cross-series replacement ancestry")
	}
}

func TestResolverConcurrency(t *testing.T) {
	r := sampleResolver(t)
	done := make(chan bool)
	go func() {
		for i := 0; i < 500; i++ {
			_, _ = r.EffectiveAt("ISO", "4309", mustParse(t, "2019-01-01"))
		}
		done <- true
	}()
	for i := 0; i < 500; i++ {
		if !r.VerifyBool("did:integin:standard:asme-b30-5-2021") {
			t.Fatal("concurrent read returned false")
		}
	}
	<-done
}
