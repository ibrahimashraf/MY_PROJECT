// Package workpackage defines the stable domain contracts for approved field work.
package workpackage

import (
	"errors"
	"time"
)

const (
	// ManifestProofProtocolVersion identifies the signed manifest-read proof grammar.
	ManifestProofProtocolVersion = "device-proof/v1"
	// ManifestProofPurpose domain-separates work-package manifest retrieval proofs.
	ManifestProofPurpose = "work_package_manifest.read"
	// ManifestProofSignatureAlgorithm is the only currently supported proof signature algorithm.
	ManifestProofSignatureAlgorithm = "Ed25519"
)

var (
	// ErrManifestProofReplayAlreadyConsumed rejects an active duplicate proof request.
	ErrManifestProofReplayAlreadyConsumed = errors.New("proof replay already consumed")
	// ErrManifestProofReplayExpired rejects a proof after its short-lived expiry.
	ErrManifestProofReplayExpired = errors.New("proof replay is expired")
)

// Assignment is the storage-neutral, server-authoritative binding between a
// Field device, inspection, approved package, and authority epoch.
type Assignment struct {
	TenantID       string
	OrganizationID string
	InspectionID   string
	DeviceID       string
	PackageID      string
	PackageVersion int
	AuthorityEpoch int64
	ExpiresAt      time.Time
	AssignedAt     time.Time
}
