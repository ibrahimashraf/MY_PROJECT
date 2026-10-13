package types

import (
	"errors"
	"fmt"
	"strings"
)

type TenantContext struct {
	TenantID       string `json:"tenant_id"`
	OrganizationID string `json:"organization_id"`
	Environment    string `json:"environment"`
}

func (c TenantContext) Validate() error {
	if strings.TrimSpace(c.TenantID) == "" {
		return errors.New("tenant_id is required")
	}
	if strings.TrimSpace(c.OrganizationID) == "" {
		return errors.New("organization_id is required")
	}
	if c.Environment != EnvironmentTesting && c.Environment != EnvironmentLive {
		return fmt.Errorf("invalid environment: %s", c.Environment)
	}
	return nil
}

const (
	EnvironmentTesting = "TESTING"
	EnvironmentLive    = "LIVE"
)

type InspectionStatus string

const (
	InspectionScheduled     InspectionStatus = "SCHEDULED"
	InspectionAssigned      InspectionStatus = "ASSIGNED"
	InspectionInProgress    InspectionStatus = "IN_PROGRESS"
	InspectionCompleted     InspectionStatus = "COMPLETED"
	InspectionPendingReview InspectionStatus = "PENDING_REVIEW"
	InspectionApproved      InspectionStatus = "APPROVED"
	InspectionRejected      InspectionStatus = "REJECTED"
	InspectionClosed        InspectionStatus = "CLOSED"
)

type Verdict string

const (
	VerdictAdvisory Verdict = "ADVISORY"
	VerdictMinor    Verdict = "MINOR"
	VerdictMajor    Verdict = "MAJOR"
	VerdictCritical Verdict = "CRITICAL"
	VerdictPass     Verdict = "PASS"
)

type Severity string

const (
	SeverityAdvisory Severity = "ADVISORY"
	SeverityMinor    Severity = "MINOR"
	SeverityMajor    Severity = "MAJOR"
	SeverityCritical Severity = "CRITICAL"
)

type DeviceTrustState string

const (
	DevicePending    DeviceTrustState = "PENDING"
	DeviceTrusted    DeviceTrustState = "TRUSTED"
	DeviceRestricted DeviceTrustState = "RESTRICTED"
	DeviceLocked     DeviceTrustState = "LOCKED"
	DeviceRevoked    DeviceTrustState = "REVOKED"
	DeviceRetired    DeviceTrustState = "RETIRED"
)

type ResponseType string

const (
	ResponsePassFail          ResponseType = "PASS_FAIL"
	ResponseNumeric           ResponseType = "NUMERIC"
	ResponseTextSingle        ResponseType = "TEXT_SINGLE"
	ResponseTextMultiline     ResponseType = "TEXT_MULTILINE"
	ResponseImageCapture      ResponseType = "IMAGE_CAPTURE"
	ResponseDropdown          ResponseType = "DROPDOWN"
	ResponseCheckboxGroup     ResponseType = "CHECKBOX_GROUP"
	ResponseRadioGroup        ResponseType = "RADIO_GROUP"
	ResponseNumericGrid       ResponseType = "NUMERIC_GRID"
	ResponseMatrixMeasurement ResponseType = "MATRIX_MEASUREMENT"
	ResponseCalibration       ResponseType = "CALIBRATION"
	ResponseAngleMeasurement  ResponseType = "ANGLE_MEASUREMENT"
	ResponseFileUpload        ResponseType = "FILE_UPLOAD"
	ResponsePressureTest      ResponseType = "PRESSURE_TEST"
)

type CalibrationStatus string

const (
	CalibrationActive     CalibrationStatus = "ACTIVE"
	CalibrationExpired    CalibrationStatus = "EXPIRED"
	CalibrationSuperseded CalibrationStatus = "SUPERSEDED"
)

type ErrorCode string

const (
	ErrUnauthorized      ErrorCode = "UNAUTHORIZED"
	ErrTenantMismatch    ErrorCode = "TENANT_MISMATCH"
	ErrInvalidTransition ErrorCode = "INVALID_TRANSITION"
	ErrConflict          ErrorCode = "CONFLICT"
	ErrHeld              ErrorCode = "HELD"
	ErrRejected          ErrorCode = "REJECTED"
	ErrExpired           ErrorCode = "EXPIRED"
	ErrValidation        ErrorCode = "VALIDATION_ERROR"
)
