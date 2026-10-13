package server

import (
	"context"
	"testing"
	"time"

	"integin/internal/certificatehttp"
	"integin/internal/certificatepg"
	"integin/internal/domain/certificate"
	"integin/internal/domain/certificateauthority"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/storage"
)

type compositionTestLifecycle struct{}

func (compositionTestLifecycle) CreateDraft(context.Context, certificateauthority.ActorContext, certificateauthority.CreateDraftRequest, time.Time) (*certificateauthority.Certificate, error) {
	return nil, nil
}
func (compositionTestLifecycle) Submit(context.Context, certificateauthority.ActorContext, string, time.Time) error {
	return nil
}
func (compositionTestLifecycle) Review(context.Context, certificateauthority.ActorContext, string, time.Time) error {
	return nil
}
func (compositionTestLifecycle) Sign(context.Context, certificateauthority.ActorContext, string, certificate.SignatureEvent, time.Time) error {
	return nil
}
func (compositionTestLifecycle) Attest(context.Context, certificateauthority.ActorContext, string, string, string, string, string, time.Time) error {
	return nil
}
func (compositionTestLifecycle) WaiveSign(context.Context, certificateauthority.ActorContext, string, string, string, string, string, time.Time) error {
	return nil
}
func (compositionTestLifecycle) Issue(context.Context, certificateauthority.ActorContext, string, time.Time) (certificatepg.IssueResult, error) {
	return certificatepg.IssueResult{}, nil
}
func (compositionTestLifecycle) Revoke(context.Context, certificateauthority.ActorContext, string, string, time.Time) error {
	return nil
}
func (compositionTestLifecycle) Renew(context.Context, certificateauthority.ActorContext, string, string, time.Time) error {
	return nil
}
func (compositionTestLifecycle) Supersede(context.Context, certificateauthority.ActorContext, string, string, time.Time) error {
	return nil
}
func (compositionTestLifecycle) Expire(context.Context, certificateauthority.ActorContext, string, time.Time) error {
	return nil
}

func TestNewCertificateHandlerFailsClosedOnMissingDependency(t *testing.T) {
	resolver := compositionTestResolver{}
	validator := &oidcauth.Validator{}
	lifecycle := compositionTestLifecycle{}

	tests := []struct {
		name      string
		lifecycle certificatehttp.Lifecycle
		validator *oidcauth.Validator
		resolver  identity.Resolver
	}{
		{name: "missing lifecycle", lifecycle: nil, validator: validator, resolver: resolver},
		{name: "missing validator", lifecycle: lifecycle, validator: nil, resolver: resolver},
		{name: "missing resolver", lifecycle: lifecycle, validator: validator, resolver: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if handler, err := NewCertificateHandlerFromLifecycle(test.lifecycle, test.validator, test.resolver); handler != nil || err == nil {
				t.Fatalf("expected dependency failure, handler=%T err=%v", handler, err)
			}
		})
	}
}

func TestNewCertificateHandlerBuildsWithoutConnecting(t *testing.T) {
	handler, err := NewCertificateHandlerFromLifecycle(compositionTestLifecycle{}, &oidcauth.Validator{}, compositionTestResolver{})
	if err != nil {
		t.Fatalf("expected local composition to succeed without connecting, got %v", err)
	}
	if handler == nil {
		t.Fatal("expected composed certificate handler")
	}
}

func TestNewCertificateHandlerWithArtifactStore(t *testing.T) {
	store := storage.NewInMemoryStore()
	handler, err := NewCertificateHandlerFromLifecycleWithStore(compositionTestLifecycle{}, &oidcauth.Validator{}, compositionTestResolver{}, store)
	if err != nil {
		t.Fatalf("expected composition with store to succeed, got %v", err)
	}
	if handler == nil {
		t.Fatal("expected composed certificate handler")
	}
}
