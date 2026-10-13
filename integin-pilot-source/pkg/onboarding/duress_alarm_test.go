package onboarding

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"integin/internal/platform/audit"
)

func TestDuressQuarantineEmitsAuditAlertAndNotifies(t *testing.T) {
	ledger := audit.NewLedger()
	dispatcher := NewDuressAlarmDispatcher("production", ledger)
	var notified []OpsNotification
	dispatcher.AddListener(func(_ context.Context, n OpsNotification) error {
		notified = append(notified, n)
		return nil
	})

	payload := DeviceSyncPayload{
		DeviceID: "dev_001", TenantID: "tenant-1", UserID: "user-oidc-1",
		State: SyncStateCoercionQuarantine, Flag: DuressQuarantineFlag,
		QueuedAt: time.Date(2026, 9, 14, 8, 30, 0, 0, time.UTC),
	}
	if err := dispatcher.HandleSync(context.Background(), payload); err != nil {
		t.Fatal(err)
	}

	records := ledger.List("tenant-1")
	if len(records) != 1 {
		t.Fatalf("expected exactly one audit alert, got %d", len(records))
	}
	record := records[0]
	if record.Action != EventSilentAlarm || record.Outcome != DuressPriority {
		t.Fatalf("audit alert must carry the silent alarm action at urgent priority, got action %q outcome %q", record.Action, record.Outcome)
	}
	if record.ResourceID != "dev_001" || record.TenantID != "tenant-1" {
		t.Fatalf("audit alert must identify the quarantined device and tenant")
	}

	if len(notified) != 1 {
		t.Fatalf("expected one ops notification, got %d", len(notified))
	}
	n := notified[0]
	if n.Event != EventSilentAlarm || n.DeviceID != "dev_001" || n.TenantID != "tenant-1" || n.UserID != "user-oidc-1" {
		t.Fatalf("ops notification must carry the full duress context, got %+v", n)
	}
	if n.CoercionState != SyncStateCoercionQuarantine || n.Flag != DuressQuarantineFlag {
		t.Fatalf("ops notification must carry the coercion state and flag, got %+v", n)
	}
}

func TestDuressQuarantineAppliesDefaultFlag(t *testing.T) {
	ledger := audit.NewLedger()
	dispatcher := NewDuressAlarmDispatcher("production", ledger)
	var notified []OpsNotification
	dispatcher.AddListener(func(_ context.Context, n OpsNotification) error {
		notified = append(notified, n)
		return nil
	})

	payload := DeviceSyncPayload{
		DeviceID: "dev_001", TenantID: "tenant-1", State: SyncStateCoercionQuarantine,
	}
	if err := dispatcher.HandleSync(context.Background(), payload); err != nil {
		t.Fatal(err)
	}
	if len(notified) != 1 || notified[0].Flag != DuressQuarantineFlag {
		t.Fatalf("missing flag must default to %s, got %+v", DuressQuarantineFlag, notified)
	}
	records := ledger.List("tenant-1")
	if len(records) != 1 || records[0].Context["flag"] != DuressQuarantineFlag {
		t.Fatalf("audit alert must carry the default flag, got %+v", records)
	}
}

func TestDuressHealthySyncIsNoOp(t *testing.T) {
	ledger := audit.NewLedger()
	dispatcher := NewDuressAlarmDispatcher("production", ledger)
	dispatched := false
	dispatcher.AddListener(func(_ context.Context, _ OpsNotification) error {
		dispatched = true
		return nil
	})

	payload := DeviceSyncPayload{
		DeviceID: "dev_001", TenantID: "tenant-1", State: SyncStateHealthy,
	}
	if err := dispatcher.HandleSync(context.Background(), payload); err != nil {
		t.Fatal(err)
	}
	if dispatched || len(ledger.List("tenant-1")) != 0 {
		t.Fatal("healthy sync must not emit an alert or dispatch notifications")
	}
}

func TestDuressRejectsPayloadMissingDeviceIdentity(t *testing.T) {
	ledger := audit.NewLedger()
	dispatcher := NewDuressAlarmDispatcher("production", ledger)
	dispatched := false
	dispatcher.AddListener(func(_ context.Context, _ OpsNotification) error {
		dispatched = true
		return nil
	})

	payload := DeviceSyncPayload{TenantID: "tenant-1", State: SyncStateCoercionQuarantine}
	if err := dispatcher.HandleSync(context.Background(), payload); err == nil || !strings.Contains(err.Error(), "device_id") {
		t.Fatalf("payload without device_id must be refused, got %v", err)
	}
	if dispatched || len(ledger.List("tenant-1")) != 0 {
		t.Fatal("refused payload must not emit an alert or dispatch notifications")
	}
}

func TestDuressListenerFailureSurfacesButAuditPersists(t *testing.T) {
	ledger := audit.NewLedger()
	dispatcher := NewDuressAlarmDispatcher("production", ledger)
	dispatcher.AddListener(func(_ context.Context, _ OpsNotification) error {
		return errors.New("ops pager unreachable")
	})

	payload := DeviceSyncPayload{
		DeviceID: "dev_001", TenantID: "tenant-1", State: SyncStateCoercionQuarantine,
	}
	if err := dispatcher.HandleSync(context.Background(), payload); err == nil || !strings.Contains(err.Error(), "ops listener failed") {
		t.Fatalf("listener failure must surface to the ingress caller, got %v", err)
	}
	if records := ledger.List("tenant-1"); len(records) != 1 {
		t.Fatalf("audit alert must persist even when ops dispatch fails, got %d records", len(records))
	}
}
