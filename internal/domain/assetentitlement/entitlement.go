package assetentitlement

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidActor      = errors.New("asset entitlement actor context is invalid")
	ErrInvalidEntitlement = errors.New("asset entitlement is invalid")
	ErrInvalidTag        = errors.New("asset tag is invalid")
	ErrInvalidPackage    = errors.New("offline package is invalid")
	ErrImmutable         = errors.New("asset entitlement conflicts with an immutable record")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrStaleRevision     = errors.New("stale revision")
)

// ActorContext is the server-derived actor for asset operations.
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

// EntitlementType represents the type of asset entitlement.
type EntitlementType string

const (
	EntitlementTypeInspection EntitlementType = "INSPECTION"
	EntitlementTypeMaintenance EntitlementType = "MAINTENANCE"
	EntitlementTypeAudit       EntitlementType = "AUDIT"
	EntitlementTypeReview      EntitlementType = "REVIEW"
)

// EntitlementStatus represents the status of an asset entitlement.
type EntitlementStatus string

const (
	EntitlementStatusActive    EntitlementStatus = "ACTIVE"
	EntitlementStatusCompleted EntitlementStatus = "COMPLETED"
	EntitlementStatusCancelled EntitlementStatus = "CANCELLED"
	EntitlementStatusExpired   EntitlementStatus = "EXPIRED"
)

// TagType represents the type of physical tag.
type TagType string

const (
	TagTypeQR          TagType = "QR"
	TagTypeBarcode     TagType = "BARCODE"
	TagTypeNFC         TagType = "NFC"
	TagTypeRFID        TagType = "RFID"
	TagTypeSerialPlate TagType = "SERIAL_PLATE"
)

// PackageStatus represents the status of an offline package.
type PackageStatus string

const (
	PackageStatusPending  PackageStatus = "PENDING"
	PackageStatusGenerated PackageStatus = "GENERATED"
	PackageStatusDelivered PackageStatus = "DELIVERED"
	PackageStatusExpired   PackageStatus = "EXPIRED"
	PackageStatusRevoked   PackageStatus = "REVOKED"
)

// AssetTag represents an untrusted physical pointer for an asset.
type AssetTag struct {
	TenantID       string
	OrganizationID string
	AssetID        string
	TagType        TagType
	TagValue       string
	TagDigest      string
	AssignedAt     time.Time
	AssignedBy     string
}

// Validate ensures the asset tag is consistent.
func (t AssetTag) Validate() error {
	if strings.TrimSpace(t.AssetID) == "" || strings.TrimSpace(t.TagValue) == "" {
		return fmt.Errorf("%w: asset_id and tag_value are required", ErrInvalidTag)
	}
	switch t.TagType {
	case TagTypeQR, TagTypeBarcode, TagTypeNFC, TagTypeRFID, TagTypeSerialPlate:
		// valid
	default:
		return fmt.Errorf("%w: unsupported tag_type %q", ErrInvalidTag, t.TagType)
	}
	if strings.TrimSpace(t.TagDigest) == "" {
		return fmt.Errorf("%w: tag_digest is required", ErrInvalidTag)
	}
	if t.AssignedAt.IsZero() {
		return fmt.Errorf("%w: assigned_at is required", ErrInvalidTag)
	}
	if strings.TrimSpace(t.AssignedBy) == "" {
		return fmt.Errorf("%w: assigned_by is required", ErrInvalidTag)
	}
	return nil
}

// ComputeTagDigest computes SHA-256 digest of tag type and value.
func ComputeTagDigest(tagType TagType, tagValue string) string {
	h := sha256.Sum256([]byte(string(tagType) + ":" + tagValue))
	return fmt.Sprintf("%x", h)
}

// OfflinePackage represents a server-reconciled, bounded, never client-authoritative offline data package.
type OfflinePackage struct {
	ID                    string
	TenantID              string
	OrganizationID        string
	EntitlementID         string
	PackageVersion        int
	FormSnapshot          json.RawMessage
	EvidencePolicySnapshot json.RawMessage
	AssetContext          json.RawMessage
	PackageHash           string
	Status                PackageStatus
	GeneratedAt           *time.Time
	DeliveredAt           *time.Time
	DeliveredToDeviceID   string
	DeliveredToInspectorID string
	ExpiresAt             *time.Time
	RevokedAt             *time.Time
	RevokedBy             string
	CreatedBy             string
	CreatedAt             time.Time
}

// Validate ensures the offline package is consistent.
func (p OfflinePackage) Validate() error {
	if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.EntitlementID) == "" {
		return fmt.Errorf("%w: id and entitlement_id are required", ErrInvalidPackage)
	}
	if p.PackageVersion <= 0 {
		return fmt.Errorf("%w: package_version must be positive", ErrInvalidPackage)
	}
	if len(p.FormSnapshot) == 0 || string(p.FormSnapshot) == "null" {
		return fmt.Errorf("%w: form_snapshot is required", ErrInvalidPackage)
	}
	if len(p.EvidencePolicySnapshot) == 0 || string(p.EvidencePolicySnapshot) == "null" {
		return fmt.Errorf("%w: evidence_policy_snapshot is required", ErrInvalidPackage)
	}
	if len(p.AssetContext) == 0 || string(p.AssetContext) == "null" {
		return fmt.Errorf("%w: asset_context is required", ErrInvalidPackage)
	}
	if strings.TrimSpace(p.PackageHash) == "" {
		return fmt.Errorf("%w: package_hash is required", ErrInvalidPackage)
	}
	switch p.Status {
	case PackageStatusPending:
		// valid
	case PackageStatusGenerated:
		if p.GeneratedAt == nil {
			return fmt.Errorf("%w: GENERATED status requires generated_at", ErrInvalidPackage)
		}
	case PackageStatusDelivered:
		if p.DeliveredAt == nil || strings.TrimSpace(p.DeliveredToDeviceID) == "" || strings.TrimSpace(p.DeliveredToInspectorID) == "" {
			return fmt.Errorf("%w: DELIVERED status requires delivered_at, device_id, and inspector_id", ErrInvalidPackage)
		}
	case PackageStatusExpired:
		// valid
	case PackageStatusRevoked:
		if p.RevokedAt == nil || strings.TrimSpace(p.RevokedBy) == "" {
			return fmt.Errorf("%w: REVOKED status requires revoked_at and revoked_by", ErrInvalidPackage)
		}
	default:
		return fmt.Errorf("%w: unsupported status %q", ErrInvalidPackage, p.Status)
	}
	if p.CreatedBy == "" || p.CreatedAt.IsZero() {
		return fmt.Errorf("%w: created_by and created_at are required", ErrInvalidPackage)
	}
	return nil
}

// CanTransitionTo checks if a status transition is valid.
func (p OfflinePackage) CanTransitionTo(target PackageStatus) error {
	switch p.Status {
	case PackageStatusPending:
		if target != PackageStatusGenerated && target != PackageStatusRevoked {
			return fmt.Errorf("%w: PENDING can only transition to GENERATED or REVOKED", ErrInvalidTransition)
		}
	case PackageStatusGenerated:
		if target != PackageStatusDelivered && target != PackageStatusExpired && target != PackageStatusRevoked {
			return fmt.Errorf("%w: GENERATED can only transition to DELIVERED, EXPIRED, or REVOKED", ErrInvalidTransition)
		}
	case PackageStatusDelivered:
		if target != PackageStatusExpired && target != PackageStatusRevoked {
			return fmt.Errorf("%w: DELIVERED can only transition to EXPIRED or REVOKED", ErrInvalidTransition)
		}
	case PackageStatusExpired:
		return fmt.Errorf("%w: EXPIRED packages are terminal", ErrInvalidTransition)
	case PackageStatusRevoked:
		return fmt.Errorf("%w: REVOKED packages are terminal", ErrInvalidTransition)
	default:
		return fmt.Errorf("%w: unknown current status", ErrInvalidTransition)
	}
	return nil
}

// AssetEntitlement represents a server-authoritative entitlement for an asset within a work order scope.
type AssetEntitlement struct {
	ID                 string
	TenantID           string
	OrganizationID     string
	WorkOrderID        string
	ScopeItemID        string
	AssetID            string
	AssetType          string
	EntitlementType    EntitlementType
	Status             EntitlementStatus
	FormVersionID      string
	AssignedInspectorID string
	EntitledAt         time.Time
	ExpiresAt          *time.Time
	CompletedAt        *time.Time
	CreatedBy          string
	CreatedAt          time.Time
	UpdatedBy          string
	UpdatedAt          time.Time
	Revision           int64
	Tags               []AssetTag
	Packages           []OfflinePackage
}

// Validate ensures the asset entitlement is consistent.
func (e AssetEntitlement) Validate() error {
	if strings.TrimSpace(e.ID) == "" || strings.TrimSpace(e.WorkOrderID) == "" || strings.TrimSpace(e.ScopeItemID) == "" {
		return fmt.Errorf("%w: id, work_order_id, and scope_item_id are required", ErrInvalidEntitlement)
	}
	if strings.TrimSpace(e.AssetID) == "" || strings.TrimSpace(e.AssetType) == "" {
		return fmt.Errorf("%w: asset_id and asset_type are required", ErrInvalidEntitlement)
	}
	switch e.EntitlementType {
	case EntitlementTypeInspection, EntitlementTypeMaintenance, EntitlementTypeAudit, EntitlementTypeReview:
		// valid
	default:
		return fmt.Errorf("%w: unsupported entitlement_type %q", ErrInvalidEntitlement, e.EntitlementType)
	}
	switch e.Status {
	case EntitlementStatusActive:
		if e.CompletedAt != nil {
			return fmt.Errorf("%w: ACTIVE status cannot have completed_at", ErrInvalidEntitlement)
		}
	case EntitlementStatusCompleted:
		if e.CompletedAt == nil {
			return fmt.Errorf("%w: COMPLETED status requires completed_at", ErrInvalidEntitlement)
		}
	case EntitlementStatusCancelled:
		if e.CompletedAt != nil {
			return fmt.Errorf("%w: CANCELLED status cannot have completed_at", ErrInvalidEntitlement)
		}
	case EntitlementStatusExpired:
		if e.ExpiresAt == nil || e.CompletedAt != nil {
			return fmt.Errorf("%w: EXPIRED status requires expires_at and no completed_at", ErrInvalidEntitlement)
		}
	default:
		return fmt.Errorf("%w: unsupported status %q", ErrInvalidEntitlement, e.Status)
	}
	if e.ExpiresAt != nil && e.ExpiresAt.Before(e.EntitledAt) {
		return fmt.Errorf("%w: expires_at must be after entitled_at", ErrInvalidEntitlement)
	}
	if e.CompletedAt != nil && e.CompletedAt.Before(e.EntitledAt) {
		return fmt.Errorf("%w: completed_at must be after entitled_at", ErrInvalidEntitlement)
	}
	if e.CreatedBy == "" || e.CreatedAt.IsZero() || e.UpdatedBy == "" || e.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: created_by, created_at, updated_by, and updated_at are required", ErrInvalidEntitlement)
	}
	if e.Revision < 1 {
		return fmt.Errorf("%w: revision must be positive", ErrInvalidEntitlement)
	}
	for _, tag := range e.Tags {
		tag.TenantID = e.TenantID
		tag.OrganizationID = e.OrganizationID
		tag.AssetID = e.AssetID
		if err := tag.Validate(); err != nil {
			return err
		}
	}
	for _, pkg := range e.Packages {
		pkg.TenantID = e.TenantID
		pkg.OrganizationID = e.OrganizationID
		pkg.EntitlementID = e.ID
		if err := pkg.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// CanTransitionTo checks if a status transition is valid.
func (e AssetEntitlement) CanTransitionTo(target EntitlementStatus) error {
	switch e.Status {
	case EntitlementStatusActive:
		if target != EntitlementStatusCompleted && target != EntitlementStatusCancelled && target != EntitlementStatusExpired {
			return fmt.Errorf("%w: ACTIVE can only transition to COMPLETED, CANCELLED, or EXPIRED", ErrInvalidTransition)
		}
	case EntitlementStatusCompleted:
		return fmt.Errorf("%w: COMPLETED entitlements are terminal", ErrInvalidTransition)
	case EntitlementStatusCancelled:
		return fmt.Errorf("%w: CANCELLED entitlements are terminal", ErrInvalidTransition)
	case EntitlementStatusExpired:
		return fmt.Errorf("%w: EXPIRED entitlements are terminal", ErrInvalidTransition)
	default:
		return fmt.Errorf("%w: unknown current status", ErrInvalidTransition)
	}
	return nil
}

// Repository defines persistence operations for asset entitlements.
type Repository interface {
	// Register creates a new ACTIVE asset entitlement.
	Register(ctx context.Context, actor ActorContext, entitlement AssetEntitlement) (AssetEntitlement, error)
	// Get retrieves an asset entitlement by ID.
	Get(ctx context.Context, actor ActorContext, entitlementID string) (AssetEntitlement, error)
	// GetByWorkOrder retrieves all entitlements for a work order.
	GetByWorkOrder(ctx context.Context, actor ActorContext, workOrderID string) ([]AssetEntitlement, error)
	// GetByAsset retrieves all entitlements for an asset.
	GetByAsset(ctx context.Context, actor ActorContext, assetID string) ([]AssetEntitlement, error)
	// Complete transitions an ACTIVE entitlement to COMPLETED.
	Complete(ctx context.Context, actor ActorContext, entitlementID string, expectedRevision int64) (AssetEntitlement, error)
	// Cancel transitions an ACTIVE entitlement to CANCELLED.
	Cancel(ctx context.Context, actor ActorContext, entitlementID string, expectedRevision int64) (AssetEntitlement, error)
	// AssignInspector sets or updates the assigned inspector for an ACTIVE entitlement.
	AssignInspector(ctx context.Context, actor ActorContext, entitlementID string, inspectorID string, expectedRevision int64) (AssetEntitlement, error)
	// AddTag adds an untrusted physical tag to an asset.
	AddTag(ctx context.Context, actor ActorContext, tag AssetTag) (AssetTag, error)
	// ListTags lists all tags for an asset.
	ListTags(ctx context.Context, actor ActorContext, assetID string) ([]AssetTag, error)
	// RemoveTag removes a physical tag from an asset.
	RemoveTag(ctx context.Context, actor ActorContext, assetID string, tagType TagType, tagValue string) error
}

// PackageRepository defines persistence operations for offline packages.
type PackageRepository interface {
	// Generate creates a new offline package from an entitlement.
	Generate(ctx context.Context, actor ActorContext, package_ OfflinePackage) (OfflinePackage, error)
	// Get retrieves an offline package by ID.
	Get(ctx context.Context, actor ActorContext, packageID string) (OfflinePackage, error)
	// GetByEntitlement retrieves all packages for an entitlement.
	GetByEntitlement(ctx context.Context, actor ActorContext, entitlementID string) ([]OfflinePackage, error)
	// Deliver transitions a GENERATED package to DELIVERED.
	Deliver(ctx context.Context, actor ActorContext, packageID string, deviceID string, inspectorID string) (OfflinePackage, error)
	// Revoke transitions a package to REVOKED.
	Revoke(ctx context.Context, actor ActorContext, packageID string, reason string) (OfflinePackage, error)
	// ListByDevice retrieves all delivered packages for a device.
	ListByDevice(ctx context.Context, actor ActorContext, deviceID string) ([]OfflinePackage, error)
}
