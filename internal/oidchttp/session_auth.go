// INTEGIN session lifecycle: single-process active-session tracking and OrganizationContext HTTP binding.
package oidchttp

import (
	"context"
	"errors"
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

// SessionRecord is the in-memory state of one active session.
type SessionRecord struct {
	Token        string
	IssuedAt     time.Time
	LastActiveAt time.Time
}

// SessionLifecycleManager tracks active sessions with issue time, sliding inactivity time, and maximum TTL.
// It is single-process only.
type SessionLifecycleManager struct {
	mu       sync.RWMutex
	sessions map[string]SessionRecord
	revoked  map[string]time.Time
	maxTTL   time.Duration
	now      func() time.Time
}

func (m *SessionLifecycleManager) sessionID(sessionID string) string {
	return strings.TrimSpace(sessionID)
}

// NewSessionLifecycleManager returns a lifecycle manager bound to the given maximum session TTL.
func NewSessionLifecycleManager(maxTTL time.Duration) *SessionLifecycleManager {
	if maxTTL <= 0 {
		maxTTL = 8 * time.Hour
	}
	return &SessionLifecycleManager{
		sessions: make(map[string]SessionRecord),
		revoked:  make(map[string]time.Time),
		maxTTL:   maxTTL,
		now:      time.Now,
	}
}

// Open records a freshly authenticated token as an active session. Revoked tokens fail closed.
func (m *SessionLifecycleManager) Open(token string, issuedAt time.Time) error {
	token = m.sessionID(token)
	if token == "" {
		return ErrSessionMissing
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, revoked := m.revoked[token]; revoked {
		return ErrSessionRevoked
	}
	if issuedAt.IsZero() {
		issuedAt = m.now()
	}
	m.sessions[token] = SessionRecord{Token: token, IssuedAt: issuedAt, LastActiveAt: m.now()}
	return nil
}

// Touch refreshes the sliding inactivity window. Expired or revoked sessions fail closed.
func (m *SessionLifecycleManager) Touch(token string) error {
	token = m.sessionID(token)
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

// RevokeSession invalidates a live session and records a tombstone so the token cannot be reopened.
func (m *SessionLifecycleManager) RevokeSession(sessionID string) error {
	sessionID = m.sessionID(sessionID)
	if sessionID == "" {
		return ErrSessionMissing
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[sessionID]; !ok {
		return ErrSessionMissing
	}
	delete(m.sessions, sessionID)
	m.revoked[sessionID] = m.now()
	return nil
}

// IsRevoked reports whether the session ID carries a revocation tombstone.
func (m *SessionLifecycleManager) IsRevoked(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, revoked := m.revoked[m.sessionID(sessionID)]
	return revoked
}

// Known reports whether the session ID was ever established and has not been externally purged.
// Expired sessions remain known until PurgeExpired, so lifecycle enforcement fails closed.
func (m *SessionLifecycleManager) Known(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.sessions[m.sessionID(sessionID)]
	return ok
}

// IsActive reports whether the session is live: issued, within lifetime and inactivity windows, and not revoked.
func (m *SessionLifecycleManager) IsActive(sessionID string) bool {
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

// PurgeExpired removes sessions past their lifetime or inactivity limit. Safe for a background sweeper.
func (m *SessionLifecycleManager) PurgeExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked()
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
		if active {
			if err := a.sessions.Touch(rawToken); err != nil {
				writeFailure(writer, http.StatusUnauthorized, "session_expired")
				return
			}
		} else if err := a.sessions.Open(rawToken, time.Now()); err != nil {
			writeFailure(writer, http.StatusUnauthorized, "session_expired")
			return
		}
		next.ServeHTTP(writer, request.WithContext(WithOrganizationContext(request.Context(), org)))
	})
}
