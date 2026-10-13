package inspection

import (
	"fmt"
	"strings"
	"time"

	"integin/internal/shared/events"
	"integin/internal/shared/types"
)

// DomainError is a deterministic engine error with a shared-kernel code.
type DomainError struct {
	Code    types.ErrorCode
	Message string
}

func (e DomainError) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

// Finding is discipline-neutral inspection evidence. The engine does not
// interpret the response type; templates and discipline-specific adapters do.
type Finding struct {
	ID            string         `json:"id"`
	InspectionID  string         `json:"inspection_id"`
	AssetID       string         `json:"asset_id"`
	SectionID     string         `json:"section_id"`
	ItemID        string         `json:"item_id"`
	ItemPrompt    string         `json:"item_prompt"`
	Response      string         `json:"response"`
	MeasuredValue *float64       `json:"measured_value,omitempty"`
	MeasuredUnit  string         `json:"measured_unit,omitempty"`
	Severity      types.Severity `json:"severity,omitempty"`
	Notes         string         `json:"notes,omitempty"`
	EvidenceRefs  []string       `json:"evidence_refs,omitempty"`
	RecordedBy    string         `json:"recorded_by"`
	RecordedAt    time.Time      `json:"recorded_at"`
}

func (f Finding) validate() error {
	required := map[string]string{
		"id": f.ID, "inspection_id": f.InspectionID, "asset_id": f.AssetID,
		"section_id": f.SectionID, "item_id": f.ItemID, "item_prompt": f.ItemPrompt,
		"response": f.Response, "recorded_by": f.RecordedBy,
	}
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return DomainError{Code: types.ErrValidation, Message: name + " is required"}
		}
	}
	switch f.Severity {
	case "", types.SeverityAdvisory, types.SeverityMinor, types.SeverityMajor, types.SeverityCritical:
	default:
		return DomainError{Code: types.ErrValidation, Message: "invalid finding severity"}
	}
	return nil
}

type RevisionSnapshot struct {
	Number         int                    `json:"number"`
	Status         types.InspectionStatus `json:"status"`
	Findings       []Finding              `json:"findings"`
	OverallResult  types.Verdict          `json:"overall_result"`
	Notes          string                 `json:"notes,omitempty"`
	ReturnedReason string                 `json:"returned_reason,omitempty"`
}

// Inspection is an aggregate whose state changes are emitted as immutable
// domain events. Revision snapshots are append-only and never rewritten.
type Inspection struct {
	id             string
	tenantID       string
	organizationID string
	environment    string
	rootAssetID    string
	inspectionType string
	scheduledDate  time.Time
	assignedTo     string
	status         types.InspectionStatus
	revision       int
	findings       []Finding
	overallResult  types.Verdict
	notes          string
	returnedReason string
	snapshots      []RevisionSnapshot
	emitted        []events.Envelope
}

func New(id, tenantID, organizationID, environment, rootAssetID, inspectionType string, scheduledDate time.Time) (Inspection, error) {
	required := map[string]string{"id": id, "tenant_id": tenantID, "organization_id": organizationID, "root_asset_id": rootAssetID, "inspection_type": inspectionType}
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return Inspection{}, DomainError{Code: types.ErrValidation, Message: name + " is required"}
		}
	}
	if scheduledDate.IsZero() {
		return Inspection{}, DomainError{Code: types.ErrValidation, Message: "scheduled_date is required"}
	}
	context := types.TenantContext{TenantID: tenantID, OrganizationID: organizationID, Environment: environment}
	if err := context.Validate(); err != nil {
		return Inspection{}, err
	}
	return Inspection{id: id, tenantID: tenantID, organizationID: organizationID, environment: environment, rootAssetID: rootAssetID, inspectionType: inspectionType, scheduledDate: scheduledDate.UTC(), status: types.InspectionScheduled, revision: 1}, nil
}

func (i Inspection) ID() string                          { return i.id }
func (i Inspection) TenantID() string                    { return i.tenantID }
func (i Inspection) OrganizationID() string              { return i.organizationID }
func (i Inspection) Environment() string                 { return i.environment }
func (i Inspection) RootAssetID() string                 { return i.rootAssetID }
func (i Inspection) InspectionType() string              { return i.inspectionType }
func (i Inspection) ScheduledDate() time.Time            { return i.scheduledDate }
func (i Inspection) AssignedTo() string                  { return i.assignedTo }
func (i Inspection) Status() types.InspectionStatus      { return i.status }
func (i Inspection) Revision() int                       { return i.revision }
func (i Inspection) OverallResult() types.Verdict        { return i.overallResult }
func (i Inspection) Notes() string                       { return i.notes }
func (i Inspection) ReturnedReason() string              { return i.returnedReason }
func (i Inspection) Findings() []Finding                 { return cloneFindings(i.findings) }
func (i Inspection) RevisionHistory() []RevisionSnapshot { return cloneSnapshots(i.snapshots) }
func (i Inspection) Events() []events.Envelope           { return append([]events.Envelope(nil), i.emitted...) }

func (i *Inspection) Assign(inspectorID string) error {
	if err := i.requireStatus(types.InspectionScheduled); err != nil {
		return err
	}
	if strings.TrimSpace(inspectorID) == "" {
		return DomainError{Code: types.ErrValidation, Message: "inspector_id is required"}
	}
	event, err := i.makeEvent("InspectionAssigned", map[string]string{"inspector_id": inspectorID})
	if err != nil {
		return err
	}
	i.assignedTo, i.status = inspectorID, types.InspectionAssigned
	i.emitted = append(i.emitted, event)
	return nil
}

func (i *Inspection) Start() error {
	if err := i.requireStatus(types.InspectionAssigned); err != nil {
		return err
	}
	event, err := i.makeEvent(events.InspectionStarted, map[string]string{"inspection_id": i.id, "revision": fmt.Sprint(i.revision)})
	if err != nil {
		return err
	}
	i.status = types.InspectionInProgress
	i.emitted = append(i.emitted, event)
	return nil
}

func (i *Inspection) RecordFinding(finding Finding) error {
	if err := i.requireStatus(types.InspectionInProgress); err != nil {
		return err
	}
	if finding.InspectionID == "" {
		finding.InspectionID = i.id
	}
	if finding.RecordedAt.IsZero() {
		finding.RecordedAt = time.Now().UTC()
	} else {
		finding.RecordedAt = finding.RecordedAt.UTC()
	}
	if err := finding.validate(); err != nil {
		return err
	}
	if finding.InspectionID != i.id {
		return DomainError{Code: types.ErrTenantMismatch, Message: "finding belongs to another inspection"}
	}
	event, err := i.makeEvent(events.FindingRecorded, finding)
	if err != nil {
		return err
	}
	i.findings = append(i.findings, cloneFinding(finding))
	i.emitted = append(i.emitted, event)
	return nil
}

func (i *Inspection) Complete(notes string) error {
	if err := i.requireStatus(types.InspectionInProgress); err != nil {
		return err
	}
	verdict := ComputeVerdict(i.findings)
	event, err := i.makeEvent("InspectionCompleted", map[string]any{"inspection_id": i.id, "revision": i.revision, "overall_result": verdict, "notes": notes})
	if err != nil {
		return err
	}
	i.overallResult, i.notes, i.status = verdict, notes, types.InspectionCompleted
	i.emitted = append(i.emitted, event)
	return nil
}

func (i *Inspection) SubmitForReview() error {
	if err := i.requireStatus(types.InspectionCompleted); err != nil {
		return err
	}
	event, err := i.makeEvent(events.InspectionSubmitted, map[string]any{"inspection_id": i.id, "revision": i.revision, "overall_result": i.overallResult})
	if err != nil {
		return err
	}
	i.status = types.InspectionPendingReview
	i.emitted = append(i.emitted, event)
	return nil
}

func (i *Inspection) Approve(reviewerID string) error {
	if err := i.requireStatus(types.InspectionPendingReview); err != nil {
		return err
	}
	if strings.TrimSpace(reviewerID) == "" {
		return DomainError{Code: types.ErrValidation, Message: "reviewer_id is required"}
	}
	if reviewerID == i.assignedTo {
		return DomainError{Code: types.ErrUnauthorized, Message: "inspector cannot approve own inspection"}
	}
	event, err := i.makeEvent(events.InspectionApproved, map[string]string{"inspection_id": i.id, "reviewer_id": reviewerID})
	if err != nil {
		return err
	}
	i.status = types.InspectionApproved
	i.emitted = append(i.emitted, event)
	return nil
}

func (i *Inspection) ReturnForRevision(reviewerID, reason string) error {
	if err := i.requireStatus(types.InspectionPendingReview); err != nil {
		return err
	}
	if strings.TrimSpace(reviewerID) == "" || strings.TrimSpace(reason) == "" {
		return DomainError{Code: types.ErrValidation, Message: "reviewer_id and reason are required"}
	}
	if reviewerID == i.assignedTo {
		return DomainError{Code: types.ErrUnauthorized, Message: "inspector cannot return own inspection"}
	}
	event, err := i.makeEvent(events.InspectionReturned, map[string]any{"inspection_id": i.id, "revision": i.revision, "reviewer_id": reviewerID, "reason": reason})
	if err != nil {
		return err
	}
	i.snapshots = append(i.snapshots, i.snapshot(types.InspectionRejected, reason))
	i.revision++
	i.findings = nil
	i.overallResult, i.notes, i.returnedReason, i.status = "", "", reason, types.InspectionInProgress
	i.emitted = append(i.emitted, event)
	return nil
}

func (i *Inspection) Close() error {
	if err := i.requireStatus(types.InspectionApproved); err != nil {
		return err
	}
	event, err := i.makeEvent("InspectionClosed", map[string]any{"inspection_id": i.id, "revision": i.revision, "overall_result": i.overallResult})
	if err != nil {
		return err
	}
	i.status = types.InspectionClosed
	i.emitted = append(i.emitted, event)
	return nil
}

func (i Inspection) requireStatus(expected types.InspectionStatus) error {
	if i.status != expected {
		return DomainError{Code: types.ErrInvalidTransition, Message: fmt.Sprintf("cannot transition from %s; expected %s", i.status, expected)}
	}
	return nil
}

func (i *Inspection) makeEvent(eventType string, payload any) (events.Envelope, error) {
	eventID := fmt.Sprintf("%s-%d-%d", i.id, i.revision, len(i.emitted)+1)
	return events.NewEnvelope(eventID, eventType, i.tenantID, i.organizationID, i.environment, "inspection", i.id, payload, time.Now().UTC())
}

func (i Inspection) snapshot(status types.InspectionStatus, reason string) RevisionSnapshot {
	return RevisionSnapshot{Number: i.revision, Status: status, Findings: cloneFindings(i.findings), OverallResult: i.overallResult, Notes: i.notes, ReturnedReason: reason}
}

// ComputeVerdict is an associative severity fold. No findings means PASS;
// otherwise the strongest recorded severity determines the verdict.
func ComputeVerdict(findings []Finding) types.Verdict {
	strongest := 0
	verdict := types.VerdictPass
	for _, finding := range findings {
		rank, candidate := severityRank(finding.Severity)
		if rank > strongest {
			strongest, verdict = rank, candidate
		}
	}
	return verdict
}

func severityRank(severity types.Severity) (int, types.Verdict) {
	switch severity {
	case types.SeverityCritical:
		return 4, types.VerdictCritical
	case types.SeverityMajor:
		return 3, types.VerdictMajor
	case types.SeverityMinor:
		return 2, types.VerdictMinor
	case types.SeverityAdvisory:
		return 1, types.VerdictAdvisory
	default:
		return 0, types.VerdictPass
	}
}

func cloneFinding(f Finding) Finding {
	cloned := f
	cloned.EvidenceRefs = append([]string(nil), f.EvidenceRefs...)
	if f.MeasuredValue != nil {
		value := *f.MeasuredValue
		cloned.MeasuredValue = &value
	}
	return cloned
}

func cloneFindings(findings []Finding) []Finding {
	cloned := make([]Finding, len(findings))
	for index, finding := range findings {
		cloned[index] = cloneFinding(finding)
	}
	return cloned
}

func cloneSnapshots(snapshots []RevisionSnapshot) []RevisionSnapshot {
	cloned := make([]RevisionSnapshot, len(snapshots))
	for index, snapshot := range snapshots {
		cloned[index] = snapshot
		cloned[index].Findings = cloneFindings(snapshot.Findings)
	}
	return cloned
}
