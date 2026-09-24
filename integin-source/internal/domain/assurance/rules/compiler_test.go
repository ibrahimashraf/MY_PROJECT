package rules_test

import (
	"encoding/json"
	"testing"
	"time"

	"integin/internal/domain/assurance/rules"
)

func TestCompiler_CompileProgram(t *testing.T) {
	compiler, err := rules.NewCompiler()
	if err != nil {
		t.Fatalf("NewCompiler failed: %v", err)
	}

	// Valid expression statically checked against native structs
	expr := "finding.MeasuredValue / asset.NominalDiameter < 0.90"
	prg, err := compiler.CompileProgram(expr)
	if err != nil {
		t.Fatalf("Expected successful compile, got err: %v", err)
	}

	// Test evaluation with native structs
	out, _, err := prg.Eval(map[string]interface{}{
		"asset":   rules.AssetFacts{NominalDiameter: 16.0},
		"finding": rules.FindingFact{MeasuredValue: 14.0}, // 14/16 = 0.875 (<0.9)
	})
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}
	if out.Value() != true {
		t.Errorf("Expected true, got %v", out.Value())
	}

	// Invalid expression (Type Checking failure)
	// 'invalid_field' does not exist on AssetFacts
	_, err = compiler.CompileProgram("finding.MeasuredValue / asset.invalid_field < 0.90")
	if err == nil {
		t.Error("Expected static type checking error for missing field, got nil")
	}

	// Invalid expression (Syntax error)
	_, err = compiler.CompileProgram("finding.MeasuredValue +++")
	if err == nil {
		t.Error("Expected syntax error, got nil")
	}
}

func TestCompiler_BuildRuleBundle(t *testing.T) {
	compiler, err := rules.NewCompiler()
	if err != nil {
		t.Fatalf("NewCompiler failed: %v", err)
	}

	inputRules := []rules.DecisionRule{
		{
			RuleID:     "ISO4309_DIAMETER_REDUCTION",
			Expression: "finding.MeasuredValue / asset.NominalDiameter",
			Evaluations: []rules.ThresholdRule{
				{Operator: "LT", Threshold: 0.90, Severity: "CRITICAL", Reason: "ISO4309_DIAMETER_DISCARD", IsBlocking: true},
			},
		},
	}

	fixedTime := time.Unix(1790278800, 0)
	bundle, err := compiler.BuildRuleBundle("LIFTING_ISO4309", "1.0.0", inputRules, fixedTime)
	if err != nil {
		t.Fatalf("BuildRuleBundle failed: %v", err)
	}

	if bundle.BundleID != "LIFTING_ISO4309" {
		t.Errorf("Expected LIFTING_ISO4309, got %v", bundle.BundleID)
	}

	// Strict determinism test: The hash must exactly match this known good output
	// If the payload envelope changes (e.g. ID, Version, Rule array structure), this hash will break.
	expectedHash := "sha256:92da53f5b915d4fa234b79da81afee52a49dd854fa5ce678d339e07a8389d6d7"
	if bundle.RuleHash != expectedHash {
		t.Errorf("Hash determinism failed!\nExpected: %s\nGot:      %s", expectedHash, bundle.RuleHash)
	}

	// Ensure it produces valid JSON
	_, err = json.Marshal(bundle)
	if err != nil {
		t.Errorf("Failed to marshal bundle: %v", err)
	}
}
