package oidcauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// TokenProvider coordinates the acquisition, caching, and automatic rotation of short-lived M2M OAuth2 tokens.
type TokenProvider interface {
	GetToken(ctx context.Context) (string, error)
}

// ClientCredentialsConfig holds the configuration needed for Keycloak OAuth2 client credentials flow.
type ClientCredentialsConfig struct {
	TokenEndpoint string
	ClientID      string
	ClientSecret  string
	Scope         string
	RefreshBuffer time.Duration
}

// ClientCredentialsTokenProvider acquires short-lived tokens from Keycloak using OAuth2 client credentials.
// It caches the token in memory and rotates it safely before expiry.
type ClientCredentialsTokenProvider struct {
	config ClientCredentialsConfig
	client *http.Client
	now    func() time.Time

	mu          sync.RWMutex
	cachedToken string
	expiresAt   time.Time
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// NewClientCredentialsTokenProvider creates a new token provider with validation.
func NewClientCredentialsTokenProvider(config ClientCredentialsConfig, client *http.Client) (*ClientCredentialsTokenProvider, error) {
	if strings.TrimSpace(config.TokenEndpoint) == "" {
		return nil, errors.New("token endpoint is required")
	}
	if strings.TrimSpace(config.ClientID) == "" {
		return nil, errors.New("client id is required")
	}
	if strings.TrimSpace(config.ClientSecret) == "" {
		return nil, errors.New("client secret is required")
	}
	if config.RefreshBuffer <= 0 {
		config.RefreshBuffer = 30 * time.Second
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	return &ClientCredentialsTokenProvider{
		config: config,
		client: client,
		now:    time.Now,
	}, nil
}

// GetToken returns a valid, non-expired OAuth2 bearer token, fetching or refreshing as necessary.
func (p *ClientCredentialsTokenProvider) GetToken(ctx context.Context) (string, error) {
	p.mu.RLock()
	now := p.now().UTC()
	if p.cachedToken != "" && now.Before(p.expiresAt.Add(-p.config.RefreshBuffer)) {
		token := p.cachedToken
		p.mu.RUnlock()
		return token, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double check after acquiring write lock
	now = p.now().UTC()
	if p.cachedToken != "" && now.Before(p.expiresAt.Add(-p.config.RefreshBuffer)) {
		return p.cachedToken, nil
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", p.config.ClientID)
	data.Set("client_secret", p.config.ClientSecret)
	if p.config.Scope != "" {
		data.Set("scope", p.config.Scope)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return "", fmt.Errorf("reading token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		return "", fmt.Errorf("parsing token response JSON: %w", err)
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return "", errors.New("token response contained empty access_token")
	}

	p.cachedToken = tokenResp.AccessToken
	p.expiresAt = now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return p.cachedToken, nil
}
