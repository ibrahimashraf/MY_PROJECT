package verification

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
)

func TestParseDIDKeyRoundTrip(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	did := DIDKeyFromPub(pub)
	got, err := ParseDIDKey(did)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(pub) {
		t.Fatal("pubkey mismatch")
	}
}

func TestParseDIDKeyRejects(t *testing.T) {
	if _, err := ParseDIDKey("did:web:example.com"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := ParseDIDKey("did:key:bad"); err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifyEnvelope(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	payload := json.RawMessage(`{"hello":"world"}`)
	sig := ed25519.Sign(priv, []byte(payload))
	env := Envelope{
		Payload:   payload,
		Signature: base64.RawURLEncoding.EncodeToString(sig),
		DID:       DIDKeyFromPub(pub),
	}
	if err := VerifyEnvelope(env); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	env.Payload = json.RawMessage(`{"hello":"tampered"}`)
	if err := VerifyEnvelope(env); err == nil {
		t.Fatal("expected tamper failure")
	}
}

func TestPublicVerifierHTMLMatchesToolsIndex(t *testing.T) {
	if len(PublicVerifierHTML) == 0 {
		t.Fatal("PublicVerifierHTML must not be empty")
	}
	want, err := os.ReadFile("../../tools/public-verifier/index.html")
	if err != nil {
		t.Fatalf("read tools index: %v", err)
	}
	if string(want) != PublicVerifierHTML {
		t.Fatal("tools/public-verifier/index.html diverges from PublicVerifierHTML")
	}
}
