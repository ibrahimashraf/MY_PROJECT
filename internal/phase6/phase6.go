package phase6

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"integin/internal/advisory"
)

type TemplateReference struct {
	ID            string
	TenantID      string
	StandardCodes []string
	ClauseRefs    []string
}
type RegulationChange struct {
	ID             string
	TenantID       string
	StandardCode   string
	OldVersion     string
	NewVersion     string
	ChangedClauses []string
	Summary        string
}
type Impact struct {
	TemplateID      string
	TenantID        string
	AffectedClauses []string
	Advisory        advisory.Insight
}

func AnalyzeRegulation(change RegulationChange, templates []TemplateReference, now time.Time) ([]Impact, error) {
	if change.ID == "" || change.TenantID == "" || change.StandardCode == "" || change.NewVersion == "" {
		return nil, errors.New("change id, tenant, standard, and new version are required")
	}
	if now.IsZero() {
		now = time.Now()
	}
	impacts := make([]Impact, 0)
	for _, template := range templates {
		if template.TenantID != change.TenantID {
			continue
		}
		if !contains(template.StandardCodes, change.StandardCode) {
			continue
		}
		affected := intersect(template.ClauseRefs, change.ChangedClauses)
		insight := advisory.Insight{ID: change.ID + "-" + template.ID, TenantID: change.TenantID, Lens: string(advisory.ZoneRegulation), Title: "Regulation change may affect template", Summary: change.Summary, Severity: "ADVISORY", Confidence: 1, Rationale: fmt.Sprintf("Standard %s changed from %s to %s; template references were matched deterministically.", change.StandardCode, change.OldVersion, change.NewVersion), EvidenceRefs: append([]string(nil), change.ChangedClauses...), Provider: "deterministic-analyzer", Model: "none", PromptVersion: "none", Blocking: false, CreatedAt: now.UTC()}
		impacts = append(impacts, Impact{TemplateID: template.ID, TenantID: template.TenantID, AffectedClauses: affected, Advisory: insight})
	}
	return impacts, nil
}
func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
func intersect(left, right []string) []string {
	result := make([]string, 0)
	for _, value := range left {
		if contains(right, value) {
			result = append(result, value)
		}
	}
	return result
}

type Conflict struct {
	Field  string
	Base   any
	Local  any
	Remote any
}
type MergeResult struct {
	Merged    map[string]any
	Conflicts []Conflict
}

func MergeFields(base, local, remote map[string]any) MergeResult {
	result := MergeResult{Merged: make(map[string]any)}
	keys := make(map[string]bool)
	for key := range base {
		keys[key] = true
	}
	for key := range local {
		keys[key] = true
	}
	for key := range remote {
		keys[key] = true
	}
	for key := range keys {
		baseValue := base[key]
		localValue, localOK := local[key]
		remoteValue, remoteOK := remote[key]
		switch {
		case reflect.DeepEqual(localValue, remoteValue):
			if localOK {
				result.Merged[key] = localValue
			}
		case reflect.DeepEqual(localValue, baseValue):
			if remoteOK {
				result.Merged[key] = remoteValue
			}
		case reflect.DeepEqual(remoteValue, baseValue):
			if localOK {
				result.Merged[key] = localValue
			}
		default:
			result.Conflicts = append(result.Conflicts, Conflict{Field: key, Base: baseValue, Local: localValue, Remote: remoteValue})
		}
	}
	return result
}

type RequestStatus string

const (
	Submitted RequestStatus = "SUBMITTED"
	InReview  RequestStatus = "IN_REVIEW"
	Approved  RequestStatus = "APPROVED"
	Scheduled RequestStatus = "SCHEDULED"
	Completed RequestStatus = "COMPLETED"
	Rejected  RequestStatus = "REJECTED"
)

type ClientRequest struct {
	ID              string
	TenantID        string
	ClientID        string
	AssetID         string
	Description     string
	Status          RequestStatus
	RejectionReason string
	ReviewedBy      string
	ScheduledFor    time.Time
}

func NewClientRequest(id, tenantID, clientID, assetID, description string) (ClientRequest, error) {
	for field, value := range map[string]string{"id": id, "tenant_id": tenantID, "client_id": clientID, "asset_id": assetID, "description": description} {
		if strings.TrimSpace(value) == "" {
			return ClientRequest{}, errors.New(field + " is required")
		}
	}
	return ClientRequest{ID: id, TenantID: tenantID, ClientID: clientID, AssetID: assetID, Description: description, Status: Submitted}, nil
}
func (r *ClientRequest) Review(actorTenant, reviewerID string) error {
	if r.Status != Submitted {
		return errors.New("request must be submitted")
	}
	if actorTenant != r.TenantID || reviewerID == "" {
		return errors.New("reviewer tenant or identity mismatch")
	}
	r.Status, r.ReviewedBy = InReview, reviewerID
	return nil
}
func (r *ClientRequest) Approve(actorTenant string) error {
	if r.Status != InReview || actorTenant != r.TenantID {
		return errors.New("request cannot be approved")
	}
	r.Status = Approved
	return nil
}
func (r *ClientRequest) Schedule(actorTenant string, at time.Time) error {
	if r.Status != Approved || actorTenant != r.TenantID || at.IsZero() {
		return errors.New("request cannot be scheduled")
	}
	r.Status, r.ScheduledFor = Scheduled, at.UTC()
	return nil
}
func (r *ClientRequest) Complete(actorTenant string) error {
	if r.Status != Scheduled || actorTenant != r.TenantID {
		return errors.New("request cannot be completed")
	}
	r.Status = Completed
	return nil
}
func (r *ClientRequest) Reject(actorTenant, reason string) error {
	if (r.Status != Submitted && r.Status != InReview) || actorTenant != r.TenantID || strings.TrimSpace(reason) == "" {
		return errors.New("request cannot be rejected")
	}
	r.Status, r.RejectionReason = Rejected, reason
	return nil
}
