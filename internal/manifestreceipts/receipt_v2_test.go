package manifestreceipts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func pointer[T any](value T) *T { return &value }

func TestV2WriterEmitsFieldBindingWithoutInventedHTTPStatus(t *testing.T) {
	directory := t.TempDir()
	now := time.Date(2026, time.August, 18, 13, 0, 0, 0, time.UTC)
	writer, err := NewV2Writer("0123456789abcdef0123456789abcdef", directory, func() time.Time { return now })
	if err != nil {
		t.Fatalf("new v2 writer: %v", err)
	}
	if err := writer.Emit(V2Observation{Case: CaseFieldBinding, EventSource: V2EventSourceFieldBinding, ObservedOutcome: "verified_cached", Transport: V2Transport{Kind: V2TransportFieldBinding}, ReplayState: V2ReplayState{Scope: V2ReplayScopeNotApplicable}, GeneratedAt: now}); err != nil {
		t.Fatalf("emit field binding: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(directory, "receipt-v2-field_binding-0123456789abcdef0123456789abcdef.json"))
	if err != nil {
		t.Fatalf("read v2 receipt: %v", err)
	}
	if strings.Contains(string(content), "http_status") {
		t.Fatalf("field receipt invented HTTP status: %s", content)
	}
	var decoded map[string]any
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatalf("decode v2 receipt: %v", err)
	}
	if decoded["status"] != "passed" || decoded["event_source"] != "field_binding" {
		t.Fatalf("unexpected v2 receipt: %#v", decoded)
	}
}

func TestV2WriterRequiresVerifiedTenantOnlyForValidProof(t *testing.T) {
	directory := t.TempDir()
	writer, err := NewV2Writer("fedcba9876543210fedcba9876543210", directory, time.Now)
	if err != nil {
		t.Fatalf("new v2 writer: %v", err)
	}
	before, after, delta, status := int64(9), int64(10), int64(1), 200
	if err := writer.Emit(V2Observation{Case: CaseValidProof, EventSource: V2EventSourceManifestHTTP, ObservedOutcome: "proof_valid", Transport: V2Transport{Kind: V2TransportHTTP, HTTPStatus: pointer(status)}, ReplayState: V2ReplayState{Scope: V2ReplayScopeVerifiedTenant, Before: pointer(before), After: pointer(after), Delta: pointer(delta)}}); err != nil {
		t.Fatalf("emit valid proof: %v", err)
	}
	if err := writer.Emit(V2Observation{Case: CaseReplay, EventSource: V2EventSourceManifestHTTP, ObservedOutcome: "replay_rejected", Transport: V2Transport{Kind: V2TransportHTTP, HTTPStatus: pointer(409)}, ReplayState: V2ReplayState{Scope: V2ReplayScopeVerifiedTenant, Before: pointer(before), After: pointer(after), Delta: pointer(delta)}}); err == nil {
		t.Fatal("replay receipt accepted verified tenant arithmetic")
	}
}

func TestV2WriterRejectsUnsafeOrWrongTransportState(t *testing.T) {
	writer, err := NewV2Writer("abcdef0123456789abcdef0123456789", t.TempDir(), time.Now)
	if err != nil {
		t.Fatalf("new v2 writer: %v", err)
	}
	if err := writer.Emit(V2Observation{Case: CaseFieldBinding, EventSource: V2EventSourceFieldBinding, ObservedOutcome: "verified_cached", Transport: V2Transport{Kind: V2TransportFieldBinding, HTTPStatus: pointer(200)}, ReplayState: V2ReplayState{Scope: V2ReplayScopeNotApplicable}}); err == nil {
		t.Fatal("field receipt accepted invented HTTP status")
	}
	if err := writer.Emit(V2Observation{Case: CaseSignatureInvalid, EventSource: V2EventSourceManifestHTTP, ObservedOutcome: "signature_invalid", Transport: V2Transport{Kind: V2TransportHTTP, HTTPStatus: pointer(403)}, ReplayState: V2ReplayState{Scope: V2ReplayScopeNotApplicable}}); err == nil {
		t.Fatal("signature receipt accepted wrong source status")
	}
}
