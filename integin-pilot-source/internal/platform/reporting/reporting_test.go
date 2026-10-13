package reporting

import (
	"strings"
	"testing"
)

func TestCSVReportAndCertificateRenderer(t *testing.T) {
	report, err := (CSVReport{Headers: []string{"asset", "result"}, Rows: [][]string{{"crane-1", "PASS"}}}).RenderCSV()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(report), "crane-1,PASS") {
		t.Fatalf("unexpected csv: %s", report)
	}
	pdf, err := RenderCertificatePDF(CertificateData{TenantID: "tenant-1", CertificateNumber: "CERT-1"}, func(data CertificateData) ([]byte, error) { return []byte("%PDF-test"), nil })
	if err != nil || string(pdf) != "%PDF-test" {
		t.Fatalf("unexpected pdf: %s %v", pdf, err)
	}
}
