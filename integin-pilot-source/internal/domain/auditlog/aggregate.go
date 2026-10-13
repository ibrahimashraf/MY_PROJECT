package auditlog

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNilRepo            = errors.New("auditlog: aggregator requires a repository")
	ErrNoTenantScopes     = errors.New("auditlog: at least one tenant scope is required")
	ErrInvalidTenantScope = errors.New("auditlog: tenant scope requires a tenant id and organization id")
)

// defaultAggregateBatchSize caps the per-tenant read size so a runaway audit
// stream cannot trigger an unbounded scan from a single summary request.
const defaultAggregateBatchSize = 1000

// TenantScope identifies one tenant slice (tenant + organization pair) whose
// audit events will be summarized under its own RLS scope.
type TenantScope struct {
	TenantID       string
	OrganizationID string
}

// TenantAuditSummary folds one tenant slice's audit stream into counters. A
// per-tenant query failure is captured on Err without aborting sibling
// tenants; Total is the full stream count reported by the repository while
// the counters are bounded by the aggregator's per-tenant read cap.
type TenantAuditSummary struct {
	TenantID       string
	OrganizationID string
	Total          int
	ByAction       map[Action]int
	ByEntityType   map[EntityType]int
	ByEventType    map[string]int
	FirstRecord    *time.Time
	LastRecord     *time.Time
	Err            error
}

// TenantAuditAggregator summarizes audit event streams safely across bounded
// tenant slices, always scoping each query to a single tenant/organization.
type TenantAuditAggregator struct {
	repo      Repository
	batchSize int
}

// NewTenantAuditAggregator builds an aggregator over the given repository.
func NewTenantAuditAggregator(repo Repository) (*TenantAuditAggregator, error) {
	if repo == nil {
		return nil, ErrNilRepo
	}
	return &TenantAuditAggregator{repo: repo, batchSize: defaultAggregateBatchSize}, nil
}

// Summarize queries each tenant scope with its tenant/org scope and the given
// time bounds, folding the returned entries into per-tenant counters. All
// scopes are validated up front so an invalid slice fails before any query is
// issued; a per-tenant repository failure is recorded on that tenant's summary
// and the remaining slices are still processed.
func (a *TenantAuditAggregator) Summarize(ctx context.Context, scopes []TenantScope, from, to *time.Time) ([]TenantAuditSummary, error) {
	if a == nil || a.repo == nil {
		return nil, ErrNilRepo
	}
	if len(scopes) == 0 {
		return nil, ErrNoTenantScopes
	}
	for _, s := range scopes {
		if s.TenantID == "" || s.OrganizationID == "" {
			return nil, ErrInvalidTenantScope
		}
	}

	results := make([]TenantAuditSummary, 0, len(scopes))
	for _, s := range scopes {
		resp, err := a.repo.Query(ctx, QueryRequest{
			TenantID:       s.TenantID,
			OrganizationID: s.OrganizationID,
			From:           from,
			To:             to,
			Limit:          a.batchSize,
		})
		if err != nil {
			results = append(results, TenantAuditSummary{
				TenantID:       s.TenantID,
				OrganizationID: s.OrganizationID,
				Err:            err,
			})
			continue
		}

		summary := TenantAuditSummary{
			TenantID:       s.TenantID,
			OrganizationID: s.OrganizationID,
			Total:          resp.Total,
			ByAction:       make(map[Action]int),
			ByEntityType:   make(map[EntityType]int),
			ByEventType:    make(map[string]int),
		}
		for _, e := range resp.Entries {
			summary.ByAction[e.Action]++
			summary.ByEntityType[e.EntityType]++
			summary.ByEventType[e.EventType]++
			if summary.FirstRecord == nil || e.CreatedAt.Before(*summary.FirstRecord) {
				at := e.CreatedAt
				summary.FirstRecord = &at
			}
			if summary.LastRecord == nil || e.CreatedAt.After(*summary.LastRecord) {
				at := e.CreatedAt
				summary.LastRecord = &at
			}
		}
		results = append(results, summary)
	}
	return results, nil
}
