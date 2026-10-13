// Evidence export manifest v1 is a pure contract package. It has no database,
// storage, HTTP, workflow, or authority mutation dependency.
package exportmanifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const Version = "evidence-export-manifest/v1"

type Manifest struct {
	ManifestVersion  string            `json:"manifest_version"`
	ExportID         string            `json:"export_id"`
	TenantID         string            `json:"tenant_id"`
	OrganizationID   string            `json:"organization_id"`
	CreatedAt        time.Time         `json:"created_at"`
	Exporter         ExporterIdentity  `json:"exporter"`
	ProcedureVersion string            `json:"procedure_version"`
	Evidence         []EvidenceRecord  `json:"evidence"`
	ObjectCount      uint64            `json:"object_count"`
	Recovery         RecoveryMetadata  `json:"recovery"`
	Privacy          PrivacyMetadata   `json:"privacy"`
	Approval         ApprovalReference `json:"approval"`
	ManifestChecksum string            `json:"manifest_checksum"`
}

type ExporterIdentity struct {
	SubjectReference    string `json:"subject_reference"`
	MembershipReference string `json:"membership_reference"`
}

type EvidenceRecord struct {
	EvidenceID          string    `json:"evidence_id"`
	ObjectKey           string    `json:"object_key"`
	ContentType         string    `json:"content_type"`
	CapturedAt          time.Time `json:"captured_at"`
	InspectionReference string    `json:"inspection_reference"`
	EntityReference     string    `json:"entity_reference"`
	PlaintextSHA256     string    `json:"plaintext_sha256"`
	CiphertextSHA256    string    `json:"ciphertext_sha256"`
	CiphertextBytes     uint64    `json:"ciphertext_bytes"`
	EncryptionAlgorithm string    `json:"encryption_algorithm"`
	EncryptionKeyRef    string    `json:"encryption_key_reference"`
	DeviceID            string    `json:"device_id"`
	AuthorityID         string    `json:"authority_id"`
	AuthorityEpoch      uint64    `json:"authority_epoch"`
	TransactionID       string    `json:"transaction_id"`
	ReceiptID           string    `json:"receipt_id"`
	SignatureAlgorithm  string    `json:"signature_algorithm"`
	KeyID               string    `json:"key_id"`
}

type RecoveryMetadata struct {
	MissingObjectIDs          []string `json:"missing_object_ids"`
	RestoreVerificationResult string   `json:"restore_verification_result"`
}

type PrivacyMetadata struct {
	Classification     string `json:"classification"`
	RetentionReference string `json:"retention_reference"`
	HoldState          string `json:"hold_state"`
	RedactionPolicyRef string `json:"redaction_policy_reference"`
}

type ApprovalReference struct {
	ExportApprovalReference string `json:"export_approval_reference"`
}

// Seal computes the v1 SHA-256 checksum over the deterministic payload and
// assigns it to ManifestChecksum. It does not persist, export, or authorize.
func (m *Manifest) Seal() error {
	payload, err := m.canonicalPayload()
	if err != nil {
		return err
	}
	sum := sha256.Sum256(payload)
	m.ManifestChecksum = hex.EncodeToString(sum[:])
	return nil
}

// Verify checks the required metadata and recomputes the deterministic v1
// checksum. It is read-only and does not access referenced evidence objects.
func (m Manifest) Verify() error {
	if strings.TrimSpace(m.ManifestChecksum) == "" {
		return errors.New("manifest_checksum is required")
	}
	payload, err := m.canonicalPayload()
	if err != nil {
		return err
	}
	sum := sha256.Sum256(payload)
	if m.ManifestChecksum != hex.EncodeToString(sum[:]) {
		return errors.New("manifest_checksum does not match canonical payload")
	}
	return nil
}

// CanonicalPayload returns the deterministic byte representation whose SHA-256
// value becomes ManifestChecksum. The checksum itself is intentionally omitted.
func (m Manifest) CanonicalPayload() ([]byte, error) {
	return m.canonicalPayload()
}

func (m Manifest) canonicalPayload() ([]byte, error) {
	if err := m.validateFields(); err != nil {
		return nil, err
	}

	records := append([]EvidenceRecord(nil), m.Evidence...)
	sort.Slice(records, func(i, j int) bool {
		if records[i].EvidenceID == records[j].EvidenceID {
			return records[i].ObjectKey < records[j].ObjectKey
		}
		return records[i].EvidenceID < records[j].EvidenceID
	})
	missing := append([]string(nil), m.Recovery.MissingObjectIDs...)
	sort.Strings(missing)

	canonical := struct {
		ManifestVersion  string            `json:"manifest_version"`
		ExportID         string            `json:"export_id"`
		TenantID         string            `json:"tenant_id"`
		OrganizationID   string            `json:"organization_id"`
		CreatedAt        string            `json:"created_at"`
		Exporter         ExporterIdentity  `json:"exporter"`
		ProcedureVersion string            `json:"procedure_version"`
		Evidence         []EvidenceRecord  `json:"evidence"`
		ObjectCount      uint64            `json:"object_count"`
		Recovery         RecoveryMetadata  `json:"recovery"`
		Privacy          PrivacyMetadata   `json:"privacy"`
		Approval         ApprovalReference `json:"approval"`
	}{
		ManifestVersion:  m.ManifestVersion,
		ExportID:         m.ExportID,
		TenantID:         m.TenantID,
		OrganizationID:   m.OrganizationID,
		CreatedAt:        m.CreatedAt.UTC().Format(time.RFC3339Nano),
		Exporter:         m.Exporter,
		ProcedureVersion: m.ProcedureVersion,
		Evidence:         records,
		ObjectCount:      m.ObjectCount,
		Recovery: RecoveryMetadata{
			MissingObjectIDs:          missing,
			RestoreVerificationResult: m.Recovery.RestoreVerificationResult,
		},
		Privacy:  m.Privacy,
		Approval: m.Approval,
	}
	return json.Marshal(canonical)
}

func (m Manifest) validateFields() error {
	for field, value := range map[string]string{
		"manifest_version":            m.ManifestVersion,
		"export_id":                   m.ExportID,
		"tenant_id":                   m.TenantID,
		"organization_id":             m.OrganizationID,
		"subject_reference":           m.Exporter.SubjectReference,
		"membership_reference":        m.Exporter.MembershipReference,
		"procedure_version":           m.ProcedureVersion,
		"classification":              m.Privacy.Classification,
		"retention_reference":         m.Privacy.RetentionReference,
		"hold_state":                  m.Privacy.HoldState,
		"redaction_policy_reference":  m.Privacy.RedactionPolicyRef,
		"export_approval_reference":   m.Approval.ExportApprovalReference,
		"restore_verification_result": m.Recovery.RestoreVerificationResult,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if m.ManifestVersion != Version {
		return fmt.Errorf("manifest_version = %q, want %q", m.ManifestVersion, Version)
	}
	if m.CreatedAt.IsZero() {
		return errors.New("created_at is required")
	}
	if m.ObjectCount != uint64(len(m.Evidence)) {
		return fmt.Errorf("object_count = %d, want %d", m.ObjectCount, len(m.Evidence))
	}

	evidenceIDs := make(map[string]struct{}, len(m.Evidence))
	objectKeys := make(map[string]struct{}, len(m.Evidence))
	for _, record := range m.Evidence {
		if err := record.validate(); err != nil {
			return err
		}
		if _, exists := evidenceIDs[record.EvidenceID]; exists {
			return fmt.Errorf("duplicate evidence_id %q", record.EvidenceID)
		}
		if _, exists := objectKeys[record.ObjectKey]; exists {
			return fmt.Errorf("duplicate object_key %q", record.ObjectKey)
		}
		evidenceIDs[record.EvidenceID] = struct{}{}
		objectKeys[record.ObjectKey] = struct{}{}
	}
	return nil
}

func (r EvidenceRecord) validate() error {
	for field, value := range map[string]string{
		"evidence_id":              r.EvidenceID,
		"object_key":               r.ObjectKey,
		"content_type":             r.ContentType,
		"inspection_reference":     r.InspectionReference,
		"entity_reference":         r.EntityReference,
		"plaintext_sha256":         r.PlaintextSHA256,
		"ciphertext_sha256":        r.CiphertextSHA256,
		"encryption_algorithm":     r.EncryptionAlgorithm,
		"encryption_key_reference": r.EncryptionKeyRef,
		"device_id":                r.DeviceID,
		"authority_id":             r.AuthorityID,
		"transaction_id":           r.TransactionID,
		"receipt_id":               r.ReceiptID,
		"signature_algorithm":      r.SignatureAlgorithm,
		"key_id":                   r.KeyID,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("evidence %s is required", field)
		}
	}
	if r.CapturedAt.IsZero() {
		return errors.New("evidence captured_at is required")
	}
	if strings.Contains(r.ObjectKey, "..") {
		return errors.New("evidence object_key must not contain traversal segments")
	}
	return nil
}
