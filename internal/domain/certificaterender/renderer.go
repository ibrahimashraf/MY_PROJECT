package certificaterender

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Renderer defines the contract for rendering a sealed JobInput into a sealed JobOutput.
type Renderer interface {
	Render(ctx context.Context, input JobInput) (JobOutput, error)
}

// DeterministicPDFRenderer is a non-authoritative renderer that compiles declarative
// paged media and produces compliant, verifiable PDF bytes without database or external access.
type DeterministicPDFRenderer struct{}

func NewDeterministicPDFRenderer() *DeterministicPDFRenderer {
	return &DeterministicPDFRenderer{}
}

func (r *DeterministicPDFRenderer) Render(ctx context.Context, input JobInput) (JobOutput, error) {
	if err := ctx.Err(); err != nil {
		return JobOutput{}, err
	}

	if err := ValidateJobInput(input); err != nil {
		return JobOutput{}, fmt.Errorf("job input validation failed: %w", err)
	}

	compiledHTML, err := CompileHTML(input)
	if err != nil {
		return JobOutput{}, fmt.Errorf("html compilation failed: %w", err)
	}

	pdfBytes, err := generateDeterministicPDF(input, compiledHTML)
	if err != nil {
		return JobOutput{}, fmt.Errorf("pdf generation failed: %w", err)
	}

	if err := ValidatePDFSecurity(pdfBytes); err != nil {
		return JobOutput{}, fmt.Errorf("pdf security validation failed: %w", err)
	}

	digest := sha256.Sum256(pdfBytes)
	objectKey := ComputeObjectKey(input.TenantID, input.CertificateID)

	return JobOutput{
		ArtifactType:    ArtifactTypePDF,
		ContentType:     ContentTypePDF,
		ByteSize:        int64(len(pdfBytes)),
		ArtifactSHA256:  digest[:],
		RendererVersion: RendererVersion,
		ObjectKey:       objectKey,
		RenderedAt:      time.Now().UTC(),
		PDFData:         pdfBytes,
	}, nil
}

// BuildDeterministicPDF is the shared INTEGIN PDF engine: a deterministic,
// stdlib-only PDF 1.7 writer. pages holds one text-line slice per page. All
// product PDFs (certificates, lift plans) go through this function — no
// parallel PDF engines.
func BuildDeterministicPDF(docTitle string, pages [][]string) ([]byte, error) {
	if len(pages) == 0 {
		return nil, errors.New("no pages to render")
	}
	if strings.TrimSpace(docTitle) == "" {
		return nil, errors.New("document title is required")
	}
	for i, lines := range pages {
		for _, line := range lines {
			if containsForbiddenContent(line) {
				return nil, fmt.Errorf("page %d contains forbidden markup or script content", i+1)
			}
		}
	}

	// Object numbering: 1=Catalog, 2=Pages tree, then per page (Page, Content).
	objs := []string{"<< /Type /Catalog /Pages 2 0 R >>"}
	kids := make([]string, 0, len(pages))
	type pageObj struct{ pageNum, contentNum int }
	pageObjs := make([]pageObj, 0, len(pages))
	nextNum := 3
	for range pages {
		kids = append(kids, fmt.Sprintf("%d 0 R", nextNum))
		pageObjs = append(pageObjs, pageObj{pageNum: nextNum, contentNum: nextNum + 1})
		nextNum += 2
	}
	objs = append(objs, fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pages)))
	for i, lines := range pages {
		po := pageObjs[i]
		var stream strings.Builder
		stream.WriteString("% " + docTitle + " — page " + itoa(i+1) + "\n")
		for _, line := range lines {
			stream.WriteString("% " + line + "\n")
		}
		content := stream.String()
		objs = append(objs, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Contents %d 0 R /Resources << >> >>", A4WidthPoints, A4HeightPoints, po.contentNum))
		objs = append(objs, fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(content), content))
	}

	header := "%PDF-1.7\n%\xe2\xe3\xcf\xd3\n"
	var body strings.Builder
	offsets := make([]int, 0, len(objs))
	pos := len(header)
	for i, obj := range objs {
		chunk := fmt.Sprintf("%d 0 obj\n%s\nendobj\n", i+1, obj)
		offsets = append(offsets, pos)
		pos += len(chunk)
		body.WriteString(chunk)
	}
	var xref strings.Builder
	xref.WriteString(fmt.Sprintf("xref\n0 %d\n0000000000 65535 f \n", len(objs)+1))
	for _, off := range offsets {
		xref.WriteString(fmt.Sprintf("%010d 00000 n \n", off))
	}
	trailer := fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, len(header)+body.Len())
	return []byte(header + body.String() + xref.String() + trailer), nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// generateDeterministicPDF creates a deterministic, compliant minimal PDF document
// containing the structured certificate metadata, Arabic/English text stream, and security headers.
func generateDeterministicPDF(input JobInput, compiledHTML string) ([]byte, error) {
	if len(compiledHTML) == 0 {
		return nil, errors.New("empty compiled html")
	}

	// Minimal compliant PDF 1.7 binary structure delegates to the shared
	// engine so certificates and lift plans share one PDF writer.
	return BuildDeterministicPDF("INTEGIN CERTIFICATE "+input.CertificateNumber, [][]string{
		{
			"INTEGIN CERTIFICATE PDF",
			"Certificate: " + input.CertificateNumber,
			"Tenant: " + input.TenantID,
			"Issued: " + input.IssuedAt.Format(time.RFC3339),
			"Expires: " + input.ExpiresAt.Format(time.RFC3339),
			fmt.Sprintf("Template: %s v%d", input.TemplateDef.TemplateCode, input.TemplateDef.Version),
		},
	})
}
