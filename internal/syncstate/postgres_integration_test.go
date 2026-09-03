package syncstate

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

const syncStateDurabilityIntegrationDSNEnv = "INTEGIN_TEST_DATABASE_URL"

func TestPostgresHeldRecoveryAndAtomicPersistenceIntegration(t *testing.T) {
	dsn := os.Getenv(syncStateDurabilityIntegrationDSNEnv)
	if dsn == "" {
		t.Skipf("set %s to run PostgreSQL sync durability integration tests", syncStateDurabilityIntegrationDSNEnv)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	db.SetMaxOpenConns(5)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}
	repository, err := NewPostgresRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	stamp := time.Now().UTC().UnixNano()
	tenantID := fmt.Sprintf("it-sync-durability-%d", stamp)
	organizationID := "org-1"
	deviceID := fmt.Sprintf("device-%d", stamp)
	heldTransactionID := fmt.Sprintf("tx-held-%d", stamp)
	firstTransactionID := fmt.Sprintf("tx-first-%d", stamp)
	rollbackTransactionID := fmt.Sprintf("tx-rollback-%d", stamp)
	now := time.Date(2026, time.August, 21, 18, 0, 0, 0, time.UTC)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		for _, statement := range []string{
			`DELETE FROM sync_held_transaction WHERE tenant_id = $1`,
			`DELETE FROM sync_receipt WHERE tenant_id = $1`,
			`DELETE FROM sync_device_state WHERE tenant_id = $1`,
			`DELETE FROM device_registry WHERE tenant_id = $1`,
		} {
			if _, err := db.ExecContext(cleanupCtx, statement, tenantID); err != nil {
				t.Errorf("cleanup sync durability fixtures: %v", err)
				return
			}
		}
	})
	if err := repository.SaveDevice(ctx, DeviceRecord{DeviceID: deviceID, TenantID: tenantID, OrganizationID: organizationID, UserID: "user-1", KeyID: "key-1", PublicKey: []byte("public-key"), State: DeviceTrusted, AuthorityEpoch: 1, EnrolledAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("seed device: %v", err)
	}
	receipt := func(transactionID string, sequence uint64, outcome string) Receipt {
		return Receipt{TransactionID: transactionID, TenantID: tenantID, OrganizationID: organizationID, DeviceID: deviceID, UserID: "user-1", SequenceNumber: sequence, Operation: "InspectionSubmitted", EntityID: "inspection-1", PayloadHash: fmt.Sprintf("payload-%d", sequence), Outcome: outcome, Reason: outcome, CapturedAt: now, ReceivedAt: now}
	}
	heldReceipt := receipt(heldTransactionID, 2, "HELD")
	heldEnvelope, err := json.Marshal(map[string]string{"transaction_id": heldReceipt.TransactionID})
	if err != nil {
		t.Fatalf("marshal held envelope: %v", err)
	}
	if err := repository.SaveHeld(ctx, HeldTransaction{Receipt: heldReceipt, ExpectedSequence: 1, Envelope: heldEnvelope, FirstHeldAt: now, LastAttemptAt: now, LastError: "sequence gap"}); err != nil {
		t.Fatalf("save held transaction: %v", err)
	}
	storedHeld, err := repository.ListHeld(ctx, tenantID, deviceID)
	if err != nil || len(storedHeld) != 1 || storedHeld[0].Receipt.Outcome != "HELD" {
		t.Fatalf("expected exactly one durable held receipt and envelope, got %#v, err=%v", storedHeld, err)
	}
	if err := repository.SaveReceipt(ctx, receipt(firstTransactionID, 1, "APPLIED")); err != nil {
		t.Fatalf("apply predecessor: %v", err)
	}
	if err := repository.SaveReceipt(ctx, Receipt{TransactionID: heldReceipt.TransactionID, TenantID: heldReceipt.TenantID, OrganizationID: heldReceipt.OrganizationID, DeviceID: heldReceipt.DeviceID, UserID: heldReceipt.UserID, SequenceNumber: heldReceipt.SequenceNumber, Operation: heldReceipt.Operation, EntityID: heldReceipt.EntityID, PayloadHash: heldReceipt.PayloadHash, Outcome: "APPLIED", Reason: "replayed after predecessor", CapturedAt: heldReceipt.CapturedAt, ReceivedAt: now.Add(time.Second)}); err != nil {
		t.Fatalf("promote held transaction: %v", err)
	}
	promoted, err := repository.GetReceipt(ctx, tenantID, heldReceipt.TransactionID)
	if err != nil || promoted.Outcome != "APPLIED" {
		t.Fatalf("expected promoted applied receipt, got %#v, err=%v", promoted, err)
	}
	storedHeld, err = repository.ListHeld(ctx, tenantID, deviceID)
	if err != nil || len(storedHeld) != 0 {
		t.Fatalf("expected promoted held envelope removal, got %#v, err=%v", storedHeld, err)
	}
	sequence, err := repository.GetLastAcceptedSequence(ctx, tenantID, deviceID)
	if err != nil || sequence != 2 {
		t.Fatalf("expected durable sequence 2 after recovery, got %d, err=%v", sequence, err)
	}

	triggerName := fmt.Sprintf("it_sync_held_rollback_%d", stamp)
	functionName := fmt.Sprintf("%s_fn", triggerName)
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger AS $$ BEGIN IF NEW.transaction_id = '%s' THEN RAISE EXCEPTION 'intentional sync held rollback'; END IF; RETURN NEW; END; $$ LANGUAGE plpgsql; CREATE TRIGGER %s BEFORE INSERT ON sync_held_transaction FOR EACH ROW EXECUTE FUNCTION %s();`, functionName, rollbackTransactionID, triggerName, functionName)); err != nil {
		t.Fatalf("install atomicity trigger: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON sync_held_transaction; DROP FUNCTION IF EXISTS %s();", triggerName, functionName))
	})
	rollbackReceipt := receipt(rollbackTransactionID, 3, "HELD")
	if err := repository.SaveHeld(ctx, HeldTransaction{Receipt: rollbackReceipt, ExpectedSequence: 3, Envelope: heldEnvelope, FirstHeldAt: now, LastAttemptAt: now, LastError: "sequence gap"}); err == nil {
		t.Fatal("expected held transaction trigger failure")
	}
	var rollbackReceiptCount, rollbackHeldCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sync_receipt WHERE tenant_id = $1 AND transaction_id = $2`, tenantID, rollbackTransactionID).Scan(&rollbackReceiptCount); err != nil {
		t.Fatalf("count rollback receipt: %v", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sync_held_transaction WHERE tenant_id = $1 AND transaction_id = $2`, tenantID, rollbackTransactionID).Scan(&rollbackHeldCount); err != nil {
		t.Fatalf("count rollback held row: %v", err)
	}
	if rollbackReceiptCount != 0 || rollbackHeldCount != 0 {
		t.Fatalf("atomic held failure left receipt=%d held=%d", rollbackReceiptCount, rollbackHeldCount)
	}
}
