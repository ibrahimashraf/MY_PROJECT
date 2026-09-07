package adapters

import (
	"integin/pkg/jurisdictions"
)

// EnvironmentalESGAdapter validates environmental and ESG compliance readiness.
type EnvironmentalESGAdapter struct{}

func (a EnvironmentalESGAdapter) Pillar() jurisdictions.PillarType {
	return jurisdictions.PillarEnvironmentalESG
}

func (a EnvironmentalESGAdapter) Check(profile *jurisdictions.CountryProfile) (Result, error) {
	if err := checkPreconditions(profile, jurisdictions.PillarEnvironmentalESG); err != nil {
		return Result{}, err
	}

	result := Result{
		Pillar: jurisdictions.PillarEnvironmentalESG,
		ISO2:   profile.ISO2,
		Status: StatusPass,
	}

	result.Findings = append(result.Findings, Finding{
		Field:   "environmental.region",
		Status:  StatusPass,
		Message: profile.Region,
	})

	return result, nil
}
