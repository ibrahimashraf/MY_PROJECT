package releasegate

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func validReleaseRecord() *ReleaseRecord {
	return &ReleaseRecord{
		ReleaseID: "rel-001",
		CreatedAt: time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC),
		GitCommit: "abc123def456",
		Version:   "1.0.0",
		ArtifactChecksums: map[string]string{
			"bin/integin-server": "deadbeef",
		},
		Migrations: []MigrationEntry{
			{Name: "0001_event_log", UpSHA256: "aa", DownSHA256: "bb", HasDown: true},
		},
		TestEvidence: TestEvidence{
			GoVetClean:  true,
			TestsPassed: 42,
			TestsFailed: 0,
			RaceClean:   true,
			RLSVerified: true,
		},
		DependencyInventory: DependencyInventory{
			GoVersion: "go1.27",
			Modules:   map[string]string{"github.com/gin-gonic/gin": "v1.10"},
		},
		Approvals: Approvals{
			Operator:        "ops-team",
			Security:        "sec-team",
			SafetyAuthority: "safety-team",
		},
		RollbackPlan: RollbackPlan{
			Trigger:                 "error rate > 5%",
			Owner:                   "oncall",
			AutomaticallyReversible: true,
			ManualSteps:             []string{"revert migration", "restart service"},
		},
		Signature: Signature{},
	}
}

func TestComputeDigestDeterminism(t *testing.T) {
	r1 := validReleaseRecord()
	r2 := validReleaseRecord()

	d1, err := ComputeDigest(r1)
	if err != nil {
		t.Fatal(err)
	}
	d2, err := ComputeDigest(r2)
	if err != nil {
		t.Fatal(err)
	}
	if d1 != d2 {
		t.Fatalf("digests differ: %s != %s", d1, d2)
	}
	if len(d1) != 64 {
		t.Fatalf("expected 64-char hex sha256, got %d chars", len(d1))
	}
}

func TestComputeDigestExcludesSignature(t *testing.T) {
	r1 := validReleaseRecord()
	r1.Signature = Signature{Algorithm: "Ed25519", KeyID: "key-1", SignatureValue: "aabb", RecordSHA256: "ccdd"}
	r2 := validReleaseRecord()
	r2.Signature = Signature{Algorithm: "SHA512", KeyID: "key-99", SignatureValue: "ff00", RecordSHA256: "1122"}

	d1, err := ComputeDigest(r1)
	if err != nil {
		t.Fatal(err)
	}
	d2, err := ComputeDigest(r2)
	if err != nil {
		t.Fatal(err)
	}
	if d1 != d2 {
		t.Fatalf("signature field leaks into digest: %s != %s", d1, d2)
	}
}

func TestComputeDigestChangesOnFieldModification(t *testing.T) {
	r1 := validReleaseRecord()
	d1, _ := ComputeDigest(r1)

	r2 := validReleaseRecord()
	r2.Version = "1.0.1"
	d2, _ := ComputeDigest(r2)

	if d1 == d2 {
		t.Fatal("digest unchanged after modifying version")
	}
}

func TestSignAndVerify(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	rec := validReleaseRecord()
	if err := Sign(rec, priv); err != nil {
		t.Fatal(err)
	}
	if rec.Signature.Algorithm != "Ed25519" {
		t.Fatalf("expected algorithm Ed25519, got %s", rec.Signature.Algorithm)
	}
	if err := Verify(rec, pub); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyFailsWithWrongKey(t *testing.T) {
	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	pub2, _, _ := ed25519.GenerateKey(rand.Reader)

	rec := validReleaseRecord()
	Sign(rec, priv1)

	if err := Verify(rec, pub2); err == nil {
		t.Fatal("expected verification failure with wrong key")
	}
}

func TestTamperDetection(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	rec := validReleaseRecord()
	Sign(rec, priv)

	fields := func() []func(*ReleaseRecord) {
		return []func(*ReleaseRecord){
			func(r *ReleaseRecord) { r.ReleaseID = "tampered" },
			func(r *ReleaseRecord) { r.GitCommit = "deadbeef" },
			func(r *ReleaseRecord) { r.Version = "9.9.9" },
			func(r *ReleaseRecord) { r.TestEvidence.TestsFailed = 1 },
			func(r *ReleaseRecord) { r.Approvals.Operator = "" },
			func(r *ReleaseRecord) { r.RollbackPlan.Owner = "" },
			func(r *ReleaseRecord) { r.Migrations[0].UpSHA256 = "ff" },
			func(r *ReleaseRecord) { r.ArtifactChecksums["bin/integin-server"] = "tampered" },
		}
	}

	for i, tamper := range fields() {
		clone := *rec
		clone.ArtifactChecksums = make(map[string]string)
		for k, v := range rec.ArtifactChecksums {
			clone.ArtifactChecksums[k] = v
		}
		migCopy := make([]MigrationEntry, len(rec.Migrations))
		copy(migCopy, rec.Migrations)
		clone.Migrations = migCopy

		tamper(&clone)
		err := Verify(&clone, pub)
		if err == nil {
			t.Fatalf("tamper field %d: expected failure but verification passed", i)
		}
	}
}

func TestComputeDigestMatchesManual(t *testing.T) {
	r := validReleaseRecord()
	digest, _ := ComputeDigest(r)

	d := releaseDigest{
		ReleaseID:           r.ReleaseID,
		CreatedAt:           r.CreatedAt,
		GitCommit:           r.GitCommit,
		Version:             r.Version,
		ArtifactChecksums:   r.ArtifactChecksums,
		Migrations:          r.Migrations,
		TestEvidence:        r.TestEvidence,
		DependencyInventory: r.DependencyInventory,
		Approvals:           r.Approvals,
		RollbackPlan:        r.RollbackPlan,
	}
	b, _ := json.Marshal(d)
	sum := sha256.Sum256(b)
	expected := hex.EncodeToString(sum[:])

	if digest != expected {
		t.Fatalf("digest mismatch: computed=%s manual=%s", digest, expected)
	}
}

func TestValidateReleaseRecordApprovals(t *testing.T) {
	rec := validReleaseRecord()
	rec.Approvals.Operator = ""
	if err := ValidateReleaseRecord(rec, "", nil); err == nil {
		t.Fatal("expected error for missing operator approval")
	}
}

func TestValidateReleaseRecordTestEvidence(t *testing.T) {
	rec := validReleaseRecord()
	rec.TestEvidence.TestsFailed = 1
	if err := ValidateReleaseRecord(rec, "", nil); err == nil {
		t.Fatal("expected error for TestsFailed > 0")
	}
	rec2 := validReleaseRecord()
	rec2.TestEvidence.GoVetClean = false
	if err := ValidateReleaseRecord(rec2, "", nil); err == nil {
		t.Fatal("expected error for GoVetClean false")
	}
}

func TestValidateReleaseRecordRollbackPlan(t *testing.T) {
	rec := validReleaseRecord()
	rec.RollbackPlan.Owner = ""
	if err := ValidateReleaseRecord(rec, "", nil); err == nil {
		t.Fatal("expected error for missing rollback owner")
	}
	rec2 := validReleaseRecord()
	rec2.RollbackPlan.Trigger = ""
	if err := ValidateReleaseRecord(rec2, "", nil); err == nil {
		t.Fatal("expected error for missing rollback trigger")
	}
	rec3 := validReleaseRecord()
	rec3.RollbackPlan.ManualSteps = nil
	if err := ValidateReleaseRecord(rec3, "", nil); err == nil {
		t.Fatal("expected error for missing manual steps")
	}
}

func TestValidateMigrationsAgainstDirectory(t *testing.T) {
	dir := t.TempDir()

	upContent := []byte("CREATE TABLE t1 (id int);")
	downContent := []byte("DROP TABLE t1;")

	os.WriteFile(filepath.Join(dir, "0001_test_migration.sql"), upContent, 0o644)
	os.WriteFile(filepath.Join(dir, "0001_test_migration.down.sql"), downContent, 0o644)

	upHash := sha256.Sum256(upContent)
	downHash := sha256.Sum256(downContent)

	rec := validReleaseRecord()
	rec.Migrations = []MigrationEntry{
		{
			Name:       "0001_test_migration",
			UpSHA256:   hex.EncodeToString(upHash[:]),
			DownSHA256: hex.EncodeToString(downHash[:]),
			HasDown:    true,
		},
	}

	if err := ValidateReleaseRecord(rec, dir, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMigrationsHashMismatch(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "0001_test.sql"), []byte("CREATE TABLE x;"), 0o644)

	rec := validReleaseRecord()
	rec.Migrations = []MigrationEntry{
		{Name: "0001_test", UpSHA256: "wronghash", DownSHA256: "", HasDown: false},
	}

	if err := ValidateReleaseRecord(rec, dir, nil); err == nil {
		t.Fatal("expected hash mismatch error")
	}
}

func TestValidateMigrationsMissingDownFile(t *testing.T) {
	dir := t.TempDir()
	upContent := []byte("CREATE TABLE x;")
	os.WriteFile(filepath.Join(dir, "0001_test.sql"), upContent, 0o644)

	upHash := sha256.Sum256(upContent)
	rec := validReleaseRecord()
	rec.Migrations = []MigrationEntry{
		{Name: "0001_test", UpSHA256: hex.EncodeToString(upHash[:]), DownSHA256: "", HasDown: true},
	}

	if err := ValidateReleaseRecord(rec, dir, nil); err == nil {
		t.Fatal("expected error for missing down file when has_down=true")
	}
}

func TestValidateMigrationsRecordHasEntryNotInDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "0001_real.sql"), []byte("SELECT 1;"), 0o644)

	hash := sha256.Sum256([]byte("SELECT 1;"))
	rec := validReleaseRecord()
	rec.Migrations = []MigrationEntry{
		{Name: "0001_real", UpSHA256: hex.EncodeToString(hash[:]), DownSHA256: "", HasDown: false},
		{Name: "9999_ghost", UpSHA256: "aa", DownSHA256: "", HasDown: false},
	}

	if err := ValidateReleaseRecord(rec, dir, nil); err == nil {
		t.Fatal("expected error for migration in record but not in directory")
	}
}

func TestValidateMigrationsDirHasFileNotInRecord(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "0001_real.sql"), []byte("SELECT 1;"), 0o644)

	rec := validReleaseRecord()
	rec.Migrations = nil

	if err := ValidateReleaseRecord(rec, dir, nil); err == nil {
		t.Fatal("expected error for migration in directory but not in record")
	}
}

func TestValidateRealMigrationsDirectory(t *testing.T) {
	migrationsDir := filepath.Join("..", "..", "migrations")
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		t.Skip("migrations/ directory not found; skipping real directory test")
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatal(err)
	}

	var migrations []MigrationEntry
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") || strings.HasSuffix(name, ".down.sql") {
			continue
		}
		base := strings.TrimSuffix(name, ".sql")
		upPath := filepath.Join(migrationsDir, name)
		upBytes, err := os.ReadFile(upPath)
		if err != nil {
			t.Fatal(err)
		}
		upHash := sha256.Sum256(upBytes)

		downPath := filepath.Join(migrationsDir, base+".down.sql")
		hasDown := fileExists(downPath)
		var downHash string
		if hasDown {
			downBytes, err := os.ReadFile(downPath)
			if err != nil {
				t.Fatal(err)
			}
			dh := sha256.Sum256(downBytes)
			downHash = hex.EncodeToString(dh[:])
		}

		migrations = append(migrations, MigrationEntry{
			Name:       base,
			UpSHA256:   hex.EncodeToString(upHash[:]),
			DownSHA256: downHash,
			HasDown:    hasDown,
		})
	}

	rec := validReleaseRecord()
	rec.Migrations = migrations

	if err := ValidateReleaseRecord(rec, migrationsDir, nil); err != nil {
		t.Fatalf("real migrations validation failed: %v", err)
	}
}
