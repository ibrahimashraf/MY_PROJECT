package connectors

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Guard errors reported by ValidateOfficialStoreURL.
var (
	ErrNotHTTPS          = errors.New("official store URL must be https")
	ErrHostNotPinned     = errors.New("official store URL host is not pinned by the referencing body")
	ErrLooksLikeDownload = errors.New("official store URL points at a document download path")
	ErrUnknownBody       = errors.New("unknown standards body")
)

// ValidateOfficialStoreURL is the Safe Harbor filter applied to every URL in a
// citation card before it may be stored or emitted. It enforces:
//
//   - scheme is https only;
//   - host is one of the hosts pinned by the referencing body adapter;
//   - the path is a metadata/catalog page, never a raw document payload
//     (.pdf/.zip extractable downloads, or /documents/ /documents/1/ files
//     style paths).
//
// A URL failing the guard is rejected rather than sanitized into something the
// body does not own, because guesses can point at the wrong host.
func ValidateOfficialStoreURL(raw string, allowedHosts []string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid store URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return ErrNotHTTPS
	}
	host := strings.ToLower(parsed.Hostname())
	pinned := false
	for _, allowed := range allowedHosts {
		if strings.ToLower(allowed) == host {
			pinned = true
			break
		}
		// Allow exact-root descendants only with an explicit port match.
	}
	if !pinned {
		return fmt.Errorf("%w: %s", ErrHostNotPinned, host)
	}
	lower := strings.ToLower(parsed.Path)
	for _, bad := range []string{".pdf", ".zip", ".doc", ".docx", "/documents/", "/files/", "/downloads/", "/webstore/download/"} {
		if strings.Contains(lower, bad) {
			return fmt.Errorf("%w: %s", ErrLooksLikeDownload, parsed.Path)
		}
	}
	return nil
}

// SafeStoreURL returns the official storefront root after confirming the body
// recognizes the code. It never fabricates per-code deep links: authoritative
// per-revision URLs come from card ingestion, not from reconstruction.
func SafeStoreURL(body Body, code string) (string, error) {
	if body == nil {
		return "", ErrUnknownBody
	}
	if !body.MatchesCode(code) {
		return "", fmt.Errorf("code %q is not recognized by %s", code, body.Slug())
	}
	maybeLink := &url.URL{Scheme: "https", Host: body.StoreHosts()[0], Path: ""}
	// Guard even the reconstructed root: cheaper to fail closed than to link wrong.
	if err := ValidateOfficialStoreURL(maybeLink.String(), body.StoreHosts()); err != nil {
		return "", err
	}
	return body.StoreBaseURL(), nil
}

// RejectPayload is an explicit name for the download-path rule so callers can
// self-document in ingest filters.
func RejectPayload(raw string, allowedHosts []string) error {
	return ValidateOfficialStoreURL(raw, allowedHosts)
}
