// Package workpackage defines the stable domain contracts for approved field work.
package workpackage

import (
	"fmt"
	"strings"
	"time"
)

// CanonicalManifestReadProof returns the exact UTF-8 grammar signed by Field
// and verified by the server for a work-package manifest read. Scope is not a
// client claim; tenant and organization are derived after proof verification.
func CanonicalManifestReadProof(
	requestID, deviceID, authorityID string,
	authorityEpoch uint64,
	inspectionID string,
	issuedAt, expiresAt time.Time,
	keyID string,
) string {
	return strings.Join([]string{
		ManifestProofProtocolVersion,
		ManifestProofPurpose,
		requestID,
		deviceID,
		authorityID,
		fmt.Sprintf("%d", authorityEpoch),
		inspectionID,
		issuedAt.UTC().Format(time.RFC3339Nano),
		expiresAt.UTC().Format(time.RFC3339Nano),
		ManifestProofSignatureAlgorithm,
		keyID,
	}, "|")
}
