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

func (r *Repository) ListConfigs(ctx context.Context, tenantID, orgID string) ([]domain.ReportConfig, error) {
	rows, err := r.db.QueryContext(ctx,
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
	return configs, rows.Err()
}

func (r *Repository) GetConfig(ctx context.Context, tenantID, orgID, id string) (domain.ReportConfig, error) {
	var c domain.ReportConfig
	var scheduleJSON, filtersJSON []byte
	err := r.db.QueryRowContext(ctx,
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
	return c, nil
}

func (r *Repository) CreateConfig(ctx context.Context, config domain.ReportConfig) (domain.ReportConfig, error) {
	now := time.Now()
	if config.ID == "" {
		config.ID = config.TenantID + ":rpt:" + now.Format("20060102150405.000000000")
	}
	config.CreatedAt = now
	config.UpdatedAt = now

	scheduleJSON, _ := json.Marshal(config.Schedule)
	filtersJSON, _ := json.Marshal(config.Filters)

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO report_config (id, tenant_id, organization_id, name, type, format,
		        schedule, filters, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		config.ID, config.TenantID, config.OrgID, config.Name, config.Type, config.Format,
		scheduleJSON, filtersJSON, config.CreatedBy, config.CreatedAt, config.UpdatedAt,
	)
	if err != nil {
		return domain.ReportConfig{}, err
	}
	return config, nil
}

func (r *Repository) UpdateConfig(ctx context.Context, config domain.ReportConfig) (domain.ReportConfig, error) {
	config.UpdatedAt = time.Now()
	scheduleJSON, _ := json.Marshal(config.Schedule)
	filtersJSON, _ := json.Marshal(config.Filters)

	result, err := r.db.ExecContext(ctx,
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
	return config, nil
}

func (r *Repository) DeleteConfig(ctx context.Context, tenantID, orgID, id string) error {
	result, err := r.db.ExecContext(ctx,
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
	return nil
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
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO generated_report (id, config_id, tenant_id, organization_id, status, file_path, row_count, error_msg, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		report.ID, report.ConfigID, report.TenantID, report.OrgID, report.Status,
		report.FilePath, report.RowCount, report.ErrorMsg, report.CreatedAt,
	)
	return err
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

	rows, err := r.db.QueryContext(ctx, query, args...)
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
	return headers, data, rows.Err()
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

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var data [][]string
	for rows.Next() {
		var id, jobNumber, clientID, requestState, executionState, commercialState, certificateState, createdAt string
		if err := rows.Scan(&id, &jobNumber, &clientID, &requestState, &executionState, &commercialState, &certificateState, &createdAt); err != nil {
			return nil, nil, err
		}
		data = append(data, []string{id, jobNumber, clientID, requestState, executionState, commercialState, certificateState, createdAt})
	}
	return headers, data, rows.Err()
}

func (r *Repository) queryAssetInventory(ctx context.Context, config domain.ReportConfig, req domain.GenerateRequest) ([]string, [][]string, error) {
	headers := []string{"ID", "Asset ID", "Asset Type", "Serial Number", "Description", "Lifecycle State", "Created At"}

	query := `SELECT id, asset_id, asset_type, serial_number, description, lifecycle_state, created_at
	          FROM asset_registry WHERE tenant_id = $1 AND organization_id = $2`
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

	maxLimit := 5000
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argIdx)
	args = append(args, maxLimit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var data [][]string
	for rows.Next() {
		var id, assetID, assetType, serialNumber, description, lifecycleState, createdAt string
		if err := rows.Scan(&id, &assetID, &assetType, &serialNumber, &description, &lifecycleState, &createdAt); err != nil {
			return nil, nil, err
		}
		data = append(data, []string{id, assetID, assetType, serialNumber, description, lifecycleState, createdAt})
	}
	return headers, data, rows.Err()
}

func (r *Repository) queryInspectorPerformance(ctx context.Context, config domain.ReportConfig, req domain.GenerateRequest) ([]string, [][]string, error) {
	headers := []string{"Inspector ID", "Total Inspections", "Finalized", "Completion Rate"}

	query := `SELECT inspector_id,
	                 COUNT(*) AS total,
	                 SUM(CASE WHEN finalization_state = 'FINALIZED' THEN 1 ELSE 0 END) AS finalized,
	                 CASE WHEN COUNT(*) > 0
	                   THEN ROUND(100.0 * SUM(CASE WHEN finalization_state = 'FINALIZED' THEN 1 ELSE 0 END) / COUNT(*), 1)
	                   ELSE 0 END AS rate
	          FROM inspection_record
	          WHERE tenant_id = $1 AND organization_id = $2`
	args := []interface{}{config.TenantID, config.OrgID}
	argIdx := 3

	if req.DateFrom != "" {
		//nolint:gosec // placeholder index only; values parameterized
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

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var data [][]string
	for rows.Next() {
		var inspectorID string
		var total, finalized int
		var rate float64
		if err := rows.Scan(&inspectorID, &total, &finalized, &rate); err != nil {
			return nil, nil, err
		}
		data = append(data, []string{
			inspectorID,
			fmt.Sprintf("%d", total), fmt.Sprintf("%d", finalized),
			fmt.Sprintf("%.1f%%", rate),
		})
	}
	return headers, data, rows.Err()
}

func (r *Repository) ListReports(ctx context.Context, tenantID, orgID string) ([]domain.GeneratedReport, error) {
	rows, err := r.db.QueryContext(ctx,
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
	return result, rows.Err()
}

func (r *Repository) GetReport(ctx context.Context, tenantID, orgID, id string) (domain.GeneratedReport, error) {
	var rpt domain.GeneratedReport
	err := r.db.QueryRowContext(ctx,
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
	return rpt, nil
}

func (r *Repository) GenerateCSVData(ctx context.Context, req domain.GenerateRequest, tenantID, orgID string) ([]string, [][]string, error) {
	config, err := r.GetConfig(ctx, tenantID, orgID, req.ConfigID)
	if err != nil {
		return nil, nil, err
	}
	return r.queryReportData(ctx, config, req)
}
