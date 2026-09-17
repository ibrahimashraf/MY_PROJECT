package certificate

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// Signature capacities. Client acknowledgement and inspector attestation are
// separate acts; the capacity records which one a SignatureEvent captures.
const (
	CapacityClient    = "client"
	CapacityInspector = "inspector"
	CapacityVerifier  = "verifier"
)

// MaxSignatureImageBytes caps the signature image payload (5 MiB). The raw
// image travels with the evidence payload; the aggregate stores only digests.
const MaxSignatureImageBytes = 5 * 1024 * 1024

// MaxTextFieldChars bounds free-text signature fields stored in audit JSONB.
const MaxTextFieldChars = 1000

// SignatureEvent is a server-recorded signature act bound to one certificate
// revision and to the exact content reviewed (snapshot digest). Timestamps,
// certificate number, and revision are set by the aggregate at record time —
// never trusted from client input.
type SignatureEvent struct {
	SignerID         string `json:"signer_id"`
	SignerName       string `json:"signer_name"`
	Capacity         string `json:"capacity"`
	StatementVersion string `json:"statement_version"`
	ImageSHA256Hex   string `json:"image_sha256_hex"`
	ImageBytes       int64  `json:"image_bytes"`
	ImageEvidenceID  string `json:"image_evidence_id"`
	SnapshotSHA256   string `json:"snapshot_sha256_hex"`

	CertificateNumber string    `json:"certificate_number"`
	Revision          int       `json:"revision"`
	SignedAt          time.Time `json:"signed_at"`
}

func (e SignatureEvent) Validate() error {
	if strings.TrimSpace(e.SignerID) == "" {
		return fmt.Errorf("signer_id is required")
	}
	if strings.TrimSpace(e.SignerName) == "" {
		return fmt.Errorf("signer_name is required")
	}
	if len(e.SignerName) > MaxTextFieldChars {
		return fmt.Errorf("signer_name exceeds %d characters", MaxTextFieldChars)
	}
	switch e.Capacity {
	case CapacityClient, CapacityInspector, CapacityVerifier:
	default:
		return fmt.Errorf("capacity must be client, inspector, or verifier")
	}
	if strings.TrimSpace(e.StatementVersion) == "" {
		return fmt.Errorf("statement_version is required")
	}
	if len(e.StatementVersion) > MaxTextFieldChars {
		return fmt.Errorf("statement_version exceeds %d characters", MaxTextFieldChars)
	}
	if err := validateSHA256Hex(e.ImageSHA256Hex, "image_sha256_hex"); err != nil {
		return err
	}
	if e.ImageBytes < 1 || e.ImageBytes > MaxSignatureImageBytes {
		return fmt.Errorf("image_bytes must be between 1 and %d", MaxSignatureImageBytes)
	}
	if strings.TrimSpace(e.ImageEvidenceID) == "" {
		return fmt.Errorf("image_evidence_id is required")
	}
	if err := validateSHA256Hex(e.SnapshotSHA256, "snapshot_sha256_hex"); err != nil {
		return err
	}
	return nil
}

func validateSHA256Hex(value, field string) error {
	if len(value) != 64 {
		return fmt.Errorf("%s must be 64 hex characters", field)
	}
	if _, err := hex.DecodeString(value); err != nil {
		return fmt.Errorf("%s must be hex-encoded: %w", field, err)
	}
	return nil
}

// SignWithEvent records a client or verifier sign-off. It requires Approved
// status, rejects the assigned inspector (separation of duties — inspectors
// attest via Attest instead), and seals the event to the current certificate
// number, revision, snapshot digest, and server time.
func (c *Certificate) SignWithEvent(event SignatureEvent) error {
	if err := c.require(Approved); err != nil {
		return err
	}
	if err := event.Validate(); err != nil {
		return err
	}
	if event.Capacity == CapacityInspector {
		return fmt.Errorf("inspector attests via Attest and cannot sign certificate")
	}
	if event.SignerID == c.inspectorID {
		return fmt.Errorf("inspector cannot sign certificate")
	}
	event.CertificateNumber = c.number
	event.Revision = c.revision
	event.SignedAt = time.Now().UTC()
	emitted, err := c.makeEvent("CertificateSigned", event)
	if err != nil {
		return err
	}
	c.status, c.signedBy = Signed, event.SignerID
	c.signatures = append(c.signatures, event)
	c.emitted = append(c.emitted, emitted)
	return nil
}

// Attest records the assigned inspector's attestation that the findings are
// accurately recorded. It changes no lifecycle status and is allowed from
// Draft, PendingApproval, or Approved — attestation precedes sign-off.
func (c *Certificate) Attest(attestorID, attestorName, qualificationBasis, statementVersion, snapshotSHA256Hex string) error {
	if c.status != Draft && c.status != PendingApproval && c.status != Approved {
		return fmt.Errorf("cannot attest certificate in status %s", c.status)
	}
	if strings.TrimSpace(attestorID) == "" {
		return fmt.Errorf("attestor_id is required")
	}
	if attestorID != c.inspectorID {
		return fmt.Errorf("only the assigned inspector can attest findings")
	}
	if strings.TrimSpace(attestorName) == "" {
		return fmt.Errorf("attestor_name is required")
	}
	if len(attestorName) > MaxTextFieldChars {
		return fmt.Errorf("attestor_name exceeds %d characters", MaxTextFieldChars)
	}
	if strings.TrimSpace(qualificationBasis) == "" {
		return fmt.Errorf("qualification_basis is required")
	}
	if len(qualificationBasis) > MaxTextFieldChars {
		return fmt.Errorf("qualification_basis exceeds %d characters", MaxTextFieldChars)
	}
	if strings.TrimSpace(statementVersion) == "" {
		return fmt.Errorf("statement_version is required")
	}
	if len(statementVersion) > MaxTextFieldChars {
		return fmt.Errorf("statement_version exceeds %d characters", MaxTextFieldChars)
	}
	if err := validateSHA256Hex(snapshotSHA256Hex, "snapshot_sha256_hex"); err != nil {
		return err
	}
	event := SignatureEvent{
		SignerID:          attestorID,
		SignerName:        attestorName,
		Capacity:          CapacityInspector,
		StatementVersion:  statementVersion,
		SnapshotSHA256:    snapshotSHA256Hex,
		CertificateNumber: c.number,
		Revision:          c.revision,
		SignedAt:          time.Now().UTC(),
	}
	emitted, err := c.makeEvent("CertificateAttested", event)
	if err != nil {
		return err
	}
	c.signatures = append(c.signatures, event)
	c.emitted = append(c.emitted, emitted)
	return nil
}

// SignWaiver records a waived sign-off: no ink was captured, and the waiver
// authority and reason stand in for the signature. The waiver still advances
// Approved to Signed so the lifecycle can continue to issuance.
type SignWaiver struct {
	GrantedBy         string    `json:"granted_by"`
	Reason            string    `json:"reason"`
	Capacity          string    `json:"capacity"`
	CertificateNumber string    `json:"certificate_number"`
	Revision          int       `json:"revision"`
	WaivedAt          time.Time `json:"waived_at"`
}

func (w SignWaiver) Validate() error {
	if strings.TrimSpace(w.GrantedBy) == "" {
		return fmt.Errorf("granted_by is required")
	}
	if strings.TrimSpace(w.Reason) == "" || len(w.Reason) > MaxTextFieldChars {
		return fmt.Errorf("reason is required and must not exceed %d characters", MaxTextFieldChars)
	}
	switch w.Capacity {
	case CapacityClient, CapacityVerifier:
	default:
		return fmt.Errorf("waiver capacity must be client or verifier")
	}
	return nil
}

// WaiveSignature records a waived sign-off for the given capacity. It requires
// Approved status and seals the waiver to the current certificate number,
// revision, and server time.
func (c *Certificate) WaiveSignature(grantedBy, reason, capacity string) error {
	if err := c.require(Approved); err != nil {
		return err
	}
	waiver := SignWaiver{GrantedBy: grantedBy, Reason: reason, Capacity: capacity}
	if err := waiver.Validate(); err != nil {
		return err
	}
	waiver.CertificateNumber = c.number
	waiver.Revision = c.revision
	waiver.WaivedAt = time.Now().UTC()
	emitted, err := c.makeEvent("CertificateSignatureWaived", waiver)
	if err != nil {
		return err
	}
	c.status, c.signedBy = Signed, grantedBy
	c.signWaiver = &waiver
	c.emitted = append(c.emitted, emitted)
	return nil
}
