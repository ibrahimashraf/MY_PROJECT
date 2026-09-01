package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"integin/internal/storage"
)

const manifestVersion = "integin-recovery-drill-v1"

type fixture struct {
	TenantID     string `json:"tenant_id"`
	Organization string `json:"organization_id"`
	ActorID      string `json:"actor_id"`
	WorkOrderID  string `json:"work_order_id"`
	ScopeID      string `json:"scope_id"`
	AssignmentID string `json:"assignment_id"`
	InspectionID string `json:"inspection_id"`
	StateEventID string `json:"state_event_id"`
}

type objectManifest struct {
	Key              string            `json:"key"`
	ContentType      string            `json:"content_type"`
	PlaintextSHA256  string            `json:"plaintext_sha256"`
	CiphertextSHA256 string            `json:"ciphertext_sha256"`
	Metadata         map[string]string `json:"metadata"`
	BackupFile       string            `json:"backup_file"`
	BackupSHA256     string            `json:"backup_sha256,omitempty"`
}

type manifest struct {
	Version   string         `json:"version"`
	CreatedAt time.Time      `json:"created_at"`
	Status    string         `json:"status"`
	Fixture   fixture        `json:"fixture"`
	Object    objectManifest `json:"object"`
}

type storeConfig struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
}

func main() {
	mode := flag.String("mode", "", "seed, backup-object, restore-object, verify-db, verify-object, or cleanup-source")
	manifestPath := flag.String("manifest", "", "path to the recovery drill manifest")
	flag.Parse()
	if strings.TrimSpace(*mode) == "" || strings.TrimSpace(*manifestPath) == "" {
		fatal(errors.New("mode and manifest are required"))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	var err error
	switch *mode {
	case "seed":
		err = seed(ctx, *manifestPath)
	case "backup-object":
		err = backupObject(ctx, *manifestPath)
	case "restore-object":
		err = restoreObject(ctx, *manifestPath)
	case "verify-db":
		err = verifyDatabase(ctx, *manifestPath)
	case "verify-object":
		err = verifyObject(ctx, *manifestPath)
	case "cleanup-source":
		err = cleanupSource(ctx, *manifestPath)
	default:
		err = fmt.Errorf("unsupported mode %q", *mode)
	}
	if err != nil {
		fatal(err)
	}
	fmt.Printf("recovery_drill_mode=%s status=ok\n", *mode)
}

func seed(ctx context.Context, manifestPath string) error {
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o700); err != nil {
		return err
	}
	id := fmt.Sprintf("it-recovery-%d", time.Now().UnixNano())
	fixture := fixture{
		TenantID:     "pilot-tenant-recovery-" + id,
		Organization: "pilot-organization-recovery-" + id,
		ActorID:      "inspector-" + id,
		WorkOrderID:  id + "-order",
		ScopeID:      id + "-scope",
		AssignmentID: id + "-assignment",
		InspectionID: id + "-inspection",
		StateEventID: id + "-event",
	}
	ciphertext := []byte("ciphertext-recovery-drill-" + id)
	plaintextDigest := sha256Hex([]byte("plaintext-recovery-drill-" + id))
	ciphertextDigest := sha256Hex(ciphertext)
	object := objectManifest{
		Key:              fixture.TenantID + "/" + fixture.Organization + "/evidence/" + id,
		ContentType:      "application/octet-stream",
		PlaintextSHA256:  plaintextDigest,
		CiphertextSHA256: ciphertextDigest,
		Metadata: map[string]string{
			"tenant_id":         fixture.TenantID,
			"organization_id":   fixture.Organization,
			"inspection_id":     fixture.InspectionID,
			"plaintext_sha256":  plaintextDigest,
			"ciphertext_sha256": ciphertextDigest,
		},
		BackupFile: filepath.Join(filepath.Dir(manifestPath), "evidence-object.bin"),
	}
	record := manifest{Version: manifestVersion, CreatedAt: time.Now().UTC(), Status: "prepared", Fixture: fixture, Object: object}
	if err := writeManifest(manifestPath, record); err != nil {
		return err
	}
	database, closeDB, err := openDatabase()
	if err != nil {
		return err
	}
	defer closeDB()
	if err := insertFixture(ctx, database, fixture); err != nil {
		return err
	}
	config, err := readStoreConfig()
	if err != nil {
		_ = cleanupFixture(ctx, database, fixture)
		return err
	}
	if err := ensureBucket(ctx, config); err != nil {
		_ = cleanupFixture(ctx, database, fixture)
		return err
	}
	store, err := newStore(config)
	if err != nil {
		_ = cleanupFixture(ctx, database, fixture)
		return err
	}
	if err := store.Put(ctx, storage.Object{Key: object.Key, ContentType: object.ContentType, Data: ciphertext, Metadata: object.Metadata}); err != nil {
		_ = cleanupFixture(ctx, database, fixture)
		return err
	}
	record.Status = "seeded"
	return writeManifest(manifestPath, record)
}

func backupObject(ctx context.Context, manifestPath string) error {
	record, err := readManifest(manifestPath)
	if err != nil {
		return err
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	object, err := store.Get(ctx, record.Object.Key)
	if err != nil {
		return err
	}
	if err := verifyObjectAgainstManifest(object, record.Object); err != nil {
		return fmt.Errorf("source object verification: %w", err)
	}
	if err := os.WriteFile(record.Object.BackupFile, object.Data, 0o600); err != nil {
		return err
	}
	record.Object.BackupSHA256 = sha256Hex(object.Data)
	record.Status = "object_backed_up"
	return writeManifest(manifestPath, record)
}

func restoreObject(ctx context.Context, manifestPath string) error {
	record, err := readManifest(manifestPath)
	if err != nil {
		return err
	}
	//nolint:gosec // path is CLI arg/config for test tool
	data, err := os.ReadFile(record.Object.BackupFile)
	if err != nil {
		return err
	}
	if sha256Hex(data) != record.Object.BackupSHA256 || sha256Hex(data) != record.Object.CiphertextSHA256 {
		return errors.New("object backup digest does not match manifest")
	}
	config, err := readStoreConfig()
	if err != nil {
		return err
	}
	if err := ensureBucket(ctx, config); err != nil {
		return err
	}
	store, err := newStore(config)
	if err != nil {
		return err
	}
	if err := store.Put(ctx, storage.Object{Key: record.Object.Key, ContentType: record.Object.ContentType, Data: data, Metadata: record.Object.Metadata}); err != nil {
		return err
	}
	record.Status = "object_restored"
	return writeManifest(manifestPath, record)
}

func verifyObject(ctx context.Context, manifestPath string) error {
	record, err := readManifest(manifestPath)
	if err != nil {
		return err
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	object, err := store.Get(ctx, record.Object.Key)
	if err != nil {
		return err
	}
	if err := verifyObjectAgainstManifest(object, record.Object); err != nil {
		return fmt.Errorf("restored object verification: %w", err)
	}
	return nil
}

func verifyObjectAgainstManifest(object storage.Object, expected objectManifest) error {
	if object.ContentType != expected.ContentType || sha256Hex(object.Data) != expected.CiphertextSHA256 {
		return errors.New("ciphertext or content type mismatch")
	}
	for key, value := range expected.Metadata {
		if object.Metadata[key] != value {
			return fmt.Errorf("metadata mismatch for %s", key)
		}
	}
	return nil
}

func verifyDatabase(ctx context.Context, manifestPath string) error {
	record, err := readManifest(manifestPath)
	if err != nil {
		return err
	}
	database, closeDB, err := openDatabase()
	if err != nil {
		return err
	}
	defer closeDB()
	for _, assertion := range []struct {
		query string
		id    string
	}{
		{"SELECT count(*) FROM work_order WHERE id=$1", record.Fixture.WorkOrderID},
		{"SELECT count(*) FROM work_order_scope_item WHERE id=$1", record.Fixture.ScopeID},
		{"SELECT count(*) FROM work_order_assignment WHERE id=$1", record.Fixture.AssignmentID},
		{"SELECT count(*) FROM inspection_record WHERE id=$1", record.Fixture.InspectionID},
		{"SELECT count(*) FROM work_order_state_event WHERE id=$1", record.Fixture.StateEventID},
	} {
		var count int
		if err := database.QueryRowContext(ctx, assertion.query, assertion.id).Scan(&count); err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("restored relational fixture count=%d for %s", count, assertion.id)
		}
	}
	if _, err := database.ExecContext(ctx, "DO $$ BEGIN CREATE ROLE integin_recovery_runtime NOLOGIN NOSUPERUSER NOBYPASSRLS; EXCEPTION WHEN duplicate_object THEN NULL; END $$; GRANT USAGE ON SCHEMA public TO integin_recovery_runtime; GRANT SELECT ON work_order, work_order_scope_item, work_order_assignment, inspection_record, work_order_state_event TO integin_recovery_runtime;"); err != nil {
		return err
	}
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SET LOCAL ROLE integin_recovery_runtime"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "SELECT set_config('integin.tenant_id',$1,true), set_config('integin.organization_id',$2,true)", record.Fixture.TenantID, record.Fixture.Organization); err != nil {
		return err
	}
	var visible int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM inspection_record WHERE id=$1", record.Fixture.InspectionID).Scan(&visible); err != nil {
		return err
	}
	if visible != 1 {
		return fmt.Errorf("matching RLS context saw %d inspection rows", visible)
	}
	if _, err := tx.ExecContext(ctx, "SELECT set_config('integin.organization_id',$1,true)", record.Fixture.Organization+"-other"); err != nil {
		return err
	}
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM inspection_record WHERE id=$1", record.Fixture.InspectionID).Scan(&visible); err != nil {
		return err
	}
	if visible != 0 {
		return fmt.Errorf("mismatched RLS context saw %d inspection rows", visible)
	}
	return tx.Commit()
}

func cleanupSource(ctx context.Context, manifestPath string) error {
	record, err := readManifest(manifestPath)
	if err != nil {
		return err
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	if err := store.Delete(ctx, record.Object.Key); err != nil {
		return err
	}
	database, closeDB, err := openDatabase()
	if err != nil {
		return err
	}
	defer closeDB()
	if err := cleanupFixture(ctx, database, record.Fixture); err != nil {
		return err
	}
	if _, err := store.Get(ctx, record.Object.Key); err == nil {
		return errors.New("source evidence object remains after cleanup")
	}
	var count int
	if err := database.QueryRowContext(ctx, "SELECT count(*) FROM work_order WHERE id=$1", record.Fixture.WorkOrderID).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return errors.New("source work-order fixture remains after cleanup")
	}
	record.Status = "source_cleaned"
	return writeManifest(manifestPath, record)
}

func insertFixture(ctx context.Context, database *sql.DB, f fixture) error {
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT set_config('integin.tenant_id',$1,true), set_config('integin.organization_id',$2,true)", f.TenantID, f.Organization); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order (id,tenant_id,organization_id,client_id,job_number,request_state,execution_state,commercial_state,certificate_state,revision,created_by,updated_by) VALUES ($1,$2,$3,'recovery-client',$1,'requested','in_progress','not_ready','not_started',1,$4,$4)", f.WorkOrderID, f.TenantID, f.Organization, f.ActorID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order_scope_item (id,tenant_id,organization_id,work_order_id,client_id,location_id,asset_id,asset_type) VALUES ($1,$2,$3,$4,'recovery-client','recovery-location','recovery-asset','equipment')", f.ScopeID, f.TenantID, f.Organization, f.WorkOrderID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order_assignment (id,tenant_id,organization_id,work_order_id,inspector_id,state,revision,effective_from,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,'active',1,now(),$5,$5)", f.AssignmentID, f.TenantID, f.Organization, f.WorkOrderID, f.ActorID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order_assignment_scope (tenant_id,organization_id,assignment_id,scope_item_id) VALUES ($1,$2,$3,$4)", f.TenantID, f.Organization, f.AssignmentID, f.ScopeID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO inspection_record (id,tenant_id,organization_id,work_order_id,scope_item_id,assignment_id,asset_id,inspector_id,lifecycle_state,revision,finalization_state,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$6,'recovery-asset',$7,'COMPLETED',1,'OPEN',$7,$7)", f.InspectionID, f.TenantID, f.Organization, f.WorkOrderID, f.ScopeID, f.AssignmentID, f.ActorID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order_state_event (id,tenant_id,organization_id,work_order_id,actor_id,event_type,previous_state,new_state,reason_code,operation_id,correlation_id) VALUES ($1,$2,$3,$4,$5,'recovery_seed','{}'::jsonb,'{}'::jsonb,'recovery_drill',$1,$1)", f.StateEventID, f.TenantID, f.Organization, f.WorkOrderID, f.ActorID); err != nil {
		return err
	}
	return tx.Commit()
}

func cleanupFixture(ctx context.Context, database *sql.DB, f fixture) error {
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT set_config('integin.tenant_id',$1,true), set_config('integin.organization_id',$2,true)", f.TenantID, f.Organization); err != nil {
		return err
	}
	for _, step := range []struct {
		statement string
		args      []any
	}{
		{"DELETE FROM work_order_state_event WHERE work_order_id=$1", []any{f.WorkOrderID}},
		{"DELETE FROM work_order_submission_item WHERE submission_segment_id IN (SELECT id FROM work_order_submission_segment WHERE work_order_id=$1)", []any{f.WorkOrderID}},
		{"DELETE FROM work_order_submission_segment WHERE work_order_id=$1", []any{f.WorkOrderID}},
		{"DELETE FROM work_order_operation WHERE aggregate_id=$1", []any{f.WorkOrderID}},
		{"DELETE FROM inspection_record WHERE work_order_id=$1", []any{f.WorkOrderID}},
		{"DELETE FROM work_order_assignment_scope WHERE assignment_id=$1", []any{f.AssignmentID}},
		{"DELETE FROM work_order_assignment WHERE work_order_id=$1", []any{f.WorkOrderID}},
		{"DELETE FROM work_order_scope_item WHERE work_order_id=$1", []any{f.WorkOrderID}},
		{"DELETE FROM work_order WHERE id=$1", []any{f.WorkOrderID}},
	} {
		if _, err := tx.ExecContext(ctx, step.statement, step.args...); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func openDatabase() (*sql.DB, func(), error) {
	dsn := strings.TrimSpace(os.Getenv("INTEGIN_RECOVERY_DB_URL"))
	if dsn == "" {
		return nil, nil, errors.New("INTEGIN_RECOVERY_DB_URL is required")
	}
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, nil, err
	}
	if err := database.Ping(); err != nil {
		_ = database.Close()
		return nil, nil, err
	}
	return database, func() { _ = database.Close() }, nil
}

func openStore() (*storage.S3Store, error) {
	config, err := readStoreConfig()
	if err != nil {
		return nil, err
	}
	return newStore(config)
}

func readStoreConfig() (storeConfig, error) {
	config := storeConfig{
		Endpoint:  strings.TrimSpace(os.Getenv("INTEGIN_RECOVERY_S3_ENDPOINT")),
		Bucket:    strings.TrimSpace(os.Getenv("INTEGIN_RECOVERY_S3_BUCKET")),
		AccessKey: strings.TrimSpace(os.Getenv("INTEGIN_RECOVERY_S3_ACCESS_KEY")),
		SecretKey: os.Getenv("INTEGIN_RECOVERY_S3_SECRET_KEY"),
		Region:    strings.TrimSpace(os.Getenv("INTEGIN_RECOVERY_S3_REGION")),
	}
	if config.Region == "" {
		config.Region = "us-east-1"
	}
	if config.Endpoint == "" || config.Bucket == "" || config.AccessKey == "" || config.SecretKey == "" {
		return storeConfig{}, errors.New("INTEGIN_RECOVERY_S3_ENDPOINT, INTEGIN_RECOVERY_S3_BUCKET, INTEGIN_RECOVERY_S3_ACCESS_KEY, and INTEGIN_RECOVERY_S3_SECRET_KEY are required")
	}
	return config, nil
}

func newStore(config storeConfig) (*storage.S3Store, error) {
	return storage.NewS3Store(config.Endpoint, config.Bucket, nil, storage.NewAWSSigV4Signer(config.AccessKey, config.SecretKey, config.Region, "s3"))
}

func ensureBucket(ctx context.Context, config storeConfig) error {
	endpoint, err := url.Parse(strings.TrimRight(config.Endpoint, "/") + "/" + url.PathEscape(config.Bucket))
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint.String(), nil)
	if err != nil {
		return err
	}
	if err := storage.NewAWSSigV4Signer(config.AccessKey, config.SecretKey, config.Region, "s3")(request); err != nil {
		return err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 200 && response.StatusCode < 300 || response.StatusCode == http.StatusConflict {
		return nil
	}
	return fmt.Errorf("bucket create failed with status %d", response.StatusCode)
}

//nolint:gosec // path is CLI arg/config for test tool
func readManifest(path string) (manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, err
	}
	var record manifest
	if err := json.Unmarshal(data, &record); err != nil {
		return manifest{}, err
	}
	if record.Version != manifestVersion || record.Fixture.WorkOrderID == "" || record.Object.Key == "" {
		return manifest{}, errors.New("invalid recovery drill manifest")
	}
	return record, nil
}

func writeManifest(path string, record manifest) error {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func sha256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "recovery drill failed:", err)
	os.Exit(1)
}
