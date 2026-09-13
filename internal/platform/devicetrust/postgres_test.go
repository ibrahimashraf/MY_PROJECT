package devicetrust

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/device_trust"
)

var errFake = errors.New("fake driver failure")

type fakeResult struct{ rowsAffected int64 }

func (r fakeResult) LastInsertId() (int64, error) { return 0, errors.New("not supported") }
func (r fakeResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

type fakeRows struct {
	cols []string
	rows [][]driver.Value
}

func (r *fakeRows) Columns() []string { return r.cols }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if len(r.rows) == 0 {
		return io.EOF
	}
	copy(dest, r.rows[0])
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
	execs        []string
	execArgs     []driver.Value
	queries      []string
	queryArgs    [][]driver.Value
	rows         *fakeRows
	execErr      error
	queryErr     error
	rowsAffected int64
	commits      int
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
	return fakeResult{rowsAffected: c.rowsAffected}, nil
}

func (c *fakeConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.queries = append(c.queries, query)
	values := make([]driver.Value, 0, len(args))
	for _, a := range args {
		values = append(values, a.Value)
	}
	c.queryArgs = append(c.queryArgs, values)
	if c.queryErr != nil {
		return nil, c.queryErr
	}
	if c.rows == nil {
		return &fakeRows{}, nil
	}
	return c.rows, nil
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

func pendingRequest(id string) *device_trust.EnrollmentRequest {
	return &device_trust.EnrollmentRequest{
		RequestID:      id,
		TenantID:       "tenant-1",
		OrganizationID: "org-1",
		UserID:         "user-oidc-1",
		DeviceID:       "device-1",
		PublicKey:      strings.Repeat("ab", 32),
		Nonce:          "nonce-1",
		Signature:      strings.Repeat("cd", 64),
		Attestation:    device_trust.EnrollmentAttestation{KeyOrigin: "STRONGBOX", BiometricBound: true, OSVersion: "Android 14"},
		RequestedAt:    time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC),
		Status:         device_trust.EnrollmentPending,
	}
}

func TestNewStoreNilGuard(t *testing.T) {
	if _, err := NewStore(nil); err != ErrNilDB {
		t.Fatalf("expected ErrNilDB, got %v", err)
	}
}

func TestSaveRequestValidatesBeforeTouchingDB(t *testing.T) {
	c := &fakeConn{execErr: errFake}
	store := newFakeStore(c)
	req := pendingRequest("req-1")
	req.RequestID = ""
	if err := store.SaveRequest(context.Background(), req); err == nil {
		t.Fatal("expected validation error for empty request id")
	}
	if len(c.execs) != 0 {
		t.Fatalf("database touched before validation: %v", c.execs)
	}
}

func TestSaveRequestRunsScopedInsertAndCommits(t *testing.T) {
	c := &fakeConn{}
	store := newFakeStore(c)
	req := pendingRequest("req-1")
	if err := store.SaveRequest(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if len(c.execs) != 2 {
		t.Fatalf("expected set_config then insert, got %d execs", len(c.execs))
	}
	if !strings.Contains(c.execs[0], "set_config") {
		t.Fatalf("first statement must configure tenant scope: %s", c.execs[0])
	}
	if c.execs[1] != sqlSaveRequest {
		t.Fatalf("unexpected insert SQL:\n%s", c.execs[1])
	}
	// execArgs[0,1] are the transaction-scoped session GUCs (tenant, org).
	if c.execArgs[0] != "tenant-1" || c.execArgs[1] != "org-1" {
		t.Fatalf("session scope args wrong: %#v", c.execArgs[:2])
	}
	rowArgs := c.execArgs[2:]
	if rowArgs[0] != "req-1" || rowArgs[1] != "tenant-1" || rowArgs[2] != "org-1" {
		t.Fatalf("scope + request identity args wrong: %#v", rowArgs[:3])
	}
	if rowArgs[3] != "user-oidc-1" || rowArgs[4] != "device-1" {
		t.Fatalf("user/device args wrong: %#v", rowArgs[3:5])
	}
	attestation, ok := rowArgs[8].([]byte)
	if !ok {
		t.Fatalf("attestation arg must be JSON bytes, got %T", rowArgs[8])
	}
	var decoded device_trust.EnrollmentAttestation
	if err := json.Unmarshal(attestation, &decoded); err != nil {
		t.Fatalf("attestation JSON invalid: %v", err)
	}
	if decoded.KeyOrigin != "STRONGBOX" || decoded.OSVersion != "Android 14" {
		t.Fatalf("attestation round-trip failed: %+v", decoded)
	}
	if rowArgs[9] != "PENDING" {
		t.Fatalf("status arg wrong: %#v", rowArgs[9])
	}
	if !rowArgs[10].(time.Time).Equal(req.RequestedAt.UTC()) {
		t.Fatalf("requested_at not UTC-normalized: %#v", rowArgs[10])
	}
	if c.commits != 1 {
		t.Fatalf("expected 1 commit, got %d", c.commits)
	}
}

func TestSaveRequestPropagatesExecError(t *testing.T) {
	c := &fakeConn{execErr: errFake}
	store := newFakeStore(c)
	if err := store.SaveRequest(context.Background(), pendingRequest("req-1")); err != errFake {
		t.Fatalf("expected fake error to propagate, got %v", err)
	}
}

func TestUpdateStatusValidatesActorBeforeDB(t *testing.T) {
	c := &fakeConn{}
	store := newFakeStore(c)
	if err := store.UpdateStatus(context.Background(), "tenant-1", "org-1", "req-1", device_trust.EnrollmentApproved, "", ""); err == nil {
		t.Fatal("approval without an actor must fail")
	}
	if err := store.UpdateStatus(context.Background(), "tenant-1", "org-1", "req-1", device_trust.EnrollmentRejected, "admin-1", ""); err == nil {
		t.Fatal("rejection without a reason must fail")
	}
	if err := store.UpdateStatus(context.Background(), "tenant-1", "org-1", "req-1", device_trust.EnrollmentPending, "admin-1", "why"); err == nil {
		t.Fatal("updating to PENDING must be rejected")
	}
	if len(c.execs) != 0 {
		t.Fatalf("database touched before validation: %v", c.execs)
	}
}

func TestUpdateStatusApprovedPersistsActorAndCommits(t *testing.T) {
	c := &fakeConn{rowsAffected: 1}
	store := newFakeStore(c)
	if err := store.UpdateStatus(context.Background(), "tenant-1", "org-1", "req-1", device_trust.EnrollmentApproved, "admin-1", ""); err != nil {
		t.Fatal(err)
	}
	if len(c.execs) != 2 {
		t.Fatalf("expected set_config then update, got %d execs", len(c.execs))
	}
	if c.execs[1] != sqlUpdateStatus {
		t.Fatalf("unexpected update SQL:\n%s", c.execs[1])
	}
	if c.execArgs[2] != "APPROVED" || c.execArgs[3] != "admin-1" {
		t.Fatalf("status/actor args wrong: %#v", c.execArgs[2:5])
	}
	if c.commits != 1 {
		t.Fatalf("expected 1 commit, got %d", c.commits)
	}
}

func TestUpdateStatusRejectedPersistsActorAndReason(t *testing.T) {
	c := &fakeConn{rowsAffected: 1}
	store := newFakeStore(c)
	if err := store.UpdateStatus(context.Background(), "tenant-1", "org-1", "req-1", device_trust.EnrollmentRejected, "admin-1", "weak key material"); err != nil {
		t.Fatal(err)
	}
	if c.execArgs[2] != "REJECTED" || c.execArgs[4] != "admin-1" || c.execArgs[5] != "weak key material" {
		t.Fatalf("rejection args wrong: %#v", c.execArgs[2:6])
	}
	if c.commits != 1 {
		t.Fatalf("expected 1 commit, got %d", c.commits)
	}
}

func TestUpdateStatusReturnsNotFoundOnZeroRows(t *testing.T) {
	c := &fakeConn{rowsAffected: 0}
	store := newFakeStore(c)
	if err := store.UpdateStatus(context.Background(), "tenant-1", "org-1", "unknown", device_trust.EnrollmentApproved, "admin-1", ""); err != ErrRequestNotFound {
		t.Fatalf("expected ErrRequestNotFound, got %v", err)
	}
}

func getRequestColumns() []string {
	return []string{"request_id", "tenant_id", "organization_id", "user_id", "device_id", "public_key", "nonce", "signature", "attestation", "status", "requested_at", "approved_by", "rejected_by", "rejection_reason"}
}

func TestGetRequestPopulatesDomainAndAttestation(t *testing.T) {
	req := pendingRequest("req-1")
	attestation := []byte(`{"key_origin":"STRONGBOX","biometric_bound":true,"os_version":"Android 14"}`)
	row := []driver.Value{
		req.RequestID, req.TenantID, req.OrganizationID, req.UserID, req.DeviceID,
		req.PublicKey, req.Nonce, req.Signature, attestation, "APPROVED", req.RequestedAt,
		"admin-1", "", "",
	}
	c := &fakeConn{rows: &fakeRows{cols: getRequestColumns(), rows: [][]driver.Value{row}}}
	store := newFakeStore(c)
	got, err := store.GetRequest(context.Background(), "tenant-1", "org-1", "req-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.RequestID != "req-1" || got.Status != device_trust.EnrollmentApproved || got.ApprovedBy != "admin-1" {
		t.Fatalf("domain fields wrong: %+v", got)
	}
	if got.Attestation.KeyOrigin != "STRONGBOX" || got.Attestation.OSVersion != "Android 14" {
		t.Fatalf("attestation not decoded: %+v", got.Attestation)
	}
	if c.queries[0] != sqlGetRequest {
		t.Fatalf("unexpected get SQL:\n%s", c.queries[0])
	}
	if len(c.queryArgs) != 1 || c.queryArgs[0][0] != "tenant-1" || c.queryArgs[0][1] != "org-1" || c.queryArgs[0][2] != "req-1" {
		t.Fatalf("get scope args wrong: %#v", c.queryArgs)
	}
}

func TestGetRequestReturnsNotFoundWhenMissing(t *testing.T) {
	c := &fakeConn{queryErr: sql.ErrNoRows}
	store := newFakeStore(c)
	if _, err := store.GetRequest(context.Background(), "tenant-1", "org-1", "missing"); err != ErrRequestNotFound {
		t.Fatalf("expected ErrRequestNotFound, got %v", err)
	}
}

func TestListPendingReturnsOnlyPendingRows(t *testing.T) {
	requestedAt := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
	cols := []string{"request_id", "tenant_id", "organization_id", "user_id", "device_id", "public_key", "nonce", "signature", "attestation", "status", "requested_at"}
	rows := &fakeRows{cols: cols, rows: [][]driver.Value{
		{"req-1", "tenant-1", "org-1", "user-1", "device-1", strings.Repeat("ab", 32), "n1", strings.Repeat("cd", 64), "", "PENDING", requestedAt},
		{"req-2", "tenant-1", "org-1", "user-2", "device-2", strings.Repeat("ef", 32), "n2", strings.Repeat("01", 64), "", "PENDING", requestedAt},
	}}
	c := &fakeConn{rows: rows}
	store := newFakeStore(c)
	got, err := store.ListPending(context.Background(), "tenant-1", "org-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].RequestID != "req-1" || got[1].RequestID != "req-2" {
		t.Fatalf("pending list wrong: %+v", got)
	}
	if got[0].Status != device_trust.EnrollmentPending {
		t.Fatalf("list must only expose PENDING, got %s", got[0].Status)
	}
}

func TestStoreSQLParameterizedPlaceholdersAndContracts(t *testing.T) {
	for i := 1; i <= 11; i++ {
		if !strings.Contains(sqlSaveRequest, fmt.Sprintf("$%d", i)) {
			t.Fatalf("insert SQL missing placeholder $%d", i)
		}
	}
	for i := 1; i <= 7; i++ {
		if !strings.Contains(sqlUpdateStatus, fmt.Sprintf("$%d", i)) {
			t.Fatalf("update SQL missing placeholder $%d", i)
		}
	}
	required := []string{"WHERE tenant_id = $1 AND organization_id = $2 AND request_id = $3",
		"status = 'PENDING'", "device_enrollment_requests"}
	for _, fragment := range required {
		if !strings.Contains(sqlGetRequest, fragment) && !strings.Contains(sqlListPending, fragment) && !strings.Contains(sqlSaveRequest, fragment) {
			t.Fatalf("SQL missing %q", fragment)
		}
	}
}
