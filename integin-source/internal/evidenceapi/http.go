package evidenceapi

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"integin/internal/storage"
)

type Handler struct {
	Store storage.Store
}

type request struct {
	TenantID         string `json:"tenant_id"`
	OrganizationID   string `json:"organization_id"`
	EvidenceID       string `json:"evidence_id"`
	InspectionID     string `json:"inspection_id"`
	ContentType      string `json:"content_type"`
	LegacySHA256     string `json:"sha256,omitempty"`
	PlaintextSHA256  string `json:"plaintext_sha256"`
	CiphertextSHA256 string `json:"ciphertext_sha256"`
	Base64Blob       string `json:"base64_blob"`
}

type response struct {
	Outcome        string `json:"outcome"`
	EvidenceID     string `json:"evidence_id,omitempty"`
	ObjectKey      string `json:"object_key,omitempty"`
	Reason         string `json:"reason,omitempty"`
	ContentHash    string `json:"sha256,omitempty"`
	PlaintextHash  string `json:"plaintext_sha256,omitempty"`
	CiphertextHash string `json:"ciphertext_sha256,omitempty"`
}

func (h Handler) ServeHTTP(writer http.ResponseWriter, requestHTTP *http.Request) {
	if requestHTTP.Method != http.MethodPost {
		writeJSON(writer, http.StatusMethodNotAllowed, response{Outcome: "REJECTED", Reason: "POST is required"})
		return
	}
	if h.Store == nil {
		writeJSON(writer, http.StatusServiceUnavailable, response{Outcome: "REJECTED", Reason: "evidence store is unavailable"})
		return
	}
	var incoming request
	if err := json.NewDecoder(requestHTTP.Body).Decode(&incoming); err != nil {
		writeJSON(writer, http.StatusBadRequest, response{Outcome: "REJECTED", Reason: "invalid JSON request"})
		return
	}
	if err := validate(incoming); err != nil {
		writeJSON(writer, http.StatusBadRequest, response{Outcome: "REJECTED", Reason: err.Error()})
		return
	}
	data, err := base64.StdEncoding.DecodeString(incoming.Base64Blob)
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, response{Outcome: "REJECTED", Reason: "base64 blob is invalid"})
		return
	}
	digest := sha256.Sum256(data)
	actualCiphertextHash := hex.EncodeToString(digest[:])
	expectedCiphertextHash := incoming.CiphertextSHA256
	if expectedCiphertextHash == "" {
		expectedCiphertextHash = incoming.LegacySHA256
	}
	if actualCiphertextHash != expectedCiphertextHash {
		writeJSON(writer, http.StatusBadRequest, response{Outcome: "SECURITY_FAILURE", EvidenceID: incoming.EvidenceID, Reason: "ciphertext digest mismatch", ContentHash: actualCiphertextHash, CiphertextHash: actualCiphertextHash, PlaintextHash: incoming.PlaintextSHA256})
		return
	}
	key := objectKey(incoming.TenantID, incoming.OrganizationID, incoming.EvidenceID)
	if existing, getErr := h.Store.Get(requestHTTP.Context(), key); getErr == nil {
		existingDigest := sha256.Sum256(existing.Data)
		if hex.EncodeToString(existingDigest[:]) == actualCiphertextHash {
			writeJSON(writer, http.StatusOK, response{Outcome: "DUPLICATE", EvidenceID: incoming.EvidenceID, ObjectKey: key, ContentHash: actualCiphertextHash, CiphertextHash: actualCiphertextHash, PlaintextHash: incoming.PlaintextSHA256})
			return
		}
		writeJSON(writer, http.StatusConflict, response{Outcome: "CONFLICT", EvidenceID: incoming.EvidenceID, ObjectKey: key, Reason: "evidence id was reused with different content"})
		return
	}
	if err := h.Store.Put(context.Background(), storage.Object{
		Key:         key,
		ContentType: incoming.ContentType,
		Data:        data,
		Metadata: map[string]string{
			"tenant_id":         incoming.TenantID,
			"organization_id":   incoming.OrganizationID,
			"inspection_id":     incoming.InspectionID,
			"plaintext_sha256":  incoming.PlaintextSHA256,
			"ciphertext_sha256": actualCiphertextHash,
		},
	}); err != nil {
		writeJSON(writer, http.StatusServiceUnavailable, response{Outcome: "REJECTED", EvidenceID: incoming.EvidenceID, Reason: err.Error()})
		return
	}
	writeJSON(writer, http.StatusOK, response{Outcome: "APPLIED", EvidenceID: incoming.EvidenceID, ObjectKey: key, ContentHash: actualCiphertextHash, CiphertextHash: actualCiphertextHash, PlaintextHash: incoming.PlaintextSHA256})
}

func validate(incoming request) error {
	for name, value := range map[string]string{"tenant_id": incoming.TenantID, "organization_id": incoming.OrganizationID, "evidence_id": incoming.EvidenceID, "inspection_id": incoming.InspectionID, "content_type": incoming.ContentType, "base64_blob": incoming.Base64Blob} {
		if strings.TrimSpace(value) == "" {
			return errors.New(name + " is required")
		}
	}
	if strings.TrimSpace(incoming.CiphertextSHA256) == "" && strings.TrimSpace(incoming.LegacySHA256) == "" {
		return errors.New("ciphertext_sha256 is required")
	}
	if strings.ContainsAny(incoming.TenantID+incoming.OrganizationID+incoming.EvidenceID, "/\\") || strings.Contains(incoming.EvidenceID, "..") {
		return errors.New("evidence identity contains an invalid path")
	}
	return nil
}

func objectKey(tenantID, organizationID, evidenceID string) string {
	return tenantID + "/" + organizationID + "/evidence/" + evidenceID
}

func writeJSON(writer http.ResponseWriter, status int, payload response) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}
