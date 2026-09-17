package rulesengine

import "time"

// Severity classifies how a failing rule must be treated. Failing a
// CRITICAL_QUARANTINE rule locks the asset across operational branches.
type Severity string

const (
	SeverityCriticalQuarantine Severity = "CRITICAL_QUARANTINE"
	SeverityWarning            Severity = "WARNING"
)

const (
	LifecycleActive     = "ACTIVE"
	LifecycleSuperseded = "SUPERSEDED"
	LifecycleWithdrawn  = "WITHDRAWN"
	LifecycleDraft      = "DRAFT"
)

// DynamicStandardDefinition carries the immutable metadata of a governed
// engineering standard (ASME, ISO, ...). It never contains copyrighted full
// text; it is a citation card plus links to the official publisher.
type DynamicStandardDefinition struct {
	StandardDID      string    `json:"standard_did"`  // "did:integin:standard:asme-b30.5-2024"
	StandardBody     string    `json:"standard_body"` // "ASME"
	Code             string    `json:"code"`          // "B30.5"
	RevisionYear     int       `json:"revision_year"` // 2024
	Title            string    `json:"title"`
	ScopeAbstract    string    `json:"scope_abstract"`
	LifecycleState   string    `json:"lifecycle_state"` // ACTIVE / SUPERSEDED / WITHDRAWN / DRAFT
	ReplacesStandard string    `json:"replaces_standard,omitempty"`
	OfficialStoreURL string    `json:"official_store_url"`
	PublishedDate    time.Time `json:"published_date"`
}

// DynamicRule is a single declarative safety rule expressed as a Google CEL
// boolean expression evaluated against typed inspection variables.
type DynamicRule struct {
	RuleID      string            `json:"rule_id"` // "asme-b30.5.2.2.3.1a"
	Description string            `json:"description"`
	Expression  string            `json:"expression"` // Google CEL boolean expression
	Severity    Severity          `json:"severity"`
	VariableDoc map[string]string `json:"variable_doc,omitempty"`
}

// ChecklistSchema models the dynamic form bindings that a field tablet renders
// for a specific standard revision. Unknown keys render as text inputs; known
// keys map to typed widgets.
type ChecklistSchema map[string]interface{}

// ChecklistField documents one typed form field in a ChecklistSchema.
type ChecklistField struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Kind    string   `json:"kind"` // STRING / NUMBER / BOOLEAN / DATE / ENUM
	Unit    string   `json:"unit,omitempty"`
	Options []string `json:"options,omitempty"`
}

// RuleResult is the deterministic outcome of evaluating one DynamicRule.
type RuleResult struct {
	RuleID      string   `json:"rule_id"`
	Passed      bool     `json:"passed"`
	Severity    Severity `json:"severity"`
	Description string   `json:"description"`
	Message     string   `json:"message"`
}

// RuleSet is a versioned, activation-immutable collection of rules.
type RuleSet struct {
	ID          string        `json:"id"`
	Revision    int           `json:"revision"`
	State       RuleSetState  `json:"state"`
	StandardDID string        `json:"standard_did"`
	ActivatedAt time.Time     `json:"activated_at,omitempty"`
	Rules       []DynamicRule `json:"rules,omitempty"`
}
