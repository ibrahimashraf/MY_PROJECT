package device_trust

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"integin/internal/shared/events"
	"integin/internal/shared/types"
)

type Device struct {
	id             string
	tenantID       string
	organizationID string
	userID         string
	publicKey      string
	state          types.DeviceTrustState
	epoch          uint64
	emitted        []events.Envelope
}

func NewDevice(id, tenantID, organizationID, userID, publicKey string) (Device, error) {
	for field, value := range map[string]string{"id": id, "tenant_id": tenantID, "organization_id": organizationID, "user_id": userID, "public_key": publicKey} {
		if strings.TrimSpace(value) == "" {
			return Device{}, fmt.Errorf("%s is required", field)
		}
	}
	return Device{id: id, tenantID: tenantID, organizationID: organizationID, userID: userID, publicKey: publicKey, state: types.DevicePending, epoch: 1}, nil
}

// RestoreDevice reconstructs an already-enrolled device from authoritative
// persistence. It deliberately emits no transition event: restoration is
// startup hydration, not a new trust decision.
func RestoreDevice(id, tenantID, organizationID, userID, publicKey string, state types.DeviceTrustState, epoch uint64) (Device, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(tenantID) == "" || strings.TrimSpace(organizationID) == "" || strings.TrimSpace(userID) == "" || strings.TrimSpace(publicKey) == "" {
		return Device{}, errors.New("device identity and public key are required")
	}
	if epoch == 0 {
		return Device{}, errors.New("device authority epoch must be positive")
	}
	switch state {
	case types.DevicePending, types.DeviceTrusted, types.DeviceRestricted, types.DeviceLocked, types.DeviceRevoked, types.DeviceRetired:
	default:
		return Device{}, fmt.Errorf("invalid device trust state: %s", state)
	}
	return Device{id: id, tenantID: tenantID, organizationID: organizationID, userID: userID, publicKey: publicKey, state: state, epoch: epoch}, nil
}

func (d Device) ID() string                    { return d.id }
func (d Device) TenantID() string              { return d.tenantID }
func (d Device) OrganizationID() string        { return d.organizationID }
func (d Device) UserID() string                { return d.userID }
func (d Device) PublicKey() string             { return d.publicKey }
func (d Device) State() types.DeviceTrustState { return d.state }
func (d Device) Epoch() uint64                 { return d.epoch }
func (d Device) Events() []events.Envelope     { return append([]events.Envelope(nil), d.emitted...) }

func (d *Device) Trust() error {
	return d.transition(types.DeviceTrusted, types.DevicePending, types.DeviceRestricted, types.DeviceLocked)
}
func (d *Device) Restrict() error { return d.transition(types.DeviceRestricted, types.DeviceTrusted) }
func (d *Device) Lock() error {
	return d.transition(types.DeviceLocked, types.DeviceTrusted, types.DeviceRestricted)
}
func (d *Device) Retire() error {
	return d.transition(types.DeviceRetired, types.DeviceTrusted, types.DeviceRestricted, types.DeviceLocked)
}
func (d *Device) Revoke(reason string) error {
	if d.state == types.DeviceRevoked || d.state == types.DeviceRetired {
		return fmt.Errorf("cannot revoke device from state %s", d.state)
	}
	if strings.TrimSpace(reason) == "" {
		return errors.New("revocation reason is required")
	}
	d.epoch++
	event, err := d.makeEvent(events.DeviceRevoked, map[string]any{"reason": reason, "epoch": d.epoch})
	if err != nil {
		d.epoch--
		return err
	}
	d.state = types.DeviceRevoked
	d.emitted = append(d.emitted, event)
	return nil
}

func (d *Device) transition(target types.DeviceTrustState, allowed ...types.DeviceTrustState) error {
	valid := false
	for _, state := range allowed {
		if d.state == state {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid device trust transition from %s to %s", d.state, target)
	}
	event, err := d.makeEvent("DeviceTrustStateChanged", map[string]string{"from": string(d.state), "to": string(target)})
	if err != nil {
		return err
	}
	d.state = target
	d.emitted = append(d.emitted, event)
	return nil
}

func (d Device) makeEvent(eventType string, payload any) (events.Envelope, error) {
	eventID := fmt.Sprintf("%s-%d-%d", d.id, d.epoch, len(d.emitted)+1)
	return events.NewEnvelope(eventID, eventType, d.tenantID, d.organizationID, "LIVE", "device", d.id, payload, time.Now().UTC())
}

type AuthorityPackage struct {
	ID        string    `json:"id"`
	DeviceID  string    `json:"device_id"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Epoch     uint64    `json:"epoch"`
	Scopes    []string  `json:"scopes"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Signature string    `json:"signature"`
}

func IssueAuthorityPackage(device Device, id, secret string, scopes []string, issuedAt time.Time, ttl time.Duration) (AuthorityPackage, error) {
	if device.state != types.DeviceTrusted {
		return AuthorityPackage{}, fmt.Errorf("device must be trusted, got %s", device.state)
	}
	if strings.TrimSpace(id) == "" || strings.TrimSpace(secret) == "" {
		return AuthorityPackage{}, errors.New("package id and signing secret are required")
	}
	if issuedAt.IsZero() || ttl <= 0 {
		return AuthorityPackage{}, errors.New("issued_at and positive ttl are required")
	}
	packageValue := AuthorityPackage{ID: id, DeviceID: device.id, TenantID: device.tenantID, UserID: device.userID, Epoch: device.epoch, Scopes: append([]string(nil), scopes...), IssuedAt: issuedAt.UTC(), ExpiresAt: issuedAt.UTC().Add(ttl)}
	packageValue.Signature = sign(packageValue, secret)
	return packageValue, nil
}

func ValidateAuthorityPackage(packageValue AuthorityPackage, device Device, secret string, at time.Time) error {
	if strings.TrimSpace(secret) == "" {
		return errors.New("signing secret is required")
	}
	if packageValue.DeviceID != device.id || packageValue.TenantID != device.tenantID || packageValue.UserID != device.userID {
		return errors.New("authority package identity mismatch")
	}
	if packageValue.Epoch != device.epoch {
		return errors.New("authority package epoch is stale")
	}
	if device.state != types.DeviceTrusted {
		return fmt.Errorf("device is not trusted: %s", device.state)
	}
	if at.Before(packageValue.IssuedAt) || !at.Before(packageValue.ExpiresAt) {
		return errors.New("authority package is expired or not yet valid")
	}
	if !hmac.Equal([]byte(packageValue.Signature), []byte(sign(packageValue, secret))) {
		return errors.New("authority package signature is invalid")
	}
	return nil
}

func sign(packageValue AuthorityPackage, secret string) string {
	canonical := fmt.Sprintf("%s|%s|%s|%s|%d|%s|%s|%s", packageValue.ID, packageValue.DeviceID, packageValue.TenantID, packageValue.UserID, packageValue.Epoch, strings.Join(packageValue.Scopes, ","), packageValue.IssuedAt.UTC().Format(time.RFC3339Nano), packageValue.ExpiresAt.UTC().Format(time.RFC3339Nano))
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}
