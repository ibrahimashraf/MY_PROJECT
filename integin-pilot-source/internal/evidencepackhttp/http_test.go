package evidencepackhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/domain/evidencepack"
)

type mockPackRepo struct {
	createFunc func(ctx context.Context, actor evidencepack.ActorContext, pack evidencepack.EvidencePack) (evidencepack.EvidencePack, error)
	updateFunc func(ctx context.Context, actor evidencepack.ActorContext, packID string, target evidencepack.PackStatus, expRev int64) (evidencepack.EvidencePack, error)
	getFunc    func(ctx context.Context, actor evidencepack.ActorContext, packID string) (evidencepack.EvidencePack, error)
}

func (m *mockPackRepo) Create(ctx context.Context, actor evidencepack.ActorContext, pack evidencepack.EvidencePack) (evidencepack.EvidencePack, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, actor, pack)
	}
	pack.ID = "pack-1"
	pack.Status = evidencepack.PackStatusDraft
	pack.Revision = 1
	return pack, nil
}

func (m *mockPackRepo) Get(ctx context.Context, actor evidencepack.ActorContext, packID string) (evidencepack.EvidencePack, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, actor, packID)
	}
	return evidencepack.EvidencePack{ID: packID, Status: evidencepack.PackStatusDraft}, nil
}

func (m *mockPackRepo) GetByName(ctx context.Context, actor evidencepack.ActorContext, packName string) (evidencepack.EvidencePack, error) {
	return evidencepack.EvidencePack{PackName: packName}, nil
}

func (m *mockPackRepo) ListByInspection(ctx context.Context, actor evidencepack.ActorContext, inspectionID string) ([]evidencepack.EvidencePack, error) {
	return []evidencepack.EvidencePack{{ID: "pack-1", InspectionID: inspectionID}}, nil
}

func (m *mockPackRepo) UpdateStatus(ctx context.Context, actor evidencepack.ActorContext, packID string, target evidencepack.PackStatus, expRev int64) (evidencepack.EvidencePack, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, actor, packID, target, expRev)
	}
	return evidencepack.EvidencePack{ID: packID, Status: target, Revision: expRev + 1}, nil
}

type mockReleaseRepo struct {
	createFunc func(ctx context.Context, actor evidencepack.ActorContext, release evidencepack.ReleasePack) (evidencepack.ReleasePack, error)
	updateFunc func(ctx context.Context, actor evidencepack.ActorContext, relID string, target evidencepack.ReleaseStatus, expRev int64) (evidencepack.ReleasePack, error)
}

func (m *mockReleaseRepo) Create(ctx context.Context, actor evidencepack.ActorContext, release evidencepack.ReleasePack) (evidencepack.ReleasePack, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, actor, release)
	}
	release.ID = "release-1"
	release.Status = evidencepack.ReleaseStatusPending
	return release, nil
}

func (m *mockReleaseRepo) Get(ctx context.Context, actor evidencepack.ActorContext, releaseID string) (evidencepack.ReleasePack, error) {
	return evidencepack.ReleasePack{ID: releaseID, Status: evidencepack.ReleaseStatusPending}, nil
}

func (m *mockReleaseRepo) ListByPack(ctx context.Context, actor evidencepack.ActorContext, packID string) ([]evidencepack.ReleasePack, error) {
	return []evidencepack.ReleasePack{{ID: "release-1", PackID: packID}}, nil
}

func (m *mockReleaseRepo) UpdateStatus(ctx context.Context, actor evidencepack.ActorContext, relID string, target evidencepack.ReleaseStatus, expRev int64) (evidencepack.ReleasePack, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, actor, relID, target, expRev)
	}
	return evidencepack.ReleasePack{ID: relID, Status: target, Revision: expRev + 1}, nil
}

func TestEvidencePackHTTP_Lifecycle(t *testing.T) {
	h := NewHandler(nil, nil, &mockPackRepo{}, &mockReleaseRepo{}, nil)

	// 1. Create Pack
	packBody, _ := json.Marshal(evidencepack.EvidencePack{
		InspectionID: "insp-1",
		PackName:     "crane-annual-evidence",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/evidence-packs", bytes.NewReader(packBody))
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Organization-ID", "org-1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}
	var createdPack evidencepack.EvidencePack
	if err := json.NewDecoder(w.Body).Decode(&createdPack); err != nil {
		t.Fatal(err)
	}
	if createdPack.PackName != "crane-annual-evidence" {
		t.Errorf("expected crane-annual-evidence, got %s", createdPack.PackName)
	}

	// 2. Seal Pack
	sealBody, _ := json.Marshal(statusUpdateRequest{ExpectedRevision: 1})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/evidence-packs/pack-1/seal", bytes.NewReader(sealBody))
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Organization-ID", "org-1")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var sealedPack evidencepack.EvidencePack
	if err := json.NewDecoder(w.Body).Decode(&sealedPack); err != nil {
		t.Fatal(err)
	}
	if sealedPack.Status != evidencepack.PackStatusSealed {
		t.Errorf("expected SEALED, got %s", sealedPack.Status)
	}

	// 3. Create Release Record
	relBody, _ := json.Marshal(evidencepack.ReleasePack{
		PackID:    "pack-1",
		Recipient: "auditor@regulator.gov",
		Purpose:   "ANNUAL_AUDIT",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/evidence-packs/releases", bytes.NewReader(relBody))
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Organization-ID", "org-1")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}

	// 4. Approve Release
	appBody, _ := json.Marshal(statusUpdateRequest{ExpectedRevision: 1})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/evidence-packs/releases/release-1/approve", bytes.NewReader(appBody))
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Organization-ID", "org-1")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}
