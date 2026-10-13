// Package telemetry ingests live ISA-95 Level 2 machine telemetry (crane LMI
// channels) at the ISA-95 Level 3 SCADA boundary and binds each stream to an
// active work order (internal/domain/workorder ID space). Every frame is
// timeguard-validated (Hazard 33) and gated by the statutory crane CEL rule
// in pkg/rulesengine before any safety action fires.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"integin/pkg/id"
	"integin/pkg/rulesengine"
	"integin/pkg/timeguard"
)

// MaxTelemetryClockSkew is the wall-clock tolerance each crane's telemetry
// stream may deviate before frames are rejected as tampered (Hazard 33).
// The payload timestamp is the UUIDv7 millis embedded in the frame, so the
// monotonic counter and the wall clock advance together per sample.
const MaxTelemetryClockSkew = 10 * time.Second

// CraneLMIData is one telemetry frame from the crane's LMI (Load Moment
// Indicator) computer. moment_utilization_pct and anti_two_block_triggered
// are reported by the machine's own safety system and are never recomputed
// here — zero LLM math on load-limit determinations.
type CraneLMIData struct {
	CraneID               string    `json:"crane_id"`
	TimestampUUIDv7       string    `json:"timestamp_uuidv7"`
	BoomLengthM           float64   `json:"boom_length_m"`
	BoomAngleDeg          float64   `json:"boom_angle_deg"`
	WorkingRadiusM        float64   `json:"working_radius_m"`
	ActualLoadTonnes      float64   `json:"actual_load_tonnes"`
	RatedCapacityTonnes   float64   `json:"rated_capacity_tonnes"`
	MomentUtilizationPct  float64   `json:"moment_utilization_pct"`
	OutriggerPressuresKpa []float64 `json:"outrigger_pressures_kpa"`
	WindSpeedMps          float64   `json:"wind_speed_mps"`
	AntiTwoBlockTriggered bool      `json:"anti_two_block_triggered"`
}

// Validate checks every scalar for physical plausibility before the CEL gate
// runs. NaN/Inf (Hazard 31) and negative readings are rejected outright;
// rated_capacity_tonnes must be positive.
func (d CraneLMIData) Validate() error {
	if strings.TrimSpace(d.CraneID) == "" {
		return errors.New("telemetry: crane_id is required")
	}
	if !id.IsValidV7(d.TimestampUUIDv7) {
		return errors.New("telemetry: timestamp must be a valid UUIDv7")
	}
	finite := func(v float64, name string) error {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("telemetry: %s must be finite, got %v", name, v)
		}
		return nil
	}
	for _, field := range []struct {
		v    float64
		name string
	}{
		{d.BoomLengthM, "boom_length_m"},
		{d.BoomAngleDeg, "boom_angle_deg"},
		{d.WorkingRadiusM, "working_radius_m"},
		{d.ActualLoadTonnes, "actual_load_tonnes"},
		{d.RatedCapacityTonnes, "rated_capacity_tonnes"},
		{d.MomentUtilizationPct, "moment_utilization_pct"},
		{d.WindSpeedMps, "wind_speed_mps"},
	} {
		if err := finite(field.v, field.name); err != nil {
			return err
		}
	}
	for i, p := range d.OutriggerPressuresKpa {
		if math.IsNaN(p) || math.IsInf(p, 0) {
			return fmt.Errorf("telemetry: outrigger_pressures_kpa[%d] must be finite, got %v", i, p)
		}
	}
	for _, field := range []struct {
		v, min float64
		name   string
	}{
		{d.BoomLengthM, 0, "boom_length_m"},
		{d.WorkingRadiusM, 0, "working_radius_m"},
		{d.ActualLoadTonnes, 0, "actual_load_tonnes"},
		{d.RatedCapacityTonnes, 1, "rated_capacity_tonnes"},
		{d.MomentUtilizationPct, 0, "moment_utilization_pct"},
		{d.WindSpeedMps, 0, "wind_speed_mps"},
	} {
		if field.v < field.min {
			return fmt.Errorf("telemetry: %s must be >= %v, got %v", field.name, field.min, field.v)
		}
	}
	// Boom angles are measured from horizontal; anything outside [-180, 180]
	// degrees is a corrupted channel, not a physical boom pose.
	if d.BoomAngleDeg < -180 || d.BoomAngleDeg > 180 {
		return fmt.Errorf("telemetry: boom_angle_deg %v outside physical range", d.BoomAngleDeg)
	}
	for i, p := range d.OutriggerPressuresKpa {
		if p < 0 {
			return fmt.Errorf("telemetry: outrigger_pressures_kpa[%d] must be >= 0, got %v", i, p)
		}
	}
	return nil
}

// CriticalAlertKind classifies the statutory violation that tripped the gate.
type CriticalAlertKind string

const (
	AlertMomentUtilization CriticalAlertKind = "MOMENT_UTILIZATION_EXCEEDED"
	AlertAntiTwoBlock      CriticalAlertKind = "ANTI_TWO_BLOCK_TRIGGERED"
)

// CriticalTelemetryAlert is emitted when the operational gate fails. It is
// bound to the work order the stream was ingested under, so the runtime can
// route the hold to the exact active order.
type CriticalTelemetryAlert struct {
	AlertID              string            `json:"alert_id"`
	WorkOrderID          string            `json:"work_order_id"`
	CraneID              string            `json:"crane_id"`
	Timestamp            time.Time         `json:"timestamp"`
	Kind                 CriticalAlertKind `json:"kind"`
	MomentUtilizationPct float64           `json:"moment_utilization_pct"`
	Message              string            `json:"message"`
}

// SafetyActions are the mandatory statutory side effects the bridge fires on
// gate failure. Both must be wired by the host; nil actions are a
// construction error (never silently disabled — Hazard 32).
type SafetyActions struct {
	// HoldWorkOrder applies the work-order safety hold (concretely the
	// ExecutionInProgress -> ExecutionSuspended transition in
	// internal/domain/workorder) for the active order the stream is bound to.
	HoldWorkOrder func(ctx context.Context, workOrderID string, reason string) error
	// EmitAlert delivers the CriticalTelemetryAlert to the alerting bus.
	EmitAlert func(ctx context.Context, alert CriticalTelemetryAlert) error
}

// IngestVerdict is the per-frame result. Safe=false means the statutory gate
// failed; Alert carries the emitted alert (non-nil exactly when the gate
// failed and the safety actions succeeded).
type IngestVerdict struct {
	Safe  bool
	Alert *CriticalTelemetryAlert
}

// SCADABridge binds live crane telemetry streams to active work orders and
// enforces the statutory moment-utilization / anti-two-block gate via the
// rulesengine CEL sandbox. Each crane gets its own ClockGuard so clock
// tampering on one stream never poisons another.
type SCADABridge struct {
	actions SafetyActions
	gate    *rulesengine.Program

	mu     sync.Mutex
	clocks map[string]*timeguard.ClockGuard
}

// NewSCADABridge builds the bridge and compiles the statutory crane LMI gate
// once. Both safety actions are required; passing either as nil fails closed.
func NewSCADABridge(actions SafetyActions) (*SCADABridge, error) {
	if actions.HoldWorkOrder == nil || actions.EmitAlert == nil {
		return nil, errors.New("telemetry: HoldWorkOrder and EmitAlert safety actions are both required")
	}
	ev, err := rulesengine.NewEvaluator()
	if err != nil {
		return nil, err
	}
	gate, err := ev.Compile(
		rulesengine.CraneLMIOperationalGateRuleID,
		rulesengine.CraneLMIOperationalGateExpression,
		rulesengine.CraneLMIOperationalGateVars(),
	)
	if err != nil {
		return nil, err
	}
	return &SCADABridge{
		actions: actions,
		gate:    gate,
		clocks:  make(map[string]*timeguard.ClockGuard),
	}, nil
}

// IngestTelemetry timeguard-validates one frame for the given work order and
// evaluates the statutory gate. The gate pipeline is: physical validation ->
// clock guard -> CEL gate -> (on failure) work-order safety hold + alert.
// Any CEL or safety-hook error is propagated: the bridge fails closed and
// never silently drops a critical frame.
func (b *SCADABridge) IngestTelemetry(ctx context.Context, workOrderID string, data CraneLMIData) (IngestVerdict, error) {
	if strings.TrimSpace(workOrderID) == "" {
		return IngestVerdict{}, errors.New("telemetry: work order id is required to bind the telemetry stream")
	}
	if err := data.Validate(); err != nil {
		return IngestVerdict{}, err
	}
	ts, err := id.ParseTime(data.TimestampUUIDv7)
	if err != nil {
		return IngestVerdict{}, fmt.Errorf("telemetry: invalid timestamp: %w", err)
	}
	if err := b.clockFor(data.CraneID).ValidateSample(timeguard.MonotonicSample{
		WallTime:      ts,
		MonotonicNano: ts.UnixMilli(),
		Source:        timeguard.MONOTONIC_LOCAL,
	}); err != nil {
		return IngestVerdict{}, fmt.Errorf("telemetry: timeguard rejected crane %s frame: %w", data.CraneID, err)
	}

	vars := rulesengine.NewVarSet("moment_utilization_pct", "anti_two_block_triggered")
	vars.Put("moment_utilization_pct", data.MomentUtilizationPct)
	vars.Put("anti_two_block_triggered", data.AntiTwoBlockTriggered)
	safe, err := b.gate.Evaluate(ctx, vars)
	if err != nil {
		return IngestVerdict{}, fmt.Errorf("telemetry: statutory gate evaluation failed: %w", err)
	}
	if safe {
		return IngestVerdict{Safe: true}, nil
	}

	alert := b.buildAlert(workOrderID, data, ts)
	// Hold first: the statutory restraint is the side effect that cannot be
	// skipped. Only if it lands do we emit the alert; both failures propagate.
	if err := b.actions.HoldWorkOrder(ctx, workOrderID, alert.Message); err != nil {
		return IngestVerdict{Safe: false, Alert: &alert}, fmt.Errorf("telemetry: work order %s safety hold failed: %w", workOrderID, err)
	}
	if err := b.actions.EmitAlert(ctx, alert); err != nil {
		return IngestVerdict{Safe: false, Alert: &alert}, fmt.Errorf("telemetry: critical alert emission failed: %w", err)
	}
	return IngestVerdict{Safe: false, Alert: &alert}, nil
}

// clockFor returns (creating on first sight) the per-crane ClockGuard. The
// map write path is mutex-guarded so concurrent streams never race (Hazard
// 26); guards themselves are internally synchronized.
func (b *SCADABridge) clockFor(craneID string) *timeguard.ClockGuard {
	b.mu.Lock()
	defer b.mu.Unlock()
	g, ok := b.clocks[craneID]
	if !ok {
		g = timeguard.NewClockGuard(MaxTelemetryClockSkew)
		b.clocks[craneID] = g
	}
	return g
}

func (b *SCADABridge) buildAlert(workOrderID string, data CraneLMIData, ts time.Time) CriticalTelemetryAlert {
	kind := AlertMomentUtilization
	message := fmt.Sprintf("crane %s moment utilization %.1f%% exceeds the statutory 90%% ceiling", data.CraneID, data.MomentUtilizationPct)
	if data.AntiTwoBlockTriggered {
		kind = AlertAntiTwoBlock
		message = fmt.Sprintf("crane %s anti-two-block interlock triggered", data.CraneID)
	}
	alertID, err := id.NewV7()
	if err != nil {
		// Clock tampering is already ruled out before this point; fall back
		// to the frame timestamp as the alert identifier rather than silently
		// dropping the alert (Hazard 32).
		alertID = ts.Format("20060102T150405.000Z")
	}
	return CriticalTelemetryAlert{
		AlertID:              alertID,
		WorkOrderID:          workOrderID,
		CraneID:              data.CraneID,
		Timestamp:            ts,
		Kind:                 kind,
		MomentUtilizationPct: data.MomentUtilizationPct,
		Message:              message,
	}
}
