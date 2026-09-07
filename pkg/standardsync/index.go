package standardsync

import (
	"hash/fnv"
	"math"
	"sort"
	"strings"
	"unicode"
)

// vectorDimensions is the fixed embedding width. Kept small enough that a few
// thousand cards fit the offline <30ms budget while still separating series.
const vectorDimensions = 128

// vectorizer deterministically embeds token streams into an L2-normalized
// fixed-dimension bag-of-hashed-features vector. It is architecture-stable (no
// map iteration order, no float accumulation order sensitivity beyond FNV), so
// the same card universe yields the same rankings on every device.
type vectorizer struct{}

// Embed converts pre-tokenized terms into a normalized vector.
func (vectorizer) Embed(terms []string) []float64 {
	vector := make([]float64, vectorDimensions)
	seen := make(map[uint64]bool)
	for i, term := range terms {
		term = strings.TrimSpace(strings.ToLower(term))
		if term == "" {
			continue
		}
		fn := fnv.New64a()
		_, _ = fn.Write([]byte(term))
		vector[fn.Sum64()%vectorDimensions] += 1
		if i > 0 {
			fn.Reset()
			_, _ = fn.Write([]byte(terms[i-1]))
			_, _ = fn.Write([]byte{0})
			_, _ = fn.Write([]byte(term))
			pair := fn.Sum64() % vectorDimensions
			if !seen[pair] {
				seen[pair] = true
				vector[pair] += 0.5
			}
		}
	}
	normalize(vector)
	return vector
}

// EmbedText tokenizes prose then delegates to Embed. Scope prose is embedded
// only for abstract matching; it is never surfaced verbatim in answers beyond
// bounded excerpts.
func (v vectorizer) EmbedText(text string) []float64 {
	return v.Embed(tokenize(text))
}

// cosine similarity between two fixed-dimension vectors. Both must be norm 1.
func cosine(a, b []float64) float64 {
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	var dot float64
	for i := 0; i < limit; i++ {
		dot += a[i] * b[i]
	}
	return dot
}

// normalize L2-normalizes a vector in place (maps the zero vector to itself).
func normalize(vector []float64) {
	var sum float64
	for _, v := range vector {
		sum += v * v
	}
	if sum == 0 {
		return
	}
	norm := math.Sqrt(sum)
	for i := range vector {
		vector[i] /= norm
	}
}

// tokenize is a conservative word tokenizer: it keeps alphanumeric runs and
// typical standard-code separators so the vectorizer can match "B30.5" as one
// feature.
func tokenize(text string) []string {
	var tokens []string
	var current strings.Builder
	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	for _, r := range strings.ToLower(text) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-':
			current.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	return tokens
}

// vectorDoc is one vectorized card internal to the abstract index.
type vectorDoc struct {
	card     StandardMetadataCard
	features []float64
	abstract []float64
}

// AbstractIndex is an in-memory vector + lexical index over citation cards.
// It provides deterministic ranking independent of insertion order: candidate
// scores are stable and ties are broken by DID.
type AbstractIndex struct {
	docs    []vectorDoc
	lexicon map[string][]int // term -> doc indices (for BM25-style weighting)
	terms   map[string]int   // term -> count in query weighting
}

// NewAbstractIndex builds the index from cards. The feature stream is the
// card's identity (body, code, year, tags); the abstract stream is its
// ScopeAbstract plus title. Titles participate in both streams.
func NewAbstractIndex(cards []StandardMetadataCard) *AbstractIndex {
	idx := &AbstractIndex{
		lexicon: make(map[string][]int),
		terms:   make(map[string]int),
	}
	v := vectorizer{}
	for n, card := range cards {
		featureTerms := append(card.Articles()[:len(card.Articles()):len(card.Articles())], tokenize(card.Title)...)
		abstractTerms := append(tokenize(card.ScopeAbstract), tokenize(card.Title)...)
		doc := vectorDoc{
			card:     card,
			features: v.Embed(featureTerms),
			abstract: v.Embed(abstractTerms),
		}
		idx.docs = append(idx.docs, doc)
		for _, term := range uniqueTerms(featureTerms) {
			idx.lexicon[term] = append(idx.lexicon[term], n)
		}
		for _, term := range abstractTerms {
			idx.terms[term]++
		}
	}
	return idx
}

// Len returns the number of indexed cards.
func (idx *AbstractIndex) Len() int { return len(idx.docs) }

// Cards returns a copy of the indexed cards without exporter mutation.
func (idx *AbstractIndex) Cards() []StandardMetadataCard {
	cards := make([]StandardMetadataCard, len(idx.docs))
	for i, doc := range idx.docs {
		cards[i] = doc.card
	}
	return cards
}

// ScoredCard pairs a card with its hybrid relevance.
type ScoredCard struct {
	Card  StandardMetadataCard
	Score float64
}

// Query performs vector + lexical scoring for a natural-language query.
// vectorT  and lexicalT weights combine the two streams; callers can bias
// toward exact codes (Search) or abstract paraphrases (semantic mode).
func (idx *AbstractIndex) Query(query string, vectorT, lexicalT float64) []ScoredCard {
	queryTokens := uniqueTerms(tokenize(query))
	if len(queryTokens) == 0 {
		return nil
	}
	embedded := vectorizer{}.Embed(queryTokens)
	total := len(idx.docs)
	results := make([]ScoredCard, 0, total)
	for i := range idx.docs {
		vectorScore := cosine(idx.docs[i].features, embedded)*0.7 + cosine(idx.docs[i].abstract, embedded)*0.3
		lexicalScore := idx.lexicalScore(i, queryTokens)
		score := vectorT*vectorScore + lexicalT*lexicalScore
		if score > 0 {
			results = append(results, ScoredCard{Card: idx.docs[i].card, Score: score})
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Card.StandardDID < results[j].Card.StandardDID
	})
	return results
}

// lexicalScore is a compact BM25-style overlap measure over the feature
// lexicon. Frequency is capped at 3 to avoid one repetitive abstract dominating.
func (idx *AbstractIndex) lexicalScore(doc int, queryTerms []string) float64 {
	var score float64
	for _, term := range queryTerms {
		postings, ok := idx.lexicon[term]
		if !ok {
			continue
		}
		matchCount := 0
		for _, hit := range postings {
			if hit == doc {
				matchCount++
				if matchCount >= 3 {
					break
				}
			}
		}
		if matchCount > 0 {
			df := len(postings)
			idf := math.Log(1 + (float64(len(idx.docs))+0.5)/(float64(df)+0.5))
			score += float64(matchCount) * idf
		}
	}
	return score
}

// uniqueTerms returns query terms in first-seen order, deduplicated.
func uniqueTerms(terms []string) []string {
	seen := make(map[string]bool, len(terms))
	out := make([]string, 0, len(terms))
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		if !seen[term] {
			seen[term] = true
			out = append(out, term)
		}
	}
	return out
}
