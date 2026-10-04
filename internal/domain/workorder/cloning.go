package workorder

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"integin/internal/domain/inspection"
)

var (
	ErrCloneIdentityConflict = errors.New("cloned inspection ID cannot match source inspection ID")
	ErrInvalidCloneTarget    = errors.New("target asset ID and new inspection ID are required for cloning")
	ErrSourceInspectionNil   = errors.New("source inspection is empty or invalid")
)

// LocationSection represents an expandable spatial section grouping scope items
// inside an operational work order.
type LocationSection struct {
	LocationID   string      `json:"location_id"`
	LocationName string      `json:"location_name,omitempty"`
	AreaZone     string      `json:"area_zone,omitempty"`
	ScopeItems   []ScopeItem `json:"scope_items"`
}

// GroupByLocation clusters scope items into deterministic, sorted LocationSections.
func GroupByLocation(items []ScopeItem) []LocationSection {
	if len(items) == 0 {
		return nil
	}

	grouped := make(map[string][]ScopeItem)
	for _, item := range items {
		locID := strings.TrimSpace(item.LocationID)
		if locID == "" {
			locID = "DEFAULT_UNASSIGNED_LOCATION"
		}
		grouped[locID] = append(grouped[locID], item)
	}

	keys := make([]string, 0, len(grouped))
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	sections := make([]LocationSection, 0, len(keys))
	for _, k := range keys {
		locItems := grouped[k]
		// Sort scope items deterministically by ID within location
		sort.Slice(locItems, func(i, j int) bool {
			return locItems[i].ID < locItems[j].ID
		})
		sections = append(sections, LocationSection{
			LocationID: k,
			ScopeItems: locItems,
		})
	}

	return sections
}

// ClonedInspectionDraft holds audit metadata tracking carried-forward values
// across similar asset inspections.
type ClonedInspectionDraft struct {
	SourceInspectionID string               `json:"source_inspection_id"`
	NewInspectionID    string               `json:"new_inspection_id"`
	NewAssetID         string               `json:"new_asset_id"`
	TargetLocationID   string               `json:"target_location_id"`
	CarriedFindings    []inspection.Finding `json:"carried_findings"`
	IsClonedDraft      bool                 `json:"is_cloned_draft"`
	ClonedAt           time.Time            `json:"cloned_at"`
}

// CloneInspectionDraft creates a fresh inspection draft from an existing inspection.
// It deep-copies editable answers, checklist prompts, measured values, units, and
// defect findings, while STRICTLY PURGING all photographic evidence, attachment hashes,
// signatures, and certificates to guarantee complete proof isolation (ISO 17020 §6.2).
func CloneInspectionDraft(
	source inspection.Inspection,
	newInspectionID string,
	newAssetID string,
	targetLocationID string,
	recordedBy string,
	scheduledDate time.Time,
) (inspection.Inspection, ClonedInspectionDraft, error) {
	if strings.TrimSpace(source.ID()) == "" {
		return inspection.Inspection{}, ClonedInspectionDraft{}, ErrSourceInspectionNil
	}
	if strings.TrimSpace(newInspectionID) == "" || strings.TrimSpace(newAssetID) == "" {
		return inspection.Inspection{}, ClonedInspectionDraft{}, ErrInvalidCloneTarget
	}
	if strings.TrimSpace(newInspectionID) == strings.TrimSpace(source.ID()) {
		return inspection.Inspection{}, ClonedInspectionDraft{}, ErrCloneIdentityConflict
	}
	if strings.TrimSpace(recordedBy) == "" {
		recordedBy = "inspector-cloning-service"
	}
	if scheduledDate.IsZero() {
		scheduledDate = time.Now().UTC()
	}

	// 1. Instantiate fresh target inspection in scheduled/draft state
	clone, err := inspection.New(
		newInspectionID,
		source.TenantID(),
		source.OrganizationID(),
		source.Environment(),
		newAssetID,
		source.InspectionType(),
		scheduledDate,
	)
	if err != nil {
		return inspection.Inspection{}, ClonedInspectionDraft{}, fmt.Errorf("create cloned inspection: %w", err)
	}

	// 2. Clone findings with STRICT PROOF ISOLATION
	srcFindings := source.Findings()
	carried := make([]inspection.Finding, 0, len(srcFindings))

	for idx, sf := range srcFindings {
		var measuredVal *float64
		if sf.MeasuredValue != nil {
			v := *sf.MeasuredValue
			measuredVal = &v
		}

		carriedFinding := inspection.Finding{
			ID:            fmt.Sprintf("%s-cf-%03d", newInspectionID, idx+1),
			InspectionID:  newInspectionID,
			AssetID:       newAssetID,
			SectionID:     sf.SectionID,
			ItemID:        sf.ItemID,
			ItemPrompt:    sf.ItemPrompt,
			Response:      sf.Response,
			MeasuredValue: measuredVal,
			MeasuredUnit:  sf.MeasuredUnit,
			Severity:      sf.Severity,
			Notes:         sf.Notes,
			EvidenceRefs:  nil, // GUARANTEED EMPTY: Zero proof bleed across assets
			RecordedBy:    recordedBy,
			RecordedAt:    scheduledDate,
		}
		carried = append(carried, carriedFinding)
	}

	draftMeta := ClonedInspectionDraft{
		SourceInspectionID: source.ID(),
		NewInspectionID:    newInspectionID,
		NewAssetID:         newAssetID,
		TargetLocationID:   targetLocationID,
		CarriedFindings:    carried,
		IsClonedDraft:      true,
		ClonedAt:           scheduledDate,
	}

	return clone, draftMeta, nil
}
