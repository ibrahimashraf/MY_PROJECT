package evidencepack

import (
	"testing"
	"time"

	"integin/internal/domain/formdefinition"
)

func basePack(name string) EvidencePack {
	now := time.Now()
	return EvidencePack{
		ID:                 "pack_001",
		TenantID:           "tenant_1",
		OrganizationID:     "org_1",
		InspectionID:       "insp_001",
		PackName:           name,
		Status:             PackStatusDraft,
		Classification:     string(formdefinition.ClassificationConfidential),
		RetentionReference: "ret_1",
		HoldState:          "NONE",
		RedactionPolicyRef: "redact_1",
		CreatedBy:          "user_1",
		CreatedAt:          now,
		UpdatedBy:          "user_1",
		UpdatedAt:          now,
		Revision:           1,
	}
}

func TestEvidencePackValidate(t *testing.T) {
	tests := []struct {
		name    string
		pack    EvidencePack
		wantErr bool
	}{
		{"valid draft", basePack("p1"), false},
		{"missing id", func() EvidencePack { p := basePack("p1"); p.ID = ""; return p }(), true},
		{"missing pack_name", basePack(""), true},
		{"unsupported status", func() EvidencePack { p := basePack("p1"); p.Status = "BOGUS"; return p }(), true},
		{"zero revision", func() EvidencePack { p := basePack("p1"); p.Revision = 0; return p }(), true},
		{"bad classification", func() EvidencePack { p := basePack("p1"); p.Classification = "NOPE"; return p }(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pack.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEvidencePackChecksumDeterministic(t *testing.T) {
	now := time.Now()
	p1 := basePack("pk")
	p1.AddRecord(PackRecord{EvidenceID: "ev_b", ObjectKey: "k2", InspectionID: "insp_001", CapturedAt: now, Classification: "CONFIDENTIAL"})
	p1.AddRecord(PackRecord{EvidenceID: "ev_a", ObjectKey: "k1", InspectionID: "insp_001", CapturedAt: now, Classification: "CONFIDENTIAL"})

	p2 := basePack("pk")
	p2.AddRecord(PackRecord{EvidenceID: "ev_a", ObjectKey: "k1", InspectionID: "insp_001", CapturedAt: now, Classification: "CONFIDENTIAL"})
	p2.AddRecord(PackRecord{EvidenceID: "ev_b", ObjectKey: "k2", InspectionID: "insp_001", CapturedAt: now, Classification: "CONFIDENTIAL"})

	if p1.ComputeChecksum() != p2.ComputeChecksum() {
		t.Error("checksum should be order-independent")
	}
}

func TestEvidencePackSealEnforcesPrivacy(t *testing.T) {
	now := time.Now()
	p := basePack("pk")
	p.AddRecord(PackRecord{EvidenceID: "ev_a", ObjectKey: "k1", InspectionID: "insp_001", CapturedAt: now, Classification: "CONFIDENTIAL"})
	p.AddRecord(PackRecord{EvidenceID: "ev_b", ObjectKey: "k2", InspectionID: "insp_001", CapturedAt: now, Classification: "RESTRICTED"})
	if err := p.Seal(now); err != ErrMixedPrivacy {
		t.Errorf("expected ErrMixedPrivacy, got %v", err)
	}
}

func TestEvidencePackSealSuccess(t *testing.T) {
	now := time.Now()
	p := basePack("pk")
	p.AddRecord(PackRecord{EvidenceID: "ev_a", ObjectKey: "k1", InspectionID: "insp_001", CapturedAt: now, Classification: "CONFIDENTIAL"})
	if err := p.Seal(now); err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	if p.Status != PackStatusSealed {
		t.Errorf("expected SEALED, got %s", p.Status)
	}
	if p.PackChecksum == "" {
		t.Error("expected non-empty checksum after seal")
	}
	if !p.VerifyChecksum() {
		t.Error("VerifyChecksum() should pass after seal")
	}
}

func TestEvidencePackAddRecordAfterSeal(t *testing.T) {
	now := time.Now()
	p := basePack("pk")
	p.AddRecord(PackRecord{EvidenceID: "ev_a", ObjectKey: "k1", InspectionID: "insp_001", CapturedAt: now, Classification: "CONFIDENTIAL"})
	if err := p.Seal(now); err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	err := p.AddRecord(PackRecord{EvidenceID: "ev_c", ObjectKey: "k3", InspectionID: "insp_001", CapturedAt: now, Classification: "CONFIDENTIAL"})
	if err != ErrSealConflict {
		t.Errorf("expected ErrSealConflict, got %v", err)
	}
}

func TestEvidencePackDuplicateRecord(t *testing.T) {
	now := time.Now()
	p := basePack("pk")
	rec := PackRecord{EvidenceID: "ev_a", ObjectKey: "k1", InspectionID: "insp_001", CapturedAt: now, Classification: "CONFIDENTIAL"}
	if err := p.AddRecord(rec); err != nil {
		t.Fatalf("first AddRecord error = %v", err)
	}
	if err := p.AddRecord(rec); err == nil {
		t.Error("expected error on duplicate evidence record")
	}
}

func TestEvidencePackCanTransitionTo(t *testing.T) {
	tests := []struct {
		name    string
		pack    EvidencePack
		target  PackStatus
		wantErr bool
	}{
		{"DRAFT to SEALED", EvidencePack{Status: PackStatusDraft}, PackStatusSealed, false},
		{"DRAFT to SUPERSEDED", EvidencePack{Status: PackStatusDraft}, PackStatusSuperseded, false},
		{"SEALED to RELEASED", EvidencePack{Status: PackStatusSealed}, PackStatusReleased, false},
		{"SEALED to SUPERSEDED", EvidencePack{Status: PackStatusSealed}, PackStatusSuperseded, false},
		{"DRAFT to RELEASED", EvidencePack{Status: PackStatusDraft}, PackStatusReleased, true},
		{"RELEASED to something", EvidencePack{Status: PackStatusReleased}, PackStatusSealed, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pack.CanTransitionTo(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanTransitionTo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func baseRelease(packID string) ReleasePack {
	now := time.Now()
	return ReleasePack{
		ID:                 "rel_001",
		TenantID:           "tenant_1",
		OrganizationID:     "org_1",
		PackID:             packID,
		Recipient:          "recipient_a",
		Purpose:            "third-party regulatory",
		Status:             ReleaseStatusPending,
		Classification:     string(formdefinition.ClassificationConfidential),
		RetentionReference: "ret_1",
		HoldState:          "NONE",
		RedactionPolicyRef: "redact_1",
		CreatedBy:          "user_1",
		CreatedAt:          now,
		UpdatedBy:          "user_1",
		UpdatedAt:          now,
		Revision:           1,
	}
}

func TestReleasePackValidate(t *testing.T) {
	tests := []struct {
		name    string
		release ReleasePack
		wantErr bool
	}{
		{"valid pending", baseRelease("pack_x"), false},
		{"missing recipient", func() ReleasePack { r := baseRelease("pack_x"); r.Recipient = ""; return r }(), true},
		{"unsupported status", func() ReleasePack { r := baseRelease("pack_x"); r.Status = "X"; return r }(), true},
		{"approved requires metadata", func() ReleasePack { r := baseRelease("pack_x"); r.Status = ReleaseStatusApproved; return r }(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.release.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReleasePackCanTransitionTo(t *testing.T) {
	tests := []struct {
		name    string
		release ReleasePack
		target  ReleaseStatus
		wantErr bool
	}{
		{"PENDING to APPROVED", ReleasePack{Status: ReleaseStatusPending}, ReleaseStatusApproved, false},
		{"APPROVED to RELEASED", ReleasePack{Status: ReleaseStatusApproved}, ReleaseStatusReleased, false},
		{"PENDING to RELEASED", ReleasePack{Status: ReleaseStatusPending}, ReleaseStatusReleased, true},
		{"RELEASED to PENDING", ReleasePack{Status: ReleaseStatusReleased}, ReleaseStatusPending, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.release.CanTransitionTo(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanTransitionTo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
