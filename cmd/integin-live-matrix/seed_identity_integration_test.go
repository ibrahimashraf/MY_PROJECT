package main

import (
	"context"
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

	_ "github.com/jackc/pgx/v5/stdlib"

	"integin/internal/identity"
	"integin/internal/workorderauth"
)

// TestSeedServiceAccountIdentityResolves gates on INTEGIN_TEST_DATABASE_URL and
// proves the seed's identity rows satisfy the exact production contract:
// integin_resolve_identity_membership returns the membership AND
// workorderauth.ActorFromMembership accepts it (non-empty capabilities).
// Without this, seed/exercise drift from the resolver goes unnoticed until a
// live run fails with authorization_failed.
func TestSeedServiceAccountIdentityRequiresOIDCEnv(t *testing.T) {
	t.Setenv("INTEGIN_OIDC_TOKEN_ENDPOINT", "")
	t.Setenv("INTEGIN_OIDC_ISSUER", "")
	t.Setenv("INTEGIN_OIDC_CLIENT_ID", "")
	t.Setenv("INTEGIN_OIDC_CLIENT_SECRET", "")
	// nil DB: must fail on missing env before any database use.
	if err := seedServiceAccountIdentity(nil, "tenant-x"); err == nil {
		t.Fatal("expected error without OIDC env, got nil")
	}
}

func TestSeedServiceAccountIdentityResolves(t *testing.T) {
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL to run seed identity integration test")
	}
	const (
		testIssuer  = "https://idp.example.test/realms/matrix"
		testSubject = "service-account-matrix-test"
		testTenant  = "matrix-identity-test-tenant"
	)
	claims, _ := json.Marshal(map[string]string{"iss": testIssuer, "sub": testSubject})
	fakeJWT := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`)) + "." +
		base64.RawURLEncoding.EncodeToString(claims) + ".sig"
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": fakeJWT, "token_type": "Bearer", "expires_in": 600})
	}))
	defer stub.Close()

	t.Setenv("INTEGIN_OIDC_TOKEN_ENDPOINT", stub.URL)
	t.Setenv("INTEGIN_OIDC_CLIENT_ID", "matrix")
	t.Setenv("INTEGIN_OIDC_CLIENT_SECRET", "secret")

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Fatalf("ping database: %v", err)
	}

	if err := seedServiceAccountIdentity(db, testTenant); err != nil {
		db.Close()
		t.Fatalf("seedServiceAccountIdentity: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		for _, q := range []string{
			`DELETE FROM identity_membership_capability WHERE membership_id IN (SELECT membership_id FROM identity_membership WHERE tenant_id=$1)`,
			`DELETE FROM identity_membership WHERE tenant_id=$1`,
			`DELETE FROM identity_subject WHERE issuer=$1 AND subject=$2`,
			`DELETE FROM identity_actor WHERE actor_id=$1 AND tenant_id=$2`,
		} {
			args := []any{testTenant}
			if strings.Contains(q, "issuer") {
				args = []any{testIssuer, testSubject}
			} else if strings.Contains(q, "actor_id") {
				args = []any{integrationActorID + "-" + testTenant, testTenant}
			}
			if _, err := db.ExecContext(ctx, q, args...); err != nil {
				t.Errorf("cleanup: %v", err)
			}
		}
		db.Close()
	})

	resolver, err := identity.NewPostgresResolver(db)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	membership, err := resolver.Resolve(ctx, identity.PrincipalKey{Issuer: testIssuer, Subject: testSubject})
	if err != nil {
		t.Fatalf("resolve seeded identity: %v", err)
	}
	if membership.TenantID != testTenant || membership.OrganizationID != integrationOrganization {
		t.Fatalf("membership scope = %s/%s, want %s/%s", membership.TenantID, membership.OrganizationID, testTenant, integrationOrganization)
	}
	actor, err := workorderauth.ActorFromMembership(membership)
	if err != nil {
		t.Fatalf("ActorFromMembership rejects seeded membership: %v", err)
	}
	if actor.ActorID != integrationActorID+"-"+testTenant || len(actor.Capabilities) == 0 {
		t.Fatalf("derived actor = %+v, want id %s with capabilities", actor, integrationActorID+"-"+testTenant)
	}
	fmt.Println("seed identity resolves and projects")
}
