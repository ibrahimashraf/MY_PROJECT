package formdefinition

import (
	"testing"
	"time"
)

func TestFormVersionValidate(t *testing.T) {
	now := time.Now()
	_APPROVED_AT := now
	tests := []struct {
		name    string
		form    FormVersion
		wantErr bool
	}{
		{
			name: "valid DRAFT form",
			form: FormVersion{
				ID:             "form_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				FormCode:       "inspection_basic",
				Version:        1,
				Title:          "Basic Inspection Form",
				Status:         FormStatusDraft,
				CatalogVersion: 1,
				CreatedBy:      "user_1",
				CreatedAt:      now,
				Fields: []FormField{
					{
						FieldID:   "field_001",
						FieldCode: "asset_id",
						FieldType: FieldTypeText,
						Label:     "Asset ID",
						Required:  true,
						SortOrder: 0,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid APPROVED form",
			form: FormVersion{
				ID:             "form_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				FormCode:       "inspection_basic",
				Version:        1,
				Title:          "Basic Inspection Form",
				Status:         FormStatusApproved,
				CatalogVersion: 1,
				CreatedBy:      "user_1",
				CreatedAt:      now,
				ApprovedBy:     "approver_1",
				ApprovedAt:     &_APPROVED_AT,
				Fields: []FormField{
					{
						FieldID:   "field_001",
						FieldCode: "asset_id",
						FieldType: FieldTypeText,
						Label:     "Asset ID",
						Required:  true,
						SortOrder: 0,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing form_code",
			form: FormVersion{
				ID:             "form_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				FormCode:       "",
				Version:        1,
				Title:          "Basic Inspection Form",
				Status:         FormStatusDraft,
				CatalogVersion: 1,
				CreatedBy:      "user_1",
				CreatedAt:      now,
				Fields: []FormField{
					{
						FieldID:   "field_001",
						FieldCode: "asset_id",
						FieldType: FieldTypeText,
						Label:     "Asset ID",
						Required:  true,
						SortOrder: 0,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "zero version",
			form: FormVersion{
				ID:             "form_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				FormCode:       "inspection_basic",
				Version:        0,
				Title:          "Basic Inspection Form",
				Status:         FormStatusDraft,
				CatalogVersion: 1,
				CreatedBy:      "user_1",
				CreatedAt:      now,
				Fields: []FormField{
					{
						FieldID:   "field_001",
						FieldCode: "asset_id",
						FieldType: FieldTypeText,
						Label:     "Asset ID",
						Required:  true,
						SortOrder: 0,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no fields",
			form: FormVersion{
				ID:             "form_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				FormCode:       "inspection_basic",
				Version:        1,
				Title:          "Basic Inspection Form",
				Status:         FormStatusDraft,
				CatalogVersion: 1,
				CreatedBy:      "user_1",
				CreatedAt:      now,
				Fields:         []FormField{},
			},
			wantErr: true,
		},
		{
			name: "DRAFT with approval metadata",
			form: FormVersion{
				ID:             "form_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				FormCode:       "inspection_basic",
				Version:        1,
				Title:          "Basic Inspection Form",
				Status:         FormStatusDraft,
				CatalogVersion: 1,
				CreatedBy:      "user_1",
				CreatedAt:      now,
				ApprovedBy:     "approver_1",
				ApprovedAt:     &_APPROVED_AT,
				Fields: []FormField{
					{
						FieldID:   "field_001",
						FieldCode: "asset_id",
						FieldType: FieldTypeText,
						Label:     "Asset ID",
						Required:  true,
						SortOrder: 0,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.form.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FormVersion.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormFieldValidate(t *testing.T) {
	tests := []struct {
		name    string
		field   FormField
		wantErr bool
	}{
		{
			name: "valid TEXT field",
			field: FormField{
				FormID:    "form_001",
				FieldID:   "field_001",
				FieldCode: "asset_id",
				FieldType: FieldTypeText,
				Label:     "Asset ID",
				Required:  true,
				SortOrder: 0,
			},
			wantErr: false,
		},
		{
			name: "valid SINGLE_SELECT field with options",
			field: FormField{
				FormID:    "form_001",
				FieldID:   "field_001",
				FieldCode: "severity",
				FieldType: FieldTypeSingleSelect,
				Label:     "Severity Level",
				Required:  true,
				SortOrder: 0,
				Options: []FieldOption{
					{Code: "critical", Label: "Critical"},
					{Code: "major", Label: "Major"},
					{Code: "minor", Label: "Minor"},
				},
			},
			wantErr: false,
		},
		{
			name: "SINGLE_SELECT without options",
			field: FormField{
				FormID:    "form_001",
				FieldID:   "field_001",
				FieldCode: "severity",
				FieldType: FieldTypeSingleSelect,
				Label:     "Severity Level",
				Required:  true,
				SortOrder: 0,
			},
			wantErr: true,
		},
		{
			name: "missing field_code",
			field: FormField{
				FormID:    "form_001",
				FieldID:   "field_001",
				FieldCode: "",
				FieldType: FieldTypeText,
				Label:     "Asset ID",
				Required:  true,
				SortOrder: 0,
			},
			wantErr: true,
		},
		{
			name: "negative sort_order",
			field: FormField{
				FormID:    "form_001",
				FieldID:   "field_001",
				FieldCode: "asset_id",
				FieldType: FieldTypeText,
				Label:     "Asset ID",
				Required:  true,
				SortOrder: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.field.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FormField.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCanTransitionTo(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		current FormVersion
		target  FormStatus
		wantErr bool
	}{
		{
			name: "DRAFT to APPROVED",
			current: FormVersion{
				Status: FormStatusDraft,
			},
			target:  FormStatusApproved,
			wantErr: false,
		},
		{
			name: "DRAFT to RETIRED",
			current: FormVersion{
				Status: FormStatusDraft,
			},
			target:  FormStatusRetired,
			wantErr: true,
		},
		{
			name: "APPROVED to RETIRED",
			current: FormVersion{
				Status:     FormStatusApproved,
				ApprovedBy: "approver_1",
				ApprovedAt: &now,
			},
			target:  FormStatusRetired,
			wantErr: false,
		},
		{
			name: "RETIRED to anything",
			current: FormVersion{
				Status:     FormStatusRetired,
				ApprovedBy: "approver_1",
				ApprovedAt: &now,
				RetiredBy:  "retirer_1",
				RetiredAt:  &now,
			},
			target:  FormStatusDraft,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.current.CanTransitionTo(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanTransitionTo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
