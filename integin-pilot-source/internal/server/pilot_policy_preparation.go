// Package server composes optional INTEGIN HTTP boundaries without registering routes.
package server

import (
	"errors"
	"os"
	"strings"
	"time"

	"integin/internal/domain/sync"
	"integin/internal/shared/featureflags"
	"integin/internal/workpackageenforcement"
)

// FlagPilotPolicyPreparation identifies source-level pilot preparation only. It
// does not authorize package enforcement in any runtime.
const FlagPilotPolicyPreparation featureflags.Key = "pilot_package_policy_preparation"

const (
	pilotPolicyPreparationEnvironment  = "INTEGIN_PILOT_PACKAGE_POLICY_PREPARATION"
	pilotPackageEnforcementEnvironment = "INTEGIN_PILOT_PACKAGE_ENFORCEMENT"
)

// PilotPolicyPreparationConfig is a fail-closed gate for constructing, but not
// registering, a pilot-only pre-acceptance policy.
type PilotPolicyPreparationConfig struct {
	RuntimeName       string
	ListenAddress     string
	PreparationState  featureflags.State
	EnforcementActive bool
	Validator         *workpackageenforcement.Validator
	Assignments       workpackageenforcement.CurrentAssignmentResolver
	Observer          workpackageenforcement.PolicyObservationSink
	Now               func() time.Time
}

// Validate refuses any non-pilot, incomplete, or enforcement-active configuration.
func (c PilotPolicyPreparationConfig) Validate() error {
	if strings.TrimSpace(c.RuntimeName) != IsolatedPilotRuntime {
		return errors.New("pilot policy preparation requires isolated pilot runtime")
	}
	if strings.TrimSpace(c.ListenAddress) != "127.0.0.1:18080" {
		return errors.New("pilot policy preparation requires isolated pilot loopback listener")
	}
	if c.PreparationState != featureflags.Enabled {
		return errors.New("pilot policy preparation requires explicit enabled preparation state")
	}
	if c.EnforcementActive {
		return errors.New("pilot policy preparation cannot enable package enforcement")
	}
	if c.Validator == nil || c.Assignments == nil || c.Observer == nil || c.Now == nil {
		return errors.New("pilot policy preparation dependencies are incomplete")
	}
	return nil
}

// NewPilotPreparedPreAcceptancePolicy returns nil while preparation is disabled.
// When explicitly prepared, it returns a policy that the caller must still not
// register on a processor until a separate activation decision is approved.
func NewPilotPreparedPreAcceptancePolicy(config PilotPolicyPreparationConfig) (sync.PreAcceptancePolicy, error) {
	if config.EnforcementActive {
		return nil, errors.New("pilot policy preparation cannot enable package enforcement")
	}
	if config.PreparationState != featureflags.Enabled {
		if config.Observer != nil {
			config.Observer.ObserveWorkPackagePolicy(workpackageenforcement.PolicyObservation{State: workpackageenforcement.PolicyObservationDisabled, Category: "policy_disabled"})
		}
		return nil, nil
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return workpackageenforcement.NewPreAcceptancePolicy(config.Validator, config.Assignments, config.Now, config.Observer)
}

// NewPilotPreparedPreAcceptancePolicyFromEnvironment reads only the explicit
// source-preparation gate. It constructs no policy unless preparation is enabled
// and never registers a policy on a processor.
func NewPilotPreparedPreAcceptancePolicyFromEnvironment(validator *workpackageenforcement.Validator, assignments workpackageenforcement.CurrentAssignmentResolver, observer workpackageenforcement.PolicyObservationSink, now func() time.Time, runtimeName, listenAddress string) (sync.PreAcceptancePolicy, error) {
	state := featureflags.Disabled
	if strings.EqualFold(strings.TrimSpace(os.Getenv(pilotPolicyPreparationEnvironment)), "enabled") {
		state = featureflags.Enabled
	}
	return NewPilotPreparedPreAcceptancePolicy(PilotPolicyPreparationConfig{
		RuntimeName:       runtimeName,
		ListenAddress:     listenAddress,
		PreparationState:  state,
		EnforcementActive: strings.EqualFold(strings.TrimSpace(os.Getenv(pilotPackageEnforcementEnvironment)), "enabled"),
		Validator:         validator,
		Assignments:       assignments,
		Observer:          observer,
		Now:               now,
	})
}
