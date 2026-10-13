package telemetry

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func validLabels() map[string]string {
	return map[string]string{
		"service":          "integin",
		"environment":      "test",
		"route_template":   "/verify/:method",
		"method":           "GET",
		"http_status_code": "200",
		"operation":        "http_server",
		"outcome":          "success",
		"error_class":      "",
	}
}

func mustWrite(t *testing.T, m *Metrics) string {
	t.Helper()
	var buffer bytes.Buffer
	if err := m.WritePrometheus(&buffer); err != nil {
		t.Fatalf("WritePrometheus: %v", err)
	}
	return buffer.String()
}

func TestMetricsRegistersAllContractNames(t *testing.T) {
	body := mustWrite(t, NewMetrics())
	for _, name := range []string{
		"integin_http_server_requests_total",
		"integin_http_server_request_duration_seconds",
		"integin_sync_transactions_total",
		"integin_evidence_operations_total",
		"integin_authority_validation_total",
		"integin_dependency_health",
		"integin_telemetry_dropped_total",
	} {
		if !strings.Contains(body, "# HELP "+name+" ") {
			t.Errorf("missing # HELP for %s", name)
		}
		if !strings.Contains(body, "# TYPE "+name+" ") {
			t.Errorf("missing # TYPE for %s", name)
		}
	}
}

func TestMetricsRecordingAndPrometheusFormat(t *testing.T) {
	m := NewMetrics()
	m.IncHTTPRequest(validLabels())
	m.IncHTTPRequest(validLabels())
	m.ObserveHTTPRequestDuration(0.007, validLabels())
	m.SetDependencyHealth(1, map[string]string{"service": "integin", "environment": "test", "dependency": "postgres"})
	m.IncSyncTransaction(map[string]string{"service": "integin", "environment": "test", "operation": "pull", "outcome": "success", "error_class": ""})
	m.IncEvidenceOperation(map[string]string{"service": "integin", "environment": "test", "operation": "persist", "outcome": "failure", "error_class": "storage"})
	m.IncAuthorityValidation(map[string]string{"service": "integin", "environment": "test", "operation": "verify", "outcome": "rejected", "error_class": "signature"})

	body := mustWrite(t, m)
	for _, want := range []string{
		`integin_http_server_requests_total{service="integin",environment="test",route_template="/verify/:method",method="GET",http_status_code="200",operation="http_server",outcome="success",error_class=""} 2`,
		`integin_dependency_health{service="integin",environment="test",dependency="postgres"} 1`,
		`integin_sync_transactions_total{service="integin",environment="test",operation="pull",outcome="success",error_class=""} 1`,
		`integin_evidence_operations_total{service="integin",environment="test",operation="persist",outcome="failure",error_class="storage"} 1`,
		`integin_authority_validation_total{service="integin",environment="test",operation="verify",outcome="rejected",error_class="signature"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("exposition missing %q\nfull body:\n%s", want, body)
		}
	}
	if !strings.Contains(body, `integin_http_server_request_duration_seconds_bucket{service="integin",environment="test",route_template="/verify/:method",method="GET",http_status_code="200",operation="http_server",outcome="success",error_class="",le="0.005"} 0`) {
		t.Errorf("histogram no bucket over 0.005\n%s", body)
	}
	if !strings.Contains(body, `integin_http_server_request_duration_seconds_bucket{service="integin",environment="test",route_template="/verify/:method",method="GET",http_status_code="200",operation="http_server",outcome="success",error_class="",le="0.01"} 1`) {
		t.Errorf("histogram bucket 0.01 missing\n%s", body)
	}
	if !strings.Contains(body, `integin_http_server_request_duration_seconds_bucket{service="integin",environment="test",route_template="/verify/:method",method="GET",http_status_code="200",operation="http_server",outcome="success",error_class="",le="+Inf"} 1`) {
		t.Errorf("histogram +Inf bucket missing\n%s", body)
	}
	if !strings.Contains(body, `integin_http_server_request_duration_seconds_sum{service="integin",environment="test",route_template="/verify/:method",method="GET",http_status_code="200",operation="http_server",outcome="success",error_class=""} 0.007`) {
		t.Errorf("histogram sum missing\n%s", body)
	}
	if !strings.Contains(body, `integin_http_server_request_duration_seconds_count{service="integin",environment="test",route_template="/verify/:method",method="GET",http_status_code="200",operation="http_server",outcome="success",error_class=""} 1`) {
		t.Errorf("histogram count missing\n%s", body)
	}
}

func TestLabelValueEscaping(t *testing.T) {
	m := NewMetrics()
	labels := validLabels()
	labels["route_template"] = `a"b\c` + "\n"
	m.IncHTTPRequest(labels)
	body := mustWrite(t, m)
	if !strings.Contains(body, `route_template="a\"b\\c\n"`) {
		t.Errorf("label value not escaped per Prometheus spec:\n%s", body)
	}
}

func TestUnknownMetricNameRejected(t *testing.T) {
	m := NewMetrics()
	if err := m.IncCounter("integin_no_such_metric", validLabels()); err == nil {
		t.Fatal("expected error for unknown metric name")
	}
}

func TestDependencyHealthGaugeOverwrites(t *testing.T) {
	m := NewMetrics()
	labels := map[string]string{"service": "integin", "environment": "test", "dependency": "postgres"}
	m.SetDependencyHealth(1, labels)
	m.SetDependencyHealth(0, labels)
	body := mustWrite(t, m)
	want := `integin_dependency_health{service="integin",environment="test",dependency="postgres"} 0`
	if !strings.Contains(body, want) {
		t.Errorf("gauge overwrite failed, want %q in:\n%s", want, body)
	}
	if strings.Contains(body, `dependency="postgres"} 1`) {
		t.Errorf("gauge stale series retained:\n%s", body)
	}
}

func TestNegativeProhibitedDimensionsDroppedAndCounted(t *testing.T) {
	m := NewMetrics()
	for _, bad := range []map[string]string{
		{"tenant_id": "tenant-1"},
		{"evidence_id": "ev-1", "route_template": "/evidence"},
		{"token": "secret", "http_status_code": "200"}, // non-allowed key, silently dropped + counted
		{"service": "integin", "method": "GET", "user_id": "u-1", "outcome": "success"},
	} {
		// Only keys allowed by the contract are exercised; each map is merged
		// over a valid base so the offending key is the only injectable.
		full := validLabels()
		for key, value := range bad {
			full[key] = value
		}
		m.IncHTTPRequest(full)
	}

	if dropped := m.DroppedCount(); dropped != 4 {
		t.Fatalf("DroppedCount() = %d, want 4 (one per offending observation)", dropped)
	}
	body := mustWrite(t, m)
	if strings.Contains(body, `tenant_id=`) || strings.Contains(body, `user_id=`) ||
		strings.Contains(body, `evidence_id=`) || strings.Contains(body, `token=`) {
		t.Errorf("prohibited dimensions leaked into exposition:\n%s", body)
	}
	if !strings.Contains(body, `integin_telemetry_dropped_total 4`) {
		t.Errorf("dropped counter not incremented:\n%s", body)
	}
	// The remaining valid labels must still surface so dropping is lossy only
	// for the offending dimension, not the whole series.
	if !strings.Contains(body, `integin_http_server_requests_total{service="integin",environment="test"`) {
		t.Errorf("valid labels lost after drop:\n%s", body)
	}
}

func TestNegativeEvidenceBytesAndTokensStillRecordedAsValidSeries(t *testing.T) {
	m := NewMetrics()
	m.IncEvidenceOperation(map[string]string{
		"service": "integin", "environment": "test", "operation": "persist",
		"outcome": "failure", "error_class": "storage",
		"evidence_bytes": "1048576",
	})
	if dropped := m.DroppedCount(); dropped != 1 {
		t.Fatalf("DroppedCount() = %d, want 1", dropped)
	}
	body := mustWrite(t, m)
	if strings.Contains(body, "evidence_bytes") {
		t.Errorf("evidence_bytes leaked:\n%s", body)
	}
	if !strings.Contains(body, `integin_evidence_operations_total{service="integin",environment="test",operation="persist",outcome="failure",error_class="storage"} 1`) {
		t.Errorf("dropped label did not preserve valid series:\n%s", body)
	}
}

func TestMetricsHandlerContentTypeAndMethod(t *testing.T) {
	m := NewMetrics()
	handler := m.Handler()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /metrics status = %d", recorder.Code)
	}
	if !strings.Contains(recorder.Header().Get("Content-Type"), "text/plain") {
		t.Errorf("Content-Type = %q", recorder.Header().Get("Content-Type"))
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/metrics", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /metrics status = %d, want 405", recorder.Code)
	}
}

func TestMetricsConcurrencyRace(t *testing.T) {
	m := NewMetrics()
	labels := validLabels()
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				m.IncHTTPRequest(labels)
				m.ObserveHTTPRequestDuration(0.005, labels)
				m.IncSyncTransaction(map[string]string{
					"service": "integin", "environment": "test",
					"operation": "pull", "outcome": "success", "error_class": "",
				})
				m.SetDependencyHealth(1, map[string]string{
					"service": "integin", "environment": "test", "dependency": "postgres",
				})
				m.IncEvidenceOperation(map[string]string{
					"service": "integin", "environment": "test", "operation": "persist",
					"outcome": "success", "error_class": "", "tenant_id": "x",
				})
				_ = mustWrite(t, m)
			}
		}()
	}
	wg.Wait()
	if got := m.DroppedCount(); got != 32*200 {
		t.Fatalf("DroppedCount() = %d, want %d", got, 32*200)
	}
	body := mustWrite(t, m)
	if !strings.Contains(body, `integin_http_server_requests_total{service="integin",environment="test"`) {
		t.Errorf("concurrent requests counter missing:\n%s", body)
	}
	if strings.Contains(body, `tenant_id=`) {
		t.Errorf("tenant_id leaked under concurrency:\n%s", body)
	}
}

func TestTraceParentParseRoundTrip(t *testing.T) {
	tc, err := NewTraceContext()
	if err != nil {
		t.Fatalf("NewTraceContext: %v", err)
	}
	parsed, err := ParseTraceParent(tc.String())
	if err != nil {
		t.Fatalf("ParseTraceParent: %v", err)
	}
	if parsed.TraceID != tc.TraceID || parsed.SpanID != tc.SpanID || !parsed.Sampled {
		t.Fatalf("round trip mismatch: %+v vs %+v", parsed, tc)
	}

	unsampled, err := ParseTraceParent("00-" + tc.TraceID + "-" + tc.SpanID + "-00")
	if err != nil {
		t.Fatal(err)
	}
	if unsampled.Sampled {
		t.Fatal("flags 00 must be unsampled")
	}
}

func TestTraceParentRejectsMalformed(t *testing.T) {
	for _, value := range []string{
		"", "00-cafebabe", "00-" + strings.Repeat("0", 32) + "-" + strings.Repeat("a", 16) + "-01",
		"ff-" + strings.Repeat("a", 32) + "-" + strings.Repeat("b", 16) + "-01",
		"00-" + strings.Repeat("a", 32) + "-" + strings.Repeat("b", 16), // missing flags
		"00-" + strings.Repeat("z", 32) + "-" + strings.Repeat("b", 16) + "-01",
		"00-" + strings.Repeat("a", 32) + "-" + strings.Repeat("0", 16) + "-01", // zero span id
	} {
		if _, err := ParseTraceParent(value); err == nil {
			t.Errorf("ParseTraceParent(%q) accepted malformed header", value)
		}
	}
}

func TestTraceContextPropagation(t *testing.T) {
	parent := context.Background()
	if _, ok := TraceContextFromContext(parent); ok {
		t.Fatal("background context must not carry a trace context")
	}
	tc, err := NewTraceContext()
	if err != nil {
		t.Fatal(err)
	}
	child := WithTraceContext(parent, tc)
	got, ok := TraceContextFromContext(child)
	if !ok || got != tc {
		t.Fatalf("propagation failed: ok=%v got=%+v", ok, got)
	}
}

func TestTraceSpanChildGeneration(t *testing.T) {
	tc, err := NewTraceContext()
	if err != nil {
		t.Fatal(err)
	}
	child, err := tc.WithNewSpan()
	if err != nil {
		t.Fatal(err)
	}
	if child.TraceID != tc.TraceID {
		t.Fatal("child span must preserve trace id")
	}
	if child.SpanID == tc.SpanID {
		t.Fatal("child span id must differ")
	}
}

func TestSafeTraceAttributesFiltersProhibited(t *testing.T) {
	raw := map[string]string{
		"service.name":              "integin",
		"http.route":                "/verify/:method",
		"http.response.status_code": "200",
		"integin.tenant_id":         "tenant-1",
		"integin.evidence_bytes":    "1048576",
		"integin.token":             "secret",
		"exception.message":         "boom",
		"db.statement":              "SELECT * FROM evidence",
		"enduser.id":                "u-1",
		"totally.unknown.attribute": "dropped",
		"integin.operation":         "verify",
		"integin.outcome":           "success",
		"error.type":                "",
		"http.request.method":       "GET",
	}
	safe := SafeTraceAttributes(raw)
	for _, keep := range []string{"service.name", "http.route", "http.response.status_code", "integin.operation", "integin.outcome", "error.type", "http.request.method"} {
		if _, ok := safe[keep]; !ok {
			t.Errorf("allowed attribute %q dropped", keep)
		}
	}
	for _, drop := range []string{"integin.tenant_id", "integin.evidence_bytes", "integin.token", "exception.message", "db.statement", "enduser.id", "totally.unknown.attribute"} {
		if _, ok := safe[drop]; ok {
			t.Errorf("prohibited attribute %q leaked", drop)
		}
	}
	if len(safe) != 7 {
		t.Errorf("SafeTraceAttributes returned %d keys, want 7 allowed keys", len(safe))
	}
	// The caller's map must not be mutated.
	if _, ok := raw["integin.token"]; !ok {
		t.Fatal("source map was mutated")
	}
}
