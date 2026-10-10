package liftviewexport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientBrandingDoesNotAlterAuthoritativeContent(t *testing.T) {
	handler := Handler{}

	// Baseline export without branding
	basePayload := planningPayload(t)
	baseBody, _ := json.Marshal(basePayload)
	reqBase := httptest.NewRequest(http.MethodPost, "/api/v1/liftviews/export?mode=verdict", bytes.NewReader(baseBody))
	reqBase.Header.Set("Content-Type", "application/json")
	recBase := httptest.NewRecorder()
	handler.ServeHTTP(recBase, reqBase)
	if recBase.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for base, got %d: %s", recBase.Code, recBase.Body.String())
	}
	var baseMeta map[string]any
	if err := json.Unmarshal(recBase.Body.Bytes(), &baseMeta); err != nil {
		t.Fatal(err)
	}

	// Branded export
	brandedPayload := planningPayload(t)
	brandedPayload["branding"] = ClientBranding{
		OrganizationName: "Global Heavy Lift Ltd",
		ClientName:       "Sovereign Energy Corp",
		LogoURL:          "https://cdn.integin.com/logos/global.png",
		PrimaryColorHex:  "#0055AA",
		CustomHeader:     "CONFIDENTIAL RIGGING DOSSIER",
		WatermarkText:    "DRAFT REVIEW ONLY",
	}
	brandedPayload["revision"] = RevisionRecord{
		RevisionCode:     "REV-B",
		CreatedAt:        time.Now().UTC(),
		AuthorDID:        "did:integin:user:engineer-42",
		ChangeSummary:    "Incorporated client rigging review and pad adjustments",
		ParentHashSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}

	brandedBody, _ := json.Marshal(brandedPayload)
	reqBranded := httptest.NewRequest(http.MethodPost, "/api/v1/liftviews/export?mode=verdict", bytes.NewReader(brandedBody))
	reqBranded.Header.Set("Content-Type", "application/json")
	recBranded := httptest.NewRecorder()
	handler.ServeHTTP(recBranded, reqBranded)
	if recBranded.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for branded, got %d: %s", recBranded.Code, recBranded.Body.String())
	}
	var brandedMeta map[string]any
	if err := json.Unmarshal(recBranded.Body.Bytes(), &brandedMeta); err != nil {
		t.Fatal(err)
	}

	// Invariant Checks: Authoritative values must remain identical
	authoritativeKeys := []string{
		"crane_util", "rigging_util", "leg_tension", "max_share_t",
		"dhl_t", "fos1", "fos2", "hook_span_m", "clearance_m",
		"certification_scope", "rigging_field_notice",
	}
	for _, key := range authoritativeKeys {
		if baseMeta[key] != brandedMeta[key] {
			t.Errorf("authoritative metric %q was corrupted by branding: base=%v, branded=%v",
				key, baseMeta[key], brandedMeta[key])
		}
	}

	// Verify branding block was populated
	brandingObj, ok := brandedMeta["branding"].(map[string]any)
	if !ok || brandingObj["client_name"] != "Sovereign Energy Corp" {
		t.Fatalf("branding metadata was not captured correctly: %#v", brandedMeta["branding"])
	}

	// Verify revision block was populated
	revObj, ok := brandedMeta["revision"].(map[string]any)
	if !ok || revObj["revision_code"] != "REV-B" {
		t.Fatalf("revision metadata was not captured correctly: %#v", brandedMeta["revision"])
	}
}

func TestClientShareTokenLifecycleAndTenantScoping(t *testing.T) {
	handler := Handler{}

	// 1. Create Share Token for tenant "tenant-offshore-a"
	createReq := CreateShareRequest{
		PlanID:                      "LP-2026-9901",
		ExpiresInHours:              48,
		AllowDownloadCAD:            false,
		RedactSensitiveSiteGeometry: true,
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/liftviews/share", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-offshore-a")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("share generation failed: %d: %s", rec.Code, rec.Body.String())
	}

	var shareResp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &shareResp); err != nil {
		t.Fatal(err)
	}
	token, ok := shareResp["share_token"].(string)
	if !ok || token == "" {
		t.Fatalf("missing share_token in response: %#v", shareResp)
	}

	// 2. Consume Share Token as tenant-offshore-a -> 200 OK
	reqAccess := httptest.NewRequest(http.MethodGet, "/api/v1/liftviews/share?token="+token, nil)
	reqAccess.Header.Set("X-Tenant-ID", "tenant-offshore-a")
	recAccess := httptest.NewRecorder()
	handler.ServeHTTP(recAccess, reqAccess)
	if recAccess.Code != http.StatusOK {
		t.Fatalf("client share access failed for valid tenant: %d: %s", recAccess.Code, recAccess.Body.String())
	}
	var portalView map[string]any
	if err := json.Unmarshal(recAccess.Body.Bytes(), &portalView); err != nil {
		t.Fatal(err)
	}
	if portalView["status"] != "AUTHORIZED" || portalView["tenant_did"] != "did:integin:tenant:tenant-offshore-a" {
		t.Fatalf("unexpected portal view: %#v", portalView)
	}

	// 3. Cross-Tenant Attempt: tenant-rogue-b attempts to access tenant-offshore-a's token -> 403 Forbidden
	reqRogue := httptest.NewRequest(http.MethodGet, "/api/v1/liftviews/share?token="+token, nil)
	reqRogue.Header.Set("X-Tenant-ID", "tenant-rogue-b")
	recRogue := httptest.NewRecorder()
	handler.ServeHTTP(recRogue, reqRogue)
	if recRogue.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for cross-tenant breach, got %d: %s", recRogue.Code, recRogue.Body.String())
	}

	// 4. Tampered Token Attempt -> 403 Forbidden
	reqTampered := httptest.NewRequest(http.MethodGet, "/api/v1/liftviews/share?token="+token+"tampered", nil)
	recTampered := httptest.NewRecorder()
	handler.ServeHTTP(recTampered, reqTampered)
	if recTampered.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for tampered token, got %d: %s", recTampered.Code, recTampered.Body.String())
	}
}

func TestClientShareTokenExpirationFailClosed(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-1 * time.Hour)

	claims := ShareTokenClaims{
		TokenID:                     "expired-token-1",
		TenantID:                    "tenant-alpha",
		PlanID:                      "LP-EXPIRED",
		IssuedAt:                    past.Add(-2 * time.Hour),
		ExpiresAt:                   past,
		AllowDownloadCAD:            true,
		RedactSensitiveSiteGeometry: false,
	}

	token, err := GenerateShareToken(claims, defaultShareHMACKey)
	if err != nil {
		t.Fatal(err)
	}

	// Access with expired token
	handler := Handler{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/liftviews/share?token="+token, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410 Gone for expired share token, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPlanMetadataRedaction(t *testing.T) {
	rawMeta := map[string]any{
		"crane_util":               78.5,
		"leg_tension":              14.2,
		"certification_scope":      "SIMULATION ONLY",
		"internal_operator_notes":  "Secret contractor margin",
		"chassis_tare_weight":      60.0,
		"outrigger_pad_details":    "Proprietary composite pad spec",
	}

	redacted := RedactPlanMetadata(rawMeta)

	if redacted["redacted"] != true || redacted["confidential_site_geometry_redacted"] != true {
		t.Fatal("redaction flags not set")
	}
	if _, exists := redacted["internal_operator_notes"]; exists {
		t.Error("internal_operator_notes was not redacted")
	}
	if _, exists := redacted["chassis_tare_weight"]; exists {
		t.Error("chassis_tare_weight was not redacted")
	}
	if _, exists := redacted["outrigger_pad_details"]; exists {
		t.Error("outrigger_pad_details was not redacted")
	}
	// Authoritative verification metrics must stay preserved
	if redacted["crane_util"] != 78.5 || redacted["leg_tension"] != 14.2 {
		t.Error("safety metrics were erroneously stripped during redaction")
	}
}

func TestEvaluatePlanRenewal(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

	// Case 1: Expired plan
	expiredAt := now.Add(-24 * time.Hour)
	statusExpired := EvaluatePlanRenewal(expiredAt, now, 14)
	if !statusExpired.IsExpired || !statusExpired.RenewalRequired || statusExpired.AlertSeverity != "EXPIRED" {
		t.Fatalf("expected EXPIRED status, got %#v", statusExpired)
	}

	// Case 2: Expiring in 5 days (within 14-day lead)
	dueAt := now.Add(5 * 24 * time.Hour)
	statusDue := EvaluatePlanRenewal(dueAt, now, 14)
	if statusDue.IsExpired || !statusDue.RenewalRequired || statusDue.AlertSeverity != "RENEWAL_DUE" {
		t.Fatalf("expected RENEWAL_DUE status, got %#v", statusDue)
	}

	// Case 3: Valid for 60 days
	validAt := now.Add(60 * 24 * time.Hour)
	statusValid := EvaluatePlanRenewal(validAt, now, 14)
	if statusValid.IsExpired || statusValid.RenewalRequired || statusValid.AlertSeverity != "VALID" {
		t.Fatalf("expected VALID status, got %#v", statusValid)
	}

	// Case 4: HTTP endpoint integration test
	handler := Handler{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/liftviews/renewal?expires_at="+dueAt.Format(time.RFC3339)+"&lead_days=14", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from renewal endpoint, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp RenewalStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.AlertSeverity != "RENEWAL_DUE" {
		t.Fatalf("expected RENEWAL_DUE from endpoint, got %s", resp.AlertSeverity)
	}
}
