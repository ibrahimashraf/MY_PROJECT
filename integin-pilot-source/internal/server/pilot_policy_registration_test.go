package server

import (
	"context"
	"testing"

	domainsync "integin/internal/domain/sync"
	"integin/internal/shared/featureflags"
	"integin/internal/workpackageenforcement"
)

type registrationPolicy struct{}

func (registrationPolicy) ValidatePreAcceptance(context.Context, domainsync.Transaction) error {
	return nil
}

type registrationObserver struct {
	observations []workpackageenforcement.PolicyObservation
}

func (o *registrationObserver) ObserveWorkPackagePolicy(observation workpackageenforcement.PolicyObservation) {
	o.observations = append(o.observations, observation)
}

func TestPilotPolicyRegistrationFailsClosedForUnsafeConfiguration(t *testing.T) {
	processor, err := domainsync.NewProcessor(map[string]string{"default": "test-secret"})
	if err != nil {
		t.Fatalf("new processor: %v", err)
	}
	config := PilotPolicyRegistrationConfig{
		RuntimeName:       "acceptance",
		ListenAddress:     "127.0.0.1:8080",
		RegistrationState: featureflags.Enabled,
		EnforcementActive: true,
		Processor:         processor,
		Policy:            registrationPolicy{},
		Observer:          &registrationObserver{},
	}
	if _, err := RegisterPilotPreAcceptancePolicy(config); err == nil {
		t.Fatal("unsafe registration configuration was accepted")
	}
}

func TestPilotPolicyRegistrationRegistersAndRollsBackIdempotently(t *testing.T) {
	processor, err := domainsync.NewProcessor(map[string]string{"default": "test-secret"})
	if err != nil {
		t.Fatalf("new processor: %v", err)
	}
	observer := &registrationObserver{}
	registration, err := RegisterPilotPreAcceptancePolicy(PilotPolicyRegistrationConfig{
		RuntimeName:       IsolatedPilotRuntime,
		ListenAddress:     "127.0.0.1:18080",
		RegistrationState: featureflags.Enabled,
		EnforcementActive: true,
		Processor:         processor,
		Policy:            registrationPolicy{},
		Observer:          observer,
	})
	if err != nil {
		t.Fatalf("register pilot policy: %v", err)
	}
	if !registration.Active() {
		t.Fatal("registration was not active")
	}
	registration.Disable()
	registration.Disable()
	if registration.Active() {
		t.Fatal("registration remained active after rollback")
	}
	if len(observer.observations) != 2 || observer.observations[0].Category != "policy_registered" || observer.observations[1].Category != "policy_rolled_back" {
		t.Fatalf("observations = %#v", observer.observations)
	}
}

func TestPilotPolicyRegistrationEnvironmentDefaultsDisabled(t *testing.T) {
	t.Setenv(pilotPolicyRegistrationEnvironment, "")
	t.Setenv(pilotPackageEnforcementEnvironment, "")
	registration, err := RegisterPilotPreAcceptancePolicyFromEnvironment(nil, nil, nil, "", "")
	if err != nil {
		t.Fatalf("default registration error: %v", err)
	}
	if registration != nil {
		t.Fatal("default registration returned active registration")
	}
	t.Setenv(pilotPolicyRegistrationEnvironment, "enabled")
	if _, err := RegisterPilotPreAcceptancePolicyFromEnvironment(nil, nil, nil, "", ""); err == nil {
		t.Fatal("partial registration environment was accepted")
	}
}
