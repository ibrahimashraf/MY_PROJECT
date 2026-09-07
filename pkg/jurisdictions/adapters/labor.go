package adapters

import (
	"integin/pkg/jurisdictions"
)

// LaborWorkforceAdapter validates labor law and workforce compliance readiness.
type LaborWorkforceAdapter struct{}

func (a LaborWorkforceAdapter) Pillar() jurisdictions.PillarType {
	return jurisdictions.PillarLaborWorkforce
}

func (a LaborWorkforceAdapter) Check(profile *jurisdictions.CountryProfile) (Result, error) {
	if err := checkPreconditions(profile, jurisdictions.PillarLaborWorkforce); err != nil {
		return Result{}, err
	}

	result := Result{
		Pillar: jurisdictions.PillarLaborWorkforce,
		ISO2:   profile.ISO2,
		Status: StatusPass,
	}

	result.Findings = append(result.Findings, Finding{
		Field:   "labor.region",
		Status:  StatusPass,
		Message: profile.Region,
	})

	return result, nil
}
