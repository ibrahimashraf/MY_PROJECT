package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DefaultTUSChunkSize caps a single chunk append at 2 MiB, sized for the weak
// offshore satellite VSAT uplink this upload contract is built for.
const DefaultTUSChunkSize = 2 << 20

// DefaultTUStaleTTL is how long an untouched session lives before PurgeStale
// reclaims it.
// lean-ctx: exported const, callers override per-deployment; no env needed.
const DefaultTUStaleTTL = 24 * time.Hour

var (
	ErrUploadNotFound       = errors.New("upload session not found")
	ErrUploadOffsetMismatch = errors.New("upload offset mismatch")
)

// UploadSession is the resumable upload contract the field app speaks to:
// media is declared up front (size + full-object SHA-256) and appended in
// offset-verified chunks. On the last chunk the server verifies the assembled
// bytes against the declared checksum.
type UploadSession struct {
	ID          string
	ContentType string
	Size        int64
	Checksum    string
	Offset      int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type tusSession struct {
	UploadSession
	file string
}

// tusSidecar is the on-disk JSON record that lets a restarted process reattach
// an in-flight session without any in-memory state.
type tusSidecar struct {
	ID          string `json:"id"`
	Size        int64  `json:"size"`
	Checksum    string `json:"checksum"`
	Offset      int64  `json:"offset"`
	ContentType string `json:"content_type"`
}

// TUSManager coordinates resumable chunked media uploads. Chunk bytes live in
// a per-session file under dir; session metadata lives in memory. All state
// is mutex-guarded, so concurrent appends to one session are rejected rather
// than racing.
type TUSManager struct {
	mu        sync.RWMutex
	dir       string
	chunkSize int64
	staleTTL  time.Duration
	sessions  map[string]*tusSession
}

// NewTUSManager creates the session directory and returns a ready manager.
// Non-positive chunkSize and staleTTL fall back to the documented defaults.
func NewTUSManager(dir string, chunkSize int64, staleTTL time.Duration) (*TUSManager, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, errors.New("upload session directory is required")
	}
	if chunkSize <= 0 {
		chunkSize = DefaultTUSChunkSize
	}
	if staleTTL <= 0 {
		staleTTL = DefaultTUStaleTTL
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	m := &TUSManager{dir: dir, chunkSize: chunkSize, staleTTL: staleTTL, sessions: make(map[string]*tusSession)}
	if _, _, err := m.RecoverOrphans(); err != nil {
		return nil, fmt.Errorf("recover orphans: %w", err)
	}
	return m, nil
}

// Create opens a new upload session for a media object of size bytes with the
// given full-object SHA-256 hex checksum. Negative and zero sizes are rejected.
func (m *TUSManager) Create(ctx context.Context, size int64, checksum, contentType string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if size <= 0 {
		return "", errors.New("upload size must be positive")
	}
	if !validSHA256Hex(checksum) {
		return "", errors.New("upload checksum must be a hex-encoded SHA-256")
	}
	if strings.TrimSpace(contentType) == "" {
		return "", errors.New("upload content type is required")
	}
	id, err := newUploadID()
	if err != nil {
		return "", err
	}
	path := filepath.Join(m.dir, id)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	now := time.Now().UTC()
	session := &tusSession{UploadSession: UploadSession{
		ID: id, ContentType: contentType, Size: size,
		Checksum:  strings.ToLower(strings.TrimSpace(checksum)),
		CreatedAt: now, UpdatedAt: now,
	}, file: path}
	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()
	if err := m.writeSidecar(session); err != nil {
		m.mu.Lock()
		delete(m.sessions, id)
		m.mu.Unlock()
		if removeErr := os.Remove(session.file); removeErr != nil {
			return "", errors.Join(err, removeErr)
		}
		return "", err
	}
	return id, nil
}

// Append writes one chunk at the session's current offset. A chunk whose
// offset does not match exactly is rejected so dropped or reordered links can
// never corrupt the object. Returns the next offset on success.
func (m *TUSManager) Append(ctx context.Context, id string, offset int64, chunk []byte) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if len(chunk) == 0 {
		return 0, errors.New("chunk must not be empty")
	}
	if int64(len(chunk)) > m.chunkSize {
		return 0, fmt.Errorf("chunk exceeds %d byte limit", m.chunkSize)
	}
	if offset < 0 {
		return 0, errors.New("offset must not be negative")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[id]
	if !ok {
		return 0, ErrUploadNotFound
	}
	if offset != session.Offset {
		return session.Offset, fmt.Errorf("%w: expected %d, got %d", ErrUploadOffsetMismatch, session.Offset, offset)
	}
	if offset+int64(len(chunk)) > session.Size {
		return session.Offset, errors.New("chunk exceeds declared upload size")
	}
	file, err := os.OpenFile(session.file, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return session.Offset, err
	}
	written, err := file.Write(chunk)
	if err != nil {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
		return session.Offset, err
	}
	if err := file.Sync(); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
		return session.Offset, err
	}
	if err := file.Close(); err != nil {
		return session.Offset, err
	}
	if int64(written) != int64(len(chunk)) {
		return session.Offset, errors.New("short write")
	}
	session.Offset += int64(len(chunk))
	session.UpdatedAt = time.Now().UTC()
	if err := m.writeSidecar(session); err != nil {
		prior := session.Offset - int64(len(chunk))
		if truncateErr := os.Truncate(session.file, prior); truncateErr != nil {
			return session.Offset, errors.Join(err, truncateErr)
		}
		session.Offset = prior
		return prior, err
	}
	return session.Offset, nil
}

// Offset returns the session's resumable state. The client asks for this after
// a link drop and replays its next chunk from Offset, never from parse guesses.
func (m *TUSManager) Offset(ctx context.Context, id string) (UploadSession, error) {
	if err := ctx.Err(); err != nil {
		return UploadSession{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	if !ok {
		return UploadSession{}, ErrUploadNotFound
	}
	return session.UploadSession, nil
}

// Complete assembles the session file into an Object after verifying the
// object was fully received (Offset == Size) and its SHA-256 matches the
// checksum declared at Create. The session is then reclaimed.
func (m *TUSManager) Complete(ctx context.Context, id string) (Object, error) {
	if err := ctx.Err(); err != nil {
		return Object{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[id]
	if !ok {
		return Object{}, ErrUploadNotFound
	}
	if session.Offset != session.Size {
		return Object{}, fmt.Errorf("upload incomplete: %d of %d bytes received", session.Offset, session.Size)
	}
	data, err := os.ReadFile(session.file)
	if err != nil {
		return Object{}, err
	}
	if sha256Hex(data) != session.Checksum {
		return Object{}, errors.New("upload checksum mismatch")
	}
	if err := os.Remove(session.file); err != nil {
		return Object{}, err
	}
	removeErr := removeIfExists(sidecarPath(m.dir, session.ID))
	delete(m.sessions, id)
	if removeErr != nil {
		return Object{}, removeErr
	}
	return Object{Key: id, ContentType: session.ContentType, Data: data}, nil
}

// Abort discards a session and its partial bytes. Unknown ids return
// ErrUploadNotFound.
func (m *TUSManager) Abort(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[id]
	if !ok {
		return ErrUploadNotFound
	}
	if err := os.Remove(session.file); err != nil {
		return err
	}
	removeErr := removeIfExists(sidecarPath(m.dir, session.ID))
	delete(m.sessions, id)
	return removeErr
}

// RecoverOrphans repairs the session directory after a restart. The chunk FILE
// is authoritative: a parseable sidecar is used only to plausibility-check the
// bytes on disk, and the re-attached offset is the actual chunk-file size. A
// crash between a chunk write and its sidecar write therefore resumes with the
// received (unacknowledged) bytes instead of discarding them — the client
// replays from the queried offset anyway. Anything that cannot be explained by
// the sidecar is deleted. Sidecar-bearing orphans whose chunk file predates the
// manager's stale TTL are pruned instead of resumed: a crash-orphaned upload
// that has sat untouched for longer than the TTL is stale garbage, not a live
// satellite resumption. Resumed and deleted sessions are counted separately.
func (m *TUSManager) RecoverOrphans() (resumed, deleted int, err error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return 0, 0, err
	}
	staleCutoff := time.Now().UTC().Add(-m.staleTTL)
	m.mu.RLock()
	tracked := make(map[string]struct{}, len(m.sessions))
	for id := range m.sessions {
		tracked[id] = struct{}{}
	}
	m.mu.RUnlock()
	var removalErrors []error
	handled := make(map[string]struct{})
	// Pass 1: authoritative sidecars decide session fate. A parseable sidecar
	// whose chunk file holds between the committed offset and committed offset +
	// one chunk (never more than the declared size) restores the session at the
	// file's real size unless the file is stale; anything else removes the
	// sidecar and its chunk together.
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		path := filepath.Join(m.dir, name)
		id, hasSidecar := strings.CutSuffix(name, ".json")
		if !hasSidecar {
			continue
		}
		if _, live := tracked[id]; live {
			continue
		}
		session, ok := loadSidecar(path, id)
		if ok {
			info, infoErr := os.Stat(filepath.Join(m.dir, id))
			if infoErr == nil && !info.IsDir() && info.Size() >= session.Offset && info.Size() <= session.Size && info.Size()-session.Offset <= m.chunkSize && !info.ModTime().Before(staleCutoff) {
				session.Offset = info.Size()
				session.UpdatedAt = time.Now().UTC()
				session.file = filepath.Join(m.dir, id)
				m.mu.Lock()
				m.sessions[id] = session
				m.mu.Unlock()
				resumed++
				continue
			}
		}
		if err := removeIfExists(filepath.Join(m.dir, id)); err != nil {
			removalErrors = append(removalErrors, err)
		}
		if err := removeIfExists(path); err != nil {
			removalErrors = append(removalErrors, err)
		}
		handled[id] = struct{}{}
		deleted++
	}
	// Pass 2: any remaining untracked file is a chunk or sidecar staging file
	// with no recoverable session and is unreferenced garbage. Files restored in
	// pass 1 are re-marked live; corrupt pairings were already removed with
	// their sidecars. A .tmp whose session is live is kept: it is the write in
	// progress of a running manager.
	m.mu.RLock()
	for id := range m.sessions {
		tracked[id] = struct{}{}
	}
	m.mu.RUnlock()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, hasSidecar := strings.CutSuffix(name, ".json"); hasSidecar {
			continue
		}
		if _, live := tracked[name]; live {
			continue
		}
		if base, ok := strings.CutSuffix(name, ".tmp"); ok {
			if _, live := tracked[base]; live {
				continue
			}
		}
		if _, done := handled[name]; done {
			continue
		}
		if err := removeIfExists(filepath.Join(m.dir, name)); err != nil {
			removalErrors = append(removalErrors, err)
		}
		deleted++
	}
	if len(removalErrors) > 0 {
		return resumed, deleted, errors.Join(removalErrors...)
	}
	return resumed, deleted, nil
}

// PurgeStale reclaims sessions untouched for the manager's stale TTL. Returns
// how many were reclaimed. Leftover files from a terminated process are not
// tracked here and are not claimed; that requires recovery on restart.
func (m *TUSManager) PurgeStale(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	cutoff := time.Now().UTC().Add(-m.staleTTL)
	m.mu.Lock()
	defer m.mu.Unlock()
	var removalErrors []error
	var count int
	for id, session := range m.sessions {
		if !session.UpdatedAt.Before(cutoff) {
			continue
		}
		if err := os.Remove(session.file); err != nil {
			removalErrors = append(removalErrors, err)
			continue
		}
		if err := removeIfExists(sidecarPath(m.dir, session.ID)); err != nil {
			removalErrors = append(removalErrors, err)
		}
		delete(m.sessions, id)
		count++
	}
	if len(removalErrors) > 0 {
		return count, errors.Join(removalErrors...)
	}
	return count, nil
}

// writeSidecar persists the session's resumable contract to <id>.json. Written
// on Create and after every Append so a crash can never reattach bytes that a
// responding client was never told were durable. The JSON is staged in the
// same directory and renamed into place so a crash mid-write leaves either the
// previous sidecar or the new one, never a truncated file. Stray <id>.tmp
// staging files are reclaimed by RecoverOrphans.
func (m *TUSManager) writeSidecar(session *tusSession) error {
	data, err := json.Marshal(tusSidecar{
		ID:          session.ID,
		Size:        session.Size,
		Checksum:    session.Checksum,
		Offset:      session.Offset,
		ContentType: session.ContentType,
	})
	if err != nil {
		return err
	}
	tmp := filepath.Join(m.dir, session.ID+".tmp")
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, sidecarPath(m.dir, session.ID))
}

// loadSidecar parses and validates a session sidecar. The sidecar id must match
// its filename and the declared fields must be sane, otherwise it is treated as
// corrupt garbage by the caller.
func loadSidecar(path, id string) (*tusSession, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var sidecar tusSidecar
	if err := json.Unmarshal(data, &sidecar); err != nil {
		return nil, false
	}
	if sidecar.ID != id || sidecar.Size <= 0 || sidecar.Offset < 0 || sidecar.Offset > sidecar.Size || !validSHA256Hex(sidecar.Checksum) || strings.TrimSpace(sidecar.ContentType) == "" {
		return nil, false
	}
	now := time.Now().UTC()
	return &tusSession{UploadSession: UploadSession{
		ID:          sidecar.ID,
		ContentType: sidecar.ContentType,
		Size:        sidecar.Size,
		Checksum:    strings.ToLower(sidecar.Checksum),
		Offset:      sidecar.Offset,
		CreatedAt:   now,
		UpdatedAt:   now,
	}}, true
}

func sidecarPath(dir, id string) string {
	return filepath.Join(dir, id+".json")
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func validSHA256Hex(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func newUploadID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}
