package adapters

import (
	"fmt"

	"integin/pkg/jurisdictions"
)

// SafetyRegulationAdapter validates that a jurisdiction has the required safety
// regulatory bodies and inspectorates for industrial inspection work. This is the
// most critical pillar for life-safety compliance.
type SafetyRegulationAdapter struct{}

func (a SafetyRegulationAdapter) Pillar() jurisdictions.PillarType {
	return jurisdictions.PillarSafetyRegulation
}

// Check validates that the jurisdiction has at least one safety regulator registered
// and that regulator entries have names. Returns BLOCKING if no regulators exist,
// FAIL if any regulator entry is malformed.
func (a SafetyRegulationAdapter) Check(profile *jurisdictions.CountryProfile) (Result, error) {
	if err := checkPreconditions(profile, jurisdictions.PillarSafetyRegulation); err != nil {
		return Result{}, err
	}

	result := Result{
		Pillar: jurisdictions.PillarSafetyRegulation,
		ISO2:   profile.ISO2,
		Status: StatusPass,
	}

	if len(profile.SafetyRegulators) == 0 {
		result.Status = StatusBlocking
		result.Findings = append(result.Findings, Finding{
			Field:   "safety_regulators",
			Status:  StatusBlocking,
			Message: "jurisdiction has no registered safety regulators — industrial inspection not legally permitted",
		})
		return result, nil
	}

	for i, reg := range profile.SafetyRegulators {
		field := fmt.Sprintf("safety_regulators[%d]", i)
		if reg.Name == "" {
			result.Status = StatusFail
			result.Findings = append(result.Findings, Finding{
				Field:   field + ".name",
				Status:  StatusFail,
				Message: "safety regulator entry has empty name",
			})
		} else {
			result.Findings = append(result.Findings, Finding{
				Field:     field,
				Status:    StatusPass,
				Message:   reg.Name,
				Regulator: reg.Name,
			})
		}
	}

	return result, nil
}
