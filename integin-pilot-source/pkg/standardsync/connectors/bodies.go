package connectors

import (
	"regexp"
	"strings"
)

var (
	// bodyPrefix strips the optional body-name token from a code value.
	bodyPrefix = regexp.MustCompile(`(?i)^(ASME|API|ISO|BS|DIN|ASTM|DNVGL|DNV|IEC|LEEA|NFPA|AWS|JIS)\s*-?\s*`)
	// whitespaceSplit keeps the last space-separated run of a code value.
	whitespaceSplit = regexp.MustCompile(`[\s]+`)
)

// staticBody is the concrete adapter shared by all twelve bodies. Each
// instance pins official storefront hosts and a code-pattern recognizer.
type staticBody struct {
	slug        string
	name        string
	country     string
	baseURL     string
	hosts       []string
	patterns    []*regexp.Regexp
	canonSuffix string // e.g. "M/" handling for AWS metric editions
}

func (b *staticBody) Slug() string         { return b.slug }
func (b *staticBody) Name() string         { return b.name }
func (b *staticBody) Country() string      { return b.country }
func (b *staticBody) StoreBaseURL() string { return b.baseURL }
func (b *staticBody) StoreHosts() []string { return b.hosts }
func (b *staticBody) MatchesCode(s string) bool {
	for _, pattern := range b.patterns {
		if pattern.MatchString(s) {
			return true
		}
	}
	return false
}
func (b *staticBody) CanonicalCode(s string) string {
	// Drop a leading body-name token if present, then rejoin the remainder
	// with single spaces: "DNV ST 0377" -> "ST 0377", "ASME B30.5" -> "B30.5".
	fields := whitespaceSplit.Split(s, -1)
	joined := make([]string, 0, len(fields))
	for i, field := range fields {
		if i == 0 && bodyPrefix.MatchString(field) {
			continue
		}
		joined = append(joined, field)
	}
	return b.slug + ":" + strings.Join(joined, " ")
}

func init() {
	registerBodies()
}

// registerBodies registers the twelve supported bodies in the registry. The
// host lists are exact https origins pinned by the Safeguard guard; links are
// official storefronts only, never document download paths.
func registerBodies() {
	Register(&staticBody{
		slug:    "asme",
		name:    "American Society of Mechanical Engineers",
		country: "US",
		baseURL: "https://www.asme.org/codes-standards",
		hosts:   []string{"www.asme.org"},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(?:ASME\s+)?B\s?(\d+(?:\.\d+)*(?:-[A-Z0-9]+)?)$`),
		},
	})
	Register(&staticBody{
		slug:    "api",
		name:    "American Petroleum Institute",
		country: "US",
		baseURL: "https://www.api.org/products-and-services/standards",
		hosts:   []string{"www.api.org"},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(?:API\s+)?(STD|SPEC|RP|MPMS|TR)?\s?(\d+[A-Za-z]?(?:-\d+)?)$`),
		},
	})
	Register(&staticBody{
		slug:    "iso",
		name:    "International Organization for Standardization",
		country: "CH",
		baseURL: "https://www.iso.org/standards.html",
		hosts:   []string{"www.iso.org"},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(?:ISO\s+)?(\d+(?:-\d+)?)(?::\d{4})?$`),
		},
	})
	Register(&staticBody{
		slug:    "bsi",
		name:    "British Standards Institution",
		country: "GB",
		baseURL: "https://shop.bsigroup.com",
		hosts:   []string{"shop.bsigroup.com"},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(?:BS(?:\s|$))?(EN|ISO)?\s?(\d+[A-Z0-9.\-]*)$`),
		},
	})
	Register(&staticBody{
		slug:    "din",
		name:    "Deutsches Institut für Normung",
		country: "DE",
		baseURL: "https://www.beuth.de",
		hosts:   []string{"www.beuth.de", "www.din.de"},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(?:DIN\s+)?(?:(?:EN|ISO)\s+)*(\d+(?:[.-][A-Z0-9]+)*)$`),
		},
	})
	Register(&staticBody{
		slug:    "astm",
		name:    "ASTM International",
		country: "US",
		baseURL: "https://www.astm.org/products-services/standards-and-publications.html",
		hosts:   []string{"www.astm.org"},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(?:ASTM\s+)?([A-Z]\d+(?:\.\d+)?(?:-[0-9A-Za-z]+)?)$`),
		},
	})
	Register(&staticBody{
		slug:    "dnv",
		name:    "DNV (Det Norske Veritas)",
		country: "NO",
		baseURL: "https://www.dnv.com/energy/standards/",
		hosts:   []string{"www.dnv.com", "standards.dnv.com"},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(?:DNV(?:GL)?[- ]?)?(?:ST|RU|C)[- ]?[A-Z0-9-]*$`),
			regexp.MustCompile(`(?i)^(?:DNV(?:GL)?[- ]?)?[A-Z]{2,}[A-Z0-9-]*[0-9][A-Z0-9-]*$`),
		},
	})
	Register(&staticBody{
		slug:    "iec",
		name:    "International Electrotechnical Commission",
		country: "CH",
		baseURL: "https://webstore.iec.ch",
		hosts:   []string{"webstore.iec.ch"},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(?:IEC\s+)?([A-Z]{0,2}\d+(?:-\d+)?)(?::20\d{2})?$`),
		},
	})
	Register(&staticBody{
		slug:    "leea",
		name:    "Lifting Equipment Engineers Association",
		country: "GB",
		baseURL: "https://www.leeaint.com",
		hosts:   []string{"www.leeaint.com"},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(?:LEEA\s+)?([A-Z]{2,4}-?\d+[A-Z0-9]*)$`),
		},
	})
	Register(&staticBody{
		slug:    "nfpa",
		name:    "National Fire Protection Association",
		country: "US",
		baseURL: "https://www.nfpa.org/codes-and-standards",
		hosts:   []string{"www.nfpa.org"},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(?:NFPA\s+)?([A-Za-z]?\d+[A-Za-z]?)$`),
		},
	})
	Register(&staticBody{
		slug:    "aws",
		name:    "American Welding Society",
		country: "US",
		baseURL: "https://www.aws.org/standards",
		hosts:   []string{"www.aws.org"},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(?:AWS\s+)?([A-Z]\d+(?:\.\d+)?[A-Z0-9]*)$`),
		},
	})
	Register(&staticBody{
		slug:    "jis",
		name:    "Japanese Industrial Standards",
		country: "JP",
		baseURL: "https://www.jsa.or.jp/en",
		hosts:   []string{"www.jsa.or.jp"},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(?:JIS\s+)?([A-Z]\s?\d+(?:-\d+)?)(?::\d{4})?$`),
		},
	})
}
