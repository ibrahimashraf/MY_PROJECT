package formdefinition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidActor    = errors.New("form definition actor context is invalid")
	ErrInvalidForm     = errors.New("form definition is invalid")
	ErrInvalidField    = errors.New("form field is invalid")
	ErrInvalidPolicy   = errors.New("evidence policy is invalid")
	ErrImmutable       = errors.New("form definition conflicts with an immutable record")
	ErrInvalidStatus   = errors.New("invalid status transition")
	ErrDuplicateField  = errors.New("duplicate field code in form")
	ErrDuplicatePolicy = errors.New("duplicate evidence policy for field")
)

// ActorContext is the server-derived actor for form operations.
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

// FormStatus represents the lifecycle status of a form version.
type FormStatus string

const (
	FormStatusDraft    FormStatus = "DRAFT"
	FormStatusApproved FormStatus = "APPROVED"
	FormStatusRetired  FormStatus = "RETIRED"
)

// FieldType represents the type of a form field.
type FieldType string

const (
	FieldTypeText          FieldType = "TEXT"
	FieldTypeNumber        FieldType = "NUMBER"
	FieldTypeDate          FieldType = "DATE"
	FieldTypeBoolean       FieldType = "BOOLEAN"
	FieldTypeSingleSelect  FieldType = "SINGLE_SELECT"
	FieldTypeMultiSelect   FieldType = "MULTI_SELECT"
	FieldTypeFileReference FieldType = "FILE_REFERENCE"
	FieldTypeSignature     FieldType = "SIGNATURE"
	FieldTypeMeasurement   FieldType = "MEASUREMENT"
)

// EvidenceType represents the type of evidence required.
type EvidenceType string

const (
	EvidenceTypePhoto       EvidenceType = "PHOTO"
	EvidenceTypeDocument    EvidenceType = "DOCUMENT"
	EvidenceTypeSignature   EvidenceType = "SIGNATURE"
	EvidenceTypeVideo       EvidenceType = "VIDEO"
	EvidenceTypeAudio       EvidenceType = "AUDIO"
	EvidenceTypeMeasurement EvidenceType = "MEASUREMENT"
)

// Classification represents the sensitivity classification of evidence.
type Classification string

const (
	ClassificationPublic       Classification = "PUBLIC"
	ClassificationInternal     Classification = "INTERNAL"
	ClassificationConfidential Classification = "CONFIDENTIAL"
	ClassificationRestricted   Classification = "RESTRICTED"
)

// ValidationRules defines optional JSON schema validation for a field.
type ValidationRules struct {
	MinLength *int     `json:"min_length,omitempty"`
	MaxLength *int     `json:"max_length,omitempty"`
	MinValue  *float64 `json:"min_value,omitempty"`
	MaxValue  *float64 `json:"max_value,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Unit      string   `json:"unit,omitempty"`
	MinItems  *int     `json:"min_items,omitempty"`
	MaxItems  *int     `json:"max_items,omitempty"`
}

// FieldOption represents a selectable option for SINGLE_SELECT or MULTI_SELECT fields.
type FieldOption struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

// EvidencePolicy defines evidence requirements for a form field.
type EvidencePolicy struct {
	TenantID            string
	OrganizationID      string
	FormID              string
	FieldID             string
	EvidenceType        EvidenceType
	Required            bool
	MaxCount            *int64
	MaxSizeBytes        *int64
	AllowedContentTypes []string
	RetentionDays       *int64
	Classification      Classification
}

// Validate ensures the evidence policy is consistent.
func (p EvidencePolicy) Validate() error {
	if strings.TrimSpace(p.FormID) == "" || strings.TrimSpace(p.FieldID) == "" {
		return fmt.Errorf("%w: form_id and field_id are required", ErrInvalidPolicy)
	}
	if p.EvidenceType == "" {
		return fmt.Errorf("%w: evidence_type is required", ErrInvalidPolicy)
	}
	switch p.EvidenceType {
	case EvidenceTypePhoto, EvidenceTypeDocument, EvidenceTypeSignature,
		EvidenceTypeVideo, EvidenceTypeAudio, EvidenceTypeMeasurement:
		// valid
	default:
		return fmt.Errorf("%w: unsupported evidence_type %q", ErrInvalidPolicy, p.EvidenceType)
	}
	if p.MaxCount != nil && *p.MaxCount <= 0 {
		return fmt.Errorf("%w: max_count must be positive", ErrInvalidPolicy)
	}
	if p.MaxSizeBytes != nil && *p.MaxSizeBytes <= 0 {
		return fmt.Errorf("%w: max_size_bytes must be positive", ErrInvalidPolicy)
	}
	if p.RetentionDays != nil && *p.RetentionDays <= 0 {
		return fmt.Errorf("%w: retention_days must be positive", ErrInvalidPolicy)
	}
	switch p.Classification {
	case ClassificationPublic, ClassificationInternal, ClassificationConfidential, ClassificationRestricted:
		// valid
	default:
		if p.Classification == "" {
			p.Classification = ClassificationConfidential
		} else {
			return fmt.Errorf("%w: unsupported classification %q", ErrInvalidPolicy, p.Classification)
		}
	}
	return nil
}

// FormField represents a field within a form version.
type FormField struct {
	TenantID         string
	OrganizationID   string
	FormID           string
	FieldID          string
	FieldCode        string
	FieldType        FieldType
	Label            string
	Description      string
	Required         bool
	Critical         bool
	SortOrder        int
	ValidationRules  *ValidationRules
	Options          []FieldOption
	DefaultValue     string
	SectionID        string
	SectionLabel     string
	EvidencePolicies []EvidencePolicy
}

// Validate ensures the form field is consistent.
func (f FormField) Validate() error {
	if strings.TrimSpace(f.FormID) == "" || strings.TrimSpace(f.FieldID) == "" || strings.TrimSpace(f.FieldCode) == "" {
		return fmt.Errorf("%w: form_id, field_id, and field_code are required", ErrInvalidField)
	}
	if strings.TrimSpace(f.Label) == "" {
		return fmt.Errorf("%w: label is required", ErrInvalidField)
	}
	if f.SortOrder < 0 {
		return fmt.Errorf("%w: sort_order must be non-negative", ErrInvalidField)
	}
	switch f.FieldType {
	case FieldTypeText, FieldTypeNumber, FieldTypeDate, FieldTypeBoolean,
		FieldTypeSingleSelect, FieldTypeMultiSelect,
		FieldTypeFileReference, FieldTypeSignature, FieldTypeMeasurement:
		// valid
	default:
		return fmt.Errorf("%w: unsupported field_type %q", ErrInvalidField, f.FieldType)
	}
	if (f.FieldType == FieldTypeSingleSelect || f.FieldType == FieldTypeMultiSelect) && len(f.Options) == 0 {
		return fmt.Errorf("%w: select fields require at least one option", ErrInvalidField)
	}
	for _, opt := range f.Options {
		if strings.TrimSpace(opt.Code) == "" || strings.TrimSpace(opt.Label) == "" {
			return fmt.Errorf("%w: option code and label are required", ErrInvalidField)
		}
	}
	return nil
}

// FormVersion represents an immutable versioned form definition.
type FormVersion struct {
	ID             string
	TenantID       string
	OrganizationID string
	FormCode       string
	Version        int
	Title          string
	Description    string
	AssetType      string
	Status         FormStatus
	CatalogVersion int
	CreatedBy      string
	CreatedAt      time.Time
	ApprovedBy     string
	ApprovedAt     *time.Time
	RetiredBy      string
	RetiredAt      *time.Time
	Fields         []FormField
}

// Validate ensures the form version is consistent.
func (v FormVersion) Validate() error {
	if strings.TrimSpace(v.ID) == "" || strings.TrimSpace(v.FormCode) == "" || strings.TrimSpace(v.Title) == "" {
		return fmt.Errorf("%w: id, form_code, and title are required", ErrInvalidForm)
	}
	if v.Version <= 0 {
		return fmt.Errorf("%w: version must be positive", ErrInvalidForm)
	}
	if v.CatalogVersion <= 0 {
		return fmt.Errorf("%w: catalog_version must be positive", ErrInvalidForm)
	}
	switch v.Status {
	case FormStatusDraft:
		if v.ApprovedBy != "" || v.ApprovedAt != nil || v.RetiredBy != "" || v.RetiredAt != nil {
			return fmt.Errorf("%w: DRAFT forms cannot have approval or retirement metadata", ErrInvalidForm)
		}
	case FormStatusApproved:
		if v.ApprovedBy == "" || v.ApprovedAt == nil {
			return fmt.Errorf("%w: APPROVED forms require approval metadata", ErrInvalidForm)
		}
		if v.RetiredBy != "" || v.RetiredAt != nil {
			return fmt.Errorf("%w: APPROVED forms cannot have retirement metadata", ErrInvalidForm)
		}
	case FormStatusRetired:
		if v.RetiredBy == "" || v.RetiredAt == nil {
			return fmt.Errorf("%w: RETIRED forms require retirement metadata", ErrInvalidForm)
		}
	default:
		return fmt.Errorf("%w: invalid status %q", ErrInvalidForm, v.Status)
	}
	if v.CreatedBy == "" || v.CreatedAt.IsZero() {
		return fmt.Errorf("%w: created_by and created_at are required", ErrInvalidForm)
	}
	if len(v.Fields) == 0 {
		return fmt.Errorf("%w: at least one field is required", ErrInvalidForm)
	}
	fieldCodes := map[string]bool{}
	for _, field := range v.Fields {
		field.FormID = v.ID
		field.TenantID = v.TenantID
		field.OrganizationID = v.OrganizationID
		if err := field.Validate(); err != nil {
			return err
		}
		if fieldCodes[field.FieldCode] {
			return fmt.Errorf("%w: duplicate field_code %q", ErrDuplicateField, field.FieldCode)
		}
		fieldCodes[field.FieldCode] = true
		for _, policy := range field.EvidencePolicies {
			policy.FormID = v.ID
			policy.FieldID = field.FieldID
			policy.TenantID = v.TenantID
			policy.OrganizationID = v.OrganizationID
			if err := policy.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

// CanTransitionTo checks if a status transition is valid.
func (v FormVersion) CanTransitionTo(target FormStatus) error {
	switch v.Status {
	case FormStatusDraft:
		if target != FormStatusApproved {
			return fmt.Errorf("%w: DRAFT can only transition to APPROVED", ErrInvalidStatus)
		}
	case FormStatusApproved:
		if target != FormStatusRetired {
			return fmt.Errorf("%w: APPROVED can only transition to RETIRED", ErrInvalidStatus)
		}
	case FormStatusRetired:
		return fmt.Errorf("%w: RETIRED forms are terminal", ErrInvalidStatus)
	default:
		return fmt.Errorf("%w: unknown current status", ErrInvalidStatus)
	}
	return nil
}

// MarshalJSON implements json.Marshaler for EvidencePolicy.AllowedContentTypes.
func (p EvidencePolicy) MarshalJSON() ([]byte, error) {
	type Alias EvidencePolicy
	return json.Marshal(struct {
		Alias
		AllowedContentTypes []string `json:"allowed_content_types,omitempty"`
	}{
		Alias:               (Alias)(p),
		AllowedContentTypes: p.AllowedContentTypes,
	})
}

// Repository defines persistence operations for form definitions.
type Repository interface {
	// RegisterDraft creates a new DRAFT form version (returns created=false if identical exists).
	RegisterDraft(ctx context.Context, actor ActorContext, form FormVersion) (FormVersion, bool, error)
	// Approve transitions a DRAFT form to APPROVED.
	Approve(ctx context.Context, actor ActorContext, formID string, expectedVersion int) (FormVersion, error)
	// Retire transitions an APPROVED form to RETIRED.
	Retire(ctx context.Context, actor ActorContext, formID string, expectedVersion int) (FormVersion, error)
	// Get retrieves a form version by ID.
	Get(ctx context.Context, actor ActorContext, formID string) (FormVersion, error)
	// GetByCode retrieves the latest version of a form by form_code.
	GetByCode(ctx context.Context, actor ActorContext, formCode string) (FormVersion, error)
	// ListByAssetType retrieves form versions for a specific asset type.
	ListByAssetType(ctx context.Context, actor ActorContext, assetType string) ([]FormVersion, error)
	// ListByStatus retrieves form versions by status.
	ListByStatus(ctx context.Context, actor ActorContext, status FormStatus) ([]FormVersion, error)
}
