package workbenchhttp

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"integin/internal/domain/device_trust"
)

const (
	testTenant = "tenant-1"
	testOrg    = "org-1"
)

func fixedClock() time.Time {
	return time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
}

func newHarness(t *testing.T, seedDevices []device_trust.Device) (*Handler, *memoryStore) {
	t.Helper()
	h := NewHandler(nil, seedDevices)
	h.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	mem := h.store.(*memoryStore)
	mem.now = fixedClock
	return h, mem
}

func seedTrustedDevice(t *testing.T, store *memoryStore, deviceID string) {
	t.Helper()
	device, err := device_trust.NewDevice(deviceID, testTenant, testOrg, "user-1", "ed25519-public-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	store.devices[deviceID] = &DeviceView{
		ID: device.ID(), TenantID: device.TenantID(), OrganizationID: device.OrganizationID(),
		UserID: device.UserID(), PublicKey: device.PublicKey(), State: string(device.State()),
		Epoch: device.Epoch(), EnrolledAt: fixedClock(),
	}
}

func seedEnrollment(store *memoryStore, requestID, deviceID string) *EnrollmentRequestView {
	request := &EnrollmentRequestView{
		RequestID: requestID, TenantID: testTenant, OrganizationID: testOrg,
		UserID: "user-1", DeviceID: deviceID, PublicKey: "ed25519-public-key",
		Status: "PENDING", RequestedAt: fixedClock(),
	}
	store.enroll[requestID] = request
	return request
}

func doJSON(t *testing.T, h http.Handler, method, path, tenant, org, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	if tenant != "" {
		request.Header.Set("X-Tenant-ID", tenant)
	}
	if org != "" {
		request.Header.Set("X-Organization-ID", org)
	}
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, request)
	return recorder
}

func decodeJSON(t *testing.T, recorder *httptest.ResponseRecorder, destination any) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), destination); err != nil {
		t.Fatalf("decode response %s: %v (body=%s)", recorder.Result().Status, err, recorder.Body.String())
	}
}

func TestDevicesListApprovalAndRevocationFlow(t *testing.T) {
	h, store := newHarness(t, nil)
	seedTrustedDevice(t, store, "device-trusted")
	seedEnrollment(store, "req-1", "device-new")

	rec := doJSON(t, h, http.MethodGet, "/api/v1/devices", testTenant, testOrg, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list devices status = %d", rec.Code)
	}
	var listing struct {
		Devices            []DeviceView            `json:"devices"`
		EnrollmentRequests []EnrollmentRequestView `json:"enrollment_requests"`
	}
	decodeJSON(t, rec, &listing)
	if len(listing.Devices) != 1 || listing.Devices[0].ID != "device-trusted" {
		t.Fatalf("unexpected devices: %+v", listing.Devices)
	}
	if len(listing.EnrollmentRequests) != 1 || listing.EnrollmentRequests[0].Status != "PENDING" {
		t.Fatalf("unexpected enrollment requests: %+v", listing.EnrollmentRequests)
	}
	if strings.Contains(rec.Body.String(), "nonce") || strings.Contains(rec.Body.String(), "signature") {
		t.Fatalf("proof-of-possession credentials leaked in devices listing: %s", rec.Body.String())
	}

	rec = doJSON(t, h, http.MethodPost, "/api/v1/devices/approve", testTenant, testOrg, `{"request_id":"req-1","epoch":2,"duration":86400}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve status = %d body=%s", rec.Code, rec.Body.String())
	}
	var approved struct {
		Device   DeviceView `json:"device"`
		Approved bool       `json:"approved"`
	}
	decodeJSON(t, rec, &approved)
	if !approved.Approved || approved.Device.State != statusTrusted || approved.Device.Epoch != 2 {
		t.Fatalf("unexpected approval result: %+v", approved)
	}

	rec = doJSON(t, h, http.MethodGet, "/api/v1/devices", testTenant, testOrg, "")
	decodeJSON(t, rec, &listing)
	if len(listing.Devices) != 2 {
		t.Fatalf("expected 2 devices after approval, got %d", len(listing.Devices))
	}
	if listing.EnrollmentRequests[0].Status != "APPROVED" {
		t.Fatalf("enrollment not marked approved: %+v", listing.EnrollmentRequests[0])
	}

	rec = doJSON(t, h, http.MethodPost, "/api/v1/devices/revoke", testTenant, testOrg, `{"device_id":"device-new","reason":"device compromised"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke status = %d body=%s", rec.Code, rec.Body.String())
	}
	var revoked struct {
		Device  DeviceView `json:"device"`
		Revoked bool       `json:"revoked"`
	}
	decodeJSON(t, rec, &revoked)
	if !revoked.Revoked || revoked.Device.State != statusRevoked || revoked.Device.RevocationReason != "device compromised" {
		t.Fatalf("unexpected revocation result: %+v", revoked)
	}

	rec = doJSON(t, h, http.MethodPost, "/api/v1/devices/approve", testTenant, testOrg, `{"request_id":"req-1","epoch":2,"duration":86400}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("double approval status = %d, want 409", rec.Code)
	}

	rec = doJSON(t, h, http.MethodPost, "/api/v1/devices/revoke", testTenant, testOrg, `{"device_id":"missing","reason":"nope"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("revoke unknown device status = %d, want 404", rec.Code)
	}
}

func TestDevicesApproveValidation(t *testing.T) {
	h, store := newHarness(t, nil)
	seedEnrollment(store, "req-1", "device-new")

	rec := doJSON(t, h, http.MethodPost, "/api/v1/devices/approve", testTenant, testOrg, `{"request_id":"req-1","epoch":0,"duration":86400}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("zero epoch status = %d, want 400", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/v1/devices/approve", testTenant, testOrg, `{"request_id":"req-1","epoch":2,"duration":-5}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("negative duration status = %d, want 400", rec.Code)
	}
}

func TestHeldListAndReconcileFlow(t *testing.T) {
	h, store := newHarness(t, nil)
	store.held["tx-1"] = &memHeld{
		TransactionID: "tx-1", TenantID: testTenant, OrganizationID: testOrg,
		DeviceID: "device-1", SequenceNumber: 7, ErrorReason: "sequence gap: expected 5",
		HeldAt: fixedClock(), PayloadHash: "deadbeef",
	}

	rec := doJSON(t, h, http.MethodGet, "/api/v1/sync/held", testTenant, testOrg, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("held status = %d", rec.Code)
	}
	var listing struct {
		HeldTransactions []HeldTransaction `json:"held_transactions"`
	}
	decodeJSON(t, rec, &listing)
	if len(listing.HeldTransactions) != 1 {
		t.Fatalf("expected 1 held transaction, got %d", len(listing.HeldTransactions))
	}
	held := listing.HeldTransactions[0]
	if held.TransactionID != "tx-1" || held.DeviceID != "device-1" || held.SequenceNumber != 7 ||
		held.Status != statusHeld || held.PayloadHash != "deadbeef" {
		t.Fatalf("unexpected held transaction: %+v", held)
	}
	if strings.Contains(rec.Body.String(), "\"payload\"") {
		t.Fatalf("raw offline payload leaked in held listing: %s", rec.Body.String())
	}

	rec = doJSON(t, h, http.MethodPost, "/api/v1/sync/reconcile", testTenant, testOrg, `{"transaction_id":"tx-1"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("reconcile status = %d body=%s", rec.Code, rec.Body.String())
	}
	var result ReconcileResult
	decodeJSON(t, rec, &result)
	if result.TransactionID != "tx-1" || result.Attempt != 1 || result.Status != statusHeld {
		t.Fatalf("unexpected reconcile result: %+v", result)
	}

	rec = doJSON(t, h, http.MethodGet, "/api/v1/sync/held", testTenant, testOrg, "")
	decodeJSON(t, rec, &listing)
	if listing.HeldTransactions[0].LastAttemptAt == nil {
		t.Fatalf("held transaction last_attempt_at not recorded: %+v", listing.HeldTransactions[0])
	}

	rec = doJSON(t, h, http.MethodPost, "/api/v1/sync/reconcile", testTenant, testOrg, `{"transaction_id":"missing"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("reconcile unknown transaction status = %d, want 404", rec.Code)
	}
}

func TestLegalHoldLifecycle(t *testing.T) {
	h, _ := newHarness(t, nil)

	rec := doJSON(t, h, http.MethodPost, "/api/v1/legal-holds", testTenant, testOrg, `{"entity_type":"inspection","entity_id":"insp-9","reason":"litigation hold"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("place legal hold status = %d body=%s", rec.Code, rec.Body.String())
	}
	var placed LegalHold
	decodeJSON(t, rec, &placed)
	if placed.Status != statusActive || placed.EntityID != "insp-9" || placed.Reason != "litigation hold" || placed.ID == "" {
		t.Fatalf("unexpected legal hold: %+v", placed)
	}

	rec = doJSON(t, h, http.MethodGet, "/api/v1/legal-holds", testTenant, testOrg, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list legal holds status = %d", rec.Code)
	}
	var listing struct {
		LegalHolds []LegalHold `json:"legal_holds"`
	}
	decodeJSON(t, rec, &listing)
	if len(listing.LegalHolds) != 1 || listing.LegalHolds[0].ID != placed.ID {
		t.Fatalf("unexpected legal holds: %+v", listing.LegalHolds)
	}

	rec = doJSON(t, h, http.MethodPost, "/api/v1/legal-holds/release", testTenant, testOrg, `{"id":"`+placed.ID+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("release legal hold status = %d body=%s", rec.Code, rec.Body.String())
	}
	var released LegalHold
	decodeJSON(t, rec, &released)
	if released.Status != statusReleased || released.ReleasedAt == nil {
		t.Fatalf("unexpected released hold: %+v", released)
	}

	rec = doJSON(t, h, http.MethodGet, "/api/v1/legal-holds", testTenant, testOrg, "")
	decodeJSON(t, rec, &listing)
	if len(listing.LegalHolds) != 0 {
		t.Fatalf("released hold still listed as active: %+v", listing.LegalHolds)
	}
}

func TestRetentionOverviewAndExportApprovalFlow(t *testing.T) {
	h, store := newHarness(t, nil)
	store.retention["inspection"] = &RetentionPolicy{
		TenantID: testTenant, EntityType: "inspection", RetentionDays: 365, Action: "ARCHIVE",
		UpdatedAt: fixedClock(), UpdatedBy: "admin-1",
	}
	store.exports["exp-1"] = &ExportApproval{
		ID: "approval-1", TenantID: testTenant, ExportID: "exp-1", RequestedBy: "user-1",
		Status: statusPending, CreatedAt: fixedClock(),
	}
	store.exports["exp-2"] = &ExportApproval{
		ID: "approval-2", TenantID: testTenant, ExportID: "exp-2", RequestedBy: "user-1",
		Status: statusPending, CreatedAt: fixedClock(),
	}

	rec := doJSON(t, h, http.MethodGet, "/api/v1/retention-policies", testTenant, testOrg, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("retention overview status = %d", rec.Code)
	}
	var overview struct {
		RetentionPolicies []RetentionPolicy `json:"retention_policies"`
		ExportApprovals   []ExportApproval  `json:"export_approvals"`
	}
	decodeJSON(t, rec, &overview)
	if len(overview.RetentionPolicies) != 1 || overview.RetentionPolicies[0].RetentionDays != 365 {
		t.Fatalf("unexpected retention policies: %+v", overview.RetentionPolicies)
	}
	if len(overview.ExportApprovals) != 2 {
		t.Fatalf("expected 2 export approvals, got %d", len(overview.ExportApprovals))
	}

	rec = doJSON(t, h, http.MethodPost, "/api/v1/exports/approvals", testTenant, testOrg, `{"export_id":"exp-1","status":"APPROVED"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve export status = %d body=%s", rec.Code, rec.Body.String())
	}
	var decided ExportApproval
	decodeJSON(t, rec, &decided)
	if decided.Status != statusApproved || decided.DecidedAt == nil {
		t.Fatalf("unexpected export decision: %+v", decided)
	}

	rec = doJSON(t, h, http.MethodPost, "/api/v1/exports/approvals", testTenant, testOrg, `{"export_id":"exp-2","status":"REJECTED"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("rejection without reason status = %d, want 400", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/v1/exports/approvals", testTenant, testOrg, `{"export_id":"exp-2","status":"REJECTED","rejection_reason":"redaction incomplete"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("reject export status = %d body=%s", rec.Code, rec.Body.String())
	}
	decodeJSON(t, rec, &decided)
	if decided.Status != statusRejected || decided.RejectionReason != "redaction incomplete" {
		t.Fatalf("unexpected rejection result: %+v", decided)
	}

	rec = doJSON(t, h, http.MethodPost, "/api/v1/exports/approvals", testTenant, testOrg, `{"export_id":"exp-1","status":"APPROVED"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("re-decision status = %d, want 409", rec.Code)
	}
}

func TestMissingTenantHeaderRejected(t *testing.T) {
	h, _ := newHarness(t, nil)

	rec := doJSON(t, h, http.MethodGet, "/api/v1/devices", "", testOrg, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing tenant header status = %d, want 400", rec.Code)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/v1/devices", testTenant, "", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing organization header status = %d, want 400", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/v1/legal-holds", "", "", `{"entity_type":"inspection","entity_id":"i-1","reason":"hold"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing both headers status = %d, want 400", rec.Code)
	}
}

func TestTenantIsolationAcrossStores(t *testing.T) {
	h, store := newHarness(t, nil)
	otherOrg := &EnrollmentRequestView{
		RequestID: "req-other", TenantID: "tenant-2", OrganizationID: testOrg,
		UserID: "user-2", DeviceID: "device-2", PublicKey: "pk-2", Status: "PENDING", RequestedAt: fixedClock(),
	}
	store.enroll["req-other"] = otherOrg
	seedEnrollment(store, "req-1", "device-1")

	var listing struct {
		EnrollmentRequests []EnrollmentRequestView `json:"enrollment_requests"`
	}
	rec := doJSON(t, h, http.MethodGet, "/api/v1/devices", testTenant, testOrg, "")
	decodeJSON(t, rec, &listing)
	if len(listing.EnrollmentRequests) != 1 || listing.EnrollmentRequests[0].RequestID != "req-1" {
		t.Fatalf("tenant isolation broken: %+v", listing.EnrollmentRequests)
	}

	rec = doJSON(t, h, http.MethodPost, "/api/v1/devices/approve", testTenant, testOrg, `{"request_id":"req-other","epoch":2,"duration":3600}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("approving foreign request status = %d, want 404", rec.Code)
	}
}

func TestConcurrentApprovalsAndListingsRaceSafe(t *testing.T) {
	h, store := newHarness(t, nil)
	const participants = 40
	for i := 0; i < participants; i++ {
		seedEnrollment(store, fmt.Sprintf("req-%d", i), fmt.Sprintf("device-%d", i))
	}

	var waitGroup sync.WaitGroup
	waitGroup.Add(participants + 2)
	for i := 0; i < participants; i++ {
		go func(index int) {
			defer waitGroup.Done()
			body := fmt.Sprintf(`{"request_id":"req-%d","epoch":2,"duration":3600}`, index)
			rec := doJSON(t, h, http.MethodPost, "/api/v1/devices/approve", testTenant, testOrg, body)
			if rec.Code != http.StatusOK {
				t.Errorf("concurrent approval %d status = %d", index, rec.Code)
			}
		}(i)
	}
	go func() {
		defer waitGroup.Done()
		for i := 0; i < 20; i++ {
			rec := doJSON(t, h, http.MethodGet, "/api/v1/devices", testTenant, testOrg, "")
			if rec.Code != http.StatusOK {
				t.Errorf("concurrent listing status = %d", rec.Code)
			}
		}
	}()
	go func() {
		defer waitGroup.Done()
		for i := 0; i < 20; i++ {
			rec := doJSON(t, h, http.MethodGet, "/api/v1/sync/held", testTenant, testOrg, "")
			if rec.Code != http.StatusOK {
				t.Errorf("concurrent held listing status = %d", rec.Code)
			}
		}
	}()
	waitGroup.Wait()

	var listing struct {
		Devices []DeviceView `json:"devices"`
	}
	rec := doJSON(t, h, http.MethodGet, "/api/v1/devices", testTenant, testOrg, "")
	decodeJSON(t, rec, &listing)
	if len(listing.Devices) != participants {
		t.Fatalf("expected %d devices after concurrent approvals, got %d", participants, len(listing.Devices))
	}
}
