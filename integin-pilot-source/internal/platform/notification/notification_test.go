package notification

import (
	"errors"
	"testing"
	"time"

	"integin/internal/shared/notifications"
)

func fixedNow() time.Time { return time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC) }

func TestPreferenceResolutionUsesUserRoleTenantAndMandatoryPolicy(t *testing.T) {
	category := notifications.CategoryCertificateIssued
	user := PreferenceSet{category: {notifications.ChannelEmail: false}}
	role := PreferenceSet{category: {notifications.ChannelEmail: true}}
	tenant := PreferenceSet{category: {notifications.ChannelEmail: false}}
	if ShouldDeliver(category, notifications.ChannelEmail, user, role, tenant) {
		t.Fatal("user preference should override role")
	}
	user = PreferenceSet{}
	if !ShouldDeliver(category, notifications.ChannelEmail, user, role, tenant) {
		t.Fatal("role preference should apply when user is absent")
	}
	if !ShouldDeliver(notifications.CategorySecurityEvent, notifications.ChannelEmail, user, PreferenceSet{}, tenant) {
		t.Fatal("mandatory category must bypass opt-out")
	}
	if !ShouldDeliver(category, notifications.ChannelInApp, PreferenceSet{}, PreferenceSet{}, PreferenceSet{}) {
		t.Fatal("in-app delivery should default on")
	}
}

func TestNotificationAlwaysCreatesInAppAndQueuesEmailWhenEnabled(t *testing.T) {
	engine := NewEngine(fixedNow, 3)
	request := Request{ID: "notification-1", TenantID: "tenant-1", UserID: "user-1", Email: "client@example.com", Category: notifications.CategoryCertificateIssued, Title: "Certificate issued", Body: "Your certificate is ready", UserPreferences: PreferenceSet{notifications.CategoryCertificateIssued: {notifications.ChannelEmail: true}}}
	notification, delivery, err := engine.Create(request)
	if err != nil {
		t.Fatal(err)
	}
	if notification.Status != "UNREAD" || delivery == nil || delivery.State != notifications.DeliveryQueued {
		t.Fatalf("unexpected notification/delivery: %#v %#v", notification, delivery)
	}
	if len(engine.Notifications()) != 1 || len(engine.Deliveries()) != 1 {
		t.Fatal("records were not retained")
	}
}

func TestNotificationRetryStateMachineAndGiveUp(t *testing.T) {
	engine := NewEngine(fixedNow, 2)
	_, delivery, err := engine.Create(Request{ID: "notification-1", TenantID: "tenant-1", UserID: "user-1", Email: "client@example.com", Category: notifications.CategoryCertificateIssued, Title: "Certificate issued", Body: "body", UserPreferences: PreferenceSet{notifications.CategoryCertificateIssued: {notifications.ChannelEmail: true}}})
	if err != nil {
		t.Fatal(err)
	}
	failed, sendErr := engine.ProcessEmail(delivery.ID, func(EmailDelivery) error { return errors.New("temporary SMTP error") })
	if sendErr == nil || failed.State != notifications.DeliveryFailed || failed.Attempts != 1 {
		t.Fatalf("expected failed attempt: %#v %v", failed, sendErr)
	}
	if _, err := engine.QueueRetry(delivery.ID, fixedNow().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	givenUp, sendErr := engine.ProcessEmail(delivery.ID, func(EmailDelivery) error { return errors.New("permanent SMTP error") })
	if sendErr == nil || givenUp.State != notifications.DeliveryGivenUp || givenUp.Attempts != 2 {
		t.Fatalf("expected give-up state: %#v %v", givenUp, sendErr)
	}
}

func TestNotificationSuccessfulDelivery(t *testing.T) {
	engine := NewEngine(fixedNow, 3)
	_, delivery, err := engine.Create(Request{ID: "notification-1", TenantID: "tenant-1", UserID: "user-1", Email: "client@example.com", Category: notifications.CategoryCertificateIssued, Title: "Certificate issued", Body: "body", UserPreferences: PreferenceSet{notifications.CategoryCertificateIssued: {notifications.ChannelEmail: true}}})
	if err != nil {
		t.Fatal(err)
	}
	sent, err := engine.ProcessEmail(delivery.ID, func(EmailDelivery) error { return nil })
	if err != nil || sent.State != notifications.DeliverySent {
		t.Fatalf("expected sent state: %#v %v", sent, err)
	}
}
