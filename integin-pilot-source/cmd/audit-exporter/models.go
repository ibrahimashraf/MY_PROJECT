package main

import "time"

// ExportRequest describes the tenant/org-scoped window of audit records a
// caller wants sealed into an export receipt.
type ExportRequest struct {
	TenantID       string    `json:"tenant_id"`
	OrganizationID string    `json:"organization_id"`
	FromTime       time.Time `json:"from_time"`
	ToTime         time.Time `json:"to_time"`
	EntityType     string    `json:"entity_type,omitempty"`
}

// ExportReceipt is the sealed, self-verifying proof of an audit chain export.
type ExportReceipt struct {
	ReceiptID         string    `json:"receipt_id"`
	TenantID          string    `json:"tenant_id"`
	OrganizationID    string    `json:"organization_id"`
	RecordCount       int       `json:"record_count"`
	MerkleRoot        string    `json:"merkle_root"`
	ExportedAt        time.Time `json:"exported_at"`
	Signatures        []string  `json:"signatures,omitempty"`
	VerifiedIntegrity bool      `json:"verified_integrity"`
}

// AuditRecord is one hashed, chain-linked entry in a sequential audit log.
type AuditRecord struct {
	ID           string    `json:"id"`
	Sequence     uint64    `json:"sequence"`
	Timestamp    time.Time `json:"timestamp"`
	ActorID      string    `json:"actor_id"`
	Action       string    `json:"action"`
	PayloadHash  string    `json:"payload_hash"`
	PreviousHash string    `json:"previous_hash"`
}

// ExportReceiptID returns the stable, deterministic identifier of a receipt;
// kept as a small alias so JSON round-trips never re-derive a flaky value.
func ExportReceiptID(r *ExportReceipt) string {
	if r == nil {
		return ""
	}
	return r.ReceiptID
}
