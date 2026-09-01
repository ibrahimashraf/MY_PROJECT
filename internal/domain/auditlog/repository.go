package auditlog

import (
	"context"
	"time"
)

type Repository interface {
	Append(ctx context.Context, entry Entry) (Entry, error)
	Query(ctx context.Context, req QueryRequest) (QueryResponse, error)
	VerifyChain(ctx context.Context, tenantID, organizationID string, from, to *time.Time) (VerifyResult, error)
}
