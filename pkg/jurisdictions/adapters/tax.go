package adapters

import (
	"fmt"
	"regexp"
	"strings"

	"integin/pkg/jurisdictions"
)

// TaxFiscalAdapter validates tax authority configuration and ID format compliance.
// Ensures tax IDs can be validated against jurisdiction-specific regex patterns.
type TaxFiscalAdapter struct{}

func (a TaxFiscalAdapter) Pillar() jurisdictions.PillarType {
	return jurisdictions.PillarTaxFiscal
}

func (a TaxFiscalAdapter) Check(profile *jurisdictions.CountryProfile) (Result, error) {
	if err := checkPreconditions(profile, jurisdictions.PillarTaxFiscal); err != nil {
		return Result{}, err
	}

	result := Result{
		Pillar: jurisdictions.PillarTaxFiscal,
		ISO2:   profile.ISO2,
		Status: StatusPass,
	}

	ta := profile.TaxAuthority

	if strings.TrimSpace(ta.Name) == "" {
		result.Status = StatusBlocking
		result.Findings = append(result.Findings, Finding{
			Field:   "tax_authority.name",
			Status:  StatusBlocking,
			Message: "no tax authority configured — fiscal compliance cannot be enforced",
		})
		return result, nil
	}

	result.Findings = append(result.Findings, Finding{
		Field:   "tax_authority.name",
		Status:  StatusPass,
		Message: ta.Name,
	})

	if ta.TaxIDRegex != "" {
		re, err := regexp.Compile(ta.TaxIDRegex)
		if err != nil {
			result.Status = StatusFail
			result.Findings = append(result.Findings, Finding{
				Field:   "tax_authority.tax_id_regex",
				Status:  StatusFail,
				Message: fmt.Sprintf("tax ID regex does not compile: %v", err),
			})
			return result, nil
		}

		testIDs := testTaxIDs(profile.ISO2)
		for _, tid := range testIDs {
			if !re.MatchString(tid) {
				result.Findings = append(result.Findings, Finding{
					Field:   "tax_authority.tax_id_regex",
					Status:  StatusWarning,
					Message: fmt.Sprintf("example ID %q does not match pattern %s", tid, ta.TaxIDRegex),
				})
			}
		}
	}

	if strings.TrimSpace(ta.CommercialReg) == "" {
		result.Findings = append(result.Findings, Finding{
			Field:   "tax_authority.commercial_reg_label",
			Status:  StatusWarning,
			Message: "commercial registration label is not set",
		})
	}

	return result, nil
}

func testTaxIDs(iso2 string) []string {
	switch strings.ToUpper(iso2) {
	case "SA":
		return []string{"310123456789012"}
	case "AE":
		return []string{"100234567890123"}
	case "US":
		return []string{"12-3456789"}
	case "GB":
		return []string{"123456789"}
	case "DE":
		return []string{"DE123456789"}
	case "SG":
		return []string{"201234567A"}
	case "AU":
		return []string{"12345678901"}
	case "NO":
		return []string{"123456789"}
	default:
		return nil
	}
}
