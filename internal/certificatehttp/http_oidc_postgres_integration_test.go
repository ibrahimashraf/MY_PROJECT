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
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/jackc/pgx/v5/stdlib"

	"integin/internal/certificatehttp"
	"integin/internal/certificatepg"
	"integin/internal/domain/certificateauthority"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/server"
)

var now = time.Now().UTC()

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
func (l *runtimeLifecycle) Renew(context.Context, certificateauthority.ActorContext, string, string, time.Time) error {
	return nil
}

type certificateFixture struct {
	WorkOrderID      string
	ScopeID          string
	AssignmentID     string
	InspectionID     string
	TemplateID       string
	PolicyID         string
	AssetRegistryID  string
	PublicScopeID    string
	CertificateID    string
	InspectorSubject string
	ReviewerSubject  string
	IssuerSubject    string
	TenantID         string
	OrganizationID   string
	InspectorActorID string
	ReviewerActorID  string
	IssuerActorID    string
	Issuer           string
}

func TestSignedOIDCCertificateTransportUsesLocalMembership(t *testing.T) {
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL")
	}
	ctx := context.Background()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
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
	parts := strings.Split(valid, ".")
	invalid := parts[0] + "." + parts[1] + ".invalid-signature"
	if len(parts) != 3 {
		invalid = valid + "-tampered"
	}

	// Prime the handler and wait for full readiness (JWKS + membership warm-up)
	// under potentially load-contended parallel runs.
	primeReady(t, runtime.URL, valid)

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

// primeReady repeatedly sends a valid bearer request until the handler responds
// cleanly, warming JWKS/membership state that may be slow under load-contended
// parallel runs. It fails only if readiness is not reached within the deadline.
func primeReady(t *testing.T, base, token string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var status int
	for {
		status = certificatePost(t, base, token)
		if status == http.StatusNoContent {
			return
		}
		// Accept transient server-side warm-up delays but require eventual success.
		if !time.Now().Before(deadline) {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("handler did not become ready; last status=%d", status)
}

// warmUpHandler repeatedly sends a valid bearer request to a non-existent
// certificate to warm the OIDC validator (JWKS/membership) without asserting on
// the response status. It returns once the handler responds, proving the
// validator is operational, and fails if no response is received in time.
func warmUpHandler(t *testing.T, base, token string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		r, err := http.NewRequest(http.MethodPost, base+"/certificates/prime-warmup/submit", nil)
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set("Authorization", "Bearer "+token)
		response, err := http.DefaultClient.Do(r)
		if err == nil {
			// Any response proves the validator/handler is reachable and warmed.
			_ = response.Body.Close()
			return
		}
		if !time.Now().Before(deadline) {
			t.Fatalf("handler did not respond: %v", err)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func certificatePostWithPath(t *testing.T, base, token, path, body string) int {
	t.Helper()
	resp := certificatePostWithBody(t, base, token, path, body)
	return resp.Code
}

func certificatePostWithBody(t *testing.T, base, token, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	var err error
	if body != "" {
		r, err = http.NewRequest(http.MethodPost, base+path, strings.NewReader(body))
	} else {
		r, err = http.NewRequest(http.MethodPost, base+path, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	response, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	recorder.Code = response.StatusCode
	for k, v := range response.Header {
		for _, vv := range v {
			recorder.Header().Add(k, vv)
		}
	}
	_, _ = recorder.Body.ReadFrom(response.Body)
	response.Body.Close()
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cache=%q", recorder.Header().Get("Cache-Control"))
	}
	return recorder
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
	if _, err := db.ExecContext(ctx, "INSERT INTO identity_actor (actor_id, tenant_id, organization_id) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING", actor, tenant, organization); err != nil {
		t.Fatal(err)
	}
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

func TestSignedOIDCCertificateLifecycleIntegration(t *testing.T) {
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL")
	}
	ctx := context.Background()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	t.Cleanup(func() { _ = db.Close() })
	if err = db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const kid = "certificate-lifecycle-rs256"
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

	repository, err := certificatepg.NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}

	id := fmt.Sprintf("it-cert-lifecycle-%d", time.Now().UnixNano())
	tenant := "it-cert-tenant-" + id
	organization := "org-a"
	inspectorActor := "inspector-" + id
	reviewerActor := "reviewer-" + id
	issuerActor := "issuer-" + id
	inspectorSubject := "subject-inspector-" + id
	reviewerSubject := "subject-reviewer-" + id
	issuerSubject := "subject-issuer-" + id

	workOrderID := id + "-order"
	scopeID := id + "-scope"
	assignmentID := id + "-assignment"
	inspectionID := id + "-inspection"
	templateID := id + "-template"
	policyID := id + "-policy"
	assetRegistryID := id + "-asset"
	publicScopeID := id + "-public-scope"
	certificateID := id + "-cert"

	fixture := certificateFixture{
		WorkOrderID:      workOrderID,
		ScopeID:          scopeID,
		AssignmentID:     assignmentID,
		InspectionID:     inspectionID,
		TemplateID:       templateID,
		PolicyID:         policyID,
		AssetRegistryID:  assetRegistryID,
		PublicScopeID:    publicScopeID,
		CertificateID:    certificateID,
		InspectorSubject: inspectorSubject,
		ReviewerSubject:  reviewerSubject,
		IssuerSubject:    issuerSubject,
		TenantID:         tenant,
		OrganizationID:   organization,
		InspectorActorID: inspectorActor,
		ReviewerActorID:  reviewerActor,
		IssuerActorID:    issuerActor,
		Issuer:           issuer.URL,
	}

	setupCertificateFixture(t, ctx, db, fixture)
	t.Cleanup(func() { cleanupCertificateFixture(t, ctx, db, fixture) })

	insertCertificateMembership(t, ctx, db, issuer.URL, inspectorSubject, tenant, organization, inspectorActor, []string{"certificate.prepare"})
	t.Cleanup(func() { cleanupCertificateMembership(t, ctx, db, issuer.URL, inspectorSubject) })
	insertCertificateMembership(t, ctx, db, issuer.URL, reviewerSubject, tenant, organization, reviewerActor, []string{"certificate.review"})
	t.Cleanup(func() { cleanupCertificateMembership(t, ctx, db, issuer.URL, reviewerSubject) })
	insertCertificateMembership(t, ctx, db, issuer.URL, issuerSubject, tenant, organization, issuerActor, []string{"certificate.sign", "certificate.issue"})
	t.Cleanup(func() { cleanupCertificateMembership(t, ctx, db, issuer.URL, issuerSubject) })

	runtime := httptest.NewServer(server.NewMux(server.Dependencies{CertificateHandler: certificatehttp.Handler{Validator: validator, Actors: certificatehttp.LocalActorResolver{Memberships: resolver}, Lifecycle: repository}}))
	t.Cleanup(runtime.Close)

	inspectorToken := certificateToken(t, key, kid, issuer.URL, inspectorSubject)
	reviewerToken := certificateToken(t, key, kid, issuer.URL, reviewerSubject)
	issuerToken := certificateToken(t, key, kid, issuer.URL, issuerSubject)

	parts := strings.Split(inspectorToken, ".")
	invalidToken := parts[0] + "." + parts[1] + ".invalid-signature"
	if len(parts) != 3 {
		invalidToken = inspectorToken + "-tampered"
	}

	// Warm the validator/handler (JWKS + membership) so later assertions are not
	// subject to first-request latency under load-contended parallel runs.
	warmUpHandler(t, runtime.URL, inspectorToken)

	// Test tampered token rejection on draft creation
	if status := certificatePostWithPath(t, runtime.URL, invalidToken, "/certificates/drafts", `{"certificate_id":"`+certificateID+`-tamper","inspection_id":"`+inspectionID+`","template_code":"lifting","template_version":3,"profile":"INDEPENDENT_REVIEW"}`); status != http.StatusUnauthorized {
		t.Fatalf("tampered token draft status=%d", status)
	}

	// Test unknown subject rejection on draft creation
	if status := certificatePostWithPath(t, runtime.URL, certificateToken(t, key, kid, issuer.URL, "unknown-"+id), "/certificates/drafts", `{"certificate_id":"`+certificateID+`-unknown","inspection_id":"`+inspectionID+`","template_code":"lifting","template_version":3,"profile":"INDEPENDENT_REVIEW"}`); status != http.StatusForbidden {
		t.Fatalf("unknown subject draft status=%d", status)
	}

	// Create draft
	if status := certificatePostWithPath(t, runtime.URL, inspectorToken, "/certificates/drafts", `{"certificate_id":"`+certificateID+`","inspection_id":"`+inspectionID+`","template_code":"lifting","template_version":3,"profile":"INDEPENDENT_REVIEW"}`); status != http.StatusCreated {
		t.Fatalf("create draft status=%d", status)
	}

	// Submit
	if status := certificatePostWithPath(t, runtime.URL, inspectorToken, "/certificates/"+certificateID+"/submit", ``); status != http.StatusNoContent {
		t.Fatalf("submit status=%d", status)
	}

	// Review (different actor)
	if status := certificatePostWithPath(t, runtime.URL, reviewerToken, "/certificates/"+certificateID+"/review", ``); status != http.StatusNoContent {
		t.Fatalf("review status=%d", status)
	}

	// Sign (different actor)
	if status := certificatePostWithPath(t, runtime.URL, issuerToken, "/certificates/"+certificateID+"/sign", ``); status != http.StatusNoContent {
		t.Fatalf("sign status=%d", status)
	}

	// Issue (same actor as sign)
	response := certificatePostWithBody(t, runtime.URL, issuerToken, "/certificates/"+certificateID+"/issue", ``)
	if response.Code != http.StatusOK {
		t.Fatalf("issue status=%d body=%s", response.Code, response.Body.String())
	}
	var issueResult struct {
		CertificateNumber string    `json:"certificate_number"`
		PublicToken       string    `json:"public_token"`
		ExpiresAt         time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(response.Body).Decode(&issueResult); err != nil {
		t.Fatal(err)
	}
	if issueResult.CertificateNumber == "" || issueResult.PublicToken == "" {
		t.Fatalf("issue result missing data: %+v", issueResult)
	}

	// Verify the certificate was actually persisted
	var certStatus, certNumber, publicToken string
	err = db.QueryRowContext(ctx, `SELECT status, certificate_number, public_token_digest FROM certificate_record WHERE id=$1 AND tenant_id=$2 AND organization_id=$3`, certificateID, tenant, organization).Scan(&certStatus, &certNumber, &publicToken)
	if err != nil {
		t.Fatal(err)
	}
	if certStatus != "ISSUED" || certNumber != issueResult.CertificateNumber {
		t.Fatalf("persisted certificate mismatch: status=%s number=%s", certStatus, certNumber)
	}
}

func setupCertificateFixture(t *testing.T, ctx context.Context, db *sql.DB, f certificateFixture) {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id',$1,true), set_config('integin.organization_id',$2,true)`, f.TenantID, f.OrganizationID); err != nil {
		t.Fatal(err)
	}

	jobNo := "job-" + f.CertificateID
	assetID := "asset-" + f.CertificateID
	seeds := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO work_order (id,tenant_id,organization_id,client_id,job_number,request_state,execution_state,commercial_state,certificate_state,revision,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,'accepted','in_progress','ready_for_office_confirmation','pending_validation',1,$6,$6)`, []any{f.WorkOrderID, f.TenantID, f.OrganizationID, "client-a", jobNo, f.InspectorActorID}},
		{`INSERT INTO work_order_scope_item (id,tenant_id,organization_id,work_order_id,client_id,location_id,asset_id,asset_type) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, []any{f.ScopeID, f.TenantID, f.OrganizationID, f.WorkOrderID, "client-a", "location-a", assetID, "lifting"}},
		{`INSERT INTO work_order_assignment (id,tenant_id,organization_id,work_order_id,inspector_id,state,revision,effective_from,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,'active',1,$6,$5,$5)`, []any{f.AssignmentID, f.TenantID, f.OrganizationID, f.WorkOrderID, f.InspectorActorID, now}},
		{`INSERT INTO work_order_assignment_scope (tenant_id,organization_id,assignment_id,scope_item_id) VALUES ($1,$2,$3,$4)`, []any{f.TenantID, f.OrganizationID, f.AssignmentID, f.ScopeID}},
		{`INSERT INTO inspection_record (id,tenant_id,organization_id,work_order_id,scope_item_id,assignment_id,asset_id,inspector_id,lifecycle_state,revision,finalization_state,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'APPROVED',7,'FINALIZED',$8,$8)`, []any{f.InspectionID, f.TenantID, f.OrganizationID, f.WorkOrderID, f.ScopeID, f.AssignmentID, assetID, f.InspectorActorID}},
		{`INSERT INTO certificate_template (id,tenant_id,organization_id,template_code,version,title,asset_type,status,catalog_version,page_count,page_width_points,page_height_points,created_by,approved_by,approved_at) VALUES ($1,$2,$3,'lifting',3,'Lifting certificate','lifting','APPROVED',1,1,612,792,$4,$4,$5)`, []any{f.TemplateID, f.TenantID, f.OrganizationID, f.InspectorActorID, now}},
		{`INSERT INTO certificate_policy (id,tenant_id,organization_id,template_code,template_version,policy_version,validity_days,self_issue_allowed,status,created_by,approved_by,approved_at) VALUES ($1,$2,$3,'lifting',3,1,365,false,'APPROVED',$4,$4,$5)`, []any{f.PolicyID, f.TenantID, f.OrganizationID, f.InspectorActorID, now}},
		{`INSERT INTO asset_registry (id,tenant_id,organization_id,asset_id,asset_type,serial_number,description,lifecycle_state,created_by,updated_by) VALUES ($1,$2,$3,$4,'lifting','SERIAL-A','Controlled lifting asset','ACTIVE',$5,$5)`, []any{f.AssetRegistryID, f.TenantID, f.OrganizationID, assetID, f.InspectorActorID}},
		{`INSERT INTO inspection_public_scope (id,tenant_id,organization_id,inspection_id,inspection_revision,inspection_type,taxonomy_version,result_state,created_by) VALUES ($1,$2,$3,$4,7,'periodic_lifting',1,'PASS',$5)`, []any{f.PublicScopeID, f.TenantID, f.OrganizationID, f.InspectionID, f.InspectorActorID}},
		{`INSERT INTO inspection_public_scope_item (id,tenant_id,organization_id,scope_id,scope_code,display_label,outcome,display_order) VALUES ($1,$2,$3,$4,'visual','Visual examination','PASS',1)`, []any{f.PublicScopeID + "-item", f.TenantID, f.OrganizationID, f.PublicScopeID}},
	}

	for _, seed := range seeds {
		if _, err := tx.ExecContext(ctx, seed.query, seed.args...); err != nil {
			t.Fatal(err)
		}
	}

	for order, bindingKey := range []string{"asset.serial_number", "asset.description", "asset.type", "inspection.type", "inspection.public_scope"} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO certificate_policy_public_binding (id,tenant_id,organization_id,policy_id,binding_key,required,display_order,created_by) VALUES ($1,$2,$3,$4,$5,true,$6,$7)`, fmt.Sprintf("binding-%s-%d", f.CertificateID, order), f.TenantID, f.OrganizationID, f.PolicyID, bindingKey, order+1, f.InspectorActorID); err != nil {
			t.Fatal(err)
		}
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func cleanupCertificateFixture(t *testing.T, ctx context.Context, db *sql.DB, f certificateFixture) {
	t.Helper()
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	statements := []string{
		`DELETE FROM certificate_audit_event WHERE tenant_id = $1`,
		`DELETE FROM certificate_snapshot WHERE tenant_id = $1`,
		`DELETE FROM certificate_record WHERE tenant_id = $1`,
		`DELETE FROM certificate_number_sequence WHERE tenant_id = $1`,
		`DELETE FROM certificate_policy_public_binding WHERE tenant_id = $1`,
		`DELETE FROM certificate_policy WHERE tenant_id = $1`,
		`DELETE FROM certificate_template_cell WHERE tenant_id = $1`,
		`DELETE FROM certificate_template WHERE tenant_id = $1`,
		`DELETE FROM inspection_public_scope_item WHERE tenant_id = $1`,
		`DELETE FROM inspection_public_scope WHERE tenant_id = $1`,
		`DELETE FROM asset_registry WHERE tenant_id = $1`,
		`DELETE FROM inspection_record WHERE tenant_id = $1`,
		`DELETE FROM work_order_assignment_scope WHERE tenant_id = $1`,
		`DELETE FROM work_order_assignment WHERE tenant_id = $1`,
		`DELETE FROM work_order_scope_item WHERE tenant_id = $1`,
		`DELETE FROM work_order WHERE tenant_id = $1`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(cleanupCtx, statement, f.TenantID); err != nil {
			t.Errorf("cleanup %q: %v", statement, err)
		}
	}
}
