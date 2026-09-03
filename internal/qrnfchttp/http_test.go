package qrnfchttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"integin/internal/domain/qrnfc"
	"integin/internal/identity"
	"integin/internal/oidcauth"
)

type mockValidator struct {
	principal oidcauth.Principal
	err       error
	mu        sync.Mutex
	tokens    []string
}

func (m *mockValidator) Validate(ctx context.Context, rawToken string) (oidcauth.Principal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens = append(m.tokens, rawToken)
	return m.principal, m.err
}

type mockResolver struct {
	membership identity.Membership
	err        error
}

func (m *mockResolver) Resolve(ctx context.Context, principal identity.PrincipalKey) (identity.Membership, error) {
	return m.membership, m.err
}

type mockVerifier struct {
	entry qrnfc.Entry
	err   error
}

func (m *mockVerifier) VerifyByDigest(ctx context.Context, actor qrnfc.ActorContext, tokenDigest string) (qrnfc.Entry, error) {
	return m.entry, m.err
}

type mockLogger struct {
	mu      sync.Mutex
	records []qrnfc.EntryLog
	err     error
}

func (m *mockLogger) LogAccess(ctx context.Context, actor qrnfc.ActorContext, log qrnfc.EntryLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.records = append(m.records, log)
	return nil
}

func (m *mockLogger) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.records)
}

func loggerNoop() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func defaultHandler(v *mockValidator, r *mockResolver, verifier EntryVerifier, lg *mockLogger) *Handler {
	if v == nil {
		v = &mockValidator{principal: oidcauth.Principal{Issuer: "iss", Subject: "sub"}}
	}
	if r == nil {
		r = &mockResolver{membership: identity.Membership{
			TenantID:       "tenant-1",
			OrganizationID: "org-1",
			ActorID:        "actor-1",
		}}
	}
	return NewHandler(v, r, verifier, lg, loggerNoop())
}

func doRequest(t *testing.T, h *Handler, method, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, "/qr-nfc/verify", reader)
	req.RemoteAddr = "192.168.1.10:1234"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestServeHTTP_Success(t *testing.T) {
	lg := &mockLogger{}
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)
	h := defaultHandler(nil, nil, &mockVerifier{
		entry: qrnfc.Entry{
			ID:          "entry-1",
			AssetID:     "asset-1",
			EntryType:   qrnfc.EntryTypeQR,
			Status:      qrnfc.EntryStatusActive,
			IssuedAt:    now,
			ExpiresAt:   &expires,
			TokenDigest: "abc",
		},
	}, lg)

	w := doRequest(t, h, http.MethodPost, "raw-token", `{"token":"raw-token"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["entry_id"] != "entry-1" {
		t.Fatalf("entry_id = %v, want entry-1", resp["entry_id"])
	}
	if _, ok := resp["token_digest"]; ok {
		t.Fatalf("response must not contain token digest")
	}
	if lg.count() != 1 {
		t.Fatalf("expected 1 access log, got %d", lg.count())
	}
}

func TestServeHTTP_RequiresBearer(t *testing.T) {
	lg := &mockLogger{}
	h := defaultHandler(nil, nil, &mockVerifier{}, lg)
	w := doRequest(t, h, http.MethodPost, "", `{"token":"x"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if lg.count() != 0 {
		t.Fatalf("no actor => no persisted access log, got %d", lg.count())
	}
}

func TestServeHTTP_InvalidToken(t *testing.T) {
	lg := &mockLogger{}
	h := defaultHandler(&mockValidator{err: errors.New("bad token")}, nil, &mockVerifier{}, lg)
	w := doRequest(t, h, http.MethodPost, "bad-token", `{"token":"x"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestServeHTTP_ForbiddenWhenResolverFails(t *testing.T) {
	h := defaultHandler(nil, &mockResolver{err: errors.New("unknown subject")}, &mockVerifier{}, nil)
	w := doRequest(t, h, http.MethodPost, "token", `{"token":"x"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestServeHTTP_NotFound(t *testing.T) {
	lg := &mockLogger{}
	h := defaultHandler(nil, nil, &mockVerifier{err: qrnfc.ErrNotFound}, lg)
	w := doRequest(t, h, http.MethodPost, "token", `{"token":"x"}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestServeHTTP_Expired(t *testing.T) {
	lg := &mockLogger{}
	h := defaultHandler(nil, nil, &mockVerifier{err: qrnfc.ErrExpired}, lg)
	w := doRequest(t, h, http.MethodPost, "token", `{"token":"x"}`)
	if w.Code != http.StatusGone {
		t.Fatalf("status = %d, want 410", w.Code)
	}
}

func TestServeHTTP_MethodNotAllowed(t *testing.T) {
	h := defaultHandler(nil, nil, &mockVerifier{}, nil)
	w := doRequest(t, h, http.MethodGet, "token", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

func TestServeHTTP_MissingTokenInBody(t *testing.T) {
	lg := &mockLogger{}
	h := defaultHandler(nil, nil, &mockVerifier{}, lg)
	w := doRequest(t, h, http.MethodPost, "token", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestExtractBearer(t *testing.T) {
	valid := []struct {
		header, expected string
	}{
		{"Bearer abc", "abc"},
		{"Bearer  abc  ", "abc"},
		{"Bearer", ""},
		{"", ""},
		{"Basic abc", ""},
	}
	for i, tc := range valid {
		token, ok := extractBearer(tc.header)
		if tc.expected == "" && ok {
			t.Fatalf("case %d: expected ok=false, got true (%q)", i, token)
		}
		if tc.expected != "" && (!ok || token != tc.expected) {
			t.Fatalf("case %d: got (%q,%v), want (%q,true)", i, token, ok, tc.expected)
		}
	}
}

func TestComputeDigest(t *testing.T) {
	d1 := computeDigest("  abc  ")
	d2 := computeDigest("abc")
	if d1 != d2 {
		t.Fatalf("digest should trim input: %q != %q", d1, d2)
	}
	if len(d1) != 64 {
		t.Fatalf("sha256 hex digest len = %d, want 64", len(d1))
	}
}
