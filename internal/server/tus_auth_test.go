package server

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/device_trust"
	"integin/internal/localprovision"
	"integin/internal/oidcauth"
	"integin/internal/storage"
	"integin/internal/syncapi"
)

type tusValidatorStub struct{ err error }

func (s tusValidatorStub) Validate(_ context.Context, _ string) (oidcauth.Principal, error) {
	return oidcauth.Principal{Issuer: "issuer", Subject: "subject"}, s.err
}

func TestTUSRoutesRejectUnauthenticatedThroughMux(t *testing.T) {
	manager, err := storage.NewTUSManager(t.TempDir(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	mux := NewMux(Dependencies{TUSHandler: storage.TUSRouteHandler{Manager: manager}, Validator: tusValidatorStub{}})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/uploads", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no-bearer create status=%d want %d body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}

	rejecting := NewMux(Dependencies{TUSHandler: storage.TUSRouteHandler{Manager: manager}, Validator: tusValidatorStub{err: errors.New("rejected")}})
	request := httptest.NewRequest(http.MethodPost, "/uploads", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")
	rec = httptest.NewRecorder()
	rejecting.ServeHTTP(rec, request)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("invalid-bearer status=%d want %d body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestTUSRoutesAcceptProvisionUploadToken(t *testing.T) {
	manager, err := storage.NewTUSManager(t.TempDir(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	store := storage.NewInMemoryStore()
	// No OIDC validator: the provision-bound token alone must authenticate.
	registry := syncapi.NewAuthorityRegistry()
	device, err := device_trust.NewDevice("device-1", "tenant-1", "org-1", "user-1", "public-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	authority, err := device_trust.IssueAuthorityPackage(device, "auth-1", "upload-secret", []string{"evidence.upload"}, time.Now().UTC().Add(-time.Minute), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	registry.Register(authority)

	mux := NewMux(Dependencies{
		TUSHandler:              storage.TUSRouteHandler{Manager: manager, Store: store},
		UploadTokenSecret:       "upload-secret",
		UploadAuthorityRegistry: registry,
	})
	createBody := `{"size":11,"checksum":"` + strings.Repeat("ab", 32) + `","content_type":"image/jpeg"}`
	post := func(token string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/uploads", strings.NewReader(createBody))
		request.Header.Set("Authorization", "Bearer "+token)
		mux.ServeHTTP(rec, request)
		return rec
	}
	token, err := localprovision.MintUploadToken("upload-secret", "device-1", "auth-1", authority.Epoch, authority.ExpiresAt)
	if err != nil {
		t.Fatal(err)
	}
	if rec := post(token); rec.Code != http.StatusCreated {
		t.Fatalf("upload-token create status=%d body=%s, want 201", rec.Code, rec.Body.String())
	}

	// A valid HMAC for an authority this server never issued must be refused.
	unregistered, err := localprovision.MintUploadToken("upload-secret", "device-1", "auth-unknown", 1, time.Now().UTC().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if rec := post(unregistered); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unregistered authority token status=%d, want 401", rec.Code)
	}

	// A token naming a different device than its authority must be refused.
	crossDevice, err := localprovision.MintUploadToken("upload-secret", "device-other", "auth-1", authority.Epoch, authority.ExpiresAt)
	if err != nil {
		t.Fatal(err)
	}
	if rec := post(crossDevice); rec.Code != http.StatusUnauthorized {
		t.Fatalf("cross-device token status=%d, want 401", rec.Code)
	}

	// A token for the wrong epoch must be refused.
	wrongEpoch, err := localprovision.MintUploadToken("upload-secret", "device-1", "auth-1", authority.Epoch+7, authority.ExpiresAt)
	if err != nil {
		t.Fatal(err)
	}
	if rec := post(wrongEpoch); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong-epoch token status=%d, want 401", rec.Code)
	}

	// A token bound to an already-expired authority must be refused even
	// though the token's own expiry nominally still lies ahead.
	expiredAuthority, err := device_trust.IssueAuthorityPackage(device, "auth-expired", "upload-secret", []string{"evidence.upload"}, time.Now().UTC().Add(-2*time.Hour), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	registry.Register(expiredAuthority)
	expired, err := localprovision.MintUploadToken("upload-secret", "device-1", "auth-expired", expiredAuthority.Epoch, time.Now().UTC().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if rec := post(expired); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired-authority token status=%d, want 401", rec.Code)
	}

	// An expired token remains rejected regardless of registry state.
	stale, err := localprovision.MintUploadToken("upload-secret", "device-1", "auth-1", authority.Epoch, time.Now().UTC().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if rec := post(stale); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired token status=%d, want 401", rec.Code)
	}
}

func TestTUSRoutesFailClosedWithoutValidator(t *testing.T) {
	manager, err := storage.NewTUSManager(t.TempDir(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	mux := NewMux(Dependencies{TUSHandler: storage.TUSRouteHandler{Manager: manager}})
	rec := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/uploads", nil)
	request.Header.Set("Authorization", "Bearer any-token")
	mux.ServeHTTP(rec, request)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("uploads without a validator must fail closed, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTUSAuthenticatedUploadFlowThroughMux(t *testing.T) {
	manager, err := storage.NewTUSManager(t.TempDir(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	store := storage.NewInMemoryStore()
	mux := NewMux(Dependencies{TUSHandler: storage.TUSRouteHandler{Manager: manager, Store: store}, Validator: tusValidatorStub{}})

	payload := make([]byte, 3000)
	for i := range payload {
		payload[i] = byte(i * 13)
	}
	digest := sha256.Sum256(payload)
	checksum := hex.EncodeToString(digest[:])

	authorized := func(request *http.Request) *http.Request {
		request.Header.Set("Authorization", "Bearer valid-token")
		return request
	}
	create := httptest.NewRecorder()
	mux.ServeHTTP(create, authorized(httptest.NewRequest(http.MethodPost, "/uploads", strings.NewReader(`{"size":3000,"checksum":"`+checksum+`","content_type":"video/mp4"}`))))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("create returned no id")
	}

	appendChunk := func(offset int, chunk []byte) {
		rec := httptest.NewRecorder()
		body := `{"offset":` + strconv.Itoa(offset) + `,"data":"` + base64.StdEncoding.EncodeToString(chunk) + `"}`
		mux.ServeHTTP(rec, authorized(httptest.NewRequest(http.MethodPost, "/uploads/"+created.ID+"/chunks", strings.NewReader(body))))
		if rec.Code != http.StatusOK {
			t.Fatalf("append at offset %d status=%d body=%s", offset, rec.Code, rec.Body.String())
		}
	}
	appendChunk(0, payload[:1500])
	appendChunk(1500, payload[1500:])

	offsetRec := httptest.NewRecorder()
	mux.ServeHTTP(offsetRec, authorized(httptest.NewRequest(http.MethodGet, "/uploads/"+created.ID+"/offset", nil)))
	if offsetRec.Code != http.StatusOK || !strings.Contains(offsetRec.Body.String(), `"offset":3000`) {
		t.Fatalf("offset status=%d body=%s", offsetRec.Code, offsetRec.Body.String())
	}

	completeRec := httptest.NewRecorder()
	mux.ServeHTTP(completeRec, authorized(httptest.NewRequest(http.MethodPost, "/uploads/"+created.ID+"/complete", nil)))
	if completeRec.Code != http.StatusOK || !strings.Contains(completeRec.Body.String(), created.ID) {
		t.Fatalf("complete status=%d body=%s", completeRec.Code, completeRec.Body.String())
	}
}
