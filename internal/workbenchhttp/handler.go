package workbenchhttp

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"integin/internal/domain/device_trust"
	"integin/internal/shared/httpresponse"
)

// Sentinel store errors mapped to HTTP status codes at the handler boundary.
var (
	ErrNotFound   = errors.New("workbench resource not found")
	ErrConflict   = errors.New("workbench resource state conflict")
	ErrValidation = errors.New("workbench request validation failed")
)

const (
	statusActive   = "ACTIVE"
	statusReleased = "RELEASED"
	statusApproved = "APPROVED"
	statusRejected = "REJECTED"
	statusPending  = "PENDING"
	statusHeld     = "HELD"
	statusTrusted  = "TRUSTED"
	statusRevoked  = "REVOKED"
)

// DeviceView is the privacy-scrubbed projection of a registered device. It
// deliberately exposes only public device identity and trust posture — never
// private key material or raw attestation evidence.
type DeviceView struct {
	ID               string     `json:"id"`
	TenantID         string     `json:"tenant_id"`
	OrganizationID   string     `json:"organization_id"`
	UserID           string     `json:"user_id"`
	PublicKey        string     `json:"public_key"`
	State            string     `json:"state"`
	Epoch            uint64     `json:"epoch"`
	EnrolledAt       time.Time  `json:"enrolled_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	RevocationReason string     `json:"revocation_reason,omitempty"`
}

// EnrollmentRequestView is the administrative projection of a device
// enrollment request. Proof-of-possession credentials (nonce, signature) are
// withheld; only tenant-facing posture and lifecycle fields are exposed.
type EnrollmentRequestView struct {
	RequestID      string                             `json:"request_id"`
	TenantID       string                             `json:"tenant_id"`
	OrganizationID string                             `json:"organization_id"`
	UserID         string                             `json:"user_id"`
	DeviceID       string                             `json:"device_id"`
	PublicKey      string                             `json:"public_key"`
	Attestation    device_trust.EnrollmentAttestation `json:"attestation,omitempty"`
	Status         string                             `json:"status"`
	RequestedAt    time.Time                          `json:"requested_at"`
	ApprovedBy     string                             `json:"approved_by,omitempty"`
	RejectedBy     string                             `json:"rejected_by,omitempty"`
}

// HeldTransaction is the held-sync projection. The raw offline payload is
// never returned — only its SHA-256 hash plus sequencing metadata.
type HeldTransaction struct {
	TransactionID  string     `json:"transaction_id"`
	DeviceID       string     `json:"device_id"`
	SequenceNumber uint64     `json:"sequence_number"`
	Status         string     `json:"status"`
	ErrorReason    string     `json:"error_reason"`
	HeldAt         time.Time  `json:"held_at"`
	LastAttemptAt  *time.Time `json:"last_attempt_at,omitempty"`
	PayloadHash    string     `json:"payload_hash"`
}

// ReconcileResult reports one reconciliation retry attempt on a held transaction.
type ReconcileResult struct {
	TransactionID string    `json:"transaction_id"`
	Status        string    `json:"status"`
	RetriedAt     time.Time `json:"retried_at"`
	Attempt       int       `json:"attempt"`
	ErrorReason   string    `json:"error_reason,omitempty"`
}

// LegalHold is a tenant-scoped evidence retention hold.
type LegalHold struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	EntityType string     `json:"entity_type"`
	EntityID   string     `json:"entity_id"`
	Reason     string     `json:"reason"`
	PlacedBy   string     `json:"placed_by"`
	PlacedAt   time.Time  `json:"placed_at"`
	Status     string     `json:"status"`
	ReleasedBy *string    `json:"released_by,omitempty"`
	ReleasedAt *time.Time `json:"released_at,omitempty"`
}

// RetentionPolicy is a tenant-scoped disposition schedule.
type RetentionPolicy struct {
	TenantID      string    `json:"tenant_id"`
	EntityType    string    `json:"entity_type"`
	RetentionDays int       `json:"retention_days"`
	Action        string    `json:"action"`
	UpdatedAt     time.Time `json:"updated_at"`
	UpdatedBy     string    `json:"updated_by"`
}

// ExportApproval is a tenant-scoped decision ledger entry for an evidence export.
type ExportApproval struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	ExportID        string     `json:"export_id"`
	RequestedBy     string     `json:"requested_by"`
	ApprovedBy      string     `json:"approved_by,omitempty"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	DecidedAt       *time.Time `json:"decided_at,omitempty"`
	RejectionReason string     `json:"rejection_reason,omitempty"`
}

type workbenchStore interface {
	ListDevices(ctx context.Context, tenantID, organizationID string) ([]DeviceView, error)
	ListEnrollmentRequests(ctx context.Context, tenantID, organizationID string) ([]EnrollmentRequestView, error)
	ApproveEnrollment(ctx context.Context, actorID, tenantID, organizationID, requestID string, epoch uint64) (DeviceView, error)
	RevokeDevice(ctx context.Context, tenantID, organizationID, deviceID, reason string) (DeviceView, error)
	ListHeld(ctx context.Context, tenantID, organizationID string) ([]HeldTransaction, error)
	ReconcileHeld(ctx context.Context, tenantID, organizationID, transactionID string) (ReconcileResult, error)
	ListLegalHolds(ctx context.Context, tenantID, organizationID string) ([]LegalHold, error)
	PlaceLegalHold(ctx context.Context, actorID, tenantID, organizationID, entityType, entityID, reason string) (LegalHold, error)
	ReleaseLegalHold(ctx context.Context, actorID, tenantID, organizationID, id string) (LegalHold, error)
	ListRetentionPolicies(ctx context.Context, tenantID, organizationID string) ([]RetentionPolicy, error)
	ListExportApprovals(ctx context.Context, tenantID, organizationID string) ([]ExportApproval, error)
	DecideExportApproval(ctx context.Context, actorID, tenantID, organizationID, exportID, status, rejectionReason string) (ExportApproval, error)
}

// Handler serves the authority REST endpoints consumed by the TypeScript
// Operations Workbench. When DB is nil the handler runs against a hermetic
// in-memory store so unit tests never need a live Postgres container.
type Handler struct {
	store workbenchStore
	log   *slog.Logger
}

// NewHandler builds a workbench authority handler. A nil DB selects the
// in-memory store, seeded from the supplied devices for read-only listing.
func NewHandler(db *sql.DB, seedDevices []device_trust.Device) *Handler {
	h := &Handler{log: slog.Default()}
	if db == nil {
		h.store = newMemoryStore(seedDevices)
		return h
	}
	h.store = &postgresStore{db: db}
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	organizationID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))
	if tenantID == "" || organizationID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing required tenant headers X-Tenant-ID / X-Organization-ID")
		return
	}
	path := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1"), "/")
	actorID := strings.TrimSpace(r.Header.Get("X-Actor-ID"))
	if actorID == "" {
		actorID = "system"
	}

	switch {
	case path == "/devices" && r.Method == http.MethodGet:
		h.listDevices(w, r, tenantID, organizationID)
	case path == "/devices/approve" && r.Method == http.MethodPost:
		h.approveDevice(w, r, tenantID, organizationID, actorID)
	case path == "/devices/revoke" && r.Method == http.MethodPost:
		h.revokeDevice(w, r, tenantID, organizationID)
	case path == "/sync/held" && r.Method == http.MethodGet:
		h.listHeld(w, r, tenantID, organizationID)
	case path == "/sync/reconcile" && r.Method == http.MethodPost:
		h.reconcileHeld(w, r, tenantID, organizationID)
	case path == "/legal-holds" && r.Method == http.MethodGet:
		h.listLegalHolds(w, r, tenantID, organizationID)
	case path == "/legal-holds" && r.Method == http.MethodPost:
		h.placeLegalHold(w, r, tenantID, organizationID, actorID)
	case path == "/legal-holds/release" && r.Method == http.MethodPost:
		h.releaseLegalHold(w, r, tenantID, organizationID, actorID)
	case path == "/retention-policies" && r.Method == http.MethodGet:
		h.retentionOverview(w, r, tenantID, organizationID)
	case path == "/exports/approvals" && r.Method == http.MethodPost:
		h.decideExportApproval(w, r, tenantID, organizationID, actorID)
	case knownPath(path):
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method not allowed")
	default:
		httpresponse.Error(w, http.StatusNotFound, "endpoint not found")
	}
}

func knownPath(path string) bool {
	switch path {
	case "/devices", "/devices/approve", "/devices/revoke", "/sync/held", "/sync/reconcile",
		"/legal-holds", "/legal-holds/release", "/retention-policies", "/exports/approvals":
		return true
	}
	return false
}

func (h *Handler) listDevices(w http.ResponseWriter, r *http.Request, tenantID, orgID string) {
	devices, err := h.store.ListDevices(r.Context(), tenantID, orgID)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	requests, err := h.store.ListEnrollmentRequests(r.Context(), tenantID, orgID)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, struct {
		Devices            []DeviceView            `json:"devices"`
		EnrollmentRequests []EnrollmentRequestView `json:"enrollment_requests"`
	}{Devices: devices, EnrollmentRequests: requests})
}

type approveDeviceRequest struct {
	RequestID string `json:"request_id"`
	Epoch     uint64 `json:"epoch"`
	Duration  int64  `json:"duration"`
}

func (h *Handler) approveDevice(w http.ResponseWriter, r *http.Request, tenantID, orgID, actorID string) {
	var req approveDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.RequestID) == "" || req.Epoch == 0 || req.Duration <= 0 {
		httpresponse.Error(w, http.StatusBadRequest, "request_id, positive epoch, and positive duration are required")
		return
	}
	device, err := h.store.ApproveEnrollment(r.Context(), actorID, tenantID, orgID, req.RequestID, req.Epoch)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, struct {
		Device   DeviceView `json:"device"`
		Approved bool       `json:"approved"`
	}{Device: device, Approved: true})
}

type revokeDeviceRequest struct {
	DeviceID string `json:"device_id"`
	Reason   string `json:"reason"`
}

func (h *Handler) revokeDevice(w http.ResponseWriter, r *http.Request, tenantID, orgID string) {
	var req revokeDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.DeviceID) == "" || strings.TrimSpace(req.Reason) == "" {
		httpresponse.Error(w, http.StatusBadRequest, "device_id and reason are required")
		return
	}
	device, err := h.store.RevokeDevice(r.Context(), tenantID, orgID, req.DeviceID, req.Reason)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, struct {
		Device  DeviceView `json:"device"`
		Revoked bool       `json:"revoked"`
	}{Device: device, Revoked: device.State == statusRevoked})
}

func (h *Handler) listHeld(w http.ResponseWriter, r *http.Request, tenantID, orgID string) {
	held, err := h.store.ListHeld(r.Context(), tenantID, orgID)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, struct {
		HeldTransactions []HeldTransaction `json:"held_transactions"`
	}{HeldTransactions: held})
}

type reconcileRequest struct {
	TransactionID string `json:"transaction_id"`
}

func (h *Handler) reconcileHeld(w http.ResponseWriter, r *http.Request, tenantID, orgID string) {
	var req reconcileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.TransactionID) == "" {
		httpresponse.Error(w, http.StatusBadRequest, "transaction_id is required")
		return
	}
	result, err := h.store.ReconcileHeld(r.Context(), tenantID, orgID, req.TransactionID)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h *Handler) listLegalHolds(w http.ResponseWriter, r *http.Request, tenantID, orgID string) {
	holds, err := h.store.ListLegalHolds(r.Context(), tenantID, orgID)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, struct {
		LegalHolds []LegalHold `json:"legal_holds"`
	}{LegalHolds: holds})
}

type placeLegalHoldRequest struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	Reason     string `json:"reason"`
}

func (h *Handler) placeLegalHold(w http.ResponseWriter, r *http.Request, tenantID, orgID, actorID string) {
	var req placeLegalHoldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	hold, err := h.store.PlaceLegalHold(r.Context(), actorID, tenantID, orgID, req.EntityType, req.EntityID, req.Reason)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, hold)
}

type releaseLegalHoldRequest struct {
	ID string `json:"id"`
}

func (h *Handler) releaseLegalHold(w http.ResponseWriter, r *http.Request, tenantID, orgID, actorID string) {
	var req releaseLegalHoldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.ID) == "" {
		httpresponse.Error(w, http.StatusBadRequest, "id is required")
		return
	}
	hold, err := h.store.ReleaseLegalHold(r.Context(), actorID, tenantID, orgID, req.ID)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, hold)
}

func (h *Handler) retentionOverview(w http.ResponseWriter, r *http.Request, tenantID, orgID string) {
	policies, err := h.store.ListRetentionPolicies(r.Context(), tenantID, orgID)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	approvals, err := h.store.ListExportApprovals(r.Context(), tenantID, orgID)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, struct {
		RetentionPolicies []RetentionPolicy `json:"retention_policies"`
		ExportApprovals   []ExportApproval  `json:"export_approvals"`
	}{RetentionPolicies: policies, ExportApprovals: approvals})
}

type decideExportApprovalRequest struct {
	ExportID        string `json:"export_id"`
	Status          string `json:"status"`
	RejectionReason string `json:"rejection_reason"`
}

func (h *Handler) decideExportApproval(w http.ResponseWriter, r *http.Request, tenantID, orgID, actorID string) {
	var req decideExportApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	status := strings.ToUpper(strings.TrimSpace(req.Status))
	if strings.TrimSpace(req.ExportID) == "" {
		httpresponse.Error(w, http.StatusBadRequest, "export_id is required")
		return
	}
	if status != statusApproved && status != statusRejected {
		httpresponse.Error(w, http.StatusBadRequest, "status must be APPROVED or REJECTED")
		return
	}
	if status == statusRejected && strings.TrimSpace(req.RejectionReason) == "" {
		httpresponse.Error(w, http.StatusBadRequest, "rejection_reason is required when rejecting")
		return
	}
	approval, err := h.store.DecideExportApproval(r.Context(), actorID, tenantID, orgID, req.ExportID, status, strings.TrimSpace(req.RejectionReason))
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, approval)
}

func (h *Handler) writeStoreError(w http.ResponseWriter, err error) {
	h.log.Warn("workbench store error", "error", err.Error())
	switch {
	case errors.Is(err, ErrNotFound):
		httpresponse.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrConflict):
		httpresponse.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrValidation):
		httpresponse.Error(w, http.StatusBadRequest, err.Error())
	default:
		httpresponse.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

func newRandomID(prefix string) (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(randomBytes), nil
}
