package server

import (
	"context"
	"testing"
	"time"

	"integin/internal/domain/workpackage"
	"integin/internal/workpackageenforcement"
)

type compositionRepository struct{}

func (compositionRepository) GetApproved(_ context.Context, _, _, _ string, _ int) (workpackage.Package, error) {
	return workpackage.Package{}, nil
}

func (compositionRepository) GetCurrentAssignment(_ context.Context, _, _, _, _ string, _ time.Time) (workpackage.Assignment, error) {
	return workpackage.Assignment{}, nil
}

type compositionObserver struct{}

func (compositionObserver) ObserveWorkPackagePolicy(workpackageenforcement.PolicyObservation) {}

func TestPilotPolicyCompositionDefaultsToNoPolicyWithoutRepository(t *testing.T) {
	t.Setenv(pilotPolicyPreparationEnvironment, "")
	t.Setenv(pilotPackageEnforcementEnvironment, "")
	policy, err := NewPilotPreparedPreAcceptancePolicyCompositionFromEnvironment(nil, nil, nil, "", "")
	if err != nil {
		t.Fatalf("disabled composition error: %v", err)
	}
	if policy != nil {
		t.Fatal("disabled composition returned policy")
	}
}

func TestPilotPolicyCompositionFailsClosedForMissingRepositoryOrEnforcement(t *testing.T) {
	t.Setenv(pilotPolicyPreparationEnvironment, "enabled")
	t.Setenv(pilotPackageEnforcementEnvironment, "")
	if _, err := NewPilotPreparedPreAcceptancePolicyCompositionFromEnvironment(nil, compositionObserver{}, time.Now, IsolatedPilotRuntime, "127.0.0.1:18080"); err == nil {
		t.Fatal("prepared composition accepted missing repository")
	}
	t.Setenv(pilotPackageEnforcementEnvironment, "enabled")
	if _, err := NewPilotPreparedPreAcceptancePolicyCompositionFromEnvironment(compositionRepository{}, compositionObserver{}, time.Now, IsolatedPilotRuntime, "127.0.0.1:18080"); err == nil {
		t.Fatal("enforcement-active composition was accepted")
	}
}

func TestPilotPolicyCompositionPreparesUnregisteredPolicyForCompletePilotScope(t *testing.T) {
	t.Setenv(pilotPolicyPreparationEnvironment, "enabled")
	t.Setenv(pilotPackageEnforcementEnvironment, "")
	policy, err := NewPilotPreparedPreAcceptancePolicyCompositionFromEnvironment(compositionRepository{}, compositionObserver{}, time.Now, IsolatedPilotRuntime, "127.0.0.1:18080")
	if err != nil {
		t.Fatalf("complete pilot composition error: %v", err)
	}
	if policy == nil {
		t.Fatal("complete pilot composition returned nil policy")
	}
}
