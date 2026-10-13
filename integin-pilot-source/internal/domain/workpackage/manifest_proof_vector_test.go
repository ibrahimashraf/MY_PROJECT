package workpackage

import (
	"testing"
	"time"
)

func TestCanonicalManifestReadProofVector(t *testing.T) {
	got := CanonicalManifestReadProof(
		"request-1",
		"device-1",
		"authority-1",
		7,
		"inspection-1",
		time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC),
		time.Date(2026, time.August, 17, 12, 5, 0, 0, time.UTC),
		"device-key-1",
	)
	const want = "device-proof/v1|work_package_manifest.read|request-1|device-1|authority-1|7|inspection-1|2026-08-17T12:00:00Z|2026-08-17T12:05:00Z|Ed25519|device-key-1"
	if got != want {
		t.Fatalf("canonical proof mismatch:\nwant %q\n got %q", want, got)
	}
}
