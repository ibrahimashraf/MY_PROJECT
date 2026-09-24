package syncapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"integin/internal/domain/device_trust"
	domainsync "integin/internal/domain/sync"
	"integin/internal/oidchttp"
	"integin/internal/workpackageenforcement"
)

type request struct {
	ProtocolVersion    string          `json:"protocol_version"`
	TransactionID      string          `json:"transaction_id"`
	TenantID           string          `json:"tenant_id"`
	OrganizationID     string          `json:"organization_id"`
	Environment        string          `json:"environment"`
	DeviceID           string          `json:"device_id"`
	UserID             string          `json:"user_id"`
	SequenceNumber     uint64          `json:"sequence_number"`
	Operation          string          `json:"operation"`
	EntityID           string          `json:"entity_id"`
	Payload            json.RawMessage `json:"payload"`
	PayloadHash        string          `json:"payload_hash"`
	CapturedAt         time.Time       `json:"captured_at"`
	AuthorityID        string          `json:"authority_id"`
	AuthorityEpoch     uint64          `json:"authority_epoch"`
	SignatureAlgorithm string          `json:"signature_algorithm"`
	KeyID              string          `json:"key_id"`
	Signature          string          `json:"signature"`
}

type response struct {
	Outcome          domainsync.Outcome `json:"outcome"`
	Reason           string             `json:"reason,omitempty"`
	TransactionID    string             `json:"transaction_id,omitempty"`
	ExpectedSequence uint64             `json:"expected_sequence,omitempty"`
	PayloadHash      string             `json:"payload_hash,omitempty"`
}

// Handler adapts the transport-neutral field-app envelope to the authoritative
// Go sync processor. Authority packages are resolved server-side and are never
// accepted from client-provided authority fields.
type Handler struct {
	Processor        *domainsync.Processor
	Authorities      *AuthorityRegistry
	PackageValidator *workpackageenforcement.Validator
	Now              func() time.Time
}

type AuthorityRegistry struct {
	mu     sync.RWMutex
	values map[string]device_trust.AuthorityPackage
	// revoked tombstones authority ids withdrawn by device revocation so the
	// withdrawal takes effect at request time instead of at next boot.
	revoked map[string]struct{}
}

func NewAuthorityRegistry() *AuthorityRegistry {
	return &AuthorityRegistry{
		values:  make(map[string]device_trust.AuthorityPackage),
		revoked: make(map[string]struct{}),
	}
}

// Revoke withdraws an authority immediately. Both the sync boundary and the
// upload-token gate resolve authorities through Get, so a tombstoned id stops
// authorizing work without waiting for a process restart.
func (r *AuthorityRegistry) Revoke(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.revoked[id] = struct{}{}
}

// IsRevoked reports whether an authority id has been withdrawn.
func (r *AuthorityRegistry) IsRevoked(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, revoked := r.revoked[id]
	return revoked
}

// RevokeDeviceAuthorities withdraws every authority issued to a device. It
// walks the registry's own contents rather than any startup snapshot, so
// authorities minted after boot (local provisioning, pilot enrolment) are
// tombstoned too.
func (r *AuthorityRegistry) RevokeDeviceAuthorities(deviceID string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	revoked := 0
	for id, authority := range r.values {
		if authority.DeviceID != deviceID {
			continue
		}
		r.revoked[id] = struct{}{}
		revoked++
	}
	return revoked
}

func NewHandler(processor *domainsync.Processor) *Handler {
	return &Handler{Processor: processor, Authorities: NewAuthorityRegistry(), Now: func() time.Time { return time.Now().UTC() }}
}

func (h *Handler) RegisterAuthority(authority device_trust.AuthorityPackage) {
	h.Authorities.mu.Lock()
	defer h.Authorities.mu.Unlock()
	h.Authorities.values[authority.ID] = authority
}

func (r *AuthorityRegistry) Get(id string) (device_trust.AuthorityPackage, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, revoked := r.revoked[id]; revoked {
		return device_trust.AuthorityPackage{}, false
	}
	authority, exists := r.values[id]
	return authority, exists
}

func (r *AuthorityRegistry) Register(authority device_trust.AuthorityPackage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, tombstoned := r.revoked[authority.ID]; tombstoned {
		// Re-enrolment legitimately reissues an authority, so a package that
		// differs from the withdrawn one clears the tombstone. An identical
		// package stays revoked so a replayed registration cannot resurrect a
		// withdrawn grant.
		if existing, exists := r.values[authority.ID]; !exists || existing.Signature != authority.Signature {
			delete(r.revoked, authority.ID)
		}
	}
	r.values[authority.ID] = authority
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST is required")
		return
	}
	if r.ContentLength > 10<<20 {
		writeError(w, http.StatusRequestEntityTooLarge, "request body exceeds 10 MiB limit")
		return
	}
	if ct := r.Header.Get("Content-Type"); ct != "" {
		mediaType := strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
		if mediaType != "application/json" {
			writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
			return
		}
	}
	if h.Processor == nil {
		writeError(w, http.StatusServiceUnavailable, "sync processor is unavailable")
		return
	}
	var incoming request
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&incoming); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if strings.TrimSpace(incoming.AuthorityID) == "" {
		writeError(w, http.StatusBadRequest, "authority_id is required")
		return
	}
	if org, ok := oidchttp.OrganizationContextFrom(r.Context()); ok {
		if strings.TrimSpace(org.TenantID) != "" && incoming.TenantID != org.TenantID {
			writeError(w, http.StatusForbidden, "tenant_scope_conflict")
			return
		}
		if strings.TrimSpace(org.ActorID) != "" && incoming.UserID != org.ActorID {
			writeError(w, http.StatusForbidden, "user_scope_conflict")
			return
		}
	}
	authority, exists := h.Authorities.Get(incoming.AuthorityID)
	if !exists {
		writeError(w, http.StatusForbidden, "authority package is not registered")
		return
	}
	payload, err := canonicalPayload(incoming.Payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, "payload must be valid JSON")
		return
	}
	transaction := domainsync.Transaction{
		ProtocolVersion:    incoming.ProtocolVersion,
		TransactionID:      incoming.TransactionID,
		TenantID:           incoming.TenantID,
		OrganizationID:     incoming.OrganizationID,
		Environment:        incoming.Environment,
		DeviceID:           incoming.DeviceID,
		UserID:             incoming.UserID,
		SequenceNumber:     incoming.SequenceNumber,
		Operation:          incoming.Operation,
		EntityID:           incoming.EntityID,
		Payload:            payload,
		PayloadHash:        incoming.PayloadHash,
		CapturedAt:         incoming.CapturedAt,
		AuthorityID:        incoming.AuthorityID,
		AuthorityEpoch:     incoming.AuthorityEpoch,
		SignatureAlgorithm: incoming.SignatureAlgorithm,
		KeyID:              incoming.KeyID,
		Signature:          incoming.Signature,
	}
	if h.PackageValidator != nil && transaction.Operation == "InspectionSubmitted" {
		if err := h.PackageValidator.ValidateInspectionPayload(r.Context(), transaction.TenantID, transaction.OrganizationID, transaction.EntityID, transaction.Payload); err != nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}
	result := h.Processor.SubmitContext(r.Context(), transaction, authority, h.Now().UTC())
	if result.Outcome == domainsync.Applied {
		// Server-side cascade drain of any held transactions awaiting this sequence
		h.Processor.DrainHeld(r.Context(), transaction.TenantID, transaction.DeviceID, h.Now().UTC())
	}
	writeJSON(w, http.StatusOK, response{Outcome: result.Outcome, Reason: result.Reason, TransactionID: result.TransactionID, ExpectedSequence: result.ExpectedSequence, PayloadHash: result.PayloadHash})
}

func canonicalPayload(raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty payload")
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func writeError(w http.ResponseWriter, status int, reason string) {
	writeJSON(w, status, response{Outcome: domainsync.Rejected, Reason: reason})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
