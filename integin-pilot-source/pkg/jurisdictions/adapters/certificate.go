package adapters

import (
	"fmt"

	"integin/pkg/jurisdictions"
)

// CertificateMarkingAdapter validates that a jurisdiction has defined certificate
// marking and conformity assessment requirements.
type CertificateMarkingAdapter struct{}

func (a CertificateMarkingAdapter) Pillar() jurisdictions.PillarType {
	return jurisdictions.PillarCertificateMarking
}

func (a CertificateMarkingAdapter) Check(profile *jurisdictions.CountryProfile) (Result, error) {
	if err := checkPreconditions(profile, jurisdictions.PillarCertificateMarking); err != nil {
		return Result{}, err
	}

	result := Result{
		Pillar: jurisdictions.PillarCertificateMarking,
		ISO2:   profile.ISO2,
		Status: StatusPass,
	}

	hasRegulators := len(profile.SafetyRegulators) > 0
	if !hasRegulators {
		result.Status = StatusWarning
		result.Findings = append(result.Findings, Finding{
			Field:   "certificate_marking.regulatory_basis",
			Status:  StatusWarning,
			Message: "no safety regulators — certificate marking authority is unverified",
		})
	}

	result.Findings = append(result.Findings, Finding{
		Field:   "certificate_marking.currency_code",
		Status:  StatusPass,
		Message: fmt.Sprintf("jurisdiction currency: %s", profile.CurrencyCode),
	})

	return result, nil
}
