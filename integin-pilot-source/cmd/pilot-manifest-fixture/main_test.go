package main

import (
	"encoding/json"
	"testing"
	"time"

	"integin/internal/domain/device_trust"
)

func TestFixtureAuthorityHMACSurvivesDatabasePrecisionRoundTrip(t *testing.T) {
	issuedAt := time.Date(2026, time.August, 20, 12, 0, 0, 123456789, time.UTC).Truncate(time.Microsecond)
	if issuedAt.Nanosecond()%int(time.Microsecond) != 0 {
		t.Fatal("fixture authority issuance time is not microsecond aligned")
	}
	device, err := device_trust.NewDevice("fixture-device", "fixture-tenant", "fixture-organization", "fixture-user", "fixture-public-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	authority, err := device_trust.IssueAuthorityPackage(device, "fixture-authority", "fixture-test-secret", []string{"work_package.read"}, issuedAt, 2*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	reloadedIssuedAt, err := time.Parse(time.RFC3339Nano, authority.IssuedAt.Format(time.RFC3339Nano))
	if err != nil {
		t.Fatal(err)
	}
	reloadedExpiresAt, err := time.Parse(time.RFC3339Nano, authority.ExpiresAt.Format(time.RFC3339Nano))
	if err != nil {
		t.Fatal(err)
	}
	authority.IssuedAt = reloadedIssuedAt
	authority.ExpiresAt = reloadedExpiresAt
	if err := device_trust.ValidateAuthorityPackage(authority, device, "fixture-test-secret", issuedAt.Add(time.Minute)); err != nil {
		t.Fatalf("database-precision authority round trip failed validation: %v", err)
	}
}

func TestFixtureAssignmentContextUsesStringFieldAssetMappings(t *testing.T) {
	var fieldAssetIDs map[string]string
	if err := json.Unmarshal([]byte(pilotFieldAssetIDsJSON), &fieldAssetIDs); err != nil {
		t.Fatalf("fixture context does not match AssignmentContext map contract: %v", err)
	}
	if fieldAssetIDs["condition"] != "pilot-manifest-demo-asset" {
		t.Fatal("fixture context condition asset mapping is incomplete")
	}
}
