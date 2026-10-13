package deployconfig

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AIEndpoint         string
	APIKey             string
	Timeout            time.Duration
	IntegrationEnabled bool
}

func Load(env map[string]string) (Config, error) {
	config := Config{AIEndpoint: strings.TrimSpace(env["INTEGIN_AI_ENDPOINT"]), APIKey: env["INTEGIN_AI_API_KEY"]}
	enabled, err := strconv.ParseBool(defaultValue(env["INTEGIN_AI_ENABLED"], "false"))
	if err != nil {
		return Config{}, errors.New("INTEGIN_AI_ENABLED must be boolean")
	}
	config.IntegrationEnabled = enabled
	if timeout := strings.TrimSpace(env["INTEGIN_AI_TIMEOUT"]); timeout != "" {
		seconds, parseErr := strconv.Atoi(timeout)
		if parseErr != nil || seconds <= 0 {
			return Config{}, errors.New("INTEGIN_AI_TIMEOUT must be positive seconds")
		}
		config.Timeout = time.Duration(seconds) * time.Second
	} else {
		config.Timeout = 5 * time.Second
	}
	if config.IntegrationEnabled && config.AIEndpoint == "" {
		return Config{}, errors.New("INTEGIN_AI_ENDPOINT is required when integration is enabled")
	}
	return config, nil
}
func defaultValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
