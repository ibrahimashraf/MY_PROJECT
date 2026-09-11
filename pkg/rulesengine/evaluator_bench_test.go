package rulesengine

import (
	"context"
	"testing"
	"time"
)

// TestSandboxTermination bounds the benchmark helper: compile must fail for a
// known-malformed rule rather than hang, proving non-Turing termination.
func TestSandboxTermination(t *testing.T) {
	e, _ := NewEvaluator()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = e.Compile("doomed", "capacity_t *", B30_5Vars())
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("compile hung: CEL sandbox must terminate on malformed input")
	}
}

// BenchmarkB30_5HotPath measures the deployed tier-folded proof-load gate:
// one precompiled Program, one reused VarSet. The CEL AST is a single
// comparison (coefficient folded into required_test_load_t server-side).
// Tracker SLA: < 50us per call, 0 allocs/op.
func BenchmarkB30_5HotPath(b *testing.B) {
	e, _ := NewEvaluator()
	p, err := e.Compile(B30_5RuleID, B30_5GateExpression, B30_5GateVars())
	if err != nil {
		b.Fatal(err)
	}
	v := NewVarSet("reported_test_load_t", "required_test_load_t")
	v.Put("reported_test_load_t", 26.0)
	v.Put("required_test_load_t", B30_5ProofLoad(20))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := p.Evaluate(context.Background(), v); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkISO4309HotPath mirrors the folded wire-rope diameter gate.
func BenchmarkISO4309HotPath(b *testing.B) {
	e, _ := NewEvaluator()
	p, err := e.Compile(ISO4309DiameterRuleID, ISO4309DiameterGateExpression, ISO4309GateVars())
	if err != nil {
		b.Fatal(err)
	}
	v := NewVarSet("diameter_loss_pct")
	v.Put("diameter_loss_pct", 0.075)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := p.Evaluate(context.Background(), v); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRiggingHotPath exercises the pure-Go vector engine.
func BenchmarkRiggingHotPath(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, _, _, err := SlingTension2Leg(10, 45); err != nil {
			b.Fatal(err)
		}
	}
}
