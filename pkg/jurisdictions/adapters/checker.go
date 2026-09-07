package adapters

import (
	"integin/pkg/jurisdictions"
)

var allAdapters = []Adapter{
	SafetyRegulationAdapter{},
	TaxFiscalAdapter{},
	AccreditationAdapter{},
	CertificateMarkingAdapter{},
	CurrencyTradeAdapter{},
	EnvironmentalESGAdapter{},
	LaborWorkforceAdapter{},
}

// adapterByPillar indexes registered adapters by their pillar type.
var adapterByPillar map[jurisdictions.PillarType]Adapter

func init() {
	adapterByPillar = make(map[jurisdictions.PillarType]Adapter, len(allAdapters))
	for _, a := range allAdapters {
		adapterByPillar[a.Pillar()] = a
	}
}

// CheckAll runs the adapter for every pillar the jurisdiction actively declares.
// Returns one Result per declared pillar in a deterministic order. Adapter-level
// errors for undeclared pillars are never returned because dispatch is scoped to
// profile.ActivePillars.
func CheckAll(profile *jurisdictions.CountryProfile) ([]Result, error) {
	if profile == nil {
		return nil, ErrNilProfile
	}
	var results []Result
	for _, pillar := range profile.ActivePillars {
		adapter, ok := adapterByPillar[pillar]
		if !ok {
			continue
		}
		result, err := adapter.Check(profile)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}
	return results, nil
}

// Check runs the adapter for a single named pillar over a profile.
func Check(profile *jurisdictions.CountryProfile, pillar jurisdictions.PillarType) (Result, error) {
	if profile == nil {
		return Result{}, ErrNilProfile
	}
	adapter, ok := adapterByPillar[pillar]
	if !ok {
		return Result{}, ErrUnsupportedPillar
	}
	return adapter.Check(profile)
}

// UnsupportedPillarError constructs a descriptive "pillar unsupported" error for
// callers that pass an unknown pillar into Check.
func UnsupportedPillarError(p jurisdictions.PillarType) error {
	return &unsupportedPillarError{p}
}

type unsupportedPillarError struct {
	Pillar jurisdictions.PillarType
}

func (e *unsupportedPillarError) Error() string {
	return "adapter not registered for pillar " + string(e.Pillar)
}
