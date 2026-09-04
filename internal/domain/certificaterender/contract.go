package certificaterender

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
	"unicode"

	"integin/internal/domain/certificatetemplate"
)

// ArtifactType represents allowed certificate artifact types in PostgreSQL.
const (
	ArtifactTypePDF = "CERTIFICATE_PDF"
	ContentTypePDF  = "application/pdf"
	RendererVersion = "integin-weasyprint-69.0-pango-harfbuzz-v1"

	// Standard ISO 216 A4 dimensions in points (72 points per inch)
	A4WidthPoints  = 595.28
	A4HeightPoints = 841.89

	// Target ISO conformance profiles
	PDFStandardCore     = "ISO-32000-2" // PDF 2.0
	PDFStandardArchival = "ISO-19005-4" // PDF/A-4b archival conformance

	MaxCellCount = 250
	MaxJobBytes  = 10 * 1024 * 1024 // 10 MiB limit on input payload
)

// JobInput is the immutable, sealed input passed to the non-authoritative renderer.
type JobInput struct {
	TenantID             string                         `json:"tenant_id"`
	OrganizationID       string                         `json:"organization_id"`
	CertificateID        string                         `json:"certificate_id"`
	CertificateNumber    string                         `json:"certificate_number"`
	Status               string                         `json:"status"`
	IssuedAt             time.Time                      `json:"issued_at"`
	ExpiresAt            time.Time                      `json:"expires_at"`
	PublicTokenDigestHex string                         `json:"public_token_digest_hex"`
	VerificationURL      string                         `json:"verification_url"`
	TemplateDef          certificatetemplate.Definition `json:"template_def"`
	TemplateCells        []certificatetemplate.Cell     `json:"template_cells"`
	PublicBindings       map[string]any                 `json:"public_bindings"`
	SnapshotSHA256       []byte                         `json:"snapshot_sha256"`
}

// JobOutput contains the output of the rendering process.
type JobOutput struct {
	ArtifactType    string    `json:"artifact_type"`
	ContentType     string    `json:"content_type"`
	ByteSize        int64     `json:"byte_size"`
	ArtifactSHA256  []byte    `json:"artifact_sha256"`
	RendererVersion string    `json:"renderer_version"`
	ObjectKey       string    `json:"object_key"`
	RenderedAt      time.Time `json:"rendered_at"`
	PDFData         []byte    `json:"-"`
}

// ValidateJobInput enforces input restrictions, geometry bounds, and security preflight.
func ValidateJobInput(input JobInput) error {
	if strings.TrimSpace(input.TenantID) == "" {
		return errors.New("tenant_id is required")
	}
	if strings.TrimSpace(input.OrganizationID) == "" {
		return errors.New("organization_id is required")
	}
	if strings.TrimSpace(input.CertificateID) == "" {
		return errors.New("certificate_id is required")
	}
	if strings.TrimSpace(input.CertificateNumber) == "" {
		return errors.New("certificate_number is required")
	}
	if len(input.SnapshotSHA256) != 32 {
		return errors.New("snapshot_sha256 must be exactly 32 bytes")
	}
	if len(input.TemplateCells) > MaxCellCount {
		return fmt.Errorf("cell count %d exceeds maximum limit %d", len(input.TemplateCells), MaxCellCount)
	}

	pageWidth := input.TemplateDef.PageWidth
	if pageWidth <= 0 {
		pageWidth = A4WidthPoints
	}
	pageHeight := input.TemplateDef.PageHeight
	if pageHeight <= 0 {
		pageHeight = A4HeightPoints
	}

	// Ensure cells are within page geometry bounds and contain no forbidden patterns
	for _, cell := range input.TemplateCells {
		if cell.PageNumber < 1 {
			return fmt.Errorf("cell %q invalid page number %d", cell.ID, cell.PageNumber)
		}
		if err := cell.Rectangle.Validate(pageWidth, pageHeight); err != nil {
			return fmt.Errorf("cell %q exceeds page bounds: %w", cell.ID, err)
		}
		if containsForbiddenContent(cell.StaticText) || containsForbiddenContent(cell.Label) {
			return fmt.Errorf("cell %q contains forbidden markup or script content", cell.ID)
		}
	}

	// Preflight bindings
	for key, val := range input.PublicBindings {
		if s, ok := val.(string); ok {
			if containsForbiddenContent(s) {
				return fmt.Errorf("binding %q contains forbidden content", key)
			}
		}
	}

	return nil
}

var forbiddenPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<script\b`),
	regexp.MustCompile(`(?i)<iframe\b`),
	regexp.MustCompile(`(?i)<embed\b`),
	regexp.MustCompile(`(?i)<object\b`),
	regexp.MustCompile(`(?i)javascript:`),
	regexp.MustCompile(`(?i)file://`),
	regexp.MustCompile(`(?i)http://`),
	regexp.MustCompile(`(?i)https://`),
	regexp.MustCompile(`(?i)data:text/html`),
}

func containsForbiddenContent(s string) bool {
	for _, pat := range forbiddenPatterns {
		if pat.MatchString(s) {
			return true
		}
	}
	return false
}

// HasArabic detects whether text contains Arabic unicode codepoints.
func HasArabic(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Arabic) {
			return true
		}
	}
	return false
}

// BidiFormatText wraps LTR technical tokens, numbers, and Latin phrases inside
// <span dir="ltr">...</span> when rendered in an RTL context, and escapes HTML characters.
func BidiFormatText(text string) string {
	escaped := html.EscapeString(text)
	if !HasArabic(text) {
		return escaped
	}

	// For predominantly Arabic text, identify embedded Latin/alphanumeric/technical tokens
	// e.g. "WO-2026-0012", "ISO-9001", "100%", "SN: 84920"
	tokenRe := regexp.MustCompile(`\b([A-Za-z0-9][A-Za-z0-9_\-\.\:\/]*)\b`)
	result := tokenRe.ReplaceAllStringFunc(escaped, func(match string) string {
		return fmt.Sprintf(`<span dir="ltr" class="ltr-isolate">%s</span>`, match)
	})
	return result
}

// ComputeObjectKey derives the canonical tenant-isolated storage key.
func ComputeObjectKey(tenantID, certificateID string) string {
	return fmt.Sprintf("tenants/%s/certs/%s/certificate.pdf", tenantID, certificateID)
}

// ValidatePDFSecurity inspects generated PDF byte stream for active/unsafe PDF dictionary elements.
// Conforms to ISO 19005 (PDF/A) security and integrity constraints.
func ValidatePDFSecurity(data []byte) error {
	if len(data) < 10 {
		return errors.New("pdf data is empty or truncated")
	}
	if !strings.HasPrefix(string(data[:5]), "%PDF-") {
		return errors.New("invalid pdf signature header")
	}

	raw := string(data)
	forbiddenPDFTokens := []string{
		"/JavaScript",
		"/JS",
		"/Launch",
		"/OpenAction",
		"/AA", // Additional Actions
		"/RichMedia",
		"/SubmitForm",
		"/ImportData",
		"/GoToR", // Remote GoTo
	}
	for _, token := range forbiddenPDFTokens {
		if strings.Contains(raw, token) {
			return fmt.Errorf("pdf contains forbidden interactive token %q", token)
		}
	}
	return nil
}

// CompileHTML produces deterministic, sanitized HTML/CSS paged media representation.
func CompileHTML(input JobInput) (string, error) {
	if err := ValidateJobInput(input); err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html lang="ar" dir="rtl">
<head>
<meta charset="utf-8">
<style>
@page {
  size: A4 portrait;
  margin: 0;
}
* {
  box-sizing: border-box;
}
body {
  margin: 0;
  padding: 0;
  font-family: 'Noto Sans Arabic', 'Noto Sans', sans-serif;
  color: #111827;
  direction: rtl;
  background-color: #ffffff;
}
.page {
  position: relative;
  width: 595.28pt;
  height: 841.89pt;
  page-break-after: always;
  overflow: hidden;
}
.page:last-child {
  page-break-after: avoid;
}
.cell {
  position: absolute;
  overflow: hidden;
  font-size: 10pt;
  line-height: 1.3;
}
.cell-border {
  border: 0.5pt solid #d1d5db;
}
.cell-label {
  font-size: 7pt;
  color: #6b7280;
  margin-bottom: 2pt;
  text-transform: uppercase;
}
.cell-value {
  font-weight: 500;
}
.ltr-isolate {
  direction: ltr;
  unicode-bidi: isolate;
  display: inline-block;
}
.qr-container {
  position: absolute;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
.qr-box {
  border: 1pt solid #374151;
  padding: 4pt;
  background: #ffffff;
}
.qr-caption {
  font-size: 6pt;
  color: #4b5563;
  margin-top: 2pt;
  text-align: center;
}
.header-bar {
  background: #1e3a8a;
  color: #ffffff;
  padding: 16pt 24pt;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.title-ar {
  font-size: 16pt;
  font-weight: bold;
}
.title-en {
  font-size: 12pt;
  color: #bfdbfe;
}
</style>
</head>
<body>
`)

	pageCount := input.TemplateDef.PageCount
	if pageCount <= 0 {
		pageCount = 1
	}

	for page := 1; page <= pageCount; page++ {
		sb.WriteString(fmt.Sprintf(`<div class="page" id="page-%d">`+"\n", page))

		// Render Header if Page 1
		if page == 1 {
			sb.WriteString(`  <div class="header-bar">
    <div>
      <div class="title-ar">شهادة فحص واختبار معتمدة</div>
      <div class="title-en" dir="ltr">Inspection &amp; Test Certificate</div>
    </div>
    <div style="text-align: left;" dir="ltr">
      <div style="font-size: 11pt; font-weight: bold;">` + html.EscapeString(input.CertificateNumber) + `</div>
      <div style="font-size: 8pt; color: #93c5fd;">` + input.IssuedAt.Format("2006-01-02") + `</div>
    </div>
  </div>` + "\n")
		}

		// Render Cells for this page
		for _, cell := range input.TemplateCells {
			if cell.PageNumber != page {
				continue
			}

			// Resolve cell value
			var textVal string
			if cell.StaticText != "" {
				textVal = cell.StaticText
			} else if cell.BindingKey != "" {
				if v, ok := input.PublicBindings[string(cell.BindingKey)]; ok && v != nil {
					textVal = fmt.Sprintf("%v", v)
				}
			}

			formattedText := BidiFormatText(textVal)
			formattedLabel := BidiFormatText(cell.Label)

			style := fmt.Sprintf("left: %.2fpt; top: %.2fpt; width: %.2fpt; height: %.2fpt;",
				cell.Rectangle.X, cell.Rectangle.Y, cell.Rectangle.Width, cell.Rectangle.Height)

			sb.WriteString(fmt.Sprintf(`  <div class="cell cell-border" style="%s">`+"\n", style))
			if formattedLabel != "" {
				sb.WriteString(fmt.Sprintf(`    <div class="cell-label">%s</div>`+"\n", formattedLabel))
			}
			sb.WriteString(fmt.Sprintf(`    <div class="cell-value">%s</div>`+"\n", formattedText))
			sb.WriteString(`  </div>` + "\n")
		}

		// Render QR Code placeholder container on page 1
		if page == 1 {
			sb.WriteString(fmt.Sprintf(`  <div class="qr-container" style="left: 460pt; top: 700pt; width: 100pt; height: 110pt;">
    <div class="qr-box">
      <div style="width: 80pt; height: 80pt; background: #f3f4f6; display: flex; align-items: center; justify-content: center; font-size: 7pt; color: #374151;" dir="ltr">
        [QR: %s]
      </div>
    </div>
    <div class="qr-caption" dir="ltr">SCAN TO VERIFY</div>
  </div>`+"\n", html.EscapeString(input.CertificateNumber)))
		}

		sb.WriteString(`</div>` + "\n")
	}

	sb.WriteString(`</body>
</html>`)
	return sb.String(), nil
}

// ComputeDigest computes SHA-256 digest of byte slice.
func ComputeDigest(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}
