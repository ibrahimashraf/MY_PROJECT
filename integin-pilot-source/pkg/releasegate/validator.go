package releasegate

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ValidateReleaseRecord(record *ReleaseRecord, migrationsDir string, pubKey ed25519.PublicKey) error {
	digest, err := ComputeDigest(record)
	if err != nil {
		return fmt.Errorf("compute digest: %w", err)
	}
	if record.Signature.RecordSHA256 != "" && record.Signature.RecordSHA256 != digest {
		return fmt.Errorf("record digest mismatch: expected %s, got %s", digest, record.Signature.RecordSHA256)
	}
	if record.Signature.SignatureValue != "" && pubKey != nil {
		if err := Verify(record, pubKey); err != nil {
			return fmt.Errorf("signature verification: %w", err)
		}
	}
	if record.Approvals.Operator == "" {
		return errors.New("approvals.operator is required")
	}
	if record.Approvals.Security == "" {
		return errors.New("approvals.security is required")
	}
	if record.Approvals.SafetyAuthority == "" {
		return errors.New("approvals.safety_authority is required")
	}
	if record.RollbackPlan.Owner == "" {
		return errors.New("rollback_plan.owner is required")
	}
	if record.RollbackPlan.Trigger == "" {
		return errors.New("rollback_plan.trigger is required")
	}
	if len(record.RollbackPlan.ManualSteps) == 0 {
		return errors.New("rollback_plan.manual_steps is required")
	}
	if !record.TestEvidence.GoVetClean {
		return errors.New("test_evidence.go_vet_clean must be true")
	}
	if record.TestEvidence.TestsFailed != 0 {
		return fmt.Errorf("test_evidence.tests_failed must be 0, got %d", record.TestEvidence.TestsFailed)
	}

	if migrationsDir != "" {
		if err := validateMigrations(record, migrationsDir); err != nil {
			return err
		}
	}
	return nil
}

func validateMigrations(record *ReleaseRecord, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	recordMap := make(map[string]MigrationEntry)
	for _, m := range record.Migrations {
		recordMap[m.Name] = m
	}
	sqlFiles := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		if strings.HasSuffix(name, ".down.sql") {
			continue
		}
		sqlFiles[name] = true
		base := strings.TrimSuffix(name, ".sql")
		me, ok := recordMap[base]
		if !ok {
			return fmt.Errorf("migration %s in directory but not in release record", name)
		}
		upPath := filepath.Join(migrationsDir, name)
		upHash, err := fileSHA256(upPath)
		if err != nil {
			return fmt.Errorf("hash %s: %w", name, err)
		}
		if me.UpSHA256 != upHash {
			return fmt.Errorf("migration %s up hash mismatch: record=%s actual=%s", base, me.UpSHA256, upHash)
		}
		downName := base + ".down.sql"
		downPath := filepath.Join(migrationsDir, downName)
		downExists := fileExists(downPath)
		if me.HasDown != downExists {
			return fmt.Errorf("migration %s has_down=%v but down file exists=%v", base, me.HasDown, downExists)
		}
		if downExists {
			downHash, err := fileSHA256(downPath)
			if err != nil {
				return fmt.Errorf("hash %s: %w", downName, err)
			}
			if me.DownSHA256 != downHash {
				return fmt.Errorf("migration %s down hash mismatch: record=%s actual=%s", base, me.DownSHA256, downHash)
			}
		}
	}
	for _, me := range recordMap {
		upName := me.Name + ".sql"
		if !sqlFiles[upName] {
			return fmt.Errorf("migration %s in record but not in directory", me.Name)
		}
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
