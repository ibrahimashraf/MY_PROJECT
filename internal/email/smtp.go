package email

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"integin/internal/shared/notifications"
)

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return errors.New("smtp host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return errors.New("smtp port is invalid")
	}
	if strings.TrimSpace(c.From) == "" || !strings.Contains(c.From, "@") {
		return errors.New("smtp from address is invalid")
	}
	if (c.Username == "") != (c.Password == "") {
		return errors.New("smtp username and password must be supplied together")
	}
	return nil
}

type Message struct {
	To          []string
	Subject     string
	Body        string
	ContentType string
}
type Sender func(addr string, auth smtp.Auth, from string, to []string, message []byte) error
type HealthChecker func(context.Context, Config) error

type Delivery struct {
	State     notifications.DeliveryState
	Attempts  int
	LastError string
}

type Provider struct {
	Config        Config
	Sender        Sender
	HealthChecker HealthChecker
}

func NewProvider(config Config, sender Sender, checker HealthChecker) (*Provider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if sender == nil {
		sender = smtp.SendMail
	}
	if checker == nil {
		checker = defaultHealth
	}
	return &Provider{Config: config, Sender: sender, HealthChecker: checker}, nil
}

func (p *Provider) Send(ctx context.Context, message Message) Delivery {
	delivery := Delivery{State: notifications.DeliveryQueued}
	if err := ctx.Err(); err != nil {
		delivery.State, delivery.LastError = notifications.DeliveryFailed, err.Error()
		return delivery
	}
	if err := validateMessage(message); err != nil {
		delivery.State, delivery.LastError = notifications.DeliveryFailed, err.Error()
		return delivery
	}
	delivery.State, delivery.Attempts = notifications.DeliverySending, 1
	body := message.Body
	contentType := message.ContentType
	if contentType == "" {
		contentType = "text/plain; charset=utf-8"
	}
	raw := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: %s\r\n\r\n%s", p.Config.From, sanitizeHeader(strings.Join(message.To, ", ")), sanitizeHeader(message.Subject), contentType, body))
	var auth smtp.Auth
	if p.Config.Username != "" {
		auth = smtp.PlainAuth("", p.Config.Username, p.Config.Password, p.Config.Host)
	}
	if err := p.Sender(net.JoinHostPort(p.Config.Host, strconv.Itoa(p.Config.Port)), auth, p.Config.From, message.To, raw); err != nil {
		delivery.State, delivery.LastError = notifications.DeliveryFailed, err.Error()
		return delivery
	}
	delivery.State = notifications.DeliverySent
	return delivery
}

func (p *Provider) Health(ctx context.Context) error {
	if err := p.Config.Validate(); err != nil {
		return err
	}
	return p.HealthChecker(ctx, p.Config)
}
func validateMessage(message Message) error {
	if len(message.To) == 0 {
		return errors.New("at least one recipient is required")
	}
	if strings.TrimSpace(message.Subject) == "" {
		return errors.New("email subject is required")
	}
	if message.Body == "" {
		return errors.New("email body is required")
	}
	return nil
}
func sanitizeHeader(v string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(v)
}
func defaultHealth(ctx context.Context, config Config) error {
	dialer := net.Dialer{Timeout: 3 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(config.Host, strconv.Itoa(config.Port)))
	if err != nil {
		return err
	}
	return connection.Close()
}

