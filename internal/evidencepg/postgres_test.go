package evidencepg

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/evidence"
	"integin/internal/domain/workorder"
)

var errFake = errors.New("fake driver failure")

type fakeResult struct {
	rowsAffected int64
}

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
	for i, val := range r.rows[0] {
		dest[i] = val
	}
	r.rows = r.rows[1:]
	return nil
}

type fakeStmt struct{}

func (s *fakeStmt) Close() error  { return nil }
func (s *fakeStmt) NumInput() int { return -1 }
func (s *fakeStmt) Exec(args []driver.Value) (driver.Result, error) {
	return nil, errors.New("fake driver only executes via context methods")
}
func (s *fakeStmt) Query(args []driver.Value) (driver.Rows, error) {
	return nil, errors.New("fake driver only queries via context methods")
}

type fakeTx struct{ c *fakeConn }

func (t *fakeTx) Commit() error   { t.c.commits++; return nil }
func (t *fakeTx) Rollback() error { t.c.rollbacks++; return nil }

type queryResult struct {
	cols []string
	rows [][]driver.Value
}

type fakeConn struct {
	execs        []string
	execArgs     []driver.Value
	queries      []string
	queryArgs    []driver.Value
	queryResults []queryResult
	rowsToReturn [][]driver.Value
	colsToReturn []string
	execErr      error
	queryErr     error
	rowsAffected int64
	commits      int
	rollbacks    int
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
	for _, a := range args {
		c.queryArgs = append(c.queryArgs, a.Value)
	}
	if c.queryErr != nil {
		return nil, c.queryErr
	}
	if len(c.queryResults) > 0 {
		res := c.queryResults[0]
		c.queryResults = c.queryResults[1:]
		rows := make([][]driver.Value, len(res.rows))
		copy(rows, res.rows)
		return &fakeRows{cols: res.cols, rows: rows}, nil
	}
	cols := c.colsToReturn
	if len(cols) == 0 {
		cols = []string{"val"}
	}
	rows := make([][]driver.Value, len(c.rowsToReturn))
	copy(rows, c.rowsToReturn)
	return &fakeRows{cols: cols, rows: rows}, nil
}

type fakeConnector struct{ c *fakeConn }

func (f *fakeConnector) Connect(context.Context) (driver.Conn, error) { return f.c, nil }
func (f *fakeConnector) Driver() driver.Driver                        { return stubDriver{} }

type stubDriver struct{}

func (stubDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("fake driver opens via connector only")
}

func newFakeRepo(c *fakeConn) *Repository {
	db := sql.OpenDB(&fakeConnector{c: c})
	repo, err := NewRepository(db)
	if err != nil {
		panic(err)
	}
	return repo
}

var evidenceColumns = []string{
	"id", "tenant_id", "organization_id", "inspection_id", "object_key",
	"content_type", "ciphertext_bytes", "plaintext_sha256", "ciphertext_sha256",
	"captured_at", "device_id", "authority_id", "authority_epoch",
	"transaction_id", "receipt_id", "signature_algorithm", "key_id",
	"encryption_algorithm", "encryption_key_reference", "classification",
	"retention_reference", "hold_state", "redaction_policy_reference",
	"registered_by", "registered_at",
}

func validMetadata(tenantID, orgID, actorID, id string) evidence.Metadata {
	hash1 := sha256.Sum256([]byte("plaintext-payload"))
	hash2 := sha256.Sum256([]byte("ciphertext-payload"))
	now := time.Now().UTC().Truncate(time.Second)
	return evidence.Metadata{
		ID:                  id,
		TenantID:            tenantID,
		OrganizationID:      orgID,
		InspectionID:        "insp-001",
		ObjectKey:           tenantID + "/" + orgID + "/evidence/" + id,
		ContentType:         "application/pdf",
		CiphertextBytes:     1024,
		PlaintextSHA256:     hex.EncodeToString(hash1[:]),
		CiphertextSHA256:    hex.EncodeToString(hash2[:]),
		CapturedAt:          now,
		DeviceID:            "dev-tablet-01",
		AuthorityID:         "auth-01",
		AuthorityEpoch:      1,
		TransactionID:       "tx-001",
		ReceiptID:           "rcpt-001",
		SignatureAlgorithm:  "ed25519",
		KeyID:               "key-01",
		EncryptionAlgorithm: "aes-256-gcm",
		EncryptionKeyRef:    "kms://vault/k1",
		Classification:      "CALIBRATION_CERTIFICATE",
		RetentionReference:  "ret-10yr",
		HoldState:           "NONE",
		RedactionPolicyRef:  "redact-pII",
		RegisteredBy:        actorID,
		RegisteredAt:        now,
	}
}

func metadataToRow(m evidence.Metadata) []driver.Value {
	return []driver.Value{
		m.ID, m.TenantID, m.OrganizationID, m.InspectionID, m.ObjectKey,
		m.ContentType, m.CiphertextBytes, m.PlaintextSHA256, m.CiphertextSHA256,
		m.CapturedAt, m.DeviceID, m.AuthorityID, int64(m.AuthorityEpoch),
		m.TransactionID, m.ReceiptID, m.SignatureAlgorithm, m.KeyID,
		m.EncryptionAlgorithm, m.EncryptionKeyRef, m.Classification,
		m.RetentionReference, m.HoldState, m.RedactionPolicyRef,
		m.RegisteredBy, m.RegisteredAt,
	}
}

func TestNewRepositoryNilDB(t *testing.T) {
	if _, err := NewRepository(nil); err != ErrNilDB {
		t.Fatalf("expected ErrNilDB, got %v", err)
	}
}

func TestRegisterValidatesAndScopesActor(t *testing.T) {
	meta := validMetadata("tenant-1", "org-1", "actor-1", "ev-1")
	c := &fakeConn{
		rowsAffected: 1,
		colsToReturn: evidenceColumns,
		rowsToReturn: [][]driver.Value{metadataToRow(meta)},
	}
	repo := newFakeRepo(c)

	actor := evidence.ActorContext{TenantID: "tenant-1", OrganizationID: "org-1", ActorID: "actor-1"}
	stored, inserted, err := repo.Register(context.Background(), actor, meta)
	if err != nil {
		t.Fatalf("unexpected error registering evidence: %v", err)
	}
	if !inserted {
		t.Fatal("expected inserted == true")
	}
	if stored.ID != meta.ID || stored.ObjectKey != meta.ObjectKey {
		t.Fatalf("stored metadata mismatch: %+v", stored)
	}

	// Verify tenant and org RLS scope were set via pgtx.BeginScope
	foundScopeSet := false
	for _, exec := range c.execs {
		if strings.Contains(exec, "set_config('integin.tenant_id'") {
			foundScopeSet = true
			break
		}
	}
	if !foundScopeSet {
		t.Fatal("expected RLS set_config call on connection")
	}
}

func TestRegisterRejectsActorScopeMismatch(t *testing.T) {
	c := &fakeConn{}
	repo := newFakeRepo(c)

	meta := validMetadata("tenant-1", "org-1", "actor-1", "ev-1")
	wrongActor := evidence.ActorContext{TenantID: "tenant-2", OrganizationID: "org-1", ActorID: "actor-1"}
	_, _, err := repo.Register(context.Background(), wrongActor, meta)
	if err == nil {
		t.Fatal("expected error on tenant scope mismatch")
	}
}

func TestRegisterHandlesDeduplicationAndImmutableConflict(t *testing.T) {
	meta := validMetadata("tenant-1", "org-1", "actor-1", "ev-1")
	actor := evidence.ActorContext{TenantID: "tenant-1", OrganizationID: "org-1", ActorID: "actor-1"}

	// 1. Same immutable content on conflict -> duplicate accepted without error (inserted = false)
	c1 := &fakeConn{
		rowsAffected: 0, // ON CONFLICT DO NOTHING
		colsToReturn: evidenceColumns,
		rowsToReturn: [][]driver.Value{metadataToRow(meta)},
	}
	repo1 := newFakeRepo(c1)
	stored, inserted, err := repo1.Register(context.Background(), actor, meta)
	if err != nil {
		t.Fatalf("unexpected error on idempotent duplicate: %v", err)
	}
	if inserted {
		t.Fatal("expected inserted == false on duplicate")
	}
	if stored.ID != meta.ID {
		t.Fatalf("stored ID mismatch: %s", stored.ID)
	}

	// 2. Differing immutable content on conflict -> ErrImmutableConflict
	differingMeta := meta
	differingMeta.PlaintextSHA256 = strings.Repeat("a", 64)
	c2 := &fakeConn{
		rowsAffected: 0,
		colsToReturn: evidenceColumns,
		rowsToReturn: [][]driver.Value{metadataToRow(differingMeta)}, // existing row has different sha
	}
	repo2 := newFakeRepo(c2)
	_, _, err = repo2.Register(context.Background(), actor, meta)
	if !errors.Is(err, evidence.ErrImmutableConflict) {
		t.Fatalf("expected ErrImmutableConflict, got %v", err)
	}
}

func TestRegisterForActiveAssignment(t *testing.T) {
	meta := validMetadata("tenant-1", "org-1", "actor-1", "ev-1")
	actor := workorder.ActorContext{
		TenantID:       "tenant-1",
		OrganizationID: "org-1",
		ActorID:        "actor-1",
		Role:           "inspector",
	}

	// Case 1: Assignment permitted
	c1 := &fakeConn{
		rowsAffected: 1,
		queryResults: []queryResult{
			{cols: []string{"exists"}, rows: [][]driver.Value{{true}}},
			{cols: evidenceColumns, rows: [][]driver.Value{metadataToRow(meta)}},
		},
	}
	repo1 := newFakeRepo(c1)
	stored, inserted, err := repo1.RegisterForActiveAssignment(context.Background(), actor, meta)
	if err != nil {
		t.Fatalf("unexpected error on active assignment: %v", err)
	}
	if !inserted || stored.ID != meta.ID {
		t.Fatalf("unexpected stored record: inserted=%v, ID=%s", inserted, stored.ID)
	}

	// Case 2: Assignment denied
	c2 := &fakeConn{
		colsToReturn: []string{"exists"},
		rowsToReturn: [][]driver.Value{{false}},
	}
	repo2 := newFakeRepo(c2)
	_, _, err = repo2.RegisterForActiveAssignment(context.Background(), actor, meta)
	if !errors.Is(err, ErrInspectionAssignmentDenied) {
		t.Fatalf("expected ErrInspectionAssignmentDenied, got: %v", err)
	}
}

func TestListByInspectionAndTenant(t *testing.T) {
	meta1 := validMetadata("tenant-1", "org-1", "actor-1", "ev-1")
	meta2 := validMetadata("tenant-1", "org-1", "actor-1", "ev-2")
	actor := evidence.ActorContext{TenantID: "tenant-1", OrganizationID: "org-1", ActorID: "actor-1"}

	c := &fakeConn{
		colsToReturn: evidenceColumns,
		rowsToReturn: [][]driver.Value{
			metadataToRow(meta1),
			metadataToRow(meta2),
		},
	}
	repo := newFakeRepo(c)

	list, err := repo.ListByInspection(context.Background(), actor, "insp-001")
	if err != nil {
		t.Fatalf("unexpected error listing by inspection: %v", err)
	}
	if len(list) != 2 || list[0].ID != "ev-1" || list[1].ID != "ev-2" {
		t.Fatalf("unexpected list: %+v", list)
	}
}
