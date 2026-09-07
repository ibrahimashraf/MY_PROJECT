package adapters

import (
	"fmt"

	"integin/pkg/jurisdictions"
)

// AccreditationAdapter validates that a jurisdiction has recognized accreditation
// bodies for inspector and laboratory certification (ISO 17020, ISO 17025).
type AccreditationAdapter struct{}

func (a AccreditationAdapter) Pillar() jurisdictions.PillarType {
	return jurisdictions.PillarAccreditation
}

func (a AccreditationAdapter) Check(profile *jurisdictions.CountryProfile) (Result, error) {
	if err := checkPreconditions(profile, jurisdictions.PillarAccreditation); err != nil {
		return Result{}, err
	}

	result := Result{
		Pillar: jurisdictions.PillarAccreditation,
		ISO2:   profile.ISO2,
		Status: StatusPass,
	}

	if len(profile.SafetyRegulators) == 0 {
		result.Status = StatusFail
		result.Findings = append(result.Findings, Finding{
			Field:   "safety_regulators",
			Status:  StatusFail,
			Message: "no regulatory bodies registered — accreditation status unknown",
		})
		return result, nil
	}

	result.Findings = append(result.Findings, Finding{
		Field:   "safety_regulators.count",
		Status:  StatusPass,
		Message: fmt.Sprintf("%d regulatory body/bodies recognized", len(profile.SafetyRegulators)),
	})

	return result, nil
}
