package certificatetemplate

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrTemplateNotApproved  = errors.New("certificate template version is not approved")
	ErrRequiredValueMissing = errors.New("required certificate value is unavailable")
	ErrValueDoesNotFit      = errors.New("certificate value exceeds the approved bounded cell policy")
)

type CanonicalValues map[BindingKey]string

type ResolvedCell struct {
	CellID       string
	BindingKey   BindingKey
	Applicable   bool
	Required     bool
	RenderedText string
}

type ResolvedTemplate struct {
	TemplateID   string
	TemplateCode string
	Version      int
	Cells        []ResolvedCell
}

func Resolve(record DefinitionRecord, values CanonicalValues) (ResolvedTemplate, error) {
	if record.Status != Approved {
		return ResolvedTemplate{}, ErrTemplateNotApproved
	}
	if err := record.Definition.Validate(DefaultCatalog()); err != nil {
		return ResolvedTemplate{}, err
	}
	resolved := ResolvedTemplate{TemplateID: record.ID, TemplateCode: record.TemplateCode, Version: record.Version}
	for _, cell := range record.SortedCells() {
		applicable, err := applies(cell.Applicability, values)
		if err != nil {
			return ResolvedTemplate{}, fmt.Errorf("cell %s: %w", cell.ID, err)
		}
		item := ResolvedCell{CellID: cell.ID, BindingKey: cell.BindingKey, Applicable: applicable, Required: cell.Required}
		if !applicable {
			resolved.Cells = append(resolved.Cells, item)
			continue
		}
		text, err := resolveCellText(cell, values)
		if err != nil {
			return ResolvedTemplate{}, fmt.Errorf("cell %s: %w", cell.ID, err)
		}
		if cell.Required && strings.TrimSpace(text) == "" {
			return ResolvedTemplate{}, fmt.Errorf("cell %s: %w", cell.ID, ErrRequiredValueMissing)
		}
		if err := validateBoundedText(cell, text); err != nil {
			return ResolvedTemplate{}, fmt.Errorf("cell %s: %w", cell.ID, err)
		}
		item.RenderedText = text
		resolved.Cells = append(resolved.Cells, item)
	}
	return resolved, nil
}

func applies(applicability *Applicability, values CanonicalValues) (bool, error) {
	if applicability == nil {
		return true, nil
	}
	actual, ok := values[applicability.BindingKey]
	if !ok {
		return false, fmt.Errorf("applicability value %q is unavailable", applicability.BindingKey)
	}
	switch applicability.Operator {
	case Equals:
		return actual == applicability.Literal, nil
	case NotEquals:
		return actual != applicability.Literal, nil
	default:
		return false, errors.New("unsupported applicability operator")
	}
}

func resolveCellText(cell Cell, values CanonicalValues) (string, error) {
	switch cell.Kind {
	case StaticTextCell:
		return cell.StaticText, nil
	case TextCell, DateCell:
		return values[cell.BindingKey], nil
	case CheckboxCell:
		value := values[cell.BindingKey]
		mapped, ok := cell.CheckboxValues[value]
		if !ok && value != "" {
			return "", fmt.Errorf("checkbox map has no approved value for %q", value)
		}
		return mapped, nil
	case RepeatingRegion:
		return "", errors.New("repeating certificate data source is not available")
	default:
		return "", errors.New("unsupported certificate cell kind")
	}
}

func validateBoundedText(cell Cell, value string) error {
	if cell.Kind == CheckboxCell || cell.Kind == RepeatingRegion {
		return nil
	}
	lines := strings.Split(value, "\n")
	if cell.FitPolicy == SingleLineRequired && len(lines) != 1 {
		return ErrValueDoesNotFit
	}
	if cell.FitPolicy == WrapRequired && len(lines) > cell.MaxLines {
		return ErrValueDoesNotFit
	}
	// Actual glyph measurement belongs to the future PDF renderer. It must use
	// this same no-clip/no-truncation policy before a certificate can issue.
	return nil
}

func InspectionValues(id, workOrderID, assetID, inspectorID, lifecycleState string, revision int, finalizationState string, createdAt, updatedAt time.Time) CanonicalValues {
	return CanonicalValues{
		InspectionID:                id,
		InspectionWorkOrderID:       workOrderID,
		InspectionAssetID:           assetID,
		InspectionInspectorID:       inspectorID,
		InspectionLifecycleState:    lifecycleState,
		InspectionRevision:          fmt.Sprint(revision),
		InspectionFinalizationState: finalizationState,
		InspectionCreatedAt:         createdAt.UTC().Format(time.RFC3339),
		InspectionUpdatedAt:         updatedAt.UTC().Format(time.RFC3339),
	}
}
