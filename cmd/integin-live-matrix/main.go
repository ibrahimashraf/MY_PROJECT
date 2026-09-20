// Command integin-live-matrix seeds an isolated durable integration fixture and
// exercises the running INTEGIN HTTP server. It is opt-in: all credentials and
// the private fixture path are supplied by local environment variables.
package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"integin/internal/domain/device_trust"
	domainsync "integin/internal/domain/sync"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/security"
	"integin/internal/shared/types"
	"integin/internal/syncstate"
	"integin/internal/workorderauth"
)

const (
	integrationOrganization = "integin-integration-org"
	integrationUser         = "integin-integration-user"
	// integrationActorID is the stable actor behind the matrix OIDC service
	// account. ActorFromMembership rejects capability-less actors, so the
	// seed grants CapabilityAddEvidenceReference below — same pattern as the
	// repo's OIDC integration tests.
	integrationActorID = "svc-live-matrix"
)

type fixture struct {
	TenantID     string `json:"tenant_id"`
	Organization string `json:"organization_id"`
	UserID       string `json:"user_id"`
	DeviceID     string `json:"device_id"`
	AuthorityID  string `json:"authority_id"`
	KeyID        string `json:"key_id"`
	PrivateKey   string `json:"private_key_base64"`
}

type syncRequest struct {
	ProtocolVersion    string          `json:"protocol_version"`
	TransactionID      string          `json:"transaction_id"`
	TenantID           string          `json:"tenant_id"`
	OrganizationID     string          `json:"organization_id"`
	Environment        string          `json:"environment"`
	DeviceID           string          `json:"device_id"`
	UserID             string          `json:"user_id"`
	SequenceNumber     uint64          `json:"sequence_number"`
	Operation          string          `json:"operation"`
	EntityID           string          `json:"entity_id"`
	Payload            json.RawMessage `json:"payload"`
	PayloadHash        string          `json:"payload_hash"`
	CapturedAt         time.Time       `json:"captured_at"`
	AuthorityID        string          `json:"authority_id"`
	AuthorityEpoch     uint64          `json:"authority_epoch"`
	SignatureAlgorithm string          `json:"signature_algorithm"`
	KeyID              string          `json:"key_id"`
	Signature          string          `json:"signature"`
}

type outcomeResponse struct {
	Outcome string `json:"outcome"`
	Reason  string `json:"reason"`
}

type evidenceRequest struct {
	TenantID         string `json:"tenant_id"`
	OrganizationID   string `json:"organization_id"`
	EvidenceID       string `json:"evidence_id"`
	InspectionID     string `json:"inspection_id"`
	ContentType      string `json:"content_type"`
	PlaintextSHA256  string `json:"plaintext_sha256"`
	CiphertextSHA256 string `json:"ciphertext_sha256"`
	Base64Blob       string `json:"base64_blob"`
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: integin-live-matrix <seed|exercise>")
	}
	switch os.Args[1] {
	case "seed":
		if err := seed(); err != nil {
			log.Fatal(err)
		}
	case "exercise":
		if err := exercise(); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("usage: integin-live-matrix <seed|exercise>")
	}
}

func seed() error {
	databaseURL, tenantID, secret, fixturePath, err := localConfig()
	if err != nil {
		return err
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	repository, err := syncstate.NewPostgresRepository(db)
	if err != nil {
		return err
	}
	publicKey, privateKey, err := security.GenerateDeviceKeyPair()
	if err != nil {
		return err
	}
	runID := time.Now().UTC().Format("20060102T150405.000000000")
	deviceID := "integin-live-device-" + runID
	authorityID := "integin-live-authority-" + runID
	encodedPublicKey := base64.StdEncoding.EncodeToString(publicKey)
	keyID := security.DeviceKeyID(publicKey)
	// PostgreSQL TIMESTAMPTZ preserves microseconds. Sign at that same precision
	// so a reloaded authority package has an identical canonical timestamp.
	now := time.Now().UTC().Truncate(time.Microsecond)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := repository.SaveDevice(ctx, syncstate.DeviceRecord{
		DeviceID: deviceID, TenantID: tenantID, OrganizationID: integrationOrganization, UserID: integrationUser,
		KeyID: keyID, PublicKey: publicKey, State: syncstate.DeviceTrusted, AuthorityEpoch: 1, EnrolledAt: now, UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("save integration device: %w", err)
	}
	device, err := device_trust.RestoreDevice(deviceID, tenantID, integrationOrganization, integrationUser, encodedPublicKey, types.DeviceTrusted, 1)
	if err != nil {
		return err
	}
	authority, err := device_trust.IssueAuthorityPackage(device, authorityID, secret, []string{"inspection.perform", "evidence.upload"}, now, 2*time.Hour)
	if err != nil {
		return err
	}
	if err := repository.SaveAuthority(ctx, syncstate.AuthorityRecord{
		AuthorityID: authority.ID, TenantID: authority.TenantID, OrganizationID: integrationOrganization, DeviceID: authority.DeviceID,
		UserID: authority.UserID, AuthorityEpoch: authority.Epoch, Scopes: authority.Scopes, ProcedureVersion: "integin-live-v1",
		IssuedAt: authority.IssuedAt, ExpiresAt: authority.ExpiresAt, Signature: []byte(authority.Signature),
	}); err != nil {
		return fmt.Errorf("save integration authority: %w", err)
	}
	encodedPrivateKey := base64.StdEncoding.EncodeToString(privateKey)
	//nolint:gosec // PrivateKey in fixture for live matrix; file written with 0600
	content, err := json.MarshalIndent(fixture{TenantID: tenantID, Organization: integrationOrganization, UserID: integrationUser, DeviceID: deviceID, AuthorityID: authorityID, KeyID: keyID, PrivateKey: encodedPrivateKey}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fixturePath), 0700); err != nil {
		return err
	}
	if err := os.WriteFile(fixturePath, content, 0600); err != nil {
		return err
	}
	fmt.Printf("seeded tenant=%s device=%s authority=%s fixture=%s\n", tenantID, deviceID, authorityID, fixturePath)
	if err := seedServiceAccountIdentity(db, tenantID); err != nil {
		return err
	}
	return nil
}

func explicitPilotServerURL(raw string) (string, error) {
	serverURL := strings.TrimRight(strings.TrimSpace(raw), "/")
	if serverURL != "http://127.0.0.1:18080" {
		return "", errors.New("INTEGIN_SERVER_URL must explicitly target the isolated pilot origin http://127.0.0.1:18080")
	}
	return serverURL, nil
}
func exercise() error {
	_, _, _, fixturePath, err := localConfig()
	if err != nil {
		return err
	}
	//nolint:gosec // path is CLI arg/config for test tool
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		return fmt.Errorf("read live fixture: %w", err)
	}
	var value fixture
	if err := json.Unmarshal(content, &value); err != nil {
		return err
	}
	privateKey, err := base64.StdEncoding.DecodeString(value.PrivateKey)
	if err != nil || len(privateKey) != ed25519.PrivateKeySize {
		return errors.New("fixture has invalid private key")
	}
	serverURL, err := explicitPilotServerURL(os.Getenv("INTEGIN_SERVER_URL"))
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 15 * time.Second}

	// Acquire OIDC token for authenticated evidence calls.
	// INTEGIN_OIDC_TOKEN_ENDPOINT overrides the legacy Keycloak-derived path so any
	// RS256 OIDC provider (Keycloak, Casdoor) can serve the pilot harness.
	tokenEndpoint := strings.TrimSpace(os.Getenv("INTEGIN_OIDC_TOKEN_ENDPOINT"))
	if tokenEndpoint == "" {
		tokenEndpoint = strings.TrimSuffix(os.Getenv("INTEGIN_OIDC_ISSUER"), "/") + "/protocol/openid-connect/token"
	}
	tokenProvider, err := oidcauth.NewClientCredentialsTokenProvider(oidcauth.ClientCredentialsConfig{
		TokenEndpoint: tokenEndpoint,
		ClientID:      os.Getenv("INTEGIN_OIDC_CLIENT_ID"),
		ClientSecret:  os.Getenv("INTEGIN_OIDC_CLIENT_SECRET"),
		RefreshBuffer: 30 * time.Second,
	}, client)
	if err != nil {
		return fmt.Errorf("creating token provider: %w", err)
	}
	token, err := tokenProvider.GetToken(context.Background())
	if err != nil {
		return fmt.Errorf("obtaining OIDC token: %w", err)
	}

	// runID scopes every idempotency key to this specific exercise invocation.
	// Without it, repeated runs collide in sync_idempotency_cache when the
	// CapturedAt timestamp differs (producing a 409 "key reused with different
	// request" error from the middleware).
	runID := fmt.Sprintf("%d", time.Now().UnixNano())
	payload := []byte(`{"inspection_id":"integin-live-inspection","status":"submitted"}`)
	txPrefix := "integin-live-" + value.DeviceID + "-" + runID
	base := signedTransaction(txPrefix+"-applied", 1, value, ed25519.PrivateKey(privateKey), payload)
	if err := expectSync(client, serverURL, base, "APPLIED", "lk-"+txPrefix+"-applied"); err != nil {
		return err
	}
	// DUPLICATE is a business-level (not idempotency-layer) outcome: the same
	// transaction id + payload is re-submitted with a distinct idempotency key
	// so the middleware passes it through to the processor, which re-derives
	// the existing-receipt DUPLICATE verdict.
	if err := expectSync(client, serverURL, base, "DUPLICATE", "lk-"+txPrefix+"-duplicate"); err != nil {
		return err
	}
	held := signedTransaction(txPrefix+"-held", 3, value, ed25519.PrivateKey(privateKey), payload)
	if err := expectSync(client, serverURL, held, "HELD", "lk-"+txPrefix+"-held"); err != nil {
		return err
	}
	conflict := signedTransaction(txPrefix+"-applied", 2, value, ed25519.PrivateKey(privateKey), []byte(`{"inspection_id":"integin-live-inspection","status":"changed"}`))
	if err := expectSync(client, serverURL, conflict, "CONFLICT", "lk-"+txPrefix+"-conflict"); err != nil {
		return err
	}
	securityFailure := signedTransaction(txPrefix+"-security", 2, value, ed25519.PrivateKey(privateKey), payload)
	securityFailure.Signature = "invalid-signature"
	if err := expectSync(client, serverURL, securityFailure, "SECURITY_FAILURE", "lk-"+txPrefix+"-security"); err != nil {
		return err
	}

	return exerciseEvidence(client, serverURL, value, token, runID)
}

func signedTransaction(id string, sequence uint64, value fixture, privateKey ed25519.PrivateKey, payload []byte) domainsync.Transaction {
	tx := domainsync.NewTransaction(id, value.TenantID, value.DeviceID, value.UserID, sequence, "InspectionSubmitted", payload)
	tx.OrganizationID = value.Organization
	tx.EntityID = "integin-live-inspection"
	tx.AuthorityID = value.AuthorityID
	tx.AuthorityEpoch = 1
	tx.CapturedAt = time.Now().UTC()
	signed, err := domainsync.SignTransactionEd25519(tx, privateKey, value.KeyID)
	if err != nil {
		panic(err)
	}
	return signed
}

func expectSync(client *http.Client, serverURL string, tx domainsync.Transaction, expected string, idempotencyKey string) error {
	requestValue := syncRequest{ProtocolVersion: tx.ProtocolVersion, TransactionID: tx.TransactionID, TenantID: tx.TenantID, OrganizationID: tx.OrganizationID, Environment: tx.Environment, DeviceID: tx.DeviceID, UserID: tx.UserID, SequenceNumber: tx.SequenceNumber, Operation: tx.Operation, EntityID: tx.EntityID, Payload: tx.Payload, PayloadHash: tx.PayloadHash, CapturedAt: tx.CapturedAt, AuthorityID: tx.AuthorityID, AuthorityEpoch: tx.AuthorityEpoch, SignatureAlgorithm: tx.SignatureAlgorithm, KeyID: tx.KeyID, Signature: tx.Signature}
	// Each logical write uses a distinct Stripe-style idempotency key so the
	// middleware never short-circuits a scenario; the sync processor's native
	// APPLIED/DUPLICATE/HELD/CONFLICT/SECURITY_FAILURE verdicts are exercised.
	response, err := postJSON(client, serverURL+"/sync", requestValue, idempotencyKey)
	if err != nil {
		return err
	}
	if response.Outcome != expected {
		return fmt.Errorf("sync %s outcome=%s want=%s reason=%s", tx.TransactionID, response.Outcome, expected, response.Reason)
	}
	return nil
}

func exerciseEvidence(client *http.Client, serverURL string, value fixture, token string, runID string) error {
	ciphertext := []byte("INTEGIN live encrypted evidence matrix bytes")
	plaintext := []byte("INTEGIN live plaintext evidence matrix bytes")
	requestValue := evidenceRequest{TenantID: value.TenantID, OrganizationID: value.Organization, EvidenceID: "integin-live-evidence-" + value.DeviceID + "-" + runID, InspectionID: "integin-live-inspection", ContentType: "application/octet-stream", PlaintextSHA256: digest(plaintext), CiphertextSHA256: digest(ciphertext), Base64Blob: base64.StdEncoding.EncodeToString(ciphertext)}
	evidenceKey := "ev-" + requestValue.EvidenceID
	response, err := postJSONAuth(client, serverURL+"/evidence", requestValue, token, evidenceKey)
	if err != nil {
		return err
	}
	if response.Outcome != "APPLIED" {
		return fmt.Errorf("evidence applied outcome=%s reason=%s", response.Outcome, response.Reason)
	}
	// DUPLICATE is the evidence handler's native outcome for re-uploading the
	// same blob under a distinct idempotency key; reusing evidenceKey would hand
	// control to the idempotency middleware replay instead of the handler.
	response, err = postJSONAuth(client, serverURL+"/evidence", requestValue, token, evidenceKey+"-duplicate")
	if err != nil {
		return err
	}
	if response.Outcome != "DUPLICATE" {
		return fmt.Errorf("evidence duplicate outcome=%s reason=%s", response.Outcome, response.Reason)
	}
	conflicting := requestValue
	conflicting.Base64Blob = base64.StdEncoding.EncodeToString([]byte("different ciphertext"))
	conflicting.CiphertextSHA256 = digest([]byte("different ciphertext"))
	// A distinct key allows the evidence handler's native CONFLICT verdict (same
	// evidence id, different content) to be produced; reusing evidenceKey would
	// hand the deterministic-failure response to the idempotency middleware
	// instead of the handler.
	response, err = postJSONAuth(client, serverURL+"/evidence", conflicting, token, evidenceKey+"-conflict")
	if err != nil {
		return err
	}
	if response.Outcome != "CONFLICT" {
		return fmt.Errorf("evidence conflict outcome=%s reason=%s", response.Outcome, response.Reason)
	}
	tampered := requestValue
	tampered.CiphertextSHA256 = strings.Repeat("0", 64)
	// A distinct key keeps the tamper probe out of the cached CONFLICT entry so
	// the evidence handler's integrity gate is what produces SECURITY_FAILURE.
	response, err = postJSONAuth(client, serverURL+"/evidence", tampered, token, evidenceKey+"-tamper")
	if err != nil {
		return err
	}
	if response.Outcome != "SECURITY_FAILURE" {
		return fmt.Errorf("evidence security_failure outcome=%s reason=%s", response.Outcome, response.Reason)
	}
	return nil
}

// postJSONAuth performs an authenticated POST request with a Bearer token (used for /evidence endpoints).
func postJSONAuth(client *http.Client, endpoint string, value any, token string, idempotencyKey string) (outcomeResponse, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return outcomeResponse{}, err
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "http" && u.Scheme != "https" || u.Host == "" {
		return outcomeResponse{}, fmt.Errorf("invalid endpoint URL: %s", endpoint)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return outcomeResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	response, err := client.Do(req) //nolint:gosec // endpoint validated above
	if err != nil {
		return outcomeResponse{}, err
	}
	defer response.Body.Close()
	resultBody, _ := io.ReadAll(response.Body)
	var result outcomeResponse
	if err := json.Unmarshal(resultBody, &result); err != nil {
		return outcomeResponse{}, fmt.Errorf("decode %s response %d: %w", endpoint, response.StatusCode, err)
	}
	// Evidence conflicts and integrity failures intentionally use non-2xx HTTP
	// statuses while still returning a structured authoritative outcome. Callers
	// assert the outcome instead of treating an expected conflict as transport
	// failure.
	return result, nil
}

// postJSON performs an unauthenticated POST request (used for /sync endpoints).
func postJSON(client *http.Client, endpoint string, value any, idempotencyKey string) (outcomeResponse, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return outcomeResponse{}, err
	}

	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "http" && u.Scheme != "https" || u.Host == "" {
		return outcomeResponse{}, fmt.Errorf("invalid endpoint URL: %s", endpoint)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return outcomeResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	// Authorization header not needed for unauthenticated postJSON
	response, err := client.Do(req) //nolint:gosec // endpoint validated above
	if err != nil {
		return outcomeResponse{}, err
	}
	defer response.Body.Close()
	resultBody, _ := io.ReadAll(response.Body)
	var result outcomeResponse
	if err := json.Unmarshal(resultBody, &result); err != nil {
		return outcomeResponse{}, fmt.Errorf("decode %s response %d: %w", endpoint, response.StatusCode, err)
	}
	// Evidence conflicts and integrity failures intentionally use non-2xx HTTP
	// statuses while still returning a structured authoritative outcome. Callers
	// assert the outcome instead of treating an expected conflict as transport
	// failure.
	return result, nil
}

func digest(value []byte) string {
	hash := sha256.Sum256(value)
	return hex.EncodeToString(hash[:])
}

// seedServiceAccountIdentity maps the matrix OIDC service account to the
// integration tenant so the exercise /evidence stage passes authorization.
// The issuer/subject are read from a freshly minted token (same env contract
// as exercise), so this works against any RS256 provider. OIDC env is
// mandatory: exercise cannot pass /evidence without it, so seed fails fast
// instead of leaving a guaranteed-later failure.
func seedServiceAccountIdentity(db *sql.DB, tenantID string) error {
	tokenEndpoint := strings.TrimSpace(os.Getenv("INTEGIN_OIDC_TOKEN_ENDPOINT"))
	if tokenEndpoint == "" {
		if issuer := strings.TrimSpace(os.Getenv("INTEGIN_OIDC_ISSUER")); issuer != "" {
			tokenEndpoint = strings.TrimSuffix(issuer, "/") + "/protocol/openid-connect/token"
		}
	}
	clientID := strings.TrimSpace(os.Getenv("INTEGIN_OIDC_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("INTEGIN_OIDC_CLIENT_SECRET"))
	if tokenEndpoint == "" || clientID == "" || clientSecret == "" {
		return errors.New("seed identity requires INTEGIN_OIDC_TOKEN_ENDPOINT (or INTEGIN_OIDC_ISSUER) + INTEGIN_OIDC_CLIENT_ID + INTEGIN_OIDC_CLIENT_SECRET")
	}
	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {clientID}, "client_secret": {clientSecret}}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("seed identity token request: %w", err)
	}
	defer resp.Body.Close()
	var tokenDoc struct {
		AccessToken string `json:"access_token"`
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || json.Unmarshal(body, &tokenDoc) != nil || tokenDoc.AccessToken == "" {
		return fmt.Errorf("seed identity token request: status %d", resp.StatusCode)
	}
	parts := strings.Split(tokenDoc.AccessToken, ".")
	if len(parts) != 3 {
		return errors.New("seed identity: token is not a JWT")
	}
	claimsRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("seed identity: decode claims: %w", err)
	}
	var claims struct {
		Issuer  string `json:"iss"`
		Subject string `json:"sub"`
	}
	if err := json.Unmarshal(claimsRaw, &claims); err != nil || claims.Issuer == "" || claims.Subject == "" {
		return errors.New("seed identity: token has no iss/sub")
	}
	// The actor is scoped per tenant: actor_id is globally unique while the
	// FK binds (actor_id, tenant_id, organization_id), so reusing one actor
	// across tenants would rebind it and strand other memberships.
	actorID := integrationActorID + "-" + tenantID
	if _, err := identity.GrantMembership(ctx, db, identity.GrantInput{
		Issuer: claims.Issuer, Subject: claims.Subject,
		TenantID: tenantID, OrganizationID: integrationOrganization,
		ActorID: actorID, Role: "administrator",
		Capabilities: []string{workorderauth.CapabilityAddEvidenceReference},
	}); err != nil {
		return fmt.Errorf("seed identity grant: %w", err)
	}
	fmt.Printf("seeded identity iss=%s sub=%s actor=%s\n", claims.Issuer, claims.Subject, actorID)
	return nil
}

func localConfig() (databaseURL, tenantID, secret, fixturePath string, err error) {
	databaseURL = strings.TrimSpace(os.Getenv("INTEGIN_DB_URL"))
	tenantID = strings.TrimSpace(os.Getenv("INTEGIN_TENANT_ID"))
	secret = strings.TrimSpace(os.Getenv("INTEGIN_SYNC_SECRET"))
	fixturePath = strings.TrimSpace(os.Getenv("INTEGIN_LIVE_FIXTURE_FILE"))
	if fixturePath == "" {
		return "", "", "", "", errors.New("INTEGIN_LIVE_FIXTURE_FILE is required and must be outside source control")
	}
	if databaseURL == "" || tenantID == "" || secret == "" {
		return "", "", "", "", errors.New("INTEGIN_DB_URL, INTEGIN_TENANT_ID, and INTEGIN_SYNC_SECRET are required")
	}
	return databaseURL, tenantID, secret, fixturePath, nil
}
