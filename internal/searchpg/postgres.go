package searchpg

import (
	"context"
	"database/sql"
	"fmt"

	"integin/internal/domain/search"
)

type Repository struct {
	DB *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) Search(ctx context.Context, req search.SearchRequest) (search.SearchResponse, error) {
	req.Normalize()

	var results []search.Result

	searchAsset := len(req.Types) == 0 || containsType(req.Types, search.EntityAsset)
	searchWorkOrder := len(req.Types) == 0 || containsType(req.Types, search.EntityWorkOrder)
	searchInspection := len(req.Types) == 0 || containsType(req.Types, search.EntityInspection)

	if searchAsset {
		assets, err := r.searchAssets(ctx, req.TenantID, req.OrganizationID, req.Query, req.Limit, req.Offset)
		if err != nil {
			return search.SearchResponse{}, fmt.Errorf("search assets: %w", err)
		}
		results = append(results, assets...)
	}

	if searchWorkOrder {
		orders, err := r.searchWorkOrders(ctx, req.TenantID, req.OrganizationID, req.Query, req.Limit, req.Offset)
		if err != nil {
			return search.SearchResponse{}, fmt.Errorf("search work orders: %w", err)
		}
		results = append(results, orders...)
	}

	if searchInspection {
		inspections, err := r.searchInspections(ctx, req.TenantID, req.OrganizationID, req.Query, req.Limit, req.Offset)
		if err != nil {
			return search.SearchResponse{}, fmt.Errorf("search inspections: %w", err)
		}
		results = append(results, inspections...)
	}

	total := len(results)
	if req.Offset >= total {
		results = nil
	} else {
		results = results[req.Offset:]
		if len(results) > req.Limit {
			results = results[:req.Limit]
		}
	}

	return search.SearchResponse{
		Results: results,
		Total:   total,
		Query:   req.Query,
	}, nil
}

func (r *Repository) searchAssets(ctx context.Context, tenantID, orgID, rawQuery string, limit, offset int) ([]search.Result, error) {
	query := `
		SELECT id, tenant_id, organization_id,
			asset_id || ' — ' || asset_type || ' ' || serial_number,
			ts_headline('english', description, plainto_tsquery('english', $3), 'StartSel=<b>, StopSel=</b>, MaxWords=50, MinWords=20'),
			ts_rank(search_vector, plainto_tsquery('english', $3)) AS rank,
			created_at
		FROM asset_registry
		WHERE tenant_id = $1 AND organization_id = $2
		  AND search_vector @@ plainto_tsquery('english', $3)
		ORDER BY rank DESC
		LIMIT $4 OFFSET $5`

	rows, err := r.DB.QueryContext(ctx, query, tenantID, orgID, rawQuery, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []search.Result
	for rows.Next() {
		var r search.Result
		r.EntityType = search.EntityAsset
		if err := rows.Scan(&r.ID, &r.TenantID, &r.OrganizationID, &r.Title, &r.Snippet, &r.Rank, &r.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (r *Repository) searchWorkOrders(ctx context.Context, tenantID, orgID, rawQuery string, limit, offset int) ([]search.Result, error) {
	query := `
		SELECT id, tenant_id, organization_id,
			job_number || ' — ' || client_id,
			ts_headline('english', job_number || ' ' || client_id || ' ' || request_state || ' ' || execution_state, plainto_tsquery('english', $3), 'StartSel=<b>, StopSel=</b>, MaxWords=50, MinWords=20'),
			ts_rank(search_vector, plainto_tsquery('english', $3)) AS rank,
			created_at
		FROM work_order
		WHERE tenant_id = $1 AND organization_id = $2
		  AND search_vector @@ plainto_tsquery('english', $3)
		ORDER BY rank DESC
		LIMIT $4 OFFSET $5`

	rows, err := r.DB.QueryContext(ctx, query, tenantID, orgID, rawQuery, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []search.Result
	for rows.Next() {
		var r search.Result
		r.EntityType = search.EntityWorkOrder
		if err := rows.Scan(&r.ID, &r.TenantID, &r.OrganizationID, &r.Title, &r.Snippet, &r.Rank, &r.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (r *Repository) searchInspections(ctx context.Context, tenantID, orgID, rawQuery string, limit, offset int) ([]search.Result, error) {
	query := `
		SELECT id, tenant_id, organization_id,
			asset_id || ' — ' || inspector_id,
			ts_headline('english', asset_id || ' ' || inspector_id || ' ' || lifecycle_state || ' ' || finalization_state, plainto_tsquery('english', $3), 'StartSel=<b>, StopSel=</b>, MaxWords=50, MinWords=20'),
			ts_rank(search_vector, plainto_tsquery('english', $3)) AS rank,
			created_at
		FROM inspection_record
		WHERE tenant_id = $1 AND organization_id = $2
		  AND search_vector @@ plainto_tsquery('english', $3)
		ORDER BY rank DESC
		LIMIT $4 OFFSET $5`

	rows, err := r.DB.QueryContext(ctx, query, tenantID, orgID, rawQuery, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []search.Result
	for rows.Next() {
		var r search.Result
		r.EntityType = search.EntityInspection
		if err := rows.Scan(&r.ID, &r.TenantID, &r.OrganizationID, &r.Title, &r.Snippet, &r.Rank, &r.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func containsType(types []search.EntityType, target search.EntityType) bool {
	for _, t := range types {
		if t == target {
			return true
		}
	}
	return false
}
