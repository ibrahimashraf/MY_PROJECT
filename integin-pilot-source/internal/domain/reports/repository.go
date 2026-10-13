package reports

import "context"

type Repository interface {
	ListConfigs(ctx context.Context, tenantID, orgID string) ([]ReportConfig, error)
	GetConfig(ctx context.Context, tenantID, orgID, id string) (ReportConfig, error)
	CreateConfig(ctx context.Context, config ReportConfig) (ReportConfig, error)
	UpdateConfig(ctx context.Context, config ReportConfig) (ReportConfig, error)
	DeleteConfig(ctx context.Context, tenantID, orgID, id string) error
	GenerateReport(ctx context.Context, req GenerateRequest, tenantID, orgID string) (GeneratedReport, error)
	GenerateCSVData(ctx context.Context, req GenerateRequest, tenantID, orgID string) ([]string, [][]string, error)
	ListReports(ctx context.Context, tenantID, orgID string) ([]GeneratedReport, error)
	GetReport(ctx context.Context, tenantID, orgID, id string) (GeneratedReport, error)
}
