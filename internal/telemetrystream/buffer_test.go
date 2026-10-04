package telemetrystream

import (
	"context"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestStreamBuffer_IngestAndFlush(t *testing.T) {
	cfg := IngestConfig{
		RingBufferSize:   1000,
		BatchFlushSize:   50,
		FlushInterval:    10 * time.Millisecond,
		RatedCapacityMax: 300.0,
	}

	buf := NewStreamBuffer(nil, cfg) // nil DB for in-memory batch drain testing
	defer buf.Close()

	now := time.Now().UTC()
	for i := 0; i < 75; i++ {
		err := buf.Ingest(TelemetryReading{
			TenantID:         "tenant-aramco",
			OrganizationID:   "org-tic-middleeast",
			SensorID:         "load-cell-rig-44",
			ReadingTimestamp: now.Add(time.Duration(i) * time.Millisecond),
			ReadingType:      "LOAD_CELL_FORCE",
			Value:            150.0 + rand.Float64()*10.0,
		})
		if err != nil {
			t.Fatalf("unexpected ingest error at %d: %v", i, err)
		}
	}

	if buf.Size() != 75 {
		t.Errorf("expected buffer size 75, got: %d", buf.Size())
	}

	// First flush drains batch of 50
	drained, err := buf.FlushBatch(context.Background(), "tenant-aramco", "org-tic-middleeast")
	if err != nil {
		t.Fatalf("unexpected flush error: %v", err)
	}
	if drained != 50 {
		t.Errorf("expected 50 drained items, got: %d", drained)
	}
	if buf.Size() != 25 {
		t.Errorf("expected remaining buffer size 25, got: %d", buf.Size())
	}

	// Second flush drains remaining 25
	drained, err = buf.FlushBatch(context.Background(), "tenant-aramco", "org-tic-middleeast")
	if err != nil {
		t.Fatalf("unexpected second flush error: %v", err)
	}
	if drained != 25 {
		t.Errorf("expected 25 drained items, got: %d", drained)
	}
	if buf.Size() != 0 {
		t.Errorf("expected buffer empty, got: %d", buf.Size())
	}
}

func TestStreamBuffer_SafetyOverloadLockout(t *testing.T) {
	cfg := IngestConfig{
		RingBufferSize:   100,
		BatchFlushSize:   50,
		RatedCapacityMax: 200.0, // 200 Tonnes max
	}

	buf := NewStreamBuffer(nil, cfg)
	defer buf.Close()

	// Safe reading
	err := buf.Ingest(TelemetryReading{
		SensorID: "sensor-01",
		Value:    185.0,
	})
	if err != nil {
		t.Fatalf("expected safe reading accepted, got: %v", err)
	}

	// Overload reading (240T > 200T)
	err = buf.Ingest(TelemetryReading{
		SensorID: "sensor-01",
		Value:    240.0,
	})
	if err == nil || !strings.Contains(err.Error(), "critical physical load capacity exceeded") {
		t.Fatalf("expected critical overload rejection, got: %v", err)
	}
}

func BenchmarkStreamBuffer_IngestHotPath(b *testing.B) {
	cfg := IngestConfig{
		RingBufferSize:   1000000,
		BatchFlushSize:   1000,
		RatedCapacityMax: 1000.0,
	}

	buf := NewStreamBuffer(nil, cfg)
	defer buf.Close()

	reading := TelemetryReading{
		TenantID:         "t1",
		OrganizationID:   "o1",
		SensorID:         "s1",
		ReadingTimestamp: time.Now().UTC(),
		ReadingType:      "LOAD",
		Value:            250.0,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if buf.Size() >= cfg.RingBufferSize-10 {
			_, _ = buf.FlushBatch(context.Background(), "t1", "o1")
		}
		if err := buf.Ingest(reading); err != nil {
			b.Fatal(err)
		}
	}
}
