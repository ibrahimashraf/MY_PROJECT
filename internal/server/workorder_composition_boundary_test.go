package server

import (
	"context"
	"testing"

	"integin/internal/domain/workorder"
	"integin/internal/identity"
	"integin/internal/oidcauth"
)

type compositionTestResolver struct{}

func (compositionTestResolver) Resolve(context.Context, identity.PrincipalKey) (identity.Membership, error) {
	return identity.Membership{}, nil
}

type compositionTestWorkOrderService struct{}

func (compositionTestWorkOrderService) CreateRequest(context.Context, workorder.CreateRequestCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (compositionTestWorkOrderService) AssignScope(context.Context, workorder.AssignScopeCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (compositionTestWorkOrderService) TransitionExecution(context.Context, workorder.TransitionExecutionCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (compositionTestWorkOrderService) SubmitPartial(context.Context, workorder.SubmitPartialCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (compositionTestWorkOrderService) ReassignScope(context.Context, workorder.ReassignScopeCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (compositionTestWorkOrderService) ReconcileProvisional(context.Context, workorder.ReconcileProvisionalCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (compositionTestWorkOrderService) RequestCertificateValidation(context.Context, workorder.RequestCertificateValidationCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}
func (compositionTestWorkOrderService) AddEvidenceReference(context.Context, workorder.AddEvidenceReferenceCommand) (workorder.MutationReceipt, error) {
	return workorder.MutationReceipt{}, nil
}

func TestNewWorkOrderPartialSubmissionHandlerFailsClosedOnMissingDependency(t *testing.T) {
	resolver := compositionTestResolver{}
	validator := &oidcauth.Validator{}
	service := compositionTestWorkOrderService{}

	tests := []struct {
		name      string
		service   workorder.Service
		validator *oidcauth.Validator
		resolver  identity.Resolver
	}{
		{name: "missing service", service: nil, validator: validator, resolver: resolver},
		{name: "missing validator", service: service, validator: nil, resolver: resolver},
		{name: "missing resolver", service: service, validator: validator, resolver: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if handler, err := NewWorkOrderPartialSubmissionHandlerFromService(test.service, test.validator, test.resolver); handler != nil || err == nil {
				t.Fatalf("expected dependency failure, handler=%T err=%v", handler, err)
			}
		})
	}
}

func TestNewWorkOrderPartialSubmissionHandlerBuildsWithoutConnecting(t *testing.T) {
	handler, err := NewWorkOrderPartialSubmissionHandlerFromService(compositionTestWorkOrderService{}, &oidcauth.Validator{}, compositionTestResolver{})
	if err != nil {
		t.Fatalf("expected local composition to succeed without connecting, got %v", err)
	}
	if handler == nil {
		t.Fatal("expected composed handler")
	}
}

func TestNewWorkOrderHandoverHandlerFailsClosedOnMissingDependency(t *testing.T) {
	resolver := compositionTestResolver{}
	validator := &oidcauth.Validator{}
	service := compositionTestWorkOrderService{}

	tests := []struct {
		name      string
		service   workorder.Service
		validator *oidcauth.Validator
		resolver  identity.Resolver
	}{
		{name: "missing service", service: nil, validator: validator, resolver: resolver},
		{name: "missing validator", service: service, validator: nil, resolver: resolver},
		{name: "missing resolver", service: service, validator: validator, resolver: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if handler, err := NewWorkOrderHandoverHandlerFromService(test.service, test.validator, test.resolver); handler != nil || err == nil {
				t.Fatalf("expected dependency failure, handler=%T err=%v", handler, err)
			}
		})
	}
}

func TestNewWorkOrderHandoverHandlerBuildsWithoutConnecting(t *testing.T) {
	handler, err := NewWorkOrderHandoverHandlerFromService(compositionTestWorkOrderService{}, &oidcauth.Validator{}, compositionTestResolver{})
	if err != nil {
		t.Fatalf("expected local composition to succeed without connecting, got %v", err)
	}
	if handler == nil {
		t.Fatal("expected composed handler")
	}
}

func TestNewWorkOrderAssignmentHandlerFailsClosedOnMissingDependency(t *testing.T) {
	resolver := compositionTestResolver{}
	validator := &oidcauth.Validator{}
	service := compositionTestWorkOrderService{}

	tests := []struct {
		name      string
		service   workorder.Service
		validator *oidcauth.Validator
		resolver  identity.Resolver
	}{
		{name: "missing service", service: nil, validator: validator, resolver: resolver},
		{name: "missing validator", service: service, validator: nil, resolver: resolver},
		{name: "missing resolver", service: service, validator: validator, resolver: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if handler, err := NewWorkOrderAssignmentHandlerFromService(test.service, test.validator, test.resolver); handler != nil || err == nil {
				t.Fatalf("expected dependency failure, handler=%T err=%v", handler, err)
			}
		})
	}
}

func TestNewWorkOrderAssignmentHandlerBuildsWithoutConnecting(t *testing.T) {
	handler, err := NewWorkOrderAssignmentHandlerFromService(compositionTestWorkOrderService{}, &oidcauth.Validator{}, compositionTestResolver{})
	if err != nil {
		t.Fatalf("expected local composition to succeed without connecting, got %v", err)
	}
	if handler == nil {
		t.Fatal("expected composed handler")
	}
}

