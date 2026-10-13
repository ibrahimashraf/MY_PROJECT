package reports

import (
	"errors"
	"time"
)

type ReportType string

const (
	ReportInspectionSummary    ReportType = "inspection_summary"
	ReportWorkOrderStatus      ReportType = "work_order_status"
	ReportAssetInventory       ReportType = "asset_inventory"
	ReportInspectorPerformance ReportType = "inspector_performance"
)

type Format string

const (
	FormatCSV Format = "csv"
	FormatPDF Format = "pdf"
)

type Frequency string

const (
	FrequencyDaily   Frequency = "daily"
	FrequencyWeekly  Frequency = "weekly"
	FrequencyMonthly Frequency = "monthly"
)

type ReportStatus string

const (
	ReportStatusPending    ReportStatus = "pending"
	ReportStatusGenerating ReportStatus = "generating"
	ReportStatusReady      ReportStatus = "ready"
	ReportStatusFailed     ReportStatus = "failed"
)

var (
	ErrInvalidConfig  = errors.New("invalid report configuration")
	ErrConfigNotFound = errors.New("report configuration not found")
	ErrReportNotFound = errors.New("report not found")
	ErrInvalidTenant  = errors.New("tenant and organization are required")
)

type Schedule struct {
	Frequency  Frequency `json:"frequency"`
	Time       string    `json:"time"`
	Recipients []string  `json:"recipients"`
}

type ReportConfig struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	OrgID     string                 `json:"organization_id"`
	Name      string                 `json:"name"`
	Type      ReportType             `json:"type"`
	Format    Format                 `json:"format"`
	Schedule  Schedule               `json:"schedule"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
	CreatedBy string                 `json:"created_by"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

func (c ReportConfig) Validate() error {
	if c.TenantID == "" || c.OrgID == "" {
		return ErrInvalidTenant
	}
	if c.Name == "" || c.Type == "" {
		return ErrInvalidConfig
	}
	return nil
}

type GenerateRequest struct {
	ConfigID       string `json:"config_id"`
	DateFrom       string `json:"date_from"`
	DateTo         string `json:"date_to"`
	FormatOverride Format `json:"format_override,omitempty"`
}

type GeneratedReport struct {
	ID        string       `json:"id"`
	ConfigID  string       `json:"config_id"`
	TenantID  string       `json:"tenant_id"`
	OrgID     string       `json:"organization_id"`
	Status    ReportStatus `json:"status"`
	FilePath  string       `json:"file_path,omitempty"`
	RowCount  int          `json:"row_count"`
	ErrorMsg  string       `json:"error,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}
