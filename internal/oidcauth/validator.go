// INTEGIN OIDC foundation: validates only issuer authentication; local authorization is deliberately out of scope.
package oidcauth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var errUnknownKeyID = errors.New("oidc token key identifier is unknown")

// Principal is the authentication assertion that may be submitted to local INTEGIN identity resolution.
// It intentionally contains no tenant, organization, role, capability, or device authority.
type Principal struct {
	Issuer  string
	Subject string
	AMR     []string
}

type discoveryDocument struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

type jwksDocument struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	KTY string `json:"kty"`
	KID string `json:"kid"`
	ALG string `json:"alg"`
	USE string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type claims struct {
	jwt.RegisteredClaims
	AuthorizedParty string   `json:"azp"`
	AMR             []string `json:"amr"`
}

// Validator validates signed access tokens with discovery and bounded JWKS refreshes.
type Validator struct {
	config  Config
	client  *http.Client
	jwksURI string
	now     func() time.Time

	mu          sync.RWMutex
	keys        map[string]*rsa.PublicKey
	refreshedAt time.Time
}

// NewValidator resolves a configured issuer discovery document and fetches its initial JWK set.
func NewValidator(ctx context.Context, config Config, client *http.Client) (*Validator, error) {
	if !config.Enabled {
		return nil, errors.New("cannot construct an OIDC validator while OIDC is disabled")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	copyClient := *client
	copyClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	validator := &Validator{config: config, client: &copyClient, now: time.Now}
	var document discoveryDocument
	if err := validator.getJSON(ctx, config.Issuer+"/.well-known/openid-configuration", &document); err != nil {
		return nil, fmt.Errorf("OIDC discovery: %w", err)
	}
	if document.Issuer != config.Issuer {
		return nil, errors.New("OIDC discovery issuer does not exactly match configured issuer")
	}
	if err := validateJWKSOrigin(config.Issuer, document.JWKSURI); err != nil {
		return nil, err
	}
	validator.jwksURI = document.JWKSURI
	if err := validator.refresh(ctx); err != nil {
		return nil, err
	}
	return validator, nil
}

// Validate authenticates the configured issuer's subject. It does not resolve INTEGIN authorization.
func (v *Validator) Validate(ctx context.Context, rawToken string) (Principal, error) {
	if strings.TrimSpace(rawToken) == "" {
		return Principal{}, errors.New("missing bearer token")
	}
	if err := v.ensureFresh(ctx); err != nil {
		return Principal{}, err
	}
	for attempt := 0; attempt < 2; attempt++ {
		principal, err := v.validateWithCurrentKeys(rawToken)
		if !errors.Is(err, errUnknownKeyID) || attempt == 1 {
			return principal, err
		}
		if err := v.refresh(ctx); err != nil {
			return Principal{}, err
		}
	}
	return Principal{}, errors.New("unreachable OIDC validation state")
}

func (v *Validator) validateWithCurrentKeys(rawToken string) (Principal, error) {
	v.mu.RLock()
	keys := make(map[string]*rsa.PublicKey, len(v.keys))
	for keyID, key := range v.keys {
		keys[keyID] = key
	}
	v.mu.RUnlock()
	parsedClaims := &claims{}
	token, err := jwt.ParseWithClaims(rawToken, parsedClaims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, errors.New("OIDC token algorithm is not permitted")
		}
		typ, _ := token.Header["typ"].(string)
		// Keycloak access tokens use the established "JWT" type while RFC 9068
		// deployments use "at+jwt". Signature, issuer, audience, and authorized
		// party validation below remain mandatory in either case.
		if typ != "Bearer" && typ != "at+jwt" && typ != "JWT" {
			return nil, errors.New("OIDC token type is not permitted")
		}
		keyID, _ := token.Header["kid"].(string)
		if keyID == "" {
			return nil, errUnknownKeyID
		}
		key, ok := keys[keyID]
		if !ok {
			return nil, errUnknownKeyID
		}
		return key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}), jwt.WithLeeway(v.config.ClockSkew), jwt.WithTimeFunc(v.now))
	if err != nil || token == nil || !token.Valid {
		if err == nil {
			err = errors.New("OIDC token is invalid")
		}
		return Principal{}, err
	}
	if err := v.validateClaims(parsedClaims); err != nil {
		return Principal{}, err
	}
	return Principal{Issuer: parsedClaims.Issuer, Subject: parsedClaims.Subject, AMR: append([]string(nil), parsedClaims.AMR...)}, nil
}

func (v *Validator) validateClaims(value *claims) error {
	now := v.now().UTC()
	if value.Issuer != v.config.Issuer {
		return errors.New("OIDC token issuer mismatch")
	}
	if !contains([]string(value.Audience), v.config.Audience) {
		return errors.New("OIDC token audience mismatch")
	}
	if len(value.Audience) > 1 && v.config.AuthorizedParty == "" {
		return errors.New("multi-audience OIDC token requires configured authorized party")
	}
	if v.config.AuthorizedParty != "" && value.AuthorizedParty != v.config.AuthorizedParty {
		return errors.New("OIDC token authorized party mismatch")
	}
	if !validSubject(value.Subject) {
		return errors.New("OIDC token subject is invalid")
	}
	if value.ExpiresAt == nil || now.After(value.ExpiresAt.Add(v.config.ClockSkew)) {
		return errors.New("OIDC token is expired or missing expiry")
	}
	if value.NotBefore != nil && now.Add(v.config.ClockSkew).Before(value.NotBefore.Time) {
		return errors.New("OIDC token is not active yet")
	}
	if value.IssuedAt == nil || value.IssuedAt.Time.After(now.Add(v.config.ClockSkew)) || now.Sub(value.IssuedAt.Time) > v.config.MaxTokenAge+v.config.ClockSkew {
		return errors.New("OIDC token issue time is invalid")
	}
	for _, required := range v.config.RequiredAMR {
		if !contains(value.AMR, required) {
			return fmt.Errorf("OIDC token is missing required authentication method %q", required)
		}
	}
	return nil
}

func (v *Validator) ensureFresh(ctx context.Context) error {
	v.mu.RLock()
	stale := len(v.keys) == 0 || v.now().Sub(v.refreshedAt) >= v.config.JWKSRefresh
	v.mu.RUnlock()
	if stale {
		return v.refresh(ctx)
	}
	return nil
}

func (v *Validator) refresh(ctx context.Context) error {
	var document jwksDocument
	if err := v.getJSON(ctx, v.jwksURI, &document); err != nil {
		return fmt.Errorf("OIDC JWKS fetch: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(document.Keys))
	for _, candidate := range document.Keys {
		if candidate.KTY != "RSA" || candidate.ALG != "RS256" || (candidate.USE != "" && candidate.USE != "sig") || candidate.KID == "" {
			continue
		}
		key, err := rsaKey(candidate)
		if err != nil {
			return fmt.Errorf("OIDC JWK %q: %w", candidate.KID, err)
		}
		if _, exists := keys[candidate.KID]; exists {
			return errors.New("OIDC JWKS contains duplicate key identifier")
		}
		keys[candidate.KID] = key
	}
	if len(keys) == 0 {
		return errors.New("OIDC JWKS contains no permitted RSA signing key")
	}
	v.mu.Lock()
	v.keys = keys
	v.refreshedAt = v.now()
	v.mu.Unlock()
	return nil
}

func (v *Validator) getJSON(ctx context.Context, endpoint string, destination any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	response, err := v.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status %d", response.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	return nil
}

func validateJWKSOrigin(issuer, jwksURI string) error {
	issuerURL, err := url.Parse(issuer)
	if err != nil {
		return err
	}
	keyURL, err := url.Parse(jwksURI)
	if err != nil || keyURL.Scheme == "" || keyURL.Host == "" {
		return errors.New("OIDC discovery returned invalid JWKS URI")
	}
	if issuerURL.Scheme != keyURL.Scheme || !strings.EqualFold(issuerURL.Host, keyURL.Host) {
		return errors.New("OIDC discovery JWKS URI changes issuer origin")
	}
	return nil
}

func rsaKey(value jwk) (*rsa.PublicKey, error) {
	modulus, err := base64.RawURLEncoding.DecodeString(value.N)
	if err != nil || len(modulus) == 0 {
		return nil, errors.New("invalid RSA modulus")
	}
	exponentBytes, err := base64.RawURLEncoding.DecodeString(value.E)
	if err != nil || len(exponentBytes) == 0 || len(exponentBytes) > 4 {
		return nil, errors.New("invalid RSA exponent")
	}
	exponent := 0
	for _, part := range exponentBytes {
		exponent = exponent<<8 | int(part)
	}
	if exponent < 3 || exponent%2 == 0 {
		return nil, errors.New("invalid RSA exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: exponent}, nil
}

func validSubject(subject string) bool {
	if subject == "" || len(subject) > 255 {
		return false
	}
	for _, value := range []byte(subject) {
		if value < 0x21 || value > 0x7e {
			return false
		}
	}
	return true
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
