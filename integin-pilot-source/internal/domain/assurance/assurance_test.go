package assurance

import (
	"testing"
	"time"
)

func TestProjectionValidate(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name       string
		projection Projection
		wantErr    bool
	}{
		{
			name: "valid projection",
			projection: Projection{
				ID:                "proj_001",
				TenantID:          "tenant_1",
				OrganizationID:    "org_1",
				CertificateID:     "cert_001",
				InspectionID:      "insp_001",
				AssetID:           "asset_001",
				AssetType:         "EQUIPMENT",
				CertificateNumber: "CERT-001",
				CertificateStatus: "ISSUED",
				InspectionStatus:  "APPROVED",
				InspectionVerdict: "PASS",
				InspectorID:       "inspector_1",
				ProjectionHash:    "abc123",
				ProjectedAt:       now,
				CreatedAt:         now,
			},
			wantErr: false,
		},
		{
			name: "missing certificate_id",
			projection: Projection{
				ID:                "proj_001",
				TenantID:          "tenant_1",
				OrganizationID:    "org_1",
				CertificateID:     "",
				InspectionID:      "insp_001",
				AssetID:           "asset_001",
				AssetType:         "EQUIPMENT",
				CertificateNumber: "CERT-001",
				CertificateStatus: "ISSUED",
				InspectionStatus:  "APPROVED",
				InspectionVerdict: "PASS",
				InspectorID:       "inspector_1",
				ProjectionHash:    "abc123",
				ProjectedAt:       now,
				CreatedAt:         now,
			},
			wantErr: true,
		},
		{
			name: "empty projection_hash",
			projection: Projection{
				ID:                "proj_001",
				TenantID:          "tenant_1",
				OrganizationID:    "org_1",
				CertificateID:     "cert_001",
				InspectionID:      "insp_001",
				AssetID:           "asset_001",
				AssetType:         "EQUIPMENT",
				CertificateNumber: "CERT-001",
				CertificateStatus: "ISSUED",
				InspectionStatus:  "APPROVED",
				InspectionVerdict: "PASS",
				InspectorID:       "inspector_1",
				ProjectionHash:    "",
				ProjectedAt:       now,
				CreatedAt:         now,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.projection.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Projection.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestComputeProjectionHash(t *testing.T) {
	p1 := Projection{
		CertificateID:     "cert_001",
		InspectionID:      "insp_001",
		AssetID:           "asset_001",
		AssetType:         "EQUIPMENT",
		CertificateNumber: "CERT-001",
		CertificateStatus: "ISSUED",
		InspectionStatus:  "APPROVED",
		InspectionVerdict: "PASS",
		InspectorID:       "inspector_1",
	}
	p2 := p1
	h1 := ComputeProjectionHash(p1)
	h2 := ComputeProjectionHash(p2)
	if h1 != h2 {
		t.Errorf("ComputeProjectionHash() not deterministic: %q != %q", h1, h2)
	}
	p2.CertificateStatus = "REVOKED"
	h3 := ComputeProjectionHash(p2)
	if h1 == h3 {
		t.Error("ComputeProjectionHash() should differ for different statuses")
	}
}

func TestCorrectiveWorkValidate(t *testing.T) {
	now := time.Now()
	completedAt := now.Add(time.Hour)
	verifiedAt := now.Add(2 * time.Hour)
	tests := []struct {
		name    string
		work    CorrectiveWork
		wantErr bool
	}{
		{
			name: "valid OPEN work",
			work: CorrectiveWork{
				ID:             "work_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				InspectionID:   "insp_001",
				FindingID:      "finding_001",
				AssetID:        "asset_001",
				Severity:       SeverityMajor,
				Description:    "Fix damaged guard",
				Status:         WorkStatusOpen,
				CreatedBy:      "user_1",
				CreatedAt:      now,
				UpdatedBy:      "user_1",
				UpdatedAt:      now,
				Revision:       1,
			},
			wantErr: false,
		},
		{
			name: "valid COMPLETED work",
			work: CorrectiveWork{
				ID:             "work_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				InspectionID:   "insp_001",
				FindingID:      "finding_001",
				AssetID:        "asset_001",
				Severity:       SeverityMajor,
				Description:    "Fix damaged guard",
				Status:         WorkStatusCompleted,
				CompletedAt:    &completedAt,
				CreatedBy:      "user_1",
				CreatedAt:      now,
				UpdatedBy:      "user_1",
				UpdatedAt:      now,
				Revision:       1,
			},
			wantErr: false,
		},
		{
			name: "valid VERIFIED work",
			work: CorrectiveWork{
				ID:             "work_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				InspectionID:   "insp_001",
				FindingID:      "finding_001",
				AssetID:        "asset_001",
				Severity:       SeverityMajor,
				Description:    "Fix damaged guard",
				Status:         WorkStatusVerified,
				CompletedAt:    &completedAt,
				VerifiedAt:     &verifiedAt,
				VerifiedBy:     "user_2",
				CreatedBy:      "user_1",
				CreatedAt:      now,
				UpdatedBy:      "user_1",
				UpdatedAt:      now,
				Revision:       1,
			},
			wantErr: false,
		},
		{
			name: "missing inspection_id",
			work: CorrectiveWork{
				ID:             "work_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				InspectionID:   "",
				FindingID:      "finding_001",
				AssetID:        "asset_001",
				Severity:       SeverityMajor,
				Description:    "Fix damaged guard",
				Status:         WorkStatusOpen,
				CreatedBy:      "user_1",
				CreatedAt:      now,
				UpdatedBy:      "user_1",
				UpdatedAt:      now,
				Revision:       1,
			},
			wantErr: true,
		},
		{
			name: "COMPLETED without completed_at",
			work: CorrectiveWork{
				ID:             "work_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				InspectionID:   "insp_001",
				FindingID:      "finding_001",
				AssetID:        "asset_001",
				Severity:       SeverityMajor,
				Description:    "Fix damaged guard",
				Status:         WorkStatusCompleted,
				CreatedBy:      "user_1",
				CreatedAt:      now,
				UpdatedBy:      "user_1",
				UpdatedAt:      now,
				Revision:       1,
			},
			wantErr: true,
		},
		{
			name: "unsupported severity",
			work: CorrectiveWork{
				ID:             "work_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				InspectionID:   "insp_001",
				FindingID:      "finding_001",
				AssetID:        "asset_001",
				Severity:       "INVALID",
				Description:    "Fix damaged guard",
				Status:         WorkStatusOpen,
				CreatedBy:      "user_1",
				CreatedAt:      now,
				UpdatedBy:      "user_1",
				UpdatedAt:      now,
				Revision:       1,
			},
			wantErr: true,
		},
		{
			name: "zero revision",
			work: CorrectiveWork{
				ID:             "work_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				InspectionID:   "insp_001",
				FindingID:      "finding_001",
				AssetID:        "asset_001",
				Severity:       SeverityMajor,
				Description:    "Fix damaged guard",
				Status:         WorkStatusOpen,
				CreatedBy:      "user_1",
				CreatedAt:      now,
				UpdatedBy:      "user_1",
				UpdatedAt:      now,
				Revision:       0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.work.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CorrectiveWork.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWorkCanTransitionTo(t *testing.T) {
	now := time.Now()
	completedAt := now.Add(time.Hour)
	verifiedAt := now.Add(2 * time.Hour)
	tests := []struct {
		name    string
		work    CorrectiveWork
		target  WorkStatus
		wantErr bool
	}{
		{
			name: "OPEN to IN_PROGRESS",
			work: CorrectiveWork{
				Status: WorkStatusOpen,
			},
			target:  WorkStatusInProgress,
			wantErr: false,
		},
		{
			name: "OPEN to CLOSED",
			work: CorrectiveWork{
				Status: WorkStatusOpen,
			},
			target:  WorkStatusClosed,
			wantErr: false,
		},
		{
			name: "IN_PROGRESS to COMPLETED",
			work: CorrectiveWork{
				Status: WorkStatusInProgress,
			},
			target:  WorkStatusCompleted,
			wantErr: false,
		},
		{
			name: "COMPLETED to VERIFIED",
			work: CorrectiveWork{
				Status:      WorkStatusCompleted,
				CompletedAt: &completedAt,
			},
			target:  WorkStatusVerified,
			wantErr: false,
		},
		{
			name: "VERIFIED to CLOSED",
			work: CorrectiveWork{
				Status:      WorkStatusVerified,
				CompletedAt: &completedAt,
				VerifiedAt:  &verifiedAt,
				VerifiedBy:  "user_2",
			},
			target:  WorkStatusClosed,
			wantErr: false,
		},
		{
			name: "OPEN to COMPLETED",
			work: CorrectiveWork{
				Status: WorkStatusOpen,
			},
			target:  WorkStatusCompleted,
			wantErr: true,
		},
		{
			name: "CLOSED to anything",
			work: CorrectiveWork{
				Status: WorkStatusClosed,
			},
			target:  WorkStatusOpen,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.work.CanTransitionTo(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanTransitionTo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
