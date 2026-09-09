package assurance

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidActor      = errors.New("assurance actor context is invalid")
	ErrInvalidProjection = errors.New("assurance projection is invalid")
	ErrInvalidWork       = errors.New("corrective work is invalid")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrStaleRevision     = errors.New("stale revision")
	ErrNotFound          = errors.New("corrective work not found")
)

// ActorContext is the server-derived actor for assurance operations.
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

// Severity represents the severity of a finding.
type Severity string

const (
	SeverityAdvisory Severity = "ADVISORY"
	SeverityMinor    Severity = "MINOR"
	SeverityMajor    Severity = "MAJOR"
	SeverityCritical Severity = "CRITICAL"
)

// WorkStatus represents the status of corrective work.
type WorkStatus string

const (
	WorkStatusOpen       WorkStatus = "OPEN"
	WorkStatusInProgress WorkStatus = "IN_PROGRESS"
	WorkStatusCompleted  WorkStatus = "COMPLETED"
	WorkStatusVerified   WorkStatus = "VERIFIED"
	WorkStatusClosed     WorkStatus = "CLOSED"
)

// Projection represents a read-only view of certificate authority status.
// Derived from certificate_record and inspection_record; never mutates authority.
type Projection struct {
	ID                string
	TenantID          string
	OrganizationID    string
	CertificateID     string
	InspectionID      string
	AssetID           string
	AssetType         string
	CertificateNumber string
	CertificateStatus string
	InspectionStatus  string
	InspectionVerdict string
	InspectorID       string
	IssuedAt          *time.Time
	ExpiresAt         *time.Time
	RevokedAt         *time.Time
	SupersededAt      *time.Time
	ProjectionHash    string
	ProjectedAt       time.Time
	CreatedAt         time.Time
}

// Validate ensures the projection is consistent.
func (p Projection) Validate() error {
	if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.CertificateID) == "" || strings.TrimSpace(p.InspectionID) == "" {
		return fmt.Errorf("%w: id, certificate_id, and inspection_id are required", ErrInvalidProjection)
	}
	if strings.TrimSpace(p.AssetID) == "" || strings.TrimSpace(p.AssetType) == "" {
		return fmt.Errorf("%w: asset_id and asset_type are required", ErrInvalidProjection)
	}
	if strings.TrimSpace(p.CertificateNumber) == "" {
		return fmt.Errorf("%w: certificate_number is required", ErrInvalidProjection)
	}
	if strings.TrimSpace(p.CertificateStatus) == "" {
		return fmt.Errorf("%w: certificate_status is required", ErrInvalidProjection)
	}
	if strings.TrimSpace(p.InspectionStatus) == "" {
		return fmt.Errorf("%w: inspection_status is required", ErrInvalidProjection)
	}
	if strings.TrimSpace(p.InspectionVerdict) == "" {
		return fmt.Errorf("%w: inspection_verdict is required", ErrInvalidProjection)
	}
	if strings.TrimSpace(p.InspectorID) == "" {
		return fmt.Errorf("%w: inspector_id is required", ErrInvalidProjection)
	}
	if strings.TrimSpace(p.ProjectionHash) == "" {
		return fmt.Errorf("%w: projection_hash is required", ErrInvalidProjection)
	}
	if p.ProjectedAt.IsZero() || p.CreatedAt.IsZero() {
		return fmt.Errorf("%w: projected_at and created_at are required", ErrInvalidProjection)
	}
	return nil
}

// ComputeProjectionHash computes SHA-256 hash of projection data for integrity.
func ComputeProjectionHash(p Projection) string {
	data := fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s:%s:%s:%s",
		p.CertificateID, p.InspectionID, p.AssetID, p.AssetType,
		p.CertificateNumber, p.CertificateStatus, p.InspectionStatus,
		p.InspectionVerdict, p.InspectorID,
		p.CertificateStatus)
	h := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", h)
}

// CorrectiveWork represents an actionable work item linked to an inspection finding.
type CorrectiveWork struct {
	ID             string
	TenantID       string
	OrganizationID string
	InspectionID   string
	FindingID      string
	AssetID        string
	Severity       Severity
	Description    string
	RequiredBy     *time.Time
	AssignedTo     string
	Status         WorkStatus
	CompletedAt    *time.Time
	VerifiedAt     *time.Time
	VerifiedBy     string
	CreatedBy      string
	CreatedAt      time.Time
	UpdatedBy      string
	UpdatedAt      time.Time
	Revision       int64
}

// Validate ensures the corrective work is consistent.
func (w CorrectiveWork) Validate() error {
	if strings.TrimSpace(w.ID) == "" || strings.TrimSpace(w.InspectionID) == "" || strings.TrimSpace(w.FindingID) == "" {
		return fmt.Errorf("%w: id, inspection_id, and finding_id are required", ErrInvalidWork)
	}
	if strings.TrimSpace(w.AssetID) == "" {
		return fmt.Errorf("%w: asset_id is required", ErrInvalidWork)
	}
	switch w.Severity {
	case SeverityAdvisory, SeverityMinor, SeverityMajor, SeverityCritical:
		// valid
	default:
		return fmt.Errorf("%w: unsupported severity %q", ErrInvalidWork, w.Severity)
	}
	if strings.TrimSpace(w.Description) == "" {
		return fmt.Errorf("%w: description is required", ErrInvalidWork)
	}
	switch w.Status {
	case WorkStatusOpen:
		if w.CompletedAt != nil || w.VerifiedAt != nil || w.VerifiedBy != "" {
			return fmt.Errorf("%w: OPEN status cannot have completed/verified metadata", ErrInvalidWork)
		}
	case WorkStatusInProgress:
		if w.CompletedAt != nil || w.VerifiedAt != nil || w.VerifiedBy != "" {
			return fmt.Errorf("%w: IN_PROGRESS status cannot have completed/verified metadata", ErrInvalidWork)
		}
	case WorkStatusCompleted:
		if w.CompletedAt == nil {
			return fmt.Errorf("%w: COMPLETED status requires completed_at", ErrInvalidWork)
		}
		if w.VerifiedAt != nil || w.VerifiedBy != "" {
			return fmt.Errorf("%w: COMPLETED status cannot have verified metadata", ErrInvalidWork)
		}
	case WorkStatusVerified:
		if w.VerifiedAt == nil || strings.TrimSpace(w.VerifiedBy) == "" {
			return fmt.Errorf("%w: VERIFIED status requires verified_at and verified_by", ErrInvalidWork)
		}
	case WorkStatusClosed:
		// valid
	default:
		return fmt.Errorf("%w: unsupported status %q", ErrInvalidWork, w.Status)
	}
	if w.CompletedAt != nil && w.CompletedAt.Before(w.CreatedAt) {
		return fmt.Errorf("%w: completed_at must be after created_at", ErrInvalidWork)
	}
	if w.VerifiedAt != nil && w.VerifiedAt.Before(w.CreatedAt) {
		return fmt.Errorf("%w: verified_at must be after created_at", ErrInvalidWork)
	}
	if w.CreatedBy == "" || w.CreatedAt.IsZero() || w.UpdatedBy == "" || w.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: created_by, created_at, updated_by, and updated_at are required", ErrInvalidWork)
	}
	if w.Revision < 1 {
		return fmt.Errorf("%w: revision must be positive", ErrInvalidWork)
	}
	return nil
}

// CanTransitionTo checks if a status transition is valid.
func (w CorrectiveWork) CanTransitionTo(target WorkStatus) error {
	switch w.Status {
	case WorkStatusOpen:
		if target != WorkStatusInProgress && target != WorkStatusClosed {
			return fmt.Errorf("%w: OPEN can only transition to IN_PROGRESS or CLOSED", ErrInvalidTransition)
		}
	case WorkStatusInProgress:
		if target != WorkStatusCompleted && target != WorkStatusClosed {
			return fmt.Errorf("%w: IN_PROGRESS can only transition to COMPLETED or CLOSED", ErrInvalidTransition)
		}
	case WorkStatusCompleted:
		if target != WorkStatusVerified && target != WorkStatusInProgress {
			return fmt.Errorf("%w: COMPLETED can only transition to VERIFIED or IN_PROGRESS", ErrInvalidTransition)
		}
	case WorkStatusVerified:
		if target != WorkStatusClosed {
			return fmt.Errorf("%w: VERIFIED can only transition to CLOSED", ErrInvalidTransition)
		}
	case WorkStatusClosed:
		return fmt.Errorf("%w: CLOSED work is terminal", ErrInvalidTransition)
	default:
		return fmt.Errorf("%w: unknown current status", ErrInvalidTransition)
	}
	return nil
}

// ProjectionRepository defines read-only operations for assurance projections.
type ProjectionRepository interface {
	// Get retrieves a projection by certificate ID.
	Get(ctx context.Context, actor ActorContext, certificateID string) (Projection, error)
	// GetByInspection retrieves a projection by inspection ID.
	GetByInspection(ctx context.Context, actor ActorContext, inspectionID string) (Projection, error)
	// ListByAsset retrieves all projections for an asset.
	ListByAsset(ctx context.Context, actor ActorContext, assetID string) ([]Projection, error)
	// ListByStatus retrieves projections by certificate status.
	ListByStatus(ctx context.Context, actor ActorContext, status string) ([]Projection, error)
}

// WorkRepository defines persistence operations for corrective work.
type WorkRepository interface {
	// Create creates a new OPEN corrective work item.
	Create(ctx context.Context, actor ActorContext, work CorrectiveWork) (CorrectiveWork, error)
	// Get retrieves corrective work by ID.
	Get(ctx context.Context, actor ActorContext, workID string) (CorrectiveWork, error)
	// ListByInspection retrieves all corrective work for an inspection.
	ListByInspection(ctx context.Context, actor ActorContext, inspectionID string) ([]CorrectiveWork, error)
	// ListByAsset retrieves all corrective work for an asset.
	ListByAsset(ctx context.Context, actor ActorContext, assetID string) ([]CorrectiveWork, error)
	// UpdateStatus transitions corrective work to a new status.
	UpdateStatus(ctx context.Context, actor ActorContext, workID string, targetStatus WorkStatus, expectedRevision int64) (CorrectiveWork, error)
	// Assign sets or updates the assigned inspector.
	Assign(ctx context.Context, actor ActorContext, workID string, inspectorID string, expectedRevision int64) (CorrectiveWork, error)
}
