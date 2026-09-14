// INTEGIN session lifecycle: active-session tracking and OrganizationContext HTTP binding.
package oidchttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"integin/internal/identity"
)

var (
	ErrSessionMissing = errors.New("session is not active")
	ErrSessionExpired = errors.New("session has exceeded its lifetime or inactivity limit")
	ErrSessionRevoked = errors.New("session has been revoked")
)

// sessionStoreTimeout bounds each store round-trip so lifecycle checks never
// hang behind an unresponsive database.
const sessionStoreTimeout = 5 * time.Second

// SessionRecord is the state of one active session.
type SessionRecord struct {
	Token        string
	Subject      string
	IssuedAt     time.Time
	LastActiveAt time.Time
}

// SessionLifecycleManager tracks active sessions with issue time, sliding
// inactivity time, and maximum TTL. With a nil store it is single-process and
// in-memory only; with a SessionStore it persists every lifecycle write so
// multi-instance restarts retain active session state.
type SessionLifecycleManager struct {
	mu       sync.RWMutex
	sessions map[string]SessionRecord
	revoked  map[string]time.Time
	maxTTL   time.Duration
	now      func() time.Time
	store    SessionStore
}

func (m *SessionLifecycleManager) sessionID(sessionID string) string {
	return strings.TrimSpace(sessionID)
}

// NewSessionLifecycleManager returns an in-memory lifecycle manager bound to
// the given maximum session TTL.
func NewSessionLifecycleManager(maxTTL time.Duration) *SessionLifecycleManager {
	return newSessionLifecycleManager(maxTTL, nil)
}

// NewSessionLifecycleManagerWithStore returns a lifecycle manager whose state
// is durably persisted through store, so a restarted instance retains active
// sessions opened by any other instance sharing the store.
func NewSessionLifecycleManagerWithStore(maxTTL time.Duration, store SessionStore) (*SessionLifecycleManager, error) {
	if store == nil {
		return nil, errors.New("session lifecycle store is required")
	}
	return newSessionLifecycleManager(maxTTL, store), nil
}

func newSessionLifecycleManager(maxTTL time.Duration, store SessionStore) *SessionLifecycleManager {
	if maxTTL <= 0 {
		maxTTL = 8 * time.Hour
	}
	return &SessionLifecycleManager{
		sessions: make(map[string]SessionRecord),
		revoked:  make(map[string]time.Time),
		maxTTL:   maxTTL,
		now:      time.Now,
		store:    store,
	}
}

func (m *SessionLifecycleManager) sessionCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), sessionStoreTimeout)
}

// Open records a freshly authenticated token as an active session bound to the
// OIDC subject. Revoked tokens fail closed.
func (m *SessionLifecycleManager) Open(token, subject string, issuedAt time.Time) error {
	token = m.sessionID(token)
	if token == "" {
		return ErrSessionMissing
	}
	if issuedAt.IsZero() {
		issuedAt = m.now()
	}
	if m.store == nil {
		m.mu.Lock()
		defer m.mu.Unlock()
		if _, revoked := m.revoked[token]; revoked {
			return ErrSessionRevoked
		}
		m.sessions[token] = SessionRecord{Token: token, Subject: subject, IssuedAt: issuedAt, LastActiveAt: m.now()}
		return nil
	}
	ctx, cancel := m.sessionCtx()
	defer cancel()
	stored, err := m.store.Get(ctx, token)
	if err != nil && !errors.Is(err, ErrSessionMissing) {
		return err
	}
	if err == nil && !stored.RevokedAt.IsZero() {
		return ErrSessionRevoked
	}
	return m.store.Save(ctx, StoredSession{
		SessionID: token, Subject: subject, IssuedAt: issuedAt,
		LastActiveAt: m.now(), ExpiresAt: issuedAt.Add(m.maxTTL),
	})
}

// Touch refreshes the sliding inactivity window. Expired or revoked sessions fail closed.
func (m *SessionLifecycleManager) Touch(token string) error {
	token = m.sessionID(token)
	if m.store == nil {
		m.mu.Lock()
		defer m.mu.Unlock()
		record, ok := m.sessions[token]
		if !ok {
			if _, revoked := m.revoked[token]; revoked {
				return ErrSessionRevoked
			}
			return ErrSessionMissing
		}
		now := m.now()
		if m.expired(record, now) {
			delete(m.sessions, token)
			return ErrSessionExpired
		}
		record.LastActiveAt = now
		m.sessions[token] = record
		return nil
	}
	ctx, cancel := m.sessionCtx()
	defer cancel()
	stored, err := m.store.Get(ctx, token)
	if err != nil {
		if errors.Is(err, ErrSessionMissing) {
			return ErrSessionMissing
		}
		return err
	}
	if !stored.RevokedAt.IsZero() {
		return ErrSessionRevoked
	}
	now := m.now()
	if m.expired(SessionRecord{IssuedAt: stored.IssuedAt, LastActiveAt: stored.LastActiveAt}, now) {
		return ErrSessionExpired
	}
	return m.store.Save(ctx, StoredSession{
		SessionID: token, Subject: stored.Subject, IssuedAt: stored.IssuedAt,
		LastActiveAt: now, ExpiresAt: now.Add(m.maxTTL),
	})
}

// RevokeSession invalidates a live session and records a tombstone so the
// token cannot be reopened.
func (m *SessionLifecycleManager) RevokeSession(sessionID string) error {
	sessionID = m.sessionID(sessionID)
	if sessionID == "" {
		return ErrSessionMissing
	}
	if m.store == nil {
		m.mu.Lock()
		defer m.mu.Unlock()
		if _, ok := m.sessions[sessionID]; !ok {
			return ErrSessionMissing
		}
		delete(m.sessions, sessionID)
		m.revoked[sessionID] = m.now()
		return nil
	}
	ctx, cancel := m.sessionCtx()
	defer cancel()
	stored, err := m.store.Get(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionMissing) {
			return ErrSessionMissing
		}
		return err
	}
	stored.RevokedAt = m.now()
	return m.store.Save(ctx, stored)
}

// IsRevoked reports whether the session ID carries a revocation tombstone.
func (m *SessionLifecycleManager) IsRevoked(sessionID string) bool {
	if m.store == nil {
		m.mu.RLock()
		defer m.mu.RUnlock()
		_, revoked := m.revoked[m.sessionID(sessionID)]
		return revoked
	}
	ctx, cancel := m.sessionCtx()
	defer cancel()
	stored, err := m.store.Get(ctx, m.sessionID(sessionID))
	if err != nil {
		return false
	}
	return !stored.RevokedAt.IsZero()
}

// Known reports whether the session ID was ever established and has not been
// externally purged. Expired sessions remain known until PurgeExpired, so
// lifecycle enforcement fails closed.
func (m *SessionLifecycleManager) Known(sessionID string) bool {
	if m.store == nil {
		m.mu.RLock()
		defer m.mu.RUnlock()
		_, ok := m.sessions[m.sessionID(sessionID)]
		return ok
	}
	ctx, cancel := m.sessionCtx()
	defer cancel()
	_, err := m.store.Get(ctx, m.sessionID(sessionID))
	return err == nil
}

// IsActive reports whether the session is live: issued, within lifetime and
// inactivity windows, and not revoked.
func (m *SessionLifecycleManager) IsActive(sessionID string) bool {
	if m.store == nil {
		m.mu.RLock()
		defer m.mu.RUnlock()
		record, ok := m.sessions[m.sessionID(sessionID)]
		if !ok {
			return false
		}
		if _, revoked := m.revoked[record.Token]; revoked {
			return false
		}
		return !m.expired(record, m.now())
	}
	ctx, cancel := m.sessionCtx()
	defer cancel()
	stored, err := m.store.Get(ctx, m.sessionID(sessionID))
	if err != nil {
		return false
	}
	if !stored.RevokedAt.IsZero() {
		return false
	}
	record := SessionRecord{IssuedAt: stored.IssuedAt, LastActiveAt: stored.LastActiveAt}
	return !m.expired(record, m.now())
}

// PurgeExpired removes sessions past their lifetime or inactivity limit. Safe
// for a background sweeper.
func (m *SessionLifecycleManager) PurgeExpired() {
	if m.store == nil {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.pruneLocked()
		return
	}
	ctx, cancel := m.sessionCtx()
	defer cancel()
	if err := m.store.PurgeExpired(ctx, m.now()); err != nil {
		// Best-effort background sweep; the failure must surface in logs
		// rather than vanish silently.
		slog.Error("purge expired oidc sessions", "error", err)
	}
}

func (m *SessionLifecycleManager) expired(record SessionRecord, now time.Time) bool {
	return now.Sub(record.IssuedAt) > m.maxTTL || now.Sub(record.LastActiveAt) > m.maxTTL
}

func (m *SessionLifecycleManager) pruneLocked() {
	now := m.now()
	for token, record := range m.sessions {
		if m.expired(record, now) {
			delete(m.sessions, token)
		}
	}
}

type orgContextKey struct{}

// WithOrganizationContext binds a validated tenant-scoped authorization context into an HTTP request context.
func WithOrganizationContext(ctx context.Context, org identity.OrganizationContext) context.Context {
	return context.WithValue(ctx, orgContextKey{}, org)
}

// OrganizationContextFrom extracts the bound organization context; ok is false when absent.
func OrganizationContextFrom(ctx context.Context) (identity.OrganizationContext, bool) {
	org, ok := ctx.Value(orgContextKey{}).(identity.OrganizationContext)
	return org, ok
}

// SessionAuthenticator validates tokens, projects the OrganizationContext, and enforces session lifecycle.
type SessionAuthenticator struct {
	validator TokenValidator
	resolver  identity.Resolver
	sessions  *SessionLifecycleManager
	mfa       identity.MFAPolicy
}

// NewSessionAuthenticator validates its dependencies and returns a session-aware middleware builder.
func NewSessionAuthenticator(validator TokenValidator, resolver identity.Resolver, sessions *SessionLifecycleManager, mfa identity.MFAPolicy) (*SessionAuthenticator, error) {
	if validator == nil {
		return nil, errors.New("session authenticator validator is required")
	}
	if resolver == nil {
		return nil, errors.New("session authenticator identity resolver is required")
	}
	if sessions == nil {
		return nil, errors.New("session authenticator lifecycle manager is required")
	}
	return &SessionAuthenticator{validator: validator, resolver: resolver, sessions: sessions, mfa: mfa}, nil
}

// Middleware enforces session validity and binds the projected OrganizationContext into the request context.
func (a *SessionAuthenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		rawToken, ok := bearerToken(request.Header.Get("Authorization"))
		if !ok {
			writeFailure(writer, http.StatusUnauthorized, "authentication_failed")
			return
		}
		if a.sessions.IsRevoked(rawToken) {
			writeFailure(writer, http.StatusUnauthorized, "session_revoked")
			return
		}
		active := a.sessions.IsActive(rawToken)
		if a.sessions.Known(rawToken) && !active {
			// Fail closed: a previously established session past its TTL or inactivity window
			// requires a new authentication rather than silent re-opening.
			writeFailure(writer, http.StatusUnauthorized, "session_expired")
			return
		}
		principal, err := a.validator.Validate(request.Context(), rawToken)
		if err != nil {
			writeFailure(writer, http.StatusUnauthorized, "authentication_failed")
			return
		}
		membership, err := a.resolver.Resolve(request.Context(), identity.PrincipalKey{Issuer: principal.Issuer, Subject: principal.Subject})
		if err != nil {
			if errors.Is(err, identity.ErrUnknownSubject) || errors.Is(err, identity.ErrAmbiguousMembership) {
				writeFailure(writer, http.StatusForbidden, "authorization_failed")
				return
			}
			writeFailure(writer, http.StatusServiceUnavailable, "authorization_unavailable")
			return
		}
		org, err := identity.ProjectMembership(membership, principal, a.mfa)
		if err != nil {
			reason := "authorization_failed"
			if errors.Is(err, identity.ErrMFARequired) {
				reason = "mfa_required"
			}
			writeFailure(writer, http.StatusForbidden, reason)
			return
		}
		if tenantScopeConflict(request, org.TenantID, org.OrganizationID) {
			// Fail closed before any session is opened: client-supplied tenant or
			// organization scoping can never override the server-resolved projection.
			writeFailure(writer, http.StatusForbidden, "tenant_scope_conflict")
			return
		}
		if active {
			if err := a.sessions.Touch(rawToken); err != nil {
				writeFailure(writer, http.StatusUnauthorized, "session_expired")
				return
			}
		} else if err := a.sessions.Open(rawToken, principal.Subject, time.Now()); err != nil {
			writeFailure(writer, http.StatusUnauthorized, "session_expired")
			return
		}
		next.ServeHTTP(writer, request.WithContext(WithOrganizationContext(request.Context(), org)))
	})
}
