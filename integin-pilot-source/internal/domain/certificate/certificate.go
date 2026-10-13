package certificate

import (
	"fmt"
	"strings"
	"time"

	"integin/internal/shared/events"
	"integin/internal/shared/types"
)

type Status string

const (
	Draft           Status = "DRAFT"
	PendingApproval Status = "PENDING_APPROVAL"
	Approved        Status = "APPROVED"
	Signed          Status = "SIGNED"
	Issued          Status = "ISSUED"
	Expired         Status = "EXPIRED"
	Revoked         Status = "REVOKED"
	Superseded      Status = "SUPERSEDED"
)

type SeparationException struct {
	GrantedBy string `json:"granted_by"`
	Reason    string `json:"reason"`
}

// Certificate is a controlled record. Once issued, business corrections must
// use a new revision or replacement rather than mutating certificate data.
type Certificate struct {
	id                    string
	tenantID              string
	organizationID        string
	environment           string
	number                string
	revision              int
	inspectionID          string
	inspectionRevision    int
	assetID               string
	inspectorID           string
	createdBy             string
	status                Status
	approvedBy            string
	signedBy              string
	issuedBy              string
	issuedAt              time.Time
	expiryDate            time.Time
	supersedesID          string
	supersededByID        string
	revocationReason      string
	exception             *SeparationException
	signatures            []SignatureEvent
	signWaiver            *SignWaiver
	emitted               []events.Envelope
	renewalAuthorityToken []byte // 32-byte authority token set by migration 0016 renewal tracking
	renewalCount          int    // incremented by SetRenewalAuthority
}

func New(id, tenantID, organizationID, environment, number, inspectionID string, inspectionRevision int, assetID, inspectorID, createdBy string, expiryDate time.Time) (Certificate, error) {
	required := map[string]string{"id": id, "tenant_id": tenantID, "organization_id": organizationID, "number": number, "inspection_id": inspectionID, "asset_id": assetID, "inspector_id": inspectorID, "created_by": createdBy}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return Certificate{}, fmt.Errorf("%s is required", field)
		}
	}
	if inspectionRevision <= 0 {
		return Certificate{}, fmt.Errorf("inspection_revision must be positive")
	}
	context := types.TenantContext{TenantID: tenantID, OrganizationID: organizationID, Environment: environment}
	if err := context.Validate(); err != nil {
		return Certificate{}, err
	}
	if expiryDate.IsZero() {
		return Certificate{}, fmt.Errorf("expiry_date is required")
	}
	return Certificate{id: id, tenantID: tenantID, organizationID: organizationID, environment: environment, number: number, revision: 1, inspectionID: inspectionID, inspectionRevision: inspectionRevision, assetID: assetID, inspectorID: inspectorID, createdBy: createdBy, status: Draft, expiryDate: expiryDate.UTC()}, nil
}

func (c Certificate) ID() string               { return c.id }
func (c Certificate) TenantID() string         { return c.tenantID }
func (c Certificate) OrganizationID() string   { return c.organizationID }
func (c Certificate) Environment() string      { return c.environment }
func (c Certificate) Number() string           { return c.number }
func (c Certificate) Revision() int            { return c.revision }
func (c Certificate) InspectionID() string     { return c.inspectionID }
func (c Certificate) InspectionRevision() int  { return c.inspectionRevision }
func (c Certificate) AssetID() string          { return c.assetID }
func (c Certificate) InspectorID() string      { return c.inspectorID }
func (c Certificate) CreatedBy() string        { return c.createdBy }
func (c Certificate) Status() Status           { return c.status }
func (c Certificate) ApprovedBy() string       { return c.approvedBy }
func (c Certificate) SignedBy() string         { return c.signedBy }
func (c Certificate) IssuedBy() string         { return c.issuedBy }
func (c Certificate) IssuedAt() time.Time      { return c.issuedAt }
func (c Certificate) ExpiryDate() time.Time    { return c.expiryDate }
func (c Certificate) SupersedesID() string     { return c.supersedesID }
func (c Certificate) SupersededByID() string   { return c.supersededByID }
func (c Certificate) RevocationReason() string { return c.revocationReason }
func (c Certificate) Exception() *SeparationException {
	if c.exception == nil {
		return nil
	}
	copy := *c.exception
	return &copy
}
func (c Certificate) Events() []events.Envelope { return append([]events.Envelope(nil), c.emitted...) }
func (c Certificate) Signatures() []SignatureEvent {
	return append([]SignatureEvent(nil), c.signatures...)
}
func (c Certificate) SignWaiver() *SignWaiver {
	if c.signWaiver == nil {
		return nil
	}
	waiver := *c.signWaiver
	return &waiver
}

func (c *Certificate) SubmitForApproval(actorID string) error {
	if err := c.require(Draft); err != nil {
		return err
	}
	if strings.TrimSpace(actorID) == "" {
		return fmt.Errorf("actor_id is required")
	}
	event, err := c.makeEvent("CertificateSubmittedForApproval", map[string]any{"actor_id": actorID, "revision": c.revision})
	if err != nil {
		return err
	}
	c.status = PendingApproval
	c.emitted = append(c.emitted, event)
	return nil
}

func (c *Certificate) Approve(approverID string, exception *SeparationException) error {
	if err := c.require(PendingApproval); err != nil {
		return err
	}
	if strings.TrimSpace(approverID) == "" {
		return fmt.Errorf("approver_id is required")
	}
	if approverID == c.inspectorID || approverID == c.createdBy {
		if exception == nil || strings.TrimSpace(exception.GrantedBy) == "" || strings.TrimSpace(exception.Reason) == "" {
			return fmt.Errorf("separation of duties violation: approval actor must differ from inspector and creator")
		}
		if exception.GrantedBy == approverID {
			return fmt.Errorf("separation exception must be granted by a distinct actor")
		}
	}
	event, err := c.makeEvent("CertificateApproved", map[string]any{"approver_id": approverID, "exception": exception})
	if err != nil {
		return err
	}
	c.status, c.approvedBy = Approved, approverID
	if exception != nil {
		copied := *exception
		c.exception = &copied
	}
	c.emitted = append(c.emitted, event)
	return nil
}

func (c *Certificate) Sign(signerID string) error {
	if err := c.require(Approved); err != nil {
		return err
	}
	if strings.TrimSpace(signerID) == "" {
		return fmt.Errorf("signer_id is required")
	}
	if signerID == c.inspectorID {
		return fmt.Errorf("inspector cannot sign certificate")
	}
	event, err := c.makeEvent("CertificateSigned", map[string]string{"signer_id": signerID})
	if err != nil {
		return err
	}
	c.status, c.signedBy = Signed, signerID
	c.emitted = append(c.emitted, event)
	return nil
}

func (c *Certificate) Issue(issuerID string, issuedAt time.Time) error {
	if err := c.require(Signed); err != nil {
		return err
	}
	if strings.TrimSpace(issuerID) == "" {
		return fmt.Errorf("issuer_id is required")
	}
	if issuerID == c.inspectorID {
		return fmt.Errorf("inspector cannot issue certificate")
	}
	if issuedAt.IsZero() {
		issuedAt = time.Now().UTC()
	} else {
		issuedAt = issuedAt.UTC()
	}
	event, err := c.makeEvent("CertificateIssued", map[string]any{"issuer_id": issuerID, "issued_at": issuedAt})
	if err != nil {
		return err
	}
	c.status, c.issuedBy, c.issuedAt = Issued, issuerID, issuedAt
	c.emitted = append(c.emitted, event)
	return nil
}

func (c *Certificate) Expire(at time.Time) error {
	if err := c.require(Issued); err != nil {
		return err
	}
	if at.IsZero() {
		return fmt.Errorf("expiry check time is required")
	}
	if at.Before(c.expiryDate) {
		return fmt.Errorf("certificate is not yet expired")
	}
	event, err := c.makeEvent("CertificateExpired", map[string]any{"expired_at": at.UTC()})
	if err != nil {
		return err
	}
	c.status = Expired
	c.emitted = append(c.emitted, event)
	return nil
}

func (c *Certificate) Revoke(actorID, reason string) error {
	if c.status != Issued && c.status != Signed {
		return fmt.Errorf("cannot revoke certificate from status %s", c.status)
	}
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(reason) == "" {
		return fmt.Errorf("actor_id and reason are required")
	}
	event, err := c.makeEvent(events.CertificateRevoked, map[string]string{"actor_id": actorID, "reason": reason})
	if err != nil {
		return err
	}
	c.status, c.revocationReason = Revoked, reason
	c.emitted = append(c.emitted, event)
	return nil
}

func (c *Certificate) Supersede(actorID, replacementID string) error {
	if err := c.require(Issued); err != nil {
		return err
	}
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(replacementID) == "" {
		return fmt.Errorf("actor_id and replacement_id are required")
	}
	event, err := c.makeEvent("CertificateSuperseded", map[string]string{"actor_id": actorID, "replacement_id": replacementID})
	if err != nil {
		return err
	}
	c.status, c.supersededByID = Superseded, replacementID
	c.emitted = append(c.emitted, event)
	return nil
}

func (c Certificate) require(expected Status) error {
	if c.status != expected {
		return fmt.Errorf("invalid certificate transition from %s; expected %s", c.status, expected)
	}
	return nil
}

// RenewalContext returns the tenant/organization scope and renewal authority token
// for this certificate. The authority token is a 32-byte value set by the
// PostgreSQL persistence layer (migration 0016) during a renewal attempt.
// Callers must validate this token against the server-derived actor context.
func (c Certificate) RenewalContext() (tenantID string, organizationID string, authorityToken []byte, hasAuthority bool) {
	hasAuthority = len(c.renewalAuthorityToken) == 32
	return c.tenantID, c.organizationID, c.renewalAuthorityToken, hasAuthority
}

// SetRenewalAuthority sets the renewal authority token and increments the renewal count.
// This is called by the PostgreSQL persistence layer after a successful renewal attempt.
// The tenant_id and organization_id must match the server-derived actor context.
func (c *Certificate) SetRenewalAuthority(tenantID, organizationID string, authorityToken []byte) {
	c.tenantID = tenantID
	c.organizationID = organizationID
	if len(authorityToken) == 32 {
		c.renewalAuthorityToken = authorityToken
	}
	c.renewalCount++
}

func (c *Certificate) makeEvent(eventType string, payload any) (events.Envelope, error) {
	eventID := fmt.Sprintf("%s-%d-%d", c.id, c.revision, len(c.emitted)+1)
	return events.NewEnvelope(eventID, eventType, c.tenantID, c.organizationID, c.environment, "certificate", c.id, payload, time.Now().UTC())
}
