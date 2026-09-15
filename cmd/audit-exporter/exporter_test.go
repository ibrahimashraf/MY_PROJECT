package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// buildChain produces count records linked into a contiguous, ascending,
// GenesisHash-anchored chain. Hashes are computed with the same padding body
// the exporter itself uses, so BuildReceipt accepts the fixture untouched.
func buildChain(count int) []AuditRecord {
	records := make([]AuditRecord, 0, count)
	prev := GenesisHash
	for i := 0; i < count; i++ {
		rec := AuditRecord{
			ID:        fmt.Sprintf("rec-%d", i),
			Sequence:  uint64(i),
			Timestamp: time.Date(2026, 9, 16, 9, i, 0, 0, time.UTC),
			ActorID:   fmt.Sprintf("actor-%d", i%3),
			Action:    fmt.Sprintf("action-%d", i%5),
		}
		rec.PreviousHash = prev
		rec.PayloadHash = ComputeRecordHash(prev, rec)
		prev = rec.PayloadHash
		records = append(records, rec)
	}
	return records
}

func TestNewExporterBuildsReceiptForValidChain(t *testing.T) {
	exporter := NewExporter()
	records := buildChain(4)
	receipt, err := exporter.BuildReceipt(records, "tenant-acme", "org-eu")
	if err != nil {
		t.Fatalf("BuildReceipt returned error for a valid chain: %v", err)
	}
	if receipt == nil {
		t.Fatal("BuildReceipt returned a nil receipt")
	}
	if receipt.TenantID != "tenant-acme" {
		t.Errorf("TenantID = %q, want tenant-acme", receipt.TenantID)
	}
	if receipt.OrganizationID != "org-eu" {
		t.Errorf("OrganizationID = %q, want org-eu", receipt.OrganizationID)
	}
	if receipt.RecordCount != 4 {
		t.Errorf("RecordCount = %d, want 4", receipt.RecordCount)
	}
	if len(receipt.MerkleRoot) != 64 {
		t.Errorf("MerkleRoot %q has length %d, want 64 hex chars", receipt.MerkleRoot, len(receipt.MerkleRoot))
	}
	if !VerifyReceipt(receipt) {
		t.Error("VerifyReceipt rejected a receipt built from a valid chain")
	}
}

func TestBuildReceiptRejectsEmptyChain(t *testing.T) {
	exporter := NewExporter()
	_, err := exporter.BuildReceipt(nil, "tenant-acme", "org-eu")
	if !errors.Is(err, ErrEmptyChain) {
		t.Fatalf("BuildReceipt(nil) error = %v, want ErrEmptyChain", err)
	}
}

func TestBuildReceiptRejectsSequenceGap(t *testing.T) {
	exporter := NewExporter()
	records := buildChain(4)
	records[2].Sequence = 9
	_, err := exporter.BuildReceipt(records, "tenant-acme", "org-eu")
	if !errors.Is(err, ErrSequenceGap) {
		t.Fatalf("BuildReceipt with a sequence gap error = %v, want ErrSequenceGap", err)
	}
}

func TestBuildReceiptRejectsDuplicateSequence(t *testing.T) {
	exporter := NewExporter()
	records := buildChain(4)
	records[3].Sequence = records[2].Sequence
	_, err := exporter.BuildReceipt(records, "tenant-acme", "org-eu")
	if !errors.Is(err, ErrSequenceGap) {
		t.Fatalf("BuildReceipt with a duplicate sequence error = %v, want ErrSequenceGap", err)
	}
}

func TestBuildReceiptRejectsBrokenLink(t *testing.T) {
	exporter := NewExporter()
	records := buildChain(4)
	records[2].PreviousHash = strings.Repeat("0", 61) + "abc"
	_, err := exporter.BuildReceipt(records, "tenant-acme", "org-eu")
	if !errors.Is(err, ErrChainBroken) && !errors.Is(err, ErrHashMismatch) {
		t.Fatalf("BuildReceipt with broken link error = %v, want ErrChainBroken", err)
	}
}

func TestBuildReceiptRejectsHashTamper(t *testing.T) {
	exporter := NewExporter()
	records := buildChain(4)
	records[1].PayloadHash = strings.Repeat("deadbeef", 8)
	_, err := exporter.BuildReceipt(records, "tenant-acme", "org-eu")
	if !errors.Is(err, ErrHashMismatch) {
		t.Fatalf("BuildReceipt with tampered payload hash error = %v, want ErrHashMismatch", err)
	}
}

func TestBuildReceiptRequiresTenant(t *testing.T) {
	exporter := NewExporter()
	records := buildChain(3)
	_, err := exporter.BuildReceipt(records, "", "org-eu")
	if !errors.Is(err, ErrTenantRequired) {
		t.Fatalf("BuildReceipt with empty tenant error = %v, want ErrTenantRequired", err)
	}
}

func TestBuildReceiptRequiresOrg(t *testing.T) {
	exporter := NewExporter()
	records := buildChain(3)
	_, err := exporter.BuildReceipt(records, "tenant-acme", "")
	if !errors.Is(err, ErrOrgIDRequired) {
		t.Fatalf("BuildReceipt with empty org error = %v, want ErrOrgIDRequired", err)
	}
}

func TestVerifyReceiptRejectsTamperedReceipt(t *testing.T) {
	exporter := NewExporter()
	receipt, err := exporter.BuildReceipt(buildChain(4), "tenant-acme", "org-eu")
	if err != nil {
		t.Fatalf("BuildReceipt: %v", err)
	}
	receipt.VerifiedIntegrity = false
	receipt.RecordCount = 99
	if VerifyReceipt(receipt) {
		t.Error("VerifyReceipt accepted a tampered receipt")
	}
}

func TestComputeRecordHashIsDeterministic(t *testing.T) {
	rec := AuditRecord{
		ID:        "rec-0",
		Sequence:  0,
		Timestamp: time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC),
		ActorID:   "actor-0",
		Action:    "action-0",
	}
	a := ComputeRecordHash(GenesisHash, rec)
	b := ComputeRecordHash(GenesisHash, rec)
	if a == "" || len(a) != 64 || a != b {
		t.Fatalf("ComputeRecordHash not deterministic or malformed: %q vs %q", a, b)
	}
}
