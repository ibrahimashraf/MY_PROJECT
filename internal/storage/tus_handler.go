package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	return &TUSManager{dir: dir, chunkSize: chunkSize, staleTTL: staleTTL, sessions: make(map[string]*tusSession)}, nil
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
	closeErr := file.Close()
	if err != nil {
		return session.Offset, err
	}
	if closeErr != nil {
		return session.Offset, closeErr
	}
	if int64(written) != int64(len(chunk)) {
		return session.Offset, errors.New("short write")
	}
	session.Offset += int64(len(chunk))
	session.UpdatedAt = time.Now().UTC()
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
	delete(m.sessions, id)
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
	delete(m.sessions, id)
	return nil
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
		delete(m.sessions, id)
		count++
	}
	if len(removalErrors) > 0 {
		return count, errors.Join(removalErrors...)
	}
	return count, nil
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
