package evidenceexport

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func makePayload(t *testing.T, size int) []byte {
	t.Helper()
	b := make([]byte, size)
	for i := range b {
		b[i] = byte(i % 251)
	}
	return b
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func testPackItems(t *testing.T) ([]EvidenceItem, map[string][]byte) {
	t.Helper()
	now := time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC)
	payloads := [][]byte{makePayload(t, 4096), makePayload(t, 2048)}
	keys := []string{
		"exports/tenant-a/2026/09/ev_2001.enc",
		"exports/tenant-a/2026/09/ev_2002.enc",
	}
	store := map[string][]byte{keys[0]: payloads[0], keys[1]: payloads[1]}
	items := []EvidenceItem{
		{
			EvidenceID: "ev_2001", ObjectKey: keys[0], ContentType: "application/octet-stream",
			CapturedAt: now, PlaintextSHA256: strings.Repeat("a", 64), CiphertextSHA256: hashBytes(payloads[0]),
			ByteLength: int64(len(payloads[0])), EncryptionAlgorithm: "AES-256-GCM", AuthorityDeviceID: "dev_9001",
		},
		{
			EvidenceID: "ev_2002", ObjectKey: keys[1], ContentType: "application/octet-stream",
			CapturedAt: now, PlaintextSHA256: strings.Repeat("c", 64), CiphertextSHA256: hashBytes(payloads[1]),
			ByteLength: int64(len(payloads[1])), EncryptionAlgorithm: "AES-256-GCM", AuthorityDeviceID: "dev_9001",
		},
	}
	return items, store
}

func getObjectFrom(store map[string][]byte) func(key string) (io.ReadCloser, error) {
	return func(key string) (io.ReadCloser, error) {
		payload, ok := store[key]
		if !ok {
			return nil, errors.New("object not found")
		}
		return io.NopCloser(bytes.NewReader(payload)), nil
	}
}

func readArchiveEntries(t *testing.T, raw []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	entries := make(map[string][]byte, len(zr.File))
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		entries[f.Name] = content
	}
	return entries
}

func TestCreateExportArchivePacksManifestAndPayloads(t *testing.T) {
	items, store := testPackItems(t)
	m, err := NewManifest("exp_1", "tenant-a", "org-a", "p_2026.1", testExporter(), items)
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}

	var buf bytes.Buffer
	if err := CreateExportArchive(m, getObjectFrom(store), &buf); err != nil {
		t.Fatalf("CreateExportArchive: %v", err)
	}

	entries := readArchiveEntries(t, buf.Bytes())
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	if len(entries) != len(items)+1 {
		t.Fatalf("got %d entries, want %d", len(entries), len(items)+1)
	}
	manifestJSON, ok := entries["manifest.json"]
	if !ok {
		t.Fatal("missing manifest.json entry")
	}
	var unpacked Manifest
	if err := json.Unmarshal(manifestJSON, &unpacked); err != nil {
		t.Fatalf("unmarshal manifest.json: %v", err)
	}
	if unpacked.ObjectCount != len(items) {
		t.Fatalf("archive object_count = %d, want %d", unpacked.ObjectCount, len(items))
	}
	for _, item := range items {
		payload, ok := entries[item.ObjectKey]
		if !ok {
			t.Fatalf("missing payload entry %s", item.ObjectKey)
		}
		if got := hashBytes(payload); got != item.CiphertextSHA256 {
			t.Fatalf("%s hash after archive = %s, want %s", item.ObjectKey, got, item.CiphertextSHA256)
		}
	}
}

func TestCreateExportArchiveDeterministic(t *testing.T) {
	items, store := testPackItems(t)
	fixed := time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC)

	base, err := NewManifest("exp_1", "tenant-a", "org-a", "p_2026.1", testExporter(), items)
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	base.CreatedAt = fixed
	if err := base.seal(); err != nil {
		t.Fatalf("seal: %v", err)
	}
	m1, m2 := *base, *base

	var b1, b2 bytes.Buffer
	if err := CreateExportArchive(&m1, getObjectFrom(store), &b1); err != nil {
		t.Fatalf("m1: %v", err)
	}
	if err := CreateExportArchive(&m2, getObjectFrom(store), &b2); err != nil {
		t.Fatalf("m2: %v", err)
	}
	if !bytes.Equal(b1.Bytes(), b2.Bytes()) {
		t.Fatalf("archives differ (%d vs %d bytes)", b1.Len(), b2.Len())
	}
}

func TestCreateExportArchiveRejectsChecksumMismatch(t *testing.T) {
	items, store := testPackItems(t)
	m, err := NewManifest("exp_1", "tenant-a", "org-a", "p_2026.1", testExporter(), items)
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	badKey := items[0].ObjectKey
	store[badKey] = makePayload(t, 4097)

	err = CreateExportArchive(m, getObjectFrom(store), &bytes.Buffer{})
	if err == nil {
		t.Fatal("checksum mismatch must error")
	}
	if !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("error %q does not mention sha256", err)
	}
}

func TestCreateExportArchiveFetchError(t *testing.T) {
	items, store := testPackItems(t)
	m, err := NewManifest("exp_1", "tenant-a", "org-a", "p_2026.1", testExporter(), items)
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	missing := items[1].ObjectKey
	delete(store, missing)

	err = CreateExportArchive(m, getObjectFrom(store), &bytes.Buffer{})
	if err == nil {
		t.Fatal("missing object must error")
	}
	if !strings.Contains(err.Error(), "fetch") {
		t.Fatalf("error %q does not mention fetch", err)
	}
}

func TestCreateExportArchiveEmptyManifest(t *testing.T) {
	m, err := NewManifest("exp_1", "tenant-a", "org-a", "p_2026.1", testExporter(), nil)
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}

	var buf bytes.Buffer
	if err := CreateExportArchive(m, getObjectFrom(nil), &buf); err != nil {
		t.Fatalf("CreateExportArchive: %v", err)
	}
	entries := readArchiveEntries(t, buf.Bytes())
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if _, ok := entries["manifest.json"]; !ok {
		t.Fatal("empty archive must still contain manifest.json")
	}
}

func TestCreateExportArchiveNilInputs(t *testing.T) {
	var buf bytes.Buffer
	store := func(key string) (io.ReadCloser, error) { return nil, nil }
	if err := CreateExportArchive(nil, store, &buf); err == nil {
		t.Fatal("nil manifest must error")
	}
	items, _ := testPackItems(t)
	m, err := NewManifest("exp_1", "tenant-a", "org-a", "p_2026.1", testExporter(), items)
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	if err := CreateExportArchive(m, nil, &buf); err == nil {
		t.Fatal("nil getObject must error")
	}
	if err := CreateExportArchive(m, getObjectFrom(nil), nil); err == nil {
		t.Fatal("nil writer must error")
	}
}
