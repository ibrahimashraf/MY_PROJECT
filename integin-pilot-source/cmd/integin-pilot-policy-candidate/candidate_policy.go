package main

import (
	"database/sql"
	"errors"
	"os"
	"strings"
	"time"

	domainsync "integin/internal/domain/sync"
	"integin/internal/server"
	"integin/internal/shared/featureflags"
	"integin/internal/workpackageenforcement"
	"integin/internal/workpackagepg"
)

// candidatePolicyObserver deliberately ignores all policy observations. The
// matrix runner obtains only bounded result categories and count deltas; it must
// not receive transaction, payload, signature, key, or fixture data.
type candidatePolicyObserver struct{}

func (candidatePolicyObserver) ObserveWorkPackagePolicy(workpackageenforcement.PolicyObservation) {}

// composePilotCandidatePolicy is callable only from the separate pilot candidate
// command. It is absent from normal server startup and requires a durable pilot
// repository. Policy preparation itself stays non-enforcing; attachment is
// delegated to the independently gated registration seam.
func composePilotCandidatePolicy(database *sql.DB, processor *domainsync.Processor, listenAddress string) (*server.PilotPolicyRegistration, error) {
	if database == nil {
		return nil, errors.New("pilot policy candidate requires durable database")
	}
	if processor == nil {
		return nil, errors.New("pilot policy candidate requires processor")
	}

	runtimeName := strings.TrimSpace(os.Getenv("INTEGIN_PILOT_RUNTIME"))
	if strings.TrimSpace(os.Getenv("INTEGIN_PILOT_PACKAGE_POLICY_PREPARATION")) != "enabled" {
		return nil, errors.New("pilot policy candidate preparation gate is not enabled")
	}
	repository, err := workpackagepg.NewRepository(database)
	if err != nil {
		return nil, err
	}
	validator, err := workpackageenforcement.NewValidator(repository)
	if err != nil {
		return nil, err
	}

	observer := candidatePolicyObserver{}
	policy, err := server.NewPilotPreparedPreAcceptancePolicy(server.PilotPolicyPreparationConfig{
		RuntimeName:       runtimeName,
		ListenAddress:     listenAddress,
		PreparationState:  featureflags.Enabled,
		EnforcementActive: false,
		Validator:         validator,
		Assignments:       repository,
		Observer:          observer,
		Now:               time.Now,
	})
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, errors.New("pilot policy candidate did not prepare policy")
	}

	return server.RegisterPilotPreAcceptancePolicyFromEnvironment(processor, policy, observer, runtimeName, listenAddress)
}
