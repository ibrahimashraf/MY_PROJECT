package certificateauthority

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Status string

const (
	Draft         Status = "DRAFT"
	PendingReview Status = "PENDING_REVIEW"
	Approved      Status = "APPROVED"
	Finalized     Status = "FINALIZED"
	Signed        Status = "SIGNED"
	Issued        Status = "ISSUED"
	Expired       Status = "EXPIRED"
	Revoked       Status = "REVOKED"
	Superseded    Status = "SUPERSEDED"
)

// RenewalAttempt records a certificate renewal attempt with authority derivation.
type RenewalAttempt struct {
	ID                      string
	TenantID                string
	OrganizationID          string
	CertificateID           string
	AttemptedAt             time.Time
	AuthorityTenantID       string
	AuthorityOrganizationID string
	PreviousStatus          Status
	NewStatus               Status
	RenewalReason           string
	AuthoritySnapshot       json.RawMessage
	ResultState             string // SUCCESS, FAILED_AUTHORITY, FAILED_INSPECTION, FAILED_POLICY, DUPLICATE
	ErrorDetail             string
}

type Profile string

const (
	IndependentReview Profile = "INDEPENDENT_REVIEW"
	SeniorSelfIssue   Profile = "SENIOR_SELF_ISSUE"
)

type ActorContext struct {
	TenantID       string
	OrganizationID string
	ActorID        string
	Capabilities   map[string]bool
}

func (a ActorContext) allows(capability string) bool {
	return a.Capabilities != nil && a.Capabilities[capability]
}

func (a ActorContext) validate() error {
	if strings.TrimSpace(a.TenantID) == "" || strings.TrimSpace(a.OrganizationID) == "" || strings.TrimSpace(a.ActorID) == "" {
		return fmt.Errorf("tenant_id, organization_id, and actor_id are required")
	}
	return nil
}

type CanonicalInspection struct {
	ID                string
	TenantID          string
	OrganizationID    string
	AssetID           string
	InspectorID       string
	Revision          int64
	LifecycleState    string
	FinalizationState string
}

func (i CanonicalInspection) eligible() error {
	if strings.TrimSpace(i.ID) == "" || strings.TrimSpace(i.TenantID) == "" || strings.TrimSpace(i.OrganizationID) == "" || strings.TrimSpace(i.AssetID) == "" || strings.TrimSpace(i.InspectorID) == "" || i.Revision <= 0 {
		return fmt.Errorf("canonical inspection identity is incomplete")
	}
	if i.LifecycleState != string(Approved) || i.FinalizationState != string(Finalized) {
		return fmt.Errorf("inspection is not certificate eligible")
	}
	return nil
}

type Policy struct {
	ID               string
	TemplateCode     string
	TemplateVersion  int64
	Version          int64
	Status           string
	ValidityDays     int
	SelfIssueAllowed bool
}

func (p Policy) validateFor(templateCode string, templateVersion int64) error {
	if strings.TrimSpace(p.ID) == "" || p.Status != string(Approved) || p.ValidityDays <= 0 || p.ValidityDays > 3650 {
		return fmt.Errorf("certificate policy is not approved")
	}
	if p.TemplateCode != templateCode || p.TemplateVersion != templateVersion {
		return fmt.Errorf("certificate policy does not match approved template")
	}
	return nil
}

type DraftRequest struct {
	ID              string
	TemplateCode    string
	TemplateVersion int64
	Profile         Profile
	SelfIssueReason string
}

type Certificate struct {
	id                 string
	tenantID           string
	organizationID     string
	inspectionID       string
	inspectionRevision int64
	assetID            string
	inspectorID        string
	templateCode       string
	templateVersion    int64
	policyID           string
	policyVersion      int64
	profile            Profile
	status             Status
	createdBy          string
	createdAt          time.Time
	reviewedBy         string
	reviewedAt         time.Time
	signedBy           string
	signedAt           time.Time
	issuedBy           string
	issuedAt           time.Time
	expiresAt          time.Time
	selfIssueReason    string
	supersedesID       string
	supersededByID     string
	revokedBy          string
	revokedAt          time.Time
	revocationReason   string
}

func NewDraft(actor ActorContext, inspection CanonicalInspection, policy Policy, request DraftRequest, now time.Time) (*Certificate, error) {
	if err := actor.validate(); err != nil {
		return nil, err
	}
	if !actor.allows("certificate.prepare") {
		return nil, fmt.Errorf("actor lacks certificate preparation capability")
	}
	if actor.TenantID != inspection.TenantID || actor.OrganizationID != inspection.OrganizationID {
		return nil, fmt.Errorf("inspection scope does not match actor scope")
	}
	if err := inspection.eligible(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(request.ID) == "" || strings.TrimSpace(request.TemplateCode) == "" || request.TemplateVersion <= 0 || now.IsZero() {
		return nil, fmt.Errorf("certificate draft identity is incomplete")
	}
	if err := policy.validateFor(request.TemplateCode, request.TemplateVersion); err != nil {
		return nil, err
	}
	switch request.Profile {
	case IndependentReview:
		if strings.TrimSpace(request.SelfIssueReason) != "" {
			return nil, fmt.Errorf("independent review does not allow self issue reason")
		}
	case SeniorSelfIssue:
		if !policy.SelfIssueAllowed || !actor.allows("certificate.self_issue") || actor.ActorID != inspection.InspectorID || strings.TrimSpace(request.SelfIssueReason) == "" || len(request.SelfIssueReason) > 500 {
			return nil, fmt.Errorf("senior self issue is not authorized")
		}
	default:
		return nil, fmt.Errorf("certificate profile is invalid")
	}
	return &Certificate{id: request.ID, tenantID: actor.TenantID, organizationID: actor.OrganizationID, inspectionID: inspection.ID, inspectionRevision: inspection.Revision, assetID: inspection.AssetID, inspectorID: inspection.InspectorID, templateCode: request.TemplateCode, templateVersion: request.TemplateVersion, policyID: policy.ID, policyVersion: policy.Version, profile: request.Profile, status: Draft, createdBy: actor.ActorID, createdAt: now.UTC(), selfIssueReason: strings.TrimSpace(request.SelfIssueReason)}, nil
}

func (c Certificate) ID() string                { return c.id }
func (c Certificate) Status() Status            { return c.status }
func (c Certificate) Profile() Profile          { return c.profile }
func (c Certificate) InspectionID() string      { return c.inspectionID }
func (c Certificate) InspectionRevision() int64 { return c.inspectionRevision }
func (c Certificate) TemplateCode() string      { return c.templateCode }
func (c Certificate) TemplateVersion() int64    { return c.templateVersion }
func (c Certificate) PolicyID() string          { return c.policyID }
func (c Certificate) PolicyVersion() int64      { return c.policyVersion }
func (c Certificate) AssetID() string           { return c.assetID }
func (c Certificate) ExpiresAt() time.Time      { return c.expiresAt }
func (c Certificate) ReviewedBy() string        { return c.reviewedBy }
func (c Certificate) SignedBy() string          { return c.signedBy }
func (c Certificate) IssuedBy() string          { return c.issuedBy }
func (c Certificate) SelfIssueReason() string   { return c.selfIssueReason }

func (c Certificate) TenantID() string         { return c.tenantID }
func (c Certificate) OrganizationID() string   { return c.organizationID }
func (c Certificate) InspectorID() string      { return c.inspectorID }
func (c Certificate) CreatedBy() string        { return c.createdBy }
func (c Certificate) CreatedAt() time.Time     { return c.createdAt }
func (c Certificate) SignedAt() time.Time      { return c.signedAt }
func (c Certificate) IssuedAt() time.Time      { return c.issuedAt }
func (c Certificate) RevokedBy() string        { return c.revokedBy }
func (c Certificate) RevokedAt() time.Time     { return c.revokedAt }
func (c Certificate) RevocationReason() string { return c.revocationReason }
func (c Certificate) SupersedesID() string     { return c.supersedesID }
func (c Certificate) SupersededByID() string   { return c.supersededByID }

func (c *Certificate) Submit(actor ActorContext) error {
	if err := c.require(Draft); err != nil {
		return err
	}
	if err := c.authorize(actor, "certificate.prepare", false); err != nil {
		return err
	}
	c.status = PendingReview
	return nil
}

func (c *Certificate) Review(actor ActorContext, now time.Time) error {
	if err := c.require(PendingReview); err != nil {
		return err
	}
	if err := c.authorize(actor, "certificate.review", true); err != nil {
		return err
	}
	c.status, c.reviewedBy, c.reviewedAt = Approved, actor.ActorID, now.UTC()
	return nil
}

func (c *Certificate) Sign(actor ActorContext, now time.Time) error {
	if err := c.require(Approved); err != nil {
		return err
	}
	if err := c.authorize(actor, "certificate.sign", true); err != nil {
		return err
	}
	c.status, c.signedBy, c.signedAt = Signed, actor.ActorID, now.UTC()
	return nil
}

func (c *Certificate) Issue(actor ActorContext, policy Policy, now time.Time) error {
	if err := c.require(Signed); err != nil {
		return err
	}
	if err := c.authorize(actor, "certificate.issue", true); err != nil {
		return err
	}
	if policy.ID != c.policyID || policy.ValidityDays <= 0 || policy.ValidityDays > 3650 || policy.Status != string(Approved) {
		return fmt.Errorf("certificate policy is not issuable")
	}
	c.status, c.issuedBy, c.issuedAt, c.expiresAt = Issued, actor.ActorID, now.UTC(), now.UTC().AddDate(0, 0, policy.ValidityDays)
	return nil
}

func (c *Certificate) Expire(at time.Time) error {
	if err := c.require(Issued); err != nil {
		return err
	}
	if at.IsZero() || at.Before(c.expiresAt) {
		return fmt.Errorf("certificate is not expired")
	}
	c.status = Expired
	return nil
}

func (c *Certificate) Revoke(actor ActorContext, reason string, now time.Time) error {
	if c.status != Issued && c.status != Expired {
		return fmt.Errorf("certificate cannot be revoked from status %s", c.status)
	}
	if err := c.authorize(actor, "certificate.revoke", false); err != nil {
		return err
	}
	if strings.TrimSpace(reason) == "" || len(reason) > 1000 {
		return fmt.Errorf("revocation reason is required")
	}
	c.status, c.revokedBy, c.revokedAt, c.revocationReason = Revoked, actor.ActorID, now.UTC(), strings.TrimSpace(reason)
	return nil
}

func (c *Certificate) Supersede(actor ActorContext, replacementID string) error {
	if c.status != Issued && c.status != Expired {
		return fmt.Errorf("certificate cannot be superseded from status %s", c.status)
	}
	if err := c.authorize(actor, "certificate.supersede", false); err != nil {
		return err
	}
	if strings.TrimSpace(replacementID) == "" || replacementID == c.id {
		return fmt.Errorf("replacement certificate id is invalid")
	}
	c.status, c.supersededByID = Superseded, replacementID
	return nil
}

func (c *Certificate) SetSupersedes(replacedID string) error {
	if strings.TrimSpace(replacedID) == "" || replacedID == c.id {
		return fmt.Errorf("superseded certificate id is invalid")
	}
	c.supersedesID = replacedID
	return nil
}

func (c *Certificate) require(expected Status) error {
	if c.status != expected {
		return fmt.Errorf("invalid certificate transition from %s; expected %s", c.status, expected)
	}
	return nil
}

func (c *Certificate) authorize(actor ActorContext, capability string, appliesSeparation bool) error {
	if err := actor.validate(); err != nil {
		return err
	}
	if actor.TenantID != c.tenantID || actor.OrganizationID != c.organizationID {
		return fmt.Errorf("actor scope does not match certificate scope")
	}
	if !actor.allows(capability) {
		return fmt.Errorf("actor lacks %s capability", capability)
	}
	if !appliesSeparation {
		return nil
	}
	if c.profile == IndependentReview && actor.ActorID == c.inspectorID {
		return fmt.Errorf("independent review forbids inspector authority")
	}
	if c.profile == SeniorSelfIssue && (actor.ActorID != c.inspectorID || !actor.allows("certificate.self_issue")) {
		return fmt.Errorf("senior self issue authority is not satisfied")
	}
	return nil
}
