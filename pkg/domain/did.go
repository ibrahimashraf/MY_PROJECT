package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	DIDMethodIntegin    = "did:integin"
	DIDTypeAsset        = "asset"
	DIDTypeStandard     = "standard"
	DIDTypeTenant       = "tenant"
	DIDTypeDevice       = "device"
	DIDTypeJurisdiction = "jurisdiction"
)

var (
	ErrInvalidDIDPrefix = errors.New("did must start with did:integin:")
	ErrMalformedDID     = errors.New("did must conform to did:integin:<type>:<identifier>")
	ErrEmptyIdentifier  = errors.New("identifier component cannot be empty")
)

// DID represents a structured W3C-compliant Decentralized Identifier for the platform.
type DID struct {
	Method     string `json:"method"`     // "did:integin"
	Type       string `json:"type"`       // "asset", "standard", "tenant", "device"
	Identifier string `json:"identifier"` // Unique content hash or slug
}

// String returns the canonical W3C DID URI representation.
func (d DID) String() string {
	return fmt.Sprintf("%s:%s:%s", DIDMethodIntegin, d.Type, d.Identifier)
}

// ParseDID validates and parses a raw DID string.
func ParseDID(raw string) (DID, error) {
	if !strings.HasPrefix(raw, DIDMethodIntegin+":") {
		return DID{}, ErrInvalidDIDPrefix
	}
	parts := strings.Split(raw, ":")
	if len(parts) != 4 {
		return DID{}, ErrMalformedDID
	}
	entityType := parts[2]
	identifier := parts[3]
	if identifier == "" {
		return DID{}, ErrEmptyIdentifier
	}

	return DID{
		Method:     DIDMethodIntegin,
		Type:       entityType,
		Identifier: identifier,
	}, nil
}

// GenerateAssetDID creates a deterministic W3C DID for an industrial asset
// derived from its immutable physical attributes (manufacturer, model, serial, initial timestamp).
func GenerateAssetDID(manufacturer, model, serial string, birthEpoch int64) DID {
	seed := fmt.Sprintf("%s|%s|%s|%d",
		strings.TrimSpace(strings.ToUpper(manufacturer)),
		strings.TrimSpace(strings.ToUpper(model)),
		strings.TrimSpace(strings.ToUpper(serial)),
		birthEpoch,
	)
	hash := sha256.Sum256([]byte(seed))
	fingerprint := hex.EncodeToString(hash[:16]) // 128-bit collision-resistant fingerprint

	return DID{
		Method:     DIDMethodIntegin,
		Type:       DIDTypeAsset,
		Identifier: fingerprint,
	}
}

// GenerateStandardDID creates a canonical DID for any international standard.
// e.g. "did:integin:standard:ASME_B30_5_2024"
func GenerateStandardDID(body, code, year string) DID {
	cleanBody := strings.TrimSpace(strings.ToUpper(body))
	cleanCode := strings.ReplaceAll(strings.TrimSpace(strings.ToUpper(code)), ".", "_")
	cleanYear := strings.TrimSpace(year)

	identifier := fmt.Sprintf("%s_%s_%s", cleanBody, cleanCode, cleanYear)
	return DID{
		Method:     DIDMethodIntegin,
		Type:       DIDTypeStandard,
		Identifier: identifier,
	}
}

// GenerateTenantDID creates a canonical DID for a global tenant organization.
func GenerateTenantDID(tenantUUID string) DID {
	return DID{
		Method:     DIDMethodIntegin,
		Type:       DIDTypeTenant,
		Identifier: strings.ToLower(strings.TrimSpace(tenantUUID)),
	}
}

// GenerateJurisdictionDID creates a canonical DID for a sovereign jurisdiction using ISO 3166-1 alpha-2.
// e.g. "did:integin:jurisdiction:SA"
func GenerateJurisdictionDID(countryISO2 string) DID {
	return DID{
		Method:     DIDMethodIntegin,
		Type:       DIDTypeJurisdiction,
		Identifier: strings.ToUpper(strings.TrimSpace(countryISO2)),
	}
}

// TimeSource provides testable timestamp generation.
type TimeSource interface {
	Now() time.Time
}

type RealTimeSource struct{}

func (RealTimeSource) Now() time.Time { return time.Now().UTC() }
