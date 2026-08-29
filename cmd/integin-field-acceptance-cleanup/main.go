package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"integin/internal/storage"
)

type receipt struct {
	TenantID       string `json:"tenant_id"`
	OrganizationID string `json:"organization_id"`
	DeviceID       string `json:"device_id"`
	AuthorityID    string `json:"authority_id"`
	InspectionID   string `json:"inspection_id"`
	EvidenceID     string `json:"evidence_id"`
}

func main() {
	receiptPath := flag.String("receipt", "", "path to the isolated Field acceptance receipt")
	flag.Parse()
	if strings.TrimSpace(*receiptPath) == "" {
		fail(errors.New("receipt path is required"))
	}
	value, err := loadReceipt(*receiptPath)
	if err != nil {
		fail(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := cleanupDatabase(ctx, value); err != nil {
		fail(err)
	}
	if err := cleanupEvidence(ctx, value); err != nil {
		fail(err)
	}
	if err := verifyDatabaseCleanup(ctx, value); err != nil {
		fail(err)
	}
	if err := os.Remove(*receiptPath); err != nil && !os.IsNotExist(err) {
		fail(fmt.Errorf("remove receipt: %w", err))
	}
	fmt.Printf("FIELD_ACCEPTANCE_CLEANUP=passed device_id=%s evidence_id=%s\n", value.DeviceID, value.EvidenceID)
}

func loadReceipt(path string) (receipt, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return receipt{}, fmt.Errorf("read receipt: %w", err)
	}
	var value receipt
	if err := json.Unmarshal(data, &value); err != nil {
		return receipt{}, fmt.Errorf("decode receipt: %w", err)
	}
	for name, field := range map[string]string{
		"tenant_id": value.TenantID, "organization_id": value.OrganizationID,
		"device_id":     value.DeviceID,
		"inspection_id": value.InspectionID, "evidence_id": value.EvidenceID,
	} {
		if strings.TrimSpace(field) == "" || !strings.HasPrefix(field, "field-") && name == "device_id" {
			return receipt{}, fmt.Errorf("receipt %s is invalid", name)
		}
	}
	if !strings.HasPrefix(value.InspectionID, "flutter-live-") || !strings.HasPrefix(value.EvidenceID, "flutter-live-evidence-") {
		return receipt{}, errors.New("receipt does not identify a namespaced Field acceptance fixture")
	}
	return value, nil
}

func cleanupDatabase(ctx context.Context, value receipt) error {
	dsn := strings.TrimSpace(os.Getenv("INTEGIN_DB_URL"))
	if dsn == "" {
		return errors.New("INTEGIN_DB_URL is required")
	}
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin cleanup transaction: %w", err)
	}
	defer tx.Rollback()
	for _, setting := range []struct{ key, value string }{{"integin.tenant_id", value.TenantID}, {"integin.organization_id", value.OrganizationID}} {
		if _, err := tx.ExecContext(ctx, "SELECT set_config($1, $2, true)", setting.key, setting.value); err != nil {
			return fmt.Errorf("set cleanup context: %w", err)
		}
	}
	for _, statement := range []struct {
		name string
		sql  string
		args []any
	}{
		{"held transactions", `DELETE FROM sync_held_transaction WHERE tenant_id = $1 AND device_id = $2`, []any{value.TenantID, value.DeviceID}},
		{"sync receipts", `DELETE FROM sync_receipt WHERE tenant_id = $1 AND device_id = $2`, []any{value.TenantID, value.DeviceID}},
		{"device state", `DELETE FROM sync_device_state WHERE tenant_id = $1 AND device_id = $2`, []any{value.TenantID, value.DeviceID}},
		{"authority package", `DELETE FROM authority_package WHERE tenant_id = $1 AND device_id = $2`, []any{value.TenantID, value.DeviceID}},
		{"device registry", `DELETE FROM device_registry WHERE tenant_id = $1 AND device_id = $2`, []any{value.TenantID, value.DeviceID}},
	} {
		if _, err := tx.ExecContext(ctx, statement.sql, statement.args...); err != nil {
			return fmt.Errorf("delete %s: %w", statement.name, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit cleanup transaction: %w", err)
	}
	return nil
}

func cleanupEvidence(ctx context.Context, value receipt) error {
	store, err := evidenceStoreFromEnvironment()
	if err != nil {
		return err
	}
	key := value.TenantID + "/" + value.OrganizationID + "/evidence/" + value.EvidenceID
	if _, err := store.Get(ctx, key); err != nil {
		if strings.Contains(err.Error(), "status 404") {
			return nil
		}
		return fmt.Errorf("read evidence object before cleanup: %w", err)
	}
	if err := store.Delete(ctx, key); err != nil {
		return fmt.Errorf("delete evidence object: %w", err)
	}
	if _, err := store.Get(ctx, key); err == nil || !strings.Contains(err.Error(), "status 404") {
		if err == nil {
			return errors.New("evidence cleanup residue remains accessible")
		}
		return fmt.Errorf("verify evidence cleanup: %w", err)
	}
	return nil
}

func verifyDatabaseCleanup(ctx context.Context, value receipt) error {
	dsn := strings.TrimSpace(os.Getenv("INTEGIN_DB_URL"))
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open database for verification: %w", err)
	}
	defer database.Close()
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin cleanup verification transaction: %w", err)
	}
	defer tx.Rollback()
	for _, setting := range []struct{ key, value string }{{"integin.tenant_id", value.TenantID}, {"integin.organization_id", value.OrganizationID}} {
		if _, err := tx.ExecContext(ctx, "SELECT set_config($1, $2, true)", setting.key, setting.value); err != nil {
			return fmt.Errorf("set cleanup verification context: %w", err)
		}
	}
	for _, check := range []struct {
		name string
		sql  string
	}{
		{"device registry", `SELECT count(*) FROM device_registry WHERE tenant_id = $1 AND device_id = $2`},
		{"device state", `SELECT count(*) FROM sync_device_state WHERE tenant_id = $1 AND device_id = $2`},
		{"authority package", `SELECT count(*) FROM authority_package WHERE tenant_id = $1 AND device_id = $2`},
		{"sync receipts", `SELECT count(*) FROM sync_receipt WHERE tenant_id = $1 AND device_id = $2`},
		{"held transactions", `SELECT count(*) FROM sync_held_transaction WHERE tenant_id = $1 AND device_id = $2`},
	} {
		var count int
		if err := tx.QueryRowContext(ctx, check.sql, value.TenantID, value.DeviceID).Scan(&count); err != nil {
			return fmt.Errorf("verify %s cleanup: %w", check.name, err)
		}
		if count != 0 {
			return fmt.Errorf("cleanup residue: %s=%d", check.name, count)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit cleanup verification transaction: %w", err)
	}
	return nil
}

func evidenceStoreFromEnvironment() (storage.Store, error) {
	endpoint := firstEnv("INTEGIN_S3_ENDPOINT", "INTEGIN_RUSTFS_ENDPOINT", "INTEGIN_MINIO_ENDPOINT")
	bucket := firstEnv("INTEGIN_S3_BUCKET", "INTEGIN_RUSTFS_BUCKET", "INTEGIN_MINIO_BUCKET")
	accessKey := firstEnv("INTEGIN_S3_ACCESS_KEY", "INTEGIN_RUSTFS_ACCESS_KEY", "INTEGIN_MINIO_ACCESS_KEY")
	secretKey := firstEnv("INTEGIN_S3_SECRET_KEY", "INTEGIN_RUSTFS_SECRET_KEY", "INTEGIN_MINIO_SECRET_KEY")
	region := firstEnv("INTEGIN_S3_REGION", "INTEGIN_RUSTFS_REGION", "INTEGIN_MINIO_REGION")
	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" || region == "" {
		return nil, errors.New("isolated S3/RustFS cleanup configuration is incomplete")
	}
	return storage.NewS3Store(endpoint, bucket, nil, storage.NewAWSSigV4Signer(accessKey, secretKey, region, "s3"))
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
