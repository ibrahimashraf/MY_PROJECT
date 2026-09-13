package telemetry

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"integin/pkg/id"
)

// recorder captures safety actions deterministically across goroutines.
type recorder struct {
	mu     sync.Mutex
	holds  []string
	alerts []CriticalTelemetryAlert
}

func (r *recorder) hold(ctx context.Context, workOrderID, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.holds = append(r.holds, workOrderID+":"+reason)
	return nil
}

func (r *recorder) emit(ctx context.Context, alert CriticalTelemetryAlert) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.alerts = append(r.alerts, alert)
	return nil
}

func (r *recorder) holdsCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.holds)
}

func (r *recorder) alertsCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.alerts)
}

func mustBridge(t *testing.T, rec *recorder) *SCADABridge {
	t.Helper()
	b, err := NewSCADABridge(SafetyActions{HoldWorkOrder: rec.hold, EmitAlert: rec.emit})
	if err != nil {
		t.Fatalf("NewSCADABridge: %v", err)
	}
	return b
}

// viableFrame builds a physically valid, gate-safe frame at the given time.
func viableFrame(craneID string, at time.Time, util float64, antiTwoBlock bool) CraneLMIData {
	ts, err := id.NewV7FromTime(at)
	if err != nil {
		panic(err)
	}
	return CraneLMIData{
		CraneID:               craneID,
		TimestampUUIDv7:       ts,
		BoomLengthM:           40.0,
		BoomAngleDeg:          45.0,
		WorkingRadiusM:        28.3,
		ActualLoadTonnes:      8.0,
		RatedCapacityTonnes:   10.0,
		MomentUtilizationPct:  util,
		OutriggerPressuresKpa: []float64{120, 118, 119, 121},
		WindSpeedMps:          6.2,
		AntiTwoBlockTriggered: antiTwoBlock,
	}
}

func TestNewSCADABridgeRequiresBothSafetyActions(t *testing.T) {
	if _, err := NewSCADABridge(SafetyActions{HoldWorkOrder: func(ctx context.Context, s, s2 string) error { return nil }}); err == nil {
		t.Fatal("bridge accepted missing EmitAlert")
	}
	if _, err := NewSCADABridge(SafetyActions{EmitAlert: func(ctx context.Context, a CriticalTelemetryAlert) error { return nil }}); err == nil {
		t.Fatal("bridge accepted missing HoldWorkOrder")
	}
}

func TestIngestSafeFramePassesWithoutSideEffects(t *testing.T) {
	rec := &recorder{}
	b := mustBridge(t, rec)
	now := time.Now().UTC()
	verdict, err := b.IngestTelemetry(context.Background(), "wo-1", viableFrame("crane-a", now.Add(-time.Second), 80.0, false))
	if err != nil {
		t.Fatalf("safe frame rejected: %v", err)
	}
	if !verdict.Safe || verdict.Alert != nil {
		t.Fatalf("safe frame returned verdict %+v", verdict)
	}
	if rec.holdsCount() != 0 || rec.alertsCount() != 0 {
		t.Fatalf("safe frame emitted %d holds / %d alerts", rec.holdsCount(), rec.alertsCount())
	}
}

func TestIngestExactStatutoryBoundaryIsSafe(t *testing.T) {
	rec := &recorder{}
	b := mustBridge(t, rec)
	now := time.Now().UTC()
	// 90.0% is the statutory ceiling: equal is within limits.
	verdict, err := b.IngestTelemetry(context.Background(), "wo-1", viableFrame("crane-b", now, 90.0, false))
	if err != nil {
		t.Fatalf("boundary frame rejected: %v", err)
	}
	if !verdict.Safe {
		t.Fatalf("90.0%% utilization falsely tripped the gate")
	}
}

func TestIngestMomentUtilizationAlertTriggersHoldAndAlert(t *testing.T) {
	rec := &recorder{}
	b := mustBridge(t, rec)
	now := time.Now().UTC()
	verdict, err := b.IngestTelemetry(context.Background(), "wo-77", viableFrame("crane-c", now, 91.3, false))
	if err != nil {
		t.Fatalf("overload frame errored: %v", err)
	}
	if verdict.Safe || verdict.Alert == nil {
		t.Fatalf("overload frame not flagged: %+v", verdict)
	}
	if verdict.Alert.Kind != AlertMomentUtilization {
		t.Fatalf("wrong alert kind: %s", verdict.Alert.Kind)
	}
	if verdict.Alert.WorkOrderID != "wo-77" || verdict.Alert.CraneID != "crane-c" {
		t.Fatalf("alert not bound to work order/crane: %+v", verdict.Alert)
	}

	if rec.holdsCount() != 1 {
		t.Fatalf("expected 1 safety hold, got %d", rec.holdsCount())
	}
	if rec.alertsCount() != 1 {
		t.Fatalf("expected 1 alert, got %d", rec.alertsCount())
	}
	hold := rec.holds[0]
	if !strings.HasPrefix(hold, "wo-77:") || !strings.Contains(hold, "crane-c") {
		t.Fatalf("hold not bound to work order with reason: %q", hold)
	}
}

func TestIngestAntiTwoBlockAlertTakesPriority(t *testing.T) {
	rec := &recorder{}
	b := mustBridge(t, rec)
	now := time.Now().UTC()
	// Both violations present: the interlock latch is the priority class.
	verdict, err := b.IngestTelemetry(context.Background(), "wo-9", viableFrame("crane-d", now, 97.0, true))
	if err != nil {
		t.Fatalf("frame errored: %v", err)
	}
	if verdict.Alert == nil || verdict.Alert.Kind != AlertAntiTwoBlock {
		t.Fatalf("anti-two-block not prioritized: %+v", verdict.Alert)
	}
	if !strings.Contains(verdict.Alert.Message, "anti-two-block") {
		t.Fatalf("unexpected message: %q", verdict.Alert.Message)
	}
}

func TestIngestRejectsMalformedAndNonFiniteFrames(t *testing.T) {
	rec := &recorder{}
	b := mustBridge(t, rec)
	now := time.Now().UTC()
	base := viableFrame("crane-e", now, 50.0, false)

	cases := []struct {
		name string
		mut  func(*CraneLMIData)
	}{
		{"empty crane id", func(d *CraneLMIData) { d.CraneID = " " }},
		{"non-uuidv7 timestamp", func(d *CraneLMIData) { d.TimestampUUIDv7 = "not-a-uuid" }},
		{"nan utilization", func(d *CraneLMIData) { d.MomentUtilizationPct = math.NaN() }},
		{"inf capacity", func(d *CraneLMIData) { d.RatedCapacityTonnes = math.Inf(1) }},
		{"negative boom length", func(d *CraneLMIData) { d.BoomLengthM = -1 }},
		{"negative radius", func(d *CraneLMIData) { d.WorkingRadiusM = -0.5 }},
		{"negative load", func(d *CraneLMIData) { d.ActualLoadTonnes = -2 }},
		{"zero capacity", func(d *CraneLMIData) { d.RatedCapacityTonnes = 0 }},
		{"negative utilization", func(d *CraneLMIData) { d.MomentUtilizationPct = -5 }},
		{"nan outrigger pressure", func(d *CraneLMIData) { d.OutriggerPressuresKpa[2] = math.NaN() }},
		{"negative wind", func(d *CraneLMIData) { d.WindSpeedMps = -1 }},
		{"boom angle out of physical range", func(d *CraneLMIData) { d.BoomAngleDeg = 270 }},
	}
	for _, tc := range cases {
		bad := base
		tc.mut(&bad)
		if _, err := b.IngestTelemetry(context.Background(), "wo-1", bad); err == nil {
			t.Fatalf("%s: malformed frame accepted", tc.name)
		}
	}
	if rec.holdsCount() != 0 || rec.alertsCount() != 0 {
		t.Fatalf("malformed frames triggered safety side effects")
	}
}

func TestIngestRejectsClockBackwardsDrift(t *testing.T) {
	rec := &recorder{}
	b := mustBridge(t, rec)
	now := time.Now().UTC()
	if _, err := b.IngestTelemetry(context.Background(), "wo-1", viableFrame("crane-f", now.Add(-time.Second), 50, false)); err != nil {
		t.Fatalf("anchor frame rejected: %v", err)
	}
	// Wall clock steps back 30s (> 10s tolerance): tamper alert.
	if _, err := b.IngestTelemetry(context.Background(), "wo-1", viableFrame("crane-f", now.Add(-31*time.Second), 50, false)); err == nil {
		t.Fatal("backwards wall-clock drift accepted")
	}
}

func TestIngestRejectsSmallMonotonicRollback(t *testing.T) {
	rec := &recorder{}
	b := mustBridge(t, rec)
	now := time.Now().UTC()
	if _, err := b.IngestTelemetry(context.Background(), "wo-1", viableFrame("crane-g", now, 50, false)); err != nil {
		t.Fatalf("anchor frame rejected: %v", err)
	}
	// A 5s rollback is within wall tolerance but violates the monotonic
	// UUIDv7 sequence: must be rejected by the timeguard monotonic check.
	if _, err := b.IngestTelemetry(context.Background(), "wo-1", viableFrame("crane-g", now.Add(-5*time.Second), 50, false)); err == nil {
		t.Fatal("monotonic rollback accepted")
	}
}

func TestIngestSafetyHoldFailureFailsClosed(t *testing.T) {
	failHold := &recorder{}
	b, err := NewSCADABridge(SafetyActions{
		HoldWorkOrder: func(ctx context.Context, workOrderID, reason string) error {
			return errors.New("work order service unavailable")
		},
		EmitAlert: failHold.emit,
	})
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}
	now := time.Now().UTC()
	verdict, err := b.IngestTelemetry(context.Background(), "wo-1", viableFrame("crane-h", now, 95, false))
	if err == nil {
		t.Fatal("safety hold failure was swallowed")
	}
	if verdict.Safe {
		t.Fatal("hold failure reported as safe")
	}
	if verdict.Alert == nil {
		t.Fatal("alert missing on hold failure (must surface intended state)")
	}
}

func TestConcurrentStreamsAcrossCranes(t *testing.T) {
	rec := &recorder{}
	b := mustBridge(t, rec)
	now := time.Now().UTC()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(crane int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				at := now.Add(time.Duration(j) * time.Second)
				_, err := b.IngestTelemetry(context.Background(),
					"wo-1", viableFrame("crane-"+string(rune('a'+crane)), at, 70, false))
				if err != nil {
					t.Errorf("crane %d frame %d: %v", crane, j, err)
				}
			}
		}(i)
	}
	wg.Wait()

	if rec.holdsCount() != 0 || rec.alertsCount() != 0 {
		t.Fatalf("concurrent safe streams emitted safety side effects")
	}
}
