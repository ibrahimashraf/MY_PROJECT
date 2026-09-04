package custodyhandover

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidActor      = errors.New("custody handover actor context is invalid")
	ErrInvalidCustody    = errors.New("custody record is invalid")
	ErrInvalidHandover   = errors.New("handover request is invalid")
	ErrInvalidTransition = errors.New("invalid custody handover transition")
	ErrStaleRevision     = errors.New("stale revision")
	ErrNotFound          = errors.New("custody or handover record not found")
	ErrTenantMismatch    = errors.New("tenant or organization mismatch")
)

// HandoverState represents the lifecycle of a work-order/asset handover.
type HandoverState string

const (
	HandoverStateRequested    HandoverState = "REQUESTED"
	HandoverStateAcknowledged HandoverState = "ACKNOWLEDGED"
	HandoverStateApproved     HandoverState = "APPROVED"
	HandoverStateTransferred  HandoverState = "TRANSFERRED"
	HandoverStateRejected     HandoverState = "REJECTED"
	HandoverStateCancelled    HandoverState = "CANCELLED"
)

// CustodyAction represents the action recorded in the chain of custody.
type CustodyAction string

const (
	CustodyActionCheckIn   CustodyAction = "CHECK_IN"
	CustodyActionCheckOut  CustodyAction = "CHECK_OUT"
	CustodyActionTransfer  CustodyAction = "TRANSFER"
	CustodyActionRelocate  CustodyAction = "RELOCATE"
	CustodyActionDispose   CustodyAction = "DISPOSE"
)

// ActorContext is the server-derived actor executing the handover/custody action.
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

// CustodyRecord represents an immutable audit entry in the chain of custody.
type CustodyRecord struct {
	ID             string        `json:"id"`
	TenantID       string        `json:"tenant_id"`
	OrganizationID string        `json:"organization_id"`
	AssetID        string        `json:"asset_id"`
	WorkOrderID    string        `json:"work_order_id,omitempty"`
	LocationID     string        `json:"location_id"`
	SiteName       string        `json:"site_name"`
	CustodianID    string        `json:"custodian_id"`
	Action         CustodyAction `json:"action"`
	VerificationHash string      `json:"verification_hash"`
	Notes          string        `json:"notes,omitempty"`
	RecordedBy     string        `json:"recorded_by"`
	RecordedAt     time.Time     `json:"recorded_at"`
}

func (c CustodyRecord) Validate() error {
	if strings.TrimSpace(c.ID) == "" {
		return fmt.Errorf("%w: missing id", ErrInvalidCustody)
	}
	if strings.TrimSpace(c.TenantID) == "" || strings.TrimSpace(c.OrganizationID) == "" {
		return fmt.Errorf("%w: missing tenant/org", ErrInvalidCustody)
	}
	if strings.TrimSpace(c.AssetID) == "" {
		return fmt.Errorf("%w: missing asset_id", ErrInvalidCustody)
	}
	if strings.TrimSpace(c.LocationID) == "" || strings.TrimSpace(c.SiteName) == "" {
		return fmt.Errorf("%w: missing location or site", ErrInvalidCustody)
	}
	if strings.TrimSpace(c.CustodianID) == "" {
		return fmt.Errorf("%w: missing custodian_id", ErrInvalidCustody)
	}
	switch c.Action {
	case CustodyActionCheckIn, CustodyActionCheckOut, CustodyActionTransfer, CustodyActionRelocate, CustodyActionDispose:
	default:
		return fmt.Errorf("%w: unsupported action %q", ErrInvalidCustody, c.Action)
	}
	if strings.TrimSpace(c.VerificationHash) == "" {
		return fmt.Errorf("%w: verification_hash is required", ErrInvalidCustody)
	}
	if c.RecordedAt.IsZero() {
		return fmt.Errorf("%w: recorded_at is required", ErrInvalidCustody)
	}
	if strings.TrimSpace(c.RecordedBy) == "" {
		return fmt.Errorf("%w: recorded_by is required", ErrInvalidCustody)
	}
	return nil
}

// ComputeVerificationHash calculates an immutable checksum for custody entry verification.
func ComputeVerificationHash(tenantID, orgID, assetID, locationID, custodianID string, action CustodyAction, ts time.Time) string {
	raw := fmt.Sprintf("%s:%s:%s:%s:%s:%s:%d", tenantID, orgID, assetID, locationID, custodianID, action, ts.UnixNano())
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum)
}

// HandoverRequest tracks formal transfer of work-order responsibility between custodians/sites.
type HandoverRequest struct {
	ID               string        `json:"id"`
	TenantID         string        `json:"tenant_id"`
	OrganizationID   string        `json:"organization_id"`
	WorkOrderID      string        `json:"work_order_id"`
	AssetID          string        `json:"asset_id,omitempty"`
	FromAssignmentID string        `json:"from_assignment_id"`
	ToAssignmentID   string        `json:"to_assignment_id"`
	FromSiteID       string        `json:"from_site_id"`
	ToSiteID         string        `json:"to_site_id"`
	State            HandoverState `json:"state"`
	Reason           string        `json:"reason"`
	RequestedBy      string        `json:"requested_by"`
	RequestedAt      time.Time     `json:"requested_at"`
	AcknowledgedBy   string        `json:"acknowledged_by,omitempty"`
	AcknowledgedAt   *time.Time    `json:"acknowledged_at,omitempty"`
	ApprovedBy       string        `json:"approved_by,omitempty"`
	ApprovedAt       *time.Time    `json:"approved_at,omitempty"`
	TransferredBy    string        `json:"transferred_by,omitempty"`
	TransferredAt    *time.Time    `json:"transferred_at,omitempty"`
	RejectionReason  string        `json:"rejection_reason,omitempty"`
	Revision         int64         `json:"revision"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

func (h HandoverRequest) Validate() error {
	if strings.TrimSpace(h.ID) == "" {
		return fmt.Errorf("%w: missing id", ErrInvalidHandover)
	}
	if strings.TrimSpace(h.TenantID) == "" || strings.TrimSpace(h.OrganizationID) == "" {
		return fmt.Errorf("%w: missing tenant/org", ErrInvalidHandover)
	}
	if strings.TrimSpace(h.WorkOrderID) == "" {
		return fmt.Errorf("%w: missing work_order_id", ErrInvalidHandover)
	}
	if strings.TrimSpace(h.FromAssignmentID) == "" || strings.TrimSpace(h.ToAssignmentID) == "" {
		return fmt.Errorf("%w: missing from/to assignment IDs", ErrInvalidHandover)
	}
	if h.FromAssignmentID == h.ToAssignmentID {
		return fmt.Errorf("%w: from_assignment_id and to_assignment_id must differ", ErrInvalidHandover)
	}
	if strings.TrimSpace(h.FromSiteID) == "" || strings.TrimSpace(h.ToSiteID) == "" {
		return fmt.Errorf("%w: missing from/to site IDs", ErrInvalidHandover)
	}
	if h.Revision < 1 {
		return fmt.Errorf("%w: revision must be >= 1", ErrInvalidHandover)
	}
	if strings.TrimSpace(h.RequestedBy) == "" || h.RequestedAt.IsZero() {
		return fmt.Errorf("%w: requested_by and requested_at are required", ErrInvalidHandover)
	}

	switch h.State {
	case HandoverStateRequested:
	case HandoverStateAcknowledged:
		if strings.TrimSpace(h.AcknowledgedBy) == "" || h.AcknowledgedAt == nil {
			return fmt.Errorf("%w: acknowledged_by and acknowledged_at required for ACKNOWLEDGED state", ErrInvalidHandover)
		}
	case HandoverStateApproved:
		if strings.TrimSpace(h.ApprovedBy) == "" || h.ApprovedAt == nil {
			return fmt.Errorf("%w: approved_by and approved_at required for APPROVED state", ErrInvalidHandover)
		}
	case HandoverStateTransferred:
		if strings.TrimSpace(h.TransferredBy) == "" || h.TransferredAt == nil {
			return fmt.Errorf("%w: transferred_by and transferred_at required for TRANSFERRED state", ErrInvalidHandover)
		}
	case HandoverStateRejected:
		if strings.TrimSpace(h.RejectionReason) == "" {
			return fmt.Errorf("%w: rejection_reason required for REJECTED state", ErrInvalidHandover)
		}
	case HandoverStateCancelled:
	default:
		return fmt.Errorf("%w: unknown handover state %q", ErrInvalidHandover, h.State)
	}
	return nil
}

// CanTransition determines valid state progression for a HandoverRequest.
func CanTransition(current, next HandoverState) bool {
	if current == next {
		return true
	}
	transitions := map[HandoverState]map[HandoverState]bool{
		HandoverStateRequested: {
			HandoverStateAcknowledged: true,
			HandoverStateApproved:     true, // Direct supervisor approval allowed
			HandoverStateRejected:     true,
			HandoverStateCancelled:    true,
		},
		HandoverStateAcknowledged: {
			HandoverStateApproved:  true,
			HandoverStateRejected:  true,
			HandoverStateCancelled: true,
		},
		HandoverStateApproved: {
			HandoverStateTransferred: true,
			HandoverStateCancelled:   true,
		},
	}
	return transitions[current][next]
}

// Service defines domain operations on custody and handover.
type Service interface {
	RecordCustody(ctx context.Context, actor ActorContext, record CustodyRecord) (CustodyRecord, error)
	RequestHandover(ctx context.Context, actor ActorContext, req HandoverRequest) (HandoverRequest, error)
	AcknowledgeHandover(ctx context.Context, actor ActorContext, id string, expectedRev int64) (HandoverRequest, error)
	ApproveHandover(ctx context.Context, actor ActorContext, id string, expectedRev int64) (HandoverRequest, error)
	CompleteHandover(ctx context.Context, actor ActorContext, id string, expectedRev int64) (HandoverRequest, error)
	RejectHandover(ctx context.Context, actor ActorContext, id string, reason string, expectedRev int64) (HandoverRequest, error)
	CancelHandover(ctx context.Context, actor ActorContext, id string, expectedRev int64) (HandoverRequest, error)
}
