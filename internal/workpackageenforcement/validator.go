// Package workpackageenforcement validates signed inspection payloads against server-approved work packages.
package workpackageenforcement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"integin/internal/domain/workpackage"
)

var ErrPayloadBinding = errors.New("inspection payload does not satisfy approved work-package binding")

// Resolver is implemented by the tenant-scoped PostgreSQL work-package repository.
type Resolver interface {
	GetApproved(ctx context.Context, tenantID, organizationID, packageID string, version int) (workpackage.Package, error)
}

type Validator struct {
	resolver Resolver
}

func NewValidator(resolver Resolver) (*Validator, error) {
	if resolver == nil {
		return nil, errors.New("work-package validator requires approved package resolver")
	}
	return &Validator{resolver: resolver}, nil
}

// ValidateInspectionPayload enforces the exact approved package against an already signed payload.
// It does not mutate workflow state and is intended to run after transaction signature verification.
func (v *Validator) ValidateInspectionPayload(ctx context.Context, tenantID, organizationID, entityID string, payload []byte) error {
	var document inspectionPayload
	if err := json.Unmarshal(payload, &document); err != nil {
		return fmt.Errorf("%w: invalid inspection JSON: %v", ErrPayloadBinding, err)
	}
	if strings.TrimSpace(entityID) == "" || document.InspectionID != entityID {
		return fmt.Errorf("%w: inspection entity binding mismatch", ErrPayloadBinding)
	}
	if strings.TrimSpace(document.PackageID) == "" || document.PackageVersion <= 0 || strings.TrimSpace(document.PackageHash) == "" {
		return fmt.Errorf("%w: package id, version, and hash are required", ErrPayloadBinding)
	}
	if len(document.FieldSequence) == 0 {
		return fmt.Errorf("%w: package field sequence is required", ErrPayloadBinding)
	}
	approved, err := v.resolver.GetApproved(ctx, tenantID, organizationID, document.PackageID, document.PackageVersion)
	if err != nil {
		return fmt.Errorf("%w: resolve approved package: %v", ErrPayloadBinding, err)
	}
	values, err := valuesForPackage(approved, document.Findings)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPayloadBinding, err)
	}
	binding := workpackage.SubmissionBinding{
		PackageID:        document.PackageID,
		PackageVersion:   document.PackageVersion,
		PackageHash:      document.PackageHash,
		CapturedFieldIDs: document.FieldSequence,
	}
	if err := approved.ValidateSubmission(binding, values); err != nil {
		return fmt.Errorf("%w: %v", ErrPayloadBinding, err)
	}
	return nil
}

type inspectionPayload struct {
	InspectionID   string    `json:"inspection_id"`
	PackageID      string    `json:"work_package_id"`
	PackageVersion int       `json:"work_package_version"`
	PackageHash    string    `json:"work_package_hash"`
	FieldSequence  []string  `json:"work_package_field_sequence"`
	Findings       []finding `json:"findings"`
}

type finding struct {
	ItemID   string `json:"item_id"`
	Response string `json:"response"`
}

func valuesForPackage(p workpackage.Package, findings []finding) (map[string]any, error) {
	fields := make(map[string]workpackage.FieldDefinition)
	for _, section := range p.Sections {
		for _, field := range section.Fields {
			fields[field.ID] = field
		}
	}
	values := make(map[string]any, len(findings))
	for _, finding := range findings {
		field, exists := fields[finding.ItemID]
		if !exists {
			return nil, fmt.Errorf("finding item %q is not in approved package", finding.ItemID)
		}
		if _, duplicate := values[finding.ItemID]; duplicate {
			return nil, fmt.Errorf("duplicate finding item %q", finding.ItemID)
		}
		value, err := convertResponse(field.Type, finding.Response)
		if err != nil {
			return nil, fmt.Errorf("finding item %q: %w", finding.ItemID, err)
		}
		values[finding.ItemID] = value
	}
	return values, nil
}

func convertResponse(fieldType workpackage.FieldType, response string) (any, error) {
	switch fieldType {
	case workpackage.FieldText, workpackage.FieldChoice, workpackage.FieldPassFailNA:
		return response, nil
	case workpackage.FieldNumber:
		number, err := strconv.ParseFloat(response, 64)
		if err != nil {
			return nil, errors.New("response must be numeric")
		}
		return number, nil
	case workpackage.FieldBoolean:
		boolean, err := strconv.ParseBool(response)
		if err != nil {
			return nil, errors.New("response must be boolean")
		}
		return boolean, nil
	default:
		return nil, errors.New("response field type is unsupported")
	}
}
