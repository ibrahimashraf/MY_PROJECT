package escalation

import (
	"context"
	"errors"
	"time"
)

type EscalationRule struct {
	ID             string
	TenantID       string
	OrganizationID string
	Name           string
	TriggerType    string
	DaysOffset     int
	TargetRole     string
	EscalationTier int
	TemplateID     string
	Enabled        bool
	CreatedBy      string
	CreatedAt      time.Time
}

type EscalationEvent struct {
	ID             string
	TenantID       string
	OrganizationID string
	RuleID         string
	InspectionID   string
	AssignmentID   string
	Tier           int
	Status         string
	TriggeredAt    time.Time
	AcknowledgedAt time.Time
	AcknowledgedBy string
}

type NotificationTemplate struct {
	ID             string
	TenantID       string
	OrganizationID string
	Code           string
	Subject        string
	BodyTemplate   string
	Channel        string
	CreatedBy      string
	CreatedAt      time.Time
}

type Repository interface {
	CreateRule(ctx context.Context, actor ActorContext, r EscalationRule) (EscalationRule, error)
	GetRule(ctx context.Context, actor ActorContext, id string) (EscalationRule, error)
	ListRules(ctx context.Context, actor ActorContext) ([]EscalationRule, error)
	UpdateRule(ctx context.Context, actor ActorContext, r EscalationRule) error
	DeleteRule(ctx context.Context, actor ActorContext, id string) error

	CreateEvent(ctx context.Context, actor ActorContext, e EscalationEvent) (EscalationEvent, error)
	GetEvent(ctx context.Context, actor ActorContext, id string) (EscalationEvent, error)
	ListEvents(ctx context.Context, actor ActorContext, ruleID, inspectionID string, tier int) ([]EscalationEvent, error)
	UpdateEvent(ctx context.Context, actor ActorContext, e EscalationEvent) error

	CreateTemplate(ctx context.Context, actor ActorContext, t NotificationTemplate) (NotificationTemplate, error)
	GetTemplate(ctx context.Context, actor ActorContext, id string) (NotificationTemplate, error)
	ListTemplates(ctx context.Context, actor ActorContext) ([]NotificationTemplate, error)
	UpdateTemplate(ctx context.Context, actor ActorContext, t NotificationTemplate) error
}

type ActorContext struct {
	TenantID       string
	OrganizationID string
	ActorID        string
}

func (a ActorContext) Validate() error {
	if a.TenantID == "" || a.OrganizationID == "" || a.ActorID == "" {
		return ErrInvalidActor
	}
	return nil
}

var ErrInvalidActor = errors.New("escalation actor context is invalid")
