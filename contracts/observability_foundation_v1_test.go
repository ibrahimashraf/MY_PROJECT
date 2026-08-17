package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestObservabilityFoundationV1PreservesAllowlistAndProhibitedDataBoundaries(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("observability_foundation_v1.json"))
	if err != nil {
		t.Fatalf("read observability foundation contract: %v", err)
	}
	var contract struct {
		ContractVersion string `json:"contract_version"`
		Status          string `json:"status"`
		MetricContract  struct {
			AllowedDimensions    []string `json:"allowed_dimensions"`
			ProhibitedDimensions []string `json:"prohibited_dimensions"`
			MetricNames          []string `json:"metric_names"`
		} `json:"metric_contract"`
		TraceContract struct {
			ProhibitedAttributes []string `json:"prohibited_attributes"`
		} `json:"trace_contract"`
		LogContract struct {
			RequiredFields   []string `json:"required_fields"`
			ProhibitedFields []string `json:"prohibited_fields"`
		} `json:"log_contract"`
		MinimumAlerts []string `json:"minimum_alerts"`
	}
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatalf("observability foundation contract must be valid JSON: %v", err)
	}
	if contract.ContractVersion != "observability-foundation/v1" || contract.Status != "governance-only-no-telemetry-deployed" {
		t.Fatalf("unexpected observability contract identity: %q / %q", contract.ContractVersion, contract.Status)
	}
	assertContains(t, contract.MetricContract.AllowedDimensions, "route_template")
	assertContains(t, contract.MetricContract.ProhibitedDimensions, "tenant_id")
	assertContains(t, contract.MetricContract.ProhibitedDimensions, "evidence_id")
	assertContains(t, contract.MetricContract.MetricNames, "integin_telemetry_dropped_total")
	assertContains(t, contract.TraceContract.ProhibitedAttributes, "http.request.body")
	assertContains(t, contract.TraceContract.ProhibitedAttributes, "integin.evidence_bytes")
	assertContains(t, contract.LogContract.RequiredFields, "trace_id")
	assertContains(t, contract.LogContract.ProhibitedFields, "authorization")
	assertContains(t, contract.LogContract.ProhibitedFields, "evidence_bytes")
	for _, alert := range []string{"integin_service_health_or_readiness_unavailable", "integin_evidence_persistence_failure", "integin_telemetry_drop_or_pipeline_silence"} {
		assertContains(t, contract.MinimumAlerts, alert)
	}
}

func assertContains(t *testing.T, values []string, expected string) {
	t.Helper()
	for _, value := range values {
		if value == expected {
			return
		}
	}
	t.Fatalf("value %q not found", expected)
}
