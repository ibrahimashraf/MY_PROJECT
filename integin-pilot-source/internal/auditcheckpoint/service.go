package auditcheckpoint

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"integin/internal/storage"
)

var (
	ErrCheckpointNotFound = errors.New("checkpoint not found")
	ErrSequenceGap        = errors.New("sequence continuity violated")
	ErrMismatchedRoot     = errors.New("previous checkpoint root mismatch")
)

// CheckpointRepository persists lightweight checkpoint metadata that is safe to
// query under multi-tenant RLS. Signed checkpoint objects live in storage.Store.
type CheckpointRepository interface {
	Save(ctx context.Context, cp Checkpoint, objectKey string) error
	GetLatest(ctx context.Context, tenantID, environment string) (*Checkpoint, error)
	GetByID(ctx context.Context, tenantID, checkpointID string) (*Checkpoint, string, error)
}

type repoRecord struct {
	checkpoint Checkpoint
	objectKey  string
}

// InMemoryCheckpointRepository is a thread-safe, hermetic implementation of
// CheckpointRepository for unit and integration tests.
type InMemoryCheckpointRepository struct {
	mu     sync.RWMutex
	byID   map[string]repoRecord
	latest map[string]string
}

func NewInMemoryCheckpointRepository() *InMemoryCheckpointRepository {
	return &InMemoryCheckpointRepository{
		byID:   make(map[string]repoRecord),
		latest: make(map[string]string),
	}
}

func recordKey(tenantID, checkpointID string) string {
	return tenantID + "\x00" + checkpointID
}

func envKey(tenantID, environment string) string {
	return tenantID + "\x00" + environment
}

func (r *InMemoryCheckpointRepository) Save(ctx context.Context, cp Checkpoint, objectKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(cp.TenantID) == "" {
		return errors.New("checkpoint tenant_id is required")
	}
	if cp.RootSHA256 == "" {
		return errors.New("checkpoint root_sha256 is required")
	}
	if strings.TrimSpace(objectKey) == "" {
		return errors.New("checkpoint object key is required")
	}
	id := checkpointRecordID(cp.RootSHA256)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[recordKey(cp.TenantID, id)] = repoRecord{checkpoint: cp, objectKey: objectKey}
	r.latest[envKey(cp.TenantID, cp.Environment)] = id
	return nil
}

func (r *InMemoryCheckpointRepository) GetLatest(ctx context.Context, tenantID, environment string) (*Checkpoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.latest[envKey(tenantID, environment)]
	if !ok {
		return nil, nil
	}
	rec, ok := r.byID[recordKey(tenantID, id)]
	if !ok {
		return nil, errors.New("checkpoint registry is inconsistent")
	}
	cp := rec.checkpoint
	return &cp, nil
}

func (r *InMemoryCheckpointRepository) GetByID(ctx context.Context, tenantID, checkpointID string) (*Checkpoint, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.byID[recordKey(tenantID, checkpointID)]
	if !ok {
		return nil, "", ErrCheckpointNotFound
	}
	cp := rec.checkpoint
	return &cp, rec.objectKey, nil
}

// Signer carries the Ed25519 private key and KeyID used to seal checkpoints.
type Signer struct {
	PrivateKey ed25519.PrivateKey
	KeyID      string
}

// Manager seals, persists, and audits signed audit checkpoints across a
// repository of metadata records and an object store of signed payloads.
type Manager struct {
	Repo    CheckpointRepository
	Storage storage.Store
	Signer  Signer
}

func NewManager(repo CheckpointRepository, store storage.Store, signer Signer) *Manager {
	return &Manager{Repo: repo, Storage: store, Signer: signer}
}

func checkpointRecordID(rootSHA256 string) string {
	return "cp-" + rootSHA256
}

// SealAndPersist seals the next checkpoint in the tenant/environment chain,
// writing the signed object to Storage and the metadata record to Repo. It
// rejects sequence gaps and a registry head whose stored root does not match
// its own canonical payload.
func (m *Manager) SealAndPersist(ctx context.Context, tenantID, environment string, sequenceStart, sequenceEnd uint64, entries []Entry) (*Checkpoint, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(environment) == "" {
		return nil, "", errors.New("tenant_id and environment are required")
	}
	if m.Repo == nil || m.Storage == nil {
		return nil, "", errors.New("checkpoint repository and storage are required")
	}
	if sequenceStart == 0 || sequenceEnd < sequenceStart {
		return nil, "", errors.New("checkpoint sequence bounds are invalid")
	}
	if len(entries) == 0 {
		return nil, "", errors.New("entries are required")
	}

	previousRoot := GenesisRoot
	previous, err := m.Repo.GetLatest(ctx, tenantID, environment)
	if err != nil {
		return nil, "", err
	}
	if previous != nil {
		if sequenceStart != previous.SequenceEnd+1 {
			return nil, "", fmt.Errorf("%w: expected %d, got %d", ErrSequenceGap, previous.SequenceEnd+1, sequenceStart)
		}
		headPayload, err := previous.CanonicalPayload()
		if err != nil {
			return nil, "", err
		}
		headSum := sha256.Sum256(headPayload)
		if previous.RootSHA256 != hex.EncodeToString(headSum[:]) {
			return nil, "", fmt.Errorf("%w: registry head root does not match its payload", ErrMismatchedRoot)
		}
		previousRoot = previous.RootSHA256
	} else if sequenceStart != 1 {
		return nil, "", fmt.Errorf("%w: first checkpoint must begin at sequence 1, got %d", ErrSequenceGap, sequenceStart)
	}

	checkpoint := Checkpoint{
		CheckpointVersion:  Version,
		TenantID:           tenantID,
		Environment:        environment,
		SequenceStart:      sequenceStart,
		SequenceEnd:        sequenceEnd,
		PreviousRootSHA256: previousRoot,
		Entries:            entries,
	}
	if err := checkpoint.Seal(m.Signer.PrivateKey, m.Signer.KeyID); err != nil {
		return nil, "", err
	}

	objectBytes, err := json.Marshal(checkpoint)
	if err != nil {
		return nil, "", err
	}
	id := checkpointRecordID(checkpoint.RootSHA256)
	objectKey := fmt.Sprintf("checkpoints/%s/%s.json", tenantID, id)
	if err := m.Storage.Put(ctx, storage.Object{Key: objectKey, ContentType: "application/json", Data: objectBytes}); err != nil {
		return nil, "", err
	}
	if err := m.Repo.Save(ctx, checkpoint, objectKey); err != nil {
		return nil, "", err
	}
	return &checkpoint, objectKey, nil
}

// VerifyAndAudit loads the metadata record and signed object, then verifies
// the stored checkpoint's root and Ed25519 signature against its canonical
// payload.
func (m *Manager) VerifyAndAudit(ctx context.Context, tenantID, checkpointID string, publicKey ed25519.PublicKey) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(checkpointID) == "" {
		return errors.New("tenant_id and checkpoint_id are required")
	}
	if m.Repo == nil || m.Storage == nil {
		return errors.New("checkpoint repository and storage are required")
	}
	metadata, objectKey, err := m.Repo.GetByID(ctx, tenantID, checkpointID)
	if err != nil {
		return err
	}
	if metadata == nil {
		return ErrCheckpointNotFound
	}

	object, err := m.Storage.Get(ctx, objectKey)
	if err != nil {
		return err
	}
	var stored Checkpoint
	if err := json.Unmarshal(object.Data, &stored); err != nil {
		return fmt.Errorf("deserialize checkpoint object: %w", err)
	}
	if stored.TenantID != tenantID || stored.Environment != metadata.Environment {
		return errors.New("stored checkpoint tenant/environment mismatch")
	}
	if stored.RootSHA256 != metadata.RootSHA256 {
		return errors.New("stored checkpoint root_sha256 does not match registry metadata")
	}
	if err := stored.Verify(publicKey); err != nil {
		return err
	}
	return nil
}
