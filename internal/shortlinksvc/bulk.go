package shortlinksvc

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"integin/internal/domain/shortlink"
)

type BulkCreateRequest struct {
	Links []BulkCreateItem `json:"links"`
}

type BulkCreateItem struct {
	TargetURL    string        `json:"target_url" binding:"required,url"`
	TTL          time.Duration `json:"ttl"`
	WebhookURL   *string       `json:"webhook_url"`
	CustomDomain *string       `json:"custom_domain"`
}

type BulkCreateResponse struct {
	Created int          `json:"created"`
	Failed  int          `json:"failed"`
	Results []BulkResult `json:"results"`
}

type BulkResult struct {
	Code      string `json:"code"`
	ShortURL  string `json:"short_url,omitempty"`
	TargetURL string `json:"target_url"`
	Error     string `json:"error,omitempty"`
}

type ExportRequest struct {
	CodePrefix    string     `json:"code_prefix"`
	CreatedAfter  *time.Time `json:"created_after"`
	CreatedBefore *time.Time `json:"created_before"`
	IncludeStats  bool       `json:"include_stats"`
	Format        string     `json:"format"` // csv, json
}

type ExportResponse struct {
	Format string `json:"format"`
	Data   string `json:"data"`
	Count  int    `json:"count"`
}

func (s *Service) BulkCreate(ctx context.Context, req BulkCreateRequest) (BulkCreateResponse, error) {
	var results []BulkResult
	created, failed := 0, 0

	for _, item := range req.Links {
		code, err := s.CreateShortLink(ctx, shortlink.CreateRequest{
			TargetURL:    item.TargetURL,
			TTL:          item.TTL,
			WebhookURL:   item.WebhookURL,
			CustomDomain: item.CustomDomain,
		})
		if err != nil {
			failed++
			results = append(results, BulkResult{
				Code:      "",
				TargetURL: item.TargetURL,
				Error:     err.Error(),
			})
			continue
		}

		if item.CustomDomain != nil && *item.CustomDomain != "" {
			results = append(results, BulkResult{
				Code:      code,
				ShortURL:  "https://" + *item.CustomDomain + "/" + code,
				TargetURL: item.TargetURL,
			})
		} else {
			results = append(results, BulkResult{
				Code:      code,
				ShortURL:  "/s/" + code,
				TargetURL: item.TargetURL,
			})
		}
	}

	return BulkCreateResponse{
		Created: created,
		Failed:  failed,
		Results: results,
	}, nil
}

func (s *Service) ExportCSV(ctx context.Context, req ExportRequest) (ExportResponse, error) {
	links, err := s.repo.List(ctx, 10000, 0) // Large limit for export
	if err != nil {
		return ExportResponse{}, err
	}

	// Filter by criteria
	var filtered []shortlink.ShortLink
	for _, l := range links {
		if req.CodePrefix != "" && !strings.HasPrefix(l.Code, req.CodePrefix) {
			continue
		}
		if req.CreatedAfter != nil && l.CreatedAt.Before(*req.CreatedAfter) {
			continue
		}
		if req.CreatedBefore != nil && l.CreatedAt.After(*req.CreatedBefore) {
			continue
		}
		filtered = append(filtered, l)
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Header
	header := []string{"code", "target_url", "created_at", "expires_at", "revoked_at", "scan_count", "webhook_url", "custom_domain"}
	if err := writer.Write(header); err != nil {
		return ExportResponse{}, err
	}

	for _, l := range filtered {
		row := []string{
			l.Code,
			l.TargetURL,
			l.CreatedAt.Format(time.RFC3339),
			formatTimePtr(l.ExpiresAt),
			formatTimePtr(l.RevokedAt),
			fmt.Sprintf("%d", l.ScanCount),
			derefString(l.WebhookURL),
			derefString(l.CustomDomain),
		}
		if err := writer.Write(row); err != nil {
			return ExportResponse{}, err
		}
	}
	writer.Flush()

	return ExportResponse{
		Format: "csv",
		Data:   buf.String(),
		Count:  len(filtered),
	}, nil
}

func (s *Service) ExportJSON(ctx context.Context, req ExportRequest) (ExportResponse, error) {
	links, err := s.repo.List(ctx, 10000, 0)
	if err != nil {
		return ExportResponse{}, err
	}

	var filtered []shortlink.ShortLink
	for _, l := range links {
		if req.CodePrefix != "" && !strings.HasPrefix(l.Code, req.CodePrefix) {
			continue
		}
		if req.CreatedAfter != nil && l.CreatedAt.Before(*req.CreatedAfter) {
			continue
		}
		if req.CreatedBefore != nil && l.CreatedAt.After(*req.CreatedBefore) {
			continue
		}
		filtered = append(filtered, l)
	}

	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return ExportResponse{}, err
	}

	return ExportResponse{
		Format: "json",
		Data:   string(data),
		Count:  len(filtered),
	}, nil
}

func (s *Service) ImportCSV(ctx context.Context, data string) (BulkCreateResponse, error) {
	reader := csv.NewReader(strings.NewReader(data))
	records, err := reader.ReadAll()
	if err != nil {
		return BulkCreateResponse{}, err
	}

	if len(records) < 2 {
		return BulkCreateResponse{}, errors.New("CSV must have header and at least one data row")
	}

	// Find column indices
	header := records[0]
	colIdx := make(map[string]int)
	for i, col := range header {
		colIdx[col] = i
	}

	required := []string{"target_url"}
	for _, col := range required {
		if _, ok := colIdx[col]; !ok {
			return BulkCreateResponse{}, fmt.Errorf("required column missing: %s", col)
		}
	}

	var items []BulkCreateItem
	for _, row := range records[1:] {
		if len(row) <= colIdx["target_url"] {
			continue
		}
		item := BulkCreateItem{
			TargetURL: row[colIdx["target_url"]],
		}

		if idx, ok := colIdx["ttl"]; ok && idx < len(row) && row[idx] != "" {
			if d, err := time.ParseDuration(row[colIdx["ttl"]]); err == nil {
				item.TTL = d
			}
		}
		if idx, ok := colIdx["webhook_url"]; ok && idx < len(row) && row[idx] != "" {
			item.WebhookURL = &row[colIdx["webhook_url"]]
		}
		if idx, ok := colIdx["custom_domain"]; ok && idx < len(row) && row[idx] != "" {
			item.CustomDomain = &row[colIdx["custom_domain"]]
		}
		items = append(items, item)
	}

	return s.BulkCreate(ctx, BulkCreateRequest{Links: items})
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
