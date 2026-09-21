package auditlogpg

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"integin/internal/domain/auditlog"
)

type fakeAuditRow struct {
	id             string
	tenantID       string
	organizationID string
	eventType      string
	entityType     string
	entityID       string
	actorID        string
	actorName      string
	action         string
	oldValue       []byte
	newValue       []byte
	metadata       []byte
	ipAddress      string
	userAgent      string
	previousHash   string
	entryHash      string
	createdAt      time.Time
	validTime      *time.Time
}

func (r fakeAuditRow) effectiveValidTime() time.Time {
	if r.validTime == nil {
		return r.createdAt
	}
	return *r.validTime
}

type fakeAuditStore struct {
	mu   sync.Mutex
	rows []fakeAuditRow
}

type fakeAuditDriver struct {
	mu     sync.Mutex
	stores map[string]*fakeAuditStore
}

func (d *fakeAuditDriver) Open(name string) (driver.Conn, error) {
	d.mu.Lock()
	st := d.stores[name]
	if st == nil {
		st = &fakeAuditStore{}
		d.stores[name] = st
	}
	d.mu.Unlock()
	return &fakeAuditConn{store: st}, nil
}

type fakeAuditConn struct {
	store *fakeAuditStore
}

func (c *fakeAuditConn) Prepare(query string) (driver.Stmt, error) {
	return &fakeAuditStmt{store: c.store, query: query}, nil
}

func (c *fakeAuditConn) Close() error { return nil }

func (c *fakeAuditConn) Begin() (driver.Tx, error) { return &fakeAuditTx{conn: c}, nil }

func (c *fakeAuditConn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	return &fakeAuditTx{conn: c}, nil
}

type fakeAuditTx struct{ conn *fakeAuditConn }

func (t *fakeAuditTx) Commit() error   { return nil }
func (t *fakeAuditTx) Rollback() error { return nil }

type fakeAuditStmt struct {
	store *fakeAuditStore
	query string
}

func (s *fakeAuditStmt) Close() error { return nil }
func (s *fakeAuditStmt) NumInput() int {
	return -1
}

func (s *fakeAuditStmt) Exec(args []driver.Value) (driver.Result, error) {
	if strings.Contains(s.query, "INSERT INTO audit_log") {
		s.store.insert(args)
		return driver.RowsAffected(1), nil
	}
	if strings.HasPrefix(s.query, "SELECT set_config") {
		return driver.RowsAffected(1), nil
	}
	return nil, fmt.Errorf("unexpected exec query: %s", s.query)
}

func (s *fakeAuditStmt) Query(args []driver.Value) (driver.Rows, error) {
	switch {
	case strings.HasPrefix(s.query, "SELECT set_config"), strings.Contains(s.query, "INSERT INTO audit_log"):
		return nil, fmt.Errorf("unexpected query: %s", s.query)
	case strings.Contains(s.query, "SELECT entry_hash FROM audit_log"):
		return s.store.lastHashRows(args), nil
	case strings.Contains(s.query, "SELECT COUNT(*)"):
		return newFakeRows([]string{"count"}, [][]driver.Value{{int64(s.store.count(s.query, args))}}), nil
	default:
		return s.store.selectRows(s.query, args), nil
	}
}

func (s *fakeAuditStore) insert(args []driver.Value) {
	row := fakeAuditRow{
		id:             asString(args[0]),
		tenantID:       asString(args[1]),
		organizationID: asString(args[2]),
		eventType:      asString(args[3]),
		entityType:     asString(args[4]),
		entityID:       asString(args[5]),
		actorID:        asString(args[6]),
		actorName:      asString(args[7]),
		action:         asString(args[8]),
		oldValue:       asBytes(args[9]),
		newValue:       asBytes(args[10]),
		metadata:       asBytes(args[11]),
		ipAddress:      asString(args[12]),
		userAgent:      asString(args[13]),
		previousHash:   asString(args[14]),
		entryHash:      asString(args[15]),
		createdAt:      asTime(args[16]),
	}
	if args[17] != nil {
		t := asTime(args[17])
		row.validTime = &t
	}
	s.mu.Lock()
	s.rows = append(s.rows, row)
	s.mu.Unlock()
}

func (s *fakeAuditStore) lastHashRows(args []driver.Value) driver.Rows {
	tenant := asString(args[0])
	org := asString(args[1])
	s.mu.Lock()
	defer s.mu.Unlock()
	var best *fakeAuditRow
	for i := range s.rows {
		r := s.rows[i]
		if (r.tenantID != tenant) || (r.organizationID != org) {
			continue
		}
		if best == nil || r.createdAt.After(best.createdAt) {
			best = &s.rows[i]
		}
	}
	if best == nil {
		return newFakeRows([]string{"entry_hash"}, nil)
	}
	return newFakeRows([]string{"entry_hash"}, [][]driver.Value{{best.entryHash}})
}

func (s *fakeAuditStore) count(query string, args []driver.Value) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, r := range s.rows {
		if matchWhere(query, args, r) {
			n++
		}
	}
	return n
}

var auditPlaceholderRe = regexp.MustCompile(`\$(\d+)`)

func matchWhere(query string, args []driver.Value, r fakeAuditRow) bool {
	start := strings.Index(query, "WHERE ")
	if start < 0 {
		return true
	}
	start += len("WHERE ")
	end := len(query)
	if idx := strings.Index(query[start:], " ORDER BY "); idx >= 0 {
		end = start + idx
	}
	where := strings.TrimSpace(query[start:end])
	if where == "" {
		return true
	}
	for _, cond := range strings.Split(where, " AND ") {
		if !matchCondition(cond, args, r) {
			return false
		}
	}
	return true
}

func matchCondition(cond string, args []driver.Value, r fakeAuditRow) bool {
	m := auditPlaceholderRe.FindStringSubmatch(cond)
	if m == nil {
		return false
	}
	var n int
	if _, err := fmt.Sscanf(m[1], "%d", &n); err != nil || n < 1 || n > len(args) {
		return false
	}
	val := args[n-1]
	switch {
	case strings.Contains(cond, "tenant_id ="):
		return asString(val) == r.tenantID
	case strings.Contains(cond, "organization_id ="):
		return asString(val) == r.organizationID
	case strings.Contains(cond, "entity_type ="):
		return asString(val) == r.entityType
	case strings.Contains(cond, "entity_id ="):
		return asString(val) == r.entityID
	case strings.Contains(cond, "actor_id ="):
		return asString(val) == r.actorID
	case strings.Contains(cond, "event_type ="):
		return asString(val) == r.eventType
	case strings.Contains(cond, "COALESCE(valid_time, created_at) >="):
		return !r.effectiveValidTime().Before(asTime(val))
	case strings.Contains(cond, "COALESCE(valid_time, created_at) <="):
		return !asTime(val).Before(r.effectiveValidTime())
	case strings.Contains(cond, "created_at >="):
		return !r.createdAt.Before(asTime(val))
	case strings.Contains(cond, "created_at <="):
		return !asTime(val).Before(r.createdAt)
	default:
		return false
	}
}

var auditSelectColumns = []string{
	"id", "tenant_id", "organization_id", "event_type", "entity_type", "entity_id",
	"actor_id", "actor_name", "action", "old_value", "new_value", "metadata",
	"ip_address", "user_agent", "previous_hash", "entry_hash", "created_at", "valid_time",
}

func (s *fakeAuditStore) selectRows(query string, args []driver.Value) driver.Rows {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []fakeAuditRow
	for _, r := range s.rows {
		if matchWhere(query, args, r) {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].createdAt.After(out[j].createdAt) })
	var data [][]driver.Value
	for _, r := range out {
		var vt driver.Value
		if r.validTime != nil {
			vt = *r.validTime
		}
		data = append(data, []driver.Value{
			r.id, r.tenantID, r.organizationID, r.eventType, r.entityType, r.entityID,
			r.actorID, r.actorName, r.action, r.oldValue, r.newValue, r.metadata,
			r.ipAddress, r.userAgent, r.previousHash, r.entryHash, r.createdAt, vt,
		})
	}
	return newFakeRows(auditSelectColumns, data)
}

type fakeRows struct {
	cols []string
	data [][]driver.Value
	idx  int
}

func newFakeRows(cols []string, data [][]driver.Value) *fakeRows {
	return &fakeRows{cols: cols, data: data}
}

func (r *fakeRows) Columns() []string { return r.cols }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.idx])
	r.idx++
	return nil
}

func asTime(v driver.Value) time.Time {
	if t, ok := v.(time.Time); ok {
		return t
	}
	return time.Time{}
}

func asString(v driver.Value) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asBytes(v driver.Value) []byte {
	if b, ok := v.([]byte); ok {
		return b
	}
	return nil
}

var (
	audFakeDriverOnce sync.Once
	audFakeDriver     = &fakeAuditDriver{stores: map[string]*fakeAuditStore{}}
)

func openAuditFakeDB(dsn string) (*sql.DB, error) {
	audFakeDriverOnce.Do(func() {
		sql.Register("integin-fake-auditlog", audFakeDriver)
	})
	return sql.Open("integin-fake-auditlog", dsn)
}

func TestAppendQueryValidTimeHermetic(t *testing.T) {
	db, err := openAuditFakeDB("t-roundtrip")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	valid, err := repo.Append(ctx, auditlog.Entry{
		ID:             "e-valid",
		TenantID:       "tenant-1",
		OrganizationID: "org-1",
		EventType:      "inspection",
		EntityType:     auditlog.EntityInspection,
		EntityID:       "asset-1",
		ActorID:        "user-1",
		ActorName:      "User One",
		Action:         auditlog.ActionCreate,
		OldValue:       map[string]interface{}{"state": "pre"},
		NewValue:       map[string]interface{}{"state": "post"},
		Metadata:       map[string]interface{}{"source": "hermetic"},
		IPAddress:      "10.0.0.1",
		UserAgent:      "fake-agent",
		CreatedAt:      t0,
		ValidTime:      t0.Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("Append(valid): %v", err)
	}
	if !valid.ValidTime.Equal(t0.Add(-time.Hour)) {
		t.Fatalf("Append returned ValidTime = %v, want %v", valid.ValidTime, t0.Add(-time.Hour))
	}

	legacy, err := repo.Append(ctx, auditlog.Entry{
		ID:             "e-legacy",
		TenantID:       "tenant-1",
		OrganizationID: "org-1",
		EventType:      "work_order",
		EntityType:     auditlog.EntityWorkOrder,
		EntityID:       "wo-1",
		ActorID:        "user-1",
		Action:         auditlog.ActionUpdate,
		CreatedAt:      t0.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("Append(legacy): %v", err)
	}
	if !legacy.ValidTime.IsZero() {
		t.Fatalf("Append(legacy) ValidTime = %v, want zero (NULL)", legacy.ValidTime)
	}

	resp, err := repo.Query(ctx, auditlog.QueryRequest{
		TenantID:       "tenant-1",
		OrganizationID: "org-1",
		Limit:          50,
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("Total = %d, want 2", resp.Total)
	}
	got := map[string]auditlog.Entry{}
	for _, e := range resp.Entries {
		got[e.ID] = e
	}
	round := got["e-valid"]
	if !round.ValidTime.Equal(t0.Add(-time.Hour)) {
		t.Fatalf("Query round-trip ValidTime = %v, want %v", round.ValidTime, t0.Add(-time.Hour))
	}
	legacyBack := got["e-legacy"]
	if !legacyBack.ValidTime.IsZero() {
		t.Fatalf("Query NULL mapping: ValidTime = %v, want zero", legacyBack.ValidTime)
	}
	if !legacyBack.ValidAt().Equal(legacyBack.CreatedAt) {
		t.Fatalf("legacy ValidAt() = %v, want CreatedAt %v", legacyBack.ValidAt(), legacyBack.CreatedAt)
	}
}

func TestQueryValidTimeFiltersHermetic(t *testing.T) {
	db, err := openAuditFakeDB("t-filters")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	t0 := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)
	created := t0.Add(10 * time.Minute)

	entries := []auditlog.Entry{
		{ID: "f-early", TenantID: "tenant-1", OrganizationID: "org-1", EventType: "inspection", EntityType: auditlog.EntityInspection, EntityID: "a", ActorID: "u", Action: auditlog.ActionCreate, CreatedAt: created, ValidTime: t0.Add(time.Minute)},
		{ID: "f-legacy", TenantID: "tenant-1", OrganizationID: "org-1", EventType: "inspection", EntityType: auditlog.EntityInspection, EntityID: "b", ActorID: "u", Action: auditlog.ActionCreate, CreatedAt: created},
		{ID: "f-late", TenantID: "tenant-1", OrganizationID: "org-1", EventType: "inspection", EntityType: auditlog.EntityInspection, EntityID: "c", ActorID: "u", Action: auditlog.ActionCreate, CreatedAt: created, ValidTime: t0.Add(30 * time.Minute)},
	}
	for _, e := range entries {
		if _, err := repo.Append(ctx, e); err != nil {
			t.Fatalf("Append(%s): %v", e.ID, err)
		}
	}

	from := t0.Add(5 * time.Minute)
	to := t0.Add(20 * time.Minute)
	resp, err := repo.Query(ctx, auditlog.QueryRequest{
		TenantID:       "tenant-1",
		OrganizationID: "org-1",
		ValidFrom:      &from,
		ValidTo:        &to,
		Limit:          50,
	})
	if err != nil {
		t.Fatalf("Query(ValidFrom+ValidTo): %v", err)
	}
	if resp.Total != 1 || len(resp.Entries) != 1 || resp.Entries[0].ID != "f-legacy" {
		t.Fatalf("range filter: total=%d entries=%d (want 1 f-legacy), got %+v", resp.Total, len(resp.Entries), resp.Entries)
	}

	resp, err = repo.Query(ctx, auditlog.QueryRequest{
		TenantID:       "tenant-1",
		OrganizationID: "org-1",
		ValidFrom:      &from,
		Limit:          50,
	})
	if err != nil {
		t.Fatalf("Query(ValidFrom): %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("ValidFrom-only Total = %d, want 2", resp.Total)
	}

	resp, err = repo.Query(ctx, auditlog.QueryRequest{
		TenantID:       "tenant-1",
		OrganizationID: "org-1",
		ValidTo:        &to,
		Limit:          50,
	})
	if err != nil {
		t.Fatalf("Query(ValidTo): %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("ValidTo-only Total = %d, want 2", resp.Total)
	}
}
