// Package fixturesignals emits a bounded non-authoritative signal from the
// source-owned pilot fixture. It contains no fixture payload or private data.
package fixturesignals

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	SignalDirectoryEnvironment = "INTEGIN_PILOT_FIXTURE_SIGNAL_DIR"
	SignalVersionEnvironment   = "INTEGIN_PILOT_FIXTURE_SIGNAL_CONTRACT_VERSION"
	SignalContractVersion      = "1"
	SignalFileName             = "fixture-signal-v1.json"
	SignalStepGenerated        = "fixture_generated"
	maxSignalBytes             = 1024
)

var ErrDuplicateSignal = errors.New("fixture signal already exists")

// Signal is the closed public-redacted fixture evidence shape. It is not a
// candidate receipt, case verdict, proof, or authoritative workflow record.
type Signal struct {
	ContractVersion string `json:"contract_version"`
	Step            string `json:"step"`
	SignalID        string `json:"signal_id"`
	GeneratedAt     string `json:"generated_at"`
}

// EmitFromEnvironment reads only the reviewed child-only fixture signal
// configuration. All-empty context preserves historical fixture behavior;
// partial or invalid context fails the hard gate without exposing data.
func EmitFromEnvironment(now func() time.Time) (bool, error) {
	return Emit(os.Getenv(SignalDirectoryEnvironment), os.Getenv(SignalVersionEnvironment), now, rand.Reader)
}

// Emit atomically publishes exactly one fixture-generated signal. The caller
// supplies the directory and contract version explicitly to keep the writer
// testable and avoid process-global test configuration.
func Emit(directory, version string, now func() time.Time, random io.Reader) (bool, error) {
	directory = strings.TrimSpace(directory)
	version = strings.TrimSpace(version)
	if directory == "" && version == "" {
		return false, nil
	}
	if directory == "" || version == "" {
		return false, errors.New("fixture signal context is incomplete")
	}
	if version != SignalContractVersion {
		return false, errors.New("fixture signal contract version is unsupported")
	}
	if err := assertRegularDirectory(directory); err != nil {
		return false, err
	}
	if now == nil {
		now = time.Now
	}
	if random == nil {
		return false, errors.New("fixture signal random source is unavailable")
	}
	identifier := make([]byte, 16)
	if _, err := io.ReadFull(random, identifier); err != nil {
		return false, fmt.Errorf("read fixture signal random identifier: %w", err)
	}
	signal := Signal{
		ContractVersion: SignalContractVersion,
		Step:            SignalStepGenerated,
		SignalID:        hex.EncodeToString(identifier),
		GeneratedAt:     now().UTC().Truncate(time.Second).Format(time.RFC3339),
	}
	encoded, err := json.Marshal(signal)
	if err != nil {
		return false, fmt.Errorf("marshal fixture signal: %w", err)
	}
	if len(encoded) > maxSignalBytes {
		return false, errors.New("fixture signal exceeds size limit")
	}

	// Use os.Root for directory-scoped operations - validates directory is safe
	root, err := os.OpenRoot(directory)
	if err != nil {
		return false, fmt.Errorf("open fixture signal directory root: %w", err)
	}
	defer root.Close()

	// Check for duplicate within the rooted directory
	if _, err := root.Stat(SignalFileName); err == nil {
		return false, ErrDuplicateSignal
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("inspect fixture signal target: %w", err)
	}

	// Create temp file using absolute directory path (validated by assertRegularDirectory)
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return false, fmt.Errorf("resolve fixture signal directory: %w", err)
	}
	temporary, err := os.CreateTemp(absDir, ".fixture-signal-*.tmp")
	if err != nil {
		return false, fmt.Errorf("create fixture signal temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return false, fmt.Errorf("restrict fixture signal temporary file: %w", err)
	}
	if _, err := temporary.Write(encoded); err != nil {
		_ = temporary.Close()
		return false, fmt.Errorf("write fixture signal: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return false, fmt.Errorf("sync fixture signal: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return false, fmt.Errorf("close fixture signal: %w", err)
	}

	// Atomic rename within the rooted directory using os.Root
	tempName := filepath.Base(temporaryPath)
	if err := root.Rename(tempName, SignalFileName); err != nil {
		if os.IsExist(err) {
			return false, ErrDuplicateSignal
		}
		return false, fmt.Errorf("publish fixture signal: %w", err)
	}
	return true, nil
}

func assertRegularDirectory(directory string) error {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return fmt.Errorf("open fixture signal directory root: %w", err)
	}
	defer root.Close()

	info, err := root.Stat(".")
	if err != nil {
		return fmt.Errorf("inspect fixture signal directory: %w", err)
	}
	if !info.IsDir() {
		return errors.New("fixture signal directory is not a directory")
	}
	// Check for symlinks
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("fixture signal directory is a symlink")
	}
	return nil
}
