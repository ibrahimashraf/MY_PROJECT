package certificatepublichttp

import (
	"net/http"

	"integin/internal/platform/featureflag"
	"integin/internal/shared/featureflags"
)

// PortalGuard wraps Handler so client portal is only served when
// FlagClientPortal is ENABLED for the organization (default DISABLED).
// No new GRANT; still uses VerifyPublic(public_token_digest) which is tenant-isolated
// via certificatepg/public.go and FORCE RLS. ReadOnly, rate-limited, no-store.
type PortalGuard struct {
	Handler *Handler
	Flags   *featureflag.Service
}

func (g *PortalGuard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if g.Handler == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	// Default deny; org must be ENABLED via featureflag override.
	// When Flags is nil (tests), allow (existing behaviour).
	if g.Flags != nil {
		// No org in public request; treat as globally gated: default must be ENABLED.
		state := g.Flags.Evaluate(featureflags.FlagClientPortal, featureflag.Request{}, g.Handler.Now().UTC())
		if state != featureflags.Enabled {
			http.Error(w, "client portal disabled", http.StatusForbidden)
			return
		}
	}
	g.Handler.ServeHTTP(w, r)
}
