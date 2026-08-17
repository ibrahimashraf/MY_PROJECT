package packagemanifestapi

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/device_trust"
	domainsync "integin/internal/domain/sync"
	"integin/internal/domain/workpackage"
	"integin/internal/packagemanifest"
	"integin/internal/workpackagepg"
)

func TestHandlerRejectsMissingAuthority(t *testing.T) {
	now := testNow()
	h := newHandler(t, fakeVerifier{context: verified(now)}, testStore{}, packagemanifest.NewInMemoryProofReplayStore(func() time.Time { return now }))
	response := serve(h, proofRequest("missing-authority", "request-1"))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestHandlerRejectsExpiredProof(t *testing.T) {
	now := testNow()
	h := newHandler(t, fakeVerifier{err: errors.New("proof expired")}, testStore{}, packagemanifest.NewInMemoryProofReplayStore(func() time.Time { return now }))
	h.Authorities = authorityLookup{"authority-1": {ID: "authority-1"}}
	response := serve(h, proofRequest("authority-1", "request-1"))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestHandlerRejectsReplay(t *testing.T) {
	now := testNow()
	replay := packagemanifest.NewInMemoryProofReplayStore(func() time.Time { return now })
	h := newHandler(t, fakeVerifier{context: verified(now)}, validStore(now), replay)
	h.Authorities = authorityLookup{"authority-1": {ID: "authority-1"}}
	request := proofRequest("authority-1", "request-1")
	if response := serve(h, request); response.Code != http.StatusOK {
		t.Fatalf("first request status = %d", response.Code)
	}
	if response := serve(h, request); response.Code != http.StatusConflict {
		t.Fatalf("replay request status = %d", response.Code)
	}
}

func TestHandlerReportsMissingAssignment(t *testing.T) {
	now := testNow()
	h := newHandler(t, fakeVerifier{context: verified(now)}, testStore{assignmentErr: sql.ErrNoRows}, packagemanifest.NewInMemoryProofReplayStore(func() time.Time { return now }))
	h.Authorities = authorityLookup{"authority-1": {ID: "authority-1"}}
	response := serve(h, proofRequest("authority-1", "request-1"))
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}
}

func TestHandlerReturnsSignedManifest(t *testing.T) {
	now := testNow()
	h := newHandler(t, fakeVerifier{context: verified(now)}, validStore(now), packagemanifest.NewInMemoryProofReplayStore(func() time.Time { return now }))
	h.Authorities = authorityLookup{"authority-1": {ID: "authority-1"}}
	response := serve(h, proofRequest("authority-1", "request-1"))
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var manifest packagemanifest.PackageManifest
	if err := json.NewDecoder(response.Body).Decode(&manifest); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if manifest.PackageID != "package-1" || manifest.PackageHash != "sha256:test" || manifest.Signature == "" {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
}

type fakeVerifier struct {
	context domainsync.VerifiedDeviceContext
	err     error
}

func (v fakeVerifier) VerifyDeviceProof(context.Context, domainsync.DeviceProof, device_trust.AuthorityPackage, time.Time) (domainsync.VerifiedDeviceContext, error) {
	return v.context, v.err
}

type testStore struct {
	assignment    workpackagepg.Assignment
	assignmentErr error
	pkg           workpackage.Package
}

func (s testStore) GetCurrentAssignment(context.Context, string, string, string, string, time.Time) (workpackagepg.Assignment, error) {
	return s.assignment, s.assignmentErr
}

func (s testStore) GetApproved(context.Context, string, string, string, int) (workpackage.Package, error) {
	return s.pkg, nil
}

type authorityLookup map[string]device_trust.AuthorityPackage

func (r authorityLookup) Get(id string) (device_trust.AuthorityPackage, bool) {
	authority, ok := r[id]
	return authority, ok
}

func newHandler(t *testing.T, verifier DeviceProofVerifier, store testStore, replay packagemanifest.ProofReplayStore) *Handler {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	issuer, err := packagemanifest.NewManifestIssuer(store, packagemanifest.StaticAssignmentContextResolver{Context: workpackage.AssignmentContext{RootAssetID: "asset-1", InspectionType: "thorough-inspection", ProcedureVersion: "v1", ScheduledAt: time.Date(2026, time.August, 17, 9, 0, 0, 0, time.UTC)}}, privateKey, "manifest-key-1", 30*time.Minute)
	if err != nil {
		t.Fatalf("new issuer: %v", err)
	}
	return &Handler{Verifier: verifier, Authorities: authorityLookup{}, ReplayStore: replay, Issuer: issuer, Now: testNow}
}

func serve(handler http.Handler, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/work-packages/manifest", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func proofRequest(authorityID, requestID string) string {
	proof := domainsync.DeviceProof{AuthorityID: authorityID, Purpose: domainsync.WorkPackageManifestReadPurpose, RequestID: requestID, InspectionID: "inspection-1", ExpiresAt: testNow().Add(time.Minute)}
	encoded, _ := json.Marshal(request{Proof: proof})
	return string(encoded)
}

func testNow() time.Time {
	return time.Date(2026, time.August, 17, 10, 0, 0, 0, time.UTC)
}

func verified(now time.Time) domainsync.VerifiedDeviceContext {
	return domainsync.VerifiedDeviceContext{TenantID: "tenant-1", OrganizationID: "organization-1", DeviceID: "device-1", AuthorityID: "authority-1", AuthorityEpoch: 7, ProofExpiresAt: now.Add(time.Minute), AuthorityExpiry: now.Add(time.Hour)}
}

func validStore(now time.Time) testStore {
	return testStore{assignment: workpackagepg.Assignment{TenantID: "tenant-1", OrganizationID: "organization-1", InspectionID: "inspection-1", DeviceID: "device-1", PackageID: "package-1", PackageVersion: 1, AuthorityEpoch: 7, AssignedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)}, pkg: workpackage.Package{ID: "package-1", TenantID: "tenant-1", OrganizationID: "organization-1", PackageVersion: 1, SchemaVersion: 1, State: workpackage.PublicationApproved, PackageHash: "sha256:test"}}
}
