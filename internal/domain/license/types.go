package license

import (
	"errors"
	"strings"
	"time"
)

type Tier string

const (
	TierStarter    Tier = "starter"
	TierPro        Tier = "pro"
	TierEnterprise Tier = "enterprise"
)

type Status string

const (
	StatusTrial   Status = "trial"
	StatusActive  Status = "active"
	StatusExpired Status = "expired"
	StatusRevoked Status = "revoked"
)

type Action string

const (
	ActionIssued        Action = "issued"
	ActionRenewed       Action = "renewed"
	ActionExpired       Action = "expired"
	ActionRevoked       Action = "revoked"
	ActionTrialExtended Action = "trial_extended"
)

var (
	ErrInvalidLicense    = errors.New("license is invalid or expired")
	ErrEntitlementExceed = errors.New("entitlement limit exceeded")
	ErrLicenseNotFound   = errors.New("license not found")
	ErrInvalidIdentity   = errors.New("license identity is incomplete")
)

type License struct {
	ID                     string          `json:"id"`
	TenantID               string          `json:"tenant_id"`
	OrganizationID         string          `json:"organization_id"`
	Tier                   Tier            `json:"tier"`
	Status                 Status          `json:"status"`
	IssuedAt               time.Time       `json:"issued_at"`
	ExpiresAt              *time.Time      `json:"expires_at,omitempty"`
	MaxInspectors          int             `json:"max_inspectors"`
	MaxInspectionsPerMonth int             `json:"max_inspections_per_month"`
	Features               map[string]bool `json:"features"`
	CreatedBy              string          `json:"created_by"`
	CreatedAt              time.Time       `json:"created_at"`
}

func (l License) Validate() error {
	fields := map[string]string{
		"id":              l.ID,
		"tenant_id":       l.TenantID,
		"organization_id": l.OrganizationID,
		"created_by":      l.CreatedBy,
	}
	for _, value := range fields {
		if strings.TrimSpace(value) == "" {
			return ErrInvalidIdentity
		}
	}
	if l.Tier == "" {
		return ErrInvalidIdentity
	}
	return nil
}

func (l License) IsExpired(at time.Time) bool {
	if l.ExpiresAt == nil {
		return false
	}
	return !at.Before(*l.ExpiresAt)
}

type Validation struct {
	Valid                  bool            `json:"valid"`
	Tier                   Tier            `json:"tier"`
	Status                 Status          `json:"status"`
	ExpiresAt              *time.Time      `json:"expires_at,omitempty"`
	MaxInspectors          int             `json:"max_inspectors"`
	MaxInspectionsPerMonth int             `json:"max_inspections_per_month"`
	Features               map[string]bool `json:"features"`
}

type AuditEntry struct {
	ID             string                 `json:"id"`
	TenantID       string                 `json:"tenant_id"`
	OrganizationID string                 `json:"organization_id"`
	LicenseID      string                 `json:"license_id"`
	Action         Action                 `json:"action"`
	ActorID        string                 `json:"actor_id"`
	OccurredAt     time.Time              `json:"occurred_at"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}
