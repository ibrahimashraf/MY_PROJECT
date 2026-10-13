package calibration

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	sharedcalibration "integin/internal/shared/calibration"
	"integin/internal/shared/types"
)

var errFake = errors.New("fake driver failure")

type fakeResult struct{}

func (fakeResult) LastInsertId() (int64, error) { return 0, errors.New("not supported") }
func (fakeResult) RowsAffected() (int64, error) { return 1, nil }

type fakeRows struct {
	rows [][]driver.Value
}

func (r *fakeRows) Columns() []string { return []string{"exists"} }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if len(r.rows) == 0 {
		return io.EOF
	}
	dest[0] = r.rows[0][0]
	r.rows = r.rows[1:]
	return nil
}

type fakeStmt struct{}

func (s *fakeStmt) Close() error  { return nil }
func (s *fakeStmt) NumInput() int { return 0 }
func (s *fakeStmt) Exec(args []driver.Value) (driver.Result, error) {
	return nil, errors.New("fake driver only executes via context methods")
}
func (s *fakeStmt) Query(args []driver.Value) (driver.Rows, error) {
	return nil, errors.New("fake driver only queries via context methods")
}

type fakeTx struct{ c *fakeConn }

func (t *fakeTx) Commit() error   { t.c.commits++; return nil }
func (t *fakeTx) Rollback() error { return nil }

type fakeConn struct {
	execs     []string
	execArgs  []driver.Value
	query     string
	queryArgs []driver.Value
	rows      []driver.Value
	execErr   error
	queryErr  error
	commits   int
}

func (c *fakeConn) Prepare(query string) (driver.Stmt, error) { return &fakeStmt{}, nil }
func (c *fakeConn) Close() error                              { return nil }
func (c *fakeConn) Begin() (driver.Tx, error)                 { return &fakeTx{c: c}, nil }

func (c *fakeConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.execs = append(c.execs, query)
	for _, a := range args {
		c.execArgs = append(c.execArgs, a.Value)
	}
	if c.execErr != nil {
		return nil, c.execErr
	}
	return fakeResult{}, nil
}

func (c *fakeConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.query = query
	c.queryArgs = nil
	for _, a := range args {
		c.queryArgs = append(c.queryArgs, a.Value)
	}
	if c.queryErr != nil {
		return nil, c.queryErr
	}
	return &fakeRows{rows: [][]driver.Value{c.rows}}, nil
}

type fakeConnector struct{ c *fakeConn }

func (f *fakeConnector) Connect(context.Context) (driver.Conn, error) { return f.c, nil }
func (f *fakeConnector) Driver() driver.Driver                        { return stubDriver{} }

type stubDriver struct{}

func (stubDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("fake driver opens via connector only")
}

func newFakeStore(c *fakeConn) *Store {
	db := sql.OpenDB(&fakeConnector{c: c})
	store, err := NewStore(db)
	if err != nil {
		panic(err)
	}
	return store
}

func validRecord(id, tenant, org, equipment string) sharedcalibration.Record {
	return sharedcalibration.Record{
		ID:                   id,
		TenantID:             tenant,
		EquipmentID:          equipment,
		StandardReference:    "ISO-17020",
		SerialNumber:         "SN-001",
		LabCertificateRef:    "LAB-2026-001",
		UncertaintyTolerance: "±0.5%",
		CalibrationDate:      time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		NextDueDate:          time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC),
		TechnicianID:         "tech-1",
		Result:               "PASS",
		Status:               types.CalibrationActive,
	}
}

func TestNewStoreNilGuard(t *testing.T) {
	if _, err := NewStore(nil); err != ErrNilDB {
		t.Fatalf("expected ErrNilDB, got %v", err)
	}
}

func TestUpsertValidatesBeforeTouchingDB(t *testing.T) {
	c := &fakeConn{execErr: errFake}
	store := newFakeStore(c)
	record := validRecord("cal-1", "tenant-1", "org-1", "meter-1")
	record.TechnicianID = ""
	if err := store.Upsert(context.Background(), "tenant-1", "org-1", record); err == nil {
		t.Fatal("expected validation error")
	}
	if len(c.execs) != 0 {
		t.Fatalf("database touched before validation: %v", c.execs)
	}
}

func TestUpsertRunsScopedInsertAndCommits(t *testing.T) {
	c := &fakeConn{}
	store := newFakeStore(c)
	record := validRecord("cal-1", "tenant-1", "org-1", "meter-1")
	if err := store.Upsert(context.Background(), "tenant-1", "org-1", record); err != nil {
		t.Fatal(err)
	}
	if len(c.execs) != 2 {
		t.Fatalf("expected set_config then upsert, got %d execs", len(c.execs))
	}
	if !strings.Contains(c.execs[0], "set_config") {
		t.Fatalf("first statement must configure tenant scope: %s", c.execs[0])
	}
	if c.execs[1] != sqlUpsert {
		t.Fatalf("unexpected upsert SQL:\n%s", c.execs[1])
	}
	if c.execArgs[0] != "tenant-1" || c.execArgs[1] != "org-1" {
		t.Fatalf("session scope args wrong: %#v", c.execArgs[:2])
	}
	if c.execArgs[3] != "tenant-1" || c.execArgs[4] != "org-1" || c.execArgs[5] != "meter-1" {
		t.Fatalf("row tenant/org/equipment args wrong: %#v", c.execArgs[2:6])
	}
	if c.execArgs[6] != "SN-001" || c.execArgs[7] != "LAB-2026-001" || c.execArgs[8] != "±0.5%" {
		t.Fatalf("tool identity args wrong: %#v", c.execArgs[6:9])
	}
	if !c.execArgs[10].(time.Time).Equal(record.NextDueDate.UTC()) {
		t.Fatalf("next_due_date not UTC-normalized: %#v", c.execArgs[10])
	}
	if c.execArgs[13] != "ACTIVE" {
		t.Fatalf("status arg wrong: %#v", c.execArgs[13])
	}
	if c.commits != 1 {
		t.Fatalf("expected 1 commit, got %d", c.commits)
	}
}

func TestUpsertPropagatesExecError(t *testing.T) {
	c := &fakeConn{execErr: errFake}
	store := newFakeStore(c)
	if err := store.Upsert(context.Background(), "tenant-1", "org-1", validRecord("cal-1", "tenant-1", "org-1", "meter-1")); err != errFake {
		t.Fatalf("expected fake error to propagate, got %v", err)
	}
}

func TestSubmissionBlockedValidatesEquipmentBeforeDB(t *testing.T) {
	c := &fakeConn{queryErr: errFake}
	store := newFakeStore(c)
	if blocked, err := store.SubmissionBlocked(context.Background(), "tenant-1", "org-1", "", time.Now()); err == nil || blocked {
		t.Fatalf("expected validation error for empty equipment id, got blocked=%v err=%v", blocked, err)
	}
	if c.query != "" {
		t.Fatalf("database touched before validation: %s", c.query)
	}
}

func TestSubmissionBlockedFalseWhenUnexpiredActiveRowExists(t *testing.T) {
	c := &fakeConn{rows: []driver.Value{true}}
	store := newFakeStore(c)
	at := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	blocked, err := store.SubmissionBlocked(context.Background(), "tenant-1", "org-1", "meter-1", at)
	if err != nil {
		t.Fatal(err)
	}
	if blocked {
		t.Fatal("unexpired active calibration should allow submission")
	}
	if c.query != sqlHasUnexpiredActive {
		t.Fatalf("unexpected blocked SQL:\n%s", c.query)
	}
	if c.queryArgs[0] != "meter-1" || !c.queryArgs[1].(time.Time).Equal(at.UTC()) {
		t.Fatalf("blocked query args wrong: %#v", c.queryArgs)
	}
}

func TestSubmissionBlockedTrueWhenExpiredOrMissing(t *testing.T) {
	c := &fakeConn{rows: []driver.Value{false}}
	store := newFakeStore(c)
	at := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	blocked, err := store.SubmissionBlocked(context.Background(), "tenant-1", "org-1", "meter-1", at)
	if err != nil {
		t.Fatal(err)
	}
	if !blocked {
		t.Fatal("expired-or-missing calibration should hard-block submission")
	}
}

func TestSubmissionBlockedPropagatesQueryError(t *testing.T) {
	c := &fakeConn{queryErr: errFake}
	store := newFakeStore(c)
	if blocked, err := store.SubmissionBlocked(context.Background(), "tenant-1", "org-1", "meter-1", time.Now()); err != errFake || blocked {
		t.Fatalf("expected fake error to propagate, got blocked=%v err=%v", blocked, err)
	}
}

func TestStoreSQLParameterizedPlaceholders(t *testing.T) {
	for i := 1; i <= 12; i++ {
		if !strings.Contains(sqlUpsert, fmt.Sprintf("$%d", i)) {
			t.Fatalf("upsert SQL missing placeholder $%d", i)
		}
	}
	required := []string{"ON CONFLICT (tenant_id, organization_id, id)",
		"next_due_date > $2", "status = 'ACTIVE'", "equipment_id = $1"}
	for _, fragment := range required {
		if !strings.Contains(sqlUpsert, fragment) && !strings.Contains(sqlHasUnexpiredActive, fragment) {
			t.Fatalf("SQL missing %q", fragment)
		}
	}
}

func TestCalibrationSubmissionBlockedWhenMissing(t *testing.T) {
	service := New(nil)
	if err := service.SubmissionAllowed("tenant-1", "unknown-meter", time.Now()); err == nil {
		t.Fatal("missing calibration should block submission")
	}
}
