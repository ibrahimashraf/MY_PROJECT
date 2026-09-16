// Package localprovision provides a deliberately constrained development-only
// device enrollment bridge. It is enabled only by explicit runtime configuration
// and accepts requests from loopback clients only.
package localprovision

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"integin/internal/domain/device_trust"
	domainsync "integin/internal/domain/sync"
	"integin/internal/security"
	"integin/internal/shared/types"
	"integin/internal/syncstate"
)

type repository interface {
	SaveDevice(context.Context, syncstate.DeviceRecord) error
	SaveAuthority(context.Context, syncstate.AuthorityRecord) error
}

type Config struct {
	Repository        repository
	Processor         *domainsync.Processor
	SigningSecret     string
	TenantID          string
	OrganizationID    string
	UserID            string
	AuthorityLifetime time.Duration
	Now               func() time.Time
}

type Handler struct {
	config            Config
	mu                sync.RWMutex
	registerAuthority func(device_trust.AuthorityPackage)
}

type request struct {
	DeviceID  string `json:"device_id"`
	KeyID     string `json:"key_id"`
	PublicKey string `json:"public_key"`
}

type authorityResponse struct {
	AuthorityID      string    `json:"authority_id"`
	AuthorityEpoch   uint64    `json:"authority_epoch"`
	Scopes           []string  `json:"scopes"`
	ProcedureVersion string    `json:"procedure_version"`
	IssuedAt         time.Time `json:"issued_at"`
	ExpiresAt        time.Time `json:"expires_at"`
	Signature        string    `json:"signature"`
}

type response struct {
	TenantID       string            `json:"tenant_id"`
	OrganizationID string            `json:"organization_id"`
	Environment    string            `json:"environment"`
	DeviceID       string            `json:"device_id"`
	UserID         string            `json:"user_id"`
	Authority      authorityResponse `json:"authority"`
}

func NewHandler(config Config) (*Handler, error) {
	if config.Repository == nil || config.Processor == nil || strings.TrimSpace(config.SigningSecret) == "" {
		return nil, errors.New("local provisioning requires repository, processor, and signing secret")
	}
	if strings.TrimSpace(config.TenantID) == "" || strings.TrimSpace(config.OrganizationID) == "" || strings.TrimSpace(config.UserID) == "" {
		return nil, errors.New("local provisioning tenant, organization, and user are required")
	}
	if config.AuthorityLifetime <= 0 || config.AuthorityLifetime > time.Hour {
		return nil, errors.New("local provisioning authority lifetime must be between zero and one hour")
	}
	if config.Now == nil {
		config.Now = func() time.Time { return time.Now().UTC() }
	}
	return &Handler{config: config}, nil
}

// SetAuthorityRegistrar connects the provisioner to the mux-owned authority
// registry before any request is served, avoiding a server restart after enrollment.
func (h *Handler) SetAuthorityRegistrar(registrar func(device_trust.AuthorityPackage)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.registerAuthority = registrar
}

func (h *Handler) ServeHTTP(writer http.ResponseWriter, httpRequest *http.Request) {
	if httpRequest.Method != http.MethodPost {
		writeError(writer, http.StatusMethodNotAllowed, "POST is required")
		return
	}
	if !isLoopback(httpRequest.RemoteAddr) {
		writeError(writer, http.StatusForbidden, "local provisioning accepts loopback clients only")
		return
	}
	h.mu.RLock()
	registrar := h.registerAuthority
	h.mu.RUnlock()
	if registrar == nil {
		writeError(writer, http.StatusServiceUnavailable, "local provisioning authority registry is unavailable")
		return
	}
	httpRequest.Body = http.MaxBytesReader(writer, httpRequest.Body, 1<<20)
	var incoming request
	decoder := json.NewDecoder(httpRequest.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&incoming); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(writer, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if !validDeviceID(incoming.DeviceID) || strings.TrimSpace(incoming.KeyID) == "" {
		writeError(writer, http.StatusBadRequest, "valid device_id and key_id are required")
		return
	}
	publicKey, err := base64.StdEncoding.DecodeString(incoming.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		writeError(writer, http.StatusBadRequest, "public_key must be an Ed25519 base64 value")
		return
	}
	if incoming.KeyID != security.DeviceKeyID(ed25519.PublicKey(publicKey)) {
		writeError(writer, http.StatusBadRequest, "key_id does not match public_key")
		return
	}
	now := h.config.Now().UTC().Truncate(time.Microsecond)
	device, err := device_trust.RestoreDevice(incoming.DeviceID, h.config.TenantID, h.config.OrganizationID, h.config.UserID, incoming.PublicKey, types.DeviceTrusted, 1)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid device identity")
		return
	}
	if err := h.config.Repository.SaveDevice(httpRequest.Context(), syncstate.DeviceRecord{
		DeviceID: incoming.DeviceID, TenantID: h.config.TenantID, OrganizationID: h.config.OrganizationID, UserID: h.config.UserID,
		KeyID: incoming.KeyID, PublicKey: publicKey, State: syncstate.DeviceTrusted, AuthorityEpoch: 1, EnrolledAt: now, UpdatedAt: now,
	}); err != nil {
		writeError(writer, http.StatusServiceUnavailable, "unable to save provisioned device")
		return
	}
	authority, err := device_trust.IssueAuthorityPackage(device, newAuthorityID(), h.config.SigningSecret, []string{"inspection.perform", "evidence.upload"}, now, h.config.AuthorityLifetime)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "unable to issue authority")
		return
	}
	if err := h.config.Repository.SaveAuthority(httpRequest.Context(), syncstate.AuthorityRecord{
		AuthorityID: authority.ID, TenantID: authority.TenantID, OrganizationID: h.config.OrganizationID, DeviceID: authority.DeviceID, UserID: authority.UserID,
		AuthorityEpoch: authority.Epoch, Scopes: authority.Scopes, ProcedureVersion: "integin-local-provision-v1", IssuedAt: authority.IssuedAt, ExpiresAt: authority.ExpiresAt, Signature: []byte(authority.Signature),
	}); err != nil {
		writeError(writer, http.StatusServiceUnavailable, "unable to save authority")
		return
	}
	h.config.Processor.RegisterDevice(device)
	registrar(authority)
	writeJSON(writer, http.StatusOK, response{
		TenantID: h.config.TenantID, OrganizationID: h.config.OrganizationID, Environment: "LIVE", DeviceID: incoming.DeviceID, UserID: h.config.UserID,
		Authority: authorityResponse{AuthorityID: authority.ID, AuthorityEpoch: authority.Epoch, Scopes: authority.Scopes, ProcedureVersion: "integin-local-provision-v1", IssuedAt: authority.IssuedAt, ExpiresAt: authority.ExpiresAt, Signature: authority.Signature},
	})
}

func isLoopback(remoteAddress string) bool {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		host = remoteAddress
	}
	host = strings.Trim(strings.TrimSpace(host), "[]")
	return host == "localhost" || (net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback())
}

func validDeviceID(value string) bool {
	if !strings.HasPrefix(value, "field-") || len(value) > 96 {
		return false
	}
	for _, character := range value {
		if !(character >= 'a' && character <= 'z') && !(character >= '0' && character <= '9') && character != '-' {
			return false
		}
	}
	return true
}

func newAuthorityID() string {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "local-authority-" + time.Now().UTC().Format("20060102T150405.000000000")
	}
	return "local-authority-" + hex.EncodeToString(bytes[:])
}

func writeError(writer http.ResponseWriter, status int, reason string) {
	writeJSON(writer, status, map[string]string{"reason": reason})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
