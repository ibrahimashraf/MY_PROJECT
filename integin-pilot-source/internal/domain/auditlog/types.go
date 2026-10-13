package auditlog

import (
	"errors"
	"time"
)

type EntityType string

const (
	EntityInspection  EntityType = "inspection"
	EntityWorkOrder   EntityType = "work_order"
	EntityCertificate EntityType = "certificate"
	EntityAsset       EntityType = "asset"
	EntityIdentity    EntityType = "identity"
)

type Action string

const (
	ActionCreate  Action = "create"
	ActionUpdate  Action = "update"
	ActionDelete  Action = "delete"
	ActionApprove Action = "approve"
	ActionReject  Action = "reject"
)

var (
	ErrInvalidEntry = errors.New("audit log entry is invalid")
	ErrNotFound     = errors.New("audit log entry not found")
	ErrChainBroken  = errors.New("audit log hash chain is broken")
)

type Entry struct {
	ID             string                 `json:"id"`
	TenantID       string                 `json:"tenant_id"`
	OrganizationID string                 `json:"organization_id"`
	EventType      string                 `json:"event_type"`
	EntityType     EntityType             `json:"entity_type"`
	EntityID       string                 `json:"entity_id"`
	ActorID        string                 `json:"actor_id"`
	ActorName      string                 `json:"actor_name"`
	Action         Action                 `json:"action"`
	OldValue       map[string]interface{} `json:"old_value,omitempty"`
	NewValue       map[string]interface{} `json:"new_value,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	IPAddress      string                 `json:"ip_address"`
	UserAgent      string                 `json:"user_agent"`
	PreviousHash   string                 `json:"previous_hash"`
	EntryHash      string                 `json:"entry_hash"`
	CreatedAt      time.Time              `json:"created_at"`
	ValidTime      time.Time              `json:"valid_time,omitempty"`
}

// ValidAt returns the time the audited event occurred (valid time),
// defaulting to the record time when ValidTime is unset.
func (e Entry) ValidAt() time.Time {
	if e.ValidTime.IsZero() {
		return e.CreatedAt
	}
	return e.ValidTime
}

type CreateEntryRequest struct {
	TenantID       string                 `json:"tenant_id"`
	OrganizationID string                 `json:"organization_id"`
	EventType      string                 `json:"event_type"`
	EntityType     EntityType             `json:"entity_type"`
	EntityID       string                 `json:"entity_id"`
	ActorID        string                 `json:"actor_id"`
	ActorName      string                 `json:"actor_name"`
	Action         Action                 `json:"action"`
	OldValue       map[string]interface{} `json:"old_value,omitempty"`
	NewValue       map[string]interface{} `json:"new_value,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	IPAddress      string                 `json:"ip_address"`
	UserAgent      string                 `json:"user_agent"`
}

func (r CreateEntryRequest) Validate() error {
	if r.TenantID == "" || r.OrganizationID == "" {
		return ErrInvalidEntry
	}
	if r.EventType == "" || r.EntityType == "" || r.EntityID == "" || r.ActorID == "" {
		return ErrInvalidEntry
	}
	if r.Action == "" {
		return ErrInvalidEntry
	}
	return nil
}

type QueryRequest struct {
	TenantID       string     `json:"tenant_id"`
	OrganizationID string     `json:"organization_id"`
	EntityType     EntityType `json:"entity_type"`
	EntityID       string     `json:"entity_id"`
	ActorID        string     `json:"actor_id"`
	EventType      string     `json:"event_type"`
	From           *time.Time `json:"from"`
	To             *time.Time `json:"to"`
	ValidFrom      *time.Time `json:"valid_from"`
	ValidTo        *time.Time `json:"valid_to"`
	Limit          int        `json:"limit"`
	Offset         int        `json:"offset"`
}

type QueryResponse struct {
	Entries []Entry `json:"entries"`
	Total   int     `json:"total"`
}

type VerifyResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}
