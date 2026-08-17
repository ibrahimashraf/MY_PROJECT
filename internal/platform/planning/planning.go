package planning

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type WorkOrderState string

const (
	Draft      WorkOrderState = "DRAFT"
	Planned    WorkOrderState = "PLANNED"
	Assigned   WorkOrderState = "ASSIGNED"
	InProgress WorkOrderState = "IN_PROGRESS"
	Completed  WorkOrderState = "COMPLETED"
	Cancelled  WorkOrderState = "CANCELLED"
)

type WorkOrder struct {
	ID                   string
	TenantID             string
	OrganizationID       string
	AssetID              string
	ScopeID              string
	RequiredCompetencies []string
	State                WorkOrderState
	AssignedUserID       string
	CreatedAt            time.Time
}
type User struct {
	ID             string
	TenantID       string
	OrganizationID string
	Active         bool
	Competencies   map[string]bool
	Scopes         map[string]bool
}

func NewWorkOrder(id, tenantID, organizationID, assetID, scopeID string, competencies []string) (WorkOrder, error) {
	for field, value := range map[string]string{"id": id, "tenant_id": tenantID, "organization_id": organizationID, "asset_id": assetID, "scope_id": scopeID} {
		if strings.TrimSpace(value) == "" {
			return WorkOrder{}, errors.New(field + " is required")
		}
	}
	return WorkOrder{ID: id, TenantID: tenantID, OrganizationID: organizationID, AssetID: assetID, ScopeID: scopeID, RequiredCompetencies: append([]string(nil), competencies...), State: Draft, CreatedAt: time.Now().UTC()}, nil
}
func (w *WorkOrder) Plan() error {
	if w.State != Draft {
		return fmt.Errorf("work order must be draft, got %s", w.State)
	}
	w.State = Planned
	return nil
}
func (w *WorkOrder) Assign(user User) error {
	if w.State != Planned {
		return fmt.Errorf("work order must be planned, got %s", w.State)
	}
	if !user.Active {
		return errors.New("user is inactive")
	}
	if user.TenantID != w.TenantID || user.OrganizationID != w.OrganizationID {
		return errors.New("user organization scope mismatch")
	}
	if !user.Scopes[w.ScopeID] && !user.Scopes["*"] {
		return errors.New("user lacks work-order scope")
	}
	for _, competency := range w.RequiredCompetencies {
		if !user.Competencies[competency] {
			return fmt.Errorf("user lacks competency %s", competency)
		}
	}
	w.AssignedUserID = user.ID
	w.State = Assigned
	return nil
}
func (w *WorkOrder) Start() error {
	if w.State != Assigned {
		return fmt.Errorf("work order must be assigned, got %s", w.State)
	}
	w.State = InProgress
	return nil
}
func (w *WorkOrder) Complete() error {
	if w.State != InProgress {
		return fmt.Errorf("work order must be in progress, got %s", w.State)
	}
	w.State = Completed
	return nil
}
func (w *WorkOrder) Cancel() error {
	if w.State == Completed || w.State == Cancelled {
		return fmt.Errorf("work order cannot cancel from %s", w.State)
	}
	w.State = Cancelled
	return nil
}
