package searchpg

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"integin/internal/domain/search"
)

type Repository struct {
	DB *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) beginTenant(ctx context.Context, tenantID, orgID string) (*sql.Tx, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)`, tenantID, orgID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func (r *Repository) Search(ctx context.Context, req search.SearchRequest) (search.SearchResponse, error) {
	req.Normalize()

	tx, err := r.beginTenant(ctx, req.TenantID, req.OrganizationID)
	if err != nil {
		return search.SearchResponse{}, err
	}
	defer tx.Rollback()

	var results []search.Result

	searchAsset := len(req.Types) == 0 || containsType(req.Types, search.EntityAsset)
	searchWorkOrder := len(req.Types) == 0 || containsType(req.Types, search.EntityWorkOrder)
	searchInspection := len(req.Types) == 0 || containsType(req.Types, search.EntityInspection)

	if searchAsset {
		assets, err := r.searchAssets(ctx, tx, req.TenantID, req.OrganizationID, req.Query, req.Limit, req.Offset)
		if err != nil {
			return search.SearchResponse{}, fmt.Errorf("search assets: %w", err)
		}
		results = append(results, assets...)
	}

	if searchWorkOrder {
		orders, err := r.searchWorkOrders(ctx, tx, req.TenantID, req.OrganizationID, req.Query, req.Limit, req.Offset)
		if err != nil {
			return search.SearchResponse{}, fmt.Errorf("search work orders: %w", err)
		}
		results = append(results, orders...)
	}

	if searchInspection {
		inspections, err := r.searchInspections(ctx, tx, req.TenantID, req.OrganizationID, req.Query, req.Limit, req.Offset)
		if err != nil {
			return search.SearchResponse{}, fmt.Errorf("search inspections: %w", err)
		}
		results = append(results, inspections...)
	}

	if err := tx.Commit(); err != nil {
		return search.SearchResponse{}, err
	}

	// Sort merged results by relevance rank descending, then by creation time
	sort.Slice(results, func(i, j int) bool {
		if results[i].Rank != results[j].Rank {
			return results[i].Rank > results[j].Rank
		}
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	total := len(results)

	// Apply pagination to merged result set
	if req.Offset > 0 {
		if req.Offset >= len(results) {
			results = nil
		} else {
			results = results[req.Offset:]
		}
	}

	if req.Limit > 0 && len(results) > req.Limit {
		results = results[:req.Limit]
	}

	return search.SearchResponse{
		Results: results,
		Total:   total,
		Query:   req.Query,
	}, nil
}

func (r *Repository) searchAssets(ctx context.Context, tx *sql.Tx, tenantID, orgID, rawQuery string, limit, offset int) ([]search.Result, error) {
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

	rows, err := tx.QueryContext(ctx, query, tenantID, orgID, rawQuery, limit, offset)
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

func (r *Repository) searchWorkOrders(ctx context.Context, tx *sql.Tx, tenantID, orgID, rawQuery string, limit, offset int) ([]search.Result, error) {
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

	rows, err := tx.QueryContext(ctx, query, tenantID, orgID, rawQuery, limit, offset)
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

func (r *Repository) searchInspections(ctx context.Context, tx *sql.Tx, tenantID, orgID, rawQuery string, limit, offset int) ([]search.Result, error) {
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

	rows, err := tx.QueryContext(ctx, query, tenantID, orgID, rawQuery, limit, offset)
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
