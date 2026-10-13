package syncstate

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type DeviceState string

const (
	DevicePending    DeviceState = "PENDING"
	DeviceTrusted    DeviceState = "TRUSTED"
	DeviceRestricted DeviceState = "RESTRICTED"
	DeviceLocked     DeviceState = "LOCKED"
	DeviceRevoked    DeviceState = "REVOKED"
	DeviceRetired    DeviceState = "RETIRED"
)

type DeviceRecord struct {
	DeviceID         string
	TenantID         string
	OrganizationID   string
	UserID           string
	KeyID            string
	PublicKey        []byte
	State            DeviceState
	AuthorityEpoch   uint64
	EnrolledAt       time.Time
	UpdatedAt        time.Time
	RevokedAt        *time.Time
	RevocationReason string
}

type AuthorityRecord struct {
	AuthorityID      string
	TenantID         string
	OrganizationID   string
	DeviceID         string
	UserID           string
	AuthorityEpoch   uint64
	Scopes           []string
	ProcedureVersion string
	IssuedAt         time.Time
	ExpiresAt        time.Time
	Signature        []byte
	RevokedAt        *time.Time
}

type Receipt struct {
	TransactionID  string
	TenantID       string
	OrganizationID string
	DeviceID       string
	UserID         string
	SequenceNumber uint64
	Operation      string
	EntityID       string
	PayloadHash    string
	Outcome        string
	Reason         string
	CapturedAt     time.Time
	ReceivedAt     time.Time
}

type HeldTransaction struct {
	Receipt          Receipt
	ExpectedSequence uint64
	Envelope         json.RawMessage
	FirstHeldAt      time.Time
	LastAttemptAt    time.Time
	LastError        string
}

type Repository interface {
	DeviceRepository
	AuthorityRepository
	SyncStateRepository
}

type DeviceRepository interface {
	GetDevice(ctx context.Context, tenantID, deviceID string) (DeviceRecord, error)
	ListDevices(ctx context.Context, tenantID string) ([]DeviceRecord, error)
	SaveDevice(ctx context.Context, device DeviceRecord) error
}

type AuthorityRepository interface {
	GetAuthority(ctx context.Context, tenantID, authorityID string) (AuthorityRecord, error)
	ListAuthorities(ctx context.Context, tenantID string) ([]AuthorityRecord, error)
	SaveAuthority(ctx context.Context, authority AuthorityRecord) error
	RevokeAuthority(ctx context.Context, tenantID, authorityID string, at time.Time) error
}

type SyncStateRepository interface {
	GetLastAcceptedSequence(ctx context.Context, tenantID, deviceID string) (uint64, error)
	SaveReceipt(ctx context.Context, receipt Receipt) error
	GetReceipt(ctx context.Context, tenantID, transactionID string) (Receipt, error)
	SaveHeld(ctx context.Context, held HeldTransaction) error
	ListHeld(ctx context.Context, tenantID, deviceID string) ([]HeldTransaction, error)
}

var ErrNotFound = errors.New("sync-state record not found")

const maxSafeSequence = 1<<63 - 1

func ValidateDeviceRecord(device DeviceRecord) error {
	if device.DeviceID == "" || device.TenantID == "" || device.OrganizationID == "" || device.UserID == "" || device.KeyID == "" || len(device.PublicKey) == 0 {
		return errors.New("device identity, key, and tenant scope are required")
	}
	if device.AuthorityEpoch == 0 {
		return errors.New("device authority epoch must be positive")
	}
	if device.AuthorityEpoch > maxSafeSequence {
		return errors.New("device authority epoch exceeds safe integer bounds")
	}
	return nil
}

func ValidateReceipt(receipt Receipt) error {
	if receipt.TransactionID == "" || receipt.TenantID == "" || receipt.OrganizationID == "" || receipt.DeviceID == "" || receipt.UserID == "" || receipt.Operation == "" || receipt.EntityID == "" || receipt.PayloadHash == "" || receipt.Outcome == "" {
		return errors.New("receipt identity, operation, payload hash, outcome, and tenant scope are required")
	}
	if receipt.SequenceNumber == 0 {
		return errors.New("receipt sequence must be positive")
	}
	if receipt.SequenceNumber > maxSafeSequence {
		return errors.New("receipt sequence exceeds safe integer bounds")
	}
	return nil
}
