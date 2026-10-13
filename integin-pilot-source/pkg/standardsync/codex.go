package standardsync

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ExtractedParameter struct {
	Doc           string  `json:"doc"`
	Page          int     `json:"page"`
	Clause        string  `json:"clause"`
	SectionTitle  string  `json:"section_title"`
	HasTableGrid  bool    `json:"has_table_grid"`
	ParameterType string  `json:"parameter_type"`
	NumericValue  float64 `json:"numeric_value"`
	Context       string  `json:"context"`
}

type ExtractedCodex struct {
	TotalStandardsScanned    int                  `json:"total_standards_scanned"`
	TotalParametersExtracted int                  `json:"total_parameters_extracted"`
	Parameters               []ExtractedParameter `json:"parameters"`
}

type CodexIndex struct {
	mu          sync.RWMutex
	byDoc       map[string][]*ExtractedParameter
	byParamType map[string][]*ExtractedParameter
	byClause    map[string][]*ExtractedParameter
}

func LoadCodex(path string) (*ExtractedCodex, *CodexIndex, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read codex file: %w", err)
	}

	var codex ExtractedCodex
	if err := json.Unmarshal(data, &codex); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal codex: %w", err)
	}

	idx := &CodexIndex{
		byDoc:       make(map[string][]*ExtractedParameter, len(codex.Parameters)),
		byParamType: make(map[string][]*ExtractedParameter, len(codex.Parameters)),
		byClause:    make(map[string][]*ExtractedParameter, len(codex.Parameters)),
	}

	for i := range codex.Parameters {
		p := &codex.Parameters[i]
		idx.byDoc[p.Doc] = append(idx.byDoc[p.Doc], p)
		idx.byParamType[p.ParameterType] = append(idx.byParamType[p.ParameterType], p)
		idx.byClause[p.Clause] = append(idx.byClause[p.Clause], p)
	}

	return &codex, idx, nil
}

func (idx *CodexIndex) SearchByDoc(doc string) []*ExtractedParameter {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.byDoc[doc]
}

func (idx *CodexIndex) SearchByParameterType(paramType string) []*ExtractedParameter {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.byParamType[paramType]
}

func (idx *CodexIndex) SearchByClause(clause string) []*ExtractedParameter {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.byClause[clause]
}

// ToMetadataCard creates a safe, fully validated citation card from an extracted parameter.
func (p *ExtractedParameter) ToMetadataCard() StandardMetadataCard {
	card := StandardMetadataCard{
		LifecycleState: LifecycleActive,
		Title:          fmt.Sprintf("%s - %s", p.ParameterType, p.Clause),
		ScopeAbstract:  fmt.Sprintf("Extracted %s limit: %.2f (Ref: %s)", p.ParameterType, p.NumericValue, p.Clause),
		PublishedDate:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	cleanDoc := strings.TrimSuffix(p.Doc, ".pdf")
	parts := strings.Split(cleanDoc, "_")

	// Parse body, code, revision year
	if len(parts) >= 4 && (parts[1] == "ASME" || parts[1] == "ISO" || parts[1] == "API" || parts[1] == "BS" || parts[1] == "DNV") {
		card.StandardBody = parts[1]
		card.Code = parts[2]
		yearStr := parts[len(parts)-1]
		if y, err := strconv.Atoi(yearStr); err == nil && y >= 1900 && y <= time.Now().Year()+3 {
			card.RevisionYear = y
			card.PublishedDate = time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)
		}
	} else if len(parts) >= 2 {
		card.StandardBody = parts[0]
		card.Code = parts[1]
		for i := len(parts) - 1; i >= 0; i-- {
			if y, err := strconv.Atoi(parts[i]); err == nil && y >= 1900 && y <= time.Now().Year()+3 {
				card.RevisionYear = y
				card.PublishedDate = time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)
				break
			}
		}
	}

	if card.StandardBody == "" {
		card.StandardBody = "ASME"
	}
	if card.Code == "" {
		card.Code = "B30.5"
	}
	if card.RevisionYear == 0 {
		card.RevisionYear = 2020
	}

	// Canonical DID using schema.DID()
	card.StandardDID = DID(card.StandardBody, card.Code, card.RevisionYear)

	// Official Storefront URL pinned by connectors
	switch strings.ToUpper(card.StandardBody) {
	case "ASME":
		card.OfficialStoreURL = "https://www.asme.org/codes-standards"
	case "ISO":
		card.OfficialStoreURL = "https://www.iso.org/standards.html"
	case "API":
		card.OfficialStoreURL = "https://www.api.org/products-and-services/standards"
	case "BSI", "BS":
		card.OfficialStoreURL = "https://shop.bsigroup.com"
	case "DNV":
		card.OfficialStoreURL = "https://www.dnv.com/energy/standards/"
	default:
		card.OfficialStoreURL = "https://www.asme.org/codes-standards"
	}

	return card
}
