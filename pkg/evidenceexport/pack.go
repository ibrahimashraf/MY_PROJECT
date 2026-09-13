package evidenceexport

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"
)

// archiveEntryTime is a fixed timestamp so every archive is reproducible.
var archiveEntryTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// CreateExportArchive writes a deterministic ZIP archive containing
// manifest.json followed by each referenced evidence payload (stored under
// its object key). Every payload is streamed through a SHA-256 hash and fails
// the archive if its digest does not match the manifest's ciphertext_sha256.
// Files are written in object-key order under a fixed mod time for byte-level
// reproducibility.
func CreateExportArchive(manifest *Manifest, getObject func(key string) (io.ReadCloser, error), writer io.Writer) (retErr error) {
	if manifest == nil {
		return errors.New("evidenceexport: manifest is nil")
	}
	if getObject == nil {
		return errors.New("evidenceexport: getObject is required")
	}
	if writer == nil {
		return errors.New("evidenceexport: writer is required")
	}
	if err := manifest.validateFields(); err != nil {
		return err
	}

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	items := append([]EvidenceItem(nil), manifest.EvidenceItems...)
	sort.Slice(items, func(i, j int) bool { return items[i].ObjectKey < items[j].ObjectKey })

	zw := zip.NewWriter(writer)
	defer func() {
		if err := zw.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("close archive: %w", err)
		}
	}()

	if err := writeArchiveEntry(zw, "manifest.json", bytes.NewReader(manifestJSON)); err != nil {
		return err
	}
	for _, item := range items {
		if err := packObject(zw, item, getObject); err != nil {
			return err
		}
	}
	return nil
}

func writeArchiveEntry(zw *zip.Writer, name string, body io.Reader) error {
	hdr := &zip.FileHeader{Name: name, Method: zip.Store}
	hdr.SetModTime(archiveEntryTime)
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return fmt.Errorf("create %s: %w", name, err)
	}
	if _, err := io.Copy(w, body); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

func packObject(zw *zip.Writer, item EvidenceItem, getObject func(key string) (io.ReadCloser, error)) error {
	return func() error {
		rc, err := getObject(item.ObjectKey)
		if err != nil {
			return fmt.Errorf("fetch %s: %w", item.ObjectKey, err)
		}
		defer rc.Close()

		hdr := &zip.FileHeader{Name: item.ObjectKey, Method: zip.Store}
		hdr.SetModTime(archiveEntryTime)
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return fmt.Errorf("create %s: %w", item.ObjectKey, err)
		}
		sum := sha256.New()
		if _, err := io.Copy(io.MultiWriter(w, sum), rc); err != nil {
			return fmt.Errorf("stream %s: %w", item.ObjectKey, err)
		}
		if got := hex.EncodeToString(sum.Sum(nil)); got != item.CiphertextSHA256 {
			return fmt.Errorf("evidence %q sha256 = %s, want %s", item.ObjectKey, got, item.CiphertextSHA256)
		}
		return nil
	}()
}
