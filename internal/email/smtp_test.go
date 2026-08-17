package email

import (
	"context"
	"errors"
	"net/smtp"
	"strings"
	"testing"

	"integin/internal/shared/notifications"
)

func validConfig() Config {
	return Config{Host: "smtp.example", Port: 587, Username: "user", Password: "password", From: "integin@example.com", UseTLS: true}
}

func TestSMTPConfigValidation(t *testing.T) {
	if err := validConfig().Validate(); err != nil {
		t.Fatal(err)
	}
	invalid := validConfig()
	invalid.Port = 0
	if err := invalid.Validate(); err == nil {
		t.Fatal("invalid port should fail")
	}
	invalid = validConfig()
	invalid.Password = ""
	if err := invalid.Validate(); err == nil {
		t.Fatal("partial credentials should fail")
	}
}

func TestSMTPProviderSendSuccessAndFailure(t *testing.T) {
	captured := ""
	provider, err := NewProvider(validConfig(), func(addr string, auth smtp.Auth, from string, to []string, message []byte) error {
		captured = string(message)
		return nil
	}, func(context.Context, Config) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	delivery := provider.Send(context.Background(), Message{To: []string{"client@example.com"}, Subject: "INTEGIN Certificate", Body: "Certificate issued"})
	if delivery.State != notifications.DeliverySent || delivery.Attempts != 1 || !strings.Contains(captured, "Subject: INTEGIN Certificate") {
		t.Fatalf("unexpected delivery: %#v %s", delivery, captured)
	}
	failing, err := NewProvider(validConfig(), func(addr string, auth smtp.Auth, from string, to []string, message []byte) error {
		return errors.New("smtp unavailable")
	}, func(context.Context, Config) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	failed := failing.Send(context.Background(), Message{To: []string{"client@example.com"}, Subject: "INTEGIN Certificate", Body: "Certificate issued"})
	if failed.State != notifications.DeliveryFailed || failed.LastError == "" {
		t.Fatalf("expected failed delivery: %#v", failed)
	}
}

func TestSMTPProviderHealthAndMessageValidation(t *testing.T) {
	provider, err := NewProvider(validConfig(), nil, func(context.Context, Config) error { return errors.New("health unavailable") })
	if err != nil {
		t.Fatal(err)
	}
	if err := provider.Health(context.Background()); err == nil {
		t.Fatal("expected health error")
	}
	delivery := provider.Send(context.Background(), Message{Subject: "Missing recipient", Body: "body"})
	if delivery.State != notifications.DeliveryFailed {
		t.Fatal("invalid message should fail")
	}
}
