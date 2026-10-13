package environment

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type Name string

const (
	Testing Name = "TESTING"
	Live    Name = "LIVE"
)

type ReleaseState string

const (
	ReleaseDraft      ReleaseState = "DRAFT"
	ReleaseApproved   ReleaseState = "APPROVED"
	ReleasePromoted   ReleaseState = "PROMOTED"
	ReleaseHealthy    ReleaseState = "HEALTHY"
	ReleaseRolledBack ReleaseState = "ROLLED_BACK"
	ReleaseFailed     ReleaseState = "FAILED"
)

type Event struct {
	Type      string
	ReleaseID string
	From      Name
	To        Name
	At        time.Time
	Reason    string
}
type RecoveryPoint struct {
	ID          string
	Environment Name
	ReleaseID   string
	CreatedAt   time.Time
	StorageKey  string
}
type Release struct {
	ID              string
	Version         string
	State           ReleaseState
	Environment     Name
	SelectedTenants []string
	Events          []Event
	RecoveryPoints  []RecoveryPoint
}

func NewRelease(id, version string) (Release, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(version) == "" {
		return Release{}, errors.New("release id and version are required")
	}
	return Release{ID: id, Version: version, State: ReleaseDraft, Environment: Testing}, nil
}
func (r *Release) Approve() error {
	if r.State != ReleaseDraft {
		return fmt.Errorf("release must be draft, got %s", r.State)
	}
	r.State = ReleaseApproved
	r.Events = append(r.Events, Event{Type: "ReleaseApproved", ReleaseID: r.ID, From: Testing, To: Testing, At: time.Now().UTC()})
	return nil
}
func (r *Release) Promote(tenants []string) error {
	if r.State != ReleaseApproved {
		return fmt.Errorf("release must be approved, got %s", r.State)
	}
	if len(tenants) == 0 {
		return errors.New("selected tenants are required")
	}
	r.SelectedTenants = append([]string(nil), tenants...)
	r.State, r.Environment = ReleasePromoted, Live
	r.Events = append(r.Events, Event{Type: "ReleasePromoted", ReleaseID: r.ID, From: Testing, To: Live, At: time.Now().UTC()})
	return nil
}
func (r *Release) HealthCheck(check func() error) error {
	if r.State != ReleasePromoted {
		return fmt.Errorf("release must be promoted, got %s", r.State)
	}
	if check == nil {
		return errors.New("health check is required")
	}
	if err := check(); err != nil {
		r.State = ReleaseFailed
		r.Events = append(r.Events, Event{Type: "ReleaseHealthFailed", ReleaseID: r.ID, From: Live, To: Live, At: time.Now().UTC(), Reason: err.Error()})
		return err
	}
	r.State = ReleaseHealthy
	r.Events = append(r.Events, Event{Type: "ReleaseHealthy", ReleaseID: r.ID, From: Live, To: Live, At: time.Now().UTC()})
	return nil
}
func (r *Release) Rollback(reason string) error {
	if r.State != ReleasePromoted && r.State != ReleaseHealthy && r.State != ReleaseFailed {
		return fmt.Errorf("release cannot rollback from %s", r.State)
	}
	if strings.TrimSpace(reason) == "" {
		return errors.New("rollback reason is required")
	}
	r.State, r.Environment = ReleaseRolledBack, Testing
	r.Events = append(r.Events, Event{Type: "ReleaseRolledBack", ReleaseID: r.ID, From: Live, To: Testing, At: time.Now().UTC(), Reason: reason})
	return nil
}
func (r *Release) AddRecoveryPoint(point RecoveryPoint) error {
	if point.ID == "" || point.StorageKey == "" {
		return errors.New("recovery point id and storage key are required")
	}
	point.ReleaseID = r.ID
	if point.Environment == "" {
		point.Environment = r.Environment
	}
	if point.CreatedAt.IsZero() {
		point.CreatedAt = time.Now().UTC()
	}
	r.RecoveryPoints = append(r.RecoveryPoints, point)
	return nil
}
