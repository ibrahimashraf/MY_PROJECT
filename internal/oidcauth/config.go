// INTEGIN OIDC foundation: configuration is disabled by default and never conveys tenant, organization, or capability authority.
package oidcauth

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultClockSkew   = 60 * time.Second
	defaultMaxTokenAge = 15 * time.Minute
	defaultJWKSRefresh = 5 * time.Minute
)

// Config contains only the verification policy for an external authentication assertion.
// INTEGIN authorization is resolved separately from a validated issuer-subject pair.
type Config struct {
	Enabled                     bool
	Issuer                      string
	Audience                    string
	AuthorizedParty             string
	RequiredAMR                 []string
	ClockSkew                   time.Duration
	MaxTokenAge                 time.Duration
	JWKSRefresh                 time.Duration
	AllowInsecureLoopbackIssuer bool
}

// LoadConfig reads OIDC settings without making a network call. An absent feature flag is disabled.
func LoadConfig(getenv func(string) string) (Config, error) {
	enabledValue := strings.TrimSpace(getenv("INTEGIN_OIDC_ENABLED"))
	if enabledValue == "" {
		return Config{Enabled: false}, nil
	}
	enabled, err := strconv.ParseBool(enabledValue)
	if err != nil {
		return Config{}, fmt.Errorf("INTEGIN_OIDC_ENABLED must be a boolean: %w", err)
	}
	if !enabled {
		return Config{Enabled: false}, nil
	}

	allowLoopback, err := parseBool(getenv("INTEGIN_OIDC_ALLOW_INSECURE_LOOPBACK"), false)
	if err != nil {
		return Config{}, fmt.Errorf("INTEGIN_OIDC_ALLOW_INSECURE_LOOPBACK: %w", err)
	}
	issuer := strings.TrimSpace(getenv("INTEGIN_OIDC_ISSUER"))
	audience := strings.TrimSpace(getenv("INTEGIN_OIDC_AUDIENCE"))
	if issuer == "" || audience == "" {
		return Config{}, fmt.Errorf("INTEGIN_OIDC_ISSUER and INTEGIN_OIDC_AUDIENCE are required when OIDC is enabled")
	}
	parsedIssuer, err := url.Parse(issuer)
	if err != nil || parsedIssuer.Scheme == "" || parsedIssuer.Host == "" || parsedIssuer.RawQuery != "" || parsedIssuer.Fragment != "" || strings.HasSuffix(issuer, "/") {
		return Config{}, fmt.Errorf("INTEGIN_OIDC_ISSUER must be an exact absolute issuer URL without a trailing slash")
	}
	if parsedIssuer.Scheme != "https" && !(allowLoopback && parsedIssuer.Scheme == "http" && isLoopbackHost(parsedIssuer.Hostname())) {
		return Config{}, fmt.Errorf("INTEGIN_OIDC_ISSUER must use HTTPS unless the explicit loopback pilot exception is enabled")
	}
	clockSkew, err := durationFromSeconds(getenv("INTEGIN_OIDC_CLOCK_SKEW_SECONDS"), defaultClockSkew, 0, 300)
	if err != nil {
		return Config{}, fmt.Errorf("INTEGIN_OIDC_CLOCK_SKEW_SECONDS: %w", err)
	}
	maxTokenAge, err := durationFromSeconds(getenv("INTEGIN_OIDC_MAX_TOKEN_AGE_SECONDS"), defaultMaxTokenAge, 60, 3600)
	if err != nil {
		return Config{}, fmt.Errorf("INTEGIN_OIDC_MAX_TOKEN_AGE_SECONDS: %w", err)
	}
	refresh, err := durationFromSeconds(getenv("INTEGIN_OIDC_JWKS_REFRESH_SECONDS"), defaultJWKSRefresh, 30, 3600)
	if err != nil {
		return Config{}, fmt.Errorf("INTEGIN_OIDC_JWKS_REFRESH_SECONDS: %w", err)
	}
	return Config{
		Enabled:                     true,
		Issuer:                      issuer,
		Audience:                    audience,
		AuthorizedParty:             strings.TrimSpace(getenv("INTEGIN_OIDC_AUTHORIZED_PARTY")),
		RequiredAMR:                 splitList(getenv("INTEGIN_OIDC_REQUIRED_AMR")),
		ClockSkew:                   clockSkew,
		MaxTokenAge:                 maxTokenAge,
		JWKSRefresh:                 refresh,
		AllowInsecureLoopbackIssuer: allowLoopback,
	}, nil
}

func parseBool(value string, fallback bool) (bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	return strconv.ParseBool(value)
}

func durationFromSeconds(value string, fallback time.Duration, minimum, maximum int) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < minimum || seconds > maximum {
		return 0, fmt.Errorf("must be an integer between %d and %d seconds", minimum, maximum)
	}
	return time.Duration(seconds) * time.Second, nil
}

func splitList(value string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0)
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		result = append(result, part)
	}
	return result
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
