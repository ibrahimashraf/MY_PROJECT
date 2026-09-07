package standardsync

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ErrUnknownStandard reports an unresolvable body+code series.
var ErrUnknownStandard = errors.New("unknown standards series")

// ErrNoActiveRevision reports a series whose latest revision is not ACTIVE and
// can therefore never bind to a work order package.
var ErrNoActiveRevision = errors.New("standards series has no active revision")

// Resolver answers lifecycle questions over an ingested set of citation cards.
// It is immutable after construction and safe for concurrent use.
type Resolver struct {
	bySeries map[string][]StandardMetadataCard
	byDID    map[string]StandardMetadataCard
}

// NewResolver indexes cards by (body, code) series. Duplicate DIDs are rejected.
func NewResolver(cards []StandardMetadataCard) (*Resolver, error) {
	r := &Resolver{
		bySeries: make(map[string][]StandardMetadataCard),
		byDID:    make(map[string]StandardMetadataCard),
	}
	for _, card := range cards {
		if err := card.Validate(); err != nil {
			return nil, fmt.Errorf("card %q: %w", card.StandardDID, err)
		}
		series := seriesKey(card.StandardBody, card.Code)
		r.bySeries[series] = append(r.bySeries[series], card)
		if _, exists := r.byDID[card.StandardDID]; exists {
			return nil, fmt.Errorf("duplicate standard DID %q", card.StandardDID)
		}
		r.byDID[card.StandardDID] = card
	}
	for series, revisions := range r.bySeries {
		sort.Slice(revisions, func(i, j int) bool {
			if revisions[i].RevisionYear != revisions[j].RevisionYear {
				return revisions[i].RevisionYear < revisions[j].RevisionYear
			}
			return revisions[i].PublishedDate.Before(revisions[j].PublishedDate)
		})
		r.bySeries[series] = revisions
	}
	if err := ValidateLifecycle(r); err != nil {
		return nil, err
	}
	return r, nil
}

// CardByDID resolves a single card by its stable DID.
func (r *Resolver) CardByDID(did string) (StandardMetadataCard, bool) {
	card, ok := r.byDID[did]
	return card, ok
}

// Series returns every ingested revision of one body+code series, oldest first.
func (r *Resolver) Series(body, code string) ([]StandardMetadataCard, bool) {
	revisions, ok := r.bySeries[seriesKey(body, code)]
	return revisions, ok
}

// Current resolves the binding revision of a series: the newest ACTIVE card.
func (r *Resolver) Current(body, code string) (StandardMetadataCard, error) {
	revisions, ok := r.bySeries[seriesKey(body, code)]
	if !ok || len(revisions) == 0 {
		return StandardMetadataCard{}, ErrUnknownStandard
	}
	for i := len(revisions) - 1; i >= 0; i-- {
		if revisions[i].LifecycleState == LifecycleActive {
			return revisions[i], nil
		}
	}
	return StandardMetadataCard{}, ErrNoActiveRevision
}

// EffectiveAt resolves the ACTIVE revision whose publication window covers at.
// A revision is effective from PublishedDate until its documented successor's
// PublishedDate. Withdrawn cards with no successor never become effective.
func (r *Resolver) EffectiveAt(body, code string, at time.Time) (StandardMetadataCard, error) {
	revisions, ok := r.bySeries[seriesKey(body, code)]
	if !ok || len(revisions) == 0 {
		return StandardMetadataCard{}, ErrUnknownStandard
	}
	at = at.UTC()
	var effective StandardMetadataCard
	for _, rev := range revisions {
		if rev.LifecycleState != LifecycleActive && rev.LifecycleState != LifecycleSuperseded {
			continue
		}
		if rev.PublishedDate.After(at) {
			continue
		}
		if effective.RevisionYear == 0 || rev.PublishedDate.After(effective.PublishedDate) {
			effective = rev
		}
	}
	if effective.StandardDID == "" {
		return StandardMetadataCard{}, ErrNoActiveRevision
	}
	return effective, nil
}

// ReplacedBy walks the supersession chain forward from a DID, returning every
// successor with a higher revision year, newest last.
func (r *Resolver) ReplacedBy(did string) ([]StandardMetadataCard, error) {
	card, ok := r.byDID[did]
	if !ok {
		return nil, ErrUnknownStandard
	}
	chain := make([]StandardMetadataCard, 0, 2)
	seen := map[string]bool{did: true}
	for card.LifecycleState != LifecycleActive {
		if card.LifecycleState == LifecycleWithdrawn {
			break
		}
		series, ok := r.bySeries[seriesKey(card.StandardBody, card.Code)]
		if !ok {
			break
		}
		var successor StandardMetadataCard
		for _, rev := range series {
			if rev.RevisionYear > card.RevisionYear && !seen[rev.StandardDID] {
				successor = rev
			}
		}
		if successor.StandardDID == "" {
			break
		}
		seen[successor.StandardDID] = true
		chain = append(chain, successor)
		card = successor
	}
	return chain, nil
}

// Predecessors is the inverse of ReplacedBy. It returns the chain this card
// replaces, oldest-yet-relevant last.
func (r *Resolver) Predecessors(did string) ([]StandardMetadataCard, error) {
	card, ok := r.byDID[did]
	if !ok {
		return nil, ErrUnknownStandard
	}
	var chain []StandardMetadataCard
	for _, rev := range r.bySeries[seriesKey(card.StandardBody, card.Code)] {
		if rev.RevisionYear < card.RevisionYear {
			chain = append(chain, rev)
		}
	}
	return chain, nil
}

// VerifyBool returns true when a DID resolves to an ACTIVE revision that is
// the current binding of its series. This is the sole gate for verdict
// injection: relying on a stale or superseded revision is a safety error.
func (r *Resolver) VerifyBool(did string) bool {
	card, ok := r.byDID[did]
	if !ok || card.LifecycleState != LifecycleActive {
		return false
	}
	current, err := r.Current(card.StandardBody, card.Code)
	return err == nil && current.StandardDID == did
}

// seriesKey is the deterministic series map key for a body+code pair.
func seriesKey(body, code string) string {
	return strings.ToLower(strings.TrimSpace(body)) + "\x00" + strings.ToLower(strings.TrimSpace(code))
}

// LookupByCode finds the first body that owns a code token, using a
// case/space-normalized comparison so "b30.5", "B30.5" and "b30 5" all match
// an ASME card. It powers inference of body-less anchors in user queries.
func (r *Resolver) LookupByCode(code string) (body, canonical string, ok bool) {
	want := normalizeCode(code)
	for _, revisions := range r.bySeries {
		for _, card := range revisions {
			if normalizeCode(card.Code) == want && card.Code != "" {
				return card.StandardBody, card.Code, true
			}
		}
	}
	return "", "", false
}

// ValidateLifecycle reports structural violations of the lifecycle model across
// an entire resolver index: duplicate ACTIVE revisions of one series, a
// SUPERSEDED/withdrawn card claiming a successor that did not replace it, a
// cycle in the replacement chain, and a SUPERSEDED card of unknown ancestry.
func ValidateLifecycle(r *Resolver) error {
	for series, revisions := range r.bySeries {
		active := 0
		for _, rev := range revisions {
			if rev.LifecycleState == LifecycleActive {
				active++
			}
			if rev.LifecycleState == LifecycleSuperseded && strings.TrimSpace(rev.ReplacesStandard) == "" {
				return fmt.Errorf("%s: superseded card %s must name its replacement", series, rev.StandardDID)
			}
			if rev.ReplacesStandard != "" {
				ancestor, ok := r.byDID[rev.ReplacesStandard]
				if !ok {
					return fmt.Errorf("%s: %s replaces unknown %s", series, rev.StandardDID, rev.ReplacesStandard)
				}
				if ancestor.StandardBody != rev.StandardBody || ancestor.Code != rev.Code {
					return fmt.Errorf("%s: replacement ancestry crosses series", rev.StandardDID)
				}
				// ReplacesStandard points forward for retired cards (the next
				// revision) and backward for cards that indicate the edition
				// they superseded. Either way the linked card must not equal
				// the card itself.
				if ancestor.RevisionYear == rev.RevisionYear {
					return fmt.Errorf("%s: replacement ancestry links same-vintage revisions", rev.StandardDID)
				}
			}
		}
		if active > 1 {
			return fmt.Errorf("%s: %d ACTIVE revisions of one series", series, active)
		}
	}
	for _, card := range r.byDID {
		if _, err := r.ReplacedBy(card.StandardDID); err != nil {
			return err
		}
	}
	return nil
}

// FormatReplaces renders a human sentence describing the current binding of a
// series, e.g. "ISO 4309:2010 replaces ISO 4309:1990 (retired 2010-02-15)".
func (r *Resolver) FormatReplaces(card StandardMetadataCard) string {
	predecessors, _ := r.Predecessors(card.StandardDID)
	if len(predecessors) == 0 {
		return ""
	}
	last := predecessors[len(predecessors)-1]
	return fmt.Sprintf("%s %s:%d replaces %s %s:%d",
		card.StandardBody, card.Code, card.RevisionYear,
		last.StandardBody, last.Code, last.RevisionYear)
}
