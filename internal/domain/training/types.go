package training

import (
	"errors"
	"time"
)

type CourseStatus string

const (
	CourseDraft   CourseStatus = "DRAFT"
	CourseActive  CourseStatus = "ACTIVE"
	CourseRetired CourseStatus = "RETIRED"
)

type CompetencyStatus string

const (
	CompetencyCurrent CompetencyStatus = "CURRENT"
	CompetencyExpired CompetencyStatus = "EXPIRED"
)

var (
	ErrCourseNotFound     = errors.New("course not found")
	ErrCompetencyNotFound = errors.New("competency not found")
)

type Course struct {
	ID             string       `json:"id"`
	TenantID       string       `json:"tenant_id"`
	OrganizationID string       `json:"organization_id"`
	Title          string       `json:"title"`
	InstructorID   string       `json:"instructor_id,omitempty"`
	Status         CourseStatus `json:"status"`
	CreatedAt      time.Time    `json:"created_at"`
}

type CourseEnrollment struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	OrganizationID string     `json:"organization_id"`
	CourseID       string     `json:"course_id"`
	TechnicianID   string     `json:"technician_id"`
	EnrolledAt     time.Time  `json:"enrolled_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

type Competency struct {
	ID               string           `json:"id"`
	TenantID         string           `json:"tenant_id"`
	OrganizationID   string           `json:"organization_id"`
	TechnicianID     string           `json:"technician_id"`
	EquipmentTypeID  string           `json:"equipment_type_id"`
	CertificationRef string           `json:"certification_ref,omitempty"`
	ExpiresAt        *time.Time       `json:"expires_at,omitempty"`
	Status           CompetencyStatus `json:"status"`
	VerifiedBy       string           `json:"verified_by,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
}

// ISO 9712 Qualification Levels and Methods for NDT Personnel
type NDTLevel string

const (
	NDTLevel1 NDTLevel = "LEVEL_1" // Operating & recording only; no evaluation/signing
	NDTLevel2 NDTLevel = "LEVEL_2" // Technique selection, result interpretation, certificate signing
	NDTLevel3 NDTLevel = "LEVEL_3" // Full technical responsibility, procedure validation, certification
)

var (
	ErrQualificationExpired     = errors.New("iso9712: qualification expired")
	ErrInsufficientNDTLevel     = errors.New("iso9712: Level 1 personnel are not authorized to interpret results or sign certificates")
	ErrLevel3Required           = errors.New("iso9712: Level 3 qualification required for procedure approval")
	ErrInvalidMethod            = errors.New("iso9712: invalid or empty NDT method")
	ErrMissingCertRef           = errors.New("iso9712: missing accreditation certificate reference")
)

type NDTQualification struct {
	TechnicianID     string    `json:"technician_id"`
	Method           string    `json:"method"` // e.g. "UT", "MT", "PT", "RT", "VT", "ET", "PAUT", "TOFD"
	Level            NDTLevel  `json:"level"`
	CertificationRef string    `json:"certification_ref"`
	AccreditingBody  string    `json:"accrediting_body"` // e.g. "BINDT", "PCN", "CSWIP", "ASNT"
	ExpiresAt        time.Time `json:"expires_at"`
}

// ValidateNDTSignOff checks if the technician has the required ISO 9712 competence level
// for the given operational action at the specified timestamp.
func (q NDTQualification) ValidateNDTSignOff(action string, at time.Time) error {
	if q.Method == "" {
		return ErrInvalidMethod
	}
	if q.CertificationRef == "" {
		return ErrMissingCertRef
	}
	if !q.ExpiresAt.IsZero() && at.After(q.ExpiresAt) {
		return ErrQualificationExpired
	}

	switch action {
	case "PERFORM_TEST", "RECORD_RAW_DATA":
		// Level 1, Level 2, and Level 3 permitted
		if q.Level != NDTLevel1 && q.Level != NDTLevel2 && q.Level != NDTLevel3 {
			return ErrInsufficientNDTLevel
		}
		return nil

	case "INTERPRET_RESULTS", "EVALUATE_DEFECT", "SIGN_CERTIFICATE", "ISSUE_REPORT":
		// ISO 9712 §5: Level 1 is strictly forbidden from interpreting results or signing certificates
		if q.Level == NDTLevel1 {
			return ErrInsufficientNDTLevel
		}
		if q.Level != NDTLevel2 && q.Level != NDTLevel3 {
			return ErrInsufficientNDTLevel
		}
		return nil

	case "APPROVE_PROCEDURE", "VALIDATE_TECHNIQUE":
		// ISO 9712 §5.3: Only Level 3 can approve procedures
		if q.Level != NDTLevel3 {
			return ErrLevel3Required
		}
		return nil

	default:
		return errors.New("iso9712: unrecognized NDT sign-off action")
	}
}
