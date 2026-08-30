package equipment

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Conflict describes a server-authoritative rejection that must not be silently merged.
type Conflict struct {
	Code             string
	AggregateID      string
	ExpectedRevision int64
	ActualRevision   int64
	OperationID      string
	Message          string
}

const (
	ConflictStaleRevision       = "stale_revision"
	ConflictIdempotencyMismatch = "idempotency_key_reused_with_different_payload"
	ConflictOutOfScope          = "out_of_scope"
	ConflictTenantMismatch      = "tenant_mismatch"
	ConflictAlreadyExists       = "already_exists"
	ConflictReferencedNotFound  = "referenced_not_found"
)

// ActorContext represents the authenticated actor performing the operation.
type ActorContext struct {
	TenantID       string
	OrganizationID string
	ActorID        string
	Role           string
}

func (a ActorContext) Validate() error {
	if strings.TrimSpace(a.TenantID) == "" {
		return ErrInvalidLocationIdentity
	}
	if strings.TrimSpace(a.OrganizationID) == "" {
		return ErrInvalidLocationIdentity
	}
	if strings.TrimSpace(a.ActorID) == "" {
		return ErrInvalidLocationIdentity
	}
	return nil
}

// MutationReceipt confirms a successful mutation.
type MutationReceipt struct {
	OperationID      string    `json:"operation_id"`
	IdempotencyKey   string    `json:"idempotency_key"`
	TenantID         string    `json:"tenant_id"`
	OrganizationID   string    `json:"organization_id"`
	AggregateID      string    `json:"aggregate_id"`
	Revision         int64     `json:"revision"`
	Status           string    `json:"status"`
	CompletedAt      time.Time `json:"completed_at"`
}

const (
	ReceiptAccepted = "accepted"
	ReceiptRejected = "rejected"
)

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

var ErrInvalidOperation = errors.New("operation metadata is incomplete")

// Repository is the authoritative persistence boundary for equipment register.
type Repository interface {
	// Location hierarchy
	GetBranch(ctx context.Context, actor ActorContext, id string) (Branch, error)
	ListBranches(ctx context.Context, actor ActorContext, status LocationStatus) ([]Branch, error)
	CreateBranch(ctx context.Context, actor ActorContext, branch Branch, op OperationMeta) (MutationReceipt, error)
	UpdateBranch(ctx context.Context, actor ActorContext, branch Branch, op OperationMeta) (MutationReceipt, error)

	GetArea(ctx context.Context, actor ActorContext, id string) (Area, error)
	ListAreas(ctx context.Context, actor ActorContext, branchID string, status LocationStatus) ([]Area, error)
	CreateArea(ctx context.Context, actor ActorContext, area Area, op OperationMeta) (MutationReceipt, error)
	UpdateArea(ctx context.Context, actor ActorContext, area Area, op OperationMeta) (MutationReceipt, error)

	GetZone(ctx context.Context, actor ActorContext, id string) (Zone, error)
	ListZones(ctx context.Context, actor ActorContext, areaID string, status LocationStatus) ([]Zone, error)
	CreateZone(ctx context.Context, actor ActorContext, zone Zone, op OperationMeta) (MutationReceipt, error)
	UpdateZone(ctx context.Context, actor ActorContext, zone Zone, op OperationMeta) (MutationReceipt, error)

	// Equipment types and custom fields
	GetEquipmentType(ctx context.Context, actor ActorContext, id string) (EquipmentType, error)
	ListEquipmentTypes(ctx context.Context, actor ActorContext, category EquipmentCategory, status EquipmentStatus) ([]EquipmentType, error)
	CreateEquipmentType(ctx context.Context, actor ActorContext, et EquipmentType, op OperationMeta) (MutationReceipt, error)
	UpdateEquipmentType(ctx context.Context, actor ActorContext, et EquipmentType, op OperationMeta) (MutationReceipt, error)

	GetEquipmentTypeField(ctx context.Context, actor ActorContext, id string) (EquipmentTypeField, error)
	ListEquipmentTypeFields(ctx context.Context, actor ActorContext, equipmentTypeID string) ([]EquipmentTypeField, error)
	CreateEquipmentTypeField(ctx context.Context, actor ActorContext, field EquipmentTypeField, op OperationMeta) (MutationReceipt, error)
	UpdateEquipmentTypeField(ctx context.Context, actor ActorContext, field EquipmentTypeField, op OperationMeta) (MutationReceipt, error)
	DeleteEquipmentTypeField(ctx context.Context, actor ActorContext, id string, op OperationMeta) (MutationReceipt, error)

	// Assets (extended asset_registry)
	GetAsset(ctx context.Context, actor ActorContext, id string) (Asset, error)
	ListAssets(ctx context.Context, actor ActorContext, filter AssetFilter) ([]Asset, error)
	CreateAsset(ctx context.Context, actor ActorContext, asset Asset, op OperationMeta) (MutationReceipt, error)
	UpdateAsset(ctx context.Context, actor ActorContext, asset Asset, op OperationMeta) (MutationReceipt, error)
	DeleteAsset(ctx context.Context, actor ActorContext, id string, op OperationMeta) (MutationReceipt, error)

	// Import/Export staging
	CreateImportStaging(ctx context.Context, actor ActorContext, staging ImportStaging, op OperationMeta) (MutationReceipt, error)
	UpdateImportStaging(ctx context.Context, actor ActorContext, staging ImportStaging, op OperationMeta) (MutationReceipt, error)
	GetImportStaging(ctx context.Context, actor ActorContext, id string) (ImportStaging, error)
	ListImportStaging(ctx context.Context, actor ActorContext, entityType string, status string) ([]ImportStaging, error)

	CreateExportStaging(ctx context.Context, actor ActorContext, staging ExportStaging, op OperationMeta) (MutationReceipt, error)
	UpdateExportStaging(ctx context.Context, actor ActorContext, staging ExportStaging, op OperationMeta) (MutationReceipt, error)
	GetExportStaging(ctx context.Context, actor ActorContext, id string) (ExportStaging, error)
	ListExportStaging(ctx context.Context, actor ActorContext, entityType string, status string) ([]ExportStaging, error)

	// Field mappings
	ListImportFieldMappings(ctx context.Context, actor ActorContext, entityType string) ([]ImportFieldMapping, error)
	UpsertImportFieldMapping(ctx context.Context, actor ActorContext, mapping ImportFieldMapping, op OperationMeta) (MutationReceipt, error)
	DeleteImportFieldMapping(ctx context.Context, actor ActorContext, id string, op OperationMeta) (MutationReceipt, error)

	ListExportColumnConfigs(ctx context.Context, actor ActorContext, entityType string) ([]ExportColumnConfig, error)
	UpsertExportColumnConfig(ctx context.Context, actor ActorContext, config ExportColumnConfig, op OperationMeta) (MutationReceipt, error)
	DeleteExportColumnConfig(ctx context.Context, actor ActorContext, id string, op OperationMeta) (MutationReceipt, error)
}

// TransactionRunner makes transaction scope explicit without binding the domain to a SQL driver.
type TransactionRunner interface {
	WithinTransaction(ctx context.Context, actor ActorContext, fn func(context.Context, Repository) error) error
}

// Authorizer is evaluated by the service before repository mutation.
type Authorizer interface {
	CanManageLocations(ctx context.Context, actor ActorContext) error
	CanManageEquipmentTypes(ctx context.Context, actor ActorContext) error
	CanManageAssets(ctx context.Context, actor ActorContext) error
	CanImport(ctx context.Context, actor ActorContext, entityType string) error
	CanExport(ctx context.Context, actor ActorContext, entityType string) error
}

// ServiceDependencies are injected into the application service.
type ServiceDependencies struct {
	Repository   Repository
	Transactions TransactionRunner
	Authorizer   Authorizer
}

func (d ServiceDependencies) Validate() error {
	if d.Repository == nil || d.Transactions == nil || d.Authorizer == nil {
		return ErrInvalidLocationIdentity
	}
	return nil
}

// Import/Export staging types
type ImportStaging struct {
	ID               string         `json:"id"`
	TenantID         string         `json:"tenant_id"`
	OrganizationID   string         `json:"organization_id"`
	EntityType       string         `json:"entity_type"` // WORK_ORDER, ASSET_REGISTRY, INSPECTION_RECORD
	SourceFormat     string         `json:"source_format"` // CSV, XLSX
	SourceFilename   string         `json:"source_filename,omitempty"`
	SourceChecksum   string         `json:"source_checksum,omitempty"`
	RawData          map[string]any `json:"raw_data"`
	ValidatedData    map[string]any `json:"validated_data,omitempty"`
	ValidationErrors map[string]any `json:"validation_errors,omitempty"`
	Status           string         `json:"status"` // PENDING, VALIDATING, VALIDATED, COMMITTING, COMMITTED, FAILED, ROLLED_BACK
	ImportedCount    int            `json:"imported_count"`
	FailedCount      int            `json:"failed_count"`
	CreatedBy        string         `json:"created_by"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	CommittedAt      *time.Time     `json:"committed_at,omitempty"`
	CommittedBy      string         `json:"committed_by,omitempty"`
}

type ExportStaging struct {
	ID              string         `json:"id"`
	TenantID        string         `json:"tenant_id"`
	OrganizationID  string         `json:"organization_id"`
	EntityType      string         `json:"entity_type"` // WORK_ORDER, ASSET_REGISTRY, INSPECTION_RECORD, ALL
	ExportFormat    string         `json:"export_format"` // CSV, XLSX
	FilterCriteria  map[string]any `json:"filter_criteria"`
	FilePath        string         `json:"file_path,omitempty"`
	FileChecksum    string         `json:"file_checksum,omitempty"`
	RowCount        int            `json:"row_count"`
	Status          string         `json:"status"` // PENDING, GENERATING, GENERATED, READY, EXPIRED, FAILED
	ErrorMessage    string         `json:"error_message,omitempty"`
	CreatedBy       string         `json:"created_by"`
	CreatedAt       time.Time      `json:"created_at"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
	ExpiresAt       time.Time      `json:"expires_at"`
}

type ImportFieldMapping struct {
	ID             string  `json:"id"`
	TenantID       string  `json:"tenant_id"`
	OrganizationID string  `json:"organization_id"`
	EntityType     string  `json:"entity_type"`
	SourceColumn   string  `json:"source_column"`
	TargetField    string  `json:"target_field"`
	TransformRule  string  `json:"transform_rule,omitempty"` // NONE, TRIM, UPPER, LOWER, DATE_ISO, DATE_EU, NUMERIC, BOOLEAN_YN, BOOLEAN_TF, JSON
	DefaultValue   any     `json:"default_value,omitempty"`
	IsRequired     bool    `json:"is_required"`
	ValidationRegex string `json:"validation_regex,omitempty"`
	DisplayOrder   int     `json:"display_order"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ExportColumnConfig struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	OrganizationID string    `json:"organization_id"`
	EntityType     string    `json:"entity_type"`
	ColumnName     string    `json:"column_name"`
	ColumnLabel    string    `json:"column_label"`
	DataType       string    `json:"data_type"` // TEXT, INTEGER, NUMERIC, BOOLEAN, DATE, DATETIME, JSON
	IsVisible      bool      `json:"is_visible"`
	DisplayOrder   int       `json:"display_order"`
	FormatRule     string    `json:"format_rule,omitempty"` // NONE, DATE_ISO, DATE_EU, DATETIME_ISO, BOOLEAN_YN, BOOLEAN_TF
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}