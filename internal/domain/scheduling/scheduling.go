package scheduling

import (
	"context"
	"errors"
	"time"
)

type CalendarEntry struct {
	ID                 string
	TenantID           string
	OrganizationID     string
	WorkOrderID        string
	AssignmentID       string
	TechnicianID       string
	StartAt            time.Time
	EndAt              time.Time
	Status             string
	CompetencyVerified bool
	CreatedBy          string
	CreatedAt          time.Time
}

type TechnicianCompetency struct {
	ID               string
	TenantID         string
	OrganizationID   string
	TechnicianID     string
	EquipmentTypeID  string
	CertificationRef string
	ExpiresAt        time.Time
	Status           string
	VerifiedBy       string
	CreatedAt        time.Time
}

type SchedulingRule struct {
	ID             string
	TenantID       string
	OrganizationID string
	Name           string
	RuleType       string
	Config         string
	Enabled        bool
}

type Repository interface {
	CreateCalendarEntry(ctx context.Context, actor ActorContext, e CalendarEntry) (CalendarEntry, error)
	GetCalendarEntry(ctx context.Context, actor ActorContext, id string) (CalendarEntry, error)
	ListCalendarEntries(ctx context.Context, actor ActorContext, from, to time.Time, technicianID string) ([]CalendarEntry, error)
	UpdateCalendarEntry(ctx context.Context, actor ActorContext, e CalendarEntry) error
	DeleteCalendarEntry(ctx context.Context, actor ActorContext, id string) error

	CreateTechnicianCompetency(ctx context.Context, actor ActorContext, c TechnicianCompetency) (TechnicianCompetency, error)
	GetTechnicianCompetency(ctx context.Context, actor ActorContext, id string) (TechnicianCompetency, error)
	ListTechnicianCompetencies(ctx context.Context, actor ActorContext, technicianID string) ([]TechnicianCompetency, error)
	UpdateTechnicianCompetency(ctx context.Context, actor ActorContext, c TechnicianCompetency) error
	DeleteTechnicianCompetency(ctx context.Context, actor ActorContext, id string) error

	CreateSchedulingRule(ctx context.Context, actor ActorContext, r SchedulingRule) (SchedulingRule, error)
	GetSchedulingRule(ctx context.Context, actor ActorContext, id string) (SchedulingRule, error)
	ListSchedulingRules(ctx context.Context, actor ActorContext) ([]SchedulingRule, error)
	UpdateSchedulingRule(ctx context.Context, actor ActorContext, r SchedulingRule) error
	DeleteSchedulingRule(ctx context.Context, actor ActorContext, id string) error

	CheckCompetency(ctx context.Context, actor ActorContext, technicianID, equipmentTypeID string) (bool, error)
	CheckScheduleConflict(ctx context.Context, actor ActorContext, technicianID string, start, end time.Time, excludeID string) (bool, error)
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

var ErrInvalidActor = errors.New("scheduling actor context is invalid")
