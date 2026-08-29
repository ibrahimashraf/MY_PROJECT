package certificatehttp_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"

	"integin/internal/certificatehttp"
	"integin/internal/certificatepg"
	"integin/internal/domain/certificateauthority"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/server"
)

type runtimeLifecycle struct {
	actor certificateauthority.ActorContext
}

func (l *runtimeLifecycle) CreateDraft(context.Context, certificateauthority.ActorContext, certificateauthority.CreateDraftRequest, time.Time) (*certificateauthority.Certificate, error) {
	return nil, errors.New("not exercised")
}
func (l *runtimeLifecycle) Submit(_ context.Context, a certificateauthority.ActorContext, _ string, _ time.Time) error {
	l.actor = a
	if !a.Capabilities["certificate.prepare"] {
		return errors.New("capability absent")
	}
	return nil
}
func (l *runtimeLifecycle) Review(context.Context, certificateauthority.ActorContext, string, time.Time) error {
	return nil
}
func (l *runtimeLifecycle) Sign(context.Context, certificateauthority.ActorContext, string, time.Time) error {
	return nil
}
func (l *runtimeLifecycle) Issue(context.Context, certificateauthority.ActorContext, string, time.Time) (certificatepg.IssueResult, error) {
	return certificatepg.IssueResult{}, nil
}
func (l *runtimeLifecycle) Revoke(context.Context, certificateauthority.ActorContext, string, string, time.Time) error {
	return nil
}
func (l *runtimeLifecycle) Supersede(context.Context, certificateauthority.ActorContext, string, string, time.Time) error {
	return nil
}
func (l *runtimeLifecycle) Expire(context.Context, certificateauthority.ActorContext, string, time.Time) error {
	return nil
}

func TestSignedOIDCCertificateTransportUsesLocalMembership(t *testing.T) {
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL")
	}
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err = db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const kid = "certificate-runtime-rs256"
	var issuer *httptest.Server
	issuer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{"issuer": issuer.URL, "jwks_uri": issuer.URL + "/keys"})
		case "/keys":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{"kty": "RSA", "kid": kid, "alg": "RS256", "use": "sig", "n": base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1})}}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(issuer.Close)
	validator, err := oidcauth.NewValidator(ctx, oidcauth.Config{Enabled: true, Issuer: issuer.URL, Audience: "integin-certificate-runtime", AuthorizedParty: "integin-certificate-client", RequiredAMR: []string{"pwd"}, ClockSkew: time.Second, MaxTokenAge: 15 * time.Minute, JWKSRefresh: time.Minute, AllowInsecureLoopbackIssuer: true}, issuer.Client())
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := identity.NewPostgresResolver(db)
	if err != nil {
		t.Fatal(err)
	}
	id := fmt.Sprintf("it-certificate-http-%d", time.Now().UnixNano())
	subject := "subject-" + id
	tenant := "it-certificate-http-tenant-" + id
	organization := "org-a"
	actor := "inspector-" + id
	insertCertificateMembership(t, ctx, db, issuer.URL, subject, tenant, organization, actor, []string{"certificate.prepare"})
	t.Cleanup(func() { cleanupCertificateMembership(t, ctx, db, issuer.URL, subject) })
	lifecycle := &runtimeLifecycle{}
	runtime := httptest.NewServer(server.NewMux(server.Dependencies{CertificateHandler: certificatehttp.Handler{Validator: validator, Actors: certificatehttp.LocalActorResolver{Memberships: resolver}, Lifecycle: lifecycle}}))
	t.Cleanup(runtime.Close)
	valid := certificateToken(t, key, kid, issuer.URL, subject)
	invalid := valid[:len(valid)-1] + "x"
	if valid[len(valid)-1] == 120 {
		invalid = valid[:len(valid)-1] + "y"
	}
	if status := certificatePost(t, runtime.URL, invalid); status != http.StatusUnauthorized {
		t.Fatalf("tampered status=%d", status)
	}
	if status := certificatePost(t, runtime.URL, certificateToken(t, key, kid, issuer.URL, "unknown-"+id)); status != http.StatusForbidden {
		t.Fatalf("unknown membership status=%d", status)
	}
	if status := certificatePost(t, runtime.URL, valid); status != http.StatusNoContent {
		t.Fatalf("valid status=%d", status)
	}
	if lifecycle.actor.TenantID != tenant || lifecycle.actor.OrganizationID != organization || lifecycle.actor.ActorID != actor || !lifecycle.actor.Capabilities["certificate.prepare"] {
		t.Fatalf("server-derived actor=%#v", lifecycle.actor)
	}
	deniedSubject := "denied-" + id
	insertCertificateMembership(t, ctx, db, issuer.URL, deniedSubject, tenant, organization, "denied-"+id, nil)
	t.Cleanup(func() { cleanupCertificateMembership(t, ctx, db, issuer.URL, deniedSubject) })
	if status := certificatePost(t, runtime.URL, certificateToken(t, key, kid, issuer.URL, deniedSubject)); status != http.StatusForbidden {
		t.Fatalf("missing capability status=%d", status)
	}
}
func certificatePost(t *testing.T, base, token string) int {
	t.Helper()
	r, err := http.NewRequest(http.MethodPost, base+"/certificates/certificate-a/submit", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("cache=%q", response.Header.Get("Cache-Control"))
	}
	return response.StatusCode
}
func certificateToken(t *testing.T, key *rsa.PrivateKey, kid, issuer, subject string) string {
	t.Helper()
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"iss": issuer, "sub": subject, "aud": []string{"integin-certificate-runtime"}, "azp": "integin-certificate-client", "amr": []string{"pwd"}, "exp": now.Add(5 * time.Minute).Unix(), "iat": now.Add(-time.Minute).Unix(), "nbf": now.Add(-time.Minute).Unix()})
	token.Header["kid"] = kid
	raw, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func insertCertificateMembership(t *testing.T, ctx context.Context, db *sql.DB, issuer, subject, tenant, organization, actor string, caps []string) {
	t.Helper()
	var sid, mid int64
	if err := db.QueryRowContext(ctx, "INSERT INTO identity_subject (issuer,subject) VALUES ($1,$2) RETURNING subject_id", issuer, subject).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "INSERT INTO identity_membership (subject_id,tenant_id,organization_id,actor_id,work_order_role) VALUES ($1,$2,$3,$4,$5) RETURNING membership_id", sid, tenant, organization, actor, "inspector").Scan(&mid); err != nil {
		t.Fatal(err)
	}
	for _, cap := range caps {
		if _, err := db.ExecContext(ctx, "INSERT INTO identity_membership_capability (membership_id,capability) VALUES ($1,$2)", mid, cap); err != nil {
			t.Fatal(err)
		}
	}
}
func cleanupCertificateMembership(t *testing.T, ctx context.Context, db *sql.DB, issuer, subject string) {
	t.Helper()
	statements := []string{
		"DELETE FROM identity_membership_capability WHERE membership_id IN (SELECT m.membership_id FROM identity_membership m JOIN identity_subject s ON s.subject_id=m.subject_id WHERE s.issuer=$1 AND s.subject=$2)",
		"DELETE FROM identity_membership WHERE subject_id IN (SELECT subject_id FROM identity_subject WHERE issuer=$1 AND subject=$2)",
		"DELETE FROM identity_subject WHERE issuer=$1 AND subject=$2",
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement, issuer, subject); err != nil {
			t.Errorf("identity cleanup: %v", err)
		}
	}
}
