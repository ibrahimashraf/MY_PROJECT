package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/ext"
)

// --- Domain Models ---

// AssetFacts represents the read-only facts about an asset for evaluation.
type AssetFacts struct {
	AssetClass      string  `json:"asset_class"`
	NominalDiameter float64 `json:"nominal_diameter"`
	RopeConstruction string `json:"rope_construction"`
}

// FindingFact represents a single inspection finding.
type FindingFact struct {
	Code          string  `json:"code"`
	MeasuredValue float64 `json:"measured_value"`
}

// Compiler validates CEL expressions against the domain facts schema
// and generates the portable JSON decision tables for edge execution.
type Compiler struct {
	env *cel.Env
}

// NewCompiler initializes a CEL compiler with strict static typing for the assurance domain.
func NewCompiler() (*Compiler, error) {
	env, err := cel.NewEnv(
		// Register strict Go struct types
		ext.NativeTypes(
			reflect.TypeOf(AssetFacts{}),
			reflect.TypeOf(FindingFact{}),
		),
		// Expose strongly-typed variables
		cel.Variable("asset", cel.ObjectType("rules.AssetFacts")),
		cel.Variable("finding", cel.ObjectType("rules.FindingFact")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL env: %w", err)
	}

	return &Compiler{env: env}, nil
}

// CompileProgram verifies a CEL rule string against the strict schema.
func (c *Compiler) CompileProgram(expr string) (cel.Program, error) {
	ast, iss := c.env.Compile(expr)
	if iss.Err() != nil {
		return nil, fmt.Errorf("CEL compile error: %w", iss.Err())
	}

	prg, err := c.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("CEL program error: %w", err)
	}
	return prg, nil
}

// DecisionRule defines a single rule with multiple threshold evaluations.
type DecisionRule struct {
	RuleID      string          `json:"rule_id"`
	TargetField string          `json:"target_field,omitempty"`
	Expression  string          `json:"expression"`
	Evaluations []ThresholdRule `json:"evaluations"`
}

type ThresholdRule struct {
	Operator   string  `json:"operator"` // LT, LTE, GT, GTE, EQ, NEQ
	Threshold  float64 `json:"threshold"`
	Severity   string  `json:"severity"` // CRITICAL, MAJOR, MINOR, ADVISORY
	Reason     string  `json:"reason"`
	IsBlocking bool    `json:"is_blocking"`
}

// RuleBundleJSON represents the deterministic offline decision table payload.
type RuleBundleJSON struct {
	BundleID     string         `json:"bundle_id"`
	Version      string         `json:"version"`
	RuleHash     string         `json:"rule_hash"`
	CreatedAtUTC int64          `json:"created_at_utc"`
	Rules        []DecisionRule `json:"rules"`
}

// BuildRuleBundle compiles a bundle definition into a JSON snapshot, generating the strict SHA-256 hash.
func (c *Compiler) BuildRuleBundle(bundleID, version string, inputRules []DecisionRule, createdAt time.Time) (*RuleBundleJSON, error) {
	// 1. Validate all CEL expressions in the bundle against the static schema
	for _, rule := range inputRules {
		if rule.Expression != "" {
			if _, err := c.CompileProgram(rule.Expression); err != nil {
				return nil, fmt.Errorf("invalid expression in rule %s: %w", rule.RuleID, err)
			}
		}
	}

	// 2. Construct intermediate bundle
	bundle := &RuleBundleJSON{
		BundleID:     bundleID,
		Version:      version,
		CreatedAtUTC: createdAt.UTC().Unix(),
		Rules:        inputRules,
		RuleHash:     "", // Excluded from its own hash
	}

	// 3. Compute deterministic hash over the ENTIRE envelope (including ID and Version)
	hashBytes, err := json.Marshal(bundle)
	if err != nil {
		return nil, fmt.Errorf("hash serialization failed: %w", err)
	}
	hasher := sha256.New()
	hasher.Write(hashBytes)
	bundle.RuleHash = "sha256:" + hex.EncodeToString(hasher.Sum(nil))

	return bundle, nil
}
