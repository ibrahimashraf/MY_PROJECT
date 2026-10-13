// Package manifestreceipts emits redacted, schema-bounded evidence for an
// isolated manifest candidate. It has no fixture, transport, credential, or
// authority dependencies.
package manifestreceipts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"
)

const maxReceiptBytes = 8 << 10

var (
	ErrDuplicateReceipt = errors.New("manifest case receipt already exists for this run")
	ErrUnsafeDirectory  = errors.New("manifest case receipt directory is unsafe")
	runIDPattern        = regexp.MustCompile(`^[a-f0-9]{32}$`)
)

// Case identifies one public proof-matrix case. It is never derived from a
// fixture payload or stdout/stderr; a source-owned caller supplies it.
type Case string

const (
	CaseFieldBinding       Case = "field_binding"
	CaseValidProof         Case = "valid_proof"
	CaseReplay             Case = "replay"
	CaseSignatureInvalid   Case = "signature_invalid"
	CasePackageHashInvalid Case = "package_hash_invalid"
	CaseExpired            Case = "expired"
	CaseAuthorityMismatch  Case = "authority_mismatch"
	CaseKeyUnknown         Case = "key_unknown"
)

var expectedOutcomes = map[Case]string{
	CaseFieldBinding:       "verified_cached",
	CaseValidProof:         "proof_valid",
	CaseReplay:             "replay_rejected",
	CaseSignatureInvalid:   "signature_invalid",
	CasePackageHashInvalid: "package_hash_invalid",
	CaseExpired:            "expired",
	CaseAuthorityMismatch:  "authority_mismatch",
	CaseKeyUnknown:         "key_unknown",
}

var allowedObservedOutcomes = map[string]struct{}{
	"verified_cached":      {},
	"proof_valid":          {},
	"replay_rejected":      {},
	"signature_invalid":    {},
	"package_hash_invalid": {},
	"expired":              {},
	"authority_mismatch":   {},
	"key_unknown":          {},
}

// Observation contains only the redacted public signals needed for one
// receipt. Raw proofs, manifests, identities, keys, tokens, and fixture output
// are deliberately absent from this type.
type Observation struct {
	Case            Case
	ObservedOutcome string
	HTTPStatus      int
	Before          int64
	After           int64
	ManifestID      string
	CorrelationID   string
	GeneratedAt     time.Time
}

// Writer stores at most one source-redacted receipt for each public case in an
// opaque candidate run. The caller must provision Directory as a fresh,
// ACL-hardened directory; this portable library independently rejects missing
// and symbolic-link paths.
type Writer struct {
	runID     string
	directory string
	now       func() time.Time
	mu        sync.Mutex
	emitted   map[Case]struct{}
}

// NewWriter validates the public run identifier and regular receipt directory.
func NewWriter(runID, directory string, now func() time.Time) (*Writer, error) {
	if !runIDPattern.MatchString(runID) {
		return nil, errors.New("manifest case receipt run ID is invalid")
	}
	if err := assertRegularDirectory(directory); err != nil {
		return nil, err
	}
	if now == nil {
		now = time.Now
	}
	return &Writer{runID: runID, directory: directory, now: now, emitted: make(map[Case]struct{})}, nil
}

// Emit writes one schema-bounded public receipt atomically without overwriting
// an existing case result. A semantic mismatch becomes a failed receipt, while
// a filesystem or validation failure returns an error and writes nothing.
func (w *Writer) Emit(observation Observation) error {
	if w == nil || w.now == nil {
		return errors.New("manifest case receipt writer is not configured")
	}
	expected, ok := expectedOutcomes[observation.Case]
	if !ok {
		return errors.New("manifest case receipt case is invalid")
	}
	if observation.HTTPStatus < 100 || observation.HTTPStatus > 599 {
		return errors.New("manifest case receipt HTTP status is invalid")
	}
	if _, allowed := allowedObservedOutcomes[observation.ObservedOutcome]; !allowed {
		return errors.New("manifest case receipt observed outcome is invalid")
	}
	if observation.Before < 0 || observation.After < 0 || observation.After < observation.Before {
		return errors.New("manifest case receipt count is invalid")
	}
	if observation.CorrelationID != "" && (len(observation.CorrelationID) < 8 || len(observation.CorrelationID) > 128) {
		return errors.New("manifest case receipt correlation ID is invalid")
	}

	delta := observation.After - observation.Before
	passed := observation.ObservedOutcome == expected && deltaAllowed(observation.Case, delta)
	receipt := receipt{
		ContractVersion: "1",
		CandidateRunID:  w.runID,
		Case:            string(observation.Case),
		ExpectedOutcome: expected,
		ObservedOutcome: observation.ObservedOutcome,
		Status:          statusFor(passed),
		HTTPStatus:      observation.HTTPStatus,
		GeneratedAt:     observation.GeneratedAt.UTC().Format(time.RFC3339),
		StateCounts:     stateCounts{Before: observation.Before, After: observation.After, Delta: delta},
	}
	if observation.GeneratedAt.IsZero() {
		receipt.GeneratedAt = w.now().UTC().Format(time.RFC3339)
	}
	if observation.CorrelationID != "" {
		receipt.CorrelationID = observation.CorrelationID
	}
	if observation.ManifestID != "" {
		digest := sha256.Sum256([]byte(observation.ManifestID))
		receipt.ManifestIDDigest = "sha256:" + hex.EncodeToString(digest[:])
	}

	encoded, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("marshal manifest case receipt: %w", err)
	}
	if len(encoded) > maxReceiptBytes {
		return errors.New("manifest case receipt exceeds public size limit")
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if _, exists := w.emitted[observation.Case]; exists {
		return ErrDuplicateReceipt
	}
	if err := assertRegularDirectory(w.directory); err != nil {
		return err
	}
	filename := "receipt-" + string(observation.Case) + "-" + w.runID + ".json"
	if filepath.Base(filename) != filename {
		return ErrUnsafeDirectory
	}
	finalPath := filepath.Join(w.directory, filename)
	if _, err := os.Lstat(finalPath); err == nil {
		return ErrDuplicateReceipt
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect manifest case receipt target: %w", err)
	}

	temporary, err := os.CreateTemp(w.directory, ".receipt-*.tmp")
	if err != nil {
		return fmt.Errorf("create manifest case receipt temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("restrict manifest case receipt temporary file: %w", err)
	}
	if _, err := temporary.Write(encoded); err != nil {
		temporary.Close()
		return fmt.Errorf("write manifest case receipt: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync manifest case receipt: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close manifest case receipt: %w", err)
	}
	if err := os.Link(temporaryPath, finalPath); err != nil {
		if errors.Is(err, os.ErrExist) {
			return ErrDuplicateReceipt
		}
		return fmt.Errorf("publish manifest case receipt: %w", err)
	}
	if runtime.GOOS != "windows" {
		directoryHandle, err := os.Open(w.directory)
		if err != nil {
			return fmt.Errorf("open manifest case receipt directory for sync: %w", err)
		}
		defer directoryHandle.Close()
		if err := directoryHandle.Sync(); err != nil {
			return fmt.Errorf("sync manifest case receipt directory: %w", err)
		}
	}
	w.emitted[observation.Case] = struct{}{}
	return nil
}

func assertRegularDirectory(directory string) error {
	if directory == "" || filepath.Base(directory) == "." {
		return ErrUnsafeDirectory
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return fmt.Errorf("inspect manifest case receipt directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return ErrUnsafeDirectory
	}
	return nil
}

func deltaAllowed(c Case, delta int64) bool {
	switch c {
	case CaseValidProof:
		return delta == 1
	default:
		return delta == 0
	}
}

func statusFor(passed bool) string {
	if passed {
		return "passed"
	}
	return "failed"
}

type stateCounts struct {
	Before int64 `json:"before"`
	After  int64 `json:"after"`
	Delta  int64 `json:"delta"`
}

type receipt struct {
	ContractVersion  string      `json:"contract_version"`
	CandidateRunID   string      `json:"candidate_run_id"`
	Case             string      `json:"case"`
	ExpectedOutcome  string      `json:"expected_outcome"`
	ObservedOutcome  string      `json:"observed_outcome"`
	Status           string      `json:"status"`
	HTTPStatus       int         `json:"http_status"`
	CorrelationID    string      `json:"correlation_id,omitempty"`
	ManifestIDDigest string      `json:"manifest_id_digest,omitempty"`
	StateCounts      stateCounts `json:"state_counts"`
	GeneratedAt      string      `json:"generated_at"`
}
