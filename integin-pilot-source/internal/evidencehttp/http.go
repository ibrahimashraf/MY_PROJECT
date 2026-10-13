package evidencehttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"integin/internal/domain/evidence"
	"integin/internal/evidencepg"
	"integin/internal/evidenceregistration"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/workorderauth"
)

type TokenValidator interface {
	Validate(context.Context, string) (oidcauth.Principal, error)
}

type Handler struct {
	Validator TokenValidator
	Resolver  identity.Resolver
	Service   evidenceregistration.Service
}

type registrationRequest struct {
	EvidenceID          string    `json:"evidence_id"`
	InspectionID        string    `json:"inspection_id"`
	ContentType         string    `json:"content_type"`
	CiphertextBytes     int64     `json:"ciphertext_bytes"`
	PlaintextSHA256     string    `json:"plaintext_sha256"`
	CiphertextSHA256    string    `json:"ciphertext_sha256"`
	CapturedAt          time.Time `json:"captured_at"`
	DeviceID            string    `json:"device_id"`
	AuthorityID         string    `json:"authority_id"`
	AuthorityEpoch      uint64    `json:"authority_epoch"`
	TransactionID       string    `json:"transaction_id"`
	ReceiptID           string    `json:"receipt_id"`
	SignatureAlgorithm  string    `json:"signature_algorithm"`
	KeyID               string    `json:"key_id"`
	EncryptionAlgorithm string    `json:"encryption_algorithm"`
	EncryptionKeyRef    string    `json:"encryption_key_reference"`
	Classification      string    `json:"classification"`
	RetentionReference  string    `json:"retention_reference"`
	HoldState           string    `json:"hold_state"`
	RedactionPolicyRef  string    `json:"redaction_policy_reference"`
}

type registrationResponse struct {
	Outcome    string `json:"outcome"`
	EvidenceID string `json:"evidence_id,omitempty"`
	Inspection string `json:"inspection_id,omitempty"`
}

func (h Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost)
		write(writer, http.StatusMethodNotAllowed, "method_not_allowed", registrationResponse{})
		return
	}
	if h.Validator == nil || h.Resolver == nil || h.Service == nil {
		write(writer, http.StatusServiceUnavailable, "service_unavailable", registrationResponse{})
		return
	}
	raw, ok := bearer(request.Header.Get("Authorization"))
	if !ok {
		write(writer, http.StatusUnauthorized, "authentication_failed", registrationResponse{})
		return
	}
	principal, err := h.Validator.Validate(request.Context(), raw)
	if err != nil {
		write(writer, http.StatusUnauthorized, "authentication_failed", registrationResponse{})
		return
	}
	membership, err := h.Resolver.Resolve(request.Context(), identity.PrincipalKey{Issuer: principal.Issuer, Subject: principal.Subject})
	if err != nil {
		write(writer, http.StatusForbidden, "authorization_failed", registrationResponse{})
		return
	}
	actor, err := workorderauth.ActorFromMembership(membership)
	if err != nil {
		write(writer, http.StatusForbidden, "authorization_failed", registrationResponse{})
		return
	}
	var body registrationRequest
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		write(writer, http.StatusBadRequest, "invalid_request", registrationResponse{})
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		write(writer, http.StatusBadRequest, "invalid_request", registrationResponse{})
		return
	}
	result, err := h.Service.Register(request.Context(), evidenceregistration.RegisterCommand{
		Actor: actor, EvidenceID: body.EvidenceID, InspectionID: body.InspectionID, ContentType: body.ContentType,
		CiphertextBytes: body.CiphertextBytes, PlaintextSHA256: body.PlaintextSHA256, CiphertextSHA256: body.CiphertextSHA256,
		CapturedAt: body.CapturedAt, DeviceID: body.DeviceID, AuthorityID: body.AuthorityID, AuthorityEpoch: body.AuthorityEpoch,
		TransactionID: body.TransactionID, ReceiptID: body.ReceiptID, SignatureAlgorithm: body.SignatureAlgorithm, KeyID: body.KeyID,
		EncryptionAlgorithm: body.EncryptionAlgorithm, EncryptionKeyRef: body.EncryptionKeyRef, Classification: body.Classification,
		RetentionReference: body.RetentionReference, HoldState: body.HoldState, RedactionPolicyRef: body.RedactionPolicyRef,
	})
	if err != nil {
		status, code := http.StatusBadRequest, "registration_rejected"
		switch {
		case errors.Is(err, evidenceregistration.ErrDenied), errors.Is(err, evidencepg.ErrInspectionAssignmentDenied):
			status, code = http.StatusForbidden, "authorization_failed"
		case errors.Is(err, evidenceregistration.ErrObjectUnavailable), errors.Is(err, evidenceregistration.ErrObjectMismatch), errors.Is(err, evidence.ErrImmutableConflict):
			status, code = http.StatusConflict, "registration_conflict"
		}
		write(writer, status, code, registrationResponse{})
		return
	}
	outcome := "APPLIED"
	if !result.Inserted {
		outcome = "DUPLICATE"
	}
	write(writer, http.StatusOK, "", registrationResponse{Outcome: outcome, EvidenceID: result.Metadata.ID, Inspection: result.Metadata.InspectionID})
}

func bearer(value string) (string, bool) {
	parts := strings.Fields(value)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func write(writer http.ResponseWriter, status int, code string, payload registrationResponse) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if code != "" {
		_ = json.NewEncoder(writer).Encode(map[string]string{"error": code})
		return
	}
	_ = json.NewEncoder(writer).Encode(payload)
}
