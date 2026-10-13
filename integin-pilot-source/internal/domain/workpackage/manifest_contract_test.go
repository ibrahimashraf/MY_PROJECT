package workpackage

import "testing"

func TestManifestProofContractIsDomainSeparated(t *testing.T) {
	if ManifestProofProtocolVersion != "device-proof/v1" {
		t.Fatalf("unexpected proof protocol version: %q", ManifestProofProtocolVersion)
	}
	if ManifestProofPurpose != "work_package_manifest.read" {
		t.Fatalf("unexpected proof purpose: %q", ManifestProofPurpose)
	}
	if ManifestProofSignatureAlgorithm != "Ed25519" {
		t.Fatalf("unexpected proof signature algorithm: %q", ManifestProofSignatureAlgorithm)
	}
	if ErrManifestProofReplayAlreadyConsumed == nil || ErrManifestProofReplayExpired == nil {
		t.Fatal("expected stable replay error sentinels")
	}
}
