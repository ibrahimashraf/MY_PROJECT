package response

import "integin/internal/shared/types"

type Definition struct {
	Type        types.ResponseType `json:"response_type"`
	Required    bool               `json:"required"`
	Unit        string             `json:"unit,omitempty"`
	Options     []string           `json:"options,omitempty"`
	GridRows    int                `json:"grid_rows,omitempty"`
	GridColumns int                `json:"grid_columns,omitempty"`
}

type NumericGridCell struct {
	Row    int     `json:"row"`
	Column int     `json:"column"`
	Value  float64 `json:"value"`
}
type MatrixMeasurement struct {
	Rows    int         `json:"rows"`
	Columns int         `json:"columns"`
	Values  [][]float64 `json:"values"`
}
type PressureTest struct {
	Pressure        float64 `json:"pressure"`
	Unit            string  `json:"unit"`
	DurationMinutes float64 `json:"duration_minutes"`
	Result          bool    `json:"result"`
	ChartReference  string  `json:"chart_reference,omitempty"`
}

type CalibrationReference struct {
	CalibrationID string `json:"calibration_id"`
	EquipmentID   string `json:"equipment_id"`
	ValidUntil    string `json:"valid_until"`
}
