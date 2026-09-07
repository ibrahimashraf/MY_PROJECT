package acceptance

import (
	"testing"
	"time"

	"integin/internal/platform/calibration"
	"integin/internal/platform/featureconsole"
	"integin/internal/platform/featureflag"
	"integin/internal/platform/notification"
	"integin/internal/platform/qrverification"
	sharedcalibration "integin/internal/shared/calibration"
	"integin/internal/shared/featureflags"
	"integin/internal/shared/notifications"
	"integin/internal/shared/types"
)

func TestPhase5PlatformWorkflow(t *testing.T) {
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	flags := featureflag.New()
	flags.SetDefault(featureflags.FlagAIAdvisory, featureflags.Enabled)
	if state := flags.Evaluate(featureflags.FlagAIAdvisory, featureflag.Request{OrganizationID: "org-1", UserID: "user-1"}, now); state != featureflags.Enabled {
		t.Fatalf("feature flag not enabled: %s", state)
	}
	calibrationService := calibration.New(nil)
	if err := calibrationService.Create(sharedcalibration.Record{ID: "cal-1", TenantID: "tenant-1", EquipmentID: "meter-1", StandardReference: "ISO-1", CalibrationDate: now.Add(-time.Hour), NextDueDate: now.Add(time.Hour), TechnicianID: "tech-1", Result: "PASS", Status: types.CalibrationActive}); err != nil {
		t.Fatal(err)
	}
	if err := calibrationService.SubmissionAllowed("tenant-1", "meter-1", now); err != nil {
		t.Fatal(err)
	}
	engine := notification.NewEngine(func() time.Time { return now }, 3)
	_, delivery, err := engine.Create(notification.Request{ID: "notification-1", TenantID: "tenant-1", UserID: "user-1", Email: "client@example.com", Category: notifications.CategoryCertificateIssued, Title: "Certificate issued", Body: "ready", UserPreferences: notification.PreferenceSet{notifications.CategoryCertificateIssued: {notifications.ChannelEmail: true}}})
	if err != nil || delivery == nil {
		t.Fatalf("notification not queued: %#v %v", delivery, err)
	}
	verifier := qrverification.New(5)
	if err := verifier.Register(qrverification.Certificate{Token: "qr-1", Environment: "LIVE", Status: "ISSUED", Number: "CERT-1", SerialNumber: "SN-1", AssetType: "container"}); err != nil {
		t.Fatal(err)
	}
	if result := verifier.PublicVerify("qr-1", "ip-1"); result.StatusCode != 200 || result.Public.SerialNumber != "SN-1" {
		t.Fatalf("qr verification failed: %#v", result)
	}
	console := featureconsole.New()
	if err := console.Register(featureconsole.Module{Name: "notification", Version: "1.0.0", EntryPoint: "internal/platform/notification"}); err != nil {
		t.Fatal(err)
	}
	if _, err := console.UpdatePath("notification", "1.0.0", "1.1.0"); err != nil {
		t.Fatal(err)
	}
}
