package auditlog

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubRepository struct {
	queryErr  error
	errTenant string
	responses map[string]QueryResponse
	captured  []QueryRequest
}

func (s *stubRepository) Append(ctx context.Context, entry Entry) (Entry, error) {
	return entry, nil
}

func (s *stubRepository) Query(ctx context.Context, req QueryRequest) (QueryResponse, error) {
	s.captured = append(s.captured, req)
	if s.queryErr != nil && req.TenantID == s.errTenant {
		return QueryResponse{}, s.queryErr
	}
	return s.responses[req.TenantID], nil
}

func (s *stubRepository) VerifyChain(ctx context.Context, tenantID, organizationID string, from, to *time.Time) (VerifyResult, error) {
	return VerifyResult{Valid: true}, nil
}

func testEntry(id, tenantID string, createdAt time.Time, action Action, etype EntityType) Entry {
	return Entry{
		TenantID:       tenantID,
		OrganizationID: "org-" + tenantID,
		EventType:      "entity." + string(action),
		EntityType:     etype,
		EntityID:       "e-" + id,
		ActorID:        "a-" + id,
		Action:         action,
		CreatedAt:      createdAt,
	}
}

func TestTenantAuditAggregatorConstruction(t *testing.T) {
	if _, err := NewTenantAuditAggregator(nil); !errors.Is(err, ErrNilRepo) {
		t.Fatalf("nil repository must fail with ErrNilRepo, got %v", err)
	}
	agg, err := NewTenantAuditAggregator(&stubRepository{})
	if err != nil || agg == nil {
		t.Fatalf("valid repository must construct cleanly, got %v", err)
	}
}

func TestTenantAuditAggregatorValidatesScopes(t *testing.T) {
	agg, _ := NewTenantAuditAggregator(&stubRepository{})
	if _, err := agg.Summarize(context.Background(), nil, nil, nil); !errors.Is(err, ErrNoTenantScopes) {
		t.Fatalf("empty scopes must fail with ErrNoTenantScopes, got %v", err)
	}
	if _, err := agg.Summarize(context.Background(), []TenantScope{{TenantID: "t1"}}, nil, nil); !errors.Is(err, ErrInvalidTenantScope) {
		t.Fatalf("scope without organization must fail with ErrInvalidTenantScope, got %v", err)
	}
	if _, err := agg.Summarize(context.Background(), []TenantScope{{OrganizationID: "o1"}}, nil, nil); !errors.Is(err, ErrInvalidTenantScope) {
		t.Fatalf("scope without tenant must fail with ErrInvalidTenantScope, got %v", err)
	}
}

func TestTenantAuditAggregatorSummarizesAcrossTenants(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	repo := &stubRepository{responses: map[string]QueryResponse{
		"t1": {
			Total: 4,
			Entries: []Entry{
				testEntry("1", "t1", now.Add(-3*time.Hour), ActionCreate, EntityInspection),
				testEntry("2", "t1", now.Add(-2*time.Hour), ActionCreate, EntityInspection),
				testEntry("3", "t1", now.Add(-1*time.Hour), ActionUpdate, EntityWorkOrder),
				testEntry("4", "t1", now, ActionApprove, EntityCertificate),
			},
		},
		"t2": {
			Total: 1,
			Entries: []Entry{
				testEntry("5", "t2", now.Add(-30*time.Minute), ActionDelete, EntityAsset),
			},
		},
	}}
	agg, err := NewTenantAuditAggregator(repo)
	if err != nil {
		t.Fatalf("failed to construct aggregator: %v", err)
	}

	from := now.Add(-24 * time.Hour)
	results, err := agg.Summarize(context.Background(), []TenantScope{
		{TenantID: "t1", OrganizationID: "org-t1"},
		{TenantID: "t2", OrganizationID: "org-t2"},
	}, &from, &now)
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 tenant summaries, got %d", len(results))
	}

	t1 := results[0]
	if t1.Err != nil {
		t.Fatalf("t1 must not carry an error: %v", t1.Err)
	}
	if t1.Total != 4 {
		t.Fatalf("t1 total must be 4, got %d", t1.Total)
	}
	if t1.ByAction[ActionCreate] != 2 || t1.ByAction[ActionUpdate] != 1 || t1.ByAction[ActionApprove] != 1 {
		t.Fatalf("t1 action counts wrong: %+v", t1.ByAction)
	}
	if t1.ByEntityType[EntityInspection] != 2 || t1.ByEntityType[EntityWorkOrder] != 1 || t1.ByEntityType[EntityCertificate] != 1 {
		t.Fatalf("t1 entity counts wrong: %+v", t1.ByEntityType)
	}
	if t1.ByEventType["entity.create"] != 2 {
		t.Fatalf("t1 event type counts wrong: %+v", t1.ByEventType)
	}
	if t1.FirstRecord == nil || !t1.FirstRecord.Equal(now.Add(-3*time.Hour)) {
		t.Fatalf("t1 first record wrong: %v", t1.FirstRecord)
	}
	if t1.LastRecord == nil || !t1.LastRecord.Equal(now) {
		t.Fatalf("t1 last record wrong: %v", t1.LastRecord)
	}

	t2 := results[1]
	if t2.Total != 1 || t2.ByAction[ActionDelete] != 1 || t2.ByEntityType[EntityAsset] != 1 {
		t.Fatalf("t2 summary wrong: %+v", t2)
	}
}

func TestTenantAuditAggregatorForwardsScopeAndBounds(t *testing.T) {
	now := time.Now()
	repo := &stubRepository{responses: map[string]QueryResponse{"t1": {Total: 0}}}
	agg, _ := NewTenantAuditAggregator(repo)
	from := now.Add(-24 * time.Hour)
	if _, err := agg.Summarize(context.Background(), []TenantScope{{TenantID: "t1", OrganizationID: "o1"}}, &from, &now); err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}
	if len(repo.captured) != 1 {
		t.Fatalf("expected exactly 1 query, got %d", len(repo.captured))
	}
	req := repo.captured[0]
	if req.TenantID != "t1" || req.OrganizationID != "o1" {
		t.Fatalf("tenant scope not forwarded: %+v", req)
	}
	if req.Limit != defaultAggregateBatchSize {
		t.Fatalf("per-tenant read cap not applied, got %d", req.Limit)
	}
	if req.From != &from || req.To != &now {
		t.Fatal("time bounds must be forwarded verbatim")
	}
}

func TestTenantAuditAggregatorSurfacesPerTenantFailure(t *testing.T) {
	repo := &stubRepository{
		queryErr:  errors.New("tenant t1 storage unavailable"),
		errTenant: "t1",
		responses: map[string]QueryResponse{
			"t1": {Total: 0},
			"t2": {Total: 1, Entries: []Entry{testEntry("5", "t2", time.Now(), ActionCreate, EntityAsset)}},
		},
	}
	agg, _ := NewTenantAuditAggregator(repo)
	results, err := agg.Summarize(context.Background(), []TenantScope{
		{TenantID: "t1", OrganizationID: "o1"},
		{TenantID: "t2", OrganizationID: "o2"},
	}, nil, nil)
	if err != nil {
		t.Fatalf("aggregate run must not fail on a per-tenant query error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(results))
	}
	if results[0].Err == nil || results[0].Total != 0 {
		t.Fatalf("t1 failure must be captured on its summary: %+v", results[0])
	}
	if results[1].Err != nil || results[1].Total != 1 {
		t.Fatalf("t2 must still be summarized despite t1 failure: %+v", results[1])
	}
}

func TestTenantAuditAggregatorNilReceiver(t *testing.T) {
	var agg *TenantAuditAggregator
	if _, err := agg.Summarize(context.Background(), []TenantScope{{TenantID: "t1", OrganizationID: "o1"}}, nil, nil); !errors.Is(err, ErrNilRepo) {
		t.Fatalf("nil aggregator must fail with ErrNilRepo, got %v", err)
	}
}
