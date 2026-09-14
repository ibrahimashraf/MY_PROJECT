package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestUniversalDIDResolverAllFiveTypes(t *testing.T) {
	resolver := NewUniversalDIDResolver()
	subjects := []string{
		GenerateAssetDID("Liebherr", "LTM-11200", "SN-991", 1700000000).String(),
		GenerateStandardDID("ASME", "B30.5", "2024").String(),
		GenerateTenantDID("tenant-saudi-contractor").String(),
		"did:integin:device:tablet-7f3a9912",
		GenerateJurisdictionDID("SA").String(),
	}
	expectedServices := []string{
		"InteginAssetRegistry",
		"InteginStandardCatalogue",
		"InteginTenantAccount",
		"InteginDeviceRegistry",
		"InteginJurisdictionProfile",
	}
	for i, raw := range subjects {
		doc, err := resolver.Resolve(raw)
		if err != nil {
			t.Fatalf("Resolve(%q) error: %v", raw, err)
		}
		if doc.ID != raw {
			t.Errorf("doc ID = %q, want %q", doc.ID, raw)
		}
		if doc.Context != DIDContextV1 {
			t.Errorf("doc @context = %q", doc.Context)
		}
		if len(doc.Controller) != 1 || doc.Controller[0] != raw {
			t.Errorf("doc controller = %v, want [%s]", doc.Controller, raw)
		}
		if len(doc.VerificationMethods) != 1 {
			t.Fatalf("expected 1 verification method, got %d", len(doc.VerificationMethods))
		}
		vm := doc.VerificationMethods[0]
		wantKeyID := raw + DefaultVerificationMethodFragment
		if vm.ID != wantKeyID {
			t.Errorf("vm ID = %q, want %q", vm.ID, wantKeyID)
		}
		if vm.Type != VerificationMethodTypeEd25519 {
			t.Errorf("vm type = %q", vm.Type)
		}
		if vm.Controller != raw {
			t.Errorf("vm controller = %q", vm.Controller)
		}
		if len(vm.PublicKeyBase64) == 0 {
			t.Errorf("vm publicKeyBase64 is empty")
		}
		if len(doc.Authentication) != 1 || doc.Authentication[0] != wantKeyID {
			t.Errorf("authentication = %v", doc.Authentication)
		}
		if len(doc.Service) != 1 {
			t.Fatalf("expected 1 service, got %d", len(doc.Service))
		}
		if doc.Service[0].Type != expectedServices[i] {
			t.Errorf("service type = %q, want %q", doc.Service[0].Type, expectedServices[i])
		}
		if doc.Service[0].ID != raw+DefaultServiceFragment {
			t.Errorf("service ID = %q", doc.Service[0].ID)
		}
		if len(doc.Service[0].ServiceEndpoint) != 1 || doc.Service[0].ServiceEndpoint[0].URL == "" {
			t.Errorf("service endpoint missing")
		}
	}
}

func TestUniversalDIDResolverDeterministicKeys(t *testing.T) {
	resolver := NewUniversalDIDResolver()
	raw := "did:integin:asset:ab12cd34ef56"
	doc1, err := resolver.Resolve(raw)
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	doc2, err := resolver.Resolve(raw)
	if err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if doc1.VerificationMethods[0].PublicKeyBase64 != doc2.VerificationMethods[0].PublicKeyBase64 {
		t.Errorf("deterministic key mismatch")
	}

	if id, err := resolver.ResolveURI(raw); err != nil || id != raw {
		t.Errorf("ResolveURI = %q, %v", id, err)
	}
}

func TestDIDDocumentJSONSerialization(t *testing.T) {
	resolver := NewUniversalDIDResolver()
	doc, err := resolver.Resolve("did:integin:tenant:tenant-saudi-contractor")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	text := string(raw)
	for _, key := range []string{`"@context"`, `"id"`, `"verificationMethod"`, `"publicKeyBase64"`, `"service"`, `"serviceEndpoint"`} {
		if !strings.Contains(text, key) {
			t.Errorf("marshaled document missing %s", key)
		}
	}
	var round DIDDocument
	if err := json.Unmarshal(raw, &round); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if round.ID != doc.ID || round.VerificationMethods[0].PublicKeyBase64 != doc.VerificationMethods[0].PublicKeyBase64 {
		t.Errorf("JSON round-trip mismatch: %s", text)
	}
}

func TestDIDDocumentRejectsUnsupportedAndMalformed(t *testing.T) {
	resolver := NewUniversalDIDResolver()
	if _, err := resolver.Resolve("did:integin:widget:w-1"); !errors.Is(err, ErrUnsupportedDIDType) {
		t.Errorf("unknown type must yield ErrUnsupportedDIDType, got %v", err)
	}
	if _, err := resolver.Resolve("did:other:asset:abc"); err != ErrInvalidDIDPrefix {
		t.Errorf("non-integin method must be rejected, got %v", err)
	}
	if _, err := resolver.Resolve("did:integin:asset"); err != ErrMalformedDID {
		t.Errorf("malformed DID must be rejected, got %v", err)
	}
}
