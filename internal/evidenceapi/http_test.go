package evidenceapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/storage"
)

func TestHandlerAppliesAndDeduplicatesTenantScopedEvidence(t *testing.T) {
	store := storage.NewInMemoryStore()
	handler := Handler{Store: store}
	body := evidenceBody(t, "tenant-1", "org-1", "photo-1", []byte("encrypted-evidence"))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/evidence", bytes.NewReader(body)))
	if first.Code != http.StatusOK || !bytes.Contains(first.Body.Bytes(), []byte(`"APPLIED"`)) {
		t.Fatalf("first response = %d %s", first.Code, first.Body.String())
	}
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/evidence", bytes.NewReader(body)))
	if second.Code != http.StatusOK || !bytes.Contains(second.Body.Bytes(), []byte(`"DUPLICATE"`)) {
		t.Fatalf("second response = %d %s", second.Code, second.Body.String())
	}
	if _, err := store.Get(httptest.NewRequest(http.MethodGet, "/", nil).Context(), "tenant-1/org-1/evidence/photo-1"); err != nil {
		t.Fatal(err)
	}
}

func TestHandlerRejectsDigestMismatchAndTenantPathInjection(t *testing.T) {
	store := storage.NewInMemoryStore()
	handler := Handler{Store: store}
	body := evidenceBody(t, "tenant-1", "org-1", "photo-1", []byte("encrypted-evidence"))
	var payload request
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	payload.CiphertextSHA256 = "bad"
	badBody, _ := json.Marshal(payload)
	record := httptest.NewRecorder()
	handler.ServeHTTP(record, httptest.NewRequest(http.MethodPost, "/evidence", bytes.NewReader(badBody)))
	if record.Code != http.StatusBadRequest || !bytes.Contains(record.Body.Bytes(), []byte(`"SECURITY_FAILURE"`)) {
		t.Fatalf("digest response = %d %s", record.Code, record.Body.String())
	}

	pathBody := evidenceBody(t, "tenant/escape", "org-1", "photo-1", []byte("encrypted-evidence"))
	record = httptest.NewRecorder()
	handler.ServeHTTP(record, httptest.NewRequest(http.MethodPost, "/evidence", bytes.NewReader(pathBody)))
	if record.Code != http.StatusBadRequest {
		t.Fatalf("path response = %d", record.Code)
	}
}

func TestHandlerRejectsMalformedEvidenceRequests(t *testing.T) {
	handler := Handler{Store: storage.NewInMemoryStore()}
	tests := []struct {
		name string
		body []byte
	}{
		{name: "malformed json", body: []byte(`{"tenant_id":`)},
		{name: "invalid base64", body: evidenceRequestBody("%%%", "digest")},
		{name: "missing ciphertext digest", body: evidenceRequestBody(base64.StdEncoding.EncodeToString([]byte("evidence")), "")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := httptest.NewRecorder()
			handler.ServeHTTP(record, httptest.NewRequest(http.MethodPost, "/evidence", bytes.NewReader(test.body)))
			if record.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body = %s", record.Code, record.Body.String())
			}
		})
	}
}

func evidenceRequestBody(blob, digest string) []byte {
	body, _ := json.Marshal(request{TenantID: "tenant-1", OrganizationID: "org-1", EvidenceID: "photo-invalid", InspectionID: "inspection-1", ContentType: "application/octet-stream", CiphertextSHA256: digest, Base64Blob: blob})
	return body
}

func evidenceBody(t *testing.T, tenant, organization, evidenceID string, data []byte) []byte {
	t.Helper()
	digest := sha256.Sum256(data)
	body, err := json.Marshal(request{TenantID: tenant, OrganizationID: organization, EvidenceID: evidenceID, InspectionID: "inspection-1", ContentType: "application/octet-stream", CiphertextSHA256: hex.EncodeToString(digest[:]), Base64Blob: base64.StdEncoding.EncodeToString(data)})
	if err != nil {
		t.Fatal(err)
	}
	return body
}
