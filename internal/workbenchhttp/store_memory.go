package workbenchhttp

import (
	"context"
	"strings"
	"sync"
	"time"

	"integin/internal/domain/device_trust"
)

// memHeld is the in-memory held-transaction record. The raw offline payload
// is intentionally never stored here — only its hash plus replay metadata.
type memHeld struct {
	TransactionID  string
	TenantID       string
	OrganizationID string
	DeviceID       string
	SequenceNumber uint64
	ErrorReason    string
	HeldAt         time.Time
	LastAttemptAt  time.Time
	Attempts       int
	PayloadHash    string
}

// memoryStore is the hermetic fallback used when no Postgres connection is
// configured. Every map is guarded by one RWMutex so test suites pass
// `go test -race` cleanly without a live database. It is a test/offline
// authority, not a durable ledger; production deployments use postgresStore.
type memoryStore struct {
	mu         sync.RWMutex
	now        func() time.Time
	devices    map[string]*DeviceView
	enroll     map[string]*EnrollmentRequestView
	held       map[string]*memHeld
	legalHolds map[string]*LegalHold
	retention  map[string]*RetentionPolicy
	exports    map[string]*ExportApproval
}

func newMemoryStore(seedDevices []device_trust.Device) *memoryStore {
	store := &memoryStore{
		now:        time.Now,
		devices:    make(map[string]*DeviceView),
		enroll:     make(map[string]*EnrollmentRequestView),
		held:       make(map[string]*memHeld),
		legalHolds: make(map[string]*LegalHold),
		retention:  make(map[string]*RetentionPolicy),
		exports:    make(map[string]*ExportApproval),
	}
	for _, device := range seedDevices {
		store.devices[device.ID()] = &DeviceView{
			ID:             device.ID(),
			TenantID:       device.TenantID(),
			OrganizationID: device.OrganizationID(),
			UserID:         device.UserID(),
			PublicKey:      device.PublicKey(),
			State:          string(device.State()),
			Epoch:          device.Epoch(),
			EnrolledAt:     store.now().UTC(),
		}
	}
	return store
}

func (s *memoryStore) ListDevices(_ context.Context, tenantID, organizationID string) ([]DeviceView, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]DeviceView, 0, len(s.devices))
	for _, device := range s.devices {
		if device.TenantID != tenantID || device.OrganizationID != organizationID {
			continue
		}
		result = append(result, *device)
	}
	return result, nil
}

func (s *memoryStore) ListEnrollmentRequests(_ context.Context, tenantID, organizationID string) ([]EnrollmentRequestView, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]EnrollmentRequestView, 0, len(s.enroll))
	for _, request := range s.enroll {
		if request.TenantID != tenantID || request.OrganizationID != organizationID {
			continue
		}
		result = append(result, *request)
	}
	return result, nil
}

func (s *memoryStore) ApproveEnrollment(_ context.Context, actorID, tenantID, organizationID, requestID string, epoch uint64) (DeviceView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	request, exists := s.enroll[requestID]
	if !exists || request.TenantID != tenantID || request.OrganizationID != organizationID {
		return DeviceView{}, ErrNotFound
	}
	if request.Status != string(device_trust.EnrollmentPending) {
		return DeviceView{}, ErrConflict
	}
	device := &DeviceView{
		ID:             request.DeviceID,
		TenantID:       request.TenantID,
		OrganizationID: request.OrganizationID,
		UserID:         request.UserID,
		PublicKey:      request.PublicKey,
		State:          statusTrusted,
		Epoch:          epoch,
		EnrolledAt:     s.now().UTC(),
	}
	s.devices[device.ID] = device
	request.Status = string(device_trust.EnrollmentApproved)
	request.ApprovedBy = actorID
	return *device, nil
}

func (s *memoryStore) RevokeDevice(_ context.Context, tenantID, organizationID, deviceID, reason string) (DeviceView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	device, exists := s.devices[deviceID]
	if !exists || device.TenantID != tenantID || device.OrganizationID != organizationID {
		return DeviceView{}, ErrNotFound
	}
	revokedAt := s.now().UTC()
	device.State = statusRevoked
	device.RevokedAt = &revokedAt
	device.RevocationReason = reason
	return *device, nil
}

func (s *memoryStore) ListHeld(_ context.Context, tenantID, organizationID string) ([]HeldTransaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]HeldTransaction, 0, len(s.held))
	for _, transaction := range s.held {
		if transaction.TenantID != tenantID || transaction.OrganizationID != organizationID {
			continue
		}
		view := HeldTransaction{
			TransactionID:  transaction.TransactionID,
			DeviceID:       transaction.DeviceID,
			SequenceNumber: transaction.SequenceNumber,
			Status:         statusHeld,
			ErrorReason:    transaction.ErrorReason,
			HeldAt:         transaction.HeldAt,
			PayloadHash:    transaction.PayloadHash,
		}
		if !transaction.LastAttemptAt.IsZero() {
			attemptedAt := transaction.LastAttemptAt
			view.LastAttemptAt = &attemptedAt
		}
		result = append(result, view)
	}
	return result, nil
}

func (s *memoryStore) ReconcileHeld(_ context.Context, tenantID, organizationID, transactionID string) (ReconcileResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	transaction, exists := s.held[transactionID]
	if !exists || transaction.TenantID != tenantID || transaction.OrganizationID != organizationID {
		return ReconcileResult{}, ErrNotFound
	}
	transaction.Attempts++
	transaction.LastAttemptAt = s.now().UTC()
	return ReconcileResult{
		TransactionID: transaction.TransactionID,
		Status:        statusHeld,
		RetriedAt:     transaction.LastAttemptAt,
		Attempt:       transaction.Attempts,
		ErrorReason:   transaction.ErrorReason,
	}, nil
}

func (s *memoryStore) ListLegalHolds(_ context.Context, tenantID, _ string) ([]LegalHold, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]LegalHold, 0, len(s.legalHolds))
	for _, hold := range s.legalHolds {
		if hold.TenantID != tenantID || hold.Status != statusActive {
			continue
		}
		result = append(result, *hold)
	}
	return result, nil
}

func (s *memoryStore) PlaceLegalHold(_ context.Context, actorID, tenantID, _, entityType, entityID, reason string) (LegalHold, error) {
	if strings.TrimSpace(entityType) == "" || strings.TrimSpace(entityID) == "" || strings.TrimSpace(reason) == "" {
		return LegalHold{}, ErrValidation
	}
	id, err := newRandomID("legalhold-")
	if err != nil {
		return LegalHold{}, err
	}
	hold := &LegalHold{
		ID:         id,
		TenantID:   tenantID,
		EntityType: entityType,
		EntityID:   entityID,
		Reason:     reason,
		PlacedBy:   actorID,
		PlacedAt:   s.now().UTC(),
		Status:     statusActive,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.legalHolds[id] = hold
	return *hold, nil
}

func (s *memoryStore) ReleaseLegalHold(_ context.Context, actorID, tenantID, _, id string) (LegalHold, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	hold, exists := s.legalHolds[id]
	if !exists || hold.TenantID != tenantID {
		return LegalHold{}, ErrNotFound
	}
	if hold.Status == statusReleased {
		return *hold, nil
	}
	releasedBy := actorID
	releasedAt := s.now().UTC()
	hold.Status = statusReleased
	hold.ReleasedBy = &releasedBy
	hold.ReleasedAt = &releasedAt
	return *hold, nil
}

func (s *memoryStore) ListRetentionPolicies(_ context.Context, tenantID, _ string) ([]RetentionPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]RetentionPolicy, 0, len(s.retention))
	for _, policy := range s.retention {
		if policy.TenantID != tenantID {
			continue
		}
		result = append(result, *policy)
	}
	return result, nil
}

func (s *memoryStore) ListExportApprovals(_ context.Context, tenantID, _ string) ([]ExportApproval, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ExportApproval, 0, len(s.exports))
	for _, approval := range s.exports {
		if approval.TenantID != tenantID {
			continue
		}
		result = append(result, *approval)
	}
	return result, nil
}

func (s *memoryStore) DecideExportApproval(_ context.Context, actorID, tenantID, _, exportID, status, rejectionReason string) (ExportApproval, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	approval, exists := s.exports[exportID]
	if !exists || approval.TenantID != tenantID {
		return ExportApproval{}, ErrNotFound
	}
	if approval.Status != statusPending {
		return ExportApproval{}, ErrConflict
	}
	decidedAt := s.now().UTC()
	approval.Status = status
	approval.ApprovedBy = actorID
	approval.DecidedAt = &decidedAt
	approval.RejectionReason = rejectionReason
	return *approval, nil
}
