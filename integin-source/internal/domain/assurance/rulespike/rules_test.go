package rulespike

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"cel.dev/cel-go/cel"
)

// --- Domain Models ---

type AssetFacts struct {
	AssetClass      string  `json:"asset_class"`
	NominalDiameter float64 `json:"nominal_diameter"`
}

type FindingFact struct {
	Code          string  `json:"code"`
	MeasuredValue float64 `json:"measured_value"`
}

// --- Rule Bundle Models (JSON Decision Table) ---

type RuleBundleJSON struct {
	BundleID     string         `json:"bundle_id"`
	Version      string         `json:"version"`
	RuleHash     string         `json:"rule_hash"`
	CreatedAtUTC int64          `json:"created_at_utc"`
	Rules        []DecisionRule `json:"rules"`
}

type DecisionRule struct {
	RuleID      string          `json:"rule_id"`
	TargetField string          `json:"target_field"`
	Expression  string          `json:"expression"`
	Evaluations []ThresholdRule `json:"evaluations"`
}

type ThresholdRule struct {
	Operator   string  `json:"operator"` // LT, GTE, etc.
	Threshold  float64 `json:"threshold"`
	Severity   string  `json:"severity"` // CRITICAL, MAJOR
	Reason     string  `json:"reason"`
	IsBlocking bool    `json:"is_blocking"`
}

func TestCELServerEvaluation(t *testing.T) {
	// 1. Setup CEL Environment
	env, err := cel.NewEnv(
		cel.Variable("asset", cel.MapType(cel.StringType, cel.AnyType)),
		cel.Variable("finding", cel.MapType(cel.StringType, cel.AnyType)),
	)
	if err != nil {
		t.Fatalf("cel.NewEnv: %v", err)
	}

	// 2. Compile Expression (ISO 4309 Diameter Reduction)
	expr := "finding.measured_diameter / asset.nominal_diameter < 0.90"
	ast, iss := env.Compile(expr)
	if iss.Err() != nil {
		t.Fatalf("Compile: %v", iss.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		t.Fatalf("Program: %v", err)
	}

	// 3. Evaluate Server-Side (Simulate CRITICAL discard)
	out, _, err := prg.Eval(map[string]interface{}{
		"asset": map[string]interface{}{
			"nominal_diameter": 16.0,
		},
		"finding": map[string]interface{}{
			"measured_diameter": 14.2, // 14.2 / 16.0 = 0.8875 (< 0.90)
		},
	})
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}

	if out.Value() != true {
		t.Errorf("Expected rule to evaluate to true, got %v", out.Value())
	}

	// 4. Evaluate Server-Side (Simulate PASS)
	outPass, _, _ := prg.Eval(map[string]interface{}{
		"asset": map[string]interface{}{
			"nominal_diameter": 16.0,
		},
		"finding": map[string]interface{}{
			"measured_diameter": 15.5, // 15.5 / 16.0 = 0.96875 (not < 0.90)
		},
	})
	if outPass.Value() != false {
		t.Errorf("Expected rule to evaluate to false, got %v", outPass.Value())
	}
}

func TestJSONRuleBundleSerialization(t *testing.T) {
	bundle := RuleBundleJSON{
		BundleID:     "ISO_4309_LIFTING_WIRE_ROPE",
		Version:      "1.0.0",
		RuleHash:     "sha256:dummy",
		CreatedAtUTC: time.Now().Unix(),
		Rules: []DecisionRule{
			{
				RuleID:      "DIAMETER_REDUCTION",
				Expression:  "finding.measured_diameter / asset.nominal_diameter",
				TargetField: "finding.measured_diameter",
				Evaluations: []ThresholdRule{
					{Operator: "LT", Threshold: 0.90, Severity: "CRITICAL", Reason: "ISO4309_DIAMETER_DISCARD", IsBlocking: true},
					{Operator: "LT", Threshold: 0.95, Severity: "MAJOR", Reason: "ISO4309_DIAMETER_WARNING", IsBlocking: false},
				},
			},
		},
	}

	bytes, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// Verify it can unmarshal safely
	var restored RuleBundleJSON
	if err := json.Unmarshal(bytes, &restored); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if !reflect.DeepEqual(bundle.BundleID, restored.BundleID) {
		t.Errorf("Mismatch in restored bundle ID")
	}
}
