package jurisdictions

import (
	"errors"
	"sort"
	"strings"
	"sync"
)

var (
	ErrJurisdictionNotFound  = errors.New("jurisdiction not found")
	ErrDuplicateJurisdiction = errors.New("jurisdiction already registered")
	ErrJurisdictionNotActive = errors.New("jurisdiction is not in ACTIVE lifecycle state")
)

// Registry is a thread-safe, zero-allocation in-memory cache of active country profiles.
// It is immutable after construction: profiles are loaded once via NewRegistry and
// never mutated thereafter. Reads use RLock for maximum concurrency.
type Registry struct {
	mu         sync.RWMutex
	byISO2     map[string]*CountryProfile
	byISO3     map[string]*CountryProfile
	byCurrency map[string][]*CountryProfile
	byPillar   map[PillarType][]*CountryProfile
	all        []*CountryProfile
}

// NewRegistry builds a Registry from a slice of CountryProfiles.
// All profiles must pass Validate() and be in ACTIVE lifecycle state.
// Returns the registry and any validation error encountered.
func NewRegistry(profiles []CountryProfile) (*Registry, error) {
	r := &Registry{
		byISO2:     make(map[string]*CountryProfile, len(profiles)),
		byISO3:     make(map[string]*CountryProfile, len(profiles)),
		byCurrency: make(map[string][]*CountryProfile),
		byPillar:   make(map[PillarType][]*CountryProfile),
		all:        make([]*CountryProfile, 0, len(profiles)),
	}

	for i := range profiles {
		p := &profiles[i]
		if err := p.Validate(); err != nil {
			return nil, err
		}
		if p.Lifecycle != LifecycleActive {
			return nil, ErrJurisdictionNotActive
		}

		iso2 := strings.ToUpper(p.ISO2)
		if _, exists := r.byISO2[iso2]; exists {
			return nil, ErrDuplicateJurisdiction
		}

		r.byISO2[iso2] = p
		r.byISO3[strings.ToUpper(p.ISO3)] = p
		r.byCurrency[strings.ToUpper(p.CurrencyCode)] = append(r.byCurrency[strings.ToUpper(p.CurrencyCode)], p)
		for _, pillar := range p.ActivePillars {
			r.byPillar[pillar] = append(r.byPillar[pillar], p)
		}
		r.all = append(r.all, p)
	}

	return r, nil
}

// ByISO2 returns the profile for a given ISO 3166-1 alpha-2 code (case-insensitive).
func (r *Registry) ByISO2(code string) (*CountryProfile, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.byISO2[strings.ToUpper(strings.TrimSpace(code))]
	return p, ok
}

// ByISO3 returns the profile for a given ISO 3166-1 alpha-3 code (case-insensitive).
func (r *Registry) ByISO3(code string) (*CountryProfile, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.byISO3[strings.ToUpper(strings.TrimSpace(code))]
	return p, ok
}

// ByCurrency returns value copies of all profiles that use a given ISO 4217 currency code.
func (r *Registry) ByCurrency(code string) []CountryProfile {
	r.mu.RLock()
	defer r.mu.RUnlock()
	profiles := r.byCurrency[strings.ToUpper(strings.TrimSpace(code))]
	result := make([]CountryProfile, len(profiles))
	for i, p := range profiles {
		result[i] = *p
	}
	return result
}

// ByPillar returns value copies of all profiles that enforce a given compliance pillar.
func (r *Registry) ByPillar(p PillarType) []CountryProfile {
	r.mu.RLock()
	defer r.mu.RUnlock()
	profiles := r.byPillar[p]
	result := make([]CountryProfile, len(profiles))
	for i, pp := range profiles {
		result[i] = *pp
	}
	return result
}

// All returns a sorted (by ISO2) copy of all registered profiles.
func (r *Registry) All() []*CountryProfile {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*CountryProfile, len(r.all))
	copy(result, r.all)
	sort.Slice(result, func(i, j int) bool {
		return result[i].ISO2 < result[j].ISO2
	})
	return result
}

// Count returns the number of registered jurisdictions.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.all)
}
