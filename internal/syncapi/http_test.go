package syncapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/device_trust"
	domainsync "integin/internal/domain/sync"
	"integin/internal/identity"
	"integin/internal/oidchttp"
)

func TestHandlerAppliesAndDeduplicatesSignedTransaction(t *testing.T) {
	processor, authority := testProcessor(t)
	handler := NewHandler(processor)
	handler.Now = func() time.Time { return time.Date(2026, 8, 13, 11, 0, 0, 0, time.UTC) }
	handler.RegisterAuthority(authority)

	payload := []byte(`{"inspection_id":"inspection-1"}`)
	transaction := domainsync.NewTransaction("tx-1", "tenant-1", "device-1", "user-1", 1, "InspectionSubmitted", payload)
	transaction.OrganizationID = "org-1"
	transaction.EntityID = "inspection-1"
	transaction.AuthorityID = authority.ID
	transaction.AuthorityEpoch = authority.Epoch
	transaction = domainsync.SignTransaction(transaction, "secret", "default")
	body := requestBody(t, transaction, authority.ID)

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body)))
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, body = %s", first.Code, first.Body.String())
	}
	var firstResponse response
	if err := json.Unmarshal(first.Body.Bytes(), &firstResponse); err != nil {
		t.Fatal(err)
	}
	if firstResponse.Outcome != domainsync.Applied {
		t.Fatalf("first outcome = %s", firstResponse.Outcome)
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body)))
	var secondResponse response
	if err := json.Unmarshal(second.Body.Bytes(), &secondResponse); err != nil {
		t.Fatal(err)
	}
	if secondResponse.Outcome != domainsync.Duplicate {
		t.Fatalf("second outcome = %s", secondResponse.Outcome)
	}
}

func TestRevokedAuthorityStopsAuthorizingImmediately(t *testing.T) {
	registry := NewAuthorityRegistry()
	registry.Register(device_trust.AuthorityPackage{ID: "auth-live", DeviceID: "device-1"})
	if _, ok := registry.Get("auth-live"); !ok {
		t.Fatal("registered authority must resolve before revocation")
	}
	registry.Revoke("auth-live")
	if _, ok := registry.Get("auth-live"); ok {
		t.Fatal("revoked authority must not resolve")
	}
	if !registry.IsRevoked("auth-live") {
		t.Fatal("IsRevoked must report the tombstone")
	}
	// Replaying the identical package must not resurrect the grant.
	revoked, _ := registry.Get("auth-live")
	registry.Register(revoked)
	if _, ok := registry.Get("auth-live"); ok {
		t.Fatal("replaying the revoked package must not resurrect it")
	}
	// A genuinely reissued authority (different signature) clears the tombstone.
	fresh := device_trust.AuthorityPackage{ID: "auth-live", DeviceID: "device-1", Signature: "new-signature"}
	registry.Register(fresh)
	if _, ok := registry.Get("auth-live"); !ok {
		t.Fatal("a reissued authority must resolve again")
	}
}

func TestRevokeDeviceAuthoritiesCoversRuntimeIssuedPackages(t *testing.T) {
	registry := NewAuthorityRegistry()
	// One authority from a boot snapshot and one minted after boot, as local
	// provisioning does: both must be withdrawn.
	registry.Register(device_trust.AuthorityPackage{ID: "auth-boot", DeviceID: "device-1"})
	registry.Register(device_trust.AuthorityPackage{ID: "auth-runtime", DeviceID: "device-1"})
	registry.Register(device_trust.AuthorityPackage{ID: "auth-other", DeviceID: "device-2"})

	if revoked := registry.RevokeDeviceAuthorities("device-1"); revoked != 2 {
		t.Fatalf("revoked authority count = %d, want 2", revoked)
	}
	for _, id := range []string{"auth-boot", "auth-runtime"} {
		if _, ok := registry.Get(id); ok {
			t.Fatalf("%s must be withdrawn", id)
		}
	}
	if _, ok := registry.Get("auth-other"); !ok {
		t.Fatal("another device's authority must remain valid")
	}
}

func TestRevokedDeviceCannotSyncWithoutRestart(t *testing.T) {
	processor, authority := testProcessor(t)
	if !processor.DeviceRegistered("device-1") {
		t.Fatal("device must be registered before revocation")
	}
	processor.RevokeDevice("device-1")
	if processor.DeviceRegistered("device-1") {
		t.Fatal("revoked device must not remain registered")
	}

	transaction := domainsync.NewTransaction("tx-revoked", "tenant-1", "device-1", "user-1", 1, "InspectionSubmitted", []byte(`{"inspection_id":"inspection-1"}`))
	transaction.OrganizationID = "org-1"
	transaction.EntityID = "inspection-1"
	transaction.AuthorityID = authority.ID
	transaction.AuthorityEpoch = authority.Epoch
	transaction = domainsync.SignTransaction(transaction, "secret", "default")

	result := processor.Submit(transaction, authority, time.Date(2026, 8, 13, 11, 0, 0, 0, time.UTC))
	if result.Outcome != domainsync.SecurityFailure {
		t.Fatalf("revoked device outcome = %s, want %s", result.Outcome, domainsync.SecurityFailure)
	}
	if !strings.Contains(result.Reason, "not registered") {
		t.Fatalf("revoked device reason = %q, want an unregistered-device rejection", result.Reason)
	}
}

func TestHandlerRejectsUnknownAuthorityAndMalformedPayload(t *testing.T) {
	processor, authority := testProcessor(t)
	handler := NewHandler(processor)
	handler.RegisterAuthority(authority)

	unknown := []byte(`{"transaction_id":"tx-1","authority_id":"unknown"}`)
	record := httptest.NewRecorder()
	handler.ServeHTTP(record, httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(unknown)))
	if record.Code != http.StatusForbidden {
		t.Fatalf("unknown authority status = %d", record.Code)
	}

	malformed := []byte(`{"transaction_id":"tx-1","authority_id":"auth-1","payload":{"unterminated":}`)
	record = httptest.NewRecorder()
	handler.ServeHTTP(record, httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(malformed)))
	if record.Code != http.StatusBadRequest {
		t.Fatalf("malformed payload status = %d", record.Code)
	}
}

func testProcessor(t *testing.T) (*domainsync.Processor, device_trust.AuthorityPackage) {
	t.Helper()
	processor, err := domainsync.NewProcessor(map[string]string{"default": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	device, err := device_trust.NewDevice("device-1", "tenant-1", "org-1", "user-1", "public-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	processor.RegisterDevice(device)
	authority, err := device_trust.IssueAuthorityPackage(device, "auth-1", "secret", []string{"inspection.perform"}, time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC), 4*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return processor, authority
}

func requestBody(t *testing.T, transaction domainsync.Transaction, authorityID string) []byte {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(transaction.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(request{ProtocolVersion: transaction.ProtocolVersion, TransactionID: transaction.TransactionID, TenantID: transaction.TenantID, OrganizationID: transaction.OrganizationID, Environment: transaction.Environment, DeviceID: transaction.DeviceID, UserID: transaction.UserID, SequenceNumber: transaction.SequenceNumber, Operation: transaction.Operation, EntityID: transaction.EntityID, Payload: transaction.Payload, PayloadHash: transaction.PayloadHash, CapturedAt: transaction.CapturedAt, AuthorityID: authorityID, AuthorityEpoch: transaction.AuthorityEpoch, SignatureAlgorithm: transaction.SignatureAlgorithm, KeyID: transaction.KeyID, Signature: transaction.Signature})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestHandlerDrainsHeldTransactionsWhenSequenceArrives(t *testing.T) {
	processor, authority := testProcessor(t)
	handler := NewHandler(processor)
	handler.Now = func() time.Time { return time.Date(2026, 8, 13, 11, 0, 0, 0, time.UTC) }
	handler.RegisterAuthority(authority)

	payload := []byte(`{"finding":"pass"}`)

	// Send tx-2 first (sequence 2) -> should be HELD
	tx2 := domainsync.NewTransaction("tx-2", "tenant-1", "device-1", "user-1", 2, "FindingRecorded", payload)
	tx2.OrganizationID = "org-1"
	tx2.EntityID = "inspection-1"
	tx2.AuthorityID = authority.ID
	tx2.AuthorityEpoch = authority.Epoch
	tx2 = domainsync.SignTransaction(tx2, "secret", "default")
	body2 := requestBody(t, tx2, authority.ID)

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body2)))
	if rec2.Code != http.StatusOK {
		t.Fatalf("tx2 status = %d", rec2.Code)
	}
	var res2 response
	_ = json.Unmarshal(rec2.Body.Bytes(), &res2)
	if res2.Outcome != domainsync.Held {
		t.Fatalf("expected tx2 held, got %s", res2.Outcome)
	}

	// Now send tx-1 (sequence 1) -> should apply tx-1 AND automatically cascade drain tx-2
	tx1 := domainsync.NewTransaction("tx-1", "tenant-1", "device-1", "user-1", 1, "FindingRecorded", payload)
	tx1.OrganizationID = "org-1"
	tx1.EntityID = "inspection-1"
	tx1.AuthorityID = authority.ID
	tx1.AuthorityEpoch = authority.Epoch
	tx1 = domainsync.SignTransaction(tx1, "secret", "default")
	body1 := requestBody(t, tx1, authority.ID)

	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body1)))
	if rec1.Code != http.StatusOK {
		t.Fatalf("tx1 status = %d", rec1.Code)
	}
	var res1 response
	_ = json.Unmarshal(rec1.Body.Bytes(), &res1)
	if res1.Outcome != domainsync.Applied {
		t.Fatalf("expected tx1 applied, got %s", res1.Outcome)
	}

	// Verify processor state: tx-2 is no longer held, it was cascaded to applied!
	if len(processor.HeldTransactions()) != 0 {
		t.Fatalf("expected 0 held transactions after cascade drain, got %d", len(processor.HeldTransactions()))
	}
}

func TestHandlerRejectsUnknownFields(t *testing.T) {
	processor, authority := testProcessor(t)
	handler := NewHandler(processor)
	handler.RegisterAuthority(authority)

	// Inject unknown/smuggled field "malicious_extra"
	payload := `{"transaction_id":"tx-1","authority_id":"` + authority.ID + `","malicious_extra":"injected"}`
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(payload)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for unknown fields, got %d", rec.Code)
	}
}

func TestHandlerEnforcesBoundOrganizationContext(t *testing.T) {
	processor, authority := testProcessor(t)
	handler := NewHandler(processor)
	handler.Now = func() time.Time { return time.Date(2026, 8, 13, 11, 0, 0, 0, time.UTC) }
	handler.RegisterAuthority(authority)

	payload := []byte(`{"inspection_id":"inspection-1"}`)
	transaction := domainsync.NewTransaction("tx-1", "tenant-1", "device-1", "user-1", 1, "InspectionSubmitted", payload)
	transaction.OrganizationID = "org-1"
	transaction.EntityID = "inspection-1"
	transaction.AuthorityID = authority.ID
	transaction.AuthorityEpoch = authority.Epoch
	transaction = domainsync.SignTransaction(transaction, "secret", "default")
	body := requestBody(t, transaction, authority.ID)

	// Tenant mismatch
	reqMismatchTenant := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body))
	reqMismatchTenant = reqMismatchTenant.WithContext(oidchttp.WithOrganizationContext(reqMismatchTenant.Context(), identity.OrganizationContext{
		TenantID: "tenant-attacker",
		ActorID:  "user-1",
	}))
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, reqMismatchTenant)
	if rec1.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for tenant mismatch, got %d", rec1.Code)
	}

	// User mismatch
	reqMismatchUser := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body))
	reqMismatchUser = reqMismatchUser.WithContext(oidchttp.WithOrganizationContext(reqMismatchUser.Context(), identity.OrganizationContext{
		TenantID: "tenant-1",
		ActorID:  "user-attacker",
	}))
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, reqMismatchUser)
	if rec2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for user mismatch, got %d", rec2.Code)
	}

	// Matching context succeeds
	reqMatch := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body))
	reqMatch = reqMatch.WithContext(oidchttp.WithOrganizationContext(reqMatch.Context(), identity.OrganizationContext{
		TenantID: "tenant-1",
		ActorID:  "user-1",
	}))
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, reqMatch)
	if rec3.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for matched context, got %d: %s", rec3.Code, rec3.Body.String())
	}
}
