// Package packagemanifestapi provides an intentionally unmounted manifest-read boundary.
package packagemanifestapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"integin/internal/domain/device_trust"
	domainsync "integin/internal/domain/sync"
	"integin/internal/packagemanifest"
)

const maxRequestBytes = 16 << 10

// DeviceProofVerifier permits the real sync processor and deterministic tests to share this boundary.
type DeviceProofVerifier interface {
	VerifyDeviceProof(ctx context.Context, proof domainsync.DeviceProof, authority device_trust.AuthorityPackage, at time.Time) (domainsync.VerifiedDeviceContext, error)
}

// AuthorityLookup is satisfied by the existing syncapi.AuthorityRegistry.
// It is deliberately read-only so a manifest request cannot alter authority state.
type AuthorityLookup interface {
	Get(id string) (device_trust.AuthorityPackage, bool)
}

// Handler verifies a device proof, atomically consumes its request ID, and issues a manifest.
// It is unmounted by design: no server router or main package registers this handler.
type Handler struct {
	Verifier      DeviceProofVerifier
	Authorities   AuthorityLookup
	ReplayStore   packagemanifest.ProofReplayStore
	ReplayCounter packagemanifest.ProofReplayCounter
	Issuer        *packagemanifest.ManifestIssuer
	Now           func() time.Time
	Observer      ObservationSink
}

// request accepts only a device proof. Tenant, organization, user, and package scope are never client input.
type request struct {
	Proof domainsync.DeviceProof `json:"proof"`
}

// ServeHTTP handles a future manifest-read request without adding a route registration.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.observe(r.Context(), "request_rejected", "unsupported_method", http.StatusMethodNotAllowed, false)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h == nil || h.Verifier == nil || h.Authorities == nil || h.ReplayStore == nil || h.Issuer == nil {
		h.observe(r.Context(), "request_rejected", "unconfigured", http.StatusServiceUnavailable, false)
		writeError(w, http.StatusServiceUnavailable, "manifest service is not configured")
		return
	}

	var body request
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		h.observe(r.Context(), "request_rejected", "invalid_request", http.StatusBadRequest, false)
		writeError(w, http.StatusBadRequest, "invalid manifest request")
		return
	}
	if strings.TrimSpace(body.Proof.AuthorityID) == "" {
		h.observe(r.Context(), "proof_rejected", "authority_mismatch", http.StatusUnauthorized, false)
		writeError(w, http.StatusUnauthorized, "manifest authentication failed")
		return
	}
	authority, ok := h.Authorities.Get(body.Proof.AuthorityID)
	if !ok {
		h.observe(r.Context(), "proof_rejected", "authority_mismatch", http.StatusUnauthorized, false)
		writeError(w, http.StatusUnauthorized, "manifest authentication failed")
		return
	}
	now := time.Now().UTC()
	if h.Now != nil {
		now = h.Now().UTC()
	}
	verified, err := h.Verifier.VerifyDeviceProof(r.Context(), body.Proof, authority, now)
	if err != nil {
		h.observe(r.Context(), "proof_rejected", proofFailureReason(err), http.StatusUnauthorized, false)
		writeError(w, http.StatusUnauthorized, "manifest authentication failed")
		return
	}
	manifest, err := h.Issuer.Issue(r.Context(), verified, body.Proof.InspectionID, now)
	if err != nil {
		if errors.Is(err, packagemanifest.ErrNotFound) {
			h.observe(r.Context(), "request_rejected", "assignment_not_found", http.StatusNotFound, true)
			writeError(w, http.StatusNotFound, "approved work package assignment was not found")
			return
		}
		if errors.Is(err, packagemanifest.ErrExpired) || errors.Is(err, packagemanifest.ErrScopeMismatch) || errors.Is(err, packagemanifest.ErrIntegrity) {
			reason := "package_hash_invalid"
			if errors.Is(err, packagemanifest.ErrExpired) {
				reason = "expired"
			} else if errors.Is(err, packagemanifest.ErrScopeMismatch) {
				reason = "authority_mismatch"
			}
			h.observe(r.Context(), "proof_rejected", reason, http.StatusForbidden, false)
			writeError(w, http.StatusForbidden, "approved work package assignment is unavailable")
			return
		}
		h.observe(r.Context(), "request_rejected", "issuance_failed", http.StatusInternalServerError, false)
		writeError(w, http.StatusInternalServerError, "manifest issuance failed")
		return
	}
	var replayBefore, replayAfter *int64
	if h.Observer != nil && h.ReplayCounter != nil {
		if before, countErr := h.ReplayCounter.Count(r.Context(), verified.TenantID, verified.OrganizationID, verified.DeviceID, body.Proof.Purpose, now); countErr == nil {
			replayBefore = &before
		}
	}
	if err := h.ReplayStore.Consume(r.Context(), verified.TenantID, verified.OrganizationID, verified.DeviceID, body.Proof.Purpose, body.Proof.RequestID, body.Proof.ExpiresAt); err != nil {
		if errors.Is(err, packagemanifest.ErrReplayAlreadyConsumed) {
			h.observe(r.Context(), "replay_rejected", "replay", http.StatusConflict, false)
			writeError(w, http.StatusConflict, "manifest proof was already used")
			return
		}
		h.observe(r.Context(), "proof_rejected", "expired", http.StatusUnauthorized, false)
		writeError(w, http.StatusUnauthorized, "manifest authentication failed")
		return
	}
	if replayBefore != nil && h.ReplayCounter != nil {
		if after, countErr := h.ReplayCounter.Count(r.Context(), verified.TenantID, verified.OrganizationID, verified.DeviceID, body.Proof.Purpose, now); countErr == nil {
			replayAfter = &after
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(manifest)
	h.observeWithReplay(r.Context(), "manifest_issued", "valid_proof", http.StatusOK, true, replayBefore, replayAfter)
}

func (h *Handler) observe(ctx context.Context, outcome, reason string, status int, replayAccepted bool) {
	h.observeWithReplay(ctx, outcome, reason, status, replayAccepted, nil, nil)
}

func (h *Handler) observeWithReplay(ctx context.Context, outcome, reason string, status int, replayAccepted bool, replayBefore, replayAfter *int64) {
	if h == nil || h.Observer == nil {
		return
	}
	h.Observer.Observe(ctx, Observation{Outcome: outcome, ReasonCode: reason, HTTPStatus: status, ReplayAccepted: replayAccepted, ReplayBefore: replayBefore, ReplayAfter: replayAfter})
}

func proofFailureReason(err error) string {
	if reason, ok := domainsync.ProofFailureReasonOf(err); ok {
		return string(reason)
	}
	return "proof_invalid"
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
