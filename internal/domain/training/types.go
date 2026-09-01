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
