package releasegate

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

type MigrationEntry struct {
	Name       string `json:"name"`
	UpSHA256   string `json:"up_sha256"`
	DownSHA256 string `json:"down_sha256"`
	HasDown    bool   `json:"has_down"`
}

type TestEvidence struct {
	GoVetClean  bool `json:"go_vet_clean"`
	TestsPassed int  `json:"tests_passed"`
	TestsFailed int  `json:"tests_failed"`
	RaceClean   bool `json:"race_clean"`
	RLSVerified bool `json:"rls_verified"`
}

type DependencyInventory struct {
	GoVersion string            `json:"go_version"`
	Modules   map[string]string `json:"modules"`
}

type Approvals struct {
	Operator        string `json:"operator"`
	Security        string `json:"security"`
	SafetyAuthority string `json:"safety_authority"`
}

type RollbackPlan struct {
	Trigger                 string   `json:"trigger"`
	Owner                   string   `json:"owner"`
	AutomaticallyReversible bool     `json:"automatically_reversible"`
	ManualSteps             []string `json:"manual_steps"`
}

type Signature struct {
	Algorithm      string `json:"algorithm"`
	KeyID          string `json:"key_id"`
	SignatureValue string `json:"signature_value"`
	RecordSHA256   string `json:"record_sha256"`
}

type ReleaseRecord struct {
	ReleaseID           string              `json:"release_id"`
	CreatedAt           time.Time           `json:"created_at"`
	GitCommit           string              `json:"git_commit"`
	Version             string              `json:"version"`
	ArtifactChecksums   map[string]string   `json:"artifact_checksums"`
	Migrations          []MigrationEntry    `json:"migrations"`
	TestEvidence        TestEvidence        `json:"test_evidence"`
	DependencyInventory DependencyInventory `json:"dependency_inventory"`
	Approvals           Approvals           `json:"approvals"`
	RollbackPlan        RollbackPlan        `json:"rollback_plan"`
	Signature           Signature           `json:"signature"`
}

type releaseDigest struct {
	ReleaseID           string              `json:"release_id"`
	CreatedAt           time.Time           `json:"created_at"`
	GitCommit           string              `json:"git_commit"`
	Version             string              `json:"version"`
	ArtifactChecksums   map[string]string   `json:"artifact_checksums"`
	Migrations          []MigrationEntry    `json:"migrations"`
	TestEvidence        TestEvidence        `json:"test_evidence"`
	DependencyInventory DependencyInventory `json:"dependency_inventory"`
	Approvals           Approvals           `json:"approvals"`
	RollbackPlan        RollbackPlan        `json:"rollback_plan"`
}

func ComputeDigest(rec *ReleaseRecord) (string, error) {
	d := releaseDigest{
		ReleaseID:           rec.ReleaseID,
		CreatedAt:           rec.CreatedAt,
		GitCommit:           rec.GitCommit,
		Version:             rec.Version,
		ArtifactChecksums:   rec.ArtifactChecksums,
		Migrations:          rec.Migrations,
		TestEvidence:        rec.TestEvidence,
		DependencyInventory: rec.DependencyInventory,
		Approvals:           rec.Approvals,
		RollbackPlan:        rec.RollbackPlan,
	}
	b, err := json.Marshal(d)
	if err != nil {
		return "", fmt.Errorf("marshal digest: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func Sign(rec *ReleaseRecord, privKey ed25519.PrivateKey) error {
	digest, err := ComputeDigest(rec)
	if err != nil {
		return err
	}
	digestBytes, err := hex.DecodeString(digest)
	if err != nil {
		return fmt.Errorf("decode digest: %w", err)
	}
	sig := ed25519.Sign(privKey, digestBytes)
	rec.Signature = Signature{
		Algorithm:      "Ed25519",
		SignatureValue: hex.EncodeToString(sig),
		RecordSHA256:   digest,
	}
	return nil
}

func Verify(rec *ReleaseRecord, pubKey ed25519.PublicKey) error {
	digest, err := ComputeDigest(rec)
	if err != nil {
		return err
	}
	if rec.Signature.RecordSHA256 != digest {
		return fmt.Errorf("record digest mismatch: expected %s, got %s", digest, rec.Signature.RecordSHA256)
	}
	sigBytes, err := hex.DecodeString(rec.Signature.SignatureValue)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	digestBytes, err := hex.DecodeString(digest)
	if err != nil {
		return fmt.Errorf("decode digest: %w", err)
	}
	if !ed25519.Verify(pubKey, digestBytes, sigBytes) {
		return fmt.Errorf("ed25519 signature verification failed")
	}
	return nil
}
