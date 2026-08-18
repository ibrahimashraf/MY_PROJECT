package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainsync "integin/internal/domain/sync"
	"integin/internal/server"
	"integin/internal/shared/featureflags"
	"integin/internal/workpackageenforcement"
)

type candidateTestPolicy struct{}

func (candidateTestPolicy) ValidatePreAcceptance(context.Context, domainsync.Transaction) error {
	return nil
}

type candidateTestObserver struct{}

func (candidateTestObserver) ObserveWorkPackagePolicy(workpackageenforcement.PolicyObservation) {}

func TestCandidateRollbackControlDisablesRegistrationBeforeShutdown(t *testing.T) {
	processor, err := domainsync.NewProcessor("candidate-rollback-test")
	if err != nil {
		t.Fatalf("new processor: %v", err)
	}
	registration, err := server.RegisterPilotPreAcceptancePolicy(server.PilotPolicyRegistrationConfig{
		RuntimeName:       server.IsolatedPilotRuntime,
		ListenAddress:     "127.0.0.1:18080",
		RegistrationState: featureflags.Enabled,
		EnforcementActive: true,
		Processor:         processor,
		Policy:            candidateTestPolicy{},
		Observer:          candidateTestObserver{},
	})
	if err != nil {
		t.Fatalf("register policy: %v", err)
	}

	shutdownCalled := make(chan struct{}, 1)
	handler := newCandidateRollbackHandler(http.NotFoundHandler(), registration, func(context.Context) error {
		if registration.Active() {
			t.Error("registration remained active when shutdown was invoked")
		}
		shutdownCalled <- struct{}{}
		return nil
	})

	request := httptest.NewRequest(http.MethodPost, candidateRollbackPath, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("rollback status = %d", response.Code)
	}
	if registration.Active() {
		t.Fatal("registration remained active after rollback control")
	}
	select {
	case <-shutdownCalled:
	case <-time.After(time.Second):
		t.Fatal("candidate shutdown was not scheduled after rollback")
	}
}
