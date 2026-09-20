// Package ledger is a compatibility alias over internal/domain/auditlog.
//
// GLOBAL_ARCHITECTURE_PLAN.md §4 names this path (pkg/ledger/append_only_log.go)
// for the Bitemporal Merkle-CRDT Audit Ledger (L10). The implementation lives
// in internal/domain/auditlog; this package re-exports its surface with zero
// new logic so both import paths resolve to the same 64-byte canonical record.
package ledger

import (
	"integin/internal/domain/auditlog"
)

// Entry and record surface.
type (
	Entry                 = auditlog.Entry
	CreateEntryRequest    = auditlog.CreateEntryRequest
	QueryRequest          = auditlog.QueryRequest
	QueryResponse         = auditlog.QueryResponse
	VerifyResult          = auditlog.VerifyResult
	Record                = auditlog.Record
	EntityType            = auditlog.EntityType
	Action                = auditlog.Action
	Repository            = auditlog.Repository
	TenantScope           = auditlog.TenantScope
	TenantAuditSummary    = auditlog.TenantAuditSummary
	TenantAuditAggregator = auditlog.TenantAuditAggregator
)

// Entity and action constants.
const (
	EntityInspection  = auditlog.EntityInspection
	EntityWorkOrder   = auditlog.EntityWorkOrder
	EntityCertificate = auditlog.EntityCertificate
	EntityAsset       = auditlog.EntityAsset

	ActionCreate  = auditlog.ActionCreate
	ActionUpdate  = auditlog.ActionUpdate
	ActionDelete  = auditlog.ActionDelete
	ActionApprove = auditlog.ActionApprove
	ActionReject  = auditlog.ActionReject
)

// Sentinel errors.
var (
	ErrInvalidEntry          = auditlog.ErrInvalidEntry
	ErrNotFound              = auditlog.ErrNotFound
	ErrChainBroken           = auditlog.ErrChainBroken
	ErrRecordInvalidPrevHash = auditlog.ErrRecordInvalidPrevHash
	ErrRecordPayloadTooLong  = auditlog.ErrRecordPayloadTooLong
)

// AppendRecord builds the canonical 64-byte hash-chain record. Zero allocations.
func AppendRecord(prevHashHex string, e Entry) (Record, error) {
	return auditlog.AppendRecord(prevHashHex, e)
}

// GenesisHash is the all-zero previous hash of the first record.
func GenesisHash() string {
	return auditlog.GenesisHash()
}

// ComputeHash chains one entry onto its predecessor.
func ComputeHash(previousHash, entryData string) string {
	return auditlog.ComputeHash(previousHash, entryData)
}

// NewTenantAuditAggregator scopes aggregation to one tenant.
func NewTenantAuditAggregator(repo Repository) (*TenantAuditAggregator, error) {
	return auditlog.NewTenantAuditAggregator(repo)
}
