package evidenceexport

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func testItems(t *testing.T) []EvidenceItem {
	t.Helper()
	now := time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC)
	return []EvidenceItem{
		{
			EvidenceID:          "ev_1001",
			ObjectKey:           "exports/tenant-a/2026/09/ev_1001.enc",
			ContentType:         "application/octet-stream",
			CapturedAt:          now,
			PlaintextSHA256:     strings.Repeat("a", 64),
			CiphertextSHA256:    strings.Repeat("b", 64),
			ByteLength:          4096,
			EncryptionAlgorithm: "AES-256-GCM",
			AuthorityDeviceID:   "dev_9001",
		},
		{
			EvidenceID:          "ev_1002",
			ObjectKey:           "exports/tenant-a/2026/09/ev_1002.enc",
			ContentType:         "application/octet-stream",
			CapturedAt:          now.Add(time.Minute),
			PlaintextSHA256:     strings.Repeat("c", 64),
			CiphertextSHA256:    strings.Repeat("d", 64),
			ByteLength:          2048,
			EncryptionAlgorithm: "AES-256-GCM",
			AuthorityDeviceID:   "dev_9001",
		},
	}
}

func testExporter() ExporterIdentity {
	return ExporterIdentity{UserID: "u_42", Role: "engineer", ClientVersion: "1.4.0"}
}

func testSigner(priv ed25519.PrivateKey) SignerFunc {
	return func(canonical []byte) ([]byte, error) {
		return ed25519.Sign(priv, canonical), nil
	}
}

func TestNewManifestComputesTotals(t *testing.T) {
	items := testItems(t)
	m, err := NewManifest("exp_1", "tenant-a", "org-a", "p_2026.1", testExporter(), items)
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	if m.ManifestVersion != ManifestVersion {
		t.Fatalf("ManifestVersion = %q, want %q", m.ManifestVersion, ManifestVersion)
	}
	if m.ObjectCount != 2 {
		t.Fatalf("ObjectCount = %d, want 2", m.ObjectCount)
	}
	if m.TotalByteLength != 4096+2048 {
		t.Fatalf("TotalByteLength = %d, want %d", m.TotalByteLength, 4096+2048)
	}
	if m.ManifestSHA256 == "" {
		t.Fatal("ManifestSHA256 must be computed")
	}
	if m.CreatedAt.IsZero() {
		t.Fatal("CreatedAt must be set")
	}
	if len(m.ManifestSHA256) != sha256.Size*2 {
		t.Fatalf("ManifestSHA256 length = %d, want %d", len(m.ManifestSHA256), sha256.Size*2)
	}
}

func TestNewManifestValidation(t *testing.T) {
	items := testItems(t)
	ex := testExporter()

	cases := []struct {
		name        string
		exportID    string
		tenantID    string
		exporter    ExporterIdentity
		items       []EvidenceItem
		wantErrPart string
	}{
		{name: "empty export id", tenantID: "tenant-a", wantErrPart: "export_id is required"},
		{name: "empty tenant", exportID: "exp_1", wantErrPart: "tenant_id is required"},
		{name: "empty exporter user", exportID: "exp_1", tenantID: "tenant-a", exporter: ExporterIdentity{Role: "r", ClientVersion: "v"}, items: items, wantErrPart: "exporter user_id is required"},
		{name: "empty exporter role", exportID: "exp_1", tenantID: "tenant-a", exporter: ExporterIdentity{UserID: "u", ClientVersion: "v"}, items: items, wantErrPart: "exporter role is required"},
		{name: "empty exporter client", exportID: "exp_1", tenantID: "tenant-a", exporter: ExporterIdentity{UserID: "u", Role: "r"}, items: items, wantErrPart: "exporter client_version is required"},
		{name: "empty item", exportID: "exp_1", tenantID: "tenant-a", exporter: ex, items: []EvidenceItem{{}}, wantErrPart: "evidence evidence_id is required"},
		{name: "bad plaintext hash", exportID: "exp_1", tenantID: "tenant-a", exporter: ex, items: []EvidenceItem{{
			EvidenceID: "e", ObjectKey: "k", ContentType: "c",
			CapturedAt: time.Now(), PlaintextSHA256: "zz", CiphertextSHA256: strings.Repeat("b", 64),
			EncryptionAlgorithm: "AES-256-GCM", AuthorityDeviceID: "d",
		}}, wantErrPart: "plaintext_sha256"},
		{name: "negative byte length", exportID: "exp_1", tenantID: "tenant-a", exporter: ex, items: []EvidenceItem{{
			EvidenceID: "e", ObjectKey: "k", ContentType: "c",
			CapturedAt: time.Now(), PlaintextSHA256: strings.Repeat("a", 64), CiphertextSHA256: strings.Repeat("b", 64),
			ByteLength: -1, EncryptionAlgorithm: "AES-256-GCM", AuthorityDeviceID: "d",
		}}, wantErrPart: "byte_length"},
		{name: "duplicate evidence id", exportID: "exp_1", tenantID: "tenant-a", exporter: ex, items: []EvidenceItem{testItems(t)[0], testItems(t)[0]}, wantErrPart: "duplicate evidence_id"},
		{name: "traversal object key", exportID: "exp_1", tenantID: "tenant-a", exporter: ex, items: []EvidenceItem{{
			EvidenceID: "e", ObjectKey: "a/../secret.enc", ContentType: "c",
			CapturedAt: time.Now(), PlaintextSHA256: strings.Repeat("a", 64), CiphertextSHA256: strings.Repeat("b", 64),
			EncryptionAlgorithm: "AES-256-GCM", AuthorityDeviceID: "d",
		}}, wantErrPart: "traversal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewManifest(tc.exportID, tc.tenantID, "org-a", "p_2026.1", tc.exporter, tc.items)
			if tc.wantErrPart == "" {
				if err == nil {
					return
				}
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErrPart) {
				t.Fatalf("error %q does not contain %q", err, tc.wantErrPart)
			}
		})
	}
}

func TestManifestSHA256Deterministic(t *testing.T) {
	items := testItems(t)
	ex := testExporter()
	m1, err := NewManifest("exp_1", "tenant-a", "org-a", "p_2026.1", ex, items)
	if err != nil {
		t.Fatalf("m1: %v", err)
	}
	m2, err := NewManifest("exp_1", "tenant-a", "org-a", "p_2026.1", ex, items)
	if err != nil {
		t.Fatalf("m2: %v", err)
	}
	m1.CreatedAt = time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC)
	m2.CreatedAt = m1.CreatedAt
	if err := m1.Sign(testSigner(priv(t))); err != nil {
		t.Fatalf("sign m1: %v", err)
	}
	if err := m2.Sign(testSigner(priv(t))); err != nil {
		t.Fatalf("sign m2: %v", err)
	}
	if m1.ManifestSHA256 != m2.ManifestSHA256 {
		t.Fatalf("sha not deterministic: %q != %q", m1.ManifestSHA256, m2.ManifestSHA256)
	}
}

func priv(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	_, k, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return k
}

func TestSignAndVerifyManifest(t *testing.T) {
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	m, err := NewManifest("exp_1", "tenant-a", "org-a", "p_2026.1", testExporter(), testItems(t))
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	if err := m.VerifyManifest(pub); err == nil {
		t.Fatal("unsigned manifest must not verify")
	} else if !strings.Contains(err.Error(), "export_signature") {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := m.Sign(testSigner(key)); err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if m.ExportSignature == "" {
		t.Fatal("ExportSignature must be set")
	}
	if err := m.VerifyManifest(pub); err != nil {
		t.Fatalf("VerifyManifest: %v", err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(m.ExportSignature)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	if len(sig) != ed25519.SignatureSize {
		t.Fatalf("signature size = %d, want %d", len(sig), ed25519.SignatureSize)
	}
}

func TestVerifyManifestWrongKey(t *testing.T) {
	_, key1, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key1: %v", err)
	}
	pub2, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key2: %v", err)
	}
	m, err := NewManifest("exp_1", "tenant-a", "org-a", "p_2026.1", testExporter(), testItems(t))
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	if err := m.Sign(testSigner(key1)); err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if err := m.VerifyManifest(pub2); err == nil {
		t.Fatal("manifest signed by another key must not verify")
	}
}

func TestVerifyManifestTamperDetection(t *testing.T) {
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tamper := func(fn func(*Manifest)) error {
		m, err := NewManifest("exp_1", "tenant-a", "org-a", "p_2026.1", testExporter(), testItems(t))
		if err != nil {
			t.Fatalf("NewManifest: %v", err)
		}
		if err := m.Sign(testSigner(key)); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		fn(m)
		return m.VerifyManifest(pub)
	}

	cases := []struct {
		name        string
		rejectError string
	}{
		{name: "object count", rejectError: "object_count"},
		{name: "byte length", rejectError: "total_byte_length"},
		{name: "item content type", rejectError: "manifest_sha256"},
		{name: "tenant id", rejectError: "manifest_sha256"},
		{name: "exporter user", rejectError: "manifest_sha256"},
		{name: "created at", rejectError: "manifest_sha256"},
		{name: "procedure version", rejectError: "manifest_sha256"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tamper(func(m *Manifest) {
				switch tc.name {
				case "object count":
					m.ObjectCount++
				case "byte length":
					m.TotalByteLength++
				case "item content type":
					m.EvidenceItems[0].ContentType = "application/pdf"
				case "tenant id":
					m.TenantID = "tenant-b"
				case "exporter user":
					m.ExporterIdentity.UserID = "u_evil"
				case "created at":
					m.CreatedAt = m.CreatedAt.Add(time.Hour)
				case "procedure version":
					m.ProcedureVersion = "p_2026.2"
				}
			})
			if err == nil {
				t.Fatalf("%s tamper must fail verification", tc.name)
			}
			if !strings.Contains(err.Error(), tc.rejectError) {
				t.Fatalf("error %q does not contain %q", err, tc.rejectError)
			}
		})
	}

	if err := tamper(func(m *Manifest) {
		m.ManifestSHA256 = strings.Repeat("0", 64)
	}); err == nil {
		t.Fatal("altered manifest_sha256 must fail verification")
	} else if !strings.Contains(err.Error(), "manifest_sha256") {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := tamper(func(m *Manifest) {
		m.ExportSignature = strings.Repeat("9", len(m.ExportSignature))
	}); err == nil {
		t.Fatal("altered export_signature must fail verification")
	} else if !strings.Contains(err.Error(), "verification failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManifestJSONRoundTrip(t *testing.T) {
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	m, err := NewManifest("exp_1", "tenant-a", "org-a", "p_2026.1", testExporter(), testItems(t))
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	if err := m.Sign(testSigner(key)); err != nil {
		t.Fatalf("Sign: %v", err)
	}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m2 Manifest
	if err := json.Unmarshal(raw, &m2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !bytes.Equal(raw, mustMarshal(t, &m2)) {
		t.Fatal("round trip changed canonical serialization")
	}
	if m2.ManifestSHA256 != m.ManifestSHA256 {
		t.Fatalf("sha changed after round trip")
	}
	if err := m2.VerifyManifest(pub); err != nil {
		t.Fatalf("round-trip manifest must verify: %v", err)
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}

func TestCanonicalDoesNotIncludeDerivedFields(t *testing.T) {
	m, err := NewManifest("exp_1", "tenant-a", "org-a", "p_2026.1", testExporter(), testItems(t))
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	before := m.ManifestSHA256
	canonical, err := m.canonicalPayload()
	if err != nil {
		t.Fatalf("canonicalPayload: %v", err)
	}
	want := hex.EncodeToString(sha256sum(canonical))
	if before != want {
		t.Fatalf("ManifestSHA256 = %q, want %q", before, want)
	}
	if bytes.Contains(canonical, []byte("manifest_sha256")) || bytes.Contains(canonical, []byte("export_signature")) {
		t.Fatal("canonical payload must exclude derived fields")
	}
}

func TestSignNilSigner(t *testing.T) {
	m, err := NewManifest("exp_1", "tenant-a", "org-a", "p_2026.1", testExporter(), testItems(t))
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	if err := m.Sign(nil); err == nil {
		t.Fatal("nil signer must error")
	}
}
