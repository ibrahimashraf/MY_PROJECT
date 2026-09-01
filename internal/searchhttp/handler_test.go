package searchhttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/domain/search"
)

type mockSearchRepo struct {
	results []search.Result
	total   int
}

func (m *mockSearchRepo) Search(ctx context.Context, req search.SearchRequest) (search.SearchResponse, error) {
	return search.SearchResponse{
		Results: m.results,
		Total:   m.total,
		Query:   req.Query,
	}, nil
}

func TestHandleSearchMissingQuery(t *testing.T) {
	repo := &mockSearchRepo{}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleSearchMissingHeaders(t *testing.T) {
	repo := &mockSearchRepo{}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/?q=test", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleSearchSuccess(t *testing.T) {
	repo := &mockSearchRepo{results: []search.Result{{ID: "1", Title: "Test"}}}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/?q=test", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleSearchMethodNotAllowed(t *testing.T) {
	repo := &mockSearchRepo{}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/?q=test", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleSearchInvalidEntityType(t *testing.T) {
	repo := &mockSearchRepo{}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/?q=test&types=INVALID", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleSearchValidEntityTypes(t *testing.T) {
	repo := &mockSearchRepo{results: []search.Result{{ID: "1", Title: "Test"}}}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/?q=test&types=ASSET,WORK_ORDER", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
