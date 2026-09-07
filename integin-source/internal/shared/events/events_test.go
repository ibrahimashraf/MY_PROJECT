package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEnvelopeRoundTripPreservesTenantAndPayload(t *testing.T) {
	original, err := NewEnvelope("event-1", InspectionStarted, "tenant-1", "org-1", "TESTING", "inspection", "inspection-1", map[string]string{"asset_id": "asset-1"}, time.Date(2026, 8, 13, 12, 0, 0, 0, time.FixedZone("UTC+3", 3*60*60)))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Envelope
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.EventID() != "event-1" || decoded.TenantID() != "tenant-1" || decoded.OrganizationID() != "org-1" {
		t.Fatalf("event context lost: %#v", decoded)
	}
	if decoded.EventType() != InspectionStarted || decoded.SchemaVersion() != 1 {
		t.Fatalf("event metadata lost: %#v", decoded)
	}
	var payload map[string]string
	if err := json.Unmarshal(decoded.Payload(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["asset_id"] != "asset-1" {
		t.Fatalf("payload lost: %#v", payload)
	}
}

func TestEnvelopeRequiresIdentityAndTenantContext(t *testing.T) {
	cases := []struct{ name, eventID, eventType, tenantID, organizationID, environment string }{
		{"missing event id", "", FindingRecorded, "tenant-1", "org-1", "LIVE"},
		{"missing tenant", "event-1", FindingRecorded, "", "org-1", "LIVE"},
		{"missing organization", "event-1", FindingRecorded, "tenant-1", "", "LIVE"},
		{"invalid environment", "event-1", FindingRecorded, "tenant-1", "org-1", "DEV"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewEnvelope(tc.eventID, tc.eventType, tc.tenantID, tc.organizationID, tc.environment, "inspection", "i-1", nil, time.Now()); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestEnvelopeDefensivelyCopiesPayload(t *testing.T) {
	original, err := NewEnvelope("event-1", FindingRecorded, "tenant-1", "org-1", "LIVE", "inspection", "i-1", map[string]string{"value": "before"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	copyOfPayload := original.Payload()
	copyOfPayload[0] = 'x'
	if original.Payload()[0] == 'x' {
		t.Fatal("payload was mutable through accessor")
	}
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Envelope
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	decodedPayload := decoded.Payload()
	decodedPayload[0] = 'x'
	if decoded.Payload()[0] == 'x' {
		t.Fatal("decoded payload was mutable through accessor")
	}
}

func TestEnvelopeNormalizesTimestampToUTC(t *testing.T) {
	input := time.Date(2026, 8, 13, 12, 0, 0, 0, time.FixedZone("UTC+3", 3*60*60))
	e, err := NewEnvelope("event-1", AssetMoved, "tenant-1", "org-1", "LIVE", "asset", "a-1", nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if e.OccurredAt().Location() != time.UTC {
		t.Fatalf("expected UTC timestamp, got %s", e.OccurredAt().Location())
	}
}
