// Package workpackage defines the stable domain contracts for approved field work.
package workpackage

import (
	"strconv"
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
	var b strings.Builder
	b.Grow(256)
	b.WriteString(ManifestProofProtocolVersion)
	b.WriteByte('|')
	b.WriteString(ManifestProofPurpose)
	b.WriteByte('|')
	b.WriteString(requestID)
	b.WriteByte('|')
	b.WriteString(deviceID)
	b.WriteByte('|')
	b.WriteString(authorityID)
	b.WriteByte('|')
	b.WriteString(strconv.FormatUint(authorityEpoch, 10))
	b.WriteByte('|')
	b.WriteString(inspectionID)
	b.WriteByte('|')
	b.WriteString(issuedAt.UTC().Format(time.RFC3339Nano))
	b.WriteByte('|')
	b.WriteString(expiresAt.UTC().Format(time.RFC3339Nano))
	b.WriteByte('|')
	b.WriteString(ManifestProofSignatureAlgorithm)
	b.WriteByte('|')
	b.WriteString(keyID)
	return b.String()
}
