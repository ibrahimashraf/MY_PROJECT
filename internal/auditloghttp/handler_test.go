package auditloghttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/auditlog"
)

type mockRepository struct {
	entries []auditlog.Entry
}

func (m *mockRepository) Append(ctx context.Context, entry auditlog.Entry) (auditlog.Entry, error) {
	if entry.PreviousHash == "" {
		entry.PreviousHash = auditlog.GenesisHash()
	}
	entryData := entry.EventType + string(entry.EntityType) + entry.EntityID + entry.ActorID + string(entry.Action) + entry.CreatedAt.Format(time.RFC3339Nano)
	entry.EntryHash = auditlog.ComputeHash(entry.PreviousHash, entryData)
	entry.CreatedAt = time.Now().UTC()
	m.entries = append(m.entries, entry)
	return entry, nil
}

func (m *mockRepository) Query(ctx context.Context, req auditlog.QueryRequest) (auditlog.QueryResponse, error) {
	var filtered []auditlog.Entry
	for _, e := range m.entries {
		if req.EntityType != "" && e.EntityType != req.EntityType {
			continue
		}
		if req.EntityID != "" && e.EntityID != req.EntityID {
			continue
		}
		filtered = append(filtered, e)
	}
	return auditlog.QueryResponse{Entries: filtered, Total: len(filtered)}, nil
}

func (m *mockRepository) VerifyChain(ctx context.Context, tenantID, organizationID string, from, to *time.Time) (auditlog.VerifyResult, error) {
	return auditlog.VerifyResult{Valid: true}, nil
}

func TestHandleAppend(t *testing.T) {
	repo := &mockRepository{}
	h := Handler{AuditLogRepository: repo}

	body := `{"tenant_id":"t1","organization_id":"o1","event_type":"inspection_created","entity_type":"inspection","entity_id":"insp-1","actor_id":"a1","actor_name":"Inspector A","action":"create"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit-log", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	if len(repo.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(repo.entries))
	}
}

func TestHandleAppendInvalidBody(t *testing.T) {
	repo := &mockRepository{}
	h := Handler{AuditLogRepository: repo}

	body := `invalid`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit-log", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleAppendMissingTenant(t *testing.T) {
	repo := &mockRepository{}
	h := Handler{AuditLogRepository: repo}

	body := `{"event_type":"x","entity_type":"inspection","entity_id":"e","actor_id":"a","action":"create"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit-log", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleQuery(t *testing.T) {
	repo := &mockRepository{}
	h := Handler{AuditLogRepository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-log", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleVerify(t *testing.T) {
	repo := &mockRepository{}
	h := Handler{AuditLogRepository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-log/verify", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleNotFound(t *testing.T) {
	repo := &mockRepository{}
	h := Handler{AuditLogRepository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
