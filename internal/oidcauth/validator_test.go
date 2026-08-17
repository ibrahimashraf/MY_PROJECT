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
	value := jwt.NewWithClaims(jwt.SigningMethodRS256, claims{RegisteredClaims: jwt.RegisteredClaims{Issuer: issuer, Subject: subject, Audience: jwt.ClaimStrings{audience}, ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)), IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now.Add(-time.Minute))}, AMR: []string{"pwd"}})
	value.Header["kid"] = "fixture-key"
	value.Header["typ"] = "Bearer"
	encoded, err := value.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return encoded
}
