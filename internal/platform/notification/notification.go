package notification

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"integin/internal/shared/notifications"
)

type PreferenceSet map[notifications.Category]map[notifications.Channel]bool

func ShouldDeliver(category notifications.Category, channel notifications.Channel, user, role, tenant PreferenceSet) bool {
	if category.Mandatory() {
		return true
	}
	if enabled, ok := lookup(user, category, channel); ok {
		return enabled
	}
	if enabled, ok := lookup(role, category, channel); ok {
		return enabled
	}
	if enabled, ok := lookup(tenant, category, channel); ok {
		return enabled
	}
	return channel == notifications.ChannelInApp
}

func lookup(preferences PreferenceSet, category notifications.Category, channel notifications.Channel) (bool, bool) {
	channels, ok := preferences[category]
	if !ok {
		return false, false
	}
	enabled, ok := channels[channel]
	return enabled, ok
}

type Request struct {
	ID                string
	TenantID          string
	UserID            string
	Email             string
	Category          notifications.Category
	Severity          string
	Title             string
	Body              string
	ActionURL         string
	CorrelationID     string
	UserPreferences   PreferenceSet
	RolePreferences   PreferenceSet
	TenantPreferences PreferenceSet
}

type Notification struct {
	ID            string
	TenantID      string
	UserID        string
	Severity      string
	Category      notifications.Category
	Title         string
	Body          string
	ActionURL     string
	CorrelationID string
	Status        string
	CreatedAt     time.Time
}

type EmailDelivery struct {
	ID             string
	NotificationID string
	To             string
	State          notifications.DeliveryState
	Attempts       int
	NextAttemptAt  time.Time
	LastError      string
}

type Engine struct {
	now           func() time.Time
	maxAttempts   int
	notifications []Notification
	deliveries    []EmailDelivery
}

func NewEngine(now func() time.Time, maxAttempts int) *Engine {
	if now == nil {
		now = time.Now
	}
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	return &Engine{now: now, maxAttempts: maxAttempts}
}

func (e *Engine) Create(request Request) (Notification, *EmailDelivery, error) {
	for field, value := range map[string]string{"id": request.ID, "tenant_id": request.TenantID, "user_id": request.UserID, "title": request.Title, "body": request.Body} {
		if strings.TrimSpace(value) == "" {
			return Notification{}, nil, fmt.Errorf("%s is required", field)
		}
	}
	if request.Category == "" {
		return Notification{}, nil, errors.New("category is required")
	}
	notification := Notification{ID: request.ID, TenantID: request.TenantID, UserID: request.UserID, Severity: request.Severity, Category: request.Category, Title: request.Title, Body: request.Body, ActionURL: request.ActionURL, CorrelationID: request.CorrelationID, Status: "UNREAD", CreatedAt: e.now().UTC()}
	e.notifications = append(e.notifications, notification)
	if request.Email == "" || !ShouldDeliver(request.Category, notifications.ChannelEmail, request.UserPreferences, request.RolePreferences, request.TenantPreferences) {
		return notification, nil, nil
	}
	delivery := EmailDelivery{ID: request.ID + "-email", NotificationID: request.ID, To: request.Email, State: notifications.DeliveryQueued}
	e.deliveries = append(e.deliveries, delivery)
	return notification, &delivery, nil
}

func (e *Engine) Notifications() []Notification {
	return append([]Notification(nil), e.notifications...)
}
func (e *Engine) Deliveries() []EmailDelivery { return append([]EmailDelivery(nil), e.deliveries...) }

func (e *Engine) ProcessEmail(deliveryID string, send func(EmailDelivery) error) (EmailDelivery, error) {
	index := e.deliveryIndex(deliveryID)
	if index < 0 {
		return EmailDelivery{}, errors.New("delivery not found")
	}
	delivery := &e.deliveries[index]
	if delivery.State != notifications.DeliveryQueued && delivery.State != notifications.DeliveryRetry {
		return *delivery, fmt.Errorf("delivery cannot send from state %s", delivery.State)
	}
	delivery.State, delivery.Attempts = notifications.DeliverySending, delivery.Attempts+1
	if send == nil {
		send = func(EmailDelivery) error { return errors.New("email sender is not configured") }
	}
	if err := send(*delivery); err != nil {
		delivery.LastError = err.Error()
		if delivery.Attempts >= e.maxAttempts {
			delivery.State = notifications.DeliveryGivenUp
		} else {
			delivery.State = notifications.DeliveryFailed
			delivery.NextAttemptAt = e.now().UTC().Add(retryDelay(delivery.Attempts))
		}
		return *delivery, err
	}
	delivery.State, delivery.LastError = notifications.DeliverySent, ""
	return *delivery, nil
}

func (e *Engine) QueueRetry(deliveryID string, at time.Time) (EmailDelivery, error) {
	index := e.deliveryIndex(deliveryID)
	if index < 0 {
		return EmailDelivery{}, errors.New("delivery not found")
	}
	delivery := &e.deliveries[index]
	if delivery.State != notifications.DeliveryFailed && delivery.State != notifications.DeliveryRetry {
		return *delivery, fmt.Errorf("delivery cannot retry from state %s", delivery.State)
	}
	if at.Before(delivery.NextAttemptAt) {
		return *delivery, fmt.Errorf("retry is not due until %s", delivery.NextAttemptAt.UTC().Format(time.RFC3339))
	}
	delivery.State = notifications.DeliveryQueued
	return *delivery, nil
}

func (e *Engine) deliveryIndex(id string) int {
	for index := range e.deliveries {
		if e.deliveries[index].ID == id {
			return index
		}
	}
	return -1
}
func retryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := time.Minute
	for step := 1; step < attempt; step++ {
		delay *= 2
	}
	return delay
}
