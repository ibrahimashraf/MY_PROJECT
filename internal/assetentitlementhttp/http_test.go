package assetentitlementhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/domain/assetentitlement"
)

type mockEntitlementRepo struct {
	registerFunc func(ctx context.Context, actor assetentitlement.ActorContext, ent assetentitlement.AssetEntitlement) (assetentitlement.AssetEntitlement, error)
	getFunc      func(ctx context.Context, actor assetentitlement.ActorContext, id string) (assetentitlement.AssetEntitlement, error)
}

func (m *mockEntitlementRepo) Register(ctx context.Context, actor assetentitlement.ActorContext, ent assetentitlement.AssetEntitlement) (assetentitlement.AssetEntitlement, error) {
	if m.registerFunc != nil {
		return m.registerFunc(ctx, actor, ent)
	}
	ent.ID = "ent-1"
	ent.Status = assetentitlement.EntitlementStatusActive
	ent.Revision = 1
	return ent, nil
}

func (m *mockEntitlementRepo) Get(ctx context.Context, actor assetentitlement.ActorContext, id string) (assetentitlement.AssetEntitlement, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, actor, id)
	}
	return assetentitlement.AssetEntitlement{ID: id, Status: assetentitlement.EntitlementStatusActive}, nil
}

func (m *mockEntitlementRepo) GetByWorkOrder(ctx context.Context, actor assetentitlement.ActorContext, woID string) ([]assetentitlement.AssetEntitlement, error) {
	return []assetentitlement.AssetEntitlement{{ID: "ent-1", WorkOrderID: woID}}, nil
}

func (m *mockEntitlementRepo) GetByAsset(ctx context.Context, actor assetentitlement.ActorContext, assetID string) ([]assetentitlement.AssetEntitlement, error) {
	return []assetentitlement.AssetEntitlement{{ID: "ent-1", AssetID: assetID}}, nil
}

func (m *mockEntitlementRepo) Complete(ctx context.Context, actor assetentitlement.ActorContext, id string, expRev int64) (assetentitlement.AssetEntitlement, error) {
	return assetentitlement.AssetEntitlement{ID: id, Status: assetentitlement.EntitlementStatusCompleted, Revision: expRev + 1}, nil
}

func (m *mockEntitlementRepo) Cancel(ctx context.Context, actor assetentitlement.ActorContext, id string, expRev int64) (assetentitlement.AssetEntitlement, error) {
	return assetentitlement.AssetEntitlement{ID: id, Status: assetentitlement.EntitlementStatusCancelled, Revision: expRev + 1}, nil
}

func (m *mockEntitlementRepo) AssignInspector(ctx context.Context, actor assetentitlement.ActorContext, id string, inspID string, expRev int64) (assetentitlement.AssetEntitlement, error) {
	return assetentitlement.AssetEntitlement{ID: id, AssignedInspectorID: inspID, Revision: expRev + 1}, nil
}

func (m *mockEntitlementRepo) AddTag(ctx context.Context, actor assetentitlement.ActorContext, tag assetentitlement.AssetTag) (assetentitlement.AssetTag, error) {
	return tag, nil
}

func (m *mockEntitlementRepo) ListTags(ctx context.Context, actor assetentitlement.ActorContext, assetID string) ([]assetentitlement.AssetTag, error) {
	return []assetentitlement.AssetTag{{AssetID: assetID, TagType: assetentitlement.TagTypeQR, TagValue: "tag-1"}}, nil
}

func (m *mockEntitlementRepo) RemoveTag(ctx context.Context, actor assetentitlement.ActorContext, assetID string, tagType assetentitlement.TagType, tagValue string) error {
	return nil
}

func TestAssetEntitlementHTTP_Lifecycle(t *testing.T) {
	h := NewHandler(nil, nil, &mockEntitlementRepo{}, nil, nil)

	// 1. Register Entitlement
	body, _ := json.Marshal(assetentitlement.AssetEntitlement{
		WorkOrderID:     "wo-1",
		ScopeItemID:     "scope-1",
		AssetID:         "crane-100",
		AssetType:       "CRANE",
		EntitlementType: assetentitlement.EntitlementTypeInspection,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/asset-entitlements", bytes.NewReader(body))
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Organization-ID", "org-1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}

	// 2. Complete Entitlement
	compBody, _ := json.Marshal(statusRequest{ExpectedRevision: 1})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/asset-entitlements/ent-1/complete", bytes.NewReader(compBody))
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Organization-ID", "org-1")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// 3. List tags
	req = httptest.NewRequest(http.MethodGet, "/api/v1/asset-entitlements/tags/crane-100", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Organization-ID", "org-1")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}
