// Package server composes optional INTEGIN HTTP boundaries without registering routes.
package server

import (
	"crypto/ed25519"
	"time"

	domainsync "integin/internal/domain/sync"
	"integin/internal/packagemanifest"
	"integin/internal/packagemanifestapi"
	"integin/internal/syncapi"
	"integin/internal/workpackagepg"
)

const pilotManifestMaxLifetime = 30 * time.Minute

// NewPilotManifestHandler assembles the package-manifest boundary for an
// explicitly controlled pilot composition. It is deliberately uncalled by all
// existing server startup paths. A nil return means configuration was invalid
// or incomplete, so callers cannot mount a partially configured handler.
func NewPilotManifestHandler(
	processor *domainsync.Processor,
	authorities *syncapi.AuthorityRegistry,
	repository *workpackagepg.Repository,
	signingKey ed25519.PrivateKey,
	keyID string,
) *packagemanifestapi.Handler {
	if processor == nil || authorities == nil || repository == nil {
		return nil
	}
	issuer, err := packagemanifest.NewManifestIssuer(
		repository,
		repository,
		signingKey,
		keyID,
		pilotManifestMaxLifetime,
	)
	if err != nil {
		return nil
	}
	replayStore := workpackagepg.NewManifestProofReplayStore(repository)
	if replayStore == nil {
		return nil
	}
	return &packagemanifestapi.Handler{
		Verifier:    processor,
		Authorities: authorities,
		ReplayStore: replayStore,
		Issuer:      issuer,
	}
}
