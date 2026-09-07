package idempotency

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// Endpoint identifiers recorded in the cache. New ingress points must be
// added here alongside their migration CHECK constraint.
const (
	EndpointSync     = "/sync"
	EndpointEvidence = "/evidence"
)

// ScopeResolver derives the tenant + organization scope of a request from its
// buffered body. Endpoints that authenticate first (such as /evidence) may
// override this with the authenticated actor's scope.
type ScopeResolver func(r *http.Request, body []byte) (tenantID, organizationID string, err error)

// Middleware enforces the Stripe idempotency contract: a mandatory
// Idempotency-Key header, atomic claim acquisition, and verbatim replay of
// committed responses without downstream execution.
type Middleware struct {
	store    Store
	endpoint string
	resolve  ScopeResolver
}

// NewMiddleware builds an idempotency guard for one endpoint. The scope
// resolver runs against the buffered request body and must tolerate the
// request already having been consumed.
func NewMiddleware(store Store, endpoint string, resolve ScopeResolver) *Middleware {
	return &Middleware{store: store, endpoint: endpoint, resolve: resolve}
}

// Wrap returns an http.Handler that guards next.
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		if key == "" {
			writeError(w, http.StatusBadRequest, `{"error":"idempotency_key_required","reason":"Idempotency-Key header is mandatory"}`)
			return
		}
		if !validIdempotencyKey(key) {
			writeError(w, http.StatusBadRequest, `{"error":"invalid_idempotency_key","reason":"Idempotency-Key must be 8-128 characters and limited to [A-Za-z0-9._-]"}`)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, `{"error":"unreadable_request_body"}`)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		tenantID, organizationID, err := m.resolve(r, body)
		if err != nil {
			writeError(w, http.StatusBadRequest, `{"error":"idempotency_scope_unresolved","reason":`+jsonString(err.Error())+`}`)
			return
		}
		scope := Scope{
			TenantID:       tenantID,
			OrganizationID: organizationID,
			Endpoint:       m.endpoint,
			KeyHash:        HashKey(organizationID, m.endpoint, key),
			RequestHash:    HashRequest(body),
		}

		claim, cached, err := m.store.Acquire(r.Context(), scope)
		switch {
		case errors.Is(err, ErrConflict):
			writeError(w, http.StatusConflict, `{"error":"idempotency_conflict","reason":"idempotency key reused with a different request"}`)
			return
		case err != nil:
			writeError(w, http.StatusServiceUnavailable, `{"error":"idempotency_unavailable","reason":`+jsonString(err.Error())+`}`)
			return
		case cached != nil:
			replay(w, cached)
			return
		}

		recorder := newResponseRecorder()
		defer func() {
			if recovered := recover(); recovered != nil {
				claim.Done()
				panic(recovered)
			}
		}()
		next.ServeHTTP(recorder, r)
		if err := claim.Commit(r.Context(), deriveStatus(recorder.buffer.Bytes(), recorder.status), recorder.status, recorder.buffer.Bytes()); err != nil {
			writeError(w, http.StatusInternalServerError, `{"error":"idempotency_commit_failed"}`)
			return
		}
		copyHeaders(w.Header(), recorder.header)
		w.WriteHeader(recorder.status)
		_, _ = w.Write(recorder.buffer.Bytes())
	})
}

// replay writes a cached response verbatim, marking it as a replay so clients
// can distinguish fresh execution. It performs no downstream execution.
func replay(w http.ResponseWriter, cached *Cached) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Idempotent-Replay", "true")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(cached.HTTPStatus)
	_, _ = w.Write(cached.Payload)
}

func validIdempotencyKey(key string) bool {
	if len(key) < 8 || len(key) > 128 {
		return false
	}
	for i := 0; i < len(key); i++ {
		switch {
		case key[i] >= 'a' && key[i] <= 'z':
		case key[i] >= 'A' && key[i] <= 'Z':
		case key[i] >= '0' && key[i] <= '9':
		case key[i] == '-' || key[i] == '_' || key[i] == '.':
		default:
			return false
		}
	}
	return true
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func jsonString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func writeError(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// SyncScopeResolver derives the tenant scope from the /sync envelope body.
func SyncScopeResolver(r *http.Request, body []byte) (tenantID, organizationID string, err error) {
	var envelope struct {
		TenantID       string `json:"tenant_id"`
		OrganizationID string `json:"organization_id"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", "", errors.New("tenant_id and organization_id are required")
	}
	tenantID = strings.TrimSpace(envelope.TenantID)
	organizationID = strings.TrimSpace(envelope.OrganizationID)
	if tenantID == "" || organizationID == "" {
		return "", "", errors.New("tenant_id and organization_id are required")
	}
	return tenantID, organizationID, nil
}

// EvidenceScopeResolver derives the tenant scope from the /evidence envelope.
func EvidenceScopeResolver(r *http.Request, body []byte) (tenantID, organizationID string, err error) {
	var envelope struct {
		TenantID       string `json:"tenant_id"`
		OrganizationID string `json:"organization_id"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", "", errors.New("tenant_id and organization_id are required")
	}
	tenantID = strings.TrimSpace(envelope.TenantID)
	organizationID = strings.TrimSpace(envelope.OrganizationID)
	if tenantID == "" || organizationID == "" {
		return "", "", errors.New("tenant_id and organization_id are required")
	}
	return tenantID, organizationID, nil
}
