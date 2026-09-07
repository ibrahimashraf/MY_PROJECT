package syncapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"integin/internal/domain/device_trust"
	domainsync "integin/internal/domain/sync"
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
	transaction = domainsync.SignTransaction(transaction, "secret")
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
	processor, err := domainsync.NewProcessor("secret")
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
