package aiintegration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"integin/internal/advisory"
	leimenv "integin/internal/shared/env"
)

const ContractVersion = "v1"

type Config struct {
	Endpoint string
	APIKey   string
	Timeout  time.Duration
}
type Client struct {
	endpoint   *url.URL
	apiKey     string
	timeout    time.Duration
	httpClient *http.Client
}
type Request struct {
	Version      string         `json:"version"`
	TenantID     string         `json:"tenant_id"`
	Zone         advisory.Zone  `json:"zone"`
	Lens         string         `json:"lens"`
	Inputs       map[string]any `json:"inputs,omitempty"`
	EvidenceRefs []string       `json:"evidence_refs,omitempty"`
}
type Response struct {
	Version       string   `json:"version"`
	TenantID      string   `json:"tenant_id"`
	Title         string   `json:"title"`
	Summary       string   `json:"summary"`
	Severity      string   `json:"severity"`
	Confidence    float64  `json:"confidence"`
	Rationale     string   `json:"rationale"`
	EvidenceRefs  []string `json:"evidence_refs"`
	Provider      string   `json:"provider"`
	Model         string   `json:"model"`
	PromptVersion string   `json:"prompt_version"`
	Limitations   []string `json:"limitations"`
	Blocking      bool     `json:"blocking"`
}

func New(config Config, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(config.Endpoint) == "" {
		return nil, errors.New("AI endpoint is required")
	}
	parsed, err := url.Parse(strings.TrimRight(config.Endpoint, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("valid AI endpoint is required")
	}
	if config.Timeout <= 0 {
		config.Timeout = leimenv.Seconds("INTEGIN_AI_TIMEOUT_S", 5*time.Second, 1, 120)
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{endpoint: parsed, apiKey: config.APIKey, timeout: config.Timeout, httpClient: httpClient}, nil
}
func (c *Client) Generate(ctx context.Context, request advisory.Request) (advisory.ProviderResult, error) {
	if !advisory.AIAllowed(request.Zone) {
		return advisory.ProviderResult{}, errors.New("AI service is not allowed in this zone")
	}
	if strings.TrimSpace(request.TenantID) == "" {
		return advisory.ProviderResult{}, errors.New("tenant id is required")
	}
	payload, err := json.Marshal(Request{Version: ContractVersion, TenantID: request.TenantID, Zone: request.Zone, Lens: request.Lens, Inputs: request.Inputs, EvidenceRefs: request.EvidenceRefs})
	if err != nil {
		return advisory.ProviderResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint.JoinPath("v1", "advisory").String(), strings.NewReader(string(payload)))
	if err != nil {
		return advisory.ProviderResult{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return advisory.ProviderResult{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return advisory.ProviderResult{}, fmt.Errorf("AI service returned status %d", response.StatusCode)
	}
	var result Response
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return advisory.ProviderResult{}, fmt.Errorf("decode AI response: %w", err)
	}
	if result.Version != ContractVersion {
		return advisory.ProviderResult{}, errors.New("unsupported AI contract version")
	}
	if result.TenantID != "" && result.TenantID != request.TenantID {
		return advisory.ProviderResult{}, errors.New("AI response tenant mismatch")
	}
	return advisory.ProviderResult{Title: result.Title, Summary: result.Summary, Severity: result.Severity, Confidence: result.Confidence, Rationale: result.Rationale, Provider: result.Provider, Model: result.Model, PromptVersion: result.PromptVersion, Limitations: append([]string(nil), result.Limitations...)}, nil
}
func (c *Client) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint.JoinPath("healthz").String(), nil)
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("AI health returned status %d", response.StatusCode)
	}
	return nil
}
