package telemetrystreamhttp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"integin/internal/telemetrystream"
	"integin/internal/telemetrystreamhttp"
)

func TestTelemetryStreamHTTP_ServeHTTP(t *testing.T) {
	cfg := telemetrystream.IngestConfig{
		RingBufferSize:   100,
		BatchFlushSize:   10,
		FlushInterval:    time.Second,
		RatedCapacityMax: 50.0,
	}
	buf := telemetrystream.NewStreamBuffer(nil, cfg)
	handler := telemetrystreamhttp.NewHandler(buf)

	reading := telemetrystream.TelemetryReading{
		TenantID:         "t-1",
		OrganizationID:   "o-1",
		SensorID:         "loadcell-99",
		ReadingTimestamp: time.Now().UTC(),
		ReadingType:      "LOAD_CELL_FORCE",
		Value:            25.5,
	}
	raw, _ := json.Marshal(reading)

	// 1. Success
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/stream", bytes.NewReader(raw))
	req.Header.Set("X-Tenant-ID", "t-1")
	req.Header.Set("X-Organization-ID", "o-1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Overload lockout
	overloadReading := reading
	overloadReading.Value = 100.0
	rawOverload, _ := json.Marshal(overloadReading)

	reqOverload := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/stream", bytes.NewReader(rawOverload))
	reqOverload.Header.Set("X-Tenant-ID", "t-1")
	reqOverload.Header.Set("X-Organization-ID", "o-1")
	recOverload := httptest.NewRecorder()

	handler.ServeHTTP(recOverload, reqOverload)
	if recOverload.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", recOverload.Code)
	}
}
