package jurisdictions

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	// ErrUnknownStandardBody reports a standard whose issuing body is outside
	// the legal-hierarchy classification table.
	ErrUnknownStandardBody = errors.New("standard body is not classified in the legal hierarchy")
	// ErrEmptyStandardSet reports a preemption request with no standards.
	ErrEmptyStandardSet = errors.New("preemption requires at least one standard")
)

// LegalHierarchyLevel is the SDO/national/site precedence tier of a governing
// standard, per the INTEGIN legal-regime hierarchy.
type LegalHierarchyLevel int

const (
	// LevelInternational is SDO Level 1 (ISO, IEC).
	LevelInternational LegalHierarchyLevel = 1
	// LevelNational is Conformity Body Level 2 (OSHA, SASO, ASME, ADNOC).
	LevelNational LegalHierarchyLevel = 2
	// LevelSiteJurisdiction is Level 3: site/regional operator rules
	// (client HSEMS, Aramco specific rules) that govern on the ground.
	LevelSiteJurisdiction LegalHierarchyLevel = 3
)

// String renders the level for audit messages.
func (l LegalHierarchyLevel) String() string {
	switch l {
	case LevelInternational:
		return "level 1 (international SDO)"
	case LevelNational:
		return "level 2 (national / conformity body)"
	case LevelSiteJurisdiction:
		return "level 3 (site jurisdiction)"
	default:
		return "unknown"
	}
}

// PreemptionRule is a named legal-preemption rule in the hierarchy engine.
type PreemptionRule string

const (
	// RuleStricterPreempts: among otherwise-equal standards the stricter
	// safety rule preempts the less strict one (deterministic severity score).
	RuleStricterPreempts PreemptionRule = "STRICTER_SAFETY_RULE_PREEMPTS"
	// RuleStatutoryOverridesAdvisory: national statutory requirements
	// override advisory international standards.
	RuleStatutoryOverridesAdvisory PreemptionRule = "STATUTORY_OVERRIDES_ADVISORY"
	// RuleSiteOverridesNational: site jurisdiction rules override national
	// conformity-body rules.
	RuleSiteOverridesNational PreemptionRule = "SITE_JURISDICTION_OVERRIDES_NATIONAL"
	// RuleNationalOverridesInternational: national statutory requirements
	// override advisory international standards.
	RuleNationalOverridesInternational PreemptionRule = "NATIONAL_STATUTORY_OVERRIDES_INTERNATIONAL"
	// RuleContractSafetyFactor: unless the contract stipulates a higher
	// safety factor, the preempted advisory standard yields.
	RuleContractSafetyFactor PreemptionRule = "CONTRACT_HIGHER_SAFETY_FACTOR_CONTROLS"
)

// StdClass is the fixed classification of a standard's issuing body.
type StdClass struct {
	Body       string
	Level      LegalHierarchyLevel
	Statutory  bool
	Strictness int // 0..3 deterministic safety-severity score for same-tier ties
}

// classifyTable maps issuing bodies to their legal-hierarchy tier.
var classifyTable = map[string]StdClass{
	"ISO":    {Body: "ISO", Level: LevelInternational, Strictness: 1},
	"IEC":    {Body: "IEC", Level: LevelInternational, Strictness: 1},
	"ISOEN":  {Body: "ISOEN", Level: LevelInternational, Strictness: 1},
	"EU":     {Body: "EU", Level: LevelInternational, Statutory: true, Strictness: 2},
	"OSHA":   {Body: "OSHA", Level: LevelNational, Statutory: true, Strictness: 2},
	"SASO":   {Body: "SASO", Level: LevelNational, Statutory: true, Strictness: 2},
	"ADNOC":  {Body: "ADNOC", Level: LevelNational, Statutory: true, Strictness: 3},
	"ASME":   {Body: "ASME", Level: LevelNational, Strictness: 2},
	"API":    {Body: "API", Level: LevelNational, Strictness: 2},
	"ANSI":   {Body: "ANSI", Level: LevelNational, Strictness: 2},
	"BS":     {Body: "BS", Level: LevelNational, Strictness: 2},
	"DIN":    {Body: "DIN", Level: LevelNational, Strictness: 2},
	"ARAMCO": {Body: "ARAMCO", Level: LevelSiteJurisdiction, Statutory: true, Strictness: 3},
	"SITE":   {Body: "SITE", Level: LevelSiteJurisdiction, Statutory: true, Strictness: 3},
}

// ClassifyStandard extracts the issuing body from a standard identifier and
// returns its fixed legal-hierarchy classification.
func ClassifyStandard(standard string) (StdClass, error) {
	clean := strings.ToUpper(strings.TrimSpace(standard))
	if clean == "" {
		return StdClass{}, ErrUnknownStandardBody
	}
	// Body is the leading token up to whitespace, dash, or slash.
	body := clean
	for i := 0; i < len(clean); i++ {
		c := clean[i]
		if c == ' ' || c == '-' || c == '/' {
			body = clean[:i]
			break
		}
	}
	cls, ok := classifyTable[body]
	if !ok {
		return StdClass{}, fmt.Errorf("%w: %q", ErrUnknownStandardBody, standard)
	}
	return cls, nil
}

// PreemptionResult is the deterministic outcome of a standards collision.
type PreemptionResult struct {
	// EffectiveOrder lists the standards in precedence order, most
	// authoritative first — the authoritative governing regime.
	EffectiveOrder []string
	// Winner is the governing standard: EffectiveOrder[0].
	Winner string
	// AppliedRules names each hierarchy rule that decided the order.
	AppliedRules []PreemptionRule
	// JurisdictionISO2 is the ISO code the resolution was scoped to.
	JurisdictionISO2 string
}

// ResolvePreemption ranks a set of colliding standards for a jurisdiction
// under the INTEGIN legal hierarchy:
//
//   - Level 3 (site) preempts Level 2 (national) which preempts Level 1 (SDO/international).
//   - Within a level, a statutory requirement preempts an advisory standard.
//   - Otherwise the stricter safety rule preempts (deterministic severity score),
//     with a lexicographic body tiebreak so the outcome never depends on input order.
//
// A contract stipulating a higher safety factor may override an advisory
// international standard; that preemption rule is reported as a caveat.
func ResolvePreemption(standards []string, jurisdictionISO2 string) (PreemptionResult, error) {
	if len(standards) == 0 {
		return PreemptionResult{}, ErrEmptyStandardSet
	}
	if jurisdictionISO2 != "" && !iso2Pattern.MatchString(strings.ToUpper(jurisdictionISO2)) {
		return PreemptionResult{}, fmt.Errorf("%w: jurisdiction %q is not ISO 3166-1 alpha-2", ErrInvalidISO2Length, jurisdictionISO2)
	}

	type ranked struct {
		std       string
		body      string
		level     LegalHierarchyLevel
		statutory bool
		strict    int
	}
	uniq := make(map[string]struct{}, len(standards))
	items := make([]ranked, 0, len(standards))
	for _, raw := range standards {
		clean := strings.ToUpper(strings.TrimSpace(raw))
		if _, dup := uniq[clean]; dup {
			continue
		}
		uniq[clean] = struct{}{}
		cls, err := ClassifyStandard(clean)
		if err != nil {
			return PreemptionResult{}, err
		}
		items = append(items, ranked{
			std:       clean,
			body:      cls.Body,
			level:     cls.Level,
			statutory: cls.Statutory,
			strict:    cls.Strictness,
		})
	}
	if len(items) == 0 {
		return PreemptionResult{}, ErrEmptyStandardSet
	}

	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		switch {
		case a.level != b.level:
			return a.level > b.level
		case a.statutory != b.statutory:
			return a.statutory
		case a.strict != b.strict:
			return a.strict > b.strict
		default:
			if a.body != b.body {
				return a.body < b.body
			}
			return a.std < b.std
		}
	})

	order := make([]string, len(items))
	var rules []PreemptionRule
	for i, item := range items {
		order[i] = item.std
		if i == 0 {
			continue
		}
		winner, loser := items[i-1], item
		switch {
		case winner.level > loser.level:
			if winner.level == LevelSiteJurisdiction {
				rules = append(rules, RuleSiteOverridesNational)
			} else {
				rules = append(rules, RuleNationalOverridesInternational)
			}
		case winner.statutory && !loser.statutory:
			rules = append(rules, RuleStatutoryOverridesAdvisory)
		case winner.strict != loser.strict:
			rules = append(rules, RuleStricterPreempts)
		}
	}
	if len(items) > 1 {
		rules = append(rules, RuleContractSafetyFactor)
	}

	return PreemptionResult{
		EffectiveOrder:   order,
		Winner:           order[0],
		AppliedRules:     rules,
		JurisdictionISO2: strings.ToUpper(jurisdictionISO2),
	}, nil
}
