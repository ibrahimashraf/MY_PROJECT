package onboarding

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newEnrollTestServer(t *testing.T) (*httptest.Server, *EnrollmentSimulator) {
	t.Helper()
	sim, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(NewEnrollServer(sim).Handler())
	t.Cleanup(server.Close)
	return server, sim
}

// softwareSubmission builds a valid SOFTWARE submission signing the given
// nonce with a fresh keypair, mirroring SimulatedAttestationProvider.
func softwareSubmission(t *testing.T, challenge DeviceEnrollmentChallenge) DeviceEnrollmentSubmission {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return DeviceEnrollmentSubmission{
		ChallengeID:       challenge.ChallengeID,
		InspectorID:       "insp_http",
		DevicePublicKey:   hex.EncodeToString(pub),
		DeviceFingerprint: "simulator-http",
		DeviceModel:       "Simulated Rugged Tablet",
		SignedNonce:       hex.EncodeToString(ed25519.Sign(priv, []byte(challenge.Nonce))),
		Attestation:       AttestationClaim{KeyOrigin: KeyOriginSoftware},
	}
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(url, "application/json", strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, out any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

// TestEnrollServerChallengeAndSubmitRoundTrip drives the same sequence the
// field_app simulator performs: create a challenge, sign its nonce with a
// fresh SOFTWARE keypair, submit, and receive a CLAIMED trust record.
func TestEnrollServerChallengeAndSubmitRoundTrip(t *testing.T) {
	server, _ := newEnrollTestServer(t)

	challengeResp := postJSON(t, server.URL+"/challenge", map[string]string{
		"tenant_id": "ten_http", "inspector_id": "insp_http",
	})
	if challengeResp.StatusCode != http.StatusCreated {
		t.Fatalf("challenge status = %d, want 201", challengeResp.StatusCode)
	}
	var challenge DeviceEnrollmentChallenge
	decodeJSON(t, challengeResp, &challenge)
	if challenge.Nonce == "" || challenge.ChallengeID == "" {
		t.Fatalf("challenge missing nonce/ID: %+v", challenge)
	}

	submitResp := postJSON(t, server.URL+"/submit", softwareSubmission(t, challenge))
	if submitResp.StatusCode != http.StatusOK {
		t.Fatalf("submit status = %d, want 200", submitResp.StatusCode)
	}
	var record DeviceTrustRecord
	decodeJSON(t, submitResp, &record)
	if record.AttestationOrigin != string(KeyOriginSoftware) {
		t.Errorf("AttestationOrigin = %q, want SOFTWARE", record.AttestationOrigin)
	}
	if record.AttestationVerified {
		t.Errorf("simulated device must enroll CLAIMED, never VERIFIED")
	}
	if record.DeviceID == "" {
		t.Errorf("DeviceID not assigned")
	}
}

// TestEnrollServerRejectsTamperedSignature proves the HTTP path actually
// exercises the ed25519 verifier: a flipped nonce byte must be rejected.
func TestEnrollServerRejectsTamperedSignature(t *testing.T) {
	server, _ := newEnrollTestServer(t)
	challengeResp := postJSON(t, server.URL+"/challenge", map[string]string{
		"tenant_id": "ten_http", "inspector_id": "insp_http",
	})
	var challenge DeviceEnrollmentChallenge
	decodeJSON(t, challengeResp, &challenge)

	tampered := softwareSubmission(t, challenge)
	replacement := byte('0')
	if tampered.SignedNonce[len(tampered.SignedNonce)-1] == '0' {
		replacement = '1'
	}
	tampered.SignedNonce = tampered.SignedNonce[:len(tampered.SignedNonce)-1] + string(replacement)

	resp := postJSON(t, server.URL+"/submit", tampered)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("tampered signature accepted with status %d", resp.StatusCode)
	}
}

// TestEnrollServerMethodAndBodyGuards asserts the HTTP boundary stays closed
// on wrong methods and undecodable bodies.
func TestEnrollServerMethodAndBodyGuards(t *testing.T) {
	server, _ := newEnrollTestServer(t)

	getResp, err := http.Get(server.URL + "/challenge")
	if err != nil {
		t.Fatal(err)
	}
	getResp.Body.Close()
	if getResp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET /challenge status = %d, want 405", getResp.StatusCode)
	}

	badBody, err := http.Post(server.URL+"/submit", "application/json", strings.NewReader("{not json"))
	if err != nil {
		t.Fatal(err)
	}
	badBody.Body.Close()
	if badBody.StatusCode != http.StatusBadRequest {
		t.Fatalf("garbage submit status = %d, want 400", badBody.StatusCode)
	}
}
