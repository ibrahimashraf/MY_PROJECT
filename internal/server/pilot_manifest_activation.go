// Package server composes optional INTEGIN HTTP boundaries without registering routes.
package server

import (
	"errors"
	"os"
	"strings"

	"integin/internal/packagemanifestapi"
	"integin/internal/shared/featureflags"
)

const (
	// IsolatedPilotRuntime is the only runtime eligible for manifest retrieval review.
	IsolatedPilotRuntime = "pilot"
)

// IsolatedPilotManifestAddress is the address used for the isolated pilot manifest gate.
// Override with the INTEGIN_PILOT_MANIFEST_ADDR environment variable in production.
var IsolatedPilotManifestAddress = func() string {
	if v := os.Getenv("INTEGIN_PILOT_MANIFEST_ADDR"); v != "" {
		return v
	}
	return "127.0.0.1:18080"
}()

// PilotManifestActivationConfig is an explicit preflight-only gate. It does not
// register a route and cannot enable package enforcement.
type PilotManifestActivationConfig struct {
	RuntimeName               string
	ListenAddress             string
	ManifestRetrievalState    featureflags.State
	PackageEnforcementEnabled bool
	Handler                   *packagemanifestapi.Handler
}

// Validate fails closed unless a separately assembled pilot runtime is the only
// target, retrieval is explicitly enabled, and processor enforcement is off.
func (c PilotManifestActivationConfig) Validate() error {
	if c.RuntimeName != IsolatedPilotRuntime {
		return errors.New("manifest retrieval may only be reviewed for isolated pilot runtime")
	}
	if c.ListenAddress != IsolatedPilotManifestAddress {
		return errors.New("manifest retrieval may only target isolated pilot address")
	}
	if c.ManifestRetrievalState != featureflags.Enabled {
		return errors.New("manifest retrieval pilot feature flag is not enabled")
	}
	if c.PackageEnforcementEnabled {
		return errors.New("package enforcement must remain disabled during manifest binding pilot")
	}
	if c.Handler == nil {
		return errors.New("pilot manifest handler is incomplete")
	}
	return nil
}

// PilotManifestObservation is a privacy-minimized event shape for later pilot
// instrumentation. It intentionally carries no proof, signature, package body,
// user identity, or credential material.
type PilotManifestObservation struct {
	Outcome    PilotManifestOutcome
	ReasonCode string
	ManifestID string
}

// PilotManifestOutcome records only safe high-level lifecycle outcomes.
type PilotManifestOutcome string

const (
	PilotManifestRequestAccepted  PilotManifestOutcome = "request_accepted"
	PilotManifestProofRejected    PilotManifestOutcome = "proof_rejected"
	PilotManifestReplayRejected   PilotManifestOutcome = "replay_rejected"
	PilotManifestIssued           PilotManifestOutcome = "manifest_issued"
	PilotManifestBindingFailed    PilotManifestOutcome = "field_binding_failed"
	PilotManifestBindingSucceeded PilotManifestOutcome = "field_binding_succeeded"
)

// Validate checks safe observation invariants before any future sink receives it.
func (o PilotManifestObservation) Validate() error {
	if strings.TrimSpace(string(o.Outcome)) == "" {
		return errors.New("pilot manifest observation outcome is required")
	}
	if strings.TrimSpace(o.ReasonCode) == "" {
		return errors.New("pilot manifest observation reason code is required")
	}
	return nil
}
