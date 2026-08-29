package evidencehttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"integin/internal/domain/evidence"
	"integin/internal/evidenceregistration"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/workorderauth"
)

type validatorStub struct{ err error }

func (v validatorStub) Validate(context.Context, string) (oidcauth.Principal, error) {
	if v.err != nil {
		return oidcauth.Principal{}, v.err
	}
	return oidcauth.Principal{Issuer: "https://issuer.example", Subject: "subject-a"}, nil
}

type resolverStub struct{ err error }

func (r resolverStub) Resolve(context.Context, identity.PrincipalKey) (identity.Membership, error) {
	if r.err != nil {
		return identity.Membership{}, r.err
	}
	return identity.Membership{ActorID: "actor-a", TenantID: "tenant-a", OrganizationID: "org-a", WorkOrderRole: "inspector", Capabilities: []string{workorderauth.CapabilitySubmitPartial}}, nil
}

type serviceStub struct {
	command evidenceregistration.RegisterCommand
	err     error
	called  bool
}

func (s *serviceStub) Register(_ context.Context, command evidenceregistration.RegisterCommand) (evidenceregistration.RegistrationResult, error) {
	s.called = true
	s.command = command
	if s.err != nil {
		return evidenceregistration.RegistrationResult{}, s.err
	}
	return evidenceregistration.RegistrationResult{Metadata: evidence.Metadata{ID: command.EvidenceID, InspectionID: command.InspectionID}, Inserted: true}, nil
}

func TestRegistrationHandlerDerivesActorAndRejectsAuthorityFields(t *testing.T) {
	service := &serviceStub{}
	handler := Handler{Validator: validatorStub{}, Resolver: resolverStub{}, Service: service}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/evidence/metadata-registrations", bytes.NewReader(requestJSON(t, map[string]any{"tenant_id": "forged"})))
	request.Header.Set("Authorization", "Bearer token")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || service.called {
		t.Fatalf("authority-field response=%d called=%v", response.Code, service.called)
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/evidence/metadata-registrations", bytes.NewReader(requestJSON(t, map[string]any{})))
	request.Header.Set("Authorization", "Bearer token")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !service.called || service.command.Actor.TenantID != "tenant-a" || service.command.Actor.OrganizationID != "org-a" || service.command.Actor.ActorID != "actor-a" {
		t.Fatalf("success response=%d command=%#v", response.Code, service.command)
	}
}

func TestRegistrationHandlerMapsIdentityAndDomainFailuresWithoutMutation(t *testing.T) {
	service := &serviceStub{}
	for _, scenario := range []struct {
		name      string
		validator validatorStub
		resolver  resolverStub
		service   error
		want      int
	}{
		{name: "bad token", validator: validatorStub{err: errors.New("invalid")}, want: http.StatusUnauthorized},
		{name: "missing membership", resolver: resolverStub{err: errors.New("missing")}, want: http.StatusForbidden},
		{name: "assignment denied", service: errors.New("generic"), want: http.StatusBadRequest},
		{name: "object conflict", service: evidenceregistration.ErrObjectMismatch, want: http.StatusConflict},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			service = &serviceStub{err: scenario.service}
			handler := Handler{Validator: scenario.validator, Resolver: scenario.resolver, Service: service}
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/evidence/metadata-registrations", bytes.NewReader(requestJSON(t, map[string]any{})))
			request.Header.Set("Authorization", "Bearer token")
			handler.ServeHTTP(response, request)
			if response.Code != scenario.want {
				t.Fatalf("response=%d want=%d", response.Code, scenario.want)
			}
		})
	}
}

func requestJSON(t *testing.T, extra map[string]any) []byte {
	t.Helper()
	payload := map[string]any{"evidence_id": "evidence-a", "inspection_id": "inspection-a", "content_type": "application/octet-stream", "ciphertext_bytes": 10, "plaintext_sha256": hash("a"), "ciphertext_sha256": hash("b"), "captured_at": time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC), "device_id": "device-a", "authority_id": "authority-a", "authority_epoch": 1, "transaction_id": "transaction-a", "receipt_id": "receipt-a", "signature_algorithm": "Ed25519", "key_id": "key-a", "encryption_algorithm": "AES-256-GCM", "encryption_key_reference": "storage-key-a", "classification": "CONFIDENTIAL", "retention_reference": "retention-v1", "hold_state": "NONE", "redaction_policy_reference": "redaction-v1"}
	for key, value := range extra {
		payload[key] = value
	}
	result, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func hash(character string) string {
	result := ""
	for len(result) < 64 {
		result += character
	}
	return result[:64]
}
