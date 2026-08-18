// Command integin-pilot-policy-probe exercises only the isolated pilot policy
// registration lifecycle. It does not listen on a port, load configuration
// files, access a database, or compose the normal server startup path.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	domainsync "integin/internal/domain/sync"
	"integin/internal/server"
	"integin/internal/workpackageenforcement"
)

const candidateAddress = "127.0.0.1:18080"

// lifecyclePolicy is intentionally inert. The probe validates registration and
// rollback plumbing; authoritative work-package decisions remain in the tested
// prepared policy and are not reimplemented here.
type lifecyclePolicy struct{}

func (lifecyclePolicy) ValidatePreAcceptance(context.Context, domainsync.Transaction) error {
	return nil
}

// lifecycleObserver retains only fixed policy lifecycle categories. It stores
// no payload, signature, credential, fixture, or transaction material.
type lifecycleObserver struct {
	categories []string
}

func (observer *lifecycleObserver) ObserveWorkPackagePolicy(observation workpackageenforcement.PolicyObservation) {
	observer.categories = append(observer.categories, observation.Category)
}

func main() {
	runtimeName := strings.TrimSpace(os.Getenv("INTEGIN_PILOT_RUNTIME"))
	if runtimeName != server.IsolatedPilotRuntime {
		fail(errors.New("pilot policy probe requires isolated pilot runtime"))
	}

	processor, err := domainsync.NewProcessor("pilot-policy-registration-probe")
	if err != nil {
		fail(fmt.Errorf("create isolated probe processor: %w", err))
	}

	observer := &lifecycleObserver{}
	registration, err := server.RegisterPilotPreAcceptancePolicyFromEnvironment(
		processor,
		lifecyclePolicy{},
		observer,
		runtimeName,
		candidateAddress,
	)
	if err != nil {
		fail(fmt.Errorf("register pilot policy: %w", err))
	}
	if registration == nil || !registration.Active() {
		fail(errors.New("pilot policy registration was not active"))
	}

	registration.Disable()
	registration.Disable()
	if registration.Active() {
		fail(errors.New("pilot policy registration remained active after rollback"))
	}
	if len(observer.categories) != 2 || observer.categories[0] != "policy_registered" || observer.categories[1] != "policy_rolled_back" {
		fail(errors.New("pilot policy lifecycle observations were incomplete"))
	}

	fmt.Println("PILOT_POLICY_REGISTRATION_PROBE=passed")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
