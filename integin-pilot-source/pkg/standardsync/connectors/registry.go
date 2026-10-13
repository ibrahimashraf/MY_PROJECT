// Package connectors maps the twelve supported standard-setting bodies to
// their official storefront roots. The connectors are registries of citation
// metadata sources; they never fetch, cache, or render raw standard text.
//
// Safe Harbor posture: every connector exposes an official storefront root +
// a code-pattern recognizer, and the Safeguard guard refuses anything that
// looks like a direct document download (PDF/ZIP/document-path).
package connectors

import "sort"

// Body identifies one standard-setting organization and the official
// storefront a citation card may deep-link to.
type Body interface {
	// Slug is the normalized registry key ("asme", "iso", "dnv", ...).
	Slug() string
	// Name is the human-readable organization name.
	Name() string
	// Country is the ISO 3166-1 alpha-2 seat of the body ("US", "GB", ...).
	Country() string
	// StoreBaseURL is the official storefront root for citation cards.
	StoreBaseURL() string
	// StoreHosts are the exact https hosts (incl. www variant) owned by the
	// body's storefront. Safeguard pins every link to these hosts.
	StoreHosts() []string
	// MatchesCode reports whether s is a plausible code of this body.
	MatchesCode(s string) bool
	// CanonicalCode normalizes a matched code for indexing.
	CanonicalCode(s string) string
}

var registry = map[string]Body{}

// Register adds a body adapter. Duplicate slugs replace the previous adapter.
func Register(b Body) {
	if b != nil && b.Slug() != "" {
		registry[b.Slug()] = b
	}
}

// Lookup returns the adapter for a body slug (case-insensitive).
func Lookup(slug string) (Body, bool) {
	b, ok := registry[normalizeSlug(slug)]
	return b, ok
}

// All returns the registered adapters sorted by slug.
func All() []Body {
	slugs := make([]string, 0, len(registry))
	for slug := range registry {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	out := make([]Body, 0, len(slugs))
	for _, slug := range slugs {
		out = append(out, registry[slug])
	}
	return out
}

// Slugs returns the sorted registry keys.
func Slugs() []string {
	out := make([]string, 0, len(registry))
	for slug := range registry {
		out = append(out, slug)
	}
	sort.Strings(out)
	return out
}

func normalizeSlug(value string) string {
	// Prefix stripping for convenience ("api/", "int:asme").
	var builder []byte
	for i := 0; i < len(value); i++ {
		b := value[i]
		if (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') {
			builder = append(builder, b)
		}
	}
	return string(builder)
}
