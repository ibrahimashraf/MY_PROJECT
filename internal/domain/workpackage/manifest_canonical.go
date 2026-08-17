// Package workpackage defines the stable domain contracts for approved field work.
package workpackage

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	// ManifestProtocolVersion identifies the signed work-package manifest grammar.
	ManifestProtocolVersion = "work-package-manifest/v1"
)

// ManifestCanonicalInput contains the signed manifest fields that define a
// device's approved package assignment. Package definition content is bound by
// PackageHash and is validated separately before Field caches it.
type ManifestCanonicalInput struct {
	ManifestVersion    string
	TenantID           string
	OrganizationID     string
	InspectionID       string
	DeviceID           string
	PackageID          string
	PackageVersion     int
	PackageHash        string
	AssignmentContext  AssignmentContext
	SchemaVersion      int
	AuthorityEpoch     uint64
	IssuedAt           time.Time
	ExpiresAt          time.Time
	SignatureAlgorithm string
	KeyID              string
}

// CanonicalManifest returns the exact bytes signed by the manifest issuer and
// reproduced by Field before Ed25519 verification.
func CanonicalManifest(input ManifestCanonicalInput) string {
	return strings.Join([]string{
		input.ManifestVersion,
		input.TenantID,
		input.OrganizationID,
		input.InspectionID,
		input.DeviceID,
		input.PackageID,
		fmt.Sprintf("%d", input.PackageVersion),
		input.PackageHash,
		CanonicalAssignmentContext(input.AssignmentContext),
		fmt.Sprintf("%d", input.SchemaVersion),
		fmt.Sprintf("%d", input.AuthorityEpoch),
		input.IssuedAt.UTC().Format(time.RFC3339Nano),
		input.ExpiresAt.UTC().Format(time.RFC3339Nano),
		input.SignatureAlgorithm,
		input.KeyID,
	}, "|")
}

// CanonicalAssignmentContext orders field-to-asset bindings so map iteration
// cannot change the manifest bytes that Field verifies.
func CanonicalAssignmentContext(context AssignmentContext) string {
	pairs := make([]string, 0, len(context.FieldAssetIDs))
	for fieldID, assetID := range context.FieldAssetIDs {
		pairs = append(pairs, fieldID+"="+assetID)
	}
	sort.Strings(pairs)
	return strings.Join([]string{
		context.RootAssetID,
		context.InspectionType,
		context.ProcedureVersion,
		context.ScheduledAt.UTC().Format(time.RFC3339Nano),
		strings.Join(pairs, ","),
	}, "|")
}
