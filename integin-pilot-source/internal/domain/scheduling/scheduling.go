package scheduling

import (
	"context"
	"errors"
	"fmt"
	"time"

	"integin/pkg/onboarding"
)

// CompetencyStatus values mirror the CHECK constraint on technician_competency
// (migration 0021_scheduling_calendar), kept local to avoid cross-domain imports.
const (
	CompetencyStatusCurrent = "CURRENT"
	CompetencyStatusExpired = "EXPIRED"
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

// IsCurrent reports whether the competency still covers its equipment type at
// the given instant: status must be CURRENT and the expiry must not have passed.
func (c TechnicianCompetency) IsCurrent(at time.Time) bool {
	return c.Status == CompetencyStatusCurrent && (c.ExpiresAt.IsZero() || !at.After(c.ExpiresAt))
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

// ErrAssignmentNotVerified is the deny-closed L6 gate error: a work order may
// not be assigned to a technician whose dynamic credential or required skill
// matrix is not verified.
var ErrAssignmentNotVerified = errors.New("scheduling skill check denied work order assignment")

// CheckAssignmentSkills is the L6 dynamic inspector credentialing and skill
// matrix verification gate. It fails closed unless the inspector credential is
// APPROVED and within its validity window AND every required skill is covered
// by a CURRENT, unexpired competency.
func CheckAssignmentSkills(now time.Time, credential onboarding.InspectorCredential, requiredSkills []string, competencies []TechnicianCompetency) error {
	if credential.VerificationStatus != onboarding.QualApproved {
		return fmt.Errorf("%w: inspector credential %q is not approved", ErrAssignmentNotVerified, credential.VerificationStatus)
	}
	if !credential.ValidFrom.IsZero() && now.Before(credential.ValidFrom) {
		return fmt.Errorf("%w: inspector credential is not yet valid", ErrAssignmentNotVerified)
	}
	if !credential.ExpiresAt.IsZero() && !now.Before(credential.ExpiresAt) {
		return fmt.Errorf("%w: inspector credential is expired", ErrAssignmentNotVerified)
	}
	for _, skill := range requiredSkills {
		if !hasCurrentCompetency(competencies, skill, now) {
			return fmt.Errorf("%w: technician %s lacks current competency for skill %q", ErrAssignmentNotVerified, credential.InspectorID, skill)
		}
	}
	return nil
}

// AssignEntry gates a work-order assignment on the skill check: the entry is
// marked with CompetencyVerified set only if CheckAssignmentSkills passes;
// denied assignments leave it unverified and return the gate error.
func AssignEntry(now time.Time, assignment *CalendarEntry, credential onboarding.InspectorCredential, requiredSkills []string, competencies []TechnicianCompetency) error {
	if err := CheckAssignmentSkills(now, credential, requiredSkills, competencies); err != nil {
		return err
	}
	assignment.CompetencyVerified = true
	return nil
}

func hasCurrentCompetency(competencies []TechnicianCompetency, skill string, at time.Time) bool {
	for _, c := range competencies {
		if c.EquipmentTypeID == skill && c.IsCurrent(at) {
			return true
		}
	}
	return false
}
