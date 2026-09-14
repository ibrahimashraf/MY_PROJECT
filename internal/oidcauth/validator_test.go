// INTEGIN OIDC foundation tests: deterministic JWKS fixtures prove issuer authentication without any INTEGIN authorization grant.
package oidcauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestValidatorAcceptsConfiguredIssuerSubject(t *testing.T) {
	privateKey, server := testOIDCServer(t)
	defer server.Close()
	validator, err := NewValidator(context.Background(), Config{Enabled: true, Issuer: server.URL, Audience: "integin-api-pilot", AllowInsecureLoopbackIssuer: true, ClockSkew: time.Minute, MaxTokenAge: time.Hour, JWKSRefresh: time.Hour}, nil)
	if err != nil {
		t.Fatalf("construct validator: %v", err)
	}
	token := signedToken(t, privateKey, server.URL, "integin-api-pilot", "subject-001", time.Now().UTC())
	principal, err := validator.Validate(context.Background(), token)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if principal.Issuer != server.URL || principal.Subject != "subject-001" {
		t.Fatalf("unexpected principal: %+v", principal)
	}
}

func TestValidatorAcceptsKeycloakJWTTypeWithConfiguredAuthorizedParty(t *testing.T) {
	privateKey, server := testOIDCServer(t)
	defer server.Close()
	validator, err := NewValidator(context.Background(), Config{Enabled: true, Issuer: server.URL, Audience: "integin-api-pilot", AuthorizedParty: "keycloak-service-client", AllowInsecureLoopbackIssuer: true, ClockSkew: time.Minute, MaxTokenAge: time.Hour, JWKSRefresh: time.Hour}, nil)
	if err != nil {
		t.Fatalf("construct validator: %v", err)
	}
	value := jwt.NewWithClaims(jwt.SigningMethodRS256, claims{RegisteredClaims: jwt.RegisteredClaims{Issuer: server.URL, Subject: "service-account-keycloak", Audience: jwt.ClaimStrings{"account", "integin-api-pilot"}, ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(10 * time.Minute)), IssuedAt: jwt.NewNumericDate(time.Now().UTC())}, AuthorizedParty: "keycloak-service-client"})
	value.Header["kid"] = "fixture-key"
	value.Header["typ"] = "JWT"
	token, err := value.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign Keycloak-style token: %v", err)
	}
	if _, err := validator.Validate(context.Background(), token); err != nil {
		t.Fatalf("validate Keycloak-style token: %v", err)
	}
}

func TestValidatorRejectsWrongAudienceAndUnknownSubjectFormat(t *testing.T) {
	privateKey, server := testOIDCServer(t)
	defer server.Close()
	validator, err := NewValidator(context.Background(), Config{Enabled: true, Issuer: server.URL, Audience: "integin-api-pilot", AllowInsecureLoopbackIssuer: true, ClockSkew: time.Minute, MaxTokenAge: time.Hour, JWKSRefresh: time.Hour}, nil)
	if err != nil {
		t.Fatalf("construct validator: %v", err)
	}
	wrongAudience := signedToken(t, privateKey, server.URL, "other-api", "subject-001", time.Now().UTC())
	if _, err := validator.Validate(context.Background(), wrongAudience); err == nil || !strings.Contains(err.Error(), "audience") {
		t.Fatalf("expected audience rejection, got %v", err)
	}
	badSubject := signedToken(t, privateKey, server.URL, "integin-api-pilot", "subject with space", time.Now().UTC())
	if _, err := validator.Validate(context.Background(), badSubject); err == nil || !strings.Contains(err.Error(), "subject") {
		t.Fatalf("expected subject rejection, got %v", err)
	}
}

func TestValidatorRejectsWrongIssuerBadSignatureAndExpiredToken(t *testing.T) {
	privateKey, server := testOIDCServer(t)
	defer server.Close()
	validator, err := NewValidator(context.Background(), Config{Enabled: true, Issuer: server.URL, Audience: "integin-api-pilot", AllowInsecureLoopbackIssuer: true, ClockSkew: time.Minute, MaxTokenAge: time.Hour, JWKSRefresh: time.Hour}, nil)
	if err != nil {
		t.Fatalf("construct validator: %v", err)
	}
	wrongIssuer := signedToken(t, privateKey, server.URL+"/unexpected", "integin-api-pilot", "subject-001", time.Now().UTC())
	if _, err := validator.Validate(context.Background(), wrongIssuer); err == nil || !strings.Contains(err.Error(), "issuer") {
		t.Fatalf("expected issuer rejection, got %v", err)
	}
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate alternate RSA key: %v", err)
	}
	badSignature := signedToken(t, otherKey, server.URL, "integin-api-pilot", "subject-001", time.Now().UTC())
	if _, err := validator.Validate(context.Background(), badSignature); err == nil {
		t.Fatal("expected bad-signature rejection")
	}
	expired := signedToken(t, privateKey, server.URL, "integin-api-pilot", "subject-001", time.Now().UTC().Add(-20*time.Minute))
	if _, err := validator.Validate(context.Background(), expired); err == nil {
		t.Fatal("expected expired-token rejection")
	}
}

func testOIDCServer(t *testing.T) (*rsa.PrivateKey, *httptest.Server) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(writer).Encode(map[string]any{"issuer": server.URL, "jwks_uri": server.URL + "/keys", "authorization_endpoint": server.URL + "/authorize"})
		case "/keys":
			_ = json.NewEncoder(writer).Encode(jwksDocument{Keys: []jwk{{KTY: "RSA", KID: "fixture-key", ALG: "RS256", USE: "sig", N: base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()), E: base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1})}}})
		default:
			http.NotFound(writer, request)
		}
	}))
	return privateKey, server
}

func signedToken(t *testing.T, privateKey *rsa.PrivateKey, issuer, audience, subject string, now time.Time) string {
	t.Helper()
	return signedTokenFrom(t, privateKey, "fixture-key", issuer, audience, subject, now, []string{"pwd"})
}

func signedTokenFrom(t *testing.T, privateKey *rsa.PrivateKey, keyID, issuer, audience, subject string, now time.Time, amr []string) string {
	t.Helper()
	value := jwt.NewWithClaims(jwt.SigningMethodRS256, claims{RegisteredClaims: jwt.RegisteredClaims{Issuer: issuer, Subject: subject, Audience: jwt.ClaimStrings{audience}, ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)), IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now.Add(-time.Minute))}, AMR: amr})
	value.Header["kid"] = keyID
	value.Header["typ"] = "Bearer"
	encoded, err := value.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return encoded
}

// rotatableOIDCServer serves discovery plus a JWKS whose key set and failure
// mode can be mutated mid-test to model key rotation and refresh outages.
type rotatableOIDCServer struct {
	server   *httptest.Server
	mu       sync.Mutex
	keys     map[string]*rsa.PrivateKey
	keysFail bool
}

func newRotatableOIDCServer(t *testing.T, initial map[string]*rsa.PrivateKey) *rotatableOIDCServer {
	t.Helper()
	rotating := &rotatableOIDCServer{keys: initial}
	rotating.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(writer).Encode(map[string]any{"issuer": rotating.server.URL, "jwks_uri": rotating.server.URL + "/keys"})
		case "/keys":
			rotating.mu.Lock()
			fail := rotating.keysFail
			keys := rotating.keys
			rotating.mu.Unlock()
			if fail {
				http.Error(writer, "keys unavailable", http.StatusServiceUnavailable)
				return
			}
			var document jwksDocument
			for keyID, key := range keys {
				document.Keys = append(document.Keys, jwk{KTY: "RSA", KID: keyID, ALG: "RS256", USE: "sig", N: base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()), E: base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1})})
			}
			_ = json.NewEncoder(writer).Encode(document)
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(rotating.server.Close)
	return rotating
}

func (s *rotatableOIDCServer) setKeys(keys map[string]*rsa.PrivateKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys = keys
}

func (s *rotatableOIDCServer) setKeysFailure(fail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keysFail = fail
}

func TestValidatorRotatesJWKSAndRejectsTokensFromRetiredKeys(t *testing.T) {
	keyA, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key A: %v", err)
	}
	keyB, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key B: %v", err)
	}
	rotating := newRotatableOIDCServer(t, map[string]*rsa.PrivateKey{"key-A": keyA})
	validator, err := NewValidator(context.Background(), Config{Enabled: true, Issuer: rotating.server.URL, Audience: "integin-api-pilot", AllowInsecureLoopbackIssuer: true, ClockSkew: time.Minute, MaxTokenAge: time.Hour, JWKSRefresh: time.Nanosecond}, nil)
	if err != nil {
		t.Fatalf("construct validator: %v", err)
	}

	tokenA := signedTokenFrom(t, keyA, "key-A", rotating.server.URL, "integin-api-pilot", "subject-001", time.Now().UTC(), []string{"pwd"})
	if _, err := validator.Validate(context.Background(), tokenA); err != nil {
		t.Fatalf("pre-rotation token must validate: %v", err)
	}

	// Rotate: key-A is retired from the JWKS and key-B becomes the active signing key.
	rotating.setKeys(map[string]*rsa.PrivateKey{"key-B": keyB})

	// A fresh token signed by the retired key fails closed (a distinct token
	// avoids the bounded verify cache masking the rotation).
	retired := signedTokenFrom(t, keyA, "key-A", rotating.server.URL, "integin-api-pilot", "subject-rotated", time.Now().UTC(), []string{"pwd"})
	if _, err := validator.Validate(context.Background(), retired); err == nil {
		t.Fatal("token signed by a rotated-away key must be rejected")
	}

	// The rotated-in key is picked up and new tokens validate against it.
	tokenB := signedTokenFrom(t, keyB, "key-B", rotating.server.URL, "integin-api-pilot", "subject-001", time.Now().UTC(), []string{"pwd"})
	principal, err := validator.Validate(context.Background(), tokenB)
	if err != nil {
		t.Fatalf("token signed by rotated-in key must validate: %v", err)
	}
	if principal.Subject != "subject-001" {
		t.Fatalf("unexpected principal: %+v", principal)
	}
}

func TestValidatorRefreshFailureFailsClosed(t *testing.T) {
	keyA, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key A: %v", err)
	}
	rotating := newRotatableOIDCServer(t, map[string]*rsa.PrivateKey{"key-A": keyA})
	validator, err := NewValidator(context.Background(), Config{Enabled: true, Issuer: rotating.server.URL, Audience: "integin-api-pilot", AllowInsecureLoopbackIssuer: true, ClockSkew: time.Minute, MaxTokenAge: time.Hour, JWKSRefresh: time.Nanosecond}, nil)
	if err != nil {
		t.Fatalf("construct validator: %v", err)
	}
	first := signedTokenFrom(t, keyA, "key-A", rotating.server.URL, "integin-api-pilot", "subject-001", time.Now().UTC(), []string{"pwd"})
	if _, err := validator.Validate(context.Background(), first); err != nil {
		t.Fatalf("token must validate while JWKS serves: %v", err)
	}

	// Break the JWKS endpoint. The next uncached validation must fail closed
	// rather than reuse stale keys. A distinct subject avoids the verify cache.
	rotating.setKeysFailure(true)
	second := signedTokenFrom(t, keyA, "key-A", rotating.server.URL, "integin-api-pilot", "subject-other", time.Now().UTC(), []string{"pwd"})
	if _, err := validator.Validate(context.Background(), second); err == nil {
		t.Fatal("validation must fail closed when the JWKS refresh fails")
	}
}

func TestValidatorRequiresAMRForPrivilegedClients(t *testing.T) {
	privateKey, server := testOIDCServer(t)
	defer server.Close()
	validator, err := NewValidator(context.Background(), Config{Enabled: true, Issuer: server.URL, Audience: "integin-api-pilot", RequiredAMR: []string{"mfa"}, AllowInsecureLoopbackIssuer: true, ClockSkew: time.Minute, MaxTokenAge: time.Hour, JWKSRefresh: time.Hour}, nil)
	if err != nil {
		t.Fatalf("construct validator: %v", err)
	}
	now := time.Now().UTC()

	pwdOnly := signedTokenFrom(t, privateKey, "fixture-key", server.URL, "integin-api-pilot", "subject-pwd", now, []string{"pwd"})
	if _, err := validator.Validate(context.Background(), pwdOnly); err == nil || !strings.Contains(err.Error(), "authentication method") {
		t.Fatalf("password-only token must fail the required-AMR check, got %v", err)
	}
	noAMR := signedTokenFrom(t, privateKey, "fixture-key", server.URL, "integin-api-pilot", "subject-nomfa", now, nil)
	if _, err := validator.Validate(context.Background(), noAMR); err == nil || !strings.Contains(err.Error(), "authentication method") {
		t.Fatalf("token without AMR claims must fail the required-AMR check, got %v", err)
	}
	backupOTP := signedTokenFrom(t, privateKey, "fixture-key", server.URL, "integin-api-pilot", "subject-otp", now, []string{"otp"})
	if _, err := validator.Validate(context.Background(), backupOTP); err == nil || !strings.Contains(err.Error(), "authentication method") {
		t.Fatalf("otp-only token must still fail when mfa is mandated, got %v", err)
	}
	withMFA := signedTokenFrom(t, privateKey, "fixture-key", server.URL, "integin-api-pilot", "subject-mfa", now, []string{"pwd", "mfa"})
	if _, err := validator.Validate(context.Background(), withMFA); err != nil {
		t.Fatalf("token carrying the mandated mfa method must validate: %v", err)
	}
}
