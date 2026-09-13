// Package evidenceexport implements Export Manifest v1: a signed, integrity-checked
// manifest describing an export of encrypted evidence objects.
package evidenceexport

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ManifestVersion is the only supported manifest version.
const ManifestVersion = "1.0"

// ExporterIdentity identifies the user, role, and client that produced an export.
type ExporterIdentity struct {
	UserID        string `json:"user_id"`
	Role          string `json:"role"`
	ClientVersion string `json:"client_version"`
}

// EvidenceItem describes one encrypted evidence object referenced by a manifest.
type EvidenceItem struct {
	EvidenceID          string    `json:"evidence_id"`
	ObjectKey           string    `json:"object_key"`
	ContentType         string    `json:"content_type"`
	CapturedAt          time.Time `json:"captured_at"`
	PlaintextSHA256     string    `json:"plaintext_sha256"`
	CiphertextSHA256    string    `json:"ciphertext_sha256"`
	ByteLength          int64     `json:"byte_length"`
	EncryptionAlgorithm string    `json:"encryption_algorithm"`
	AuthorityDeviceID   string    `json:"authority_device_id"`
}

// Manifest is Export Manifest v1. ManifestSHA256 and ExportSignature are
// derived fields and are excluded from the canonical payload they describe.
type Manifest struct {
	ManifestVersion  string           `json:"manifest_version"`
	ExportID         string           `json:"export_id"`
	TenantID         string           `json:"tenant_id"`
	OrganizationID   string           `json:"organization_id"`
	CreatedAt        time.Time        `json:"created_at"`
	ExporterIdentity ExporterIdentity `json:"exporter_identity"`
	ProcedureVersion string           `json:"procedure_version"`
	EvidenceItems    []EvidenceItem   `json:"evidence_items"`
	ObjectCount      int              `json:"object_count"`
	TotalByteLength  int64            `json:"total_byte_length"`
	ManifestSHA256   string           `json:"manifest_sha256"`
	ExportSignature  string           `json:"export_signature"`
}

// SignerFunc produces a signature over the canonical manifest payload.
type SignerFunc func(canonicalPayload []byte) ([]byte, error)

// NewManifest builds a manifest, computing ObjectCount, TotalByteLength, and
// ManifestSHA256 from the canonical payload of all other fields.
func NewManifest(exportID, tenantID, orgID, procedureVersion string, exporter ExporterIdentity, items []EvidenceItem) (*Manifest, error) {
	for _, check := range []struct{ field, value string }{
		{"export_id", exportID},
		{"tenant_id", tenantID},
		{"organization_id", orgID},
		{"procedure_version", procedureVersion},
	} {
		if strings.TrimSpace(check.value) == "" {
			return nil, fmt.Errorf("%s is required", check.field)
		}
	}
	if err := exporter.validate(); err != nil {
		return nil, err
	}
	copied := append([]EvidenceItem(nil), items...)
	for i := range copied {
		if err := copied[i].validate(); err != nil {
			return nil, err
		}
	}
	if err := validateUnique(copied); err != nil {
		return nil, err
	}

	m := &Manifest{
		ManifestVersion:  ManifestVersion,
		ExportID:         exportID,
		TenantID:         tenantID,
		OrganizationID:   orgID,
		CreatedAt:        time.Now().UTC(),
		ExporterIdentity: exporter,
		ProcedureVersion: procedureVersion,
		EvidenceItems:    copied,
		ObjectCount:      len(copied),
	}
	for _, item := range copied {
		m.TotalByteLength += item.ByteLength
	}
	if err := m.seal(); err != nil {
		return nil, err
	}
	return m, nil
}

// Sign sets ManifestSHA256 and ExportSignature over the canonical payload.
func (m *Manifest) Sign(signer SignerFunc) error {
	if m == nil {
		return errors.New("manifest is nil")
	}
	if signer == nil {
		return errors.New("signer is required")
	}
	canonical, err := m.canonicalPayload()
	if err != nil {
		return err
	}
	sig, err := signer(canonical)
	if err != nil {
		return fmt.Errorf("sign manifest: %w", err)
	}
	if len(sig) == 0 {
		return errors.New("signer produced an empty signature")
	}
	m.ManifestSHA256 = hex.EncodeToString(sha256sum(canonical))
	m.ExportSignature = base64.RawURLEncoding.EncodeToString(sig)
	return nil
}

// VerifyManifest read-only validates object count, checksums, byte length,
// signature, and non-empty identifiers.
func (m *Manifest) VerifyManifest(publicKey []byte) error {
	if m == nil {
		return errors.New("manifest is nil")
	}
	if err := m.validateFields(); err != nil {
		return err
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("public key length = %d, want %d", len(publicKey), ed25519.PublicKeySize)
	}
	canonical, err := m.canonicalPayload()
	if err != nil {
		return err
	}
	if m.ManifestSHA256 != hex.EncodeToString(sha256sum(canonical)) {
		return errors.New("manifest_sha256 does not match canonical payload")
	}
	if strings.TrimSpace(m.ExportSignature) == "" {
		return errors.New("export_signature is required")
	}
	sig, err := base64.RawURLEncoding.DecodeString(m.ExportSignature)
	if err != nil {
		sig, err = base64.StdEncoding.DecodeString(m.ExportSignature)
	}
	if err != nil {
		return errors.New("export_signature is not valid base64")
	}
	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("signature length = %d, want %d", len(sig), ed25519.SignatureSize)
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), canonical, sig) {
		return errors.New("export_signature verification failed")
	}
	return nil
}

// seal computes ManifestSHA256 over the canonical payload. Shared by
// NewManifest and Sign so the digest is always derived from the same bytes.
func (m *Manifest) seal() error {
	canonical, err := m.canonicalPayload()
	if err != nil {
		return err
	}
	m.ManifestSHA256 = hex.EncodeToString(sha256sum(canonical))
	return nil
}

// canonicalPayload serializes all fields that precede the signature in a
// stable, sorted order.
func (m *Manifest) canonicalPayload() ([]byte, error) {
	items := append([]EvidenceItem(nil), m.EvidenceItems...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].EvidenceID == items[j].EvidenceID {
			return items[i].ObjectKey < items[j].ObjectKey
		}
		return items[i].EvidenceID < items[j].EvidenceID
	})
	canonical := struct {
		ManifestVersion  string           `json:"manifest_version"`
		ExportID         string           `json:"export_id"`
		TenantID         string           `json:"tenant_id"`
		OrganizationID   string           `json:"organization_id"`
		CreatedAt        string           `json:"created_at"`
		ExporterIdentity ExporterIdentity `json:"exporter_identity"`
		ProcedureVersion string           `json:"procedure_version"`
		EvidenceItems    []EvidenceItem   `json:"evidence_items"`
		ObjectCount      int              `json:"object_count"`
		TotalByteLength  int64            `json:"total_byte_length"`
	}{
		ManifestVersion:  m.ManifestVersion,
		ExportID:         m.ExportID,
		TenantID:         m.TenantID,
		OrganizationID:   m.OrganizationID,
		CreatedAt:        m.CreatedAt.UTC().Format(time.RFC3339Nano),
		ExporterIdentity: m.ExporterIdentity,
		ProcedureVersion: m.ProcedureVersion,
		EvidenceItems:    items,
		ObjectCount:      m.ObjectCount,
		TotalByteLength:  m.TotalByteLength,
	}
	return json.Marshal(canonical)
}

// validateFields performs all structural and integrity checks except the
// cryptographic signature.
func (m *Manifest) validateFields() error {
	for _, check := range []struct{ field, value string }{
		{"manifest_version", m.ManifestVersion},
		{"export_id", m.ExportID},
		{"tenant_id", m.TenantID},
		{"organization_id", m.OrganizationID},
		{"procedure_version", m.ProcedureVersion},
	} {
		if strings.TrimSpace(check.value) == "" {
			return fmt.Errorf("%s is required", check.field)
		}
	}
	if m.ManifestVersion != ManifestVersion {
		return fmt.Errorf("manifest_version = %q, want %q", m.ManifestVersion, ManifestVersion)
	}
	if m.CreatedAt.IsZero() {
		return errors.New("created_at is required")
	}
	if err := m.ExporterIdentity.validate(); err != nil {
		return err
	}
	if m.ObjectCount != len(m.EvidenceItems) {
		return fmt.Errorf("object_count = %d, want %d", m.ObjectCount, len(m.EvidenceItems))
	}
	var total int64
	for _, item := range m.EvidenceItems {
		if err := item.validate(); err != nil {
			return err
		}
		total += item.ByteLength
	}
	if m.TotalByteLength != total {
		return fmt.Errorf("total_byte_length = %d, want %d", m.TotalByteLength, total)
	}
	return validateUnique(m.EvidenceItems)
}

func (e ExporterIdentity) validate() error {
	for _, check := range []struct{ field, value string }{
		{"user_id", e.UserID},
		{"role", e.Role},
		{"client_version", e.ClientVersion},
	} {
		if strings.TrimSpace(check.value) == "" {
			return fmt.Errorf("exporter %s is required", check.field)
		}
	}
	return nil
}

func (i EvidenceItem) validate() error {
	for _, check := range []struct{ field, value string }{
		{"evidence_id", i.EvidenceID},
		{"object_key", i.ObjectKey},
		{"content_type", i.ContentType},
		{"plaintext_sha256", i.PlaintextSHA256},
		{"ciphertext_sha256", i.CiphertextSHA256},
		{"encryption_algorithm", i.EncryptionAlgorithm},
		{"authority_device_id", i.AuthorityDeviceID},
	} {
		if strings.TrimSpace(check.value) == "" {
			return fmt.Errorf("evidence %s is required", check.field)
		}
	}
	if i.CapturedAt.IsZero() {
		return errors.New("evidence captured_at is required")
	}
	if i.ByteLength < 0 {
		return fmt.Errorf("evidence %q byte_length = %d, want >= 0", i.EvidenceID, i.ByteLength)
	}
	if err := isSHA256Hex(i.PlaintextSHA256); err != nil {
		return fmt.Errorf("evidence %q plaintext_sha256: %w", i.EvidenceID, err)
	}
	if err := isSHA256Hex(i.CiphertextSHA256); err != nil {
		return fmt.Errorf("evidence %q ciphertext_sha256: %w", i.EvidenceID, err)
	}
	if strings.Contains(i.ObjectKey, "..") {
		return errors.New("evidence object_key must not contain traversal segments")
	}
	return nil
}

func validateUnique(items []EvidenceItem) error {
	ids := make(map[string]struct{}, len(items))
	keys := make(map[string]struct{}, len(items))
	for _, item := range items {
		if _, exists := ids[item.EvidenceID]; exists {
			return fmt.Errorf("duplicate evidence_id %q", item.EvidenceID)
		}
		if _, exists := keys[item.ObjectKey]; exists {
			return fmt.Errorf("duplicate object_key %q", item.ObjectKey)
		}
		ids[item.EvidenceID] = struct{}{}
		keys[item.ObjectKey] = struct{}{}
	}
	return nil
}

func isSHA256Hex(s string) error {
	if len(s) != sha256.Size*2 {
		return fmt.Errorf("must be %d hex characters, got %d", sha256.Size*2, len(s))
	}
	if _, err := hex.DecodeString(s); err != nil {
		return errors.New("must be valid lowercase hex")
	}
	return nil
}

func sha256sum(payload []byte) []byte {
	sum := sha256.Sum256(payload)
	return sum[:]
}
