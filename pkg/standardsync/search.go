package standardsync

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// excerptBudget caps every excerpt released in a synthesized answer. Answers
// are assembled from ingested citation metadata only and never reproduce full
// standard text.
const excerptBudget = 220

var (
	// anchorPattern recognizes common standards citations such as "ASME B30.5",
	// "ISO 4309", "DNV-ST-0377" (letters, digits, dots, dashes, slashes).
	anchorPattern = regexp.MustCompile(`(?i)\b([a-zπ]{2,10})\s*([A-Z]{0,3}-?[A-Z0-9][A-Z0-9.\-/]*\d(?:\.[A-Z0-9]+)*)\b`)

	// codeOnlyPattern catches body-less code mentions like "b30.5" or "B30.5".
	// Body inference is resolved against the index in resolveAnchorBodies.
	codeOnlyPattern = regexp.MustCompile(`\b([A-Za-z]*\d+(?:\.\d+)*(?:-[0-9]+)?)\b`)

	// seriesSuffixes are words that follow an anchor in engineering language,
	// used to bound a code token without swallowing the rest of the sentence.
	seriesSuffixes = map[string]bool{"standard": true, "section": true, "clause": true, "annex": true}
)

// Engine is the hybrid search synthesizer. It combines a vector abstract index
// with anchor-aware lexical retrieval and deterministically assembles answers.
type Engine struct {
	resolver *Resolver
	idx      *AbstractIndex
}

// NewEngine builds a search engine over an already-validated resolver.
func NewEngine(resolver *Resolver, idx *AbstractIndex) *Engine {
	return &Engine{resolver: resolver, idx: idx}
}

// Search answers a natural-language question with a synthesized summary plus
// cited primary and deprecated matches. It performs no network access and
// never returns raw standard text.
func (e *Engine) Search(query string) (HybridSearchResult, error) {
	started := time.Now()
	query = strings.TrimSpace(query)
	if query == "" {
		return HybridSearchResult{}, errors.New("query is empty")
	}
	anchors := extractAnchors(query)
	anchors = e.resolveAnchorBodies(anchors)
	scored := e.idx.Query(query, 1.0, 1.0)

	primary := make([]Match, 0, 8)
	deprecated := make([]Match, 0, 8)
	for _, hit := range scored {
		if len(primary)+len(deprecated) >= 16 {
			break
		}
		match := Match{
			DID:      hit.Card.StandardDID,
			Body:     hit.Card.StandardBody,
			Code:     hit.Card.Code,
			Revision: hit.Card.RevisionYear,
			State:    string(hit.Card.LifecycleState),
			Title:    hit.Card.Title,
			Excerpt:  clip(hit.Card.ScopeAbstract, excerptBudget),
			Score:    hit.Score,
			StoreURL: hit.Card.OfficialStoreURL,
		}
		if hit.Card.LifecycleState == LifecycleActive {
			primary = append(primary, match)
		} else {
			deprecated = append(deprecated, match)
		}
	}

	lexHits := len(primary) + len(deprecated)
	vecHits := lexHits
	if anchorsFound(anchors) || lexHits == 0 {
		// Anchor queries surface the exact series first regardless of vector rank.
		primary = e.anchorMatches(query, anchors, primary)
		vecHits = e.idx.Len()
	}

	// Pin current bindings for any anchors so the answer never recommends a
	// superseded revision as authority.
	suggestions := buildSuggestions(e, anchors, len(primary))

	return HybridSearchResult{
		QuerySummary:      synthesize(e, query, anchors, primary, deprecated),
		PrimaryMatches:    primary,
		DeprecatedMatches: deprecated,
		SuggestedActions:  suggestions,
		Query:             query,
		Elapsed:           time.Since(started),
		Anchors:           anchors,
		Metadata:          SearchMetadata{CardsIndexed: e.idx.Len(), LexicalHits: lexHits, VectorHits: vecHits},
	}, nil
}

// anchorMatches moves the ACTIVE revision of any anchored series to the head of
// the primary result set, so exact code questions prioritize authority.
func (e *Engine) anchorMatches(query string, anchors []CodeAnchor, primary []Match) []Match {
	for _, anchor := range anchors {
		current, err := e.resolver.Current(anchor.Body, anchor.Code)
		if err != nil {
			continue
		}
		match := Match{
			DID:      current.StandardDID,
			Body:     current.StandardBody,
			Code:     current.Code,
			Revision: current.RevisionYear,
			State:    string(current.LifecycleState),
			Title:    current.Title,
			Excerpt:  clip(current.ScopeAbstract, excerptBudget),
			Score:    2.0,
			StoreURL: current.OfficialStoreURL,
		}
		moved := false
		for i, existing := range primary {
			if existing.DID == match.DID {
				primary[i] = match
				moved = true
				break
			}
		}
		if !moved {
			primary = append([]Match{match}, primary...)
		}
	}
	return primary
}

// synthesize assembles the human answer from ingested metadata and citations.
// The shape is deterministic: a lede naming the governing current revision for
// any anchors, then a brief scope synthesis that paraphrases card abstracts.
func synthesize(e *Engine, query string, anchors []CodeAnchor, primary, deprecated []Match) string {
	var b strings.Builder
	if len(anchors) > 0 {
		b.WriteString(ledeForAnchors(e, anchors, primary))
	}
	if len(primary) > 0 {
		top := primary[0]
		b.WriteString(" The governing citation is ")
		b.WriteString(top.DID)
		b.WriteString(" (")
		b.WriteString(top.Title)
		b.WriteString("). Scope: ")
		b.WriteString(clip(top.Excerpt, excerptBudget))
		b.WriteString(" Official publisher metadata: ")
		b.WriteString(top.StoreURL)
	}
	if len(deprecated) > 0 {
		b.WriteString(" Superseded revisions remain indexed for historical lookup: ")
		names := make([]string, 0, len(deprecated))
		for _, hit := range deprecated {
			names = append(names, hit.DID)
		}
		b.WriteString(strings.Join(names, ", "))
	}
	if b.Len() == 0 {
		b.WriteString("No governed standard matched the query within the ingested citation index.")
	}
	return strings.TrimSpace(b.String())
}

// ledeForAnchors writes the deterministic first sentence about a quoted code.
func ledeForAnchors(e *Engine, anchors []CodeAnchor, primary []Match) string {
	var b strings.Builder
	for i, anchor := range anchors {
		current, err := e.resolver.Current(anchor.Body, anchor.Code)
		if err != nil {
			continue
		}
		if i > 0 {
			b.WriteString(" ")
		}
		if replacement := e.resolver.FormatReplaces(current); replacement != "" {
			b.WriteString(replacement)
		} else {
			b.WriteString(current.StandardBody + " " + current.Code + ":" + itoa(current.RevisionYear) + " is the current revision")
		}
		b.WriteString(".")
	}
	return b.String()
}

// buildSuggestions produces auditable follow-up actions, never content.
func buildSuggestions(e *Engine, anchors []CodeAnchor, primaryCount int) []string {
	var actions []string
	for _, anchor := range anchors {
		current, err := e.resolver.Current(anchor.Body, anchor.Code)
		if err != nil {
			continue
		}
		actions = append(actions, "Verify official "+current.StandardBody+" metadata at "+current.OfficialStoreURL)
	}
	if primaryCount > 0 {
		actions = append(actions, "Top primary match resolves ACTIVE; bind it to the work order package for enforcement")
	}
	if len(actions) == 0 {
		actions = append(actions, "Ingest additional citation cards before relying on this query")
	}
	return actions
}

// resolveAnchorBodies infers the owning body for body-less code anchors (e.g.
// "b30.5" -> ASME) against the resolver index, and drops anchors no series
// owns. The result is deterministic: first owning series wins.
func (e *Engine) resolveAnchorBodies(anchors []CodeAnchor) []CodeAnchor {
	out := anchors[:0:len(anchors)]
	for _, anchor := range anchors {
		if anchor.Body != "" {
			out = append(out, anchor)
			continue
		}
		body, canonical, ok := e.resolver.LookupByCode(anchor.Code)
		if !ok {
			continue
		}
		anchor.Body = body
		anchor.Code = canonical
		out = append(out, anchor)
	}
	return out
}

// extractAnchors finds standards code mentions in a query. It normalizes
// matches against the registered body names and preferred code spellings from
// the resolver index so "b30.5", "ASME B30.5", and "B30.5-2021" collapse.
func extractAnchors(query string) []CodeAnchor {
	seen := map[string]bool{}
	var anchors []CodeAnchor
	for _, raw := range anchorPattern.FindAllStringSubmatch(query, -1) {
		body := normalizeBody(raw[1])
		code := normalizeCode(raw[2])
		key := strings.ToLower(body + "|" + code)
		if seen[key] {
			continue
		}
		seen[key] = true
		anchors = append(anchors, CodeAnchor{Body: body, Code: code})
	}
	// Code-only mentions are captured too; their body is inferred later.
	for _, raw := range codeOnlyPattern.FindAllStringSubmatch(query, -1) {
		code := normalizeCode(raw[1])
		if code == "" || len(code) < 2 {
			continue
		}
		key := "|" + code
		if seen[key] {
			continue
		}
		seen[key] = true
		anchors = append(anchors, CodeAnchor{Code: code})
	}
	return anchors
}

// normalizeBody maps a mention to the canonical body spelling.
func normalizeBody(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "asme":
		return "ASME"
	case "iso":
		return "ISO"
	case "bsi", "bs":
		return "BSI"
	case "din":
		return "DIN"
	case "astm":
		return "ASTM"
	case "dnv", "dnvgl":
		return "DNV"
	case "iec":
		return "IEC"
	case "leea":
		return "LEEA"
	case "nfpa":
		return "NFPA"
	case "aws":
		return "AWS"
	case "jis":
		return "JIS"
	case "api":
		return "API"
	default:
		return ""
	}
}

// normalizeCode collapses spacing and punctuation variants of a code token.
func normalizeCode(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	var b strings.Builder
	lastSpace := false
	for _, r := range value {
		if unicode.IsSpace(r) {
			lastSpace = true
			continue
		}
		if lastSpace {
			b.WriteByte('-')
			lastSpace = false
		}
		b.WriteRune(r)
	}
	return strings.Trim(b.String(), ".-")
}

// anchorsFound reports whether any usable code anchors were extracted.
func anchorsFound(anchors []CodeAnchor) bool { return len(anchors) > 0 }

// clip bounds a string to w characters at a token boundary.
func clip(value string, w int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= w {
		return value
	}
	cut := value[:w]
	if i := strings.LastIndexByte(cut, ' '); i > w/2 {
		cut = cut[:i]
	}
	return cut + "…"
}

// clip bounds a string to w characters at a token boundary.
