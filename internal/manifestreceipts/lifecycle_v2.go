// Package manifestreceipts provides source-owned, public-redacted receipt types.
package manifestreceipts

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"
)

// V2FailureCode is a closed public reason vocabulary. It intentionally omits
// raw filesystem, transport, database, identity, proof, and runtime errors.
type V2FailureCode string

const (
	V2FailureDuplicateCase       V2FailureCode = "duplicate_case"
	V2FailureReceiptPublish      V2FailureCode = "receipt_publish_failed"
	V2FailureUnsafeDirectory     V2FailureCode = "unsafe_receipt_directory"
	V2FailureUnexpectedArtifact  V2FailureCode = "unexpected_public_artifact"
	V2FailureMissingExpectedCase V2FailureCode = "missing_expected_case"
	V2FailureBridgeFailure       V2FailureCode = "bridge_failure_present"
)

// V2FinalizationStatus reports only public lifecycle completeness. It does not
// establish that a case receipt is semantically valid; the public adapter owns
// that separate validation.
type V2FinalizationStatus string

const (
	V2FinalizationComplete   V2FinalizationStatus = "complete"
	V2FinalizationIncomplete V2FinalizationStatus = "incomplete"
	V2FinalizationFailed     V2FinalizationStatus = "failed"
)

// V2Finalization is the source-returned projection of the public terminal
// artifact. It contains no protected or authority-bearing values.
type V2Finalization struct {
	Status               V2FinalizationStatus
	EmittedCaseCount     int
	BridgeFailurePresent bool
	ReasonCode           V2FailureCode
}

type v2FailureArtifact struct {
	ContractVersion string        `json:"contract_version"`
	CandidateRunID  string        `json:"candidate_run_id"`
	Artifact        string        `json:"artifact"`
	FailureCode     V2FailureCode `json:"failure_code"`
	Case            string        `json:"case,omitempty"`
	GeneratedAt     string        `json:"generated_at"`
}

type v2FinalizationArtifact struct {
	ContractVersion      string               `json:"contract_version"`
	CandidateRunID       string               `json:"candidate_run_id"`
	Artifact             string               `json:"artifact"`
	Status               V2FinalizationStatus `json:"status"`
	EmittedCaseCount     int                  `json:"emitted_case_count"`
	BridgeFailurePresent bool                 `json:"bridge_failure_present"`
	ReasonCode           V2FailureCode        `json:"reason_code,omitempty"`
	GeneratedAt          string               `json:"generated_at"`
}

const (
	v2FailureArtifactNamePrefix      = "bridge-failure-v2-"
	v2FinalizationArtifactNamePrefix = "bridge-finalization-v2-"
	v2CandidateControlDirectoryName  = "candidate-control"
)

// EmitWithFailure publishes a normal v2 receipt and, when a recognized case
// cannot be published, records one fixed public failure artifact. The returned
// error remains local to the evidence bridge caller and cannot change an
// already-determined manifest or Field workflow result.
func (w *V2Writer) EmitWithFailure(observation V2Observation) error {
	err := w.Emit(observation)
	if err == nil {
		return nil
	}
	if _, known := expectedOutcomes[observation.Case]; !known {
		return err
	}
	if failureErr := w.recordFailure(v2FailureCodeFor(err), observation.Case); failureErr != nil {
		return fmt.Errorf("record manifest v2 bridge failure: %w", failureErr)
	}
	return err
}

// FinalizeV2 writes exactly one public terminal lifecycle artifact after the
// candidate and any separately provisioned Field evidence activity have ended.
// It is source-only and does not start, stop, mount, or alter any runtime.
func FinalizeV2(runID, directory string, now func() time.Time) (V2Finalization, error) {
	if !runIDPattern.MatchString(runID) {
		return V2Finalization{}, errors.New("manifest v2 finalization run ID is invalid")
	}
	if err := assertRegularDirectory(directory); err != nil {
		return V2Finalization{}, err
	}
	if now == nil {
		now = time.Now
	}

	expected := expectedV2ReceiptNames(runID)
	failureName := v2FailureArtifactName(runID)
	finalizationName := v2FinalizationArtifactName(runID)
	entries, err := os.ReadDir(directory)
	if err != nil {
		return V2Finalization{}, fmt.Errorf("read manifest v2 receipt directory: %w", err)
	}

	present := make(map[string]struct{}, len(expected))
	bridgeFailurePresent := false
	unexpected := false
	for _, entry := range entries {
		if entry.Name() == finalizationName {
			return V2Finalization{}, ErrDuplicateReceipt
		}
		// The hardened runtime wrapper owns this restricted directory for
		// transient PID and output artifacts. It is the sole non-receipt
		// directory allowed in a v2 run; all other non-regular entries fail.
		if entry.Name() == v2CandidateControlDirectoryName && entry.IsDir() {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			unexpected = true
			continue
		}
		switch entry.Name() {
		case failureName:
			bridgeFailurePresent = true
		default:
			if _, known := expected[entry.Name()]; known {
				present[entry.Name()] = struct{}{}
			} else {
				unexpected = true
			}
		}
	}

	missing := len(expected) - len(present)
	result := V2Finalization{Status: V2FinalizationComplete, EmittedCaseCount: len(present), BridgeFailurePresent: bridgeFailurePresent}
	switch {
	case unexpected:
		result.Status = V2FinalizationFailed
		result.ReasonCode = V2FailureUnexpectedArtifact
	case bridgeFailurePresent:
		result.Status = V2FinalizationFailed
		result.ReasonCode = V2FailureBridgeFailure
	case missing > 0:
		result.Status = V2FinalizationIncomplete
		result.ReasonCode = V2FailureMissingExpectedCase
	}

	artifact := v2FinalizationArtifact{
		ContractVersion:      "2",
		CandidateRunID:       runID,
		Artifact:             "bridge_finalization",
		Status:               result.Status,
		EmittedCaseCount:     result.EmittedCaseCount,
		BridgeFailurePresent: result.BridgeFailurePresent,
		ReasonCode:           result.ReasonCode,
		GeneratedAt:          now().UTC().Format(time.RFC3339),
	}
	if err := writeV2PublicArtifact(directory, finalizationName, artifact); err != nil {
		return V2Finalization{}, err
	}
	return result, nil
}

func (w *V2Writer) recordFailure(code V2FailureCode, observedCase Case) error {
	if w == nil || w.now == nil {
		return errors.New("manifest v2 receipt writer is not configured")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := assertRegularDirectory(w.directory); err != nil {
		return err
	}
	name := v2FailureArtifactName(w.runID)
	if _, err := os.Lstat(filepath.Join(w.directory, name)); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("inspect manifest v2 bridge failure target: %w", err)
	}
	return writeV2PublicArtifact(w.directory, name, v2FailureArtifact{
		ContractVersion: "2",
		CandidateRunID:  w.runID,
		Artifact:        "bridge_failure",
		FailureCode:     code,
		Case:            string(observedCase),
		GeneratedAt:     w.now().UTC().Format(time.RFC3339),
	})
}

func v2FailureCodeFor(err error) V2FailureCode {
	switch {
	case errors.Is(err, ErrDuplicateReceipt):
		return V2FailureDuplicateCase
	case errors.Is(err, ErrUnsafeDirectory):
		return V2FailureUnsafeDirectory
	default:
		return V2FailureReceiptPublish
	}
}

func expectedV2ReceiptNames(runID string) map[string]struct{} {
	result := make(map[string]struct{}, len(expectedOutcomes))
	for c := range expectedOutcomes {
		result["receipt-v2-"+string(c)+"-"+runID+".json"] = struct{}{}
	}
	return result
}

func v2FailureArtifactName(runID string) string {
	return v2FailureArtifactNamePrefix + runID + ".json"
}

func v2FinalizationArtifactName(runID string) string {
	return v2FinalizationArtifactNamePrefix + runID + ".json"
}

func writeV2PublicArtifact(directory, name string, value any) error {
	if filepath.Base(name) != name {
		return ErrUnsafeDirectory
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal manifest v2 lifecycle artifact: %w", err)
	}
	if len(encoded) > maxReceiptBytes {
		return errors.New("manifest v2 lifecycle artifact exceeds public size limit")
	}
	temporary, err := os.CreateTemp(directory, ".manifest-v2-lifecycle-*.tmp")
	if err != nil {
		return fmt.Errorf("create manifest v2 lifecycle temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("restrict manifest v2 lifecycle temporary file: %w", err)
	}
	if _, err := temporary.Write(encoded); err != nil {
		temporary.Close()
		return fmt.Errorf("write manifest v2 lifecycle artifact: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync manifest v2 lifecycle artifact: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close manifest v2 lifecycle temporary file: %w", err)
	}
	if err := os.Link(temporaryPath, filepath.Join(directory, name)); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return ErrDuplicateReceipt
		}
		return fmt.Errorf("publish manifest v2 lifecycle artifact: %w", err)
	}
	if runtime.GOOS != "windows" {
		directoryHandle, err := os.Open(directory)
		if err != nil {
			return fmt.Errorf("open manifest v2 lifecycle directory for sync: %w", err)
		}
		defer directoryHandle.Close()
		if err := directoryHandle.Sync(); err != nil {
			return fmt.Errorf("sync manifest v2 lifecycle directory: %w", err)
		}
	}
	return nil
}

// ExpectedV2Cases returns the stable public case inventory for source-only
// tests and finalization callers. It does not expose any protected values.
func ExpectedV2Cases() []Case {
	result := make([]Case, 0, len(expectedOutcomes))
	for c := range expectedOutcomes {
		result = append(result, c)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
