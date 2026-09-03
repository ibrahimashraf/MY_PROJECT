package assetentitlement

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAssetEntitlementValidate(t *testing.T) {
	now := time.Now()
	completedAt := now.Add(time.Hour)
	tests := []struct {
		name    string
		ent     AssetEntitlement
		wantErr bool
	}{
		{
			name: "valid ACTIVE entitlement",
			ent: AssetEntitlement{
				ID:              "ent_001",
				TenantID:        "tenant_1",
				OrganizationID:  "org_1",
				WorkOrderID:     "wo_001",
				ScopeItemID:     "scope_001",
				AssetID:         "asset_001",
				AssetType:       "EQUIPMENT",
				EntitlementType: EntitlementTypeInspection,
				Status:          EntitlementStatusActive,
				EntitledAt:      now,
				CreatedBy:       "user_1",
				CreatedAt:       now,
				UpdatedBy:       "user_1",
				UpdatedAt:       now,
				Revision:        1,
			},
			wantErr: false,
		},
		{
			name: "valid COMPLETED entitlement",
			ent: AssetEntitlement{
				ID:              "ent_001",
				TenantID:        "tenant_1",
				OrganizationID:  "org_1",
				WorkOrderID:     "wo_001",
				ScopeItemID:     "scope_001",
				AssetID:         "asset_001",
				AssetType:       "EQUIPMENT",
				EntitlementType: EntitlementTypeInspection,
				Status:          EntitlementStatusCompleted,
				EntitledAt:      now,
				CompletedAt:     &completedAt,
				CreatedBy:       "user_1",
				CreatedAt:       now,
				UpdatedBy:       "user_1",
				UpdatedAt:       now,
				Revision:        1,
			},
			wantErr: false,
		},
		{
			name: "missing work_order_id",
			ent: AssetEntitlement{
				ID:              "ent_001",
				TenantID:        "tenant_1",
				OrganizationID:  "org_1",
				WorkOrderID:     "",
				ScopeItemID:     "scope_001",
				AssetID:         "asset_001",
				AssetType:       "EQUIPMENT",
				EntitlementType: EntitlementTypeInspection,
				Status:          EntitlementStatusActive,
				EntitledAt:      now,
				CreatedBy:       "user_1",
				CreatedAt:       now,
				UpdatedBy:       "user_1",
				UpdatedAt:       now,
				Revision:        1,
			},
			wantErr: true,
		},
		{
			name: "COMPLETED without completed_at",
			ent: AssetEntitlement{
				ID:              "ent_001",
				TenantID:        "tenant_1",
				OrganizationID:  "org_1",
				WorkOrderID:     "wo_001",
				ScopeItemID:     "scope_001",
				AssetID:         "asset_001",
				AssetType:       "EQUIPMENT",
				EntitlementType: EntitlementTypeInspection,
				Status:          EntitlementStatusCompleted,
				EntitledAt:      now,
				CreatedBy:       "user_1",
				CreatedAt:       now,
				UpdatedBy:       "user_1",
				UpdatedAt:       now,
				Revision:        1,
			},
			wantErr: true,
		},
		{
			name: "zero revision",
			ent: AssetEntitlement{
				ID:              "ent_001",
				TenantID:        "tenant_1",
				OrganizationID:  "org_1",
				WorkOrderID:     "wo_001",
				ScopeItemID:     "scope_001",
				AssetID:         "asset_001",
				AssetType:       "EQUIPMENT",
				EntitlementType: EntitlementTypeInspection,
				Status:          EntitlementStatusActive,
				EntitledAt:      now,
				CreatedBy:       "user_1",
				CreatedAt:       now,
				UpdatedBy:       "user_1",
				UpdatedAt:       now,
				Revision:        0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ent.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AssetEntitlement.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCanTransitionTo(t *testing.T) {
	now := time.Now()
	completedAt := now.Add(time.Hour)
	tests := []struct {
		name    string
		ent     AssetEntitlement
		target  EntitlementStatus
		wantErr bool
	}{
		{
			name: "ACTIVE to COMPLETED",
			ent: AssetEntitlement{
				Status: EntitlementStatusActive,
			},
			target:  EntitlementStatusCompleted,
			wantErr: false,
		},
		{
			name: "ACTIVE to CANCELLED",
			ent: AssetEntitlement{
				Status: EntitlementStatusActive,
			},
			target:  EntitlementStatusCancelled,
			wantErr: false,
		},
		{
			name: "COMPLETED to anything",
			ent: AssetEntitlement{
				Status:      EntitlementStatusCompleted,
				CompletedAt: &completedAt,
			},
			target:  EntitlementStatusActive,
			wantErr: true,
		},
		{
			name: "CANCELLED to anything",
			ent: AssetEntitlement{
				Status: EntitlementStatusCancelled,
			},
			target:  EntitlementStatusActive,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ent.CanTransitionTo(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanTransitionTo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOfflinePackageValidate(t *testing.T) {
	now := time.Now()
	generatedAt := now.Add(time.Minute)
	deliveredAt := now.Add(time.Hour)
	formSnapshot, _ := json.Marshal(map[string]interface{}{"fields": []interface{}{}})
	evidencePolicySnapshot, _ := json.Marshal(map[string]interface{}{"policies": []interface{}{}})
	assetContext, _ := json.Marshal(map[string]interface{}{"asset_id": "asset_001"})

	tests := []struct {
		name    string
		pkg     OfflinePackage
		wantErr bool
	}{
		{
			name: "valid PENDING package",
			pkg: OfflinePackage{
				ID:                     "pkg_001",
				TenantID:               "tenant_1",
				OrganizationID:         "org_1",
				EntitlementID:          "ent_001",
				PackageVersion:         1,
				FormSnapshot:           formSnapshot,
				EvidencePolicySnapshot: evidencePolicySnapshot,
				AssetContext:           assetContext,
				PackageHash:            "abc123",
				Status:                 PackageStatusPending,
				CreatedBy:              "user_1",
				CreatedAt:              now,
			},
			wantErr: false,
		},
		{
			name: "valid GENERATED package",
			pkg: OfflinePackage{
				ID:                     "pkg_001",
				TenantID:               "tenant_1",
				OrganizationID:         "org_1",
				EntitlementID:          "ent_001",
				PackageVersion:         1,
				FormSnapshot:           formSnapshot,
				EvidencePolicySnapshot: evidencePolicySnapshot,
				AssetContext:           assetContext,
				PackageHash:            "abc123",
				Status:                 PackageStatusGenerated,
				GeneratedAt:            &generatedAt,
				CreatedBy:              "user_1",
				CreatedAt:              now,
			},
			wantErr: false,
		},
		{
			name: "valid DELIVERED package",
			pkg: OfflinePackage{
				ID:                     "pkg_001",
				TenantID:               "tenant_1",
				OrganizationID:         "org_1",
				EntitlementID:          "ent_001",
				PackageVersion:         1,
				FormSnapshot:           formSnapshot,
				EvidencePolicySnapshot: evidencePolicySnapshot,
				AssetContext:           assetContext,
				PackageHash:            "abc123",
				Status:                 PackageStatusDelivered,
				GeneratedAt:            &generatedAt,
				DeliveredAt:            &deliveredAt,
				DeliveredToDeviceID:    "device_001",
				DeliveredToInspectorID: "inspector_001",
				CreatedBy:              "user_1",
				CreatedAt:              now,
			},
			wantErr: false,
		},
		{
			name: "GENERATED without generated_at",
			pkg: OfflinePackage{
				ID:                     "pkg_001",
				TenantID:               "tenant_1",
				OrganizationID:         "org_1",
				EntitlementID:          "ent_001",
				PackageVersion:         1,
				FormSnapshot:           formSnapshot,
				EvidencePolicySnapshot: evidencePolicySnapshot,
				AssetContext:           assetContext,
				PackageHash:            "abc123",
				Status:                 PackageStatusGenerated,
				CreatedBy:              "user_1",
				CreatedAt:              now,
			},
			wantErr: true,
		},
		{
			name: "empty form_snapshot",
			pkg: OfflinePackage{
				ID:                     "pkg_001",
				TenantID:               "tenant_1",
				OrganizationID:         "org_1",
				EntitlementID:          "ent_001",
				PackageVersion:         1,
				FormSnapshot:           nil,
				EvidencePolicySnapshot: evidencePolicySnapshot,
				AssetContext:           assetContext,
				PackageHash:            "abc123",
				Status:                 PackageStatusPending,
				CreatedBy:              "user_1",
				CreatedAt:              now,
			},
			wantErr: true,
		},
		{
			name: "REVOKED without revoked_at",
			pkg: OfflinePackage{
				ID:                     "pkg_001",
				TenantID:               "tenant_1",
				OrganizationID:         "org_1",
				EntitlementID:          "ent_001",
				PackageVersion:         1,
				FormSnapshot:           formSnapshot,
				EvidencePolicySnapshot: evidencePolicySnapshot,
				AssetContext:           assetContext,
				PackageHash:            "abc123",
				Status:                 PackageStatusRevoked,
				CreatedBy:              "user_1",
				CreatedAt:              now,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pkg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("OfflinePackage.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPackageCanTransitionTo(t *testing.T) {
	now := time.Now()
	generatedAt := now.Add(time.Minute)
	deliveredAt := now.Add(time.Hour)
	tests := []struct {
		name    string
		pkg     OfflinePackage
		target  PackageStatus
		wantErr bool
	}{
		{
			name: "PENDING to GENERATED",
			pkg: OfflinePackage{
				Status: PackageStatusPending,
			},
			target:  PackageStatusGenerated,
			wantErr: false,
		},
		{
			name: "PENDING to REVOKED",
			pkg: OfflinePackage{
				Status: PackageStatusPending,
			},
			target:  PackageStatusRevoked,
			wantErr: false,
		},
		{
			name: "GENERATED to DELIVERED",
			pkg: OfflinePackage{
				Status:      PackageStatusGenerated,
				GeneratedAt: &generatedAt,
			},
			target:  PackageStatusDelivered,
			wantErr: false,
		},
		{
			name: "DELIVERED to EXPIRED",
			pkg: OfflinePackage{
				Status:      PackageStatusDelivered,
				GeneratedAt: &generatedAt,
				DeliveredAt: &deliveredAt,
			},
			target:  PackageStatusExpired,
			wantErr: false,
		},
		{
			name: "EXPIRED to anything",
			pkg: OfflinePackage{
				Status: PackageStatusExpired,
			},
			target:  PackageStatusPending,
			wantErr: true,
		},
		{
			name: "REVOKED to anything",
			pkg: OfflinePackage{
				Status: PackageStatusRevoked,
			},
			target:  PackageStatusPending,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pkg.CanTransitionTo(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanTransitionTo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAssetTagValidate(t *testing.T) {
	now := time.Now()
	digest := ComputeTagDigest(TagTypeQR, "ABC123")
	tests := []struct {
		name    string
		tag     AssetTag
		wantErr bool
	}{
		{
			name: "valid QR tag",
			tag: AssetTag{
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				AssetID:        "asset_001",
				TagType:        TagTypeQR,
				TagValue:       "ABC123",
				TagDigest:      digest,
				AssignedAt:     now,
				AssignedBy:     "user_1",
			},
			wantErr: false,
		},
		{
			name: "empty tag_value",
			tag: AssetTag{
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				AssetID:        "asset_001",
				TagType:        TagTypeQR,
				TagValue:       "",
				TagDigest:      digest,
				AssignedAt:     now,
				AssignedBy:     "user_1",
			},
			wantErr: true,
		},
		{
			name: "unsupported tag_type",
			tag: AssetTag{
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				AssetID:        "asset_001",
				TagType:        "INVALID",
				TagValue:       "ABC123",
				TagDigest:      digest,
				AssignedAt:     now,
				AssignedBy:     "user_1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tag.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AssetTag.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestComputeTagDigest(t *testing.T) {
	digest1 := ComputeTagDigest(TagTypeQR, "ABC123")
	digest2 := ComputeTagDigest(TagTypeQR, "ABC123")
	digest3 := ComputeTagDigest(TagTypeBarcode, "ABC123")
	if digest1 != digest2 {
		t.Errorf("ComputeTagDigest not deterministic: %q != %q", digest1, digest2)
	}
	if digest1 == digest3 {
		t.Errorf("ComputeTagDigest should differ for different tag types: %q == %q", digest1, digest3)
	}
	if len(digest1) != 64 {
		t.Errorf("ComputeTagDigest digest length = %d, want 64", len(digest1))
	}
}
