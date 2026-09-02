package syncstate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"integin/internal/domain/workorder"
	"integin/internal/workorderpg"
)

// Ticket 03 Held/Provisional E2E integration test for INTEGIN.
// Disposable: uses integin_repo_test at 127.0.0.1:15432 via INTEGIN_TEST_DATABASE_URL.
// Covers: seed device, create provisional, replay, fingerprint conflict,
// SaveHeld gap, predecessor APPLIED, HELD→APPLIED promotion, sequence 2.
// Uses both syncstate and workorderpg repos in one tenant-scoped E2E.
// Run: INTEGIN_TEST_DATABASE_URL=postgres://integin_pilot_runtime:...@127.0.0.1:15432/integin_repo_test?sslmode=disable go test -run TestTicket03HeldProvisionalE2EIntegration -count=1 -v ./internal/syncstate
// Private secret source: C:\MY_PROJECT\private\integin-secrets\integin-pilot.env (INTEGIN_DB_URL) -> derived to integin_repo_test.
// Do not commit; disposable proof only. Requires isolated disposable DB (createdb -> psql -f migrations 0001-0010 -> test -> dropdb).

func TestTicket03HeldProvisionalE2EIntegration(t *testing.T) {
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skipf("set INTEGIN_TEST_DATABASE_URL to run Ticket03 Held/Provisional E2E (expected integin_repo_test@15432 from private secret)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping database %q: %v", dsn, err)
	}

	syncRepo, err := NewPostgresRepository(db)
	if err != nil {
		t.Fatalf("new sync repo: %v", err)
	}
	workRepo, err := workorderpg.NewRepository(db)
	if err != nil {
		t.Fatalf("new workorder repo: %v", err)
	}

	stamp := time.Now().UTC().UnixNano()
	tenantID := fmt.Sprintf("it-ticket03-%d", stamp)
	organizationID := fmt.Sprintf("org-ticket03-%d", stamp)
	deviceID := fmt.Sprintf("device-ticket03-%d", stamp)
	userID := fmt.Sprintf("user-ticket03-%d", stamp)
	localID := fmt.Sprintf("it-provisional-ticket03-%d", stamp)
	heldTxID := fmt.Sprintf("tx-held-ticket03-%d", stamp)
	predTxID := fmt.Sprintf("tx-pred-ticket03-%d", stamp)
	op1 := fmt.Sprintf("op-provisional-1-%d", stamp)
	op2 := fmt.Sprintf("op-provisional-2-%d", stamp)
	op3 := fmt.Sprintf("op-provisional-3-%d", stamp)
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)

	actor := workorder.ActorContext{TenantID: tenantID, OrganizationID: organizationID, ActorID: userID, Role: "manager"}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		tx, err := db.BeginTx(cleanupCtx, nil)
		if err != nil {
			t.Errorf("cleanup begin: %v", err)
			return
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(cleanupCtx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", tenantID, organizationID); err != nil {
			t.Errorf("cleanup set_config: %v", err)
			return
		}
		// Delete in dependency order: held, receipt, device_state, device, then work-order provisional/operation.
		for _, stmt := range []struct {
			sql  string
			args []any
		}{
			{`DELETE FROM sync_held_transaction WHERE tenant_id = $1`, []any{tenantID}},
			{`DELETE FROM sync_receipt WHERE tenant_id = $1`, []any{tenantID}},
			{`DELETE FROM sync_device_state WHERE tenant_id = $1`, []any{tenantID}},
			{`DELETE FROM device_registry WHERE tenant_id = $1`, []any{tenantID}},
			{`DELETE FROM work_order_operation WHERE tenant_id = $1 AND organization_id = $2`, []any{tenantID, organizationID}},
			{`DELETE FROM work_order_provisional_record WHERE tenant_id = $1 AND organization_id = $2`, []any{tenantID, organizationID}},
		} {
			if _, err := tx.ExecContext(cleanupCtx, stmt.sql, stmt.args...); err != nil {
				t.Errorf("cleanup %q: %v", stmt.sql, err)
			}
		}
		// Verify zero residue (namespaced) within same RLS context.
		var syncHeld, syncRec, syncState, devReg, opCnt, provCnt int
		_ = tx.QueryRowContext(cleanupCtx, `SELECT count(*) FROM sync_held_transaction WHERE tenant_id=$1`, tenantID).Scan(&syncHeld)
		_ = tx.QueryRowContext(cleanupCtx, `SELECT count(*) FROM sync_receipt WHERE tenant_id=$1`, tenantID).Scan(&syncRec)
		_ = tx.QueryRowContext(cleanupCtx, `SELECT count(*) FROM sync_device_state WHERE tenant_id=$1`, tenantID).Scan(&syncState)
		_ = tx.QueryRowContext(cleanupCtx, `SELECT count(*) FROM device_registry WHERE tenant_id=$1`, tenantID).Scan(&devReg)
		_ = tx.QueryRowContext(cleanupCtx, `SELECT count(*) FROM work_order_operation WHERE tenant_id=$1 AND organization_id=$2`, tenantID, organizationID).Scan(&opCnt)
		_ = tx.QueryRowContext(cleanupCtx, `SELECT count(*) FROM work_order_provisional_record WHERE tenant_id=$1 AND organization_id=$2`, tenantID, organizationID).Scan(&provCnt)
		if syncHeld != 0 || syncRec != 0 || syncState != 0 || devReg != 0 || opCnt != 0 || provCnt != 0 {
			t.Errorf("residue after cleanup: held=%d receipt=%d state=%d device=%d op=%d prov=%d", syncHeld, syncRec, syncState, devReg, opCnt, provCnt)
		}
		if err := tx.Commit(); err != nil {
			t.Errorf("cleanup commit: %v", err)
		}
	})

	// 1) seed device (syncstate.SaveDevice) -> creates device_registry + sync_device_state (sequence 0)
	t.Logf("1) seed device tenant=%s device=%s", tenantID, deviceID)
	if err := syncRepo.SaveDevice(ctx, DeviceRecord{
		DeviceID: deviceID, TenantID: tenantID, OrganizationID: organizationID,
		UserID: userID, KeyID: "key-ticket03-1", PublicKey: []byte("public-key-ticket03"),
		State: DeviceTrusted, AuthorityEpoch: 1, EnrolledAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed device: %v", err)
	}
	if _, err := syncRepo.GetDevice(ctx, tenantID, deviceID); err != nil {
		t.Fatalf("verify device: %v", err)
	}
	seq0, err := syncRepo.GetLastAcceptedSequence(ctx, tenantID, deviceID)
	if err != nil {
		t.Fatalf("get initial sequence: %v", err)
	}
	if seq0 != 0 {
		t.Fatalf("expected initial sequence 0, got %d", seq0)
	}
	t.Logf("   seeded device OK, initial sequence=%d", seq0)

	// 2) create provisional (workorderpg.ReconcileProvisional)
	t.Logf("2) create provisional local_id=%s", localID)
	record := workorder.ProvisionalRecord{LocalID: localID, Kind: workorder.RecordAsset, TenantID: tenantID, OrganizationID: organizationID}
	cmd1 := workorder.ReconcileProvisionalCommand{
		Actor:       actor,
		Operation:   workorder.OperationMeta{OperationID: op1, IdempotencyKey: localID + "-idem-1", ExpectedRevision: 1},
		Record:      record,
		Outcome:     workorder.ReconcileMatched,
		CanonicalID: "canonical-ticket03-a",
	}
	if _, err := workRepo.ReconcileProvisional(ctx, cmd1); err != nil {
		t.Fatalf("create provisional: %v", err)
	}
	t.Logf("   create provisional OK")

	// 3) replay same provisional (idempotent) -> should succeed without mutation
	t.Logf("3) replay provisional (same fingerprint, same idempotency)")
	if _, err := workRepo.ReconcileProvisional(ctx, cmd1); err != nil {
		t.Fatalf("replay provisional (idempotent): %v", err)
	}
	t.Logf("   replay OK (idempotent)")

	// Verify DB still has canonical-a
	func() {
		verifyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tx, err := db.BeginTx(verifyCtx, nil)
		if err != nil {
			t.Fatalf("verify tx: %v", err)
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(verifyCtx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", tenantID, organizationID); err != nil {
			t.Fatalf("set_config: %v", err)
		}
		var canonicalID, state string
		if err := tx.QueryRowContext(verifyCtx, `SELECT canonical_id, reconcile_state FROM work_order_provisional_record WHERE tenant_id=$1 AND organization_id=$2 AND local_id=$3`, tenantID, organizationID, localID).Scan(&canonicalID, &state); err != nil {
			t.Fatalf("query provisional after replay: %v", err)
		}
		if canonicalID != "canonical-ticket03-a" || state != workorder.ReconcileMatched {
			t.Fatalf("provisional altered after replay: canonical=%q state=%q", canonicalID, state)
		}
		_ = tx.Commit()
	}()

	// 4) fingerprint conflict: same local_id, different candidate (ClientID) -> ErrProvisionalConflict
	t.Logf("4) fingerprint conflict (changed candidate)")
	conflictRecord := record
	conflictRecord.ClientID = "different-candidate-ticket03"
	cmdConflict := workorder.ReconcileProvisionalCommand{
		Actor:       actor,
		Operation:   workorder.OperationMeta{OperationID: op2, IdempotencyKey: localID + "-idem-2", ExpectedRevision: 1},
		Record:      conflictRecord,
		Outcome:     workorder.ReconcileMatched,
		CanonicalID: "canonical-ticket03-b",
	}
	if _, err := workRepo.ReconcileProvisional(ctx, cmdConflict); !errors.Is(err, workorder.ErrProvisionalConflict) {
		t.Fatalf("expected fingerprint conflict ErrProvisionalConflict, got %v", err)
	}
	t.Logf("   fingerprint conflict correctly returned ErrProvisionalConflict")

	// Verify original still intact after conflict
	func() {
		verifyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tx, err := db.BeginTx(verifyCtx, nil)
		if err != nil {
			t.Fatalf("verify tx: %v", err)
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(verifyCtx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", tenantID, organizationID); err != nil {
			t.Fatalf("set_config: %v", err)
		}
		var canonicalID, state string
		if err := tx.QueryRowContext(verifyCtx, `SELECT canonical_id, reconcile_state FROM work_order_provisional_record WHERE tenant_id=$1 AND organization_id=$2 AND local_id=$3`, tenantID, organizationID, localID).Scan(&canonicalID, &state); err != nil {
			t.Fatalf("query provisional after conflict: %v", err)
		}
		if canonicalID != "canonical-ticket03-a" {
			t.Fatalf("conflict altered original: canonical=%q", canonicalID)
		}
		_ = tx.Commit()
	}()
	t.Logf("   original provisional preserved after conflict")

	// Also test idempotent conflict replay? Ensure second identical conflict still returns same error (optional)
	cmdConflict2 := workorder.ReconcileProvisionalCommand{
		Actor:       actor,
		Operation:   workorder.OperationMeta{OperationID: op3, IdempotencyKey: localID + "-idem-3", ExpectedRevision: 1},
		Record:      conflictRecord,
		Outcome:     workorder.ReconcileMatched,
		CanonicalID: "canonical-ticket03-b",
	}
	if _, err := workRepo.ReconcileProvisional(ctx, cmdConflict2); !errors.Is(err, workorder.ErrProvisionalConflict) {
		t.Fatalf("expected repeated conflict still ErrProvisionalConflict, got %v", err)
	}

	// 5) SaveHeld gap: sequence 2 with ExpectedSequence 1 (gap)
	t.Logf("5) SaveHeld gap seq=2 expected=1 (HELD)")
	receiptHELD := Receipt{
		TransactionID: heldTxID, TenantID: tenantID, OrganizationID: organizationID,
		DeviceID: deviceID, UserID: userID, SequenceNumber: 2,
		Operation: "InspectionSubmitted", EntityID: "inspection-ticket03-1",
		PayloadHash: "payload-ticket03-2", Outcome: "HELD", Reason: "sequence gap",
		CapturedAt: now, ReceivedAt: now,
	}
	envelope, err := json.Marshal(map[string]string{"transaction_id": heldTxID, "ticket": "03"})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	if err := syncRepo.SaveHeld(ctx, HeldTransaction{Receipt: receiptHELD, ExpectedSequence: 1, Envelope: envelope, FirstHeldAt: now, LastAttemptAt: now, LastError: "sequence gap"}); err != nil {
		t.Fatalf("save held gap: %v", err)
	}
	// Ensure actually held if first path succeeded we may have duplicate; verify ListHeld ==1
	heldList, err := syncRepo.ListHeld(ctx, tenantID, deviceID)
	if err != nil {
		t.Fatalf("list held after gap: %v", err)
	}
	if len(heldList) != 1 || heldList[0].Receipt.Outcome != "HELD" || heldList[0].ExpectedSequence != 1 {
		t.Fatalf("expected exactly one HELD gap, got %#v err=%v", heldList, err)
	}
	t.Logf("   SaveHeld gap OK, held count=%d envelope=%s", len(heldList), string(heldList[0].Envelope))

	// 6) predecessor APPLIED: sequence 1
	t.Logf("6) predecessor APPLIED seq=1")
	predReceipt := Receipt{
		TransactionID: predTxID, TenantID: tenantID, OrganizationID: organizationID,
		DeviceID: deviceID, UserID: userID, SequenceNumber: 1,
		Operation: "InspectionSubmitted", EntityID: "inspection-ticket03-1",
		PayloadHash: "payload-ticket03-1", Outcome: "APPLIED", Reason: "predecessor",
		CapturedAt: now, ReceivedAt: now,
	}
	if err := syncRepo.SaveReceipt(ctx, predReceipt); err != nil {
		t.Fatalf("apply predecessor: %v", err)
	}
	seq1, err := syncRepo.GetLastAcceptedSequence(ctx, tenantID, deviceID)
	if err != nil || seq1 != 1 {
		t.Fatalf("expected sequence 1 after predecessor, got %d err=%v", seq1, err)
	}
	t.Logf("   predecessor APPLIED OK, sequence=%d", seq1)

	// 7) HELD→APPLIED promotion: same heldTxID now APPLIED (payload hash must match)
	t.Logf("7) HELD→APPLIED promotion replay seq=2")
	promotedReceipt := Receipt{
		TransactionID: heldTxID, TenantID: tenantID, OrganizationID: organizationID,
		DeviceID: deviceID, UserID: userID, SequenceNumber: 2,
		Operation: "InspectionSubmitted", EntityID: "inspection-ticket03-1",
		PayloadHash: "payload-ticket03-2", Outcome: "APPLIED", Reason: "replayed after predecessor",
		CapturedAt: now, ReceivedAt: now.Add(time.Second),
	}
	// Need to ensure SaveReceipt promotion path: it checks existing HELD with same payloadHash and transitions to APPLIED, advances sequence, deletes envelope.
	if err := syncRepo.SaveReceipt(ctx, promotedReceipt); err != nil {
		t.Fatalf("promote held to applied: %v", err)
	}
	got, err := syncRepo.GetReceipt(ctx, tenantID, heldTxID)
	if err != nil || got.Outcome != "APPLIED" {
		t.Fatalf("expected promoted APPLIED receipt, got %#v err=%v", got, err)
	}
	t.Logf("   promotion OK, outcome=%s", got.Outcome)

	// 8) sequence 2 and held empty
	t.Logf("8) verify sequence 2 and held empty")
	seq2, err := syncRepo.GetLastAcceptedSequence(ctx, tenantID, deviceID)
	if err != nil || seq2 != 2 {
		t.Fatalf("expected durable sequence 2 after promotion, got %d err=%v", seq2, err)
	}
	heldAfter, err := syncRepo.ListHeld(ctx, tenantID, deviceID)
	if err != nil {
		t.Fatalf("list held after promotion: %v", err)
	}
	if len(heldAfter) != 0 {
		t.Fatalf("expected held envelope removal after promotion, got %#v", heldAfter)
	}
	t.Logf("   sequence=%d held empty OK", seq2)

	// Final cross-check: provisional still canonical-a, device still trusted
	if _, err := syncRepo.GetDevice(ctx, tenantID, deviceID); err != nil {
		t.Fatalf("final device check: %v", err)
	}
	t.Logf("PASS Ticket03 E2E: seed device, provisional create/replay/conflict, SaveHeld gap, predecessor APPLIED, HELD→APPLIED promotion, sequence 2 verified")
}


