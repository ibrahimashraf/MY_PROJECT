package workorder

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type RequestState string

const (
	RequestDraft     RequestState = "draft"
	RequestRequested RequestState = "requested"
	RequestAccepted  RequestState = "accepted"
	RequestCancelled RequestState = "cancelled"
)

type ExecutionState string

const (
	ExecutionReady              ExecutionState = "ready"
	ExecutionAssigned           ExecutionState = "assigned"
	ExecutionInProgress         ExecutionState = "in_progress"
	ExecutionPartiallySubmitted ExecutionState = "partially_submitted"
	ExecutionAwaitingClient     ExecutionState = "awaiting_client"
	ExecutionAwaitingReview     ExecutionState = "awaiting_review"
	ExecutionCompleted          ExecutionState = "completed"
)

type CommercialState string

const (
	CommercialNotReady             CommercialState = "not_ready_for_invoice"
	CommercialReadyForOfficeReview CommercialState = "ready_for_office_confirmation"
	CommercialReleasedForInvoice   CommercialState = "released_for_invoice"
	CommercialInvoiced             CommercialState = "invoiced"
	CommercialClosed               CommercialState = "closed"
)

type CertificateState string

const (
	CertificateNotStarted        CertificateState = "not_started"
	CertificatePendingValidation CertificateState = "pending_validation"
	CertificatePartiallyIssued   CertificateState = "partially_issued"
	CertificateIssued            CertificateState = "issued"
	CertificateNeedsCorrection   CertificateState = "needs_correction"
	CertificateRevoked           CertificateState = "revoked_or_superseded"
)

type AssignmentState string

const (
	AssignmentActive      AssignmentState = "active"
	AssignmentTransferred AssignmentState = "transferred"
	AssignmentCompleted   AssignmentState = "completed"
	AssignmentRevoked     AssignmentState = "revoked"
)

type RecordKind string

const (
	RecordClient     RecordKind = "client"
	RecordWorkOrder  RecordKind = "work_order"
	RecordLocation   RecordKind = "location"
	RecordAsset      RecordKind = "asset"
	RecordInspection RecordKind = "inspection"
)

var (
	ErrInvalidIdentity     = errors.New("work-order identity is incomplete")
	ErrInvalidTransition   = errors.New("work-order lifecycle transition is not allowed")
	ErrInvalidScope        = errors.New("work-order scope is incomplete")
	ErrInvalidAssignment   = errors.New("work-order assignment is incomplete")
	ErrInvalidOperation    = errors.New("operation idempotency metadata is incomplete")
	ErrProvisionalConflict = errors.New("provisional record candidate fingerprint conflict")
)

type WorkOrder struct {
	ID               string           `json:"work_order_id"`
	JobNumber        string           `json:"job_number"`
	TenantID         string           `json:"tenant_id"`
	OrganizationID   string           `json:"organization_id"`
	ClientID         string           `json:"client_id"`
	RequestState     RequestState     `json:"request_state"`
	ExecutionState   ExecutionState   `json:"execution_state"`
	CommercialState  CommercialState  `json:"commercial_state"`
	CertificateState CertificateState `json:"certificate_state"`
	Revision         int64            `json:"revision"`
}

func (w WorkOrder) ValidateIdentity() error {
	fields := map[string]string{"work_order_id": w.ID, "job_number": w.JobNumber, "tenant_id": w.TenantID, "organization_id": w.OrganizationID, "client_id": w.ClientID}
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%w: %s", ErrInvalidIdentity, name)
		}
	}
	if w.Revision < 1 {
		return fmt.Errorf("%w: revision must be positive", ErrInvalidIdentity)
	}
	return nil
}

type ScopeItem struct {
	ID             string `json:"scope_item_id"`
	TenantID       string `json:"tenant_id"`
	OrganizationID string `json:"organization_id"`
	ClientID       string `json:"client_id"`
	LocationID     string `json:"location_id"`
	AssetID        string `json:"asset_id"`
	AssetType      string `json:"asset_type"`
}

func (s ScopeItem) ValidateFor(w WorkOrder) error {
	if strings.TrimSpace(s.ID) == "" || strings.TrimSpace(s.TenantID) == "" || strings.TrimSpace(s.ClientID) == "" {
		return fmt.Errorf("%w: scope identity", ErrInvalidScope)
	}
	if s.TenantID != w.TenantID || s.OrganizationID != w.OrganizationID || s.ClientID != w.ClientID {
		return fmt.Errorf("%w: tenant or client mismatch", ErrInvalidScope)
	}
	if strings.TrimSpace(s.LocationID) == "" || strings.TrimSpace(s.AssetID) == "" || strings.TrimSpace(s.AssetType) == "" {
		return fmt.Errorf("%w: location, asset, and asset type are required", ErrInvalidScope)
	}
	return nil
}

type Assignment struct {
	ID             string          `json:"assignment_id"`
	TenantID       string          `json:"tenant_id"`
	OrganizationID string          `json:"organization_id"`
	WorkOrderID    string          `json:"work_order_id"`
	InspectorID    string          `json:"inspector_id"`
	ScopeItemIDs   []string        `json:"scope_item_ids"`
	State          AssignmentState `json:"state"`
	Revision       int64           `json:"revision"`
	EffectiveFrom  time.Time       `json:"effective_from"`
	EffectiveUntil *time.Time      `json:"effective_until,omitempty"`
}

func (a Assignment) ValidateFor(w WorkOrder) error {
	if strings.TrimSpace(a.ID) == "" || strings.TrimSpace(a.TenantID) == "" || strings.TrimSpace(a.WorkOrderID) == "" || strings.TrimSpace(a.InspectorID) == "" {
		return fmt.Errorf("%w: assignment identity", ErrInvalidAssignment)
	}
	if a.TenantID != w.TenantID || a.OrganizationID != w.OrganizationID || a.WorkOrderID != w.ID {
		return fmt.Errorf("%w: tenant or work-order mismatch", ErrInvalidAssignment)
	}
	if len(a.ScopeItemIDs) == 0 {
		return fmt.Errorf("%w: at least one scope item is required", ErrInvalidAssignment)
	}
	if a.Revision < 1 || a.EffectiveFrom.IsZero() {
		return fmt.Errorf("%w: revision and effective_from are required", ErrInvalidAssignment)
	}
	return nil
}

type OperationMeta struct {
	OperationID      string `json:"operation_id"`
	IdempotencyKey   string `json:"idempotency_key"`
	ExpectedRevision int64  `json:"expected_revision"`
}

func (m OperationMeta) Validate() error {
	if strings.TrimSpace(m.OperationID) == "" || strings.TrimSpace(m.IdempotencyKey) == "" || m.ExpectedRevision < 1 {
		return ErrInvalidOperation
	}
	return nil
}

type ProvisionalRecord struct {
	LocalID        string     `json:"local_id"`
	Kind           RecordKind `json:"kind"`
	TenantID       string     `json:"tenant_id"`
	OrganizationID string     `json:"organization_id"`
	WorkOrderID    string     `json:"work_order_id,omitempty"`
	ClientID       string     `json:"client_id,omitempty"`
	CanonicalID    string     `json:"canonical_id,omitempty"`
	ReconcileState string     `json:"reconcile_state"`
}

const (
	ReconcilePending  = "pending"
	ReconcileMatched  = "matched"
	ReconcileCreated  = "created"
	ReconcileConflict = "conflict"
	ReconcileRejected = "rejected"
)

func ValidateProvisionalFingerprint(existing, incoming string) error {
	if strings.TrimSpace(existing) == "" || strings.TrimSpace(incoming) == "" {
		return fmt.Errorf("%w: provisional candidate fingerprint", ErrInvalidIdentity)
	}
	if existing != incoming {
		return ErrProvisionalConflict
	}
	return nil
}

func (p ProvisionalRecord) Validate() error {
	if strings.TrimSpace(p.LocalID) == "" || strings.TrimSpace(p.TenantID) == "" || strings.TrimSpace(p.OrganizationID) == "" || p.Kind == "" {
		return fmt.Errorf("%w: provisional record identity", ErrInvalidIdentity)
	}
	if p.CanonicalID != "" && p.ReconcileState == ReconcilePending {
		return fmt.Errorf("%w: canonical ID cannot exist while reconciliation is pending", ErrInvalidIdentity)
	}
	return nil
}

func CanTransition(from, to ExecutionState) bool {
	if from == to {
		return true
	}
	allowed := map[ExecutionState]map[ExecutionState]bool{
		ExecutionReady:              {ExecutionAssigned: true},
		ExecutionAssigned:           {ExecutionInProgress: true},
		ExecutionInProgress:         {ExecutionPartiallySubmitted: true, ExecutionAwaitingClient: true, ExecutionAwaitingReview: true, ExecutionCompleted: true},
		ExecutionPartiallySubmitted: {ExecutionInProgress: true, ExecutionAwaitingReview: true, ExecutionCompleted: true},
		ExecutionAwaitingClient:     {ExecutionInProgress: true, ExecutionCompleted: true},
		ExecutionAwaitingReview:     {ExecutionInProgress: true, ExecutionCompleted: true},
	}
	return allowed[from][to]
}
