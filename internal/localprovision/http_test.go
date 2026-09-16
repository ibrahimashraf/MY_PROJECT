package localprovision

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/device_trust"
	domainsync "integin/internal/domain/sync"
	"integin/internal/security"
	"integin/internal/syncstate"
)

type memoryRepository struct {
	device    syncstate.DeviceRecord
	authority syncstate.AuthorityRecord
}

func (r *memoryRepository) SaveDevice(_ context.Context, value syncstate.DeviceRecord) error {
	r.device = value
	return nil
}

func (r *memoryRepository) SaveAuthority(_ context.Context, value syncstate.AuthorityRecord) error {
	r.authority = value
	return nil
}

func TestHandlerProvisionsLoopbackDeviceWithPublicKeyOnly(t *testing.T) {
	publicKey, _, err := security.GenerateDeviceKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	repository := &memoryRepository{}
	processor, err := domainsync.NewProcessor(map[string]string{"default": "local-secret"})
	if err != nil {
		t.Fatal(err)
	}
	registered := make(chan device_trust.AuthorityPackage, 1)
	handler, err := NewHandler(Config{
		Repository: repository, Processor: processor, SigningSecret: "local-secret",
		TenantID: "tenant-live", OrganizationID: "org-live", UserID: "user-live",
		AuthorityLifetime: 30 * time.Minute, Now: func() time.Time { return time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	handler.SetAuthorityRegistrar(func(authority device_trust.AuthorityPackage) { registered <- authority })
	keyID := security.DeviceKeyID(ed25519.PublicKey(publicKey))
	payload, err := json.Marshal(request{DeviceID: "field-" + keyID[:32], KeyID: keyID, PublicKey: base64.StdEncoding.EncodeToString(publicKey)})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "http://localhost/local/provision", bytes.NewReader(payload))
	request.RemoteAddr = "127.0.0.1:40101"
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if repository.device.DeviceID == "" || len(repository.device.PublicKey) != ed25519.PublicKeySize {
		t.Fatalf("device was not durably saved: %#v", repository.device)
	}
	if repository.authority.AuthorityID == "" || repository.authority.DeviceID != repository.device.DeviceID {
		t.Fatalf("authority was not durably saved: %#v", repository.authority)
	}
	select {
	case authority := <-registered:
		if authority.ID != repository.authority.AuthorityID {
			t.Fatalf("registered authority=%q want=%q", authority.ID, repository.authority.AuthorityID)
		}
	default:
		t.Fatal("authority was not registered for immediate sync")
	}
	if bytes.Contains(recorder.Body.Bytes(), []byte("private_key")) {
		t.Fatal("provisioning response exposed private key material")
	}
}

func TestHandlerRejectsNonLoopbackClient(t *testing.T) {
	repository := &memoryRepository{}
	processor, err := domainsync.NewProcessor(map[string]string{"default": "local-secret"})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandler(Config{Repository: repository, Processor: processor, SigningSecret: "local-secret", TenantID: "tenant", OrganizationID: "org", UserID: "user", AuthorityLifetime: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	handler.SetAuthorityRegistrar(func(device_trust.AuthorityPackage) {})
	request := httptest.NewRequest(http.MethodPost, "http://localhost/local/provision", nil)
	request.RemoteAddr = "203.0.113.9:40101"
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=%d", recorder.Code, http.StatusForbidden)
	}
}

func TestHandlerRejectsUnknownFieldsAndMalformedJSON(t *testing.T) {
	repository := &memoryRepository{}
	processor, err := domainsync.NewProcessor(map[string]string{"default": "local-secret"})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandler(Config{Repository: repository, Processor: processor, SigningSecret: "local-secret", TenantID: "tenant", OrganizationID: "org", UserID: "user", AuthorityLifetime: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	handler.SetAuthorityRegistrar(func(device_trust.AuthorityPackage) {})

	// Unknown field
	req := httptest.NewRequest(http.MethodPost, "http://localhost/local/provision", strings.NewReader(`{"device_id":"dev-1","key_id":"k-1","public_key":"pk","unknown_field":"rejected"}`))
	req.RemoteAddr = "127.0.0.1:40101"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status=%d want 400", rec.Code)
	}

	// Extra json token after EOF
	req = httptest.NewRequest(http.MethodPost, "http://localhost/local/provision", strings.NewReader(`{"device_id":"dev-1","key_id":"k-1","public_key":"pk"}{"trailing":"data"}`))
	req.RemoteAddr = "127.0.0.1:40101"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("trailing json status=%d want 400", rec.Code)
	}
}

