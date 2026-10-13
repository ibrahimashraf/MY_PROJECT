package evidence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidActor      = errors.New("evidence actor context is invalid")
	ErrInvalidMetadata   = errors.New("evidence metadata is invalid")
	ErrImmutableConflict = errors.New("evidence metadata conflicts with an immutable registered record")
)

type ActorContext struct {
	TenantID       string
	OrganizationID string
	ActorID        string
}

func (a ActorContext) Validate() error {
	if strings.TrimSpace(a.TenantID) == "" || strings.TrimSpace(a.OrganizationID) == "" || strings.TrimSpace(a.ActorID) == "" {
		return ErrInvalidActor
	}
	return nil
}

type Metadata struct {
	ID                  string
	TenantID            string
	OrganizationID      string
	InspectionID        string
	ObjectKey           string
	ContentType         string
	CiphertextBytes     int64
	PlaintextSHA256     string
	CiphertextSHA256    string
	CapturedAt          time.Time
	DeviceID            string
	AuthorityID         string
	AuthorityEpoch      uint64
	TransactionID       string
	ReceiptID           string
	SignatureAlgorithm  string
	KeyID               string
	EncryptionAlgorithm string
	EncryptionKeyRef    string
	Classification      string
	RetentionReference  string
	HoldState           string
	RedactionPolicyRef  string
	RegisteredBy        string
	RegisteredAt        time.Time
}

type Repository interface {
	Register(context.Context, ActorContext, Metadata) (Metadata, bool, error)
	ListByInspection(context.Context, ActorContext, string) ([]Metadata, error)
	ListByTenantOrganization(context.Context, ActorContext) ([]Metadata, error)
}

func (m Metadata) ValidateForRegistration(actor ActorContext) error {
	if err := actor.Validate(); err != nil {
		return err
	}
	if m.TenantID != actor.TenantID || m.OrganizationID != actor.OrganizationID || m.RegisteredBy != actor.ActorID {
		return fmt.Errorf("%w: server-derived actor does not match metadata scope", ErrInvalidMetadata)
	}
	for field, value := range map[string]string{
		"id":                         m.ID,
		"tenant_id":                  m.TenantID,
		"organization_id":            m.OrganizationID,
		"inspection_id":              m.InspectionID,
		"object_key":                 m.ObjectKey,
		"content_type":               m.ContentType,
		"plaintext_sha256":           m.PlaintextSHA256,
		"ciphertext_sha256":          m.CiphertextSHA256,
		"device_id":                  m.DeviceID,
		"authority_id":               m.AuthorityID,
		"transaction_id":             m.TransactionID,
		"receipt_id":                 m.ReceiptID,
		"signature_algorithm":        m.SignatureAlgorithm,
		"key_id":                     m.KeyID,
		"encryption_algorithm":       m.EncryptionAlgorithm,
		"encryption_key_reference":   m.EncryptionKeyRef,
		"classification":             m.Classification,
		"retention_reference":        m.RetentionReference,
		"hold_state":                 m.HoldState,
		"redaction_policy_reference": m.RedactionPolicyRef,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%w: %s is required", ErrInvalidMetadata, field)
		}
	}
	if m.CiphertextBytes <= 0 || m.CapturedAt.IsZero() || m.AuthorityEpoch == 0 {
		return fmt.Errorf("%w: bytes, captured time, and authority epoch are required", ErrInvalidMetadata)
	}
	if !isSHA256(m.PlaintextSHA256) || !isSHA256(m.CiphertextSHA256) {
		return fmt.Errorf("%w: digests must be lowercase SHA-256 hex", ErrInvalidMetadata)
	}
	expectedKey := m.TenantID + "/" + m.OrganizationID + "/evidence/" + m.ID
	// Per-question linkage (Bswagic 090s Upload images per question) is encoded as fragment #q=<code>
	// so existing rows (no fragment) stay valid and RLS/immutability is untouched.
	baseKey := m.ObjectKey
	if idx := strings.Index(m.ObjectKey, "#q="); idx >= 0 {
		baseKey = m.ObjectKey[:idx]
		q := strings.TrimSpace(m.ObjectKey[idx+3:])
		if q == "" || len(q) > 64 || strings.Contains(q, "/") || strings.Contains(q, "..") {
			return fmt.Errorf("%w: question code in object key is invalid", ErrInvalidMetadata)
		}
	}
	if baseKey != expectedKey || strings.Contains(m.ObjectKey, "..") {
		return fmt.Errorf("%w: object key must be contained evidence key", ErrInvalidMetadata)
	}
	return nil
}

func QuestionEvidenceObjectKey(tenantID, organizationID, evidenceID, questionCode string) string {
	base := tenantID + "/" + organizationID + "/evidence/" + evidenceID
	if strings.TrimSpace(questionCode) == "" {
		return base
	}
	return base + "#q=" + strings.TrimSpace(questionCode)
}

func ParseQuestionCode(objectKey string) string {
	parts := strings.Split(objectKey, "#q=")
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func (m Metadata) SameImmutableContent(other Metadata) bool {
	return m.ID == other.ID &&
		m.TenantID == other.TenantID &&
		m.OrganizationID == other.OrganizationID &&
		m.InspectionID == other.InspectionID &&
		m.ObjectKey == other.ObjectKey &&
		m.ContentType == other.ContentType &&
		m.CiphertextBytes == other.CiphertextBytes &&
		m.PlaintextSHA256 == other.PlaintextSHA256 &&
		m.CiphertextSHA256 == other.CiphertextSHA256 &&
		m.CapturedAt.Equal(other.CapturedAt) &&
		m.DeviceID == other.DeviceID &&
		m.AuthorityID == other.AuthorityID &&
		m.AuthorityEpoch == other.AuthorityEpoch &&
		m.TransactionID == other.TransactionID &&
		m.ReceiptID == other.ReceiptID &&
		m.SignatureAlgorithm == other.SignatureAlgorithm &&
		m.KeyID == other.KeyID &&
		m.EncryptionAlgorithm == other.EncryptionAlgorithm &&
		m.EncryptionKeyRef == other.EncryptionKeyRef &&
		m.Classification == other.Classification &&
		m.RetentionReference == other.RetentionReference &&
		m.HoldState == other.HoldState &&
		m.RedactionPolicyRef == other.RedactionPolicyRef
}

func isSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if !(character >= '0' && character <= '9' || character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}
