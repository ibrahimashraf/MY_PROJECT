package evidencepack

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"integin/internal/domain/formdefinition"
)

var (
	ErrInvalidActor        = errors.New("evidence pack actor context is invalid")
	ErrInvalidPack         = errors.New("evidence pack is invalid")
	ErrInvalidRelease      = errors.New("release pack is invalid")
	ErrInvalidTransition   = errors.New("invalid evidence pack status transition")
	ErrInvalidReleaseState = errors.New("invalid release status transition")
	ErrStaleRevision       = errors.New("stale revision")
	ErrNotFound            = errors.New("evidence pack or release not found")
	ErrSealConflict        = errors.New("cannot modify a sealed pack")
	ErrMixedPrivacy        = errors.New("evidence set has mixed privacy attributes")
	ErrPolicyViolation     = errors.New("evidence violates an evidence policy")
)

// PackStatus represents the lifecycle of an evidence pack.
type PackStatus string

const (
	PackStatusDraft      PackStatus = "DRAFT"
	PackStatusSealed     PackStatus = "SEALED"
	PackStatusReleased   PackStatus = "RELEASED"
	PackStatusSuperseded PackStatus = "SUPERSEDED"
)

// ReleaseStatus represents the lifecycle of a release ledger entry.
type ReleaseStatus string

const (
	ReleaseStatusPending    ReleaseStatus = "PENDING"
	ReleaseStatusApproved   ReleaseStatus = "APPROVED"
	ReleaseStatusReleased   ReleaseStatus = "RELEASED"
	ReleaseStatusSuperseded ReleaseStatus = "SUPERSEDED"
)

// ActorContext is the server-derived actor for evidence pack operations.
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

// PackRecord represents a single evidence object referenced by a pack.
type PackRecord struct {
	EvidenceID       string    `json:"evidence_id"`
	ObjectKey        string    `json:"object_key"`
	ContentType      string    `json:"content_type"`
	CapturedAt       time.Time `json:"captured_at"`
	InspectionID     string    `json:"inspection_id"`
	PlaintextSHA256  string    `json:"plaintext_sha256"`
	CiphertextSHA256 string    `json:"ciphertext_sha256"`
	CiphertextBytes  int64     `json:"ciphertext_bytes"`
	Classification   string    `json:"classification"`
}

// EvidencePack is the aggregate: a composed, sealed collection of evidence.
type EvidencePack struct {
	ID                 string
	TenantID           string
	OrganizationID     string
	InspectionID       string
	PackName           string
	Status             PackStatus
	Records            []PackRecord
	Classification     string
	RetentionReference string
	HoldState          string
	RedactionPolicyRef string
	PackChecksum       string
	CreatedBy          string
	CreatedAt          time.Time
	UpdatedBy          string
	UpdatedAt          time.Time
	Revision           int64
}

// Validate ensures the evidence pack is consistent.
func (p EvidencePack) Validate() error {
	if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.InspectionID) == "" || strings.TrimSpace(p.PackName) == "" {
		return fmt.Errorf("%w: id, inspection_id, and pack_name are required", ErrInvalidPack)
	}
	switch p.Status {
	case PackStatusDraft, PackStatusSealed, PackStatusReleased, PackStatusSuperseded:
		// valid
	default:
		return fmt.Errorf("%w: unsupported pack status %q", ErrInvalidPack, p.Status)
	}
	switch p.Classification {
	case string(formdefinition.ClassificationPublic), string(formdefinition.ClassificationInternal),
		string(formdefinition.ClassificationConfidential), string(formdefinition.ClassificationRestricted):
		// valid
	default:
		return fmt.Errorf("%w: unsupported classification %q", ErrInvalidPack, p.Classification)
	}
	if p.CreatedBy == "" || p.CreatedAt.IsZero() || p.UpdatedBy == "" || p.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: created_by, created_at, updated_by, and updated_at are required", ErrInvalidPack)
	}
	if p.Revision < 1 {
		return fmt.Errorf("%w: revision must be positive", ErrInvalidPack)
	}
	if p.Status == PackStatusSealed || p.Status == PackStatusReleased {
		if len(p.Records) == 0 {
			return fmt.Errorf("%w: sealed or released pack must have at least one evidence record", ErrInvalidPack)
		}
		if p.PackChecksum == "" {
			return fmt.Errorf("%w: sealed or released pack requires a checksum", ErrInvalidPack)
		}
	}
	return nil
}

// ComputeChecksum computes a deterministic SHA-256 over the sorted records.
func (p EvidencePack) ComputeChecksum() string {
	records := make([]PackRecord, len(p.Records))
	copy(records, p.Records)
	sort.Slice(records, func(i, j int) bool {
		if records[i].EvidenceID != records[j].EvidenceID {
			return records[i].EvidenceID < records[j].EvidenceID
		}
		return records[i].ObjectKey < records[j].ObjectKey
	})
	payload := struct {
		PackName     string       `json:"pack_name"`
		InspectionID string       `json:"inspection_id"`
		Records      []PackRecord `json:"records"`
	}{
		PackName:     p.PackName,
		InspectionID: p.InspectionID,
		Records:      records,
	}
	b, _ := json.Marshal(payload)
	sum := sha256.Sum256(b)
	return fmt.Sprintf("%x", sum)
}

// VerifyChecksum returns true if the stored checksum matches the recomputed one.
func (p EvidencePack) VerifyChecksum() bool {
	if p.PackChecksum == "" {
		return false
	}
	return p.ComputeChecksum() == p.PackChecksum
}

// CanTransitionTo checks valid pack status transitions.
func (p EvidencePack) CanTransitionTo(target PackStatus) error {
	switch p.Status {
	case PackStatusDraft:
		if target != PackStatusSealed && target != PackStatusSuperseded {
			return fmt.Errorf("%w: DRAFT can only transition to SEALED or SUPERSEDED", ErrInvalidTransition)
		}
	case PackStatusSealed:
		if target != PackStatusReleased && target != PackStatusSuperseded {
			return fmt.Errorf("%w: SEALED can only transition to RELEASED or SUPERSEDED", ErrInvalidTransition)
		}
	case PackStatusReleased:
		if target != PackStatusSuperseded {
			return fmt.Errorf("%w: RELEASED can only transition to SUPERSEDED", ErrInvalidTransition)
		}
	default:
		return fmt.Errorf("%w: SUPERSEDED is terminal", ErrInvalidTransition)
	}
	return nil
}

// AddRecord adds a record while the pack is still DRAFT (immutable after seal).
func (p *EvidencePack) AddRecord(rec PackRecord) error {
	if p.Status != PackStatusDraft {
		return ErrSealConflict
	}
	if rec.EvidenceID == "" || rec.ObjectKey == "" || rec.InspectionID == "" {
		return fmt.Errorf("%w: incomplete evidence record", ErrInvalidPack)
	}
	if rec.InspectionID != p.InspectionID {
		return fmt.Errorf("%w: inspection_id mismatch on evidence record", ErrInvalidPack)
	}
	for _, existing := range p.Records {
		if existing.EvidenceID == rec.EvidenceID {
			return fmt.Errorf("%w: duplicate evidence %q", ErrInvalidPack, rec.EvidenceID)
		}
	}
	p.Records = append(p.Records, rec)
	return nil
}

// Seal finalizes the pack: enforces single uniform privacy set and seals the checksum.
func (p *EvidencePack) Seal(now time.Time) error {
	if p.Status != PackStatusDraft {
		return fmt.Errorf("%w: only a DRAFT pack can be sealed", ErrInvalidTransition)
	}
	if len(p.Records) == 0 {
		return fmt.Errorf("%w: cannot seal an empty pack", ErrInvalidPack)
	}
	if err := p.uniformPrivacy(); err != nil {
		return err
	}
	p.PackChecksum = p.ComputeChecksum()
	p.Status = PackStatusSealed
	p.UpdatedAt = now
	return nil
}

// uniformPrivacy ensures all records share a single privacy set.
func (p *EvidencePack) uniformPrivacy() error {
	for _, rec := range p.Records {
		if rec.Classification != p.Classification {
			return ErrMixedPrivacy
		}
	}
	return nil
}

// ReleasePack records a recipient-purpose release of a sealed pack, under uniform privacy.
type ReleasePack struct {
	ID                 string
	TenantID           string
	OrganizationID     string
	PackID             string
	Recipient          string
	Purpose            string
	Status             ReleaseStatus
	Classification     string
	RetentionReference string
	HoldState          string
	RedactionPolicyRef string
	ApprovedBy         string
	ApprovedAt         *time.Time
	ReleasedBy         string
	ReleasedAt         *time.Time
	CreatedBy          string
	CreatedAt          time.Time
	UpdatedBy          string
	UpdatedAt          time.Time
	Revision           int64
}

// Validate ensures the release pack is consistent.
func (r ReleasePack) Validate() error {
	if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.PackID) == "" || strings.TrimSpace(r.Recipient) == "" || strings.TrimSpace(r.Purpose) == "" {
		return fmt.Errorf("%w: id, pack_id, recipient, and purpose are required", ErrInvalidRelease)
	}
	switch r.Status {
	case ReleaseStatusPending, ReleaseStatusApproved, ReleaseStatusReleased, ReleaseStatusSuperseded:
		// valid
	default:
		return fmt.Errorf("%w: unsupported release status %q", ErrInvalidRelease, r.Status)
	}
	switch r.Classification {
	case string(formdefinition.ClassificationPublic), string(formdefinition.ClassificationInternal),
		string(formdefinition.ClassificationConfidential), string(formdefinition.ClassificationRestricted):
		// valid
	default:
		return fmt.Errorf("%w: unsupported classification %q", ErrInvalidRelease, r.Classification)
	}
	if r.CreatedBy == "" || r.CreatedAt.IsZero() || r.UpdatedBy == "" || r.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: created_by, created_at, updated_by, updated_at required", ErrInvalidRelease)
	}
	if r.Revision < 1 {
		return fmt.Errorf("%w: revision must be positive", ErrInvalidRelease)
	}
	if r.Status == ReleaseStatusApproved && (r.ApprovedBy == "" || r.ApprovedAt == nil) {
		return fmt.Errorf("%w: APPROVED requires approved_by and approved_at", ErrInvalidRelease)
	}
	if r.Status == ReleaseStatusReleased && (r.ReleasedBy == "" || r.ReleasedAt == nil) {
		return fmt.Errorf("%w: RELEASED requires released_by and released_at", ErrInvalidRelease)
	}
	return nil
}

// CanTransitionTo checks valid release status transitions.
func (r ReleasePack) CanTransitionTo(target ReleaseStatus) error {
	switch r.Status {
	case ReleaseStatusPending:
		if target != ReleaseStatusApproved && target != ReleaseStatusSuperseded {
			return fmt.Errorf("%w: PENDING can only transition to APPROVED or SUPERSEDED", ErrInvalidReleaseState)
		}
	case ReleaseStatusApproved:
		if target != ReleaseStatusReleased && target != ReleaseStatusSuperseded {
			return fmt.Errorf("%w: APPROVED can only transition to RELEASED or SUPERSEDED", ErrInvalidReleaseState)
		}
	case ReleaseStatusReleased:
		if target != ReleaseStatusSuperseded {
			return fmt.Errorf("%w: RELEASED can only transition to SUPERSEDED", ErrInvalidReleaseState)
		}
	default:
		return fmt.Errorf("%w: SUPERSEDED is terminal", ErrInvalidReleaseState)
	}
	return nil
}

// PackRepository defines persistence for evidence packs.
type PackRepository interface {
	Create(ctx context.Context, actor ActorContext, pack EvidencePack) (EvidencePack, error)
	Get(ctx context.Context, actor ActorContext, packID string) (EvidencePack, error)
	GetByName(ctx context.Context, actor ActorContext, packName string) (EvidencePack, error)
	ListByInspection(ctx context.Context, actor ActorContext, inspectionID string) ([]EvidencePack, error)
	UpdateStatus(ctx context.Context, actor ActorContext, packID string, target PackStatus, expectedRevision int64) (EvidencePack, error)
}

// ReleaseRepository defines persistence for release ledger entries.
type ReleaseRepository interface {
	Create(ctx context.Context, actor ActorContext, release ReleasePack) (ReleasePack, error)
	Get(ctx context.Context, actor ActorContext, releaseID string) (ReleasePack, error)
	ListByPack(ctx context.Context, actor ActorContext, packID string) ([]ReleasePack, error)
	UpdateStatus(ctx context.Context, actor ActorContext, releaseID string, target ReleaseStatus, expectedRevision int64) (ReleasePack, error)
}
