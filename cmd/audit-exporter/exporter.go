package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"time"
)

// GenesisHash anchors the first record of every export chain. It mirrors the
// all-zero anchor used by internal/domain/auditlog.GenesisHash(). GenesisHash
// is also exported so that CLI output and tests can reference the anchor.
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

const sha256HexLen = 64

var (
	// ErrEmptyChain is returned when a seal is attempted over an empty chain.
	ErrEmptyChain = errors.New("audit-exporter: no records to export")

	// ErrNilRecord is returned when the input slice contains a zero-value
	// (nil) audit record.
	ErrNilRecord = errors.New("audit-exporter: nil audit record in chain")

	// ErrSequenceGap is returned when the chained records are not contiguous
	// (a gap, a duplicate, or a non-ascending sequence was detected).
	ErrSequenceGap = errors.New("audit-exporter: sequence gap or duplicate sequence detected")

	// ErrChainBroken is returned when previous_hash does not link to the prior
	// record's payload_hash.
	ErrChainBroken = errors.New("audit-exporter: previous_hash does not link to prior record's payload_hash")

	// ErrHashMismatch is returned when payload_hash does not match the hash
	// recomputed from the record's canonical fields.
	ErrHashMismatch = errors.New("audit-exporter: payload_hash does not match the recomputed hash")

	// ErrHashBadLength is returned when payload_hash is not 64 hex characters.
	ErrHashBadLength = errors.New("audit-exporter: payload_hash has an invalid length")

	// ErrHashNotLowerHex is returned when payload_hash is not lowercase hex.
	ErrHashNotLowerHex = errors.New("audit-exporter: payload_hash is not lowercase hex")

	// ErrHashNotHex is returned when payload_hash contains a non-hex byte.
	ErrHashNotHex = errors.New("audit-exporter: payload_hash contains a non-hex character")

	// ErrTenantRequired is returned when tenant_id is empty.
	ErrTenantRequired = errors.New("audit-exporter: tenant_id is required")

	// ErrOrgIDRequired is returned when organization_id is empty.
	ErrOrgIDRequired = errors.New("audit-exporter: organization_id is required")
)

// ComputeRecordHash deterministically hashes a record chained onto
// previousHash. The canonical body is:
//
//	previousHash | sequence | timestamp(RFC3339Nano, UTC) | actor_id | action
//
// so that identical logical events always yield identical digests and the
// chain is recomputable offline by an independent verifier.
func ComputeRecordHash(previousHash string, rec AuditRecord) string {
	h := sha256.New()
	h.Write([]byte(previousHash))
	h.Write([]byte{'|'})
	h.Write(uint64Bytes(rec.Sequence))
	h.Write([]byte{'|'})
	h.Write([]byte(rec.Timestamp.UTC().Format(time.RFC3339Nano)))
	h.Write([]byte{'|'})
	h.Write([]byte(rec.ActorID))
	h.Write([]byte{'|'})
	h.Write([]byte(rec.Action))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func uint64Bytes(v uint64) []byte {
	var b [8]byte
	for i := 7; i >= 0; i-- {
		b[i] = byte(v)
		v >>= 8
	}
	return b[:]
}

func isLowerHex(s string) bool {
	if len(s) != sha256HexLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		default:
			return false
		}
	}
	return true
}

func nodeHash(l, r [32]byte) [32]byte {
	h := sha256.New()
	h.Write(l[:])
	h.Write(r[:])
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

func merkleRoot(leaves [][32]byte) [32]byte {
	if len(leaves) == 0 {
		return sha256.Sum256(nil)
	}
	level := make([][32]byte, len(leaves))
	copy(level, leaves)
	for len(level) > 1 {
		next := make([][32]byte, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			if i+1 < len(level) {
				next = append(next, nodeHash(level[i], level[i+1]))
			} else {
				next = append(next, level[i])
			}
		}
		level = next
	}
	return level[0]
}

func deterministicID(tenantID, orgID string, count int, rootHex string) string {
	h := sha256.New()
	h.Write([]byte(tenantID))
	h.Write([]byte{'|'})
	h.Write([]byte(orgID))
	h.Write([]byte{'|'})
	h.Write(uint64Bytes(uint64(count)))
	h.Write([]byte{'|'})
	h.Write([]byte(rootHex))
	return "REC-" + fmt.Sprintf("%x", h.Sum(nil))
}

func deterministicSignature(receipt *ExportReceipt) string {
	h := sha256.New()
	h.Write([]byte(receipt.ReceiptID))
	h.Write([]byte{'|'})
	h.Write([]byte(receipt.TenantID))
	h.Write([]byte{'|'})
	h.Write([]byte(receipt.OrganizationID))
	h.Write([]byte{'|'})
	h.Write(uint64Bytes(uint64(receipt.RecordCount)))
	h.Write([]byte{'|'})
	h.Write([]byte(receipt.MerkleRoot))
	return "SIG-" + fmt.Sprintf("%x", h.Sum(nil))
}

// VerifyReceipt recomputes the deterministic signature over the receipt's
// stable facts and confirms at least one attached signature matches, i.e. the
// integrity claim is sound and the export was not silently altered.
func VerifyReceipt(receipt *ExportReceipt) bool {
	if receipt == nil {
		return false
	}
	want := deterministicSignature(receipt)
	for _, sig := range receipt.Signatures {
		if subtle.ConstantTimeCompare([]byte(sig), []byte(want)) == 1 {
			return true
		}
	}
	return false
}

// AuditExporter validates audit chains and seals them into export receipts.
type AuditExporter struct{}

// NewExporter returns a ready-to-use AuditExporter.
func NewExporter() *AuditExporter { return &AuditExporter{} }

func (e *AuditExporter) sortedCopy(records []AuditRecord) []AuditRecord {
	out := make([]AuditRecord, len(records))
	copy(out, records)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Sequence != out[j].Sequence {
			return out[i].Sequence < out[j].Sequence
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// validateChain ensures, in strict order:
//   - records are anchored: first previous_hash == GenesisHash
//   - each record links: previous_hash == prior record's payload_hash
//   - each payload_hash is well-formed and matches the recomputed hash
//
// Sequence values must be contiguous (no gap, no duplicate, ascending).
func (e *AuditExporter) validateChain(sorted []AuditRecord) error {
	if len(sorted) == 0 {
		return ErrEmptyChain
	}
	for i, rec := range sorted {
		if i > 0 && rec.Sequence <= sorted[i-1].Sequence {
			return ErrSequenceGap
		}
		if i > 0 && rec.Sequence > sorted[i-1].Sequence+1 {
			return ErrSequenceGap
		}
		wantLink := GenesisHash
		if i > 0 {
			wantLink = sorted[i-1].PayloadHash
		}
		if rec.PreviousHash != wantLink {
			return ErrChainBroken
		}
		if !isLowerHex(rec.PayloadHash) {
			if len(rec.PayloadHash) != sha256HexLen {
				return ErrHashBadLength
			}
			return ErrHashNotLowerHex
		}
		computed := ComputeRecordHash(rec.PreviousHash, rec)
		if computed != rec.PayloadHash {
			return ErrHashMismatch
		}
	}
	return nil
}

// BuildReceipt seals the given audit records into a deterministic,
// self-verifying export receipt: it validates the hash chain, sorts the
// records by ascending sequence, computes a Merkle root over the payload
// hashesable leaves, and signs the receipt's stable facts.
func (e *AuditExporter) BuildReceipt(records []AuditRecord, tenantID, orgID string) (*ExportReceipt, error) {
	if len(records) == 0 {
		return nil, ErrEmptyChain
	}
	if tenantID == "" {
		return nil, ErrTenantRequired
	}
	if orgID == "" {
		return nil, ErrOrgIDRequired
	}

	sorted := e.sortedCopy(records)
	if err := e.validateChain(sorted); err != nil {
		return nil, err
	}

	leaves := make([][32]byte, 0, len(sorted))
	for _, rec := range sorted {
		b, err := hex.DecodeString(rec.PayloadHash)
		if err != nil {
			return nil, ErrHashNotHex
		}
		var leaf [32]byte
		copy(leaf[:], b)
		leaves = append(leaves, leaf)
	}
	root := merkleRoot(leaves)
	rootHex := hex.EncodeToString(root[:])

	receipt := &ExportReceipt{
		TenantID:        tenantID,
		OrganizationID:  orgID,
		RecordCount:     len(sorted),
		MerkleRoot:      rootHex,
		ExportedAt:      time.Now().UTC().Truncate(time.Second),
		VerifiedIntegrity: true,
	}
	receipt.ReceiptID = deterministicID(tenantID, orgID, len(sorted), rootHex)
	receipt.Signatures = []string{deterministicSignature(receipt)}
	return receipt, nil
}
