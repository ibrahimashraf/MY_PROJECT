// Package server composes optional INTEGIN HTTP boundaries without registering routes.
package server

import (
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"errors"
	"os"
	"strings"

	"integin/internal/domain/device_trust"
	domainsync "integin/internal/domain/sync"
	"integin/internal/manifestreceiptbridge"
	"integin/internal/packagemanifest"
	"integin/internal/packagemanifestapi"
	"integin/internal/shared/featureflags"
	"integin/internal/syncapi"
	"integin/internal/workpackagepg"
)

const (
	pilotManifestRetrievalEnvironment      = "INTEGIN_PILOT_MANIFEST_RETRIEVAL"
	pilotRuntimeEnvironment                = "INTEGIN_PILOT_RUNTIME"
	pilotManifestPrivateKeyEnvironment     = "INTEGIN_PILOT_MANIFEST_PRIVATE_KEY_BASE64URL"
	pilotManifestKeyIDEnvironment          = "INTEGIN_PILOT_MANIFEST_KEY_ID"
	pilotManifestRunIDEnvironment          = "INTEGIN_PILOT_MANIFEST_CANDIDATE_RUN_ID"
	pilotManifestReceiptsEnvironment       = "INTEGIN_PILOT_MANIFEST_RECEIPTS_DIR"
	pilotManifestReceiptVersionEnvironment = "INTEGIN_PILOT_MANIFEST_RECEIPT_CONTRACT_VERSION"
)

// PilotManifestHandlerFromEnvironment returns nil when retrieval is not
// explicitly enabled. When enabled, it fails closed on any incomplete or
// non-pilot configuration. Private key material is decoded only in memory and
// never returned, logged, or persisted.
func PilotManifestHandlerFromEnvironment(
	processor *domainsync.Processor,
	authorities []device_trust.AuthorityPackage,
	database *sql.DB,
	listenAddress string,
) (*packagemanifestapi.Handler, *syncapi.AuthorityRegistry, error) {
	if strings.TrimSpace(os.Getenv(pilotManifestRetrievalEnvironment)) == "" {
		return nil, nil, nil
	}
	if os.Getenv(pilotManifestRetrievalEnvironment) != "enabled" {
		return nil, nil, errors.New("pilot manifest retrieval environment must be enabled")
	}
	if os.Getenv(pilotRuntimeEnvironment) != "pilot" {
		return nil, nil, errors.New("pilot manifest retrieval requires isolated pilot runtime")
	}
	if processor == nil || authorities == nil {
		return nil, nil, errors.New("pilot manifest retrieval requires processor and authorities")
	}
	if database == nil {
		return nil, nil, errors.New("pilot manifest retrieval requires database")
	}
	encodedKey := strings.TrimSpace(os.Getenv(pilotManifestPrivateKeyEnvironment))
	keyID := strings.TrimSpace(os.Getenv(pilotManifestKeyIDEnvironment))
	if encodedKey == "" || keyID == "" {
		return nil, nil, errors.New("pilot manifest signing configuration is incomplete")
	}
	privateKey, err := base64.RawURLEncoding.DecodeString(encodedKey)
	if err != nil || len(privateKey) != ed25519.PrivateKeySize {
		return nil, nil, errors.New("pilot manifest signing key is invalid")
	}
	defer func() { clear(privateKey) }()
	repository, err := workpackagepg.NewRepository(database)
	if err != nil {
		return nil, nil, err
	}
	registryHandler := syncapi.NewHandler(processor)
	for _, authority := range authorities {
		registryHandler.RegisterAuthority(authority)
	}
	registry := registryHandler.Authorities
	handler := NewPilotManifestHandler(processor, registry, repository, ed25519.PrivateKey(privateKey), keyID)
	config := PilotManifestActivationConfig{
		RuntimeName:               os.Getenv(pilotRuntimeEnvironment),
		ListenAddress:             listenAddress,
		ManifestRetrievalState:    featureflags.Enabled,
		PackageEnforcementEnabled: false,
		Handler:                   handler,
	}
	if err := config.Validate(); err != nil {
		return nil, nil, err
	}
	if err := attachPilotManifestReceiptBridge(handler); err != nil {
		return nil, nil, err
	}
	return handler, registry, nil
}

// attachPilotManifestReceiptBridge adds only an optional source-owned observer
// to an already validated explicit pilot handler. All-empty context is inert;
// partial or unsafe context fails closed. No launcher, wrapper, fixture, Field,
// route, or default-runtime code calls this helper.
func attachPilotManifestReceiptBridge(handler *packagemanifestapi.Handler) error {
	if handler == nil {
		return errors.New("pilot manifest receipt bridge requires handler")
	}
	_, writer, err := manifestreceiptbridge.Load(
		os.Getenv(pilotManifestRunIDEnvironment),
		os.Getenv(pilotManifestReceiptsEnvironment),
		os.Getenv(pilotManifestReceiptVersionEnvironment),
		nil,
	)
	if err != nil {
		return err
	}
	if writer == nil {
		return nil
	}
	replayStore, ok := handler.ReplayStore.(*workpackagepg.ManifestProofReplayStore)
	if !ok || !replayStore.Ready() {
		return errors.New("pilot manifest receipt bridge requires durable replay store")
	}
	counter, ok := any(replayStore).(packagemanifest.ProofReplayCounter)
	if !ok || counter == nil {
		return errors.New("pilot manifest receipt bridge requires durable replay counter")
	}
	handler.Observer = manifestreceiptbridge.NewHTTPObserver(writer)
	handler.ReplayCounter = counter
	return nil
}
