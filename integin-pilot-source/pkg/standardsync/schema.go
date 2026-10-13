// Package standardsync is the copyright-safe standards discovery engine.
//
// It resolves and searches engineering standard metadata (ASME, ISO, BSI, DIN,
// ASTM, DNV, IEC, LEEA, NFPA, AWS, JIS, API) as citation cards only. Cards carry
// a scope abstract, lifecycle state, applicable assets, and an official store
// URL for the authoritative publisher. Full copyrighted text is never stored,
// synthesized, or re-emitted; every search answer is a deterministic synthesis
// over the bounded metadata an operator has ingested.
package standardsync

import (
	"errors"
	"strings"
	"time"
	"unicode"
)

// LifecycleState is the publication lifecycle of a standard revision. Only
// ACTIVE revisions may be bound to a work order package or gate a verdict.
type LifecycleState string

const (
	LifecycleActive     LifecycleState = "ACTIVE"
	LifecycleSuperseded LifecycleState = "SUPERSEDED"
	LifecycleWithdrawn  LifecycleState = "WITHDRAWN"
	LifecycleDraft      LifecycleState = "DRAFT"
)

// Valid reports whether s is one of the four supported lifecycle states.
func (s LifecycleState) Valid() bool {
	switch s {
	case LifecycleActive, LifecycleSuperseded, LifecycleWithdrawn, LifecycleDraft:
		return true
	default:
		return false
	}
}

// Binding is the only lifecycle state that may be injected into a work order
// manifest or gate an engineering verdict.
func (s LifecycleState) Binding() bool { return s == LifecycleActive }

// StandardMetadataCard is one revision of an engineering standard. It is a
// citation card, never raw standard text: safety-critical scope information is
// carried by ScopeAbstract exclusively from operator-supplied ingest data.
type StandardMetadataCard struct {
	StandardDID      string         `json:"standard_did"`  // "did:integin:standard:asme-b30.5-2021"
	StandardBody     string         `json:"standard_body"` // "ASME"
	Code             string         `json:"code"`          // "B30.5"
	RevisionYear     int            `json:"revision_year"` // 2021
	Title            string         `json:"title"`
	ScopeAbstract    string         `json:"scope_abstract"`
	LifecycleState   LifecycleState `json:"lifecycle_state"` // ACTIVE / SUPERSEDED / WITHDRAWN / DRAFT
	ReplacesStandard string         `json:"replaces_standard,omitempty"`
	OfficialStoreURL string         `json:"official_store_url"`
	PublishedDate    time.Time      `json:"published_date"`
	ApplicableAssets []string       `json:"applicable_assets,omitempty"`
	Tags             []string       `json:"tags,omitempty"`
}

// ErrMalformedCard reports a metadata card that fails Validate.
var ErrMalformedCard = errors.New("standards metadata card is malformed")

// ErrBadDID reports a standard DID that cannot be parsed or verified.
var ErrBadDID = errors.New("standard DID is invalid")

// Validate enforces the citation-card contract: no empty identity, a parseable
// DID that matches the card fields, a known lifecycle state, and an https
// official store URL that is a storefront (never a raw document link).
func (c StandardMetadataCard) Validate() error {
	switch {
	case strings.TrimSpace(c.StandardDID) == "":
		return ErrMalformedCard
	case strings.TrimSpace(c.StandardBody) == "":
		return ErrMalformedCard
	case strings.TrimSpace(c.Code) == "":
		return ErrMalformedCard
	case c.RevisionYear < 1900 || c.RevisionYear > time.Now().UTC().Year()+3:
		return ErrMalformedCard
	case c.Title == "":
		return ErrMalformedCard
	case !c.LifecycleState.Valid():
		return ErrMalformedCard
	case c.PublishedDate.IsZero() || c.OfficialStoreURL == "":
		return ErrMalformedCard
	}
	if want := DID(c.StandardBody, c.Code, c.RevisionYear); c.StandardDID != want {
		return ErrMalformedCard
	}
	if !validOfficialStoreURL(c.OfficialStoreURL) {
		return ErrMalformedCard
	}
	return nil
}

// Articles returns the card's scope tokens for indexing and synthesis: body,
// code, revision year, and tag list. ScopeAbstract is deliberately excluded so
// the search index never stores copyrighted-adjacent prose beyond the card.
func (c StandardMetadataCard) Articles() []string {
	articles := []string{
		c.StandardBody,
		c.Code,
		strings.ToLower(c.ReplacesStandard),
	}
	articles = append(articles, strings.ToLower(c.Code))
	articles = append(articles, tagTokens(c.Tags)...)
	return articles
}

// DID derives the canonical intrinsec DID for a card revision. The DID is the
// stable key for lifecycle resolution and manifest binding.
func DID(body, code string, revisionYear int) string {
	slug := slugify(body) + "-" + slugify(code)
	if revisionYear > 0 {
		slug += "-" + itoa(revisionYear)
	}
	return "did:integin:standard:" + slug
}

// ParseDID splits a standard DID and reports whether it is well formed.
func ParseDID(value string) (body, code string, revisionYear int, ok bool) {
	const prefix = "did:integin:standard:"
	if !strings.HasPrefix(value, prefix) || len(value) == len(prefix) {
		return "", "", 0, false
	}
	rest := strings.TrimSuffix(value[len(prefix):], "/")
	if rest == "" {
		return "", "", 0, false
	}
	y := lastInt(rest)
	if y < 1900 {
		return "", "", 0, false
	}
	stem := strings.TrimSuffix(rest, "-"+itoa(y))
	firstDash := strings.IndexByte(stem, '-')
	if firstDash <= 0 || firstDash == len(stem)-1 {
		return "", "", 0, false
	}
	body = stem[:firstDash]
	code = stem[firstDash+1:]
	if body == "" || code == "" || !slugEquals(body, strings.ToLower(body)) {
		return "", "", 0, false
	}
	return body, code, y, true
}

// slugify lowercases and collapses non-alphanumeric runs into single hyphens.
func slugify(value string) string {
	var b strings.Builder
	inDash := false
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			inDash = false
			continue
		}
		if !inDash {
			b.WriteRune('-')
			inDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// lastInt extracts the trailing integer revision year, returning 0 when absent.
func lastInt(value string) int {
	var digits []rune
	for i := len(value) - 1; i >= 0; i-- {
		r := rune(value[i])
		if r >= '0' && r <= '9' {
			digits = append([]rune{r}, digits...)
			continue
		}
		if len(digits) > 0 {
			break
		}
		if r != '-' && r != '/' {
			return 0
		}
	}
	if len(digits) == 0 {
		return 0
	}
	n := 0
	for _, d := range digits {
		if n > 20000 {
			return 0
		}
		n = n*10 + int(d-'0')
	}
	return n
}

// itoa formats a small non-negative integer without importing strconv ordering
// concerns; the compiler keeps this dependency-minimal.
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[i:])
}

// slugEquals compares a stored slug against a lower-case candidate used for
// cold path validation.
func slugEquals(a, b string) bool { return a == b }

func tagTokens(tags []string) []string {
	var tokens []string
	for _, tag := range tags {
		for _, token := range strings.FieldsFunc(tag, func(r rune) bool {
			return unicode.IsSpace(r) || r == ',' || r == ';' || r == '/'
		}) {
			tokens = append(tokens, strings.ToLower(token))
		}
	}
	return tokens
}

// validOfficialStoreURL accepts only https storefront URLs: no raw documents.
func validOfficialStoreURL(raw string) bool {
	if !strings.HasPrefix(raw, "https://") {
		return false
	}
	lower := strings.ToLower(raw)
	if strings.Contains(lower, ".pdf") {
		return false
	}
	for _, forbidden := range []string{"/documents/", "/files/", "%2f", "%2F"} {
		if strings.Contains(lower, forbidden) {
			return false
		}
	}
	return true
}

// HybridSearchResult is the deterministic answer to a natural-language query.
// QuerySummary is synthesized exclusively from ingested citation metadata.
type HybridSearchResult struct {
	QuerySummary      string         `json:"query_summary"`
	PrimaryMatches    []Match        `json:"primary_matches"`
	DeprecatedMatches []Match        `json:"deprecated_matches"`
	SuggestedActions  []string       `json:"suggested_actions"`
	Query             string         `json:"query"`
	Elapsed           time.Duration  `json:"elapsed_ms"`
	Anchors           []CodeAnchor   `json:"anchors,omitempty"`
	Metadata          SearchMetadata `json:"metadata"`
}

// Match is one scored card in a search answer. Excerpt is a bounded excerpt of
// ScopeAbstract (never raw standard text).
type Match struct {
	DID      string  `json:"did"`
	Body     string  `json:"body"`
	Code     string  `json:"code"`
	Revision int     `json:"revision_year"`
	State    string  `json:"lifecycle_state"`
	Title    string  `json:"title"`
	Excerpt  string  `json:"excerpt"`
	Score    float64 `json:"score"`
	StoreURL string  `json:"official_store_url"`
}

// CodeAnchor is a recognized standards code mention in the query, e.g.
// "ASME B30.5". Anchors drive precise lexical retrieval for complex questions.
type CodeAnchor struct {
	Body string `json:"body"`
	Code string `json:"code"`
}

// SearchMetadata records what happened during a search for auditability.
type SearchMetadata struct {
	CardsIndexed int `json:"cards_indexed"`
	LexicalHits  int `json:"lexical_hits"`
	VectorHits   int `json:"vector_hits"`
}
