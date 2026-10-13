package device_trust

import (
	"testing"
	"time"

	"integin/internal/shared/events"
	"integin/internal/shared/types"
)

func newTestDevice(t *testing.T) Device {
	t.Helper()
	device, err := NewDevice("device-1", "tenant-1", "org-1", "user-1", "public-key")
	if err != nil {
		t.Fatal(err)
	}
	return device
}

func TestDeviceTrustLifecycle(t *testing.T) {
	device := newTestDevice(t)
	if device.State() != types.DevicePending || device.Epoch() != 1 {
		t.Fatal("device should start pending at epoch 1")
	}
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	if err := device.Restrict(); err != nil {
		t.Fatal(err)
	}
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	if err := device.Lock(); err != nil {
		t.Fatal(err)
	}
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	if err := device.Retire(); err != nil {
		t.Fatal(err)
	}
	if device.State() != types.DeviceRetired {
		t.Fatal("device should be retired")
	}
	if len(device.Events()) != 6 {
		t.Fatalf("expected six state events, got %d", len(device.Events()))
	}
}

func TestAuthorityPackageIsSignedBoundedAndEpochScoped(t *testing.T) {
	device := newTestDevice(t)
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	issuedAt := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	authority, err := IssueAuthorityPackage(device, "authority-1", "secret", []string{"inspection-1"}, issuedAt, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateAuthorityPackage(authority, device, "secret", issuedAt.Add(30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAuthorityPackage(authority, device, "wrong-secret", issuedAt.Add(30*time.Minute)); err == nil {
		t.Fatal("wrong secret should fail")
	}
	if err := ValidateAuthorityPackage(authority, device, "secret", issuedAt.Add(time.Hour)); err == nil {
		t.Fatal("expiry boundary should fail")
	}
	if err := device.Revoke("lost device"); err != nil {
		t.Fatal(err)
	}
	if device.Epoch() != 2 || device.State() != types.DeviceRevoked {
		t.Fatal("revocation should increment epoch and revoke device")
	}
	if err := ValidateAuthorityPackage(authority, device, "secret", issuedAt.Add(30*time.Minute)); err == nil {
		t.Fatal("revoked device should invalidate authority")
	}
	if len(device.Events()) != 2 || device.Events()[1].EventType() != events.DeviceRevoked {
		t.Fatal("revocation event missing")
	}
}

func TestDeviceRejectsInvalidTransitionsAndAuthorityIssuance(t *testing.T) {
	device := newTestDevice(t)
	if err := device.Revoke(""); err == nil {
		t.Fatal("empty revocation reason should fail")
	}
	if _, err := IssueAuthorityPackage(device, "authority-1", "secret", nil, time.Now(), time.Hour); err == nil {
		t.Fatal("pending device should not receive authority")
	}
	if err := device.Retire(); err == nil {
		t.Fatal("pending device should not retire")
	}
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	if err := device.Revoke("lost device"); err != nil {
		t.Fatal(err)
	}
	if err := device.Trust(); err == nil {
		t.Fatal("revoked device should not become trusted")
	}
}
