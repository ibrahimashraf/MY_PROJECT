// Package manifestreceipts provides source-owned, public-redacted receipt types.
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
	"sync"
	"time"
)

// V2EventSource records the redacted boundary that produced a receipt.
type V2EventSource string

const (
	V2EventSourceManifestHTTP V2EventSource = "manifest_http"
	V2EventSourceFieldBinding V2EventSource = "field_binding"
)

// V2TransportKind distinguishes a real HTTP response from a Field bind event.
type V2TransportKind string

const (
	V2TransportHTTP         V2TransportKind = "http"
	V2TransportFieldBinding V2TransportKind = "field_binding"
)

// V2ReplayScope states whether aggregate replay arithmetic is trustworthy.
type V2ReplayScope string

const (
	V2ReplayScopeVerifiedTenant V2ReplayScope = "verified_tenant"
	V2ReplayScopeNotApplicable  V2ReplayScope = "not_applicable"
)

var v2CorrelationIDPattern = regexp.MustCompile(`^[a-f0-9]{8,128}$`)

// V2Transport contains only a source-observed transport result. Field binding
// intentionally has no invented HTTP status.
type V2Transport struct {
	Kind       V2TransportKind
	HTTPStatus *int
}

// V2ReplayState publishes arithmetic only after a verified tenant context.
type V2ReplayState struct {
	Scope  V2ReplayScope
	Before *int64
	After  *int64
	Delta  *int64
}

// V2Observation is a closed, source-redacted projection for one public case.
// It contains no proofs, signatures, identities, credentials, payloads, or keys.
type V2Observation struct {
	Case            Case
	EventSource     V2EventSource
	ObservedOutcome string
	Transport       V2Transport
	ReplayState     V2ReplayState
	ManifestID      string
	CorrelationID   string
	GeneratedAt     time.Time
}

// V2Writer atomically emits at most one public v2 receipt per case/run.
// It is source-only infrastructure; constructing it does not mount a route or
// make any receipt mandatory for an authoritative workflow.
type V2Writer struct {
	runID     string
	directory string
	now       func() time.Time
	mu        sync.Mutex
	emitted   map[Case]struct{}
}

// NewV2Writer validates the nonsecret opaque run ID and a regular receipt directory.
func NewV2Writer(runID, directory string, now func() time.Time) (*V2Writer, error) {
	if !runIDPattern.MatchString(runID) {
		return nil, errors.New("manifest v2 receipt run ID is invalid")
	}
	if err := assertRegularDirectory(directory); err != nil {
		return nil, err
	}
	if now == nil {
		now = time.Now
	}
	return &V2Writer{runID: runID, directory: directory, now: now, emitted: make(map[Case]struct{})}, nil
}

// Emit writes a strict v2 receipt. Structural boundary violations return an
// error and write nothing; an expected-outcome or valid-proof delta mismatch is
// represented as a failed public receipt for synthetic/adversarial testing.
func (w *V2Writer) Emit(observation V2Observation) error {
	if w == nil || w.now == nil {
		return errors.New("manifest v2 receipt writer is not configured")
	}
	if err := validateV2Observation(observation); err != nil {
		return err
	}

	generatedAt := observation.GeneratedAt.UTC()
	if generatedAt.IsZero() {
		generatedAt = w.now().UTC()
	}
	passed := observation.ObservedOutcome == expectedOutcomes[observation.Case] && validV2Delta(observation)
	receipt := v2Receipt{
		ContractVersion:  "2",
		CandidateRunID:   w.runID,
		Case:             string(observation.Case),
		ExpectedOutcome:  expectedOutcomes[observation.Case],
		ObservedOutcome:  observation.ObservedOutcome,
		Status:           statusFor(passed),
		EventSource:      string(observation.EventSource),
		Transport:        v2Transport{Kind: string(observation.Transport.Kind), HTTPStatus: observation.Transport.HTTPStatus},
		ReplayState:      projectV2ReplayState(observation.ReplayState),
		GeneratedAt:      generatedAt.Format(time.RFC3339),
		CorrelationID:    observation.CorrelationID,
		ManifestIDDigest: digestManifestID(observation.ManifestID),
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("marshal manifest v2 receipt: %w", err)
	}
	if len(encoded) > maxReceiptBytes {
		return errors.New("manifest v2 receipt exceeds public size limit")
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if _, exists := w.emitted[observation.Case]; exists {
		return ErrDuplicateReceipt
	}
	if err := assertRegularDirectory(w.directory); err != nil {
		return err
	}
	filename := "receipt-v2-" + string(observation.Case) + "-" + w.runID + ".json"
	if filepath.Base(filename) != filename {
		return ErrUnsafeDirectory
	}
	finalPath := filepath.Join(w.directory, filename)
	if _, err := os.Lstat(finalPath); err == nil {
		return ErrDuplicateReceipt
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect manifest v2 receipt target: %w", err)
	}
	temporary, err := os.CreateTemp(w.directory, ".receipt-v2-*.tmp")
	if err != nil {
		return fmt.Errorf("create manifest v2 receipt temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("restrict manifest v2 receipt temporary file: %w", err)
	}
	if _, err := temporary.Write(encoded); err != nil {
		temporary.Close()
		return fmt.Errorf("write manifest v2 receipt: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync manifest v2 receipt: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close manifest v2 receipt: %w", err)
	}
	if err := os.Link(temporaryPath, finalPath); err != nil {
		if errors.Is(err, os.ErrExist) {
			return ErrDuplicateReceipt
		}
		return fmt.Errorf("publish manifest v2 receipt: %w", err)
	}
	w.emitted[observation.Case] = struct{}{}
	return nil
}

func validateV2Observation(observation V2Observation) error {
	if _, ok := expectedOutcomes[observation.Case]; !ok {
		return errors.New("manifest v2 receipt case is invalid")
	}
	if _, ok := allowedObservedOutcomes[observation.ObservedOutcome]; !ok {
		return errors.New("manifest v2 receipt observed outcome is invalid")
	}
	if observation.CorrelationID != "" && !v2CorrelationIDPattern.MatchString(observation.CorrelationID) {
		return errors.New("manifest v2 receipt correlation ID is invalid")
	}
	if observation.Case == CaseFieldBinding {
		if observation.EventSource != V2EventSourceFieldBinding || observation.Transport.Kind != V2TransportFieldBinding || observation.Transport.HTTPStatus != nil {
			return errors.New("manifest v2 field binding transport is invalid")
		}
		return validateV2NotApplicableReplayState(observation.ReplayState)
	}
	if observation.EventSource != V2EventSourceManifestHTTP || observation.Transport.Kind != V2TransportHTTP || observation.Transport.HTTPStatus == nil {
		return errors.New("manifest v2 HTTP transport is invalid")
	}
	if !allowedHTTPStatus(observation.Case, *observation.Transport.HTTPStatus) {
		return errors.New("manifest v2 HTTP status is invalid for case")
	}
	if observation.Case != CaseValidProof {
		return validateV2NotApplicableReplayState(observation.ReplayState)
	}
	if observation.ReplayState.Scope != V2ReplayScopeVerifiedTenant || observation.ReplayState.Before == nil || observation.ReplayState.After == nil || observation.ReplayState.Delta == nil {
		return errors.New("manifest v2 verified replay state is invalid")
	}
	if *observation.ReplayState.Before < 0 || *observation.ReplayState.After < *observation.ReplayState.Before || *observation.ReplayState.Delta != *observation.ReplayState.After-*observation.ReplayState.Before {
		return errors.New("manifest v2 replay state counts are invalid")
	}
	return nil
}

func validateV2NotApplicableReplayState(state V2ReplayState) error {
	if state.Scope != V2ReplayScopeNotApplicable || state.Before != nil || state.After != nil || state.Delta != nil {
		return errors.New("manifest v2 replay state must be not applicable")
	}
	return nil
}

func validV2Delta(observation V2Observation) bool {
	if observation.Case != CaseValidProof {
		return true
	}
	return observation.ReplayState.Delta != nil && *observation.ReplayState.Delta == 1
}

func allowedHTTPStatus(c Case, status int) bool {
	switch c {
	case CaseValidProof:
		return status == 200
	case CaseReplay:
		return status == 409
	case CaseSignatureInvalid, CaseExpired, CaseKeyUnknown:
		return status == 401
	case CasePackageHashInvalid:
		return status == 403
	case CaseAuthorityMismatch:
		return status == 401 || status == 403
	default:
		return false
	}
}

func projectV2ReplayState(state V2ReplayState) v2ReplayState {
	return v2ReplayState{Scope: string(state.Scope), Before: state.Before, After: state.After, Delta: state.Delta}
}

func digestManifestID(manifestID string) string {
	if manifestID == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(manifestID))
	return "sha256:" + hex.EncodeToString(digest[:])
}

type v2Transport struct {
	Kind       string `json:"kind"`
	HTTPStatus *int   `json:"http_status,omitempty"`
}

type v2ReplayState struct {
	Scope  string `json:"scope"`
	Before *int64 `json:"before,omitempty"`
	After  *int64 `json:"after,omitempty"`
	Delta  *int64 `json:"delta,omitempty"`
}

type v2Receipt struct {
	ContractVersion  string        `json:"contract_version"`
	CandidateRunID   string        `json:"candidate_run_id"`
	Case             string        `json:"case"`
	ExpectedOutcome  string        `json:"expected_outcome"`
	ObservedOutcome  string        `json:"observed_outcome"`
	Status           string        `json:"status"`
	EventSource      string        `json:"event_source"`
	Transport        v2Transport   `json:"transport"`
	ReplayState      v2ReplayState `json:"replay_state"`
	CorrelationID    string        `json:"correlation_id,omitempty"`
	ManifestIDDigest string        `json:"manifest_id_digest,omitempty"`
	GeneratedAt      string        `json:"generated_at"`
}
