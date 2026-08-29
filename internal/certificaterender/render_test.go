package certificaterender

import (
	"context"
	"crypto/sha256"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/certificatetemplate"
	"integin/internal/storage"
)

func TestRenderProducesFixedCellPDFAndBoundedQR(t *testing.T) {
	template := testTemplate()
	digest := sha256.Sum256([]byte("snapshot"))
	result, err := Render(Request{Template: template, Catalog: certificatetemplate.DefaultCatalog(), Values: map[certificatetemplate.BindingKey]string{certificatetemplate.InspectionAssetID: "asset-a"}, CertificateID: "certificate-a", CertificateNo: "CERT-00000001", SnapshotSHA256: digest[:], IssuedAt: time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC), VerifierBaseURL: "https://verify.example.test", PublicToken: strings.Repeat("a", 43), QRPage: 1, QRRectangle: certificatetemplate.Rectangle{X: 140, Y: 20, Width: 40, Height: 40}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(result.PDF), "%PDF") || len(result.QRPNG) == 0 || !strings.Contains(result.QRURL, "/verify/certificates/") || result.Metadata["artifact_sha256"] != result.PDFSHA256 {
		t.Fatalf("unexpected renderer result: %#v", result)
	}
	store := storage.NewInMemoryStore()
	if err := StoreArtifact(context.Background(), store, result); err != nil {
		t.Fatal(err)
	}
	stored, err := store.Get(context.Background(), result.ObjectKey)
	if err != nil || !strings.HasPrefix(string(stored.Data), "%PDF") || stored.Metadata["artifact_sha256"] != result.PDFSHA256 {
		t.Fatalf("unexpected stored artifact value=%#v err=%v", stored, err)
	}
	if strings.Contains(strings.Join(metadataValues(stored.Metadata), "|"), "https://") {
		t.Fatalf("storage metadata leaked QR URL: %#v", stored.Metadata)
	}
}

func TestRenderRejectsCellOverflowAndInsecureVerifierURL(t *testing.T) {
	template := testTemplate()
	digest := sha256.Sum256([]byte("snapshot"))
	_, err := Render(Request{Template: template, Catalog: certificatetemplate.DefaultCatalog(), Values: map[certificatetemplate.BindingKey]string{certificatetemplate.InspectionAssetID: strings.Repeat("overflow ", 100)}, CertificateID: "certificate-a", CertificateNo: "CERT-00000001", SnapshotSHA256: digest[:], IssuedAt: time.Now().UTC(), VerifierBaseURL: "https://verify.example.test", PublicToken: strings.Repeat("a", 43), QRPage: 1, QRRectangle: certificatetemplate.Rectangle{X: 140, Y: 20, Width: 40, Height: 40}})
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected overflow rejection, got %v", err)
	}
	_, err = Render(Request{Template: template, Catalog: certificatetemplate.DefaultCatalog(), Values: map[certificatetemplate.BindingKey]string{certificatetemplate.InspectionAssetID: "asset-a"}, CertificateID: "certificate-a", CertificateNo: "CERT-00000001", SnapshotSHA256: digest[:], IssuedAt: time.Now().UTC(), VerifierBaseURL: "http://verify.example.test", PublicToken: strings.Repeat("a", 43), QRPage: 1, QRRectangle: certificatetemplate.Rectangle{X: 140, Y: 20, Width: 40, Height: 40}})
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("expected HTTPS URL rejection, got %v", err)
	}
}

func BenchmarkRenderFixedCellCertificate(b *testing.B) {
	template := testTemplate()
	digest := sha256.Sum256([]byte("snapshot"))
	request := Request{Template: template, Catalog: certificatetemplate.DefaultCatalog(), Values: map[certificatetemplate.BindingKey]string{certificatetemplate.InspectionAssetID: "asset-a"}, CertificateID: "certificate-a", CertificateNo: "CERT-00000001", SnapshotSHA256: digest[:], IssuedAt: time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC), VerifierBaseURL: "https://verify.example.test", PublicToken: strings.Repeat("a", 43), QRPage: 1, QRRectangle: certificatetemplate.Rectangle{X: 140, Y: 20, Width: 40, Height: 40}}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result, err := Render(request)
		if err != nil {
			b.Fatal(err)
		}
		if len(result.PDF) == 0 {
			b.Fatal("empty PDF")
		}
	}
}

func BenchmarkStoreRenderedCertificateArtifact(b *testing.B) {
	digest := sha256.Sum256([]byte("snapshot"))
	result, err := Render(Request{Template: testTemplate(), Catalog: certificatetemplate.DefaultCatalog(), Values: map[certificatetemplate.BindingKey]string{certificatetemplate.InspectionAssetID: "asset-a"}, CertificateID: "certificate-a", CertificateNo: "CERT-00000001", SnapshotSHA256: digest[:], IssuedAt: time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC), VerifierBaseURL: "https://verify.example.test", PublicToken: strings.Repeat("a", 43), QRPage: 1, QRRectangle: certificatetemplate.Rectangle{X: 140, Y: 20, Width: 40, Height: 40}})
	if err != nil {
		b.Fatal(err)
	}
	store := storage.NewInMemoryStore()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if err := StoreArtifact(context.Background(), store, result); err != nil {
			b.Fatal(err)
		}
	}
}

func testTemplate() certificatetemplate.Definition {
	return certificatetemplate.Definition{ID: "certificate_template", TemplateCode: "lifting", Version: 1, Title: "Lifting", AssetType: "lifting", Status: certificatetemplate.Approved, CatalogVersion: 1, PageCount: 1, PageWidth: 200, PageHeight: 200, Cells: []certificatetemplate.Cell{{ID: "asset_id", PageNumber: 1, Rectangle: certificatetemplate.Rectangle{X: 20, Y: 20, Width: 100, Height: 20}, Kind: certificatetemplate.TextCell, BindingKey: certificatetemplate.InspectionAssetID, FitPolicy: certificatetemplate.SingleLineRequired, MaxLines: 1, Required: true}}}
}

func metadataValues(values map[string]string) []string {
	output := make([]string, 0, len(values))
	for _, value := range values {
		output = append(output, value)
	}
	return output
}
