package domain

import (
	"errors"

	"integin/pkg/rulesengine"
)

var ErrNilPassport = errors.New("asset passport is nil")

// ApplyProofLoadVerdict autonomously quarantines an asset when a proof-load
// failure is classified as CRITICAL_QUARANTINE. Returns true when a
// quarantine was applied.
func ApplyProofLoadVerdict(passport *UniversalAssetPassport, severity rulesengine.Severity, detail string) (bool, error) {
	if passport == nil {
		return false, ErrNilPassport
	}
	if severity == rulesengine.SeverityCriticalQuarantine {
		passport.Quarantine(detail)
		return true, nil
	}
	return false, nil
}
