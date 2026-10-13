package workorderhttp_test

import (
	"context"
	"database/sql"
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

	_ "github.com/jackc/pgx/v5/stdlib"

	"integin/internal/domain/workorder"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/server"
	"integin/internal/workorderauth"
	"integin/internal/workorderhttp"
	"integin/internal/workorderpg"
)

func TestCasdoorPartialSubmissionHTTPPostgresIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("INTEGIN_TEST_DATABASE_URL"))
	issuer := strings.TrimSpace(os.Getenv("INTEGIN_CASDOOR_TEST_ISSUER"))
	clientID := strings.TrimSpace(os.Getenv("INTEGIN_CASDOOR_TEST_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("INTEGIN_CASDOOR_TEST_CLIENT_SECRET"))
	username := strings.TrimSpace(os.Getenv("INTEGIN_CASDOOR_TEST_USERNAME"))
	password := os.Getenv("INTEGIN_CASDOOR_TEST_PASSWORD")
	if dsn == "" || issuer == "" || clientID == "" || clientSecret == "" || username == "" || password == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL and all INTEGIN_CASDOOR_TEST_* values to run the disposable Casdoor integration test")
	}
	ctx := context.Background()
	database, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(5)
	t.Cleanup(func() { _ = database.Close() })
	if err := database.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	fixtureDatabase := openHTTPFixtureDatabase(t, ctx)

	token := casdoorPasswordGrant(t, ctx, issuer, clientID, clientSecret, username, password)
	validator, err := oidcauth.NewValidator(ctx, oidcauth.Config{
		Enabled: true, Issuer: issuer, Audience: clientID, AuthorizedParty: "",
		ClockSkew:   5 * time.Second,
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
		t.Fatalf("unexpected Casdoor principal: %+v", principal)
	}

	id := fmt.Sprintf("it-casdoor-%d", time.Now().UnixNano())
	tenantID, organizationID, actorID := "pilot-tenant-casdoor", "pilot-organization-casdoor", "inspector-"+id
	workOrderID, scopeID, assignmentID := id+"-order", id+"-scope", id+"-assignment"
	inspectionIDs := []string{id + "-inspection-1", id + "-inspection-2"}
	insertRuntimeMembership(t, ctx, fixtureDatabase, issuer, principal.Subject, tenantID, organizationID, actorID)
	t.Cleanup(func() { cleanupRuntimeMembership(t, ctx, fixtureDatabase, issuer, principal.Subject, actorID) })
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
	tampered := token[:len(token)-1] + "x"
	if token[len(token)-1] == 'x' {
		tampered = token[:len(token)-1] + "y"
	}
	if rejected := postPartialSubmission(t, runtime.URL, tampered, body); rejected.Code != http.StatusUnauthorized {
		t.Fatalf("tampered Casdoor token status = %d body=%s, want %d", rejected.Code, rejected.Body.String(), http.StatusUnauthorized)
	}
	accepted := postPartialSubmission(t, runtime.URL, token, body)
	if accepted.Code != http.StatusOK {
		t.Fatalf("Casdoor submission status = %d body=%s", accepted.Code, accepted.Body.String())
	}
	assertHTTPSubmissionPersistence(t, ctx, database, tenantID, organizationID, workOrderID, inspectionIDs)

	cleanupHTTPWorkOrderFixture(t, ctx, database, tenantID, organizationID, workOrderID, assignmentID)
	cleanupRuntimeMembership(t, ctx, fixtureDatabase, issuer, principal.Subject, actorID)
	assertHTTPRuntimeFixtureCleanup(t, ctx, database, fixtureDatabase, tenantID, organizationID, workOrderID, issuer, principal.Subject, actorID, issuer+"-unused", principal.Subject+"-unused", "unused-actor")
}

func casdoorPasswordGrant(t *testing.T, ctx context.Context, issuer, clientID, clientSecret, username, password string) string {
	t.Helper()
	values := url.Values{
		"grant_type": {"password"}, "client_id": {clientID}, "client_secret": {clientSecret},
		"username": {username}, "password": {password},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(issuer, "/")+"/api/login/oauth/access_token", strings.NewReader(values.Encode()))
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
		t.Fatalf("Casdoor token status = %d error=%q description=%q", response.StatusCode, rejected.Error, rejected.ErrorDescription)
	}
	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.AccessToken == "" {
		t.Fatal("Casdoor token response omitted access_token")
	}
	return payload.AccessToken
}
