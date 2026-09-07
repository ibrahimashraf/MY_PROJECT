package acceptance

import (
	"testing"
	"time"

	"integin/internal/domain/authorization"
	"integin/internal/domain/device_trust"
	syncengine "integin/internal/domain/sync"
	"integin/internal/shared/types"
)

func TestPhase3AuthorizedOfflineTransactionAndRevocation(t *testing.T) {
	device, err := device_trust.NewDevice("device-1", "tenant-1", "org-1", "user-1", "public-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	issuedAt := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	authority, err := device_trust.IssueAuthorityPackage(device, "authority-1", "secret", []string{"inspection-1"}, issuedAt, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	authorizationRequest := authorization.Request{
		Subject:     authorization.Subject{UserID: "user-1", TenantID: "tenant-1", OrganizationID: "org-1", Capabilities: map[authorization.Capability]bool{authorization.CapabilityInspectPerform: true}, Scopes: map[string]bool{"site-1": true}},
		Resource:    authorization.Resource{ID: "inspection-1", Type: "inspection", TenantID: "tenant-1", OrganizationID: "org-1", ScopeID: "site-1", WorkflowState: "ASSIGNED"},
		Action:      authorization.CapabilityInspectPerform,
		Environment: types.EnvironmentLive,
		Offline:     true,
		Device:      authorization.DeviceContext{ID: "device-1", TrustState: device.State(), AuthorityValid: true},
	}
	if decision := authorization.Evaluate(authorizationRequest); !decision.Allowed {
		t.Fatalf("expected authorized offline work: %#v", decision)
	}

	processor, err := syncengine.NewProcessor("secret")
	if err != nil {
		t.Fatal(err)
	}
	processor.RegisterDevice(device)
	transaction := syncengine.SignTransaction(syncengine.NewTransaction("tx-1", "tenant-1", "device-1", "user-1", 1, "InspectionStarted", []byte("inspection-1")), "secret")
	if result := processor.Submit(transaction, authority, issuedAt.Add(10*time.Minute)); result.Outcome != syncengine.Applied {
		t.Fatalf("expected applied transaction: %#v", result)
	}

	if err := device.Revoke("device reported lost"); err != nil {
		t.Fatal(err)
	}
	processor.RegisterDevice(device)
	authorizationRequest.Device.TrustState = device.State()
	authorizationRequest.Device.AuthorityValid = false
	if decision := authorization.Evaluate(authorizationRequest); decision.Allowed {
		t.Fatal("revoked device should not authorize offline work")
	}
	transaction2 := syncengine.SignTransaction(syncengine.NewTransaction("tx-2", "tenant-1", "device-1", "user-1", 2, "FindingRecorded", []byte("finding")), "secret")
	if result := processor.Submit(transaction2, authority, issuedAt.Add(20*time.Minute)); result.Outcome != syncengine.SecurityFailure {
		t.Fatalf("expected revoked sync security failure: %#v", result)
	}
}
