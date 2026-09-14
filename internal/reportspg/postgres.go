package reportspg

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	domain "integin/internal/domain/reports"
	csvgen "integin/internal/reports"
)

var (
	ErrNilDB = errors.New("reports postgres repository requires a database")
)

type Repository struct {
	db *sql.DB
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

func (r *Repository) ListConfigs(ctx context.Context, tenantID, orgID string) ([]domain.ReportConfig, error) {
	tx, err := r.beginTenant(ctx, tenantID, orgID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx,
		`SELECT id, tenant_id, organization_id, name, type, format,
		        schedule, filters, created_by, created_at, updated_at
		 FROM report_config
		 WHERE tenant_id = $1 AND organization_id = $2
		 ORDER BY created_at DESC`,
		tenantID, orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []domain.ReportConfig
	for rows.Next() {
		var c domain.ReportConfig
		var scheduleJSON, filtersJSON []byte
		if err := rows.Scan(&c.ID, &c.TenantID, &c.OrgID, &c.Name, &c.Type, &c.Format,
			&scheduleJSON, &filtersJSON, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		if scheduleJSON != nil {
			_ = json.Unmarshal(scheduleJSON, &c.Schedule)
		}
		if filtersJSON != nil {
			_ = json.Unmarshal(filtersJSON, &c.Filters)
		}
		configs = append(configs, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return configs, tx.Commit()
}

func (r *Repository) GetConfig(ctx context.Context, tenantID, orgID, id string) (domain.ReportConfig, error) {
	tx, err := r.beginTenant(ctx, tenantID, orgID)
	if err != nil {
		return domain.ReportConfig{}, err
	}
	defer tx.Rollback()

	var c domain.ReportConfig
	var scheduleJSON, filtersJSON []byte
	err = tx.QueryRowContext(ctx,
		`SELECT id, tenant_id, organization_id, name, type, format,
		        schedule, filters, created_by, created_at, updated_at
		 FROM report_config
		 WHERE id = $1 AND tenant_id = $2 AND organization_id = $3`,
		id, tenantID, orgID,
	).Scan(&c.ID, &c.TenantID, &c.OrgID, &c.Name, &c.Type, &c.Format,
		&scheduleJSON, &filtersJSON, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ReportConfig{}, domain.ErrConfigNotFound
	}
	if err != nil {
		return domain.ReportConfig{}, err
	}
	if scheduleJSON != nil {
		_ = json.Unmarshal(scheduleJSON, &c.Schedule)
	}
	if filtersJSON != nil {
		_ = json.Unmarshal(filtersJSON, &c.Filters)
	}
	return c, tx.Commit()
}

func (r *Repository) CreateConfig(ctx context.Context, config domain.ReportConfig) (domain.ReportConfig, error) {
	tx, err := r.beginTenant(ctx, config.TenantID, config.OrgID)
	if err != nil {
		return domain.ReportConfig{}, err
	}
	defer tx.Rollback()

	now := time.Now()
	if config.ID == "" {
		config.ID = config.TenantID + ":rpt:" + now.Format("20060102150405.000000000")
	}
	config.CreatedAt = now
	config.UpdatedAt = now

	scheduleJSON, _ := json.Marshal(config.Schedule)
	filtersJSON, _ := json.Marshal(config.Filters)

	_, err = tx.ExecContext(ctx,
		`INSERT INTO report_config (id, tenant_id, organization_id, name, type, format,
		        schedule, filters, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		config.ID, config.TenantID, config.OrgID, config.Name, config.Type, config.Format,
		scheduleJSON, filtersJSON, config.CreatedBy, config.CreatedAt, config.UpdatedAt,
	)
	if err != nil {
		return domain.ReportConfig{}, err
	}
	return config, tx.Commit()
}

func (r *Repository) UpdateConfig(ctx context.Context, config domain.ReportConfig) (domain.ReportConfig, error) {
	tx, err := r.beginTenant(ctx, config.TenantID, config.OrgID)
	if err != nil {
		return domain.ReportConfig{}, err
	}
	defer tx.Rollback()

	config.UpdatedAt = time.Now()
	scheduleJSON, _ := json.Marshal(config.Schedule)
	filtersJSON, _ := json.Marshal(config.Filters)

	result, err := tx.ExecContext(ctx,
		`UPDATE report_config
		 SET name = $4, type = $5, format = $6, schedule = $7, filters = $8, updated_at = $9
		 WHERE id = $1 AND tenant_id = $2 AND organization_id = $3`,
		config.ID, config.TenantID, config.OrgID, config.Name, config.Type, config.Format,
		scheduleJSON, filtersJSON, config.UpdatedAt,
	)
	if err != nil {
		return domain.ReportConfig{}, err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ReportConfig{}, domain.ErrConfigNotFound
	}
	return config, tx.Commit()
}

func (r *Repository) DeleteConfig(ctx context.Context, tenantID, orgID, id string) error {
	tx, err := r.beginTenant(ctx, tenantID, orgID)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		`DELETE FROM report_config WHERE id = $1 AND tenant_id = $2 AND organization_id = $3`,
		id, tenantID, orgID,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrConfigNotFound
	}
	return tx.Commit()
}

func (r *Repository) GenerateReport(ctx context.Context, req domain.GenerateRequest, tenantID, orgID string) (domain.GeneratedReport, error) {
	config, err := r.GetConfig(ctx, tenantID, orgID, req.ConfigID)
	if err != nil {
		return domain.GeneratedReport{}, err
	}

	format := config.Format
	if req.FormatOverride != "" {
		format = req.FormatOverride
	}

	reportID := tenantID + ":rptgen:" + time.Now().Format("20060102150405.000000000")

	headers, rows, err := r.queryReportData(ctx, config, req)
	if err != nil {
		_ = r.insertReport(ctx, domain.GeneratedReport{
			ID:        reportID,
			ConfigID:  req.ConfigID,
			TenantID:  tenantID,
			OrgID:     orgID,
			Status:    domain.ReportStatusFailed,
			ErrorMsg:  err.Error(),
			CreatedAt: time.Now(),
		})
		return domain.GeneratedReport{}, err
	}

	csvData, err := csvgen.GenerateCSV(headers, rows)
	if err != nil {
		_ = r.insertReport(ctx, domain.GeneratedReport{
			ID:        reportID,
			ConfigID:  req.ConfigID,
			TenantID:  tenantID,
			OrgID:     orgID,
			Status:    domain.ReportStatusFailed,
			ErrorMsg:  err.Error(),
			CreatedAt: time.Now(),
		})
		return domain.GeneratedReport{}, err
	}

	filePath := fmt.Sprintf("reports/%s_%s.%s", reportID, time.Now().Format("20060102"), string(format))

	report := domain.GeneratedReport{
		ID:        reportID,
		ConfigID:  req.ConfigID,
		TenantID:  tenantID,
		OrgID:     orgID,
		Status:    domain.ReportStatusReady,
		FilePath:  filePath,
		RowCount:  len(rows),
		CreatedAt: time.Now(),
	}

	if err := r.insertReport(ctx, report); err != nil {
		return domain.GeneratedReport{}, err
	}

	_ = csvData

	return report, nil
}

func (r *Repository) insertReport(ctx context.Context, report domain.GeneratedReport) error {
	tx, err := r.beginTenant(ctx, report.TenantID, report.OrgID)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO generated_report (id, config_id, tenant_id, organization_id, status, file_path, row_count, error_msg, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		report.ID, report.ConfigID, report.TenantID, report.OrgID, report.Status,
		report.FilePath, report.RowCount, report.ErrorMsg, report.CreatedAt,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) queryReportData(ctx context.Context, config domain.ReportConfig, req domain.GenerateRequest) ([]string, [][]string, error) {
	switch config.Type {
	case domain.ReportInspectionSummary:
		return r.queryInspectionSummary(ctx, config, req)
	case domain.ReportWorkOrderStatus:
		return r.queryWorkOrderStatus(ctx, config, req)
	case domain.ReportAssetInventory:
		return r.queryAssetInventory(ctx, config, req)
	case domain.ReportInspectorPerformance:
		return r.queryInspectorPerformance(ctx, config, req)
	default:
		return nil, nil, fmt.Errorf("unsupported report type: %s", config.Type)
	}
}

func (r *Repository) queryInspectionSummary(ctx context.Context, config domain.ReportConfig, req domain.GenerateRequest) ([]string, [][]string, error) {
	headers := []string{"ID", "Work Order ID", "Asset ID", "Inspector ID", "Lifecycle State", "Finalization State", "Created At"}

	query := `SELECT id, work_order_id, asset_id, inspector_id, lifecycle_state, finalization_state, created_at
	          FROM inspection_record WHERE tenant_id = $1 AND organization_id = $2`
	args := []interface{}{config.TenantID, config.OrgID}
	argIdx := 3

	if req.DateFrom != "" {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, req.DateFrom)
		argIdx++
	}
	if req.DateTo != "" {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx) //nolint:G202 // placeholder index only; values parameterized
		args = append(args, req.DateTo)
		argIdx++
	}
	_ = argIdx // used conditionally

	query += " ORDER BY created_at DESC"

	tx, err := r.beginTenant(ctx, config.TenantID, config.OrgID)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var data [][]string
	for rows.Next() {
		var id, woID, assetID, inspectorID, lifecycleState, finalizationState, createdAt string
		if err := rows.Scan(&id, &woID, &assetID, &inspectorID, &lifecycleState, &finalizationState, &createdAt); err != nil {
			return nil, nil, err
		}
		data = append(data, []string{id, woID, assetID, inspectorID, lifecycleState, finalizationState, createdAt})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return headers, data, tx.Commit()
}

func (r *Repository) queryWorkOrderStatus(ctx context.Context, config domain.ReportConfig, req domain.GenerateRequest) ([]string, [][]string, error) {
	headers := []string{"ID", "Job Number", "Client ID", "Request State", "Execution State", "Commercial State", "Certificate State", "Created At"}

	query := `SELECT id, job_number, client_id, request_state, execution_state, commercial_state, certificate_state, created_at
	          FROM work_order WHERE tenant_id = $1 AND organization_id = $2`
	args := []interface{}{config.TenantID, config.OrgID}
	argIdx := 3

	if req.DateFrom != "" {
		//nolint:gosec // placeholder index only; values parameterized
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, req.DateFrom)
		argIdx++
	}
	if req.DateTo != "" {
		//nolint:gosec // placeholder index only; values parameterized
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, req.DateTo)
		argIdx++
	}
	_ = argIdx // used conditionally

	query += " ORDER BY created_at DESC"

	tx, err := r.beginTenant(ctx, config.TenantID, config.OrgID)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var data [][]string
	for rows.Next() {
		var id, jobNum, clientID, reqState, execState, commState, certState, createdAt string
		if err := rows.Scan(&id, &jobNum, &clientID, &reqState, &execState, &commState, &certState, &createdAt); err != nil {
			return nil, nil, err
		}
		data = append(data, []string{id, jobNum, clientID, reqState, execState, commState, certState, createdAt})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return headers, data, tx.Commit()
}

func (r *Repository) queryAssetInventory(ctx context.Context, config domain.ReportConfig, req domain.GenerateRequest) ([]string, [][]string, error) {
	headers := []string{"ID", "Tag", "Type", "Status", "Created At"}

	query := `SELECT id, tag, type, status, created_at
	          FROM asset WHERE tenant_id = $1 AND organization_id = $2`
	args := []interface{}{config.TenantID, config.OrgID}
	argIdx := 3

	if req.DateFrom != "" {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, req.DateFrom)
		argIdx++
	}
	if req.DateTo != "" {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx) //nolint:G202 // placeholder index only; values parameterized
		args = append(args, req.DateTo)
		argIdx++
	}
	_ = argIdx // used conditionally

	query += " ORDER BY created_at DESC"

	tx, err := r.beginTenant(ctx, config.TenantID, config.OrgID)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var data [][]string
	for rows.Next() {
		var id, tag, assetType, status, createdAt string
		if err := rows.Scan(&id, &tag, &assetType, &status, &createdAt); err != nil {
			return nil, nil, err
		}
		data = append(data, []string{id, tag, assetType, status, createdAt})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return headers, data, tx.Commit()
}

func (r *Repository) queryInspectorPerformance(ctx context.Context, config domain.ReportConfig, req domain.GenerateRequest) ([]string, [][]string, error) {
	headers := []string{"Inspector ID", "Total Inspections", "Finalized Inspections", "Finalization Rate"}

	query := `SELECT inspector_id,
	                 COUNT(*) as total,
	                 COUNT(*) FILTER (WHERE finalization_state = 'FINALIZED') as finalized
	          FROM inspection_record
	          WHERE tenant_id = $1 AND organization_id = $2`
	args := []interface{}{config.TenantID, config.OrgID}
	argIdx := 3

	if req.DateFrom != "" {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, req.DateFrom)
		argIdx++
	}
	if req.DateTo != "" {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx) //nolint:G202 // placeholder index only; values parameterized
		args = append(args, req.DateTo)
		argIdx++
	}
	_ = argIdx // used conditionally

	query += " GROUP BY inspector_id ORDER BY total DESC"

	tx, err := r.beginTenant(ctx, config.TenantID, config.OrgID)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var data [][]string
	for rows.Next() {
		var inspectorID string
		var total, finalized int
		if err := rows.Scan(&inspectorID, &total, &finalized); err != nil {
			return nil, nil, err
		}
		rate := 0.0
		if total > 0 {
			rate = float64(finalized) / float64(total) * 100.0
		}
		data = append(data, []string{
			inspectorID,
			fmt.Sprintf("%d", total), fmt.Sprintf("%d", finalized),
			fmt.Sprintf("%.1f%%", rate),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return headers, data, tx.Commit()
}

func (r *Repository) ListReports(ctx context.Context, tenantID, orgID string) ([]domain.GeneratedReport, error) {
	tx, err := r.beginTenant(ctx, tenantID, orgID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx,
		`SELECT id, config_id, tenant_id, organization_id, status, file_path, row_count, error_msg, created_at
		 FROM generated_report
		 WHERE tenant_id = $1 AND organization_id = $2
		 ORDER BY created_at DESC`,
		tenantID, orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.GeneratedReport
	for rows.Next() {
		var rpt domain.GeneratedReport
		if err := rows.Scan(&rpt.ID, &rpt.ConfigID, &rpt.TenantID, &rpt.OrgID, &rpt.Status,
			&rpt.FilePath, &rpt.RowCount, &rpt.ErrorMsg, &rpt.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, rpt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, tx.Commit()
}

func (r *Repository) GetReport(ctx context.Context, tenantID, orgID, id string) (domain.GeneratedReport, error) {
	tx, err := r.beginTenant(ctx, tenantID, orgID)
	if err != nil {
		return domain.GeneratedReport{}, err
	}
	defer tx.Rollback()

	var rpt domain.GeneratedReport
	err = tx.QueryRowContext(ctx,
		`SELECT id, config_id, tenant_id, organization_id, status, file_path, row_count, error_msg, created_at
		 FROM generated_report
		 WHERE id = $1 AND tenant_id = $2 AND organization_id = $3`,
		id, tenantID, orgID,
	).Scan(&rpt.ID, &rpt.ConfigID, &rpt.TenantID, &rpt.OrgID, &rpt.Status,
		&rpt.FilePath, &rpt.RowCount, &rpt.ErrorMsg, &rpt.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.GeneratedReport{}, domain.ErrReportNotFound
	}
	if err != nil {
		return domain.GeneratedReport{}, err
	}
	return rpt, tx.Commit()
}

func (r *Repository) GenerateCSVData(ctx context.Context, req domain.GenerateRequest, tenantID, orgID string) ([]string, [][]string, error) {
	config, err := r.GetConfig(ctx, tenantID, orgID, req.ConfigID)
	if err != nil {
		return nil, nil, err
	}
	return r.queryReportData(ctx, config, req)
}
