package server

import (
	"context"
	"testing"
	"time"

	"integin/internal/domain/workpackage"
	"integin/internal/shared/featureflags"
	"integin/internal/workpackageenforcement"
)

type preparationPackageResolver struct{ pkg workpackage.Package }

func (r preparationPackageResolver) GetApproved(_ context.Context, _, _, _ string, _ int) (workpackage.Package, error) {
	return r.pkg, nil
}

type preparationAssignmentResolver struct{}

func (preparationAssignmentResolver) GetCurrentAssignment(_ context.Context, _, _, _, _ string, _ time.Time) (workpackage.Assignment, error) {
	return workpackage.Assignment{}, nil
}

type preparationObserver struct{}

func (preparationObserver) ObserveWorkPackagePolicy(workpackageenforcement.PolicyObservation) {}

func TestNewPilotPreparedPreAcceptancePolicyIsNilUnlessExplicitlyPrepared(t *testing.T) {
	policy, err := NewPilotPreparedPreAcceptancePolicy(PilotPolicyPreparationConfig{PreparationState: featureflags.Disabled})
	if err != nil {
		t.Fatalf("disabled preparation error: %v", err)
	}
	if policy != nil {
		t.Fatal("disabled preparation returned policy")
	}
}

func TestPilotPolicyPreparationRejectsEnforcementActiveAndNonPilotScope(t *testing.T) {
	config := PilotPolicyPreparationConfig{RuntimeName: "acceptance", ListenAddress: "127.0.0.1:8080", PreparationState: featureflags.Enabled, EnforcementActive: true}
	if _, err := NewPilotPreparedPreAcceptancePolicy(config); err == nil {
		t.Fatal("unsafe preparation configuration was accepted")
	}
}

func TestPilotPolicyPreparationReturnsUnregisteredPolicyOnlyForCompletePilotConfiguration(t *testing.T) {
	validator, err := workpackageenforcement.NewValidator(preparationPackageResolver{})
	if err != nil {
		t.Fatalf("new validator: %v", err)
	}
	policy, err := NewPilotPreparedPreAcceptancePolicy(PilotPolicyPreparationConfig{
		RuntimeName:       IsolatedPilotRuntime,
		ListenAddress:     "127.0.0.1:18080",
		PreparationState:  featureflags.Enabled,
		EnforcementActive: false,
		Validator:         validator,
		Assignments:       preparationAssignmentResolver{},
		Observer:          preparationObserver{},
		Now:               time.Now,
	})
	if err != nil {
		t.Fatalf("complete pilot preparation error: %v", err)
	}
	if policy == nil {
		t.Fatal("complete pilot preparation returned nil policy")
	}
}

func TestPilotPolicyPreparationEnvironmentDefaultsDisabledAndRejectsEnforcement(t *testing.T) {
	t.Setenv(pilotPolicyPreparationEnvironment, "")
	t.Setenv(pilotPackageEnforcementEnvironment, "")
	policy, err := NewPilotPreparedPreAcceptancePolicyFromEnvironment(nil, nil, nil, nil, "", "")
	if err != nil {
		t.Fatalf("default environment error: %v", err)
	}
	if policy != nil {
		t.Fatal("default environment returned policy")
	}
	t.Setenv(pilotPackageEnforcementEnvironment, "enabled")
	if _, err := NewPilotPreparedPreAcceptancePolicyFromEnvironment(nil, nil, nil, nil, "", ""); err == nil {
		t.Fatal("enforcement-active environment was accepted")
	}
}
