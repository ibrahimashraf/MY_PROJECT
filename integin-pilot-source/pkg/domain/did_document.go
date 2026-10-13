package domain

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

// W3C DID Core 1.0 context and verification method shape used by every
// did:integin document.
const (
	DIDContextV1                      = "https://www.w3.org/ns/did/v1"
	VerificationMethodTypeEd25519     = "Ed25519VerificationKey2018"
	DefaultVerificationMethodFragment = "#key-1"
	DefaultServiceFragment            = "#service-1"
	universalResolverBase             = "https://uniresolver.integin.io/1.0/identifiers/"
)

// ErrUnsupportedDIDType reports a did:integin type the resolver does not serve.
var ErrUnsupportedDIDType = errors.New("unsupported did:integin DID type")

// DIDDocument is the W3C DID Core 1.0 representation of a did:integin subject.
type DIDDocument struct {
	Context             string               `json:"@context"`
	ID                  string               `json:"id"`
	Controller          []string             `json:"controller,omitempty"`
	VerificationMethods []VerificationMethod `json:"verificationMethod"`
	Authentication      []string             `json:"authentication,omitempty"`
	Service             []Service            `json:"service,omitempty"`
}

// VerificationMethod is a W3C DID Core verification relationship.
type VerificationMethod struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	Controller      string `json:"controller"`
	PublicKeyBase64 string `json:"publicKeyBase64"`
}

// Service is a W3C DID Core service entry pointing at a deterministic
// platform endpoint for the subject's type.
type Service struct {
	ID              string            `json:"id"`
	Type            string            `json:"type"`
	ServiceEndpoint []ServiceEndpoint `json:"serviceEndpoint"`
}

// ServiceEndpoint is one concrete endpoint URL of a service.
type ServiceEndpoint struct {
	Type string `json:"type,omitempty"`
	URL  string `json:"url,omitempty"`
}

// UniversalDIDResolver resolves any valid did:integin:<type>:<id> to a W3C
// DID Core 1.0 document with deterministic verificationMethod keys derived
// from the DID itself.
type UniversalDIDResolver struct{}

// NewUniversalDIDResolver returns a resolver for the did:integin method.
func NewUniversalDIDResolver() *UniversalDIDResolver { return &UniversalDIDResolver{} }

// supportedDIDTypes is the closed set of resolvable did:integin types.
var supportedDIDTypes = map[string]struct{}{
	DIDTypeAsset:        {},
	DIDTypeStandard:     {},
	DIDTypeTenant:       {},
	DIDTypeDevice:       {},
	DIDTypeJurisdiction: {},
}

// Resolve parses and resolves a raw DID string.
func (r *UniversalDIDResolver) Resolve(raw string) (DIDDocument, error) {
	did, err := ParseDID(raw)
	if err != nil {
		return DIDDocument{}, err
	}
	return r.ResolveDID(did)
}

// ResolveDID resolves a parsed DID to its deterministic W3C DID document.
func (r *UniversalDIDResolver) ResolveDID(did DID) (DIDDocument, error) {
	if did.Method != DIDMethodIntegin {
		return DIDDocument{}, ErrInvalidDIDPrefix
	}
	if _, ok := supportedDIDTypes[did.Type]; !ok {
		return DIDDocument{}, fmt.Errorf("%w: %q", ErrUnsupportedDIDType, did.Type)
	}

	uri := did.String()
	keyID := uri + DefaultVerificationMethodFragment
	verificationMethod := VerificationMethod{
		ID:              keyID,
		Type:            VerificationMethodTypeEd25519,
		Controller:      uri,
		PublicKeyBase64: deterministicPublicKey(uri),
	}

	return DIDDocument{
		Context:             DIDContextV1,
		ID:                  uri,
		Controller:          []string{uri},
		VerificationMethods: []VerificationMethod{verificationMethod},
		Authentication:      []string{keyID},
		Service:             serviceFor(did),
	}, nil
}

// deterministicPublicKey derives a fixed Ed25519 public key for a DID so the
// same DID always resolves to the same verificationMethod key, without
// storing any secret.
func deterministicPublicKey(didURI string) string {
	seed := sha256.Sum256([]byte(didURI))
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub := priv.Public().(ed25519.PublicKey)
	return base64.StdEncoding.EncodeToString(pub)
}

// serviceFor returns the type-specific deterministic service entry.
func serviceFor(did DID) []Service {
	uri := did.String()
	serviceType, url := serviceShape(did.Type, did.Identifier)
	return []Service{{
		ID:   uri + DefaultServiceFragment,
		Type: serviceType,
		ServiceEndpoint: []ServiceEndpoint{{
			Type: "InteginServiceEndpoint",
			URL:  url,
		}},
	}}
}

// serviceShape maps each DID type to a canonical platform service and its
// deterministic base URL.
func serviceShape(typ, identifier string) (serviceType, url string) {
	switch typ {
	case DIDTypeAsset:
		return "InteginAssetRegistry", "https://passport.integin.io/assets/" + identifier
	case DIDTypeStandard:
		return "InteginStandardCatalogue", "https://catalogue.integin.io/standards/" + identifier
	case DIDTypeTenant:
		return "InteginTenantAccount", "https://registry.integin.io/tenants/" + identifier
	case DIDTypeDevice:
		return "InteginDeviceRegistry", "https://registry.integin.io/devices/" + identifier
	case DIDTypeJurisdiction:
		return "InteginJurisdictionProfile", "https://registry.integin.io/jurisdictions/" + identifier
	default:
		return "InteginGeneric", universalResolverBase + identifier
	}
}

// ResolveURI is a convenience helper for callers that only need the document
// ID of a subject, resolving through the same validator the full resolver uses.
func (r *UniversalDIDResolver) ResolveURI(raw string) (string, error) {
	doc, err := r.Resolve(raw)
	if err != nil {
		return "", err
	}
	return doc.ID, nil
}
