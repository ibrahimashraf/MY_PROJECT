// Package packagemanifestapi provides an intentionally unmounted manifest-read boundary.
package packagemanifestapi

import "context"

// Observation is the closed, source-redacted event shape emitted by the
// manifest HTTP boundary. It deliberately excludes proofs, signatures,
// manifests, package content, keys, identities, and credentials.
type Observation struct {
	Outcome        string
	ReasonCode     string
	HTTPStatus     int
	ManifestID     string
	ReplayAccepted bool
}

// ObservationSink receives source-redacted manifest outcomes. Implementations
// must treat an unavailable or rejected sink as non-authoritative and must not
// expose data through fixture stdout or stderr.
type ObservationSink interface {
	Observe(context.Context, Observation)
}
