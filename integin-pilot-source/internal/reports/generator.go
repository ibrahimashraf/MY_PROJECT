package reports

import (
	"bytes"
	"encoding/csv"
	"fmt"
)

func GenerateCSV(headers []string, rows [][]string) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if err := w.Write(headers); err != nil {
		return nil, fmt.Errorf("writing CSV header: %w", err)
	}

	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return nil, fmt.Errorf("writing CSV row: %w", err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("flushing CSV: %w", err)
	}

	return buf.Bytes(), nil
}
