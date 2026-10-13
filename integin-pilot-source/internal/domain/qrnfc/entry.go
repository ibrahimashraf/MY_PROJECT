package qrnfc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidActor  = errors.New("QR/NFC entry actor context is invalid")
	ErrInvalidEntry  = errors.New("QR/NFC entry is invalid")
	ErrInvalidToken  = errors.New("token is invalid")
	ErrNotFound      = errors.New("QR/NFC entry not found")
	ErrExpired       = errors.New("QR/NFC entry is expired")
	ErrRevoked       = errors.New("QR/NFC entry is revoked")
	ErrInvalidStatus = errors.New("invalid status transition")
)

// ActorContext is the server-derived actor for QR/NFC operations.
type ActorContext struct {
	TenantID       string
	OrganizationID string
	ActorID        string
}

func (a ActorContext) Validate() error {
	if strings.TrimSpace(a.TenantID) == "" || strings.TrimSpace(a.OrganizationID) == "" || strings.TrimSpace(a.ActorID) == "" {
		return ErrInvalidActor
	}
	return nil
}

// EntryType represents the type of physical entry.
type EntryType string

const (
	EntryTypeQR  EntryType = "QR"
	EntryTypeNFC EntryType = "NFC"
)

// EntryStatus represents the status of a QR/NFC entry.
type EntryStatus string

const (
	EntryStatusActive  EntryStatus = "ACTIVE"
	EntryStatusRevoked EntryStatus = "REVOKED"
	EntryStatusExpired EntryStatus = "EXPIRED"
)

// AccessOutcome represents the outcome of an access attempt.
type AccessOutcome string

const (
	AccessOutcomeSuccess AccessOutcome = "SUCCESS"
	AccessOutcomeDenied  AccessOutcome = "DENIED"
	AccessOutcomeExpired AccessOutcome = "EXPIRED"
	AccessOutcomeRevoked AccessOutcome = "REVOKED"
)

// Entry represents an opaque, unguessable QR/NFC token stored only as SHA-256 digest.
type Entry struct {
	ID             string
	TenantID       string
	OrganizationID string
	EntitlementID  string
	AssetID        string
	EntryType      EntryType
	TokenDigest    string
	TokenVersion   int
	Status         EntryStatus
	IssuedAt       time.Time
	ExpiresAt      *time.Time
	RevokedAt      *time.Time
	RevokedBy      string
	CreatedBy      string
	CreatedAt      time.Time
}

// Validate ensures the entry is consistent.
func (e Entry) Validate() error {
	if strings.TrimSpace(e.ID) == "" || strings.TrimSpace(e.EntitlementID) == "" || strings.TrimSpace(e.AssetID) == "" {
		return fmt.Errorf("%w: id, entitlement_id, and asset_id are required", ErrInvalidEntry)
	}
	switch e.EntryType {
	case EntryTypeQR, EntryTypeNFC:
		// valid
	default:
		return fmt.Errorf("%w: unsupported entry_type %q", ErrInvalidEntry, e.EntryType)
	}
	if strings.TrimSpace(e.TokenDigest) == "" {
		return fmt.Errorf("%w: token_digest is required", ErrInvalidEntry)
	}
	if e.TokenVersion <= 0 {
		return fmt.Errorf("%w: token_version must be positive", ErrInvalidEntry)
	}
	switch e.Status {
	case EntryStatusActive:
		if e.RevokedAt != nil || e.RevokedBy != "" {
			return fmt.Errorf("%w: ACTIVE status cannot have revoked_at or revoked_by", ErrInvalidEntry)
		}
	case EntryStatusRevoked:
		if e.RevokedAt == nil || strings.TrimSpace(e.RevokedBy) == "" {
			return fmt.Errorf("%w: REVOKED status requires revoked_at and revoked_by", ErrInvalidEntry)
		}
	case EntryStatusExpired:
		// valid
	default:
		return fmt.Errorf("%w: unsupported status %q", ErrInvalidEntry, e.Status)
	}
	if e.ExpiresAt != nil && e.ExpiresAt.Before(e.IssuedAt) {
		return fmt.Errorf("%w: expires_at must be after issued_at", ErrInvalidEntry)
	}
	if e.CreatedBy == "" || e.CreatedAt.IsZero() {
		return fmt.Errorf("%w: created_by and created_at are required", ErrInvalidEntry)
	}
	return nil
}

// IsExpired checks if the entry is expired at the given time.
func (e Entry) IsExpired(at time.Time) bool {
	return e.ExpiresAt != nil && at.After(*e.ExpiresAt)
}

// CanTransitionTo checks if a status transition is valid.
func (e Entry) CanTransitionTo(target EntryStatus) error {
	switch e.Status {
	case EntryStatusActive:
		if target != EntryStatusRevoked && target != EntryStatusExpired {
			return fmt.Errorf("%w: ACTIVE can only transition to REVOKED or EXPIRED", ErrInvalidStatus)
		}
	case EntryStatusRevoked:
		return fmt.Errorf("%w: REVOKED entries are terminal", ErrInvalidStatus)
	case EntryStatusExpired:
		return fmt.Errorf("%w: EXPIRED entries are terminal", ErrInvalidStatus)
	default:
		return fmt.Errorf("%w: unknown current status", ErrInvalidStatus)
	}
	return nil
}

// EntryLog represents a privacy-preserving access log entry.
type EntryLog struct {
	ID             string
	TenantID       string
	OrganizationID string
	EntryID        string
	EntryType      EntryType
	AccessedAt     time.Time
	AccessorID     string
	AccessorIP     string
	Outcome        AccessOutcome
}

// Validate ensures the log entry is consistent.
func (l EntryLog) Validate() error {
	if strings.TrimSpace(l.ID) == "" || strings.TrimSpace(l.EntryID) == "" {
		return fmt.Errorf("%w: id and entry_id are required", ErrInvalidEntry)
	}
	if strings.TrimSpace(l.AccessorID) == "" {
		return fmt.Errorf("%w: accessor_id is required", ErrInvalidEntry)
	}
	if l.AccessedAt.IsZero() {
		return fmt.Errorf("%w: accessed_at is required", ErrInvalidEntry)
	}
	switch l.Outcome {
	case AccessOutcomeSuccess, AccessOutcomeDenied, AccessOutcomeExpired, AccessOutcomeRevoked:
		// valid
	default:
		return fmt.Errorf("%w: unsupported outcome %q", ErrInvalidEntry, l.Outcome)
	}
	return nil
}

// GenerateToken creates a new opaque, unguessable token and returns (rawToken, digest).
// The raw token is URL-safe base64 encoded, 43 characters.
// The digest is SHA-256 of the raw token.
func GenerateToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("failed to generate random token: %w", err)
	}
	rawToken := base64.RawURLEncoding.EncodeToString(b)
	digest := sha256.Sum256([]byte(rawToken))
	return rawToken, fmt.Sprintf("%x", digest), nil
}

// VerifyTokenDigest verifies a raw token against a stored digest.
func VerifyTokenDigest(rawToken, storedDigest string) bool {
	if len(rawToken) < 32 || len(rawToken) > 128 {
		return false
	}
	digest := sha256.Sum256([]byte(strings.TrimSpace(rawToken)))
	return fmt.Sprintf("%x", digest) == storedDigest
}

// RedactTokenForLog returns a redacted version of a token for logging.
func RedactTokenForLog(rawToken string) string {
	if len(rawToken) <= 8 {
		return "****"
	}
	return rawToken[:4] + "****" + rawToken[len(rawToken)-4:]
}

// Repository defines persistence operations for QR/NFC entries.
type Repository interface {
	// Issue creates a new ACTIVE QR/NFC entry with a generated token.
	// Returns the raw token (once) and the stored entry.
	Issue(ctx context.Context, actor ActorContext, entry Entry) (Entry, string, error)
	// Get retrieves an entry by ID.
	Get(ctx context.Context, actor ActorContext, entryID string) (Entry, error)
	// VerifyByDigest verifies a token digest and returns the entry if valid.
	VerifyByDigest(ctx context.Context, actor ActorContext, tokenDigest string) (Entry, error)
	// Revoke transitions an ACTIVE entry to REVOKED.
	Revoke(ctx context.Context, actor ActorContext, entryID string) (Entry, error)
	// ListByAsset retrieves all entries for an asset.
	ListByAsset(ctx context.Context, actor ActorContext, assetID string) ([]Entry, error)
	// LogAccess records a privacy-preserving access log entry.
	LogAccess(ctx context.Context, actor ActorContext, log EntryLog) error
}
