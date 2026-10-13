package migrations

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed *.sql
var Files embed.FS

// File represents a discovered migration file.
type File struct {
	Name             string
	Content          string
	Checksum         string
	IsDown           bool
	IsDraft          bool
	NonTransactional bool
}

// DiscoverUpMigrations parses and returns all active forward migration files sorted by name.
func DiscoverUpMigrations() ([]File, error) {
	entries, err := Files.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations dir: %w", err)
	}

	var upFiles []File
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		// Filter out down, draft, and candidate files per INTEGIN convention
		if strings.Contains(name, ".down.") || strings.Contains(name, ".draft.") || strings.Contains(name, ".candidate.") {
			continue
		}

		raw, err := Files.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("read embedded migration %s: %w", name, err)
		}

		content := strings.TrimPrefix(string(raw), "\ufeff")
		hash := sha256.Sum256(raw)
		checksum := hex.EncodeToString(hash[:])

		nonTx := strings.Contains(content, "-- non-transactional") ||
			strings.Contains(content, "CREATE INDEX CONCURRENTLY") ||
			strings.Contains(content, "REINDEX CONCURRENTLY")

		upFiles = append(upFiles, File{
			Name:             name,
			Content:          content,
			Checksum:         checksum,
			NonTransactional: nonTx,
		})
	}

	sort.Slice(upFiles, func(i, j int) bool {
		return upFiles[i].Name < upFiles[j].Name
	})

	return upFiles, nil
}

// DiscoverDownFile finds the matching rollback file for a given up-migration name.
func DiscoverDownFile(upName string) (*File, error) {
	base := strings.TrimSuffix(strings.TrimSuffix(upName, ".up.sql"), ".sql")
	downName := base + ".down.sql"

	raw, err := Files.ReadFile(downName)
	if err != nil {
		return nil, fmt.Errorf("rollback file not found for %s: %w", upName, err)
	}

	hash := sha256.Sum256(raw)
	return &File{
		Name:     downName,
		Content:  strings.TrimPrefix(string(raw), "\ufeff"),
		Checksum: hex.EncodeToString(hash[:]),
		IsDown:   true,
	}, nil
}

// CanonicalName returns the normalized sequence-sorted name of the file.
func (f File) CanonicalName() string {
	return filepath.Base(f.Name)
}
