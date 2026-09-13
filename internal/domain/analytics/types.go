package analytics

import "time"

// SIUnit identifies the typed SI unit from pkg/units (e.g. "Pascals", "Kilonewtons", "Meters").
type SIUnit string

const (
	SIUnitPascals      SIUnit = "Pascals"
	SIUnitKilonewtons  SIUnit = "Kilonewtons"
	SIUnitMeters       SIUnit = "Meters"
	SIUnitNewtons      SIUnit = "Newtons"
	SIUnitMegapascals  SIUnit = "Megapascals"
	SIUnitMetersPerSec SIUnit = "MetersPerSecond"
)

type TrendDirection string

const (
	TrendUp   TrendDirection = "up"
	TrendDown TrendDirection = "down"
	TrendFlat TrendDirection = "flat"
)

type KPI struct {
	Name      string         `json:"name"`
	Value     float64        `json:"value"`
	Unit      string         `json:"unit,omitzero"`
	SIUnit    SIUnit         `json:"si_unit,omitzero"`
	Trend     TrendDirection `json:"trend"`
	ChangePct float64        `json:"change_pct"`
}

type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp,format:rfc3339"`
	Value     float64   `json:"value"`
	Label     string    `json:"label,omitzero"`
}

type StateCount struct {
	State string `json:"state"`
	Count int64  `json:"count"`
}

type InspectorPerformance struct {
	InspectorID   string  `json:"inspector_id"`
	InspectorName string  `json:"inspector_name,omitzero"`
	Total         int64   `json:"total"`
	PassCount     int64   `json:"pass_count"`
	FailCount     int64   `json:"fail_count"`
	PassRate      float64 `json:"pass_rate"`
}

type AssetBreakdown struct {
	AssetType string `json:"asset_type"`
	Count     int64  `json:"count"`
}

type DashboardResponse struct {
	Summary              []KPI                  `json:"summary"`
	WorkOrdersByState    []StateCount           `json:"work_orders_by_state"`
	InspectionsByState   []StateCount           `json:"inspections_by_state"`
	InspectionsTrend     []TimeSeriesPoint      `json:"inspections_trend"`
	InspectorPerformance []InspectorPerformance `json:"inspector_performance"`
	AssetBreakdown       []AssetBreakdown       `json:"asset_breakdown"`
}

type DashboardRequest struct {
	TenantID       string
	OrganizationID string
	From           time.Time
	To             time.Time
	InspectorID    string
	AssetType      string
}
