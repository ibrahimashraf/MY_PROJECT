// Package server composes optional INTEGIN HTTP boundaries without registering routes.
package server

import (
	"errors"
	"os"
	"strings"
	"time"

	"integin/internal/domain/sync"
	"integin/internal/workpackageenforcement"
)

// PilotPolicyRepository combines the two server-held reads required to prepare
// a work-package policy. The interface deliberately excludes sync.Processor so
// this composition seam cannot register or activate a policy.
type PilotPolicyRepository interface {
	workpackageenforcement.Resolver
	workpackageenforcement.CurrentAssignmentResolver
}

// NewPilotPreparedPreAcceptancePolicyCompositionFromEnvironment prepares a
// policy only for a complete isolated-pilot configuration. A disabled
// preparation gate returns nil. This function never calls SetPreAcceptancePolicy
// and therefore is not an enforcement activation path.
func NewPilotPreparedPreAcceptancePolicyCompositionFromEnvironment(repository PilotPolicyRepository, observer workpackageenforcement.PolicyObservationSink, now func() time.Time, runtimeName, listenAddress string) (sync.PreAcceptancePolicy, error) {
	preparationRequested := strings.EqualFold(strings.TrimSpace(os.Getenv(pilotPolicyPreparationEnvironment)), "enabled")
	enforcementRequested := strings.EqualFold(strings.TrimSpace(os.Getenv(pilotPackageEnforcementEnvironment)), "enabled")
	if !preparationRequested || enforcementRequested {
		return NewPilotPreparedPreAcceptancePolicyFromEnvironment(nil, nil, observer, now, runtimeName, listenAddress)
	}
	if repository == nil {
		return nil, errors.New("pilot policy composition requires repository when preparation is enabled")
	}
	validator, err := workpackageenforcement.NewValidator(repository)
	if err != nil {
		return nil, err
	}
	return NewPilotPreparedPreAcceptancePolicyFromEnvironment(validator, repository, observer, now, runtimeName, listenAddress)
}
