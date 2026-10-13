package featureflag

import (
	"errors"
	"time"

	"integin/internal/shared/featureflags"
)

type Override struct {
	Key       featureflags.Key
	Scope     featureflags.Scope
	ScopeID   string
	State     featureflags.State
	ExpiresAt time.Time
}
type Request struct {
	OrganizationID string
	UserID         string
	ClientID       string
	ProjectID      string
	DeviceID       string
}
type Service struct {
	defaults  map[featureflags.Key]featureflags.State
	overrides []Override
}

func New() *Service                                                          { return &Service{defaults: make(map[featureflags.Key]featureflags.State)} }
func (s *Service) SetDefault(key featureflags.Key, state featureflags.State) { s.defaults[key] = state }
func (s *Service) SetOverride(override Override) error {
	if override.Key == "" || override.Scope == "" || override.ScopeID == "" {
		return errors.New("feature override key, scope, and id are required")
	}
	s.overrides = append(s.overrides, override)
	return nil
}
func (s *Service) Evaluate(key featureflags.Key, request Request, at time.Time) featureflags.State {
	best := featureflags.Inherited
	bestRank := -1
	if state, ok := s.defaults[key]; ok {
		best = state
		bestRank = 0
	}
	for _, override := range s.overrides {
		if override.Key != key || (!override.ExpiresAt.IsZero() && !at.Before(override.ExpiresAt)) {
			continue
		}
		if !matches(override, request) {
			continue
		}
		rank := scopeRank(override.Scope)
		if rank > bestRank {
			best, bestRank = override.State, rank
		}
	}
	return best
}
func matches(override Override, request Request) bool {
	switch override.Scope {
	case featureflags.ScopeOrganization:
		return override.ScopeID == request.OrganizationID
	case featureflags.ScopeUser:
		return override.ScopeID == request.UserID
	case featureflags.ScopeClient:
		return override.ScopeID == request.ClientID
	case featureflags.ScopeProject:
		return override.ScopeID == request.ProjectID
	case featureflags.ScopeDevice:
		return override.ScopeID == request.DeviceID
	}
	return false
}
func scopeRank(scope featureflags.Scope) int {
	switch scope {
	case featureflags.ScopeOrganization:
		return 1
	case featureflags.ScopeUser:
		return 2
	case featureflags.ScopeClient:
		return 3
	case featureflags.ScopeProject:
		return 4
	case featureflags.ScopeDevice:
		return 5
	}
	return -1
}
