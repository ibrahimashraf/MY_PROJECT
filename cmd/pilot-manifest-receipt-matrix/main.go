// Command pilot-manifest-receipt-matrix exercises only the isolated pilot
// manifest candidate. It never prints or persists private values, proof bodies,
// keys, database URLs, or identifiers. Its retained output is aggregate-only.
package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"database/sql"
	_ "github.com/lib/pq"

	"integin/internal/domain/sync"
	"integin/internal/domain/workpackage"
	"integin/internal/manifestreceipts"
)

const manifestPath = "/work-package-manifest"

type fixture struct {
	DeviceID       string `json:"device_id"`
	AuthorityID    string `json:"authority_id"`
	DeviceKeyID    string `json:"device_key_id"`
	TenantID       string `json:"tenant_id"`
	OrganizationID string `json:"organization_id"`
	InspectionID   string `json:"inspection_id"`
}

type request struct {
	Proof sync.DeviceProof `json:"proof"`
}

type summary struct {
	ContractVersion         string `json:"contract_version"`
	HTTPCaseCount           int    `json:"http_case_count"`
	FieldBindingCaseEmitted bool   `json:"field_binding_case_emitted"`
	FixtureHashRestored     bool   `json:"fixture_hash_restored"`
	FinalizationStatus      string `json:"finalization_status"`
	PrivateOutputSuppressed bool   `json:"private_output_suppressed"`
}

type fixtureIdentitySummary struct {
	DeviceRows     int `json:"device_rows"`
	AuthorityRows  int `json:"authority_rows"`
	PackageRows    int `json:"package_rows"`
	AssignmentRows int `json:"assignment_rows"`
}

func main() {
	serverURL := flag.String("server-url", "", "isolated loopback candidate URL")
	fixturePath := flag.String("fixture", "", "public pilot fixture")
	privateKeyPath := flag.String("private-key-file", "", "isolated pilot demo signing-key path")
	runID := flag.String("candidate-run-id", "", "opaque receipt run ID")
	receiptsDir := flag.String("receipts-dir", "", "restricted receipt run directory")
	publicErrorCodeFile := flag.String("public-error-code-file", "", "restricted wrapper-owned fixed error-code artifact")
	restoreOnly := flag.Bool("restore-only", false, "restore the deterministic isolated fixture package hash and exit")
	diagnoseFixtureIdentity := flag.Bool("diagnose-fixture-identity", false, "emit aggregate-only isolated fixture identity counts")
	flag.Parse()

	if err := run(*serverURL, *fixturePath, *privateKeyPath, *runID, *receiptsDir, *restoreOnly, *diagnoseFixtureIdentity); err != nil {
		code := matrixErrorCode(err)
		_ = writePublicErrorCode(*publicErrorCodeFile, code)
		fmt.Fprintln(os.Stderr, "MATRIX_ERROR_CODE="+code)
		os.Exit(1)
	}
}

func writePublicErrorCode(path, code string) error {
	if path == "" {
		return nil
	}
	if !isPublicMatrixErrorCode(code) {
		return errors.New("invalid public matrix error code")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.WriteString(f, code+"\n")
	return err
}

func isPublicMatrixErrorCode(code string) bool {
	switch code {
	case "MATRIX_ARGUMENTS_INVALID", "FIXTURE_LOAD_FAILED", "DB_INPUT_UNAVAILABLE", "DB_CONNECTION_FAILED", "FIXTURE_IDENTITY_MISSING", "MATRIX_LOCK_UNAVAILABLE", "FIXTURE_MUTATION_FAILED", "FIXTURE_RESTORE_FAILED", "FIXTURE_RESTORE_VERIFICATION_FAILED", "MATRIX_EXECUTION_FAILED":
		return true
	default:
		return false
	}
}

func matrixErrorCode(err error) string {
	switch err.Error() {
	case "isolated matrix database input is unavailable":
		return "DB_INPUT_UNAVAILABLE"
	case "isolated fixture database connection failed":
		return "DB_CONNECTION_FAILED"
	case "isolated pilot fixture identity is not present":
		return "FIXTURE_IDENTITY_MISSING"
	case "isolated matrix lock is unavailable":
		return "MATRIX_LOCK_UNAVAILABLE"
	case "isolated fixture package mutation failed":
		return "FIXTURE_MUTATION_FAILED"
	case "isolated fixture package restore failed":
		return "FIXTURE_RESTORE_FAILED"
	case "isolated fixture package restore verification failed":
		return "FIXTURE_RESTORE_VERIFICATION_FAILED"
	default:
		return "MATRIX_EXECUTION_FAILED"
	}
}

func run(serverURL, fixturePath, privateKeyPath, runID, receiptsDir string, restoreOnly, diagnoseFixtureIdentity bool) error {
	if fixturePath == "" {
		return errors.New("isolated matrix arguments are incomplete")
	}

	fx, err := loadFixture(fixturePath)
	if err != nil {
		return err
	}
	dbURL := strings.TrimSpace(os.Getenv("INTEGIN_PILOT_MATRIX_DB_URL"))
	if dbURL == "" {
		return errors.New("isolated matrix database input is unavailable")
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return errors.New("isolated fixture database connection failed")
	}
	defer db.Close()
	// The session-scoped advisory lock and RLS-scoped work each require their own
	// connection. This standalone driver does not share the application pool.
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)
	if diagnoseFixtureIdentity {
		return json.NewEncoder(os.Stdout).Encode(fixtureIdentityCounts(db, fx))
	}
	lockConn, err := db.Conn(context.Background())
	if err != nil {
		return errors.New("isolated matrix lock is unavailable")
	}
	defer lockConn.Close()
	if _, err := lockConn.ExecContext(context.Background(), `SELECT pg_advisory_lock(hashtext('integin-pilot-manifest-receipt-matrix'))`); err != nil {
		return errors.New("isolated matrix lock is unavailable")
	}
	defer lockConn.ExecContext(context.Background(), `SELECT pg_advisory_unlock(hashtext('integin-pilot-manifest-receipt-matrix'))`)
	if err := assertIsolatedFixtureIdentity(db, fx); err != nil {
		return err
	}
	expectedHash, err := expectedFixtureHash(fx)
	if err != nil {
		return err
	}
	if restoreOnly {
		return restorePackageHash(db, fx, expectedHash)
	}
	if !strings.HasPrefix(serverURL, "http://127.0.0.1:18080") || privateKeyPath == "" || runID == "" || receiptsDir == "" {
		return errors.New("isolated matrix arguments are incomplete")
	}
	privateKey, err := loadPrivateKey(privateKeyPath)
	if err != nil {
		return err
	}
	defer zero(privateKey)

	client := &http.Client{Timeout: 10 * time.Second}
	baseURL := strings.TrimRight(serverURL, "/") + manifestPath
	now := time.Now().UTC()
	newProof := func(requestID string) sync.DeviceProof {
		return signedProof(fx, privateKey, requestID, now, now.Add(2*time.Minute))
	}

	// 1. valid proof and 2. its replay share one signed request ID.
	valid := newProof(randomHex())
	if err := expectStatus(client, baseURL, valid, http.StatusOK); err != nil {
		return err
	}
	if err := expectStatus(client, baseURL, valid, http.StatusConflict); err != nil {
		return err
	}

	// 3. signature invalid uses an otherwise valid proof with a malformed signature.
	signatureInvalid := newProof(randomHex())
	signatureInvalid.Signature = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	if err := expectStatus(client, baseURL, signatureInvalid, http.StatusUnauthorized); err != nil {
		return err
	}

	// 4. expiry must be rejected before signature verification.
	expired := signedProof(fx, privateKey, randomHex(), now.Add(-2*time.Minute), now.Add(-time.Minute))
	if err := expectStatus(client, baseURL, expired, http.StatusUnauthorized); err != nil {
		return err
	}

	// 5. authority mismatch has a valid signature for a mismatched epoch.
	authorityMismatch := newProof(randomHex())
	authorityMismatch.AuthorityEpoch++
	authorityMismatch = resign(authorityMismatch, privateKey)
	if err := expectStatus(client, baseURL, authorityMismatch, http.StatusUnauthorized); err != nil {
		return err
	}

	// 6. key unknown mutates only the declared key ID, then signs the altered proof.
	keyUnknown := newProof(randomHex())
	keyUnknown.KeyID = "pilot-matrix-unknown-key"
	keyUnknown = resign(keyUnknown, privateKey)
	if err := expectStatus(client, baseURL, keyUnknown, http.StatusUnauthorized); err != nil {
		return err
	}

	// 7. package hash mutation occurs only in the isolated pilot fixture row and
	// is restored in defer even if candidate validation or HTTP execution fails.
	if err := restorePackageHash(db, fx, expectedHash); err != nil {
		return err
	}
	if err := mutatePackageHash(db, fx, expectedHash); err != nil {
		return err
	}
	restored := false
	defer func() {
		if err := restorePackageHash(db, fx, expectedHash); err == nil {
			restored = true
		}
	}()
	packageInvalid := newProof(randomHex())
	if err := expectStatus(client, baseURL, packageInvalid, http.StatusForbidden); err != nil {
		_ = restorePackageHash(db, fx, expectedHash)
		return err
	}
	if err := restorePackageHash(db, fx, expectedHash); err != nil {
		return err
	}
	restored = true

	// 8. This intentionally represents the source contract after the verified
	// manifest has been received; it is not a Flutter app execution claim.
	writer, err := manifestreceipts.NewV2Writer(runID, receiptsDir, time.Now)
	if err != nil {
		return err
	}
	if err := writer.EmitWithFailure(manifestreceipts.V2Observation{
		Case:            manifestreceipts.CaseFieldBinding,
		EventSource:     manifestreceipts.V2EventSourceFieldBinding,
		ObservedOutcome: "verified_cached",
		Transport:       manifestreceipts.V2Transport{Kind: manifestreceipts.V2TransportFieldBinding},
		ReplayState:     manifestreceipts.V2ReplayState{Scope: manifestreceipts.V2ReplayScopeNotApplicable},
		CorrelationID:   randomHex(),
		GeneratedAt:     time.Now().UTC(),
	}); err != nil {
		return err
	}
	final, err := manifestreceipts.FinalizeV2(runID, receiptsDir, time.Now)
	if err != nil {
		return err
	}
	if final.Status != manifestreceipts.V2FinalizationComplete || !restored {
		return errors.New("isolated matrix finalization is incomplete")
	}
	return json.NewEncoder(os.Stdout).Encode(summary{
		ContractVersion:         "1",
		HTTPCaseCount:           7,
		FieldBindingCaseEmitted: true,
		FixtureHashRestored:     restored,
		FinalizationStatus:      string(final.Status),
		PrivateOutputSuppressed: true,
	})
}

func loadFixture(path string) (fixture, error) {
	var result fixture
	raw, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(raw, &result) != nil || result.DeviceID == "" || result.AuthorityID == "" || result.DeviceKeyID == "" || result.TenantID == "" || result.OrganizationID == "" || result.InspectionID == "" {
		return fixture{}, errors.New("public pilot fixture is invalid")
	}
	return result, nil
}

func loadPrivateKey(path string) (ed25519.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("pilot signing key is unavailable")
	}
	defer zeroBytes(raw)
	raw = bytes.TrimSpace(raw)
	decoded := make([]byte, base64.RawURLEncoding.DecodedLen(len(raw)))
	n, err := base64.RawURLEncoding.Decode(decoded, raw)
	if err != nil {
		zeroBytes(decoded)
		decoded = make([]byte, base64.StdEncoding.DecodedLen(len(raw)))
		n, err = base64.StdEncoding.Decode(decoded, raw)
	}
	if err != nil || n != ed25519.PrivateKeySize {
		zeroBytes(decoded)
		return nil, errors.New("pilot signing key is invalid")
	}
	return ed25519.PrivateKey(decoded[:n]), nil
}

func signedProof(f fixture, privateKey ed25519.PrivateKey, requestID string, issuedAt, expiresAt time.Time) sync.DeviceProof {
	proof := sync.DeviceProof{
		ProtocolVersion:    sync.DeviceProofProtocolVersion,
		Purpose:            sync.WorkPackageManifestReadPurpose,
		RequestID:          requestID,
		DeviceID:           f.DeviceID,
		AuthorityID:        f.AuthorityID,
		AuthorityEpoch:     1,
		InspectionID:       f.InspectionID,
		IssuedAt:           issuedAt.UTC(),
		ExpiresAt:          expiresAt.UTC(),
		SignatureAlgorithm: workpackage.ManifestProofSignatureAlgorithm,
		KeyID:              f.DeviceKeyID,
	}
	return resign(proof, privateKey)
}

func resign(proof sync.DeviceProof, privateKey ed25519.PrivateKey) sync.DeviceProof {
	canonical := workpackage.CanonicalManifestReadProof(proof.RequestID, proof.DeviceID, proof.AuthorityID, proof.AuthorityEpoch, proof.InspectionID, proof.IssuedAt, proof.ExpiresAt, proof.KeyID)
	proof.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(canonical)))
	return proof
}

func expectStatus(client *http.Client, endpoint string, proof sync.DeviceProof, expected int) error {
	body, err := json.Marshal(request{Proof: proof})
	if err != nil {
		return err
	}
	response, err := client.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return errors.New("isolated manifest request failed")
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 16<<10))
	if response.StatusCode != expected {
		return errors.New("isolated manifest case returned an unexpected status")
	}
	return nil
}

func expectedFixtureHash(f fixture) (string, error) {
	p := workpackage.Package{ID: "pilot-manifest-demo-package", TenantID: f.TenantID, OrganizationID: f.OrganizationID, TemplateCode: "pilot-manifest-demo", TemplateVersion: 1, PackageVersion: 1, SchemaVersion: 1, State: workpackage.PublicationApproved, Sections: []workpackage.Section{{ID: "pilot-section", Title: "Pilot manifest binding", Fields: []workpackage.FieldDefinition{{ID: "condition", Prompt: "Pilot condition", Type: workpackage.FieldText, Required: true}}}}}
	computed, err := p.WithComputedHash()
	if err != nil {
		return "", err
	}
	return computed.PackageHash, nil
}

func assertIsolatedFixtureIdentity(db *sql.DB, f fixture) error {
	const query = `SELECT EXISTS (
		SELECT 1
		FROM device_registry d
		JOIN authority_package a ON a.authority_id = $1 AND a.device_id = d.device_id AND a.tenant_id = d.tenant_id AND a.organization_id = d.organization_id
		JOIN work_package p ON p.tenant_id = d.tenant_id AND p.organization_id = d.organization_id AND p.package_id = 'pilot-manifest-demo-package' AND p.package_version = 1
		JOIN work_package_assignment w ON w.tenant_id = d.tenant_id AND w.organization_id = d.organization_id AND w.device_id = d.device_id AND w.package_id = p.package_id AND w.package_version = p.package_version AND w.inspection_id = $6
		WHERE d.device_id = $2 AND d.tenant_id = $3 AND d.organization_id = $4 AND d.key_id = $5 AND a.authority_epoch = 1 AND w.authority_epoch = 1
	)`
	present := false
	if err := withFixtureScope(db, f, func(tx *sql.Tx) error {
		return tx.QueryRow(query, f.AuthorityID, f.DeviceID, f.TenantID, f.OrganizationID, f.DeviceKeyID, f.InspectionID).Scan(&present)
	}); err != nil || !present {
		return errors.New("isolated pilot fixture identity is not present")
	}
	return nil
}

func fixtureIdentityCounts(db *sql.DB, f fixture) fixtureIdentitySummary {
	summary := fixtureIdentitySummary{DeviceRows: -1, AuthorityRows: -1, PackageRows: -1, AssignmentRows: -1}
	count := func(tx *sql.Tx, query string, args ...any) (int, error) {
		var value int
		err := tx.QueryRow(query, args...).Scan(&value)
		return value, err
	}
	if err := withFixtureScope(db, f, func(tx *sql.Tx) error {
		var err error
		if summary.DeviceRows, err = count(tx, `SELECT count(*) FROM device_registry WHERE device_id = $1 AND tenant_id = $2 AND organization_id = $3 AND key_id = $4`, f.DeviceID, f.TenantID, f.OrganizationID, f.DeviceKeyID); err != nil {
			return err
		}
		if summary.AuthorityRows, err = count(tx, `SELECT count(*) FROM authority_package WHERE authority_id = $1 AND tenant_id = $2 AND organization_id = $3 AND device_id = $4 AND authority_epoch = 1`, f.AuthorityID, f.TenantID, f.OrganizationID, f.DeviceID); err != nil {
			return err
		}
		if summary.PackageRows, err = count(tx, `SELECT count(*) FROM work_package WHERE tenant_id = $1 AND organization_id = $2 AND package_id = 'pilot-manifest-demo-package' AND package_version = 1`, f.TenantID, f.OrganizationID); err != nil {
			return err
		}
		summary.AssignmentRows, err = count(tx, `SELECT count(*) FROM work_package_assignment WHERE tenant_id = $1 AND organization_id = $2 AND inspection_id = $3 AND device_id = $4 AND package_id = 'pilot-manifest-demo-package' AND package_version = 1 AND authority_epoch = 1`, f.TenantID, f.OrganizationID, f.InspectionID, f.DeviceID)
		return err
	}); err != nil {
		return fixtureIdentitySummary{DeviceRows: -1, AuthorityRows: -1, PackageRows: -1, AssignmentRows: -1}
	}
	return summary
}

func mutatePackageHash(db *sql.DB, f fixture, expectedHash string) error {
	if err := withFixtureScope(db, f, func(tx *sql.Tx) error {
		result, err := tx.Exec(`UPDATE work_package SET package_hash = 'sha256:pilot-matrix-intentionally-invalid' WHERE tenant_id = $1 AND organization_id = $2 AND package_id = $3 AND package_version = 1 AND package_hash = $4`, f.TenantID, f.OrganizationID, "pilot-manifest-demo-package", expectedHash)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil || count != 1 {
			return errors.New("isolated fixture package mutation failed")
		}
		return nil
	}); err != nil {
		return errors.New("isolated fixture package mutation failed")
	}
	return nil
}

func restorePackageHash(db *sql.DB, f fixture, expectedHash string) error {
	if err := withFixtureScope(db, f, func(tx *sql.Tx) error {
		result, err := tx.Exec(`UPDATE work_package SET package_hash = $1 WHERE tenant_id = $2 AND organization_id = $3 AND package_id = $4 AND package_version = 1 AND package_hash IN ($1, 'sha256:pilot-matrix-intentionally-invalid')`, expectedHash, f.TenantID, f.OrganizationID, "pilot-manifest-demo-package")
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil || count != 1 {
			return errors.New("isolated fixture package restore failed")
		}
		var persisted string
		if err := tx.QueryRow(`SELECT package_hash FROM work_package WHERE tenant_id = $1 AND organization_id = $2 AND package_id = $3 AND package_version = 1`, f.TenantID, f.OrganizationID, "pilot-manifest-demo-package").Scan(&persisted); err != nil || persisted != expectedHash {
			return errors.New("isolated fixture package restore verification failed")
		}
		return nil
	}); err != nil {
		if err.Error() == "isolated fixture package restore verification failed" {
			return err
		}
		return errors.New("isolated fixture package restore failed")
	}
	return nil
}

func withFixtureScope(db *sql.DB, f fixture, action func(*sql.Tx) error) error {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`SELECT set_config('integin.tenant_id', $1, true)`, f.TenantID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`SELECT set_config('integin.organization_id', $1, true)`, f.OrganizationID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := action(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func randomHex() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic("randomness unavailable")
	}
	return fmt.Sprintf("%x", bytes)
}

func zero(key ed25519.PrivateKey) {
	zeroBytes(key)
}

func zeroBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
