// Package server composes optional INTEGIN HTTP boundaries without registering routes.
package server

import (
	"errors"
	"os"
	"strings"
	stdsync "sync"

	domainsync "integin/internal/domain/sync"
	"integin/internal/shared/featureflags"
	"integin/internal/workpackageenforcement"
)

const pilotPolicyRegistrationEnvironment = "INTEGIN_PILOT_PACKAGE_POLICY_REGISTRATION"

// PilotPolicyRegistrationConfig is a fail-closed contract for a future,
// isolated-pilot-only policy registration. It is not used by server startup.
type PilotPolicyRegistrationConfig struct {
	RuntimeName       string
	ListenAddress     string
	RegistrationState featureflags.State
	EnforcementActive bool
	Processor         *domainsync.Processor
	Policy            domainsync.PreAcceptancePolicy
	Observer          workpackageenforcement.PolicyObservationSink
}

// Validate accepts only a complete, explicitly enabled isolated-pilot policy
// registration. It rejects acceptance scope and every partial configuration.
func (c PilotPolicyRegistrationConfig) Validate() error {
	if strings.TrimSpace(c.RuntimeName) != IsolatedPilotRuntime {
		return errors.New("pilot policy registration requires isolated pilot runtime")
	}
	if strings.TrimSpace(c.ListenAddress) != "127.0.0.1:18080" {
		return errors.New("pilot policy registration requires isolated pilot loopback listener")
	}
	if c.RegistrationState != featureflags.Enabled {
		return errors.New("pilot policy registration requires explicit enabled registration state")
	}
	if !c.EnforcementActive {
		return errors.New("pilot policy registration requires explicit enforcement activation")
	}
	if c.Processor == nil || c.Policy == nil || c.Observer == nil {
		return errors.New("pilot policy registration dependencies are incomplete")
	}
	return nil
}

// PilotPolicyRegistration owns the one pilot policy registration it creates and
// exposes an idempotent rollback that removes only that policy from its processor.
type PilotPolicyRegistration struct {
	mu        stdsync.Mutex
	processor *domainsync.Processor
	observer  workpackageenforcement.PolicyObservationSink
	active    bool
}

// RegisterPilotPreAcceptancePolicy registers a policy only after the explicit
// configuration passes. It is intentionally uncalled by all current runtimes.
func RegisterPilotPreAcceptancePolicy(config PilotPolicyRegistrationConfig) (*PilotPolicyRegistration, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	config.Processor.SetPreAcceptancePolicy(config.Policy)
	config.Observer.ObserveWorkPackagePolicy(workpackageenforcement.PolicyObservation{State: workpackageenforcement.PolicyObservationPrepared, Category: "policy_registered"})
	return &PilotPolicyRegistration{processor: config.Processor, observer: config.Observer, active: true}, nil
}

// RegisterPilotPreAcceptancePolicyFromEnvironment is a disabled-by-default
// future activation seam. It cannot register a policy unless both explicit
// registration and enforcement environment gates are enabled.
func RegisterPilotPreAcceptancePolicyFromEnvironment(processor *domainsync.Processor, policy domainsync.PreAcceptancePolicy, observer workpackageenforcement.PolicyObservationSink, runtimeName, listenAddress string) (*PilotPolicyRegistration, error) {
	registrationRequested := strings.EqualFold(strings.TrimSpace(os.Getenv(pilotPolicyRegistrationEnvironment)), "enabled")
	enforcementRequested := strings.EqualFold(strings.TrimSpace(os.Getenv(pilotPackageEnforcementEnvironment)), "enabled")
	if !registrationRequested && !enforcementRequested {
		return nil, nil
	}
	if !registrationRequested || !enforcementRequested {
		return nil, errors.New("pilot policy registration and enforcement gates must be enabled together")
	}
	return RegisterPilotPreAcceptancePolicy(PilotPolicyRegistrationConfig{
		RuntimeName:       runtimeName,
		ListenAddress:     listenAddress,
		RegistrationState: featureflags.Enabled,
		EnforcementActive: true,
		Processor:         processor,
		Policy:            policy,
		Observer:          observer,
	})
}

// Disable removes this registration's policy from its processor. It is safe to
// call repeatedly and does not delete any package, receipt, assignment, or draft.
func (r *PilotPolicyRegistration) Disable() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.active {
		return
	}
	r.processor.SetPreAcceptancePolicy(nil)
	r.active = false
	r.observer.ObserveWorkPackagePolicy(workpackageenforcement.PolicyObservation{State: workpackageenforcement.PolicyObservationDisabled, Category: "policy_rolled_back"})
}

// Active reports whether this registration has not yet been rolled back.
func (r *PilotPolicyRegistration) Active() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active
}
