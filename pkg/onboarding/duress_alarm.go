package onboarding

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"integin/internal/platform/audit"
)

// SyncState is the operational state a device reports in its sync payload.
type SyncState string

const (
	// SyncStateHealthy is a normal state that never triggers duress dispatch.
	SyncStateHealthy SyncState = "HEALTHY"
	// SyncStateCoercionQuarantine is the duress signal: a coerced operator
	// submitted the silent alarm while the device surface kept behaving
	// normally, so nothing in the UI betrays that operations were alerted.
	SyncStateCoercionQuarantine SyncState = "STATE_COERCION_QUARANTINE"
)

const (
	// EventSilentAlarm is the ops notification type and audit action emitted
	// for a coercion quarantine sync. It matches the field_app silent alarm
	// payload contract.
	EventSilentAlarm = "SILENT_ALARM"
	// DuressPriority is the urgent audit outcome assigned to duress alerts.
	DuressPriority = "urgent"
	// DuressQuarantineFlag is the flag recorded on quarantine events that
	// arrive without one, matching the field_app quarantine payload flag.
	DuressQuarantineFlag = "COERCION_QUARANTINE"
)

// DeviceSyncPayload is the minimal sync envelope the dispatcher inspects.
type DeviceSyncPayload struct {
	DeviceID string    `json:"device_id"`
	TenantID string    `json:"tenant_id"`
	UserID   string    `json:"user_id"`
	State    SyncState `json:"state"`
	Flag     string    `json:"flag,omitempty"`
	QueuedAt time.Time `json:"queued_at"`
}

// OpsNotification is the urgent event delivered to operations listeners when
// a device sync reports coercion quarantine.
type OpsNotification struct {
	Event         string    `json:"event"`
	DeviceID      string    `json:"device_id"`
	TenantID      string    `json:"tenant_id"`
	UserID        string    `json:"user_id"`
	Flag          string    `json:"flag"`
	CoercionState SyncState `json:"coercion_state"`
	QueuedAt      time.Time `json:"queued_at"`
}

// OpsListener receives urgent duress notifications. Implementations must treat
// delivery failure as an operational incident and return the error so it
// surfaces to the ingress caller.
type OpsListener func(ctx context.Context, notification OpsNotification) error

// DuressAlarmDispatcher turns a sync payload bearing STATE_COERCION_QUARANTINE
// into an urgent audit alert plus ops notifications. Any other state is
// ignored. Listener dispatch runs against a lock-protected snapshot so
// registration during dispatch can never race.
type DuressAlarmDispatcher struct {
	mu          sync.RWMutex
	ledger      *audit.Ledger
	environment string
	listeners   []OpsListener
	now         func() time.Time
}

// NewDuressAlarmDispatcher wires the dispatcher to the audit sink and records
// the environment tag on every emitted alert.
func NewDuressAlarmDispatcher(environment string, ledger *audit.Ledger) *DuressAlarmDispatcher {
	if ledger == nil {
		ledger = audit.NewLedger()
	}
	return &DuressAlarmDispatcher{
		ledger: ledger, environment: environment, now: time.Now, listeners: make([]OpsListener, 0),
	}
}

// AddListener registers an ops listener for duress notifications.
func (d *DuressAlarmDispatcher) AddListener(listener OpsListener) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.listeners = append(d.listeners, listener)
}

// HandleSync inspects one sync payload. A coercion quarantine state emits an
// urgent audit alert and notifies every ops listener; any other state is a
// no-op.
func (d *DuressAlarmDispatcher) HandleSync(ctx context.Context, payload DeviceSyncPayload) error {
	if strings.TrimSpace(payload.DeviceID) == "" {
		return errors.New("device_id is required")
	}
	if strings.TrimSpace(payload.TenantID) == "" {
		return errors.New("tenant_id is required")
	}
	if payload.State != SyncStateCoercionQuarantine {
		return nil
	}
	if payload.QueuedAt.IsZero() {
		payload.QueuedAt = d.now()
	}
	if strings.TrimSpace(payload.Flag) == "" {
		payload.Flag = DuressQuarantineFlag
	}

	d.mu.RLock()
	listeners := make([]OpsListener, len(d.listeners))
	copy(listeners, d.listeners)
	d.mu.RUnlock()

	idBytes := make([]byte, 8)
	if _, err := rand.Read(idBytes); err != nil {
		return fmt.Errorf("cannot generate duress alert id: %w", err)
	}
	alert := audit.Record{
		ID:           fmt.Sprintf("alert_%s", hex.EncodeToString(idBytes)),
		TenantID:     payload.TenantID,
		Environment:  d.environment,
		ActorType:    "device",
		ActorID:      payload.DeviceID,
		Action:       EventSilentAlarm,
		ResourceType: "device",
		ResourceID:   payload.DeviceID,
		Outcome:      DuressPriority,
		Context: map[string]any{
			"user_id":             payload.UserID,
			"flag":                payload.Flag,
			"coercion_quarantine": true,
			"queued_at":           payload.QueuedAt,
		},
		CreatedAt: payload.QueuedAt.UTC(),
	}
	if err := d.ledger.Append(alert); err != nil {
		return fmt.Errorf("duress audit alert not recorded: %w", err)
	}

	notification := OpsNotification{
		Event: EventSilentAlarm, DeviceID: payload.DeviceID, TenantID: payload.TenantID,
		UserID: payload.UserID, Flag: payload.Flag,
		CoercionState: payload.State, QueuedAt: payload.QueuedAt,
	}
	notifyErrs := make([]error, 0, len(listeners))
	for _, listener := range listeners {
		if err := listener(ctx, notification); err != nil {
			notifyErrs = append(notifyErrs, fmt.Errorf("ops listener failed: %w", err))
		}
	}
	if len(notifyErrs) > 0 {
		return errors.Join(notifyErrs...)
	}
	return nil
}
