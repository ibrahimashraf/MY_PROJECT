package telemetry

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Contract metric names (observability_foundation_v1.json metric_contract.metric_names).
const (
	HTTPRequestsTotal          = "integin_http_server_requests_total"
	HTTPRequestDurationSeconds = "integin_http_server_request_duration_seconds"
	SyncTransactionsTotal      = "integin_sync_transactions_total"
	EvidenceOperationsTotal    = "integin_evidence_operations_total"
	AuthorityValidationTotal   = "integin_authority_validation_total"
	DependencyHealth           = "integin_dependency_health"
	TelemetryDroppedTotal      = "integin_telemetry_dropped_total"
)

var (
	// allowedDimensions is the closed set of label keys observable metrics may
	// carry (metric_contract.allowed_dimensions).
	allowedDimensions = strSet(
		"service", "environment", "route_template", "method",
		"http_status_code", "operation", "outcome", "dependency", "error_class",
	)
	// prohibitedDimensions is the explicit negative list
	// (metric_contract.prohibited_dimensions).
	prohibitedDimensions = strSet(
		"tenant_id", "organization_id", "user_id", "device_id",
		"authority_id", "resource_id", "evidence_id", "object_key",
		"payload_hash", "request_id", "exception_message",
	)
	// durationBuckets are histogram upper bounds in seconds.
	durationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
)

func strSet(values ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

// IsAllowedDimension reports whether a label key is permitted by the contract.
func IsAllowedDimension(key string) bool {
	_, ok := allowedDimensions[key]
	return ok
}

// IsProhibitedDimension reports whether a label key is on the contract
// negative list. Every prohibited dimension is also non-allowed; callers that
// want the strictest gate should use IsAllowedDimension.
func IsProhibitedDimension(key string) bool {
	_, ok := prohibitedDimensions[key]
	return ok
}

type metricKind uint8

const (
	kindCounter metricKind = iota
	kindGauge
	kindHistogram
)

type metricDef struct {
	name   string
	help   string
	kind   metricKind
	labels []string
}

type metricSeries struct {
	labels map[string]string
	value  float64
}

type histogram struct {
	labels     map[string]string
	boundaries []float64
	counts     []uint64
	sum        float64
	count      uint64
}

func (h *histogram) observe(value float64) {
	if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return
	}
	h.count++
	h.sum += value
	for i, bound := range h.boundaries {
		if value <= bound {
			h.counts[i]++
		}
	}
}

// Metrics is a thread-safe, dependency-free Prometheus text-format registry
// implementing only the observability_foundation_v1.json metric contract.
type Metrics struct {
	mu         sync.RWMutex
	defs       map[string]*metricDef
	counters   map[string]map[string]*metricSeries
	gauges     map[string]map[string]*metricSeries
	histograms map[string]map[string]*histogram
}

// NewMetrics constructs an empty contract registry. The singleton used by the
// server is DefaultMetrics.
func NewMetrics() *Metrics {
	m := &Metrics{
		defs: map[string]*metricDef{
			HTTPRequestsTotal: {
				name: HTTPRequestsTotal,
				help: "Total HTTP server requests handled by route template and outcome.",
				kind: kindCounter,
				labels: []string{
					"service", "environment", "route_template", "method",
					"http_status_code", "operation", "outcome", "error_class",
				},
			},
			HTTPRequestDurationSeconds: {
				name: HTTPRequestDurationSeconds,
				help: "HTTP server request duration in seconds.",
				kind: kindHistogram,
				labels: []string{
					"service", "environment", "route_template", "method",
					"http_status_code", "operation", "outcome", "error_class",
				},
			},
			SyncTransactionsTotal: {
				name:   SyncTransactionsTotal,
				help:   "Total sync transactions processed by outcome.",
				kind:   kindCounter,
				labels: []string{"service", "environment", "operation", "outcome", "error_class"},
			},
			EvidenceOperationsTotal: {
				name:   EvidenceOperationsTotal,
				help:   "Total evidence operations by outcome.",
				kind:   kindCounter,
				labels: []string{"service", "environment", "operation", "outcome", "error_class"},
			},
			AuthorityValidationTotal: {
				name:   AuthorityValidationTotal,
				help:   "Total authority signature validations by outcome.",
				kind:   kindCounter,
				labels: []string{"service", "environment", "operation", "outcome", "error_class"},
			},
			DependencyHealth: {
				name:   DependencyHealth,
				help:   "Dependency health (1 healthy, 0 degraded).",
				kind:   kindGauge,
				labels: []string{"service", "environment", "dependency"},
			},
			TelemetryDroppedTotal: {
				name:   TelemetryDroppedTotal,
				help:   "Total observations dropped for prohibited or non-allowed dimensions.",
				kind:   kindCounter,
				labels: nil,
			},
		},
		counters:   make(map[string]map[string]*metricSeries),
		gauges:     make(map[string]map[string]*metricSeries),
		histograms: make(map[string]map[string]*histogram),
	}
	for name, def := range m.defs {
		switch def.kind {
		case kindCounter:
			m.counters[name] = make(map[string]*metricSeries)
			if name == TelemetryDroppedTotal {
				// Always emit the dropped counter, including the zero case,
				// so the pipeline-silence alert never misfires.
				m.counters[name][""] = &metricSeries{}
			}
		case kindGauge:
			m.gauges[name] = make(map[string]*metricSeries)
		case kindHistogram:
			m.histograms[name] = make(map[string]*histogram)
		}
	}
	return m
}

var defaultMetrics = NewMetrics()

// DefaultMetrics returns the process-wide registry used by the server.
func DefaultMetrics() *Metrics { return defaultMetrics }

// MetricsHandler returns a Prometheus text-format HTTP handler bound to the
// process-wide registry.
func MetricsHandler() http.Handler { return defaultMetrics.Handler() }

// Handler returns a Prometheus text-format HTTP handler for this registry.
func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			writer.Header().Set("Allow", "GET, HEAD")
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writer.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		if err := m.WritePrometheus(writer); err != nil {
			return // client disconnected; nothing further to do
		}
	})
}

// sanitizeLabels keeps only contract-allowed dimensions and reports whether
// any key (prohibited or unknown) was dropped.
func sanitizeLabels(raw map[string]string) (map[string]string, bool) {
	clean := make(map[string]string, len(raw))
	dropped := false
	for key, value := range raw {
		if _, ok := allowedDimensions[key]; !ok {
			dropped = true
			continue
		}
		clean[key] = value
	}
	return clean, dropped
}

// seriesKey builds an unambiguous internal lookup key for a canonical label
// value list. strconv.Quote is injective and every comma between quoted values
// is produced by the Join, so two distinct value lists never collide.
func seriesKey(def *metricDef, values []string) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = strconv.Quote(value)
	}
	return strings.Join(parts, ",")
}

func (m *Metrics) record(def *metricDef, labels map[string]string, amount float64) {
	clean, dropped := sanitizeLabels(labels)
	m.mu.Lock()
	defer m.mu.Unlock()
	if dropped {
		m.incDroppedLocked()
	}
	values := make([]string, len(def.labels))
	labelMap := make(map[string]string, len(clean))
	for i, name := range def.labels {
		values[i] = clean[name]
		labelMap[name] = clean[name]
	}
	key := seriesKey(def, values)
	switch def.kind {
	case kindCounter:
		series := m.counters[def.name][key]
		if series == nil {
			series = &metricSeries{labels: labelMap}
			m.counters[def.name][key] = series
		}
		series.value += amount
	case kindGauge:
		series := m.gauges[def.name][key]
		if series == nil {
			series = &metricSeries{labels: labelMap}
			m.gauges[def.name][key] = series
		}
		series.value = amount
	case kindHistogram:
		h := m.histograms[def.name][key]
		if h == nil {
			h = &histogram{labels: labelMap, boundaries: durationBuckets, counts: make([]uint64, len(durationBuckets))}
			m.histograms[def.name][key] = h
		}
		h.observe(amount)
	}
}

// IncHTTPRequest records one handled HTTP request.
func (m *Metrics) IncHTTPRequest(labels map[string]string) {
	m.record(m.defs[HTTPRequestsTotal], labels, 1)
}

// ObserveHTTPRequestDuration records one HTTP request duration in seconds.
func (m *Metrics) ObserveHTTPRequestDuration(seconds float64, labels map[string]string) {
	m.record(m.defs[HTTPRequestDurationSeconds], labels, seconds)
}

// IncCounter records a 1-increment on any registered counter metric. It
// returns an error for unknown metric names.
func (m *Metrics) IncCounter(name string, labels map[string]string) error {
	def := m.defs[name]
	if def == nil {
		return fmt.Errorf("telemetry: unknown counter %q", name)
	}
	m.record(def, labels, 1)
	return nil
}

// IncSyncTransaction records one sync transaction by outcome.
func (m *Metrics) IncSyncTransaction(labels map[string]string) {
	m.record(m.defs[SyncTransactionsTotal], labels, 1)
}

// IncEvidenceOperation records one evidence operation by outcome.
func (m *Metrics) IncEvidenceOperation(labels map[string]string) {
	m.record(m.defs[EvidenceOperationsTotal], labels, 1)
}

// IncAuthorityValidation records one authority signature validation by outcome.
func (m *Metrics) IncAuthorityValidation(labels map[string]string) {
	m.record(m.defs[AuthorityValidationTotal], labels, 1)
}

// SetDependencyHealth sets dependency health (1 healthy, 0 degraded).
func (m *Metrics) SetDependencyHealth(healthy float64, labels map[string]string) {
	m.record(m.defs[DependencyHealth], labels, healthy)
}

// IncDropped increments the dropped-observation counter. It is used by
// callers that filter non-metric telemetry (e.g. log hooks) at the source.
func (m *Metrics) IncDropped() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.incDroppedLocked()
}

func (m *Metrics) incDroppedLocked() {
	m.counters[TelemetryDroppedTotal][""].value++
}

// DroppedCount returns the total observations dropped for prohibited or
// non-allowed dimensions.
func (m *Metrics) DroppedCount() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return uint64(m.counters[TelemetryDroppedTotal][""].value)
}

// WritePrometheus emits the registry in Prometheus text exposition format.
func (m *Metrics) WritePrometheus(writer io.Writer) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := sortedMetricNames(m.defs)
	for _, name := range names {
		def := m.defs[name]
		if err := writeLine(writer, "# HELP", def.name, def.help); err != nil {
			return err
		}
		if err := writeLine(writer, "# TYPE", def.name, typeName(def.kind)); err != nil {
			return err
		}
		switch def.kind {
		case kindCounter:
			if err := writeSeriesSet(writer, def, m.counters[name]); err != nil {
				return err
			}
		case kindGauge:
			if err := writeSeriesSet(writer, def, m.gauges[name]); err != nil {
				return err
			}
		case kindHistogram:
			if err := writeHistogramSet(writer, def, m.histograms[name]); err != nil {
				return err
			}
		}
	}
	return nil
}

func typeName(kind metricKind) string {
	switch kind {
	case kindCounter:
		return "counter"
	case kindGauge:
		return "gauge"
	case kindHistogram:
		return "histogram"
	}
	return "untyped"
}

func writeLine(writer io.Writer, parts ...string) error {
	_, err := fmt.Fprintln(writer, strings.Join(parts, " "))
	return err
}

func writeSeriesSet(writer io.Writer, def *metricDef, series map[string]*metricSeries) error {
	keys := make([]string, 0, len(series))
	for key := range series {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, err := fmt.Fprintln(writer, formatSample(def, series[key])); err != nil {
			return err
		}
	}
	return nil
}

func writeHistogramSet(writer io.Writer, def *metricDef, series map[string]*histogram) error {
	keys := make([]string, 0, len(series))
	for key := range series {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		h := series[key]
		labelText := labelBlock(def, h.labels)
		for i, bound := range h.boundaries {
			line := def.name + "_bucket" + labelText + `,le="` + formatFloat(bound) + `"} ` + strconv.FormatUint(h.counts[i], 10)
			if _, err := fmt.Fprintln(writer, line); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(writer, def.name+"_bucket"+labelText+`,le="+Inf"} `+strconv.FormatUint(h.count, 10)); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(writer, def.name+"_sum"+labelText+"} "+formatFloat(h.sum)); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(writer, def.name+"_count"+labelText+"} "+strconv.FormatUint(h.count, 10)); err != nil {
			return err
		}
	}
	return nil
}

func formatSample(def *metricDef, s *metricSeries) string {
	if len(def.labels) == 0 {
		return def.name + " " + formatFloat(s.value)
	}
	return def.name + labelBlock(def, s.labels) + "} " + formatFloat(s.value)
}

// labelBlock renders "{k1="v1",k2="v2"" in canonical label order, without the
// trailing brace. Callers append the closing brace.
func labelBlock(def *metricDef, labels map[string]string) string {
	var builder strings.Builder
	builder.WriteString("{")
	for i, name := range def.labels {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(name)
		builder.WriteString(`="`)
		builder.WriteString(escapeLabelValue(labels[name]))
		builder.WriteString(`"`)
	}
	return builder.String()
}

func escapeLabelValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}

func sortedMetricNames(defs map[string]*metricDef) []string {
	names := make([]string, 0, len(defs))
	for name := range defs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
