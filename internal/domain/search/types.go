package search

import "time"

type EntityType string

const (
	EntityAsset      EntityType = "ASSET"
	EntityWorkOrder  EntityType = "WORK_ORDER"
	EntityInspection EntityType = "INSPECTION"
)

type Result struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	OrganizationID string     `json:"organization_id"`
	EntityType     EntityType `json:"entity_type"`
	Title          string     `json:"title"`
	Snippet        string     `json:"snippet"`
	Rank           float64    `json:"rank"`
	CreatedAt      time.Time  `json:"created_at"`
}

type SearchRequest struct {
	Query          string       `json:"query"`
	TenantID       string       `json:"tenant_id"`
	OrganizationID string       `json:"organization_id"`
	Types          []EntityType `json:"types,omitempty"`
	Limit          int          `json:"limit"`
	Offset         int          `json:"offset"`
}

func (s *SearchRequest) Normalize() {
	if s.Limit <= 0 || s.Limit > 100 {
		s.Limit = 20
	}
	if s.Offset < 0 {
		s.Offset = 0
	}
}

type SearchResponse struct {
	Results []Result `json:"results"`
	Total   int      `json:"total"`
	Query   string   `json:"query"`
}
