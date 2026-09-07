package adapters

import (
	"errors"
	"fmt"
	"integin/pkg/jurisdictions"
)

// ComplianceStatus represents the outcome of a compliance check.
type ComplianceStatus string

const (
	StatusPass          ComplianceStatus = "PASS"
	StatusFail          ComplianceStatus = "FAIL"
	StatusBlocking      ComplianceStatus = "BLOCKING"
	StatusWarning       ComplianceStatus = "WARNING"
	StatusNotApplicable ComplianceStatus = "NOT_APPLICABLE"
)

// Finding is a single compliance finding within a pillar check.
type Finding struct {
	Field     string           `json:"field"`
	Status    ComplianceStatus `json:"status"`
	Message   string           `json:"message"`
	Regulator string           `json:"regulator,omitempty"`
}

// Result is the aggregate outcome of a pillar adapter check.
type Result struct {
	Pillar   jurisdictions.PillarType `json:"pillar"`
	ISO2     string                   `json:"iso2"`
	Status   ComplianceStatus         `json:"status"`
	Findings []Finding                `json:"findings"`
}

// HasBlocking reports whether any finding is BLOCKING.
func (r Result) HasBlocking() bool {
	for _, f := range r.Findings {
		if f.Status == StatusBlocking {
			return true
		}
	}
	return false
}

// HasFailures reports whether any finding is FAIL or BLOCKING.
func (r Result) HasFailures() bool {
	for _, f := range r.Findings {
		if f.Status == StatusFail || f.Status == StatusBlocking {
			return true
		}
	}
	return false
}

var (
	ErrNilProfile        = errors.New("jurisdiction profile cannot be nil")
	ErrUnsupportedPillar = errors.New("adapter does not support this pillar type")
)

// Adapter is the common interface for all 7 sovereign compliance pillar adapters.
// Each implementation validates a CountryProfile against its pillar's rules
// and returns a Result with granular findings.
type Adapter interface {
	Pillar() jurisdictions.PillarType
	Check(profile *jurisdictions.CountryProfile) (Result, error)
}

// checkPreconditions validates the profile is non-nil and the jurisdiction
// declares this pillar as active.
func checkPreconditions(profile *jurisdictions.CountryProfile, pillar jurisdictions.PillarType) error {
	if profile == nil {
		return ErrNilProfile
	}
	if !profile.HasPillar(pillar) {
		return fmt.Errorf("jurisdiction %s does not declare pillar %s", profile.ISO2, pillar)
	}
	return nil
}
