package assurancehttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/domain/assurance"
)

type mockProjectionRepo struct {
	getFunc func(ctx context.Context, actor assurance.ActorContext, certID string) (assurance.Projection, error)
}

func (m *mockProjectionRepo) Get(ctx context.Context, actor assurance.ActorContext, certID string) (assurance.Projection, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, actor, certID)
	}
	return assurance.Projection{
		ID:                "proj-1",
		CertificateID:     certID,
		InspectionID:      "insp-1",
		AssetID:           "asset-1",
		CertificateStatus: "VALID",
	}, nil
}

func (m *mockProjectionRepo) GetByInspection(ctx context.Context, actor assurance.ActorContext, inspID string) (assurance.Projection, error) {
	return assurance.Projection{InspectionID: inspID}, nil
}

func (m *mockProjectionRepo) ListByAsset(ctx context.Context, actor assurance.ActorContext, assetID string) ([]assurance.Projection, error) {
	return []assurance.Projection{{AssetID: assetID}}, nil
}

func (m *mockProjectionRepo) ListByStatus(ctx context.Context, actor assurance.ActorContext, status string) ([]assurance.Projection, error) {
	return []assurance.Projection{{CertificateStatus: status}}, nil
}

type mockWorkRepo struct {
	createFunc func(ctx context.Context, actor assurance.ActorContext, work assurance.CorrectiveWork) (assurance.CorrectiveWork, error)
	updateFunc func(ctx context.Context, actor assurance.ActorContext, workID string, targetStatus assurance.WorkStatus, rev int64) (assurance.CorrectiveWork, error)
}

func (m *mockWorkRepo) Create(ctx context.Context, actor assurance.ActorContext, work assurance.CorrectiveWork) (assurance.CorrectiveWork, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, actor, work)
	}
	work.ID = "work-1"
	work.Status = assurance.WorkStatusOpen
	work.Revision = 1
	return work, nil
}

func (m *mockWorkRepo) Get(ctx context.Context, actor assurance.ActorContext, workID string) (assurance.CorrectiveWork, error) {
	return assurance.CorrectiveWork{ID: workID, Status: assurance.WorkStatusOpen, Revision: 1}, nil
}

func (m *mockWorkRepo) ListByInspection(ctx context.Context, actor assurance.ActorContext, inspID string) ([]assurance.CorrectiveWork, error) {
	return []assurance.CorrectiveWork{{InspectionID: inspID}}, nil
}

func (m *mockWorkRepo) ListByAsset(ctx context.Context, actor assurance.ActorContext, assetID string) ([]assurance.CorrectiveWork, error) {
	return []assurance.CorrectiveWork{{AssetID: assetID}}, nil
}

func (m *mockWorkRepo) UpdateStatus(ctx context.Context, actor assurance.ActorContext, workID string, targetStatus assurance.WorkStatus, rev int64) (assurance.CorrectiveWork, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, actor, workID, targetStatus, rev)
	}
	return assurance.CorrectiveWork{ID: workID, Status: targetStatus, Revision: rev + 1}, nil
}

func (m *mockWorkRepo) Assign(ctx context.Context, actor assurance.ActorContext, workID string, inspectorID string, rev int64) (assurance.CorrectiveWork, error) {
	return assurance.CorrectiveWork{ID: workID, AssignedTo: inspectorID, Revision: rev + 1}, nil
}

func TestAssuranceHTTP_ProjectionAndWorkLifecycle(t *testing.T) {
	h := NewHandler(nil, nil, &mockProjectionRepo{}, &mockWorkRepo{}, nil)

	// 1. Get Projection
	req := httptest.NewRequest(http.MethodGet, "/api/v1/assurance/projections/by-certificate/cert-123", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Organization-ID", "org-1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var p assurance.Projection
	if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if p.CertificateID != "cert-123" {
		t.Errorf("expected cert-123, got %s", p.CertificateID)
	}

	// 2. Create Corrective Work
	body, _ := json.Marshal(assurance.CorrectiveWork{
		InspectionID: "insp-1",
		FindingID:    "find-1",
		AssetID:      "asset-1",
		Severity:     assurance.SeverityCritical,
		Description:  "Crack on lifting hook",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/assurance/corrective-work", bytes.NewReader(body))
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Organization-ID", "org-1")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}

	// 3. Update Status
	statusBody, _ := json.Marshal(updateStatusRequest{
		TargetStatus:     assurance.WorkStatusInProgress,
		ExpectedRevision: 1,
	})
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/assurance/corrective-work/work-1/status", bytes.NewReader(statusBody))
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Organization-ID", "org-1")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var updated assurance.CorrectiveWork
	if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status != assurance.WorkStatusInProgress {
		t.Errorf("expected IN_PROGRESS, got %s", updated.Status)
	}
}
