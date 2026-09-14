// Package deviceenrollhttp exposes the production device enrollment and
// offline authority lifecycle as authenticated HTTP endpoints:
//
//	POST /v1/device-enrollment/requests                      issue challenge
//	POST /v1/device-enrollment/requests/{id}/proof           verify proof of possession
//	POST /v1/device-enrollment/requests/{id}/approve         tenant-admin approval + authority issuance
//	POST /v1/devices/{device_id}/revoke                      tenant-admin revocation
//
// Every endpoint requires an authenticated human session bound into the
// request context via oidchttp.WithOrganizationContext; tenant and
// organization scope is derived exclusively from that projection and any
// client-supplied scoping is rejected. Device keys are Ed25519, offer
// proof-of-possession through a single-use expiring challenge, and authority
// packages are HMAC-signed with the server signing secret.
package deviceenrollhttp

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"integin/internal/domain/device_trust"
	"integin/internal/identity"
	"integin/internal/oidchttp"
	"integin/internal/security"
	"integin/internal/shared/httpresponse"
	"integin/internal/shared/types"
	"integin/internal/syncstate"
)

const (
	protocolVersion = "v1"

	defaultChallengeTTL   = 5 * time.Minute
	defaultAuthorityTTL   = 8 * time.Hour
	defaultApprovalWindow = 24 * time.Hour
)

// defaultOfflineScopes are the capabilities granted by an issued offline
// authority package unless the caller overrides them in Config.Scopes.
var defaultOfflineScopes = []string{"work_package.read", "inspection.receipt.write"}

var (
	errUnauthenticated = errors.New("authentication_failed")
	errForbidden       = errors.New("forbidden")
)

// Config configures the enrollment handler. Now is injectable for hermetic
// tests; the zero value yields real time and production defaults.
type Config struct {
	// SigningSecret is the HMAC secret that signs issued authority packages.
	SigningSecret  string
	AuthorityTTL   time.Duration
	ChallengeTTL   time.Duration
	ApprovalWindow time.Duration
	Scopes         []string
	// Repo persists devices and authority packages. Nil uses an in-memory
	// store so the handler runs hermetically without any external daemon.
	Repo Repository
	Now  func() time.Time
}

// enrollmentRecord is the handler-local lifecycle state of one enrollment
// request. The domain EnrollmentRequest carries the tenant-bound identity
// and proof-of-possession; Challenge/ProofVerified track the two-phase
// challenge handshake that precedes admin approval.
type enrollmentRecord struct {
	Request          *device_trust.EnrollmentRequest
	Challenge        string
	ChallengeExpires time.Time
	ProofVerified    bool
	ApprovalDeadline time.Time
}

// Handler implements the production device enrollment and offline authority
// API. It is safe for concurrent use.
type Handler struct {
	mu             sync.Mutex
	now            func() time.Time
	secret         string
	challengeTTL   time.Duration
	authorityTTL   time.Duration
	approvalWindow time.Duration
	scopes         []string
	repo           Repository
	requests       map[string]*enrollmentRecord
}

// NewHandler validates configuration and returns the enrollment handler.
// An empty SigningSecret is tolerated for development but every issued
// authority package would carry a trivially forgeable signature, so callers
// must supply one before production use.
func NewHandler(cfg Config) (*Handler, error) {
	ttl := cfg.AuthorityTTL
	if ttl <= 0 {
		ttl = defaultAuthorityTTL
	}
	challengeTTL := cfg.ChallengeTTL
	if challengeTTL <= 0 {
		challengeTTL = defaultChallengeTTL
	}
	approvalWindow := cfg.ApprovalWindow
	if approvalWindow <= 0 {
		approvalWindow = defaultApprovalWindow
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	repo := cfg.Repo
	if repo == nil {
		repo = newInMemoryRepository()
	}
	scopes := append([]string(nil), cfg.Scopes...)
	if len(scopes) == 0 {
		scopes = append(scopes, defaultOfflineScopes...)
	}
	return &Handler{
		now:            now,
		secret:         cfg.SigningSecret,
		challengeTTL:   challengeTTL,
		authorityTTL:   ttl,
		approvalWindow: approvalWindow,
		scopes:         scopes,
		repo:           repo,
		requests:       make(map[string]*enrollmentRecord),
	}, nil
}

// Handler mounts the four enrollment routes. Requests traverse
// oidchttp.RejectTenantScopeConflict / SessionAuthenticator upstream so an
// OrganizationContext is bound; when it is absent the handler fails closed.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/device-enrollment/requests", h.createRequest)
	mux.HandleFunc("POST /v1/device-enrollment/requests/{request_id}/proof", h.submitProof)
	mux.HandleFunc("POST /v1/device-enrollment/requests/{request_id}/approve", h.approveRequest)
	mux.HandleFunc("POST /v1/devices/{device_id}/revoke", h.revokeDevice)
	return mux
}

// Device returns the stored device record for a tenant (nil repo access for
// tests and revocation auditing).
func (h *Handler) Device(ctx context.Context, tenantID, deviceID string) (syncstate.DeviceRecord, error) {
	return h.repo.GetDevice(ctx, tenantID, deviceID)
}

// Authorities returns the stored authority packages for a tenant/device.
func (h *Handler) Authorities(ctx context.Context, tenantID, deviceID string) ([]syncstate.AuthorityRecord, error) {
	return h.repo.ListAuthorities(ctx, tenantID, deviceID)
}

type createRequestBody struct {
	ProtocolVersion string `json:"protocol_version"`
	DevicePublicKey string `json:"device_public_key"`
	KeyID           string `json:"key_id"`

	// Client-supplied scoping is never honored; presence is rejected.
	TenantID       string `json:"tenant_id"`
	OrganizationID string `json:"organization_id"`
	Role           string `json:"role"`
	State          string `json:"state"`
}

// createRequest issues a single-use, short-lived challenge bound to the
// authenticated human session and a validated device public key.
func (h *Handler) createRequest(w http.ResponseWriter, r *http.Request) {
	org, err := humanSession(r)
	if err != nil {
		writeSessionFailure(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var body createRequestBody
	if err := httpresponse.ReadJSON(r, &body); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.ProtocolVersion != protocolVersion {
		httpresponse.Error(w, http.StatusBadRequest, "unsupported protocol_version")
		return
	}
	if body.TenantID != "" || body.OrganizationID != "" || body.Role != "" || body.State != "" {
		httpresponse.Error(w, http.StatusBadRequest, "tenant scope and role/state are server-derived and client-supplied values are rejected")
		return
	}
	publicKey, err := device_trust.DecodePublicKey(body.DevicePublicKey)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid device_public_key")
		return
	}
	if body.KeyID != security.DeviceKeyID(publicKey) {
		httpresponse.Error(w, http.StatusBadRequest, "key_id does not match device_public_key")
		return
	}

	requestID, err := newID("enr")
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "challenge issuance failed")
		return
	}
	nonceBytes := make([]byte, 32)
	if _, err := rand.Read(nonceBytes); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "challenge issuance failed")
		return
	}
	nonce := hex.EncodeToString(nonceBytes)

	now := h.now().UTC()
	expiresAt := now.Add(h.challengeTTL)
	h.mu.Lock()
	h.requests[requestKey(org.TenantID, requestID)] = &enrollmentRecord{
		Request: &device_trust.EnrollmentRequest{
			RequestID:      requestID,
			TenantID:       org.TenantID,
			OrganizationID: org.OrganizationID,
			UserID:         org.ActorID,
			PublicKey:      body.DevicePublicKey,
			Status:         device_trust.EnrollmentPending,
			RequestedAt:    now,
		},
		Challenge:        nonce,
		ChallengeExpires: expiresAt,
	}
	h.mu.Unlock()

	httpresponse.JSON(w, http.StatusCreated, map[string]string{
		"request_id": requestID,
		"challenge":  nonce,
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

type proofRequestBody struct {
	DevicePublicKey string `json:"device_public_key"`
	Signature       string `json:"signature"`
}

// submitProof consumes the single-use challenge by verifying the device's
// Ed25519 signature over the challenge nonce, then promotes the request to a
// proof-verified PENDING enrollment awaiting tenant-admin approval.
func (h *Handler) submitProof(w http.ResponseWriter, r *http.Request) {
	org, err := humanSession(r)
	if err != nil {
		writeSessionFailure(w, err)
		return
	}
	requestID := r.PathValue("request_id")
	if requestID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "request_id is required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var body proofRequestBody
	if err := httpresponse.ReadJSON(r, &body); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	submittedPublicKey, err := device_trust.DecodePublicKey(body.DevicePublicKey)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid device_public_key")
		return
	}

	h.mu.Lock()
	record := h.requests[requestKey(org.TenantID, requestID)]
	h.mu.Unlock()
	if record == nil {
		httpresponse.Error(w, http.StatusNotFound, "enrollment request not found")
		return
	}
	if record.ProofVerified {
		httpresponse.Error(w, http.StatusGone, "challenge already consumed")
		return
	}
	if h.now().UTC().After(record.ChallengeExpires) {
		httpresponse.Error(w, http.StatusBadRequest, "challenge expired")
		return
	}
	boundPublicKey, err := device_trust.DecodePublicKey(record.Request.PublicKey)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "stored public key is invalid")
		return
	}
	if !bytes.Equal(submittedPublicKey, boundPublicKey) {
		httpresponse.Error(w, http.StatusForbidden, "public key does not match enrollment request")
		return
	}
	signature, err := decodeSignature(body.Signature)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid signature")
		return
	}
	if !security.VerifyDeviceMutation(submittedPublicKey, []byte(record.Challenge), signature) {
		httpresponse.Error(w, http.StatusBadRequest, "proof of possession failed")
		return
	}

	deviceID := deriveDeviceID(submittedPublicKey)
	request, err := device_trust.CreateEnrollmentRequest(
		record.Request.RequestID, record.Request.TenantID, record.Request.OrganizationID,
		record.Request.UserID, deviceID, record.Request.PublicKey,
		record.Challenge, body.Signature, device_trust.EnrollmentAttestation{}, h.now().UTC(),
	)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "enrollment request validation failed")
		return
	}

	h.mu.Lock()
	deadline := h.now().UTC().Add(h.approvalWindow)
	if existing := h.requests[requestKey(org.TenantID, requestID)]; existing != nil && existing.ProofVerified {
		h.mu.Unlock()
		httpresponse.Error(w, http.StatusGone, "challenge already consumed")
		return
	}
	h.requests[requestKey(org.TenantID, requestID)] = &enrollmentRecord{
		Request:          request,
		Challenge:        record.Challenge,
		ChallengeExpires: record.ChallengeExpires,
		ProofVerified:    true,
		ApprovalDeadline: deadline,
	}
	h.mu.Unlock()

	httpresponse.JSON(w, http.StatusOK, map[string]string{
		"request_id": requestID,
		"status":     "proof_verified",
	})
}

// approveRequest is the tenant-admin gate: it validates the approval window,
// rejects self-approval, promotes the enrollment into a trusted device, and
// issues the signed offline authority package (default 8h TTL).
func (h *Handler) approveRequest(w http.ResponseWriter, r *http.Request) {
	org, err := humanSession(r)
	if err != nil {
		writeSessionFailure(w, err)
		return
	}
	if !hasTenantAdmin(org) {
		httpresponse.Error(w, http.StatusForbidden, "tenant administrator capability required")
		return
	}
	requestID := r.PathValue("request_id")
	if requestID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "request_id is required")
		return
	}

	key := requestKey(org.TenantID, requestID)
	h.mu.Lock()
	record := h.requests[key]
	h.mu.Unlock()
	if record == nil {
		httpresponse.Error(w, http.StatusNotFound, "enrollment request not found")
		return
	}
	if !record.ProofVerified {
		httpresponse.Error(w, http.StatusBadRequest, "enrollment request has not verified proof of possession")
		return
	}
	if !record.ApprovalDeadline.IsZero() && h.now().UTC().After(record.ApprovalDeadline) {
		httpresponse.Error(w, http.StatusBadRequest, "approval window expired")
		return
	}
	if record.Request.UserID == org.ActorID {
		httpresponse.Error(w, http.StatusForbidden, "self-approval is not permitted")
		return
	}
	// The domain approval mutates the shared EnrollmentRequest status in
	// place, so it runs under the handler mutex: concurrent approvals of the
	// same request serialize and the second one fails closed on a 409.
	h.mu.Lock()
	live := h.requests[key]
	if live != record {
		h.mu.Unlock()
		httpresponse.Error(w, http.StatusConflict, "enrollment request changed during approval")
		return
	}
	device, err := device_trust.ApproveEnrollment(record.Request, org.ActorID)
	h.mu.Unlock()
	if err != nil {
		httpresponse.Error(w, http.StatusConflict, err.Error())
		return
	}
	now := h.now().UTC()

	publicKeyBytes, err := device_trust.DecodePublicKey(device.PublicKey())
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "device public key is invalid")
		return
	}
	if err := h.repo.SaveDevice(r.Context(), syncstate.DeviceRecord{
		DeviceID:       device.ID(),
		TenantID:       device.TenantID(),
		OrganizationID: device.OrganizationID(),
		UserID:         device.UserID(),
		KeyID:          security.DeviceKeyID(publicKeyBytes),
		PublicKey:      publicKeyBytes,
		State:          syncstate.DeviceTrusted,
		AuthorityEpoch: device.Epoch(),
		EnrolledAt:     now,
		UpdatedAt:      now,
	}); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "failed to persist trusted device")
		return
	}

	packageID, err := newID("aut")
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "authority issuance failed")
		return
	}
	authority, err := device_trust.IssueAuthorityPackage(*device, packageID, h.secret, h.scopes, now, h.authorityTTL)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.repo.SaveAuthority(r.Context(), syncstate.AuthorityRecord{
		AuthorityID:      authority.ID,
		TenantID:         authority.TenantID,
		OrganizationID:   device.OrganizationID(),
		DeviceID:         authority.DeviceID,
		UserID:           authority.UserID,
		AuthorityEpoch:   authority.Epoch,
		Scopes:           authority.Scopes,
		ProcedureVersion: protocolVersion,
		IssuedAt:         authority.IssuedAt,
		ExpiresAt:        authority.ExpiresAt,
		Signature:        []byte(authority.Signature),
	}); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "failed to persist authority package")
		return
	}

	httpresponse.JSON(w, http.StatusOK, map[string]any{"authority_package": authority})
}

type revokeRequestBody struct {
	Reason string `json:"reason"`
}

// revokeDevice revokes a trusted device: the domain device is transitioned to
// REVOKED (bumping its authority epoch), every active authority package is
// revoked, and the explicit reason is recorded.
func (h *Handler) revokeDevice(w http.ResponseWriter, r *http.Request) {
	org, err := humanSession(r)
	if err != nil {
		writeSessionFailure(w, err)
		return
	}
	if !hasTenantAdmin(org) {
		httpresponse.Error(w, http.StatusForbidden, "tenant administrator capability required")
		return
	}
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "device_id is required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var body revokeRequestBody
	if err := httpresponse.ReadJSON(r, &body); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(body.Reason) == "" {
		httpresponse.Error(w, http.StatusBadRequest, "revocation reason is required")
		return
	}

	deviceRecord, err := h.repo.GetDevice(r.Context(), org.TenantID, deviceID)
	if err != nil {
		if errors.Is(err, syncstate.ErrNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "device not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "failed to load device")
		return
	}
	device, err := device_trust.RestoreDevice(
		deviceRecord.DeviceID, deviceRecord.TenantID, deviceRecord.OrganizationID,
		deviceRecord.UserID, hex.EncodeToString(deviceRecord.PublicKey),
		types.DeviceTrustState(deviceRecord.State), deviceRecord.AuthorityEpoch,
	)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "failed to restore device trust state")
		return
	}
	if err := device.Revoke(body.Reason); err != nil {
		httpresponse.Error(w, http.StatusConflict, err.Error())
		return
	}
	now := h.now().UTC()
	deviceRecord.State = syncstate.DeviceRevoked
	deviceRecord.AuthorityEpoch = device.Epoch()
	deviceRecord.RevokedAt = &now
	deviceRecord.RevocationReason = body.Reason
	deviceRecord.UpdatedAt = now
	if err := h.repo.SaveDevice(r.Context(), deviceRecord); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "failed to persist revoked device")
		return
	}

	authorities, err := h.repo.ListAuthorities(r.Context(), org.TenantID, deviceID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "failed to list authority packages")
		return
	}
	for _, authority := range authorities {
		if authority.RevokedAt != nil {
			continue
		}
		if err := h.repo.RevokeAuthority(r.Context(), org.TenantID, authority.AuthorityID, now); err != nil {
			httpresponse.Error(w, http.StatusInternalServerError, "failed to revoke authority package")
			return
		}
	}

	httpresponse.JSON(w, http.StatusOK, map[string]any{
		"device_id":       deviceID,
		"status":          "REVOKED",
		"authority_epoch": device.Epoch(),
	})
}

// humanSession extracts the authenticated human organization context. Any
// other principal type or missing context fails closed.
func humanSession(r *http.Request) (identity.OrganizationContext, error) {
	org, ok := oidchttp.OrganizationContextFrom(r.Context())
	if !ok {
		return identity.OrganizationContext{}, errUnauthenticated
	}
	if org.PrincipalType != identity.PrincipalTypeHuman {
		return identity.OrganizationContext{}, errForbidden
	}
	return org, nil
}

func writeSessionFailure(w http.ResponseWriter, err error) {
	if errors.Is(err, errUnauthenticated) {
		httpresponse.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	httpresponse.Error(w, http.StatusForbidden, err.Error())
}

// hasTenantAdmin reports whether the org context carries the Tenant
// Administrator role or an "admin" capability.
func hasTenantAdmin(org identity.OrganizationContext) bool {
	for _, capability := range org.Capabilities {
		if strings.EqualFold(capability, "admin") {
			return true
		}
	}
	for _, role := range org.Roles {
		if strings.EqualFold(role, "Tenant Administrator") {
			return true
		}
	}
	return false
}

// requestKey scopes a request ID under its tenant so cross-tenant collisions
// can never resolve.
func requestKey(tenantID, requestID string) string {
	return tenantID + "/" + requestID
}

// deriveDeviceID deterministically binds the device identity to its signing
// key so a spoofed device id cannot be substituted for another key.
func deriveDeviceID(publicKey ed25519.PublicKey) string {
	return "dev_" + hex.EncodeToString(publicKey[:6])
}

// decodeSignature decodes a 64-byte Ed25519 signature provided as hex or
// base64, failing closed on ambiguity.
func decodeSignature(value string) ([]byte, error) {
	if strings.TrimSpace(value) == "" {
		return nil, errors.New("empty signature")
	}
	if b, err := hex.DecodeString(value); err == nil && len(b) == ed25519.SignatureSize {
		return b, nil
	}
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := encoding.DecodeString(value); err == nil && len(b) == ed25519.SignatureSize {
			return b, nil
		}
	}
	return nil, fmt.Errorf("expected %d bytes encoded as hex or base64", ed25519.SignatureSize)
}

// newID returns a short random identifier with the given prefix.
func newID(prefix string) (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(bytes), nil
}
