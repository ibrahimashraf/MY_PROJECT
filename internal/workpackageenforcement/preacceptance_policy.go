// Package workpackageenforcement validates signed inspection payloads against
// server-approved work packages.
package workpackageenforcement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"integin/internal/domain/sync"
	"integin/internal/domain/workpackage"
)

const inspectionSubmissionOperation = "InspectionSubmitted"

var (
	ErrAssignmentBinding = errors.New("inspection payload does not match the current work-package assignment")
	ErrPolicyScope       = errors.New("verified transaction is missing work-package policy scope")
)

type CurrentAssignmentResolver interface {
	GetCurrentAssignment(ctx context.Context, tenantID, organizationID, inspectionID, deviceID string, now time.Time) (workpackage.Assignment, error)
}

type PolicyObservationState string

const (
	PolicyObservationPrepared  PolicyObservationState = "prepared"
	PolicyObservationEvaluated PolicyObservationState = "evaluated"
	PolicyObservationAllowed   PolicyObservationState = "allowed"
	PolicyObservationRejected  PolicyObservationState = "rejected"
	PolicyObservationDisabled  PolicyObservationState = "disabled"
)

type PolicyObservation struct {
	State    PolicyObservationState
	Category string
}

func (o PolicyObservation) Validate() error {
	switch o.State {
	case PolicyObservationPrepared, PolicyObservationEvaluated, PolicyObservationAllowed, PolicyObservationRejected, PolicyObservationDisabled:
	default:
		return errors.New("work-package policy observation state is invalid")
	}
	switch o.Category {
	case "policy_prepared", "policy_disabled", "policy_registered", "policy_rolled_back", "inspection_submission", "missing_verified_scope", "assignment_unavailable", "malformed_payload", "assignment_mismatch", "package_validation_failed", "binding_valid":
	default:
		return errors.New("work-package policy observation category is invalid")
	}
	return nil
}

type PolicyObservationSink interface {
	ObserveWorkPackagePolicy(PolicyObservation)
}

type PreAcceptancePolicy struct {
	validator   *Validator
	assignments CurrentAssignmentResolver
	now         func() time.Time
	observer    PolicyObservationSink
}

func NewPreAcceptancePolicy(validator *Validator, assignments CurrentAssignmentResolver, now func() time.Time, observer PolicyObservationSink) (*PreAcceptancePolicy, error) {
	if validator == nil {
		return nil, errors.New("work-package pre-acceptance policy requires validator")
	}
	if assignments == nil {
		return nil, errors.New("work-package pre-acceptance policy requires assignment resolver")
	}
	if now == nil {
		return nil, errors.New("work-package pre-acceptance policy requires clock")
	}
	if observer == nil {
		return nil, errors.New("work-package pre-acceptance policy requires observation sink")
	}
	policy := &PreAcceptancePolicy{validator: validator, assignments: assignments, now: now, observer: observer}
	policy.observe(PolicyObservation{State: PolicyObservationPrepared, Category: "policy_prepared"})
	return policy, nil
}

func (p *PreAcceptancePolicy) ValidatePreAcceptance(ctx context.Context, transaction sync.Transaction) error {
	if p == nil || transaction.Operation != inspectionSubmissionOperation {
		return nil
	}
	p.observe(PolicyObservation{State: PolicyObservationEvaluated, Category: "inspection_submission"})
	if strings.TrimSpace(transaction.TenantID) == "" || strings.TrimSpace(transaction.OrganizationID) == "" || strings.TrimSpace(transaction.EntityID) == "" || strings.TrimSpace(transaction.DeviceID) == "" {
		return p.reject("missing_verified_scope", ErrPolicyScope)
	}
	assignment, err := p.assignments.GetCurrentAssignment(ctx, transaction.TenantID, transaction.OrganizationID, transaction.EntityID, transaction.DeviceID, p.now().UTC())
	if err != nil {
		return p.reject("assignment_unavailable", fmt.Errorf("resolve current work-package assignment: %w", err))
	}
	var reference struct {
		InspectionID   string `json:"inspection_id"`
		PackageID      string `json:"work_package_id"`
		PackageVersion int    `json:"work_package_version"`
	}
	if err := json.Unmarshal(transaction.Payload, &reference); err != nil {
		return p.reject("malformed_payload", fmt.Errorf("decode inspection work-package reference: %w", err))
	}
	if reference.InspectionID != transaction.EntityID || reference.PackageID != assignment.PackageID || reference.PackageVersion != assignment.PackageVersion || assignment.AuthorityEpoch <= 0 || transaction.AuthorityEpoch != uint64(assignment.AuthorityEpoch) {
		return p.reject("assignment_mismatch", ErrAssignmentBinding)
	}
	if err := p.validator.ValidateInspectionPayload(ctx, transaction.TenantID, transaction.OrganizationID, transaction.EntityID, transaction.Payload); err != nil {
		return p.reject("package_validation_failed", err)
	}
	p.observe(PolicyObservation{State: PolicyObservationAllowed, Category: "binding_valid"})
	return nil
}

func (p *PreAcceptancePolicy) reject(category string, err error) error {
	p.observe(PolicyObservation{State: PolicyObservationRejected, Category: category})
	return err
}

func (p *PreAcceptancePolicy) observe(observation PolicyObservation) {
	if p == nil || p.observer == nil || observation.Validate() != nil {
		return
	}
	p.observer.ObserveWorkPackagePolicy(observation)
}
