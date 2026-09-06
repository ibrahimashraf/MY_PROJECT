package dpphttp

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/domain/dpp"
)

type mockDPPRepo struct {
	passports map[string]dpp.ProductPassportDPP
}

func newMockDPPRepo() *mockDPPRepo {
	return &mockDPPRepo{passports: make(map[string]dpp.ProductPassportDPP)}
}

func (m *mockDPPRepo) CreateProductPassportDPP(ctx context.Context, actor dpp.ActorContext, d dpp.ProductPassportDPP) (dpp.ProductPassportDPP, error) {
	m.passports[d.ID] = d
	return d, nil
}

func (m *mockDPPRepo) GetProductPassportDPP(ctx context.Context, actor dpp.ActorContext, id string) (dpp.ProductPassportDPP, error) {
	p, ok := m.passports[id]
	if !ok {
		return dpp.ProductPassportDPP{}, dpp.ErrInvalidActor
	}
	return p, nil
}

func (m *mockDPPRepo) GetProductPassportDPPByAsset(ctx context.Context, actor dpp.ActorContext, assetID string) (dpp.ProductPassportDPP, bool, error) {
	for _, p := range m.passports {
		if p.AssetID == assetID {
			return p, true, nil
		}
	}
	return dpp.ProductPassportDPP{}, false, nil
}

func (m *mockDPPRepo) UpdateProductPassportDPP(ctx context.Context, actor dpp.ActorContext, d dpp.ProductPassportDPP) error {
	m.passports[d.ID] = d
	return nil
}

func (m *mockDPPRepo) CreateRegulatoryMonitor(ctx context.Context, actor dpp.ActorContext, r dpp.RegulatoryMonitor) (dpp.RegulatoryMonitor, error) {
	return r, nil
}
func (m *mockDPPRepo) GetRegulatoryMonitor(ctx context.Context, actor dpp.ActorContext, id string) (dpp.RegulatoryMonitor, error) {
	return dpp.RegulatoryMonitor{}, nil
}
func (m *mockDPPRepo) ListRegulatoryMonitors(ctx context.Context, actor dpp.ActorContext, source, status string) ([]dpp.RegulatoryMonitor, error) {
	return nil, nil
}
func (m *mockDPPRepo) UpdateRegulatoryMonitor(ctx context.Context, actor dpp.ActorContext, r dpp.RegulatoryMonitor) error {
	return nil
}
func (m *mockDPPRepo) CreateComplianceAction(ctx context.Context, actor dpp.ActorContext, a dpp.ComplianceAction) (dpp.ComplianceAction, error) {
	return a, nil
}
func (m *mockDPPRepo) GetComplianceAction(ctx context.Context, actor dpp.ActorContext, id string) (dpp.ComplianceAction, error) {
	return dpp.ComplianceAction{}, nil
}
func (m *mockDPPRepo) ListComplianceActions(ctx context.Context, actor dpp.ActorContext, monitorID, status string) ([]dpp.ComplianceAction, error) {
	return nil, nil
}
func (m *mockDPPRepo) UpdateComplianceAction(ctx context.Context, actor dpp.ActorContext, a dpp.ComplianceAction) error {
	return nil
}

func TestDPPHTTPHandler(t *testing.T) {
	repo := newMockDPPRepo()
	resolver := func(r *http.Request) (dpp.ActorContext, error) {
		return dpp.ActorContext{
			TenantID:       "tenant-1",
			OrganizationID: "org-1",
			ActorID:        "user-1",
		}, nil
	}

	handler := NewHandler(repo, nil, resolver)

	// 1. POST /dpp/passports (Pillar 1 Assignment)
	reqBody := `{"id":"dpp-100","asset_id":"crane-100","serial_number":"SN-100","dpp_status":"ASSIGNMENT","assignment_payload":"{\"mfr\":\"Liebherr\"}"}`
	req := httptest.NewRequest(http.MethodPost, "/dpp/passports", bytes.NewBufferString(reqBody))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. GET /dpp/passports/dpp-100
	reqGet := httptest.NewRequest(http.MethodGet, "/dpp/passports/dpp-100", nil)
	recGet := httptest.NewRecorder()
	handler.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", recGet.Code, recGet.Body.String())
	}

	// 3. PUT /dpp/passports/dpp-100 (Pillar 2 Update)
	updateBody := `{"dpp_status":"UPDATING","update_payload":"{\"event\":\"rope_replacement\"}"}`
	reqPut := httptest.NewRequest(http.MethodPut, "/dpp/passports/dpp-100", bytes.NewBufferString(updateBody))
	recPut := httptest.NewRecorder()
	handler.ServeHTTP(recPut, reqPut)

	if recPut.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", recPut.Code, recPut.Body.String())
	}
}
