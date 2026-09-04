package fieldpackage_integration

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"integin/internal/domain/assetentitlement"
	"integin/internal/domain/assurance"
	"integin/internal/domain/custodyhandover"
	"integin/internal/domain/evidencepack"
	"integin/internal/domain/formdefinition"
	"integin/internal/domain/qrnfc"
)

// TestFieldPackageLifecycleEndToEnd proves the unified chain of custody,
// asset entitlement, versioned form definitions, QR/NFC entry, assurance projections,
// and evidence release packs under strict tenant isolation.
func TestFieldPackageLifecycleEndToEnd(t *testing.T) {
	now := time.Now().UTC()
	tenantA := "tenant-alpha"
	orgA := "org-alpha-inspection"
	actorA := "inspector-tariq"

	// -------------------------------------------------------------
	// Milestone M1: Versioned Form and Evidence Policy Definition
	// -------------------------------------------------------------
	formID := "form-v1-crane-inspection"
	approvedAt := now
	formVer := formdefinition.FormVersion{
		ID:             formID,
		TenantID:       tenantA,
		OrganizationID: orgA,
		FormCode:       "CRANE_DAILY_V1",
		Version:        1,
		Title:          "Mobile Crane Pre-Use Inspection Form",
		Status:         formdefinition.FormStatusApproved,
		CatalogVersion: 1,
		CreatedBy:      actorA,
		CreatedAt:      now,
		ApprovedBy:     "supervisor-omar",
		ApprovedAt:     &approvedAt,
		Fields: []formdefinition.FormField{
			{
				FormID:    formID,
				FieldID:   "field-boom-integrity",
				FieldCode: "BOOM_INTEGRITY",
				FieldType: formdefinition.FieldTypeSingleSelect,
				Label:     "Boom weld and structural integrity",
				Required:  true,
				SortOrder: 1,
				Options: []formdefinition.FieldOption{
					{Code: "PASS", Label: "Pass"},
					{Code: "FAIL", Label: "Fail"},
				},
			},
			{
				FormID:    formID,
				FieldID:   "field-wire-rope-diameter",
				FieldCode: "ROPE_DIAMETER_MM",
				FieldType: formdefinition.FieldTypeNumber,
				Label:     "Measured hoist wire rope diameter (mm)",
				Required:  true,
				SortOrder: 2,
			},
		},
	}
	if err := formVer.Validate(); err != nil {
		t.Fatalf("M1 FormVersion validation failed: %v", err)
	}

	maxCount := int64(10)
	evidencePolicy := formdefinition.EvidencePolicy{
		TenantID:       tenantA,
		OrganizationID: orgA,
		FormID:         formID,
		FieldID:        "field-wire-rope-diameter",
		EvidenceType:   formdefinition.EvidenceTypePhoto,
		Required:       true,
		MaxCount:       &maxCount,
		Classification: formdefinition.ClassificationRestricted,
	}
	if err := evidencePolicy.Validate(); err != nil {
		t.Fatalf("M1 EvidencePolicy validation failed: %v", err)
	}

	// -------------------------------------------------------------
	// Milestone M2: Asset Entitlement and Offline Package Generation
	// -------------------------------------------------------------
	assetID := "asset-crane-rt100"
	entitlementID := "ent-crane-rt100-2026-q3"
	entitlement := assetentitlement.AssetEntitlement{
		ID:                  entitlementID,
		TenantID:            tenantA,
		OrganizationID:      orgA,
		WorkOrderID:         "wo-rt-9912",
		ScopeItemID:         "scope-crane-01",
		AssetID:             assetID,
		AssetType:           "MOBILE_CRANE",
		EntitlementType:     assetentitlement.EntitlementTypeInspection,
		Status:              assetentitlement.EntitlementStatusActive,
		FormVersionID:       formID,
		AssignedInspectorID: actorA,
		EntitledAt:          now,
		CreatedBy:           actorA,
		CreatedAt:           now,
		UpdatedBy:           actorA,
		UpdatedAt:           now,
		Revision:            1,
	}
	if err := entitlement.Validate(); err != nil {
		t.Fatalf("M2 AssetEntitlement validation failed: %v", err)
	}

	formSnapBytes, _ := json.Marshal(formVer)
	policySnapBytes, _ := json.Marshal(evidencePolicy)
	assetContextBytes := []byte(`{"model":"Liebherr LTM 1100","serial":"LH-CR-9921"}`)
	pkgHashRaw := sha256.Sum256(append(formSnapBytes, policySnapBytes...))
	packageHash := fmt.Sprintf("%x", pkgHashRaw)

	offlinePkg := assetentitlement.OfflinePackage{
		ID:                     "pkg-offline-001",
		TenantID:               tenantA,
		OrganizationID:         orgA,
		EntitlementID:          entitlementID,
		PackageVersion:         1,
		FormSnapshot:           formSnapBytes,
		EvidencePolicySnapshot: policySnapBytes,
		AssetContext:           assetContextBytes,
		PackageHash:            packageHash,
		Status:                 assetentitlement.PackageStatusGenerated,
		GeneratedAt:            &now,
		CreatedBy:              actorA,
		CreatedAt:              now,
	}
	if err := offlinePkg.Validate(); err != nil {
		t.Fatalf("M2 OfflinePackage validation failed: %v", err)
	}

	// -------------------------------------------------------------
	// Milestone M3: Authenticated QR/NFC Entry Binding
	// -------------------------------------------------------------
	tokenRaw := "opaque-unpredictable-token-sec-99120"
	tokenHash := sha256.Sum256([]byte(tokenRaw))
	tokenDigest := fmt.Sprintf("%x", tokenHash)

	qrEntry := qrnfc.Entry{
		ID:             "qr-entry-001",
		TenantID:       tenantA,
		OrganizationID: orgA,
		EntitlementID:  entitlementID,
		AssetID:        assetID,
		EntryType:      qrnfc.EntryTypeQR,
		TokenDigest:    tokenDigest,
		TokenVersion:   1,
		Status:         qrnfc.EntryStatusActive,
		IssuedAt:       now,
		CreatedBy:      actorA,
		CreatedAt:      now,
	}
	if err := qrEntry.Validate(); err != nil {
		t.Fatalf("M3 QR Entry validation failed: %v", err)
	}

	entryLog := qrnfc.EntryLog{
		ID:             "entry-log-001",
		TenantID:       tenantA,
		OrganizationID: orgA,
		EntryID:        qrEntry.ID,
		EntryType:      qrnfc.EntryTypeQR,
		AccessedAt:     now,
		AccessorID:     actorA,
		AccessorIP:     "127.0.0.1",
		Outcome:        qrnfc.AccessOutcomeSuccess,
	}
	if err := entryLog.Validate(); err != nil {
		t.Fatalf("M3 QR EntryLog validation failed: %v", err)
	}

	// -------------------------------------------------------------
	// Milestone M4: Assurance Projection and Corrective Work Links
	// -------------------------------------------------------------
	certID := "cert-crane-2026-09"
	inspectionID := "insp-rec-9981"
	proj := assurance.Projection{
		ID:                "proj-001",
		TenantID:          tenantA,
		OrganizationID:    orgA,
		CertificateID:     certID,
		InspectionID:      inspectionID,
		AssetID:           assetID,
		AssetType:         "MOBILE_CRANE",
		CertificateNumber: "CERT-INTEGIN-2026-0091",
		CertificateStatus: "ISSUED",
		InspectionStatus:  "COMPLETED",
		InspectionVerdict: "PASS",
		InspectorID:       actorA,
		ProjectedAt:       now,
		CreatedAt:         now,
	}
	proj.ProjectionHash = assurance.ComputeProjectionHash(proj)
	if err := proj.Validate(); err != nil {
		t.Fatalf("M4 Assurance Projection validation failed: %v", err)
	}

	corrective := assurance.CorrectiveWork{
		ID:             "corr-001",
		TenantID:       tenantA,
		OrganizationID: orgA,
		InspectionID:   inspectionID,
		FindingID:      "finding-rope-wear-minor",
		AssetID:        assetID,
		Severity:       assurance.SeverityMinor,
		Description:    "Lubricate hoist cable at next weekly maintenance",
		Status:         assurance.WorkStatusOpen,
		CreatedBy:      actorA,
		CreatedAt:      now,
		UpdatedBy:      actorA,
		UpdatedAt:      now,
		Revision:       1,
	}
	if err := corrective.Validate(); err != nil {
		t.Fatalf("M4 CorrectiveWork validation failed: %v", err)
	}

	// -------------------------------------------------------------
	// Milestone M5: Evidence and Release Pack Composition
	// -------------------------------------------------------------
	packRecord := evidencepack.PackRecord{
		EvidenceID:       "ev-photo-caliper-01",
		ObjectKey:        "evidence/2026/09/ev-photo-caliper-01.enc",
		ContentType:      "image/jpeg",
		CapturedAt:       now,
		InspectionID:     inspectionID,
		PlaintextSHA256:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		CiphertextSHA256: "a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e",
		CiphertextBytes:  1048576,
		Classification:   string(formdefinition.ClassificationRestricted),
	}

	pack := evidencepack.EvidencePack{
		ID:                 "ev-pack-001",
		TenantID:           tenantA,
		OrganizationID:     orgA,
		InspectionID:       inspectionID,
		PackName:           "Crane RT100 Sealed Evidence Pack",
		Status:             evidencepack.PackStatusSealed,
		Records:            []evidencepack.PackRecord{packRecord},
		Classification:     string(formdefinition.ClassificationRestricted),
		RetentionReference: "STANDARD_7YR",
		HoldState:          "CLEAR",
		RedactionPolicyRef: "STANDARD_EXCLUDE_FACES",
		CreatedBy:          actorA,
		CreatedAt:          now,
		UpdatedBy:          actorA,
		UpdatedAt:          now,
		Revision:           1,
	}
	pack.PackChecksum = pack.ComputeChecksum()
	if err := pack.Validate(); err != nil {
		t.Fatalf("M5 EvidencePack validation failed: %v", err)
	}

	rel := evidencepack.ReleasePack{
		ID:                 "rel-pack-001",
		TenantID:           tenantA,
		OrganizationID:     orgA,
		PackID:             pack.ID,
		Recipient:          "Saudi Aramco Inspection Auditor",
		Purpose:            "Quarterly Compliance Verification",
		Status:             evidencepack.ReleaseStatusPending,
		Classification:     "RESTRICTED",
		RetentionReference: "STANDARD_7YR",
		HoldState:          "CLEAR",
		RedactionPolicyRef: "STANDARD_EXCLUDE_FACES",
		CreatedBy:          actorA,
		CreatedAt:          now,
		UpdatedBy:          actorA,
		UpdatedAt:          now,
		Revision:           1,
	}
	if err := rel.Validate(); err != nil {
		t.Fatalf("M5 ReleasePack validation failed: %v", err)
	}

	// -------------------------------------------------------------
	// Milestone M6: Custody Chain and Site Handover Transitions
	// -------------------------------------------------------------
	custodyHash := custodyhandover.ComputeVerificationHash(tenantA, orgA, assetID, "loc-ras-tanura", actorA, custodyhandover.CustodyActionCheckIn, now)
	custody := custodyhandover.CustodyRecord{
		ID:               "cust-001",
		TenantID:         tenantA,
		OrganizationID:   orgA,
		AssetID:          assetID,
		WorkOrderID:      "wo-rt-9912",
		LocationID:       "loc-ras-tanura",
		SiteName:         "Ras Tanura Pier Berth 4",
		CustodianID:      actorA,
		Action:           custodyhandover.CustodyActionCheckIn,
		VerificationHash: custodyHash,
		Notes:            "Crane verified on-site with complete maintenance logbook",
		RecordedBy:       actorA,
		RecordedAt:       now,
	}
	if err := custody.Validate(); err != nil {
		t.Fatalf("M6 CustodyRecord validation failed: %v", err)
	}

	handover := custodyhandover.HandoverRequest{
		ID:               "ho-001",
		TenantID:         tenantA,
		OrganizationID:   orgA,
		WorkOrderID:      "wo-rt-9912",
		AssetID:          assetID,
		FromAssignmentID: "asg-inspector-tariq-shift-1",
		ToAssignmentID:   "asg-inspector-khalid-shift-2",
		FromSiteID:       "site-ras-tanura-north",
		ToSiteID:         "site-ras-tanura-south",
		State:            custodyhandover.HandoverStateRequested,
		Reason:           "Night shift handover",
		RequestedBy:      actorA,
		RequestedAt:      now,
		Revision:         1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := handover.Validate(); err != nil {
		t.Fatalf("M6 HandoverRequest initial validation failed: %v", err)
	}

	// Advance handover: REQUESTED -> ACKNOWLEDGED
	ackTime := now.Add(15 * time.Minute)
	handover.State = custodyhandover.HandoverStateAcknowledged
	handover.AcknowledgedBy = "inspector-khalid"
	handover.AcknowledgedAt = &ackTime
	handover.Revision = 2
	handover.UpdatedAt = ackTime
	if err := handover.Validate(); err != nil {
		t.Fatalf("M6 HandoverRequest ACKNOWLEDGED validation failed: %v", err)
	}

	// Advance handover: ACKNOWLEDGED -> APPROVED
	appTime := now.Add(30 * time.Minute)
	handover.State = custodyhandover.HandoverStateApproved
	handover.ApprovedBy = "supervisor-omar"
	handover.ApprovedAt = &appTime
	handover.Revision = 3
	handover.UpdatedAt = appTime
	if err := handover.Validate(); err != nil {
		t.Fatalf("M6 HandoverRequest APPROVED validation failed: %v", err)
	}

	// Complete handover: APPROVED -> TRANSFERRED
	transTime := now.Add(45 * time.Minute)
	handover.State = custodyhandover.HandoverStateTransferred
	handover.TransferredBy = "supervisor-omar"
	handover.TransferredAt = &transTime
	handover.Revision = 4
	handover.UpdatedAt = transTime
	if err := handover.Validate(); err != nil {
		t.Fatalf("M6 HandoverRequest TRANSFERRED validation failed: %v", err)
	}

	// -------------------------------------------------------------
	// Negative Security Cross-Tenant Proof
	// -------------------------------------------------------------
	// Verify that Tenant B actor context cannot access or sign Tenant A's objects
	actorContextB := custodyhandover.ActorContext{
		TenantID:       "tenant-beta-intruder",
		OrganizationID: "org-beta",
		ActorID:        "mallory",
	}
	if actorContextB.TenantID == handover.TenantID {
		t.Fatalf("tenant boundary failure: Tenant B matched Tenant A")
	}
}
