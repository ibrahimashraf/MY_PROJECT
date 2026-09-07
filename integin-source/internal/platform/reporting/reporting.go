package reporting

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"strings"
)

type CSVReport struct {
	Headers []string
	Rows    [][]string
}

func (r CSVReport) RenderCSV() ([]byte, error) {
	if len(r.Headers) == 0 {
		return nil, errors.New("report headers are required")
	}
	for _, row := range r.Rows {
		if len(row) != len(r.Headers) {
			return nil, errors.New("report row width does not match headers")
		}
	}
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write(r.Headers); err != nil {
		return nil, err
	}
	for _, row := range r.Rows {
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

type CertificateData struct {
	TenantID          string
	CertificateNumber string
	AssetName         string
	SerialNumber      string
	Result            string
	StandardReference string
}
type PDFRenderer func(CertificateData) ([]byte, error)

func RenderCertificatePDF(data CertificateData, renderer PDFRenderer) ([]byte, error) {
	if strings.TrimSpace(data.TenantID) == "" || strings.TrimSpace(data.CertificateNumber) == "" {
		return nil, errors.New("tenant id and certificate number are required")
	}
	if renderer == nil {
		return nil, errors.New("pdf renderer is required")
	}
	output, err := renderer(data)
	if err != nil {
		return nil, fmt.Errorf("render certificate: %w", err)
	}
	if len(output) == 0 {
		return nil, errors.New("pdf renderer returned empty document")
	}
	return output, nil
}
