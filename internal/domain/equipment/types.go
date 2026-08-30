package equipment

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type LocationLevel string

const (
	LocationBranch LocationLevel = "BRANCH"
	LocationArea   LocationLevel = "AREA"
	LocationZone   LocationLevel = "ZONE"
)

type EquipmentCategory string

const (
	CategoryChains      EquipmentCategory = "CHAINS"
	CategorySlings      EquipmentCategory = "SLINGS"
	CategoryShackles    EquipmentCategory = "SHACKLES"
	CategoryHoists      EquipmentCategory = "HOISTS"
	CategoryCranes      EquipmentCategory = "CRANES"
	CategoryHarnesses   EquipmentCategory = "HARNESSES"
	CategoryLanyards    EquipmentCategory = "LANYARDS"
	CategoryAnchors     EquipmentCategory = "ANCHORS"
	CategoryVehicles    EquipmentCategory = "VEHICLES"
	CategoryFixedPlant  EquipmentCategory = "FIXED_PLANT"
	CategoryOther       EquipmentCategory = "OTHER"
)

type EquipmentStatus string

const (
	EquipmentActive  EquipmentStatus = "ACTIVE"
	EquipmentRetired EquipmentStatus = "RETIRED"
)

type LocationStatus string

const (
	LocationActive  LocationStatus = "ACTIVE"
	LocationRetired LocationStatus = "RETIRED"
)

var (
	ErrInvalidLocationIdentity = errors.New("location identity is incomplete")
	ErrInvalidEquipmentType    = errors.New("equipment type is invalid")
	ErrInvalidAssetIdentity    = errors.New("asset identity is incomplete")
	ErrInvalidHierarchy        = errors.New("location hierarchy is invalid")
	ErrCustomFieldValidation   = errors.New("custom field validation failed")
	ErrFieldCodeFormat         = errors.New("field_code must match ^[a-z_][a-z0-9_]{0,63}$")
	ErrInvalidDataType         = errors.New("invalid data type")
	ErrEnumValuesRequired      = errors.New("enum values required for ENUM data type")
)

type Branch struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	OrganizationID string          `json:"organization_id"`
	Code           string          `json:"code"`
	Name           string          `json:"name"`
	Address        string          `json:"address,omitempty"`
	Status         LocationStatus  `json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
}

func (b Branch) Validate() error {
	if strings.TrimSpace(b.ID) == "" || strings.TrimSpace(b.TenantID) == "" || strings.TrimSpace(b.OrganizationID) == "" {
		return fmt.Errorf("%w: id, tenant_id, organization_id required", ErrInvalidLocationIdentity)
	}
	if strings.TrimSpace(b.Code) == "" || len(b.Code) > 32 {
		return fmt.Errorf("%w: code must be 1-32 chars", ErrInvalidLocationIdentity)
	}
	if strings.TrimSpace(b.Name) == "" || len(b.Name) > 200 {
		return fmt.Errorf("%w: name must be 2-200 chars", ErrInvalidLocationIdentity)
	}
	if b.Status != LocationActive && b.Status != LocationRetired {
		return fmt.Errorf("%w: invalid status", ErrInvalidLocationIdentity)
	}
	return nil
}

type Area struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	OrganizationID string          `json:"organization_id"`
	BranchID       string          `json:"branch_id"`
	Code           string          `json:"code"`
	Name           string          `json:"name"`
	Status         LocationStatus  `json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
}

func (a Area) Validate() error {
	if strings.TrimSpace(a.ID) == "" || strings.TrimSpace(a.TenantID) == "" || strings.TrimSpace(a.OrganizationID) == "" || strings.TrimSpace(a.BranchID) == "" {
		return fmt.Errorf("%w: id, tenant_id, organization_id, branch_id required", ErrInvalidLocationIdentity)
	}
	if strings.TrimSpace(a.Code) == "" || len(a.Code) > 32 {
		return fmt.Errorf("%w: code must be 1-32 chars", ErrInvalidLocationIdentity)
	}
	if strings.TrimSpace(a.Name) == "" || len(a.Name) > 200 {
		return fmt.Errorf("%w: name must be 2-200 chars", ErrInvalidLocationIdentity)
	}
	if a.Status != LocationActive && a.Status != LocationRetired {
		return fmt.Errorf("%w: invalid status", ErrInvalidLocationIdentity)
	}
	return nil
}

type Zone struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	OrganizationID string          `json:"organization_id"`
	AreaID         string          `json:"area_id"`
	Code           string          `json:"code"`
	Name           string          `json:"name"`
	Status         LocationStatus  `json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
}

func (z Zone) Validate() error {
	if strings.TrimSpace(z.ID) == "" || strings.TrimSpace(z.TenantID) == "" || strings.TrimSpace(z.OrganizationID) == "" || strings.TrimSpace(z.AreaID) == "" {
		return fmt.Errorf("%w: id, tenant_id, organization_id, area_id required", ErrInvalidLocationIdentity)
	}
	if strings.TrimSpace(z.Code) == "" || len(z.Code) > 32 {
		return fmt.Errorf("%w: code must be 1-32 chars", ErrInvalidLocationIdentity)
	}
	if strings.TrimSpace(z.Name) == "" || len(z.Name) > 200 {
		return fmt.Errorf("%w: name must be 2-200 chars", ErrInvalidLocationIdentity)
	}
	if z.Status != LocationActive && z.Status != LocationRetired {
		return fmt.Errorf("%w: invalid status", ErrInvalidLocationIdentity)
	}
	return nil
}

type EquipmentType struct {
	ID                 string            `json:"id"`
	TenantID           string            `json:"tenant_id"`
	OrganizationID     string            `json:"organization_id"`
	Code               string            `json:"code"`
	Name               string            `json:"name"`
	Category           EquipmentCategory `json:"category"`
	StandardIntervalDays int             `json:"standard_interval_days"`
	Status             EquipmentStatus   `json:"status"`
	CreatedAt          time.Time         `json:"created_at"`
}

func (et EquipmentType) Validate() error {
	if strings.TrimSpace(et.ID) == "" || strings.TrimSpace(et.TenantID) == "" || strings.TrimSpace(et.OrganizationID) == "" {
		return fmt.Errorf("%w: id, tenant_id, organization_id required", ErrInvalidEquipmentType)
	}
	if strings.TrimSpace(et.Code) == "" || len(et.Code) > 64 {
		return fmt.Errorf("%w: code must be 1-64 chars", ErrInvalidEquipmentType)
	}
	if strings.TrimSpace(et.Name) == "" || len(et.Name) > 200 {
		return fmt.Errorf("%w: name must be 2-200 chars", ErrInvalidEquipmentType)
	}
	validCategories := map[EquipmentCategory]bool{
		CategoryChains: true, CategorySlings: true, CategoryShackles: true,
		CategoryHoists: true, CategoryCranes: true, CategoryHarnesses: true,
		CategoryLanyards: true, CategoryAnchors: true, CategoryVehicles: true,
		CategoryFixedPlant: true, CategoryOther: true,
	}
	if !validCategories[et.Category] {
		return fmt.Errorf("%w: invalid category", ErrInvalidEquipmentType)
	}
	if et.StandardIntervalDays <= 0 {
		return fmt.Errorf("%w: standard_interval_days must be positive", ErrInvalidEquipmentType)
	}
	if et.Status != EquipmentActive && et.Status != EquipmentRetired {
		return fmt.Errorf("%w: invalid status", ErrInvalidEquipmentType)
	}
	return nil
}

type FieldDataType string

const (
	FieldTypeText     FieldDataType = "TEXT"
	FieldTypeInteger  FieldDataType = "INTEGER"
	FieldTypeNumeric  FieldDataType = "NUMERIC"
	FieldTypeBoolean  FieldDataType = "BOOLEAN"
	FieldTypeDate     FieldDataType = "DATE"
	FieldTypeEnum     FieldDataType = "ENUM"
)

type EquipmentTypeField struct {
	ID               string        `json:"id"`
	TenantID         string        `json:"tenant_id"`
	OrganizationID   string        `json:"organization_id"`
	EquipmentTypeID  string        `json:"equipment_type_id"`
	FieldCode        string        `json:"field_code"`
	Label            string        `json:"label"`
	DataType         FieldDataType `json:"data_type"`
	EnumValues       []string      `json:"enum_values,omitempty"`
	Required         bool          `json:"required"`
	DisplayOrder     int           `json:"display_order"`
	HelpText         string        `json:"help_text,omitempty"`
}

func (f EquipmentTypeField) Validate() error {
	if strings.TrimSpace(f.ID) == "" || strings.TrimSpace(f.TenantID) == "" || strings.TrimSpace(f.OrganizationID) == "" || strings.TrimSpace(f.EquipmentTypeID) == "" {
		return fmt.Errorf("%w: id, tenant_id, organization_id, equipment_type_id required", ErrInvalidEquipmentType)
	}
	if !fieldCodeRegex.MatchString(f.FieldCode) {
		return ErrFieldCodeFormat
	}
	if strings.TrimSpace(f.Label) == "" || len(f.Label) > 100 {
		return fmt.Errorf("%w: label must be 1-100 chars", ErrInvalidEquipmentType)
	}
	validTypes := map[FieldDataType]bool{
		FieldTypeText: true, FieldTypeInteger: true, FieldTypeNumeric: true,
		FieldTypeBoolean: true, FieldTypeDate: true, FieldTypeEnum: true,
	}
	if !validTypes[f.DataType] {
		return ErrInvalidDataType
	}
	if f.DataType == FieldTypeEnum && len(f.EnumValues) == 0 {
		return ErrEnumValuesRequired
	}
	return nil
}

// Asset extends the existing asset_registry with hierarchy + custom fields
type Asset struct {
	ID               string         `json:"id"`
	TenantID         string         `json:"tenant_id"`
	OrganizationID   string         `json:"organization_id"`
	AssetID          string         `json:"asset_id"`
	AssetType        string         `json:"asset_type"`
	SerialNumber     string         `json:"serial_number"`
	Description      string         `json:"description"`
	LifecycleState   string         `json:"lifecycle_state"`
	BranchID         string         `json:"branch_id,omitempty"`
	AreaID           string         `json:"area_id,omitempty"`
	ZoneID           string         `json:"zone_id,omitempty"`
	EquipmentTypeID  string         `json:"equipment_type_id,omitempty"`
	CustomFields     map[string]any `json:"custom_fields,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	CreatedBy        string         `json:"created_by"`
	UpdatedBy        string         `json:"updated_by"`
}

func (a Asset) Validate() error {
	if strings.TrimSpace(a.ID) == "" || strings.TrimSpace(a.TenantID) == "" || strings.TrimSpace(a.OrganizationID) == "" {
		return fmt.Errorf("%w: id, tenant_id, organization_id required", ErrInvalidAssetIdentity)
	}
	if strings.TrimSpace(a.AssetID) == "" || strings.TrimSpace(a.AssetType) == "" || strings.TrimSpace(a.SerialNumber) == "" || strings.TrimSpace(a.Description) == "" {
		return fmt.Errorf("%w: asset_id, asset_type, serial_number, description required", ErrInvalidAssetIdentity)
	}
	if a.LifecycleState != "ACTIVE" && a.LifecycleState != "RETIRED" {
		return fmt.Errorf("%w: invalid lifecycle_state", ErrInvalidAssetIdentity)
	}
	return nil
}

func (a Asset) ValidateCustomFields(fields []EquipmentTypeField) error {
	if len(fields) == 0 {
		return nil
	}
	custom := a.CustomFields
	if custom == nil {
		custom = map[string]any{}
	}
for _, f := range fields {
			val, exists := custom[f.FieldCode]
			isEmptyString := false
			if val != nil {
				if valStr, ok := val.(string); ok {
					isEmptyString = strings.TrimSpace(valStr) == ""
				}
			}
			if f.Required && (!exists || val == nil || isEmptyString) {
				return fmt.Errorf("%w: required field %s is missing", ErrCustomFieldValidation, f.FieldCode)
			}
		if exists && val != nil {
			if err := validateFieldValue(f, val); err != nil {
				return fmt.Errorf("%w: field %s: %v", ErrCustomFieldValidation, f.FieldCode, err)
			}
		}
	}
	return nil
}

func validateFieldValue(f EquipmentTypeField, val any) error {
	switch f.DataType {
	case FieldTypeText:
		if _, ok := val.(string); !ok {
			return fmt.Errorf("expected string")
		}
	case FieldTypeInteger:
		switch v := val.(type) {
		case float64:
			if v != float64(int64(v)) {
				return fmt.Errorf("expected integer")
			}
		case int, int64:
		default:
			return fmt.Errorf("expected integer")
		}
	case FieldTypeNumeric:
		if _, ok := val.(float64); !ok {
			if _, ok := val.(int); !ok {
				return fmt.Errorf("expected numeric")
			}
		}
	case FieldTypeBoolean:
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("expected boolean")
		}
	case FieldTypeDate:
		if _, ok := val.(string); !ok {
			return fmt.Errorf("expected date string")
		}
	case FieldTypeEnum:
		strVal, ok := val.(string)
		if !ok {
			return fmt.Errorf("expected string for enum")
		}
		found := false
		for _, ev := range f.EnumValues {
			if ev == strVal {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("value not in enum: %v", f.EnumValues)
		}
	}
	return nil
}

var fieldCodeRegex = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,63}$`)

type AssetFilter struct {
	TenantID        string
	OrganizationID  string
	BranchID        string
	AreaID          string
	ZoneID          string
	EquipmentTypeID string
	AssetType       string
	LifecycleState  string
	CustomFields    map[string]any
	Limit           int
	Offset          int
}