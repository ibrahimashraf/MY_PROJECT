package deviceenrollhttp

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"integin/internal/domain/device_trust"
	"integin/internal/identity"
	"integin/internal/oidchttp"
	"integin/internal/security"
	"integin/internal/shared/types"
	"integin/internal/syncstate"
)

const (
	testTenant  = "tenant-1"
	testOrg     = "org-1"
	testSecret  = "test-signing-secret"
	inspectorID = "user-inspector"
	adminID     = "user-admin"
)

type testClient struct {
	t       *testing.T
	handler *Handler
	mux     http.Handler
	now     *time.Time
}

func newTestClient(t *testing.T) *testClient {
	t.Helper()
	now := time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC)
	h, err := NewHandler(Config{
		SigningSecret: testSecret,
		// The closure captures the same variable the test advances through
		// client.now, making expiration hermetic.
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return &testClient{t: t, handler: h, mux: h.Routes(), now: &now}
}

func (c *testClient) advance(d time.Duration) {
	c.t.Helper()
	*c.now = c.now.Add(d)
}

func inspectorOrg() identity.OrganizationContext {
	return identity.OrganizationContext{
		TenantID:       testTenant,
		OrganizationID: testOrg,
		ActorID:        inspectorID,
		PrincipalType:  identity.PrincipalTypeHuman,
	}
}

func adminOrg() identity.OrganizationContext {
	org := inspectorOrg()
	org.ActorID = adminID
	org.Capabilities = []string{"admin"}
	return org
}

// adminInspectorOrg is an admin-capable actor that is also the enrollment
// requester, used to reach the self-approval guard past the admin gate.
func adminInspectorOrg() identity.OrganizationContext {
	org := inspectorOrg()
	org.Capabilities = []string{"admin"}
	return org
}

func adminByRoleOrg() identity.OrganizationContext {
	org := inspectorOrg()
	org.ActorID = adminID
	org.Roles = []string{"Tenant Administrator"}
	return org
}

func nonAdminOrg() identity.OrganizationContext {
	org := inspectorOrg()
	org.ActorID = adminID
	org.Capabilities = []string{"inspection.read"}
	return org
}

func serviceAccountOrg() identity.OrganizationContext {
	org := inspectorOrg()
	org.ActorID = "sa:provisioner"
	org.PrincipalType = identity.PrincipalTypeServiceAccount
	return org
}

func (c *testClient) perform(org identity.OrganizationContext, method, path string, body any) *httptest.ResponseRecorder {
	c.t.Helper()
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(payload)
	}
	req := httptest.NewRequest(method, path, reader)
	if org.ActorID != "" {
		req = req.WithContext(oidchttp.WithOrganizationContext(req.Context(), org))
	}
	rec := httptest.NewRecorder()
	c.mux.ServeHTTP(rec, req)
	return rec
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, out any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
}

func errorMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return body.Error
}

// newKey derives a keypair and returns the hex public key and derived key id.
func newKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, string, string) {
	t.Helper()
	publicKey, privateKey, err := security.GenerateDeviceKeyPair()
	if err != nil {
		t.Fatalf("GenerateDeviceKeyPair: %v", err)
	}
	return publicKey, privateKey, hex.EncodeToString(publicKey), security.DeviceKeyID(publicKey)
}

func signNonce(t *testing.T, privateKey ed25519.PrivateKey, nonce string) string {
	t.Helper()
	signature, err := security.SignDeviceMutation(privateKey, []byte(nonce))
	if err != nil {
		t.Fatalf("SignDeviceMutation: %v", err)
	}
	return hex.EncodeToString(signature)
}

type createResponse struct {
	RequestID string `json:"request_id"`
	Challenge string `json:"challenge"`
	ExpiresAt string `json:"expires_at"`
}

type proofResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
}

type approveResponse struct {
	AuthorityPackage device_trust.AuthorityPackage `json:"authority_package"`
}

// createChallenge posts a valid challenge request and returns the parsed response.
func (c *testClient) createChallenge(org identity.OrganizationContext, publicKeyHex, keyID string) (createResponse, *httptest.ResponseRecorder) {
	c.t.Helper()
	rec := c.perform(org, http.MethodPost, "/v1/device-enrollment/requests", map[string]any{
		"protocol_version":  protocolVersion,
		"device_public_key": publicKeyHex,
		"key_id":            keyID,
	})
	var response createResponse
	if rec.Code == http.StatusCreated {
		decodeJSON(c.t, rec, &response)
	}
	return response, rec
}

// prove completes the challenge proof for a request.
func (c *testClient) prove(org identity.OrganizationContext, requestID, publicKeyHex, signature string) proofResponse {
	c.t.Helper()
	rec := c.perform(org, http.MethodPost, "/v1/device-enrollment/requests/"+requestID+"/proof", map[string]any{
		"device_public_key": publicKeyHex,
		"signature":         signature,
	})
	if rec.Code != http.StatusOK {
		c.t.Fatalf("proof status %d: %s", rec.Code, rec.Body.String())
	}
	var response proofResponse
	decodeJSON(c.t, rec, &response)
	return response
}

// signAndProve completes the full proof phase and returns the proof response.
func (c *testClient) signAndProve(org identity.OrganizationContext, requestID, publicKeyHex string, privateKey ed25519.PrivateKey, nonce string) proofResponse {
	c.t.Helper()
	return c.prove(org, requestID, publicKeyHex, signNonce(c.t, privateKey, nonce))
}

// approveRequest approves a request and returns the authority package.
func (c *testClient) approveRequest(org identity.OrganizationContext, requestID string) approveResponse {
	c.t.Helper()
	rec := c.perform(org, http.MethodPost, "/v1/device-enrollment/requests/"+requestID+"/approve", nil)
	if rec.Code != http.StatusOK {
		c.t.Fatalf("approve status %d: %s", rec.Code, rec.Body.String())
	}
	var response approveResponse
	decodeJSON(c.t, rec, &response)
	return response
}

// enroll drives the request -> proof -> approve pipeline and returns the
// issued authority package, the derived device id, and the challenge.
func (c *testClient) enroll() (device_trust.AuthorityPackage, string, ed25519.PublicKey, ed25519.PrivateKey) {
	c.t.Helper()
	publicKey, privateKey, publicKeyHex, keyID := newKey(c.t)
	challenge, rec := c.createChallenge(inspectorOrg(), publicKeyHex, keyID)
	if rec.Code != http.StatusCreated {
		c.t.Fatalf("create status %d: %s", rec.Code, rec.Body.String())
	}
	proof := c.signAndProve(inspectorOrg(), challenge.RequestID, publicKeyHex, privateKey, challenge.Challenge)
	if proof.Status != "proof_verified" {
		c.t.Fatalf("unexpected proof status %q", proof.Status)
	}
	approval := c.approveRequest(adminOrg(), challenge.RequestID)
	return approval.AuthorityPackage, deriveDeviceID(publicKey), publicKey, privateKey
}

func TestFullEnrollmentLifecycle(t *testing.T) {
	client := newTestClient(t)
	authority, deviceID, publicKey, _ := client.enroll()

	if authority.Epoch != 1 {
		t.Fatalf("expected epoch 1, got %d", authority.Epoch)
	}
	if authority.TenantID != testTenant || authority.UserID != inspectorID {
		t.Fatalf("unexpected authority scope: %+v", authority)
	}
	if len(authority.Scopes) != 2 {
		t.Fatalf("expected default scopes, got %v", authority.Scopes)
	}
	if !authority.ExpiresAt.Equal(authority.IssuedAt.Add(8 * time.Hour)) {
		t.Fatalf("expected 8h authority TTL, got %v -> %v", authority.IssuedAt, authority.ExpiresAt)
	}
	if authority.Signature == "" {
		t.Fatal("expected signed authority package")
	}

	device, err := client.handler.Device(context.Background(), testTenant, deviceID)
	if err != nil {
		t.Fatalf("Device: %v", err)
	}
	if device.State != syncstate.DeviceTrusted || device.AuthorityEpoch != 1 {
		t.Fatalf("unexpected device record: %+v", device)
	}
	if device.KeyID != security.DeviceKeyID(publicKey) {
		t.Fatalf("key id mismatch: %q", device.KeyID)
	}

	restored, err := device_trust.RestoreDevice(
		device.DeviceID, device.TenantID, device.OrganizationID,
		device.UserID, hex.EncodeToString(device.PublicKey),
		types.DeviceTrustState(device.State), device.AuthorityEpoch,
	)
	if err != nil {
		t.Fatalf("RestoreDevice: %v", err)
	}
	if err := device_trust.ValidateAuthorityPackage(authority, restored, testSecret, time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("auth authority package: %v", err)
	}
}

func TestCreateRequestValidation(t *testing.T) {
	_, _, publicKeyHex, keyID := newKey(t)
	base := "/v1/device-enrollment/requests"
	valid := map[string]any{"protocol_version": protocolVersion, "device_public_key": publicKeyHex, "key_id": keyID}

	cases := []struct {
		name       string
		org        identity.OrganizationContext
		body       any
		wantStatus int
		wantError  string
	}{
		{"client-supplied tenant", inspectorOrg(), func() map[string]any {
			m := map[string]any{}
			for k, v := range valid {
				m[k] = v
			}
			m["tenant_id"] = "other-tenant"
			return m
		}(), http.StatusBadRequest, "server-derived"},
		{"client-supplied role", inspectorOrg(), func() map[string]any {
			m := map[string]any{}
			for k, v := range valid {
				m[k] = v
			}
			m["role"] = "Super Admin"
			return m
		}(), http.StatusBadRequest, "server-derived"},
		{"bad protocol version", inspectorOrg(), func() map[string]any {
			m := map[string]any{}
			for k, v := range valid {
				m[k] = v
			}
			m["protocol_version"] = "v0"
			return m
		}(), http.StatusBadRequest, "unsupported"},
		{"bad public key", inspectorOrg(), func() map[string]any {
			m := map[string]any{}
			for k, v := range valid {
				m[k] = v
			}
			m["device_public_key"] = "abcd"
			return m
		}(), http.StatusBadRequest, "invalid device_public_key"},
		{"key id mismatch", inspectorOrg(), func() map[string]any {
			m := map[string]any{}
			for k, v := range valid {
				m[k] = v
			}
			m["key_id"] = strings.Repeat("00", 32)
			return m
		}(), http.StatusBadRequest, "key_id"},
		{"unauthenticated", identity.OrganizationContext{}, valid, http.StatusUnauthorized, "authentication_failed"},
		{"service account", serviceAccountOrg(), valid, http.StatusForbidden, "forbidden"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(t)
			rec := client.perform(tc.org, http.MethodPost, base, tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status %d, want %d: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantError != "" && !strings.Contains(errorMessage(t, rec), tc.wantError) {
				t.Fatalf("error %q, want substring %q", errorMessage(t, rec), tc.wantError)
			}
		})
	}

	t.Run("method not allowed", func(t *testing.T) {
		client := newTestClient(t)
		rec := client.perform(inspectorOrg(), http.MethodGet, base, nil)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status %d, want 405", rec.Code)
		}
	})

	t.Run("unknown member rejected", func(t *testing.T) {
		client := newTestClient(t)
		rec := client.perform(inspectorOrg(), http.MethodPost, base, map[string]any{
			"protocol_version": protocolVersion, "device_public_key": publicKeyHex, "key_id": keyID, "evil": true,
		})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d, want 400", rec.Code)
		}
	})
}

func TestChallengeSingleUseFailsClosed(t *testing.T) {
	client := newTestClient(t)
	_, privateKey, publicKeyHex, keyID := newKey(t)
	challenge, rec := client.createChallenge(inspectorOrg(), publicKeyHex, keyID)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status %d", rec.Code)
	}
	signature := signNonce(t, privateKey, challenge.Challenge)
	client.prove(inspectorOrg(), challenge.RequestID, publicKeyHex, signature)

	replay := client.perform(inspectorOrg(), http.MethodPost, "/v1/device-enrollment/requests/"+challenge.RequestID+"/proof", map[string]any{
		"device_public_key": publicKeyHex,
		"signature":         signature,
	})
	if replay.Code != http.StatusGone {
		t.Fatalf("replay status %d, want 410: %s", replay.Code, replay.Body.String())
	}
	if !strings.Contains(errorMessage(t, replay), "consumed") {
		t.Fatalf("unexpected replay error: %q", errorMessage(t, replay))
	}
}

func TestChallengeExpirationFailsClosed(t *testing.T) {
	client := newTestClient(t)
	_, privateKey, publicKeyHex, keyID := newKey(t)
	challenge, rec := client.createChallenge(inspectorOrg(), publicKeyHex, keyID)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status %d", rec.Code)
	}

	client.advance(6 * time.Minute)
	expired := client.perform(inspectorOrg(), http.MethodPost, "/v1/device-enrollment/requests/"+challenge.RequestID+"/proof", map[string]any{
		"device_public_key": publicKeyHex,
		"signature":         signNonce(t, privateKey, challenge.Challenge),
	})
	if expired.Code != http.StatusBadRequest || !strings.Contains(errorMessage(t, expired), "expired") {
		t.Fatalf("expired proof status %d: %s", expired.Code, expired.Body.String())
	}
}

func TestApprovalWindowExpirationFailsClosed(t *testing.T) {
	client := newTestClient(t)
	_, privateKey, publicKeyHex, keyID := newKey(t)
	challenge, rec := client.createChallenge(inspectorOrg(), publicKeyHex, keyID)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status %d", rec.Code)
	}
	client.signAndProve(inspectorOrg(), challenge.RequestID, publicKeyHex, privateKey, challenge.Challenge)

	client.advance(25 * time.Hour)
	approval := client.perform(adminOrg(), http.MethodPost, "/v1/device-enrollment/requests/"+challenge.RequestID+"/approve", nil)
	if approval.Code != http.StatusBadRequest || !strings.Contains(errorMessage(t, approval), "approval window") {
		t.Fatalf("expired approval status %d: %s", approval.Code, approval.Body.String())
	}
}

func TestSelfApprovalRejected(t *testing.T) {
	client := newTestClient(t)
	_, privateKey, publicKeyHex, keyID := newKey(t)
	challenge, rec := client.createChallenge(inspectorOrg(), publicKeyHex, keyID)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status %d", rec.Code)
	}
	client.signAndProve(inspectorOrg(), challenge.RequestID, publicKeyHex, privateKey, challenge.Challenge)

	approval := client.perform(adminInspectorOrg(), http.MethodPost, "/v1/device-enrollment/requests/"+challenge.RequestID+"/approve", nil)
	if approval.Code != http.StatusForbidden {
		t.Fatalf("self-approval status %d, want 403: %s", approval.Code, approval.Body.String())
	}
	if !strings.Contains(errorMessage(t, approval), "self-approval") {
		t.Fatalf("unexpected error: %q", errorMessage(t, approval))
	}

	if _, err := client.handler.Device(context.Background(), testTenant, "dev_whatever"); err == nil {
		t.Fatal("device must not exist after rejected self-approval")
	}
}

func TestInsufficientRoleRejected(t *testing.T) {
	client := newTestClient(t)
	_, privateKey, publicKeyHex, keyID := newKey(t)
	challenge, rec := client.createChallenge(inspectorOrg(), publicKeyHex, keyID)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status %d", rec.Code)
	}
	client.signAndProve(inspectorOrg(), challenge.RequestID, publicKeyHex, privateKey, challenge.Challenge)

	approval := client.perform(nonAdminOrg(), http.MethodPost, "/v1/device-enrollment/requests/"+challenge.RequestID+"/approve", nil)
	if approval.Code != http.StatusForbidden {
		t.Fatalf("non-admin approval status %d, want 403: %s", approval.Code, approval.Body.String())
	}
	if !strings.Contains(errorMessage(t, approval), "tenant administrator") {
		t.Fatalf("unexpected error: %q", errorMessage(t, approval))
	}
}

func TestAdminViaRoleAllowed(t *testing.T) {
	client := newTestClient(t)
	_, privateKey, publicKeyHex, keyID := newKey(t)
	challenge, rec := client.createChallenge(inspectorOrg(), publicKeyHex, keyID)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status %d", rec.Code)
	}
	client.signAndProve(inspectorOrg(), challenge.RequestID, publicKeyHex, privateKey, challenge.Challenge)
	approval := client.perform(adminByRoleOrg(), http.MethodPost, "/v1/device-enrollment/requests/"+challenge.RequestID+"/approve", nil)
	if approval.Code != http.StatusOK {
		t.Fatalf("role-admin approval status %d: %s", approval.Code, approval.Body.String())
	}
}

func TestInvalidProofSignatureRejected(t *testing.T) {
	client := newTestClient(t)
	_, privateKey, publicKeyHex, keyID := newKey(t)
	challenge, rec := client.createChallenge(inspectorOrg(), publicKeyHex, keyID)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status %d", rec.Code)
	}

	wrongKeyContext := "forged"
	forged := client.perform(inspectorOrg(), http.MethodPost, "/v1/device-enrollment/requests/"+challenge.RequestID+"/proof", map[string]any{
		"device_public_key": publicKeyHex,
		"signature":         signNonce(t, privateKey, wrongKeyContext),
	})
	if forged.Code != http.StatusBadRequest || !strings.Contains(errorMessage(t, forged), "proof of possession") {
		t.Fatalf("forged proof status %d: %s", forged.Code, forged.Body.String())
	}

	_, _, wrongKeyHex, _ := newKey(t)
	substitution := client.perform(inspectorOrg(), http.MethodPost, "/v1/device-enrollment/requests/"+challenge.RequestID+"/proof", map[string]any{
		"device_public_key": wrongKeyHex,
		"signature":         signNonce(t, privateKey, challenge.Challenge),
	})
	if substitution.Code != http.StatusForbidden {
		t.Fatalf("key substitution status %d, want 403: %s", substitution.Code, substitution.Body.String())
	}

	approval := client.perform(adminOrg(), http.MethodPost, "/v1/device-enrollment/requests/"+challenge.RequestID+"/approve", nil)
	if approval.Code != http.StatusBadRequest || !strings.Contains(errorMessage(t, approval), "proof of possession") {
		t.Fatalf("approve after failed proof status %d: %s", approval.Code, approval.Body.String())
	}
}

func TestRevocationFlowBumpsEpochAndRevokesAuthority(t *testing.T) {
	client := newTestClient(t)
	authority, deviceID, _, _ := client.enroll()

	noReason := client.perform(adminOrg(), http.MethodPost, "/v1/devices/"+deviceID+"/revoke", map[string]string{"reason": " "})
	if noReason.Code != http.StatusBadRequest {
		t.Fatalf("missing-reason revoke status %d, want 400", noReason.Code)
	}

	missing := client.perform(adminOrg(), http.MethodPost, "/v1/devices/dev_nonexistent/revoke", map[string]string{"reason": "lost"})
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing-device revoke status %d, want 404: %s", missing.Code, missing.Body.String())
	}

	nonAdmin := client.perform(nonAdminOrg(), http.MethodPost, "/v1/devices/"+deviceID+"/revoke", map[string]string{"reason": "lost"})
	if nonAdmin.Code != http.StatusForbidden {
		t.Fatalf("non-admin revoke status %d, want 403: %s", nonAdmin.Code, nonAdmin.Body.String())
	}

	revoked := client.perform(adminOrg(), http.MethodPost, "/v1/devices/"+deviceID+"/revoke", map[string]string{"reason": "device reported stolen"})
	if revoked.Code != http.StatusOK {
		t.Fatalf("revoke status %d: %s", revoked.Code, revoked.Body.String())
	}
	var revokedBody struct {
		DeviceID       string `json:"device_id"`
		Status         string `json:"status"`
		AuthorityEpoch uint64 `json:"authority_epoch"`
	}
	decodeJSON(t, revoked, &revokedBody)
	if revokedBody.Status != "REVOKED" || revokedBody.AuthorityEpoch != 2 {
		t.Fatalf("unexpected revoke body: %+v", revokedBody)
	}

	handler := client.handler
	device, err := handler.Device(context.Background(), testTenant, deviceID)
	if err != nil {
		t.Fatalf("Device: %v", err)
	}
	if device.State != syncstate.DeviceRevoked || device.AuthorityEpoch != 2 {
		t.Fatalf("device not revoked: %+v", device)
	}
	if device.RevokedAt == nil || device.RevocationReason != "device reported stolen" {
		t.Fatalf("revocation metadata missing: %+v", device)
	}

	authorities, err := handler.Authorities(context.Background(), testTenant, deviceID)
	if err != nil {
		t.Fatalf("Authorities: %v", err)
	}
	if len(authorities) != 1 || authorities[0].AuthorityID != authority.ID {
		t.Fatalf("unexpected authorities: %+v", authorities)
	}
	if authorities[0].RevokedAt == nil {
		t.Fatal("authority package must be revoked")
	}

	restored, err := device_trust.RestoreDevice(
		device.DeviceID, device.TenantID, device.OrganizationID,
		device.UserID, hex.EncodeToString(device.PublicKey),
		types.DeviceTrustState(device.State), device.AuthorityEpoch,
	)
	if err != nil {
		t.Fatalf("RestoreDevice: %v", err)
	}
	if err := device_trust.ValidateAuthorityPackage(authority, restored, testSecret, time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("stale authority package must not validate after revocation")
	}
}

func TestConcurrentEnrollmentLifecyclesRaceClean(t *testing.T) {
	client := newTestClient(t)

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := client.enrollConcurrent()
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

// enrollConcurrent runs the full pipeline sharing one client; races on
// internal maps are caught by -race. It never calls t.Fatalf so it is safe to
// run from goroutines.
func (c *testClient) enrollConcurrent() (device_trust.AuthorityPackage, string, error) {
	publicKey, privateKey, publicKeyHex, keyID := newKey(c.t)
	challenge, rec := c.createChallenge(inspectorOrg(), publicKeyHex, keyID)
	if rec.Code != http.StatusCreated {
		return device_trust.AuthorityPackage{}, "", fmt.Errorf("create status %d: %s", rec.Code, rec.Body.String())
	}
	proof := c.prove(inspectorOrg(), challenge.RequestID, publicKeyHex, signNonce(c.t, privateKey, challenge.Challenge))
	if proof.Status != "proof_verified" {
		return device_trust.AuthorityPackage{}, "", fmt.Errorf("proof status %q", proof.Status)
	}
	rec = c.perform(adminOrg(), http.MethodPost, "/v1/device-enrollment/requests/"+challenge.RequestID+"/approve", nil)
	if rec.Code != http.StatusOK {
		return device_trust.AuthorityPackage{}, "", fmt.Errorf("approve status %d: %s", rec.Code, rec.Body.String())
	}
	var approval approveResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &approval); err != nil {
		return device_trust.AuthorityPackage{}, "", err
	}
	return approval.AuthorityPackage, deriveDeviceID(publicKey), nil
}
