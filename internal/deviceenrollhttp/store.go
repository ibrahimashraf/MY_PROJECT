package deviceenrollhttp

import (
	"context"
	"sync"
	"time"

	"integin/internal/syncstate"
)

// Repository persists trusted devices and issued authority packages. It
// deliberately omits the sync receipt machinery: the enrollment lifecycle
// deals with device trust and offline authority, not transaction syncing.
type Repository interface {
	GetDevice(ctx context.Context, tenantID, deviceID string) (syncstate.DeviceRecord, error)
	SaveDevice(ctx context.Context, device syncstate.DeviceRecord) error
	SaveAuthority(ctx context.Context, authority syncstate.AuthorityRecord) error
	ListAuthorities(ctx context.Context, tenantID, deviceID string) ([]syncstate.AuthorityRecord, error)
	RevokeAuthority(ctx context.Context, tenantID, authorityID string, at time.Time) error
}

// inMemoryRepository is the hermetic backing store used by default and in
// every unit test. All maps are mutex-guarded; no daemon is required.
type inMemoryRepository struct {
	mu          sync.Mutex
	devices     map[string]syncstate.DeviceRecord
	authorities map[string]syncstate.AuthorityRecord
}

func newInMemoryRepository() *inMemoryRepository {
	return &inMemoryRepository{
		devices:     make(map[string]syncstate.DeviceRecord),
		authorities: make(map[string]syncstate.AuthorityRecord),
	}
}

func (s *inMemoryRepository) GetDevice(_ context.Context, tenantID, deviceID string) (syncstate.DeviceRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	device, ok := s.devices[tenantID+"/"+deviceID]
	if !ok {
		return syncstate.DeviceRecord{}, syncstate.ErrNotFound
	}
	return device, nil
}

func (s *inMemoryRepository) SaveDevice(_ context.Context, device syncstate.DeviceRecord) error {
	if err := syncstate.ValidateDeviceRecord(device); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devices[device.TenantID+"/"+device.DeviceID] = device
	return nil
}

func (s *inMemoryRepository) SaveAuthority(_ context.Context, authority syncstate.AuthorityRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authorities[authority.TenantID+"/"+authority.AuthorityID] = authority
	return nil
}

func (s *inMemoryRepository) ListAuthorities(_ context.Context, tenantID, deviceID string) ([]syncstate.AuthorityRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]syncstate.AuthorityRecord, 0)
	for _, authority := range s.authorities {
		if authority.TenantID == tenantID && authority.DeviceID == deviceID {
			result = append(result, authority)
		}
	}
	return result, nil
}

func (s *inMemoryRepository) RevokeAuthority(_ context.Context, tenantID, authorityID string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	authority, ok := s.authorities[tenantID+"/"+authorityID]
	if !ok {
		return syncstate.ErrNotFound
	}
	authority.RevokedAt = &at
	s.authorities[tenantID+"/"+authorityID] = authority
	return nil
}

// syncRepositoryAdapter backs the device/authority halves of the enrollment
// flow with a database-backed syncstate.Repository. Enrollment request
// lifecycle state (challenges, proof status) remains in-memory on the
// handler until an enrollment request table exists.
type syncRepositoryAdapter struct {
	repo syncstate.Repository
}

func (a syncRepositoryAdapter) GetDevice(ctx context.Context, tenantID, deviceID string) (syncstate.DeviceRecord, error) {
	return a.repo.GetDevice(ctx, tenantID, deviceID)
}

func (a syncRepositoryAdapter) SaveDevice(ctx context.Context, device syncstate.DeviceRecord) error {
	return a.repo.SaveDevice(ctx, device)
}

func (a syncRepositoryAdapter) SaveAuthority(ctx context.Context, authority syncstate.AuthorityRecord) error {
	return a.repo.SaveAuthority(ctx, authority)
}

func (a syncRepositoryAdapter) ListAuthorities(ctx context.Context, tenantID, deviceID string) ([]syncstate.AuthorityRecord, error) {
	all, err := a.repo.ListAuthorities(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	result := make([]syncstate.AuthorityRecord, 0)
	for _, authority := range all {
		if authority.DeviceID == deviceID {
			result = append(result, authority)
		}
	}
	return result, nil
}

func (a syncRepositoryAdapter) RevokeAuthority(ctx context.Context, tenantID, authorityID string, at time.Time) error {
	return a.repo.RevokeAuthority(ctx, tenantID, authorityID, at)
}

// SyncRepository wraps a database-backed syncstate.Repository so it can be
// used as the device/authority persistence for the enrollment handler.
func SyncRepository(repo syncstate.Repository) Repository {
	if repo == nil {
		return nil
	}
	return syncRepositoryAdapter{repo: repo}
}
