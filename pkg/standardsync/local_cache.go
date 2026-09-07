package standardsync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// cacheSchemaVersion is incremented on any on-disk format change. Load rejects
// foreign versions so a shipboard device never silently serves stale vectors.
const cacheSchemaVersion = 1

// ErrCacheCorrupt reports an integrity or schema failure when loading a cache.
var ErrCacheCorrupt = errors.New("standards local cache is corrupt")

// LocalCache is an air-gapped, file-backed copy of an AbstractIndex. A cache
// file is fully self-contained: cards, checksum, schema version, and vector
// metadata can be rebuilt on any offline device. It carries only citation
// metadata, never raw standard text.
type LocalCache struct {
	idx  *AbstractIndex
	path string
}

// cacheFile is the on-disk envelope written by Store.
type cacheFile struct {
	SchemaVersion int                    `json:"schema_version"`
	GeneratedAt   time.Time              `json:"generated_at"`
	Checksum      string                 `json:"checksum"`
	Cards         []StandardMetadataCard `json:"cards"`
}

// OpenLocalCache loads a cache file, verifying schema version and checksum.
func OpenLocalCache(path string) (*LocalCache, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var envelope cacheFile
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCacheCorrupt, err)
	}
	if envelope.SchemaVersion != cacheSchemaVersion {
		return nil, fmt.Errorf("%w: schema version %d (want %d)", ErrCacheCorrupt, envelope.SchemaVersion, cacheSchemaVersion)
	}
	if want := checksum(envelope.Cards); envelope.Checksum != want {
		return nil, fmt.Errorf("%w: checksum mismatch", ErrCacheCorrupt)
	}
	idx := NewAbstractIndex(envelope.Cards)
	return &LocalCache{idx: idx, path: path}, nil
}

// Index exposes the in-memory index for search.
func (c *LocalCache) Index() *AbstractIndex { return c.idx }

// Cards returns the cached cards.
func (c *LocalCache) Cards() []StandardMetadataCard { return c.idx.Cards() }

// Store writes the index atomically (temp file + rename) so a power cut on an
// offshore device can never leave a half-written cache behind.
func (c *LocalCache) Store(path string, cards []StandardMetadataCard, now time.Time) error {
	if len(cards) == 0 {
		return errors.New("standards cache requires at least one card")
	}
	stable := append([]StandardMetadataCard(nil), cards...)
	sort.Slice(stable, func(i, j int) bool { return stable[i].StandardDID < stable[j].StandardDID })
	envelope := cacheFile{
		SchemaVersion: cacheSchemaVersion,
		GeneratedAt:   now.UTC(),
		Checksum:      checksum(stable),
		Cards:         stable,
	}
	raw, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	c.path = path
	c.idx = NewAbstractIndex(stable)
	return nil
}

// BuildLocalCacheFromCards is a convenience constructor for the compiler CLI
// that does not require touching the filesystem before search.
func BuildLocalCacheFromCards(cards []StandardMetadataCard) (*LocalCache, error) {
	for _, card := range cards {
		if err := card.Validate(); err != nil {
			return nil, fmt.Errorf("card %q: %w", card.StandardDID, err)
		}
	}
	return &LocalCache{idx: NewAbstractIndex(cards)}, nil
}

// EnsureCachePath sanitizes a cache path for this package.
func EnsureCachePath(path string) (string, error) {
	clean := filepath.Clean(path)
	if clean == "" || strings.ContainsRune(clean, 0) {
		return "", errors.New("invalid cache path")
	}
	return clean, nil
}

// checksum is the deterministic SHA-256 over the canonical card JSON.
func checksum(cards []StandardMetadataCard) string {
	sorted := append([]StandardMetadataCard(nil), cards...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].StandardDID < sorted[j].StandardDID })
	canonical, _ := json.Marshal(sorted)
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:])
}
