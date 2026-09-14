package analyticspg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"integin/internal/domain/analytics"
)

var ErrNilDB = errors.New("analytics postgres repository requires a database")

type dashboardCacheEntry struct {
	response  analytics.DashboardResponse
	expiresAt time.Time
}

type Repository struct {
	db    *sql.DB
	cache sync.Map
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

func (r *Repository) beginTenant(ctx context.Context, tenantID, orgID string) (*sql.Tx, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)`, tenantID, orgID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func (r *Repository) GetDashboard(ctx context.Context, req analytics.DashboardRequest) (analytics.DashboardResponse, error) {
	resp := analytics.DashboardResponse{}

	if req.TenantID == "" || req.OrganizationID == "" {
		return resp, errors.New("tenant_id and organization_id are required")
	}

	if req.To.IsZero() {
		req.To = time.Now()
	}
	if req.From.IsZero() {
		req.From = req.To.AddDate(0, -1, 0)
	}

	// In-memory 30s TTL cache: prevents repeated dashboard loads from saturating connection pool.
	// Round default To timestamp to 10-second boundaries to achieve cache hits across near-simultaneous requests.
	toUnix := req.To.Unix()
	if req.To.Equal(req.From.AddDate(0, 1, 0)) || req.To.Sub(time.Now()).Abs() < 2*time.Second {
		toUnix = (toUnix / 10) * 10
	}
	cacheKey := fmt.Sprintf("%s|%s|%d|%d|%s|%s",
		req.TenantID, req.OrganizationID,
		req.From.Unix(), toUnix,
		req.InspectorID, req.AssetType)

	now := time.Now()
	if val, ok := r.cache.Load(cacheKey); ok {
		entry := val.(dashboardCacheEntry)
		if now.Before(entry.expiresAt) {
			return entry.response, nil
		}
		r.cache.Delete(cacheKey)
	}

	// Proactive bounded cleanup: purge expired cache keys periodically
	if now.Unix()%30 == 0 {
		r.cache.Range(func(k, v any) bool {
			if e, ok := v.(dashboardCacheEntry); ok && now.After(e.expiresAt) {
				r.cache.Delete(k)
			}
			return true
		})
	}

	tx, err := r.beginTenant(ctx, req.TenantID, req.OrganizationID)
	if err != nil {
		return resp, err
	}
	defer tx.Rollback()

	summary, err := r.getSummaryKPIs(ctx, tx, req)
	if err != nil {
		return resp, err
	}
	resp.Summary = summary

	woByState, err := r.getWorkOrdersByState(ctx, tx, req)
	if err != nil {
		return resp, err
	}
	resp.WorkOrdersByState = woByState

	insByState, err := r.getInspectionsByState(ctx, tx, req)
	if err != nil {
		return resp, err
	}
	resp.InspectionsByState = insByState

	trend, err := r.getInspectionsTrend(ctx, tx, req)
	if err != nil {
		return resp, err
	}
	resp.InspectionsTrend = trend

	insPerf, err := r.getInspectorPerformance(ctx, tx, req)
	if err != nil {
		return resp, err
	}
	resp.InspectorPerformance = insPerf

	assetBD, err := r.getAssetBreakdown(ctx, tx, req)
	if err != nil {
		return resp, err
	}
	resp.AssetBreakdown = assetBD

	if err := tx.Commit(); err != nil {
		return resp, err
	}

	r.cache.Store(cacheKey, dashboardCacheEntry{
		response:  resp,
		expiresAt: time.Now().Add(30 * time.Second),
	})

	return resp, nil
}

func (r *Repository) getSummaryKPIs(ctx context.Context, tx *sql.Tx, req analytics.DashboardRequest) ([]analytics.KPI, error) {
	var totalWO int64
	var completionRate float64
	var totalIns int64
	var avgCycleTime float64

	// Unified work_order aggregation: 1 query instead of 2 roundtrips
	err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*),
		        COALESCE(
					CASE WHEN COUNT(*) = 0 THEN 0
					ELSE SUM(CASE WHEN execution_state = 'completed' THEN 1 ELSE 0 END)::float8 / COUNT(*) * 100
					END, 0)
		 FROM work_order
		 WHERE tenant_id = $1 AND organization_id = $2
		   AND created_at BETWEEN $3 AND $4`,
		req.TenantID, req.OrganizationID, req.From, req.To,
	).Scan(&totalWO, &completionRate)
	if err != nil {
		return nil, err
	}

	// Unified inspection_record aggregation: 1 query instead of 2 roundtrips
	err = tx.QueryRowContext(ctx,
		`SELECT COUNT(*),
		        COALESCE(AVG(CASE WHEN lifecycle_state IN ('approved', 'rejected', 'completed')
		                          THEN EXTRACT(EPOCH FROM (updated_at - created_at))
		                          ELSE NULL END), 0)
		 FROM inspection_record
		 WHERE tenant_id = $1 AND organization_id = $2
		   AND created_at BETWEEN $3 AND $4`,
		req.TenantID, req.OrganizationID, req.From, req.To,
	).Scan(&totalIns, &avgCycleTime)
	if err != nil {
		return nil, err
	}

	return []analytics.KPI{
		{Name: "total_work_orders", Value: float64(totalWO), Trend: analytics.TrendFlat, ChangePct: 0},
		{Name: "total_inspections", Value: float64(totalIns), Trend: analytics.TrendFlat, ChangePct: 0},
		{Name: "completion_rate", Value: completionRate, Unit: "%", Trend: analytics.TrendFlat, ChangePct: 0},
		{Name: "avg_cycle_time", Value: avgCycleTime, Unit: "seconds", Trend: analytics.TrendFlat, ChangePct: 0},
	}, nil
}

func (r *Repository) getWorkOrdersByState(ctx context.Context, tx *sql.Tx, req analytics.DashboardRequest) ([]analytics.StateCount, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT request_state, COUNT(*) FROM work_order
		 WHERE tenant_id = $1 AND organization_id = $2
		   AND created_at BETWEEN $3 AND $4
		 GROUP BY request_state ORDER BY request_state`,
		req.TenantID, req.OrganizationID, req.From, req.To,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []analytics.StateCount
	for rows.Next() {
		var sc analytics.StateCount
		if err := rows.Scan(&sc.State, &sc.Count); err != nil {
			return nil, err
		}
		result = append(result, sc)
	}
	if result == nil {
		result = []analytics.StateCount{}
	}
	return result, rows.Err()
}

func (r *Repository) getInspectionsByState(ctx context.Context, tx *sql.Tx, req analytics.DashboardRequest) ([]analytics.StateCount, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT lifecycle_state, COUNT(*) FROM inspection_record
		 WHERE tenant_id = $1 AND organization_id = $2
		   AND created_at BETWEEN $3 AND $4
		 GROUP BY lifecycle_state ORDER BY lifecycle_state`,
		req.TenantID, req.OrganizationID, req.From, req.To,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []analytics.StateCount
	for rows.Next() {
		var sc analytics.StateCount
		if err := rows.Scan(&sc.State, &sc.Count); err != nil {
			return nil, err
		}
		result = append(result, sc)
	}
	if result == nil {
		result = []analytics.StateCount{}
	}
	return result, rows.Err()
}

func (r *Repository) getInspectionsTrend(ctx context.Context, tx *sql.Tx, req analytics.DashboardRequest) ([]analytics.TimeSeriesPoint, error) {
	daysDiff := req.To.Sub(req.From).Hours() / 24
	var trunc string
	switch {
	case daysDiff <= 31:
		trunc = "day"
	case daysDiff <= 90:
		trunc = "week"
	default:
		trunc = "month"
	}
	// whitelist to avoid SQL injection via fmt.Sprintf
	allowed := map[string]bool{"day": true, "week": true, "month": true}
	if !allowed[trunc] {
		trunc = "day"
	}

	query := fmt.Sprintf(
		`SELECT date_trunc('%s', created_at) AS period, COUNT(*) FROM inspection_record
		 WHERE tenant_id = $1 AND organization_id = $2
		   AND created_at BETWEEN $3 AND $4
		 GROUP BY period ORDER BY period`, trunc) //nolint:G201 // trunc whitelisted
	rows, err := tx.QueryContext(ctx, query,
		req.TenantID, req.OrganizationID, req.From, req.To,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []analytics.TimeSeriesPoint
	for rows.Next() {
		var tp analytics.TimeSeriesPoint
		if err := rows.Scan(&tp.Timestamp, &tp.Value); err != nil {
			return nil, err
		}
		result = append(result, tp)
	}
	if result == nil {
		result = []analytics.TimeSeriesPoint{}
	}
	return result, rows.Err()
}

func (r *Repository) getInspectorPerformance(ctx context.Context, tx *sql.Tx, req analytics.DashboardRequest) ([]analytics.InspectorPerformance, error) {
	query :=
		`SELECT inspector_id, COUNT(*) AS total,
		        SUM(CASE WHEN finalization_state = 'approved' THEN 1 ELSE 0 END) AS pass_count,
		        SUM(CASE WHEN finalization_state = 'rejected' THEN 1 ELSE 0 END) AS fail_count
		 FROM inspection_record
		 WHERE tenant_id = $1 AND organization_id = $2
		   AND created_at BETWEEN $3 AND $4`
	args := []interface{}{req.TenantID, req.OrganizationID, req.From, req.To}

	if req.InspectorID != "" {
		query += " AND inspector_id = $5"
		args = append(args, req.InspectorID)
	}

	query += " GROUP BY inspector_id ORDER BY total DESC"

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []analytics.InspectorPerformance
	for rows.Next() {
		var ip analytics.InspectorPerformance
		if err := rows.Scan(&ip.InspectorID, &ip.Total, &ip.PassCount, &ip.FailCount); err != nil {
			return nil, err
		}
		if ip.Total > 0 {
			ip.PassRate = float64(ip.PassCount) / float64(ip.Total) * 100
		}
		result = append(result, ip)
	}
	if result == nil {
		result = []analytics.InspectorPerformance{}
	}
	return result, rows.Err()
}

func (r *Repository) getAssetBreakdown(ctx context.Context, tx *sql.Tx, req analytics.DashboardRequest) ([]analytics.AssetBreakdown, error) {
	query :=
		`SELECT ar.asset_type, COUNT(DISTINCT ir.id) AS count
		 FROM inspection_record ir
		 JOIN asset_registry ar ON ir.asset_id = ar.asset_id AND ar.tenant_id = ir.tenant_id AND ar.organization_id = ir.organization_id
		 WHERE ir.tenant_id = $1 AND ir.organization_id = $2
		   AND ir.created_at BETWEEN $3 AND $4`
	args := []interface{}{req.TenantID, req.OrganizationID, req.From, req.To}

	if req.AssetType != "" {
		query += " AND ar.asset_type = $5"
		args = append(args, req.AssetType)
	}

	query += " GROUP BY ar.asset_type ORDER BY count DESC"

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []analytics.AssetBreakdown
	for rows.Next() {
		var ab analytics.AssetBreakdown
		if err := rows.Scan(&ab.AssetType, &ab.Count); err != nil {
			return nil, err
		}
		result = append(result, ab)
	}
	if result == nil {
		result = []analytics.AssetBreakdown{}
	}
	return result, rows.Err()
}
