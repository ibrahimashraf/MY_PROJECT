package connectors

import (
	"errors"
	"strings"
	"testing"
)

func TestRegistryHasTwelveBodies(t *testing.T) {
	bodies := All()
	if len(bodies) != 12 {
		t.Fatalf("registered bodies = %d, want 12", len(bodies))
	}
	seen := map[string]bool{}
	for _, body := range bodies {
		if seen[body.Slug()] {
			t.Fatalf("duplicate slug %q", body.Slug())
		}
		seen[body.Slug()] = true
		if body.Name() == "" || body.Country() == "" || body.StoreBaseURL() == "" || len(body.StoreHosts()) == 0 {
			t.Fatalf("incomplete adapter for %q", body.Name())
		}
	}
	// The twelve expected slugs must all be reachable.
	for _, slug := range []string{"api", "asme", "bsi", "iso", "din", "astm", "dnv", "iec", "leea", "nfpa", "aws", "jis"} {
		if _, ok := Lookup(slug); !ok {
			t.Fatalf("missing body %q", slug)
		}
	}
}

func TestCodeRecognitionSamples(t *testing.T) {
	samples := map[string][]string{
		"asme": {"ASME B30.5", "B30.5", "B30.31"},
		"iso":  {"ISO 4309", "4309", "ISO 4309:2010", "9001:2015"},
		"bsi":  {"BS EN 12390", "BS ISO 4309"},
		"din":  {"DIN EN ISO 6892", "DIN 18800"},
		"astm": {"ASTM D1.1", "E119"},
		"dnv":  {"DNV-ST-0377", "DNV ST 0377"},
		"iec":  {"IEC 60529:2013", "IEC 60068-2"},
		"leea": {"LEEA COP-14", "COP-14"},
		"nfpa": {"NFPA 70E", "NFPA 70"},
		"aws":  {"AWS D1.1", "D1.1M"},
		"jis":  {"JIS B 7516", "JIS B7516:2015"},
		"api":  {"API 610", "API Std 800"},
	}
	for slug, codes := range samples {
		body, ok := Lookup(slug)
		if !ok {
			t.Fatalf("missing %q", slug)
		}
		for _, code := range codes {
			if !body.MatchesCode(code) {
				t.Errorf("%s: MatchesCode(%q) = false", slug, code)
			}
		}
	}
	for slug, codes := range map[string][]string{
		"asme": {"4309", "X99"},
		"iso":  {"B30.5", "ABC"},
		"dnv":  {"B30.5", "4309"},
		"nfpa": {"D1.11"},
	} {
		body, _ := Lookup(slug)
		for _, code := range codes {
			if body.MatchesCode(code) {
				t.Errorf("%s: MatchesCode(%q) unexpectedly true", slug, code)
			}
		}
	}
}

func TestCanonicalCodeNormalizes(t *testing.T) {
	cases := []struct{ slug, input, want string }{
		{"asme", "ASME B30.5", "asme:B30.5"},
		{"iso", "ISO 4309:2010", "iso:4309:2010"},
		{"dnv", "DNV ST 0377", "dnv:ST 0377"},
		{"nfpa", "NFPA 70E", "nfpa:70E"},
	}
	for _, c := range cases {
		body, _ := Lookup(c.slug)
		if got := body.CanonicalCode(c.input); got != c.want {
			t.Errorf("CanonicalCode(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestGuardPinsHTTPSHostAndPath(t *testing.T) {
	iso, _ := Lookup("iso")
	hosts := iso.StoreHosts()

	if err := ValidateOfficialStoreURL("https://www.iso.org/standards.html", hosts); err != nil {
		t.Fatalf("valid storefront URL rejected: %v", err)
	}
	if err := ValidateOfficialStoreURL("http://www.iso.org/standards", hosts); !errors.Is(err, ErrNotHTTPS) {
		t.Fatalf("http URL = %v, want ErrNotHTTPS", err)
	}
	if err := ValidateOfficialStoreURL("https://evil.example.org/standards", hosts); !errors.Is(err, ErrHostNotPinned) {
		t.Fatalf("foreign host = %v, want ErrHostNotPinned", err)
	}
	if err := ValidateOfficialStoreURL("https://www.iso.org/documents/4309.pdf", hosts); !errors.Is(err, ErrLooksLikeDownload) {
		t.Fatalf("pdf path = %v, want ErrLooksLikeDownload", err)
	}
	if err := ValidateOfficialStoreURL("https://www.iso.org/files/4309", hosts); !errors.Is(err, ErrLooksLikeDownload) {
		t.Fatalf("files path = %v, want ErrLooksLikeDownload", err)
	}
}

func TestSafeStoreURLRefusesUnrecognizedCode(t *testing.T) {
	asme, _ := Lookup("asme")
	url, err := SafeStoreURL(asme, "4309")
	if err == nil {
		t.Fatalf("SafeStoreURL accepted wrong-body code, url=%q", url)
	}
	url, err = SafeStoreURL(asme, "B30.5")
	if err != nil || !strings.HasPrefix(url, "https://") {
		t.Fatalf("SafeStoreURL(B30.5) = %q, %v", url, err)
	}
}
