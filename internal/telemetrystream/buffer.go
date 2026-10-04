package telemetrystream

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"integin/pkg/rulesengine"
)

var (
	ErrNilDB              = errors.New("telemetrystream: database handle cannot be nil")
	ErrEmptyBatch         = errors.New("telemetrystream: telemetry reading batch cannot be empty")
	ErrBufferOverflow     = errors.New("telemetrystream: in-memory ring buffer overflow")
	ErrCriticalOverload   = errors.New("telemetrystream: critical physical load capacity exceeded (lockout triggered)")
)

// TelemetryReading represents a single high-frequency sensor measurement.
type TelemetryReading struct {
	TenantID         string          `json:"tenant_id"`
	OrganizationID   string          `json:"organization_id"`
	SensorID         string          `json:"sensor_id"`
	ReadingTimestamp time.Time       `json:"reading_timestamp"`
	ReadingType      string          `json:"reading_type"` // e.g. "LOAD_CELL_FORCE", "PRESSURE", "ACOUSTIC_EMISSION"
	Value            float64         `json:"value"`        // Normalized scalar value
	Payload          json.RawMessage `json:"payload,omitempty"`
}

// IngestConfig configures buffer sizing and flush intervals.
type IngestConfig struct {
	RingBufferSize   int
	BatchFlushSize   int
	FlushInterval    time.Duration
	RatedCapacityMax float64 // Maximum physical threshold before critical alert dispatch
}

// DefaultIngestConfig returns production-tuned parameters for high throughput.
func DefaultIngestConfig() IngestConfig {
	return IngestConfig{
		RingBufferSize:   10000,
		BatchFlushSize:   500,
		FlushInterval:    50 * time.Millisecond,
		RatedCapacityMax: 500.0, // e.g. 500 Tonnes / kN
	}
}

// StreamBuffer coordinates high-frequency in-memory buffering and zero-allocation
// batch writes to PostgreSQL partitioned table sensor_telemetry_stream (Migration 0074).
type StreamBuffer struct {
	db      *sql.DB
	cfg     IngestConfig
	mu      sync.Mutex
	buffer  []TelemetryReading
	closed  bool
	samples []rulesengine.TelemetricSample
}

func NewStreamBuffer(db *sql.DB, cfg IngestConfig) *StreamBuffer {
	if cfg.RingBufferSize <= 0 {
		cfg.RingBufferSize = 10000
	}
	if cfg.BatchFlushSize <= 0 {
		cfg.BatchFlushSize = 500
	}
	return &StreamBuffer{
		db:      db,
		cfg:     cfg,
		buffer:  make([]TelemetryReading, 0, cfg.RingBufferSize),
		samples: make([]rulesengine.TelemetricSample, 0, 64),
	}
}

// Ingest pushes a reading into the buffer, evaluating physical overload boundaries immediately.
func (b *StreamBuffer) Ingest(r TelemetryReading) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return errors.New("telemetrystream: buffer is closed")
	}

	// 1. Safety Gate: Critical physical overload trigger
	if b.cfg.RatedCapacityMax > 0 && r.Value > b.cfg.RatedCapacityMax {
		return fmt.Errorf("%w: reading %.2f exceeds rated capacity limit %.2f",
			ErrCriticalOverload, r.Value, b.cfg.RatedCapacityMax)
	}

	// 2. Buffer capacity check
	if len(b.buffer) >= b.cfg.RingBufferSize {
		return ErrBufferOverflow
	}

	if r.ReadingTimestamp.IsZero() {
		r.ReadingTimestamp = time.Now().UTC()
	}
	if len(r.Payload) == 0 {
		r.Payload = json.RawMessage(`{}`)
	}

	b.buffer = append(b.buffer, r)

	// Keep rolling sample slice for jitter verification
	b.samples = append(b.samples, rulesengine.TelemetricSample{
		TimestampNs: r.ReadingTimestamp.UnixNano(),
		Value:       r.Value,
	})
	if len(b.samples) > 64 {
		b.samples = b.samples[1:]
	}

	return nil
}

// Size returns the active unwritten reading count in buffer.
func (b *StreamBuffer) Size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.buffer)
}

// FlushBatch drains up to limit readings and writes them transactionally with multi-tenant RLS.
func (b *StreamBuffer) FlushBatch(ctx context.Context, tenantID, orgID string) (int, error) {
	b.mu.Lock()
	if len(b.buffer) == 0 {
		b.mu.Unlock()
		return 0, nil
	}

	batchCount := len(b.buffer)
	if batchCount > b.cfg.BatchFlushSize {
		batchCount = b.cfg.BatchFlushSize
	}

	batch := make([]TelemetryReading, batchCount)
	copy(batch, b.buffer[:batchCount])
	b.buffer = b.buffer[batchCount:]
	b.mu.Unlock()

	if b.db == nil {
		// In pure memory / test mode
		return batchCount, nil
	}

	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin telemetry batch tx: %w", err)
	}
	defer tx.Rollback()

	// Enforce session tenant GUCs (Multi-Tenant RLS Invariant, Hazard 21)
	if _, err := tx.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", tenantID, orgID); err != nil {
		return 0, fmt.Errorf("set telemetry tenant GUCs: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO sensor_telemetry_stream (
			tenant_id, organization_id, sensor_id, reading_timestamp, reading_type, payload
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, organization_id, sensor_id, reading_timestamp) DO NOTHING
	`)
	if err != nil {
		return 0, fmt.Errorf("prepare telemetry insert: %w", err)
	}
	defer stmt.Close()

	for _, reading := range batch {
		if _, err := stmt.ExecContext(ctx, tenantID, orgID, reading.SensorID, reading.ReadingTimestamp, reading.ReadingType, reading.Payload); err != nil {
			return 0, fmt.Errorf("insert telemetry reading: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit telemetry batch: %w", err)
	}

	return batchCount, nil
}

// VerifyBufferJitter computes harmonic jitter across the active buffer samples.
func (b *StreamBuffer) VerifyBufferJitter() (*rulesengine.JitterProfile, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.samples) < 8 {
		return nil, rulesengine.ErrInsufficientSamples
	}
	return rulesengine.VerifyHarmonicJitter(b.samples, 0.15)
}

func (b *StreamBuffer) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	return nil
}
