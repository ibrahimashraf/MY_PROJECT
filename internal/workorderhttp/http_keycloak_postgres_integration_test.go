package workorderhttp_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"integin/internal/domain/workorder"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/server"
	"integin/internal/workorderauth"
	"integin/internal/workorderhttp"
	"integin/internal/workorderpg"
)

func TestKeycloakPartialSubmissionHTTPPostgresIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("INTEGIN_TEST_DATABASE_URL"))
	issuer := strings.TrimSpace(os.Getenv("INTEGIN_KEYCLOAK_TEST_ISSUER"))
	clientID := strings.TrimSpace(os.Getenv("INTEGIN_KEYCLOAK_TEST_CLIENT_ID"))
	username := strings.TrimSpace(os.Getenv("INTEGIN_KEYCLOAK_TEST_USERNAME"))
	password := os.Getenv("INTEGIN_KEYCLOAK_TEST_PASSWORD")
	if dsn == "" || issuer == "" || clientID == "" || username == "" || password == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL and all INTEGIN_KEYCLOAK_TEST_* values to run the disposable Keycloak integration test")
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
	identityFixtureDatabase := openKeycloakIdentityFixtureDatabase(t, ctx, fixtureDatabase)

	token := keycloakPasswordGrant(t, ctx, issuer, clientID, username, password)
	validator, err := oidcauth.NewValidator(ctx, oidcauth.Config{
		Enabled: true, Issuer: issuer, Audience: clientID, AuthorizedParty: clientID, ClockSkew: 5 * time.Second,
		MaxTokenAge: time.Hour, JWKSRefresh: time.Minute, AllowInsecureLoopbackIssuer: true,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := validator.Validate(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if principal.Issuer != issuer || principal.Subject == "" {
		t.Fatalf("unexpected Keycloak principal: %+v", principal)
	}

	id := fmt.Sprintf("it-keycloak-%d", time.Now().UnixNano())
	tenantID, organizationID, actorID := "pilot-tenant-keycloak", "pilot-organization-keycloak", "inspector-"+id
	workOrderID, scopeID, assignmentID := id+"-order", id+"-scope", id+"-assignment"
	inspectionIDs := []string{id + "-inspection-1", id + "-inspection-2"}
	insertRuntimeMembership(t, ctx, identityFixtureDatabase, issuer, principal.Subject, tenantID, organizationID, actorID)
	t.Cleanup(func() { cleanupRuntimeMembership(t, ctx, identityFixtureDatabase, issuer, principal.Subject, actorID) })
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
	body := map[string]any{
		"operation_id": id + "-operation", "idempotency_key": id + "-idempotency", "expected_revision": 1,
		"work_order_id": workOrderID, "assignment_id": assignmentID, "inspection_ids": inspectionIDs,
	}
	tampered := token[:len(token)-10] + "x" + token[len(token)-9:]
	if tampered == token {
		tampered = token[:len(token)-10] + "y" + token[len(token)-9:]
	}
	if rejected := postPartialSubmission(t, runtime.URL, tampered, body); rejected.Code != http.StatusUnauthorized {
		t.Fatalf("tampered Keycloak token status = %d body=%s, want %d", rejected.Code, rejected.Body.String(), http.StatusUnauthorized)
	}
	accepted := postPartialSubmission(t, runtime.URL, token, body)
	if accepted.Code != http.StatusOK {
		t.Fatalf("Keycloak submission status = %d body=%s", accepted.Code, accepted.Body.String())
	}
	assertHTTPSubmissionPersistence(t, ctx, fixtureDatabase, workOrderID, inspectionIDs)

	cleanupHTTPWorkOrderFixture(t, ctx, database, tenantID, organizationID, workOrderID, assignmentID)
	cleanupRuntimeMembership(t, ctx, identityFixtureDatabase, issuer, principal.Subject, actorID)
	assertHTTPRuntimeFixtureCleanup(t, ctx, fixtureDatabase, identityFixtureDatabase, workOrderID, issuer, principal.Subject, actorID, issuer+"-unused", principal.Subject+"-unused", "unused-actor")
}

const keycloakIdentityFixtureRole = "integin_keycloak_identity_fixture"

func openKeycloakIdentityFixtureDatabase(t *testing.T, ctx context.Context, adminDatabase *sql.DB) *sql.DB {
	t.Helper()
	if _, err := adminDatabase.ExecContext(ctx, "DROP ROLE IF EXISTS "+keycloakIdentityFixtureRole); err != nil {
		if _, ownedErr := adminDatabase.ExecContext(ctx, "DROP OWNED BY "+keycloakIdentityFixtureRole); ownedErr == nil {
			if _, retryErr := adminDatabase.ExecContext(ctx, "DROP ROLE IF EXISTS "+keycloakIdentityFixtureRole); retryErr != nil {
				t.Fatalf("reset disposable identity fixture role: %v", retryErr)
			}
		} else {
			t.Fatalf("reset disposable identity fixture role: %v", err)
		}
	}
	password := randomKeycloakIdentityFixturePassword(t)
	if _, err := adminDatabase.ExecContext(ctx, "CREATE ROLE "+keycloakIdentityFixtureRole+" LOGIN; ALTER ROLE "+keycloakIdentityFixtureRole+" PASSWORD '"+password+"'; GRANT USAGE ON SCHEMA public TO "+keycloakIdentityFixtureRole+"; GRANT SELECT, INSERT, DELETE ON identity_actor, identity_subject, identity_membership, identity_membership_capability TO "+keycloakIdentityFixtureRole+"; GRANT USAGE ON SEQUENCE identity_subject_subject_id_seq, identity_membership_membership_id_seq TO "+keycloakIdentityFixtureRole); err != nil {
		t.Fatalf("configure disposable identity fixture role: %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminDatabase.ExecContext(context.Background(), "DROP OWNED BY "+keycloakIdentityFixtureRole)
		_, _ = adminDatabase.ExecContext(context.Background(), "DROP ROLE IF EXISTS "+keycloakIdentityFixtureRole)
	})
	identityFixtureDatabase, err := sql.Open("postgres", keycloakIdentityFixtureDSN(t, os.Getenv("INTEGIN_TEST_FIXTURE_DATABASE_URL"), password))
	if err != nil {
		t.Fatalf("open disposable identity fixture database: %v", err)
	}
	t.Cleanup(func() { _ = identityFixtureDatabase.Close() })
	if err := identityFixtureDatabase.PingContext(ctx); err != nil {
		t.Fatalf("ping disposable identity fixture database: %v", err)
	}
	return identityFixtureDatabase
}

func keycloakIdentityFixtureDSN(t *testing.T, dsn, password string) string {
	t.Helper()
	parsed, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		t.Fatal("INTEGIN_TEST_FIXTURE_DATABASE_URL must be a PostgreSQL URL for disposable identity fixture setup")
	}
	parsed.User = url.UserPassword(keycloakIdentityFixtureRole, password)
	return parsed.String()
}

func randomKeycloakIdentityFixturePassword(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("generate disposable identity fixture role password: %v", err)
	}
	return hex.EncodeToString(raw)
}

func keycloakPasswordGrant(t *testing.T, ctx context.Context, issuer, clientID, username, password string) string {
	t.Helper()
	values := url.Values{
		"grant_type": {"password"}, "client_id": {clientID}, "username": {username}, "password": {password},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(issuer, "/")+"/protocol/openid-connect/token", strings.NewReader(values.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		var rejected struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		_ = json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&rejected)
		t.Fatalf("Keycloak token status = %d error=%q description=%q", response.StatusCode, rejected.Error, rejected.ErrorDescription)
	}
	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.AccessToken == "" {
		t.Fatal("Keycloak token response omitted access_token")
	}
	return payload.AccessToken
}
