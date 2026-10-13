package adapters

import (
	"fmt"

	"integin/pkg/jurisdictions"
)

// CurrencyTradeAdapter validates currency configuration and trade compliance readiness.
type CurrencyTradeAdapter struct{}

func (a CurrencyTradeAdapter) Pillar() jurisdictions.PillarType {
	return jurisdictions.PillarCurrencyTrade
}

func (a CurrencyTradeAdapter) Check(profile *jurisdictions.CountryProfile) (Result, error) {
	if err := checkPreconditions(profile, jurisdictions.PillarCurrencyTrade); err != nil {
		return Result{}, err
	}

	result := Result{
		Pillar: jurisdictions.PillarCurrencyTrade,
		ISO2:   profile.ISO2,
		Status: StatusPass,
	}

	if len(profile.CurrencyCode) != 3 {
		result.Status = StatusFail
		result.Findings = append(result.Findings, Finding{
			Field:   "currency_code",
			Status:  StatusFail,
			Message: "currency code must be exactly 3 uppercase ISO 4217 letters",
		})
		return result, nil
	}

	result.Findings = append(result.Findings, Finding{
		Field:   "currency_code",
		Status:  StatusPass,
		Message: fmt.Sprintf("ISO 4217: %s", profile.CurrencyCode),
	})

	return result, nil
}
