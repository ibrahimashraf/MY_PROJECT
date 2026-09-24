package assurance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid/v5"
)

// AuditLedgerEntry represents a single link in the cryptographic hash chain.
type AuditLedgerEntry struct {
	CID             string    `json:"cid"`              // SHA-256 of the canonical JSON form of this struct (excluding CID)
	TenantID        uuid.UUID `json:"tenant_id"`
	AssetID         uuid.UUID `json:"asset_id"`
	SequenceNumber  uint64    `json:"seq_num"`
	RecordHash      string    `json:"record_hash"`      // Cryptographic hash of the Assurance Record
	PreviousCID     string    `json:"prev_cid"`         // Distance 1 predecessor
	EpochCheckpoint string    `json:"epoch_cid"`        // Root hash of previous calendar year
	Timestamp       time.Time `json:"timestamp"`
}

// GenerateCID computes the SHA-256 Content ID for the ledger entry.
func (e *AuditLedgerEntry) GenerateCID() (string, error) {
	// Create an identical struct without the CID field to ensure stable hashing
	type canonicalEntry struct {
		TenantID        uuid.UUID `json:"tenant_id"`
		AssetID         uuid.UUID `json:"asset_id"`
		SequenceNumber  uint64    `json:"seq_num"`
		RecordHash      string    `json:"record_hash"`
		PreviousCID     string    `json:"prev_cid"`
		EpochCheckpoint string    `json:"epoch_cid"`
		Timestamp       int64     `json:"timestamp"` // Use Unix timestamp for stability
	}

	canonical := canonicalEntry{
		TenantID:        e.TenantID,
		AssetID:         e.AssetID,
		SequenceNumber:  e.SequenceNumber,
		RecordHash:      e.RecordHash,
		PreviousCID:     e.PreviousCID,
		EpochCheckpoint: e.EpochCheckpoint,
		Timestamp:       e.Timestamp.UTC().Unix(),
	}

	hashBytes, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ledger entry for hashing: %w", err)
	}

	hasher := sha256.New()
	hasher.Write(hashBytes)
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), nil
}

// LedgerBuilder handles the construction of linear hash chains.
type LedgerBuilder struct{}

// BuildNextEntry creates a cryptographically bound ledger entry.
func (b *LedgerBuilder) BuildNextEntry(
	tenantID uuid.UUID,
	assetID uuid.UUID,
	seqNum uint64,
	recordHash string,
	prevCID string,
	epochCID string,
	timestamp time.Time,
) (*AuditLedgerEntry, error) {
	
	entry := &AuditLedgerEntry{
		TenantID:        tenantID,
		AssetID:         assetID,
		SequenceNumber:  seqNum,
		RecordHash:      recordHash,
		PreviousCID:     prevCID,
		EpochCheckpoint: epochCID,
		Timestamp:       timestamp.UTC(), // Force UTC
	}

	cid, err := entry.GenerateCID()
	if err != nil {
		return nil, err
	}
	entry.CID = cid

	return entry, nil
}
