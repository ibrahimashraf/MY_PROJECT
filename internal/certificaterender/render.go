package certificaterender

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	qrcode "github.com/skip2/go-qrcode"
	"integin/internal/domain/certificatetemplate"
	"integin/internal/storage"
)

const RendererVersion = "v1"

type Request struct {
	Template        certificatetemplate.Definition
	Catalog         certificatetemplate.Catalog
	Values          map[certificatetemplate.BindingKey]string
	CertificateID   string
	CertificateNo   string
	SnapshotSHA256  []byte
	IssuedAt        time.Time
	VerifierBaseURL string
	PublicToken     string
	QRPage          int
	QRRectangle     certificatetemplate.Rectangle
	// Safe additive appendix for Bswagic 270s/300s table pattern (tables/images inside Hash, not Word):
	// If set, rendered as deterministic extra page(s) after the template pages so existing
	// fixed-cell Hash stays stable when empty, and Hash includes tables when present.
	TableRows [][]string
	TableTitle string
}

type Result struct {
	PDF       []byte
	PDFSHA256 string
	QRPNG     []byte
	QRURL     string
	ObjectKey string
	Metadata  map[string]string
}

func StoreArtifact(ctx context.Context, store storage.Store, result Result) error {
	if store == nil || len(result.PDF) == 0 || strings.TrimSpace(result.ObjectKey) == "" || strings.TrimSpace(result.PDFSHA256) == "" {
		return errors.New("storage, rendered PDF, object key, and artifact digest are required")
	}
	if _, ok := result.Metadata["public_token"]; ok {
		return errors.New("artifact metadata must not contain a public token")
	}
	return store.Put(ctx, storage.Object{Key: result.ObjectKey, ContentType: "application/pdf", Data: result.PDF, Metadata: result.Metadata})
}

func Render(request Request) (Result, error) {
	if err := request.Template.Validate(request.Catalog); err != nil {
		return Result{}, fmt.Errorf("template: %w", err)
	}
	if strings.TrimSpace(request.CertificateID) == "" || strings.TrimSpace(request.CertificateNo) == "" || len(request.SnapshotSHA256) != sha256.Size || request.IssuedAt.IsZero() {
		return Result{}, errors.New("certificate id, number, snapshot digest, and issuance time are required")
	}
	if request.QRPage <= 0 || request.QRPage > request.Template.PageCount {
		return Result{}, errors.New("QR page is outside the template")
	}
	if err := request.QRRectangle.Validate(request.Template.PageWidth, request.Template.PageHeight); err != nil {
		return Result{}, fmt.Errorf("QR rectangle: %w", err)
	}
	qrURL, err := verifierURL(request.VerifierBaseURL, request.PublicToken)
	if err != nil {
		return Result{}, err
	}
	qrPNG, err := qrcode.Encode(qrURL, qrcode.High, 512)
	if err != nil {
		return Result{}, fmt.Errorf("QR encode: %w", err)
	}
	pdf := fpdf.NewCustom(&fpdf.InitType{OrientationStr: "P", UnitStr: "pt", Size: fpdf.SizeType{Wd: request.Template.PageWidth, Ht: request.Template.PageHeight}})
	pdf.SetCreationDate(request.IssuedAt.UTC())
	pdf.SetCatalogSort(true)
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	for page := 1; page <= request.Template.PageCount; page++ {
		pdf.AddPage()
		for _, cell := range request.Template.SortedCells() {
			if cell.PageNumber != page {
				continue
			}
			if err := renderCell(pdf, cell, request.Values); err != nil {
				return Result{}, err
			}
		}
		if page == request.QRPage {
			options := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
			pdf.RegisterImageOptionsReader("certificate-qr", options, bytes.NewReader(qrPNG))
			pdf.ImageOptions("certificate-qr", request.QRRectangle.X, request.QRRectangle.Y, request.QRRectangle.Width, request.QRRectangle.Height, false, options, 0, "")
		}
	}
	if len(request.TableRows) > 0 {
		if err := renderTableAppendix(pdf, request); err != nil {
			return Result{}, err
		}
	}
	if pdf.Err() {
		return Result{}, fmt.Errorf("PDF render: %w", pdf.Error())
	}
	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return Result{}, fmt.Errorf("PDF output: %w", err)
	}
	pdfBytes := output.Bytes()
	digest := sha256.Sum256(pdfBytes)
	objectKey := "certificates/" + safeKeyPart(request.CertificateID) + "/" + safeKeyPart(request.CertificateNo) + ".pdf"
	return Result{PDF: pdfBytes, PDFSHA256: hex.EncodeToString(digest[:]), QRPNG: qrPNG, QRURL: qrURL, ObjectKey: objectKey, Metadata: map[string]string{"certificate_id": request.CertificateID, "certificate_number": request.CertificateNo, "snapshot_sha256": hex.EncodeToString(request.SnapshotSHA256), "artifact_sha256": hex.EncodeToString(digest[:]), "renderer_version": RendererVersion, "content_disposition_name": safeKeyPart(request.CertificateNo) + ".pdf"}}, nil
}

func renderCell(pdf *fpdf.Fpdf, cell certificatetemplate.Cell, values map[certificatetemplate.BindingKey]string) error {
	text := cell.StaticText
	if cell.Kind != certificatetemplate.StaticTextCell {
		text = values[cell.BindingKey]
		if cell.Required && strings.TrimSpace(text) == "" {
			return fmt.Errorf("cell %s requires %s", cell.ID, cell.BindingKey)
		}
	}
	if cell.Kind == certificatetemplate.CheckboxCell {
		mapped, ok := cell.CheckboxValues[text]
		if !ok && strings.TrimSpace(text) != "" {
			return fmt.Errorf("cell %s has no checkbox mapping for %q", cell.ID, text)
		}
		text = mapped
	}
	pdf.SetFont("Helvetica", "", 9)
	lineHeight := cell.Rectangle.Height / float64(cell.MaxLines)
	if lineHeight <= 0 {
		return fmt.Errorf("cell %s has invalid line height", cell.ID)
	}
	lines := pdf.SplitText(text, cell.Rectangle.Width)
	if len(lines) == 0 && text == "" {
		lines = []string{""}
	}
	if cell.FitPolicy == certificatetemplate.SingleLineRequired && len(lines) != 1 {
		return fmt.Errorf("cell %s exceeds single-line policy", cell.ID)
	}
	if len(lines) > cell.MaxLines {
		return fmt.Errorf("cell %s exceeds maximum lines", cell.ID)
	}
	pdf.SetXY(cell.Rectangle.X, cell.Rectangle.Y)
	pdf.MultiCell(cell.Rectangle.Width, lineHeight, text, "", "L", false)
	return nil
}

func renderTableAppendix(pdf *fpdf.Fpdf, request Request) error {
	pdf.AddPage()
	title := strings.TrimSpace(request.TableTitle)
	if title == "" {
		title = "Inspection Detail — Evidence Table"
	}
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetXY(20, 20)
	pdf.CellFormat(request.Template.PageWidth-40, 14, title, "1", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 8)
	y := 36.0
	colW := (request.Template.PageWidth - 40) / float64(len(request.TableRows[0]))
	rowH := 10.0
	for _, row := range request.TableRows {
		if y+rowH > request.Template.PageHeight-20 {
			pdf.AddPage()
			y = 20
		}
		x := 20.0
		for _, col := range row {
			pdf.SetXY(x, y)
			// Fixed-cell: each cell has deterministic size; Hash stable.
			pdf.Rect(x, y, colW, rowH, "D")
			pdf.SetXY(x+2, y+2)
			pdf.CellFormat(colW-4, rowH-4, col, "", 0, "L", false, 0, "")
			x += colW
		}
		y += rowH
	}
	return nil
}

func verifierURL(baseURL, token string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || strings.TrimSpace(token) == "" {
		return "", errors.New("HTTPS verifier base URL and token are required")
	}
	if strings.Contains(token, "/") || strings.Contains(token, "?") || len(token) < 32 || len(token) > 128 {
		return "", errors.New("invalid public token for QR URL")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/verify/certificates/" + token
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func safeKeyPart(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, value)
	return strings.Trim(value, "-")
}
