package certificaterender

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/certificatetemplate"
)

func sampleJobInput() JobInput {
	snapDigest := sha256.Sum256([]byte("test-snapshot"))
	return JobInput{
		TenantID:             "tenant-1",
		OrganizationID:       "org-1",
		CertificateID:        "cert-12345",
		CertificateNumber:    "CERT-2026-0001",
		Status:               "ISSUED",
		IssuedAt:             time.Date(2026, 9, 4, 10, 0, 0, 0, time.UTC),
		ExpiresAt:            time.Date(2027, 9, 4, 10, 0, 0, 0, time.UTC),
		PublicTokenDigestHex: "abcdef123456",
		VerificationURL:      "https://example.com/verify/cert",
		SnapshotSHA256:       snapDigest[:],
		TemplateDef: certificatetemplate.Definition{
			ID:           "tpl-001",
			TemplateCode: "ARABIC_EN_INSPECTION",
			Version:      1,
			PageCount:    1,
			PageWidth:    A4WidthPoints,
			PageHeight:   A4HeightPoints,
		},
		TemplateCells: []certificatetemplate.Cell{
			{
				ID:         "cell-1",
				PageNumber: 1,
				Rectangle:  certificatetemplate.Rectangle{X: 20, Y: 80, Width: 200, Height: 30},
				Kind:       certificatetemplate.TextCell,
				Label:      "رقم الأصل / Asset ID",
				StaticText: "AST-8492",
			},
			{
				ID:         "cell-2",
				PageNumber: 1,
				Rectangle:  certificatetemplate.Rectangle{X: 230, Y: 80, Width: 300, Height: 30},
				Kind:       certificatetemplate.TextCell,
				Label:      "الوصف / Description",
				StaticText: "صمام أمان عالي الضغط - Pressure Relief Valve PRV-102",
			},
		},
		PublicBindings: map[string]any{
			"asset.id": "AST-8492",
		},
	}
}

func TestDeterministicPDFRenderer(t *testing.T) {
	renderer := NewDeterministicPDFRenderer()
	input := sampleJobInput()

	output1, err := renderer.Render(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	if output1.ArtifactType != ArtifactTypePDF {
		t.Fatalf("unexpected artifact type: %s", output1.ArtifactType)
	}
	if output1.ContentType != ContentTypePDF {
		t.Fatalf("unexpected content type: %s", output1.ContentType)
	}
	if len(output1.PDFData) == 0 || output1.ByteSize != int64(len(output1.PDFData)) {
		t.Fatalf("mismatched byte size: output=%d, actual=%d", output1.ByteSize, len(output1.PDFData))
	}
	if len(output1.ArtifactSHA256) != 32 {
		t.Fatalf("expected 32-byte sha256 digest, got %d", len(output1.ArtifactSHA256))
	}
	if output1.ObjectKey != "tenants/tenant-1/certs/cert-12345/certificate.pdf" {
		t.Fatalf("unexpected object key: %s", output1.ObjectKey)
	}

	// Byte determinism test: repeated render of identical input produces identical hash
	output2, err := renderer.Render(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected second render error: %v", err)
	}
	if string(output1.ArtifactSHA256) != string(output2.ArtifactSHA256) {
		// non-security use: "sha" prefix is only a log label; the digests compared are ArtifactSHA256.
		t.Fatalf("renders are not byte deterministic: sha1=%x sha2=%x", output1.ArtifactSHA256, output2.ArtifactSHA256)
	}
}

func TestBuildDeterministicPDFWritesVisibleText(t *testing.T) {
	raw, err := BuildDeterministicPDF("LIFT PLAN", [][]string{{"GROUND PRESSURE", "PASS"}})
	if err != nil {
		t.Fatal(err)
	}
	pdf := string(raw)
	for _, want := range []string{"/Font << /F1", "BT", "/F1 12 Tf", "72 760 Td", "(GROUND PRESSURE) Tj", "ET"} {
		if !strings.Contains(pdf, want) {
			t.Fatalf("PDF missing visible text operator %q", want)
		}
	}
	if strings.Contains(pdf, "% GROUND PRESSURE") {
		t.Fatal("visible text must not be encoded as a PDF comment")
	}
}

func TestRendererInputValidation(t *testing.T) {
	renderer := NewDeterministicPDFRenderer()

	// Missing tenant ID
	badInput := sampleJobInput()
	badInput.TenantID = ""
	if _, err := renderer.Render(context.Background(), badInput); err == nil {
		t.Fatal("expected error on missing tenant_id")
	}

	// Invalid snapshot sha256 length
	badInput2 := sampleJobInput()
	badInput2.SnapshotSHA256 = []byte("short")
	if _, err := renderer.Render(context.Background(), badInput2); err == nil {
		t.Fatal("expected error on invalid snapshot_sha256")
	}

	// Forbidden script content
	badInput3 := sampleJobInput()
	badInput3.TemplateCells[0].StaticText = "<script>alert('xss')</script>"
	if _, err := renderer.Render(context.Background(), badInput3); err == nil {
		t.Fatal("expected error on forbidden script tag")
	}

	// Geometry overflow
	badInput4 := sampleJobInput()
	badInput4.TemplateCells[0].Rectangle = certificatetemplate.Rectangle{X: 500, Y: 800, Width: 200, Height: 100}
	if _, err := renderer.Render(context.Background(), badInput4); err == nil {
		t.Fatal("expected error on cell exceeding page bounds")
	}
}

func TestBidiFormatText(t *testing.T) {
	// Arabic text with embedded English/technical token
	input := "صمام أمان PRV-901 تم فحصه بنسبة 100%"
	formatted := BidiFormatText(input)

	if !strings.Contains(formatted, `<span dir="ltr" class="ltr-isolate">PRV-901</span>`) {
		t.Fatalf("expected PRV-901 wrapped in ltr span, got: %s", formatted)
	}
	if !strings.Contains(formatted, `<span dir="ltr" class="ltr-isolate">100</span>`) {
		t.Fatalf("expected 100 wrapped in ltr span, got: %s", formatted)
	}
}

func TestValidatePDFSecurity(t *testing.T) {
	cleanPDF := []byte("%PDF-1.7\n1 0 obj\n<< >>\nendobj\n%%EOF")
	if err := ValidatePDFSecurity(cleanPDF); err != nil {
		t.Fatalf("expected clean pdf to pass validation: %v", err)
	}

	maliciousPDF := []byte("%PDF-1.7\n/JavaScript (app.alert(1))\n%%EOF")
	if err := ValidatePDFSecurity(maliciousPDF); err == nil {
		t.Fatal("expected malicious pdf with /JavaScript to fail validation")
	}
}

func TestWorkerProtocolContract(t *testing.T) {
	// Verify that JobInput produces valid JSON and compiled HTML accepted by worker.py schema
	input := sampleJobInput()
	htmlDoc, err := CompileHTML(input)
	if err != nil {
		t.Fatalf("failed to compile html: %v", err)
	}

	payload := map[string]any{
		"tenant_id":          input.TenantID,
		"certificate_id":     input.CertificateID,
		"certificate_number": input.CertificateNumber,
		"snapshot_sha256":    input.SnapshotSHA256,
		"compiled_html":      htmlDoc,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal worker payload: %v", err)
	}

	if len(raw) > MaxJobBytes {
		t.Fatalf("worker payload %d exceeds MaxJobBytes %d", len(raw), MaxJobBytes)
	}

	// Verify that forbidden patterns are caught before dispatching to worker
	payload["compiled_html"] = "<div><script>alert(1)</script></div>"
	if containsForbiddenContent(payload["compiled_html"].(string)) != true {
		t.Fatal("expected forbidden script to be flagged by containsForbiddenContent")
	}
}
