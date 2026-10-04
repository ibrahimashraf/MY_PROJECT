package reinspection

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"integin/internal/shared/types"
)

type CARStatus string

const (
	CAROpen                  CARStatus = "OPEN"
	CARRemediated            CARStatus = "REMEDIATED"
	CARReinspectionScheduled CARStatus = "REINSPECTION_SCHEDULED"
	CARVerifiedClosed        CARStatus = "VERIFIED_CLOSED"
	CARRejected              CARStatus = "REJECTED"
)

var (
	ErrInvalidSeverity            = errors.New("CAR can only be issued for Major or Critical defect severity")
	ErrCARNotOpen                 = errors.New("remediation can only be submitted for an open CAR")
	ErrCARNotRemediated           = errors.New("reinspection verification requires a remediated or scheduled CAR")
	ErrMissingRemediationEvidence = errors.New("remediation submission requires at least one evidence reference")
	ErrReinspectionFailed         = errors.New("reinspection found defects persisting; CAR rejected")
	ErrInvalidIdentity            = errors.New("incomplete CAR identity fields")
)

// CorrectiveActionRequest manages the remediation lifecycle when an inspection
// uncovers a major or critical physical non-conformance.
type CorrectiveActionRequest struct {
	ID                      string          `json:"car_id"`
	TenantID                string          `json:"tenant_id"`
	OrganizationID          string          `json:"organization_id"`
	AssetID                 string          `json:"asset_id"`
	SourceInspectionID      string          `json:"source_inspection_id"`
	SourceFindingID         string          `json:"source_finding_id"`
	DefectSeverity          types.Severity  `json:"defect_severity"`
	DefectDescription       string          `json:"defect_description"`
	Status                  CARStatus       `json:"status"`
	RemediationNotes        string          `json:"remediation_notes,omitempty"`
	RemediationEvidenceRefs []string        `json:"remediation_evidence_refs,omitempty"`
	RemediatedAt            *time.Time      `json:"remediated_at,omitempty"`
	RemediatedBy            string          `json:"remediated_by,omitempty"`
	ReinspectionID          string          `json:"reinspection_id,omitempty"`
	ClosedAt                *time.Time      `json:"closed_at,omitempty"`
	VerifiedBy              string          `json:"verified_by,omitempty"`
	CreatedAt               time.Time       `json:"created_at"`
}

// ReinspectionBinding records the forward link connecting the original failed
// inspection, the CAR, and the targeted re-inspection verification record.
type ReinspectionBinding struct {
	ReinspectionID       string        `json:"reinspection_id"`
	CARID                string        `json:"car_id"`
	OriginalInspectionID string        `json:"original_inspection_id"`
	AssetID              string        `json:"asset_id"`
	TargetItemIDs        []string      `json:"target_item_ids"`
	Outcome              types.Verdict `json:"outcome"`
	ConductedBy          string        `json:"conducted_by"`
	ConductedAt          time.Time     `json:"conducted_at"`
}

// IssueCAR creates a new Corrective Action Request from an inspection finding.
// Only findings with SeverityMajor or SeverityCritical are eligible.
func IssueCAR(
	id string,
	tenantID string,
	organizationID string,
	assetID string,
	sourceInspectionID string,
	sourceFindingID string,
	severity types.Severity,
	description string,
	createdAt time.Time,
) (CorrectiveActionRequest, error) {
	required := map[string]string{
		"car_id": id, "tenant_id": tenantID, "organization_id": organizationID,
		"asset_id": assetID, "source_inspection_id": sourceInspectionID,
		"source_finding_id": sourceFindingID, "defect_description": description,
	}
	for field, val := range required {
		if strings.TrimSpace(val) == "" {
			return CorrectiveActionRequest{}, fmt.Errorf("%w: %s is required", ErrInvalidIdentity, field)
		}
	}

	if severity != types.SeverityMajor && severity != types.SeverityCritical {
		return CorrectiveActionRequest{}, fmt.Errorf("%w: got %s", ErrInvalidSeverity, severity)
	}

	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	return CorrectiveActionRequest{
		ID:                 id,
		TenantID:           tenantID,
		OrganizationID:     organizationID,
		AssetID:            assetID,
		SourceInspectionID: sourceInspectionID,
		SourceFindingID:    sourceFindingID,
		DefectSeverity:     severity,
		DefectDescription:  description,
		Status:             CAROpen,
		CreatedAt:          createdAt.UTC(),
	}, nil
}

// SubmitRemediation records the client or contractor's physical defect repair,
// requiring explanatory notes and at least one proof of remediation evidence reference.
func (c *CorrectiveActionRequest) SubmitRemediation(notes string, evidenceRefs []string, remediatedBy string, at time.Time) error {
	if c.Status != CAROpen && c.Status != CARRejected {
		return ErrCARNotOpen
	}
	if strings.TrimSpace(notes) == "" {
		return errors.New("remediation notes are required")
	}
	if len(evidenceRefs) == 0 {
		return ErrMissingRemediationEvidence
	}
	if strings.TrimSpace(remediatedBy) == "" {
		return errors.New("remediated_by actor identity is required")
	}

	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}

	c.RemediationNotes = notes
	c.RemediationEvidenceRefs = append([]string(nil), evidenceRefs...)
	c.RemediatedAt = &at
	c.RemediatedBy = remediatedBy
	c.Status = CARRemediated
	return nil
}

// ScheduleReinspection transitions the CAR to REINSPECTION_SCHEDULED.
func (c *CorrectiveActionRequest) ScheduleReinspection(reinspectionID string) error {
	if c.Status != CARRemediated {
		return ErrCARNotRemediated
	}
	if strings.TrimSpace(reinspectionID) == "" {
		return errors.New("reinspection_id is required")
	}
	c.ReinspectionID = reinspectionID
	c.Status = CARReinspectionScheduled
	return nil
}

// VerifyOutcome records the technical authority's re-inspection outcome.
// If the verdict is PASS or ADVISORY/MINOR, the CAR is VERIFIED_CLOSED.
// If the defect persists (MAJOR or CRITICAL), the CAR is REJECTED and returned for further work.
func (c *CorrectiveActionRequest) VerifyOutcome(
	reinspectionID string,
	verdict types.Verdict,
	verifiedBy string,
	targetItemIDs []string,
	at time.Time,
) (ReinspectionBinding, error) {
	if c.Status != CARRemediated && c.Status != CARReinspectionScheduled {
		return ReinspectionBinding{}, ErrCARNotRemediated
	}
	if strings.TrimSpace(reinspectionID) == "" || strings.TrimSpace(verifiedBy) == "" {
		return ReinspectionBinding{}, errors.New("reinspection_id and verified_by are required")
	}
	if len(targetItemIDs) == 0 {
		return ReinspectionBinding{}, errors.New("at least one target checklist item ID is required")
	}

	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}

	c.ReinspectionID = reinspectionID
	c.VerifiedBy = verifiedBy

	binding := ReinspectionBinding{
		ReinspectionID:       reinspectionID,
		CARID:                c.ID,
		OriginalInspectionID: c.SourceInspectionID,
		AssetID:              c.AssetID,
		TargetItemIDs:        append([]string(nil), targetItemIDs...),
		Outcome:              verdict,
		ConductedBy:          verifiedBy,
		ConductedAt:          at,
	}

	if verdict == types.VerdictPass || verdict == types.VerdictAdvisory || verdict == types.VerdictMinor {
		c.Status = CARVerifiedClosed
		c.ClosedAt = &at
		return binding, nil
	}

	// Defect was not remediated or failed again: CAR rejected
	c.Status = CARRejected
	return binding, ErrReinspectionFailed
}
