package workorderhttp_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"

	"integin/internal/domain/workorder"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/server"
	"integin/internal/workorderauth"
	"integin/internal/workorderhttp"
	"integin/internal/workorderpg"
)

func TestSignedOIDCPartialSubmissionHTTPPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL to run the controlled signed-OIDC PostgreSQL integration test")
	}
	ctx := context.Background()
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := database.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	fixtureDatabase := openHTTPFixtureDatabase(t, ctx)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const keyID = "controlled-oidc-rs256-key"
	var issuerServer *httptest.Server
	issuerServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{"issuer": issuerServer.URL, "jwks_uri": issuerServer.URL + "/keys"})
		case "/keys":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
				"kty": "RSA", "kid": keyID, "alg": "RS256", "use": "sig",
				"n": base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
				"e": base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
			}}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(issuerServer.Close)

	validator, err := oidcauth.NewValidator(ctx, oidcauth.Config{
		Enabled: true, Issuer: issuerServer.URL, Audience: "integin-controlled-workorder", AuthorizedParty: "integin-controlled-client",
		RequiredAMR: []string{"pwd"}, ClockSkew: time.Second, MaxTokenAge: 15 * time.Minute, JWKSRefresh: time.Minute,
		AllowInsecureLoopbackIssuer: true,
	}, issuerServer.Client())
	if err != nil {
		t.Fatal(err)
	}

	id := fmt.Sprintf("it-oidc-http-%d", time.Now().UnixNano())
	subject := "subject-" + id
	tenantID, organizationID, actorID := "pilot-tenant-oidc-http", "pilot-organization-oidc-http", "inspector-"+id
	workOrderID, scopeID, assignmentID := id+"-order", id+"-scope", id+"-assignment"
	inspectionIDs := []string{id + "-inspection-1", id + "-inspection-2"}
	insertRuntimeMembership(t, ctx, fixtureDatabase, issuerServer.URL, subject, tenantID, organizationID, actorID)
	t.Cleanup(func() { cleanupRuntimeMembership(t, ctx, fixtureDatabase, issuerServer.URL, subject, actorID) })
	setupHTTPWorkOrderFixture(t, ctx, database, workorder.ActorContext{TenantID: tenantID, OrganizationID: organizationID, ActorID: actorID, Role: "inspector"}, workOrderID, scopeID, assignmentID, inspectionIDs)
	t.Cleanup(func() {
		cleanupHTTPWorkOrderFixture(t, ctx, database, tenantID, organizationID, workOrderID, assignmentID)
	})

	repository, err := workorderpg.NewRepository(database, workorderpg.NewPostgresInspectionMembershipValidator())
	if err != nil {
		t.Fatal(err)
	}
	service, err := workorder.NewService(workorder.ServiceDependencies{Repository: repository, Transactions: repository, Authorizer: workorderauth.New()})
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := identity.NewPostgresResolver(database)
	if err != nil {
		t.Fatal(err)
	}
	runtime := httptest.NewServer(server.NewMux(server.Dependencies{WorkOrderHandler: workorderhttp.Handler{Validator: validator, Resolver: resolver, Service: service}}))
	t.Cleanup(runtime.Close)

	token := controlledOIDCToken(t, privateKey, keyID, issuerServer.URL, subject)
	body := map[string]any{
		"operation_id": id + "-operation", "idempotency_key": id + "-idempotency", "expected_revision": 1,
		"work_order_id": workOrderID, "assignment_id": assignmentID, "inspection_ids": inspectionIDs,
	}
	signatureOffset := strings.LastIndexByte(token, '.') + 1
	if signatureOffset <= 0 || signatureOffset >= len(token) {
		t.Fatal("signed OIDC token has no signature segment")
	}
	signatureIndex := signatureOffset + (len(token)-signatureOffset)/2
	replacement := byte('A')
	if token[signatureIndex] == replacement {
		replacement = 'B'
	}
	invalid := token[:signatureIndex] + string(replacement) + token[signatureIndex+1:]
	if invalidResponse := postPartialSubmission(t, runtime.URL, invalid, body); invalidResponse.Code != http.StatusUnauthorized {
		t.Fatalf("signature-invalid status = %d body=%s, want %d", invalidResponse.Code, invalidResponse.Body.String(), http.StatusUnauthorized)
	}
	if expiredResponse := postPartialSubmission(t, runtime.URL, controlledExpiredOIDCToken(t, privateKey, keyID, issuerServer.URL, subject), body); expiredResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expired-token status = %d body=%s, want %d", expiredResponse.Code, expiredResponse.Body.String(), http.StatusUnauthorized)
	}
	validResponse := postPartialSubmission(t, runtime.URL, token, body)
	if validResponse.Code != http.StatusOK {
		t.Fatalf("signed OIDC submission status = %d body=%s", validResponse.Code, validResponse.Body.String())
	}
	var receipt workorder.MutationReceipt
	if err := json.NewDecoder(validResponse.Body).Decode(&receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.WorkOrderID != workOrderID || receipt.TenantID != tenantID || receipt.Revision != 2 {
		t.Fatalf("unexpected signed OIDC receipt: %+v", receipt)
	}
	assertHTTPSubmissionPersistence(t, ctx, fixtureDatabase, workOrderID, inspectionIDs)

	cleanupHTTPWorkOrderFixture(t, ctx, database, tenantID, organizationID, workOrderID, assignmentID)
	cleanupRuntimeMembership(t, ctx, fixtureDatabase, issuerServer.URL, subject, actorID)
	assertHTTPRuntimeFixtureCleanup(t, ctx, fixtureDatabase, fixtureDatabase, workOrderID, issuerServer.URL, subject, actorID, issuerServer.URL+"-unused", subject+"-unused", "unused-actor")
}

func controlledOIDCToken(t *testing.T, privateKey *rsa.PrivateKey, keyID, issuer, subject string) string {
	t.Helper()
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": issuer, "sub": subject, "aud": []string{"integin-controlled-workorder"}, "azp": "integin-controlled-client",
		"amr": []string{"pwd"}, "exp": now.Add(5 * time.Minute).Unix(), "iat": now.Add(-time.Minute).Unix(), "nbf": now.Add(-time.Minute).Unix(),
	})
	token.Header["kid"] = keyID
	token.Header["typ"] = "Bearer"
	raw, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func controlledExpiredOIDCToken(t *testing.T, privateKey *rsa.PrivateKey, keyID, issuer, subject string) string {
	t.Helper()
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": issuer, "sub": subject, "aud": []string{"integin-controlled-workorder"}, "azp": "integin-controlled-client",
		"amr": []string{"pwd"}, "exp": now.Add(-2 * time.Minute).Unix(), "iat": now.Add(-5 * time.Minute).Unix(), "nbf": now.Add(-5 * time.Minute).Unix(),
	})
	token.Header["kid"] = keyID
	token.Header["typ"] = "Bearer"
	raw, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
