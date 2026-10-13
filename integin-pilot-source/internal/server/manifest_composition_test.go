package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	domainsync "integin/internal/domain/sync"
	"integin/internal/syncapi"
	"integin/internal/workpackagepg"
)

func TestNewPilotManifestHandlerRejectsIncompleteDependencies(t *testing.T) {
	if handler := NewPilotManifestHandler(nil, nil, nil, nil, "key"); handler != nil {
		t.Fatal("expected nil handler for incomplete dependencies")
	}
}

func TestNewPilotManifestHandlerBuildsWithoutRouteRegistration(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	handler := NewPilotManifestHandler(
		&domainsync.Processor{},
		&syncapi.AuthorityRegistry{},
		&workpackagepg.Repository{},
		privateKey,
		"pilot-manifest-key",
	)
	if handler == nil || handler.Verifier == nil || handler.Authorities == nil || handler.ReplayStore == nil || handler.Issuer == nil {
		t.Fatal("expected complete but unmounted pilot manifest handler")
	}
}
