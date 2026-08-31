package certificaterender

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"integin/internal/storage"
)

// WordRequest derives a .docx-like artifact from the same cell_snapshot that
// produced the PDF Hash. The PDF remains primary (artifact_sha256); Word is
// secondary (artifact_sha256_word) so editing Word never invalidates PDF.
// Uses only fixed-cell data; no new GRANT, no RLS change.
// For now rendered as minimal HTML→PDF-compatible bytes with .docx extension
// so storage/x-amz-meta-* and Hash chain stay verifiable without adding a
// heavy docx library. Caller can set FeatureFlagWordDerived to gate serving.
type WordRequest struct {
	CertificateID  string
	CertificateNo  string
	SnapshotSHA256 []byte // same as PDF
	TableRows      [][]string
	TableTitle     string
}

type WordResult struct {
	Bytes     []byte
	SHA256    string
	ObjectKey string
	Metadata  map[string]string
}

func RenderWord(req WordRequest, pdfSHA256 string) (WordResult, error) {
	if strings.TrimSpace(req.CertificateID) == "" || strings.TrimSpace(req.CertificateNo) == "" || len(req.SnapshotSHA256) != sha256.Size {
		return WordResult{}, errors.New("certificate id, number, snapshot digest are required")
	}
	if strings.TrimSpace(pdfSHA256) == "" || len(pdfSHA256) != 64 {
		return WordResult{}, errors.New("pdf artifact sha256 is required for Word derivation")
	}
	var buf bytes.Buffer
	buf.WriteString("<html><head><meta charset=\"utf-8\"/><title>")
	buf.WriteString(safeKeyPart(req.CertificateNo))
	buf.WriteString("</title></head><body><h2>")
	title := strings.TrimSpace(req.TableTitle)
	if title == "" {
		title = "Inspection Detail — Derived Word Artifact"
	}
	buf.WriteString(title)
	buf.WriteString("</h2><p>Derived from snapshot ")
	buf.WriteString(hex.EncodeToString(req.SnapshotSHA256))
	buf.WriteString(" / PDF ")
	buf.WriteString(pdfSHA256)
	buf.WriteString("</p><table border=\"1\" cellpadding=\"4\" cellspacing=\"0\">")
	for _, row := range req.TableRows {
		buf.WriteString("<tr>")
		for _, col := range row {
			buf.WriteString("<td>")
			buf.WriteString(col)
			buf.WriteString("</td>")
		}
		buf.WriteString("</tr>")
	}
	buf.WriteString("</table></body></html>")
	bs := buf.Bytes()
	digest := sha256.Sum256(bs)
	objectKey := "certificates/" + safeKeyPart(req.CertificateID) + "/" + safeKeyPart(req.CertificateNo) + ".derived.html"
	return WordResult{
		Bytes:     bs,
		SHA256:    hex.EncodeToString(digest[:]),
		ObjectKey: objectKey,
		Metadata: map[string]string{
			"certificate_id":       req.CertificateID,
			"certificate_number":   req.CertificateNo,
			"snapshot_sha256":      hex.EncodeToString(req.SnapshotSHA256),
			"artifact_sha256_pdf":  pdfSHA256,
			"artifact_sha256_word": hex.EncodeToString(digest[:]),
			"renderer_version":     RendererVersion + "+word-derived",
		},
	}, nil
}

func StoreWordArtifact(ctx context.Context, store storage.Store, result WordResult) error {
	if store == nil || len(result.Bytes) == 0 || strings.TrimSpace(result.ObjectKey) == "" || strings.TrimSpace(result.SHA256) == "" {
		return errors.New("storage, rendered Word, object key, and artifact digest are required")
	}
	return store.Put(ctx, storage.Object{Key: result.ObjectKey, ContentType: "text/html", Data: result.Bytes, Metadata: result.Metadata})
}
