package formdefinitionhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/domain/formdefinition"
)

type mockRepo struct {
	registerFunc func(ctx context.Context, actor formdefinition.ActorContext, form formdefinition.FormVersion) (formdefinition.FormVersion, bool, error)
	approveFunc  func(ctx context.Context, actor formdefinition.ActorContext, formID string, expVer int) (formdefinition.FormVersion, error)
	retireFunc   func(ctx context.Context, actor formdefinition.ActorContext, formID string, expVer int) (formdefinition.FormVersion, error)
	getFunc      func(ctx context.Context, actor formdefinition.ActorContext, formID string) (formdefinition.FormVersion, error)
}

func (m *mockRepo) RegisterDraft(ctx context.Context, actor formdefinition.ActorContext, form formdefinition.FormVersion) (formdefinition.FormVersion, bool, error) {
	if m.registerFunc != nil {
		return m.registerFunc(ctx, actor, form)
	}
	form.ID = "form-1"
	form.Status = formdefinition.FormStatusDraft
	form.Version = 1
	return form, true, nil
}

func (m *mockRepo) Approve(ctx context.Context, actor formdefinition.ActorContext, formID string, expVer int) (formdefinition.FormVersion, error) {
	if m.approveFunc != nil {
		return m.approveFunc(ctx, actor, formID, expVer)
	}
	return formdefinition.FormVersion{ID: formID, Status: formdefinition.FormStatusApproved, Version: expVer}, nil
}

func (m *mockRepo) Retire(ctx context.Context, actor formdefinition.ActorContext, formID string, expVer int) (formdefinition.FormVersion, error) {
	if m.retireFunc != nil {
		return m.retireFunc(ctx, actor, formID, expVer)
	}
	return formdefinition.FormVersion{ID: formID, Status: formdefinition.FormStatusRetired, Version: expVer}, nil
}

func (m *mockRepo) Get(ctx context.Context, actor formdefinition.ActorContext, formID string) (formdefinition.FormVersion, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, actor, formID)
	}
	return formdefinition.FormVersion{ID: formID, Status: formdefinition.FormStatusDraft}, nil
}

func (m *mockRepo) GetByCode(ctx context.Context, actor formdefinition.ActorContext, formCode string) (formdefinition.FormVersion, error) {
	return formdefinition.FormVersion{FormCode: formCode}, nil
}

func (m *mockRepo) ListByAssetType(ctx context.Context, actor formdefinition.ActorContext, assetType string) ([]formdefinition.FormVersion, error) {
	return []formdefinition.FormVersion{{AssetType: assetType}}, nil
}

func (m *mockRepo) ListByStatus(ctx context.Context, actor formdefinition.ActorContext, status formdefinition.FormStatus) ([]formdefinition.FormVersion, error) {
	return []formdefinition.FormVersion{{Status: status}}, nil
}

func TestFormDefinitionHTTP_Lifecycle(t *testing.T) {
	h := NewHandler(nil, nil, &mockRepo{}, nil)

	// 1. Register Draft
	body, _ := json.Marshal(formdefinition.FormVersion{
		FormCode: "crane_annual_v1",
		Title:    "Crane Annual Inspection",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/form-definitions", bytes.NewReader(body))
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Organization-ID", "org-1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}
	var f formdefinition.FormVersion
	if err := json.NewDecoder(w.Body).Decode(&f); err != nil {
		t.Fatal(err)
	}
	if f.FormCode != "crane_annual_v1" {
		t.Errorf("expected crane_annual_v1, got %s", f.FormCode)
	}

	// 2. Approve
	appBody, _ := json.Marshal(transitionRequest{ExpectedVersion: 1})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/form-definitions/form-1/approve", bytes.NewReader(appBody))
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Organization-ID", "org-1")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var approved formdefinition.FormVersion
	if err := json.NewDecoder(w.Body).Decode(&approved); err != nil {
		t.Fatal(err)
	}
	if approved.Status != formdefinition.FormStatusApproved {
		t.Errorf("expected APPROVED, got %s", approved.Status)
	}
}
