// Package auditcheckpoint defines a pure canonical-root contract. It does not
// query, persist, sign, authorize, or expose audit records.
package auditcheckpoint

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	Version     = "audit-checkpoint/v1"
	GenesisRoot = "GENESIS"
)

type Checkpoint struct {
	CheckpointVersion  string  `json:"checkpoint_version"`
	TenantID           string  `json:"tenant_id"`
	Environment        string  `json:"environment"`
	SequenceStart      uint64  `json:"sequence_start"`
	SequenceEnd        uint64  `json:"sequence_end"`
	PreviousRootSHA256 string  `json:"previous_root_sha256"`
	Entries            []Entry `json:"entries"`
	RootSHA256         string  `json:"root_sha256"`
}

// Entry holds allowlisted audit metadata only. ContextSHA256 protects the
// original context without placing raw context, payload, tokens, or evidence in
// a checkpoint export.
type Entry struct {
	Sequence      uint64    `json:"sequence"`
	RecordID      string    `json:"record_id"`
	TenantID      string    `json:"tenant_id"`
	Environment   string    `json:"environment"`
	ActorType     string    `json:"actor_type"`
	ActorID       string    `json:"actor_id"`
	Action        string    `json:"action"`
	ResourceType  string    `json:"resource_type"`
	ResourceID    string    `json:"resource_id"`
	Outcome       string    `json:"outcome"`
	CorrelationID string    `json:"correlation_id"`
	ContextSHA256 string    `json:"context_sha256"`
	CreatedAt     time.Time `json:"created_at"`
}

// Seal computes a deterministic SHA-256 canonical root. It performs no
// signature operation and does not read or write external systems.
func (c *Checkpoint) Seal() error {
	payload, err := c.canonicalPayload()
	if err != nil {
		return err
	}
	sum := sha256.Sum256(payload)
	c.RootSHA256 = hex.EncodeToString(sum[:])
	return nil
}

// Verify recomputes the canonical root. A valid root is integrity evidence for
// this payload only; it is not an attestation signature or storage proof.
func (c Checkpoint) Verify() error {
	if !isSHA256(c.RootSHA256) {
		return errors.New("root_sha256 must be a lowercase SHA-256 digest")
	}
	payload, err := c.canonicalPayload()
	if err != nil {
		return err
	}
	sum := sha256.Sum256(payload)
	if c.RootSHA256 != hex.EncodeToString(sum[:]) {
		return errors.New("root_sha256 does not match canonical payload")
	}
	return nil
}

// CanonicalPayload returns the exact deterministic JSON byte payload covered by
// RootSHA256. RootSHA256 itself is intentionally not included.
func (c Checkpoint) CanonicalPayload() ([]byte, error) {
	return c.canonicalPayload()
}

func (c Checkpoint) canonicalPayload() ([]byte, error) {
	if err := c.validateFields(); err != nil {
		return nil, err
	}
	entries := append([]Entry(nil), c.Entries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Sequence < entries[j].Sequence })

	canonical := struct {
		CheckpointVersion  string  `json:"checkpoint_version"`
		TenantID           string  `json:"tenant_id"`
		Environment        string  `json:"environment"`
		SequenceStart      uint64  `json:"sequence_start"`
		SequenceEnd        uint64  `json:"sequence_end"`
		PreviousRootSHA256 string  `json:"previous_root_sha256"`
		Entries            []Entry `json:"entries"`
	}{
		CheckpointVersion:  c.CheckpointVersion,
		TenantID:           c.TenantID,
		Environment:        c.Environment,
		SequenceStart:      c.SequenceStart,
		SequenceEnd:        c.SequenceEnd,
		PreviousRootSHA256: c.PreviousRootSHA256,
		Entries:            entries,
	}
	return json.Marshal(canonical)
}

func (c Checkpoint) validateFields() error {
	for field, value := range map[string]string{
		"checkpoint_version": c.CheckpointVersion,
		"tenant_id":          c.TenantID,
		"environment":        c.Environment,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if c.CheckpointVersion != Version {
		return fmt.Errorf("checkpoint_version = %q, want %q", c.CheckpointVersion, Version)
	}
	if c.SequenceStart == 0 || c.SequenceEnd < c.SequenceStart {
		return errors.New("checkpoint sequence bounds are invalid")
	}
	if !validPreviousRoot(c.PreviousRootSHA256) {
		return errors.New("previous_root_sha256 must be GENESIS or a lowercase SHA-256 digest")
	}
	if len(c.Entries) == 0 {
		return errors.New("entries are required")
	}
	if uint64(len(c.Entries)) != c.SequenceEnd-c.SequenceStart+1 {
		return errors.New("sequence bounds do not match entry count")
	}

	seenRecords := make(map[string]struct{}, len(c.Entries))
	seenSequences := make(map[uint64]struct{}, len(c.Entries))
	for _, entry := range c.Entries {
		if err := entry.validate(c.TenantID, c.Environment); err != nil {
			return err
		}
		if _, exists := seenSequences[entry.Sequence]; exists {
			return fmt.Errorf("duplicate sequence %d", entry.Sequence)
		}
		if _, exists := seenRecords[entry.RecordID]; exists {
			return fmt.Errorf("duplicate record_id %q", entry.RecordID)
		}
		seenSequences[entry.Sequence] = struct{}{}
		seenRecords[entry.RecordID] = struct{}{}
	}
	for sequence := c.SequenceStart; sequence <= c.SequenceEnd; sequence++ {
		if _, exists := seenSequences[sequence]; !exists {
			return fmt.Errorf("missing sequence %d", sequence)
		}
	}
	return nil
}

func (e Entry) validate(tenantID, environment string) error {
	for field, value := range map[string]string{
		"record_id":      e.RecordID,
		"tenant_id":      e.TenantID,
		"environment":    e.Environment,
		"actor_type":     e.ActorType,
		"actor_id":       e.ActorID,
		"action":         e.Action,
		"resource_type":  e.ResourceType,
		"resource_id":    e.ResourceID,
		"outcome":        e.Outcome,
		"correlation_id": e.CorrelationID,
		"context_sha256": e.ContextSHA256,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("entry %s is required", field)
		}
	}
	if e.Sequence == 0 {
		return errors.New("entry sequence is required")
	}
	if e.TenantID != tenantID || e.Environment != environment {
		return errors.New("entry tenant/environment must match checkpoint scope")
	}
	if e.CreatedAt.IsZero() {
		return errors.New("entry created_at is required")
	}
	if !isSHA256(e.ContextSHA256) {
		return errors.New("entry context_sha256 must be a lowercase SHA-256 digest")
	}
	return nil
}

func validPreviousRoot(value string) bool {
	return value == GenesisRoot || isSHA256(value)
}

func isSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}
