// Package retention implements the P1.3 retention, legal hold, and export
// access policy engine. Deterministic disposition decisions are evaluated
// server-side; tombstone hashes provide tamper-evident deletion evidence.
package retention

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

// RetentionAction is the disposition taken once the retention window elapses.
type RetentionAction string

const (
	RetentionActionArchive RetentionAction = "ARCHIVE"
	RetentionActionPurge   RetentionAction = "PURGE"
)

// RetentionPolicy is a tenant-scoped per-entity-type retention profile.
type RetentionPolicy struct {
	TenantID      string
	EntityType    string
	RetentionDays int
	Action        RetentionAction
	UpdatedAt     time.Time
	UpdatedBy     string
}

// LegalHoldStatus tracks the lifecycle of a legal hold.
type LegalHoldStatus string

const (
	LegalHoldActive   LegalHoldStatus = "ACTIVE"
	LegalHoldReleased LegalHoldStatus = "RELEASED"
)

// LegalHold pins one entity so no disposition may impair preservation of it.
type LegalHold struct {
	ID         string
	TenantID   string
	EntityType string
	EntityID   string
	Reason     string
	PlacedBy   string
	PlacedAt   time.Time
	Status     LegalHoldStatus
	ReleasedBy string
	ReleasedAt *time.Time
}

// ExportApprovalStatus tracks the approval lifecycle of an export request.
type ExportApprovalStatus string

const (
	ExportApprovalPending  ExportApprovalStatus = "PENDING"
	ExportApprovalApproved ExportApprovalStatus = "APPROVED"
	ExportApprovalRejected ExportApprovalStatus = "REJECTED"
)

// ExportApproval is the recorded authority decision for one export request.
type ExportApproval struct {
	ID              string
	TenantID        string
	ExportID        string
	RequestedBy     string
	ApprovedBy      string
	Status          ExportApprovalStatus
	CreatedAt       time.Time
	DecidedAt       *time.Time
	RejectionReason string
}

// DeletionCertificate is the tamper-evident receipt emitted when an entity is
// purged. TombstoneHash deterministically binds the deletion inputs.
type DeletionCertificate struct {
	TenantID      string
	EntityType    string
	EntityID      string
	ActorID       string
	DeletedAt     time.Time
	TombstoneHash string
}

var (
	ErrLegalHoldActive        = errors.New("retention: active legal hold blocks disposition")
	ErrRetentionPeriodActive  = errors.New("retention: retention window has not elapsed")
	ErrExportNotApproved      = errors.New("retention: export is not approved")
	ErrUnauthorizedApprover   = errors.New("retention: caller is not the authorized requester")
	ErrSelfApprovalProhibited = errors.New("retention: self-approval is prohibited")
)

// CanPurge evaluates whether an entity may be purged. An active legal hold
// always blocks disposition; otherwise the entity must have outlived its
// retention window. A nil policy imposes no retention window.
func CanPurge(ctx context.Context, tenantID, entityType, entityID string, createdAt time.Time, holds []LegalHold, policy *RetentionPolicy) (bool, error) {
	_ = ctx
	for _, h := range holds {
		if h.Status == LegalHoldActive && h.TenantID == tenantID && h.EntityType == entityType && h.EntityID == entityID {
			return false, ErrLegalHoldActive
		}
	}
	if policy != nil {
		window := time.Duration(policy.RetentionDays) * 24 * time.Hour
		if time.Since(createdAt) < window {
			return false, ErrRetentionPeriodActive
		}
	}
	return true, nil
}

// AuthorizeExport enforces the export access policy: self-approval is
// prohibited and only an APPROVED approval bound to the requesting actor
// authorizes the export.
func AuthorizeExport(ctx context.Context, approval *ExportApproval, requestedBy, approverID string) error {
	_ = ctx
	if approval == nil || approval.Status != ExportApprovalApproved {
		return ErrExportNotApproved
	}
	if approverID != "" && approverID == requestedBy {
		return ErrSelfApprovalProhibited
	}
	if approval.RequestedBy != requestedBy {
		return ErrUnauthorizedApprover
	}
	return nil
}

// GenerateDeletionCertificate produces a deterministic SHA-256 tombstone hash
// over the deletion inputs plus the secret seed, binding the receipt to the
// exact record purged. Identical inputs yield an identical hash; any tamper
// with tenant, entity, actor, timestamp, or seed diverges it.
func GenerateDeletionCertificate(tenantID, entityType, entityID, actorID string, deletedAt time.Time, secretSeed []byte) DeletionCertificate {
	h := sha256.New()
	for _, part := range [][]byte{
		[]byte(tenantID), []byte{0},
		[]byte(entityType), []byte{0},
		[]byte(entityID), []byte{0},
		[]byte(deletedAt.UTC().Format(time.RFC3339Nano)), []byte{0},
		[]byte(actorID), []byte{0},
		secretSeed,
	} {
		h.Write(part)
	}
	return DeletionCertificate{
		TenantID:      tenantID,
		EntityType:    entityType,
		EntityID:      entityID,
		ActorID:       actorID,
		DeletedAt:     deletedAt.UTC(),
		TombstoneHash: hex.EncodeToString(h.Sum(nil)),
	}
}
