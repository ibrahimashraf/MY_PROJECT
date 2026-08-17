package vectors

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"integin/internal/domain/workpackage"
)

type manifestReadProofVector struct {
	ProtocolVersion string `json:"protocol_version"`
	Purpose         string `json:"purpose"`
	SignatureAlg    string `json:"signature_algorithm"`
	RequestID       string `json:"request_id"`
	DeviceID        string `json:"device_id"`
	AuthorityID     string `json:"authority_id"`
	AuthorityEpoch  uint64 `json:"authority_epoch"`
	InspectionID    string `json:"inspection_id"`
	IssuedAt        string `json:"issued_at"`
	ExpiresAt       string `json:"expires_at"`
	KeyID           string `json:"key_id"`
	CanonicalUTF8   string `json:"canonical_utf8"`
}

func TestManifestReadProofV1Vector(t *testing.T) {
	raw, err := os.ReadFile("manifest_read_proof_v1.json")
	if err != nil {
		t.Fatalf("read vector: %v", err)
	}
	var vector manifestReadProofVector
	if err := json.Unmarshal(raw, &vector); err != nil {
		t.Fatalf("decode vector: %v", err)
	}
	issuedAt, err := time.Parse(time.RFC3339Nano, vector.IssuedAt)
	if err != nil {
		t.Fatalf("parse issued_at: %v", err)
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, vector.ExpiresAt)
	if err != nil {
		t.Fatalf("parse expires_at: %v", err)
	}
	if vector.ProtocolVersion != workpackage.ManifestProofProtocolVersion ||
		vector.Purpose != workpackage.ManifestProofPurpose ||
		vector.SignatureAlg != workpackage.ManifestProofSignatureAlgorithm {
		t.Fatal("vector contract identifiers do not match domain contract")
	}
	got := workpackage.CanonicalManifestReadProof(
		vector.RequestID,
		vector.DeviceID,
		vector.AuthorityID,
		vector.AuthorityEpoch,
		vector.InspectionID,
		issuedAt,
		expiresAt,
		vector.KeyID,
	)
	if got != vector.CanonicalUTF8 {
		t.Fatalf("canonical proof mismatch:\nwant %q\n got %q", vector.CanonicalUTF8, got)
	}
}
