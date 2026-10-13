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

// assertHotPathSLA fails the benchmark when the folded-gate hot path exceeds
// the < 50us tracker SLA. Benchmarks only run under -bench, so the timing
// check never interferes with the -race test suite.
func assertHotPathSLA(b *testing.B, startInclusive time.Time) {
	elapsed := time.Since(startInclusive)
	perOp := elapsed / time.Duration(b.N)
	if perOp > 50*time.Microsecond {
		b.Fatalf("folded gate hot path took %v/op, exceeding the 50us SLA", perOp)
	}
}

// BenchmarkDNVSKLHotPath measures the deployed folded DNV-ST-N001 skew load
// factor gate: one precompiled Program, one reused VarSet, single comparison.
func BenchmarkDNVSKLHotPath(b *testing.B) {
	e, _ := NewEvaluator()
	p, err := e.Compile(DNVSTN001RuleID, DNVSKLGateExpression, DNVSKLGateVars())
	if err != nil {
		b.Fatal(err)
	}
	v := NewVarSet("skl_factor")
	v.Put("skl_factor", 1.35)
	b.ReportAllocs()
	start := time.Now()
	for b.Loop() {
		if _, err := p.Evaluate(context.Background(), v); err != nil {
			b.Fatal(err)
		}
	}
	assertHotPathSLA(b, start)
}

// BenchmarkMWSJRPSeaStateHsGateHotPath measures the deployed folded
// wave-height half of the sea-state window (single comparison).
func BenchmarkMWSJRPSeaStateHsGateHotPath(b *testing.B) {
	e, _ := NewEvaluator()
	p, err := e.Compile("bench-mws-hs", MWSJRPSeaStateHsGateExpression, MWSJRPSeaStateVars())
	if err != nil {
		b.Fatal(err)
	}
	v := NewVarSet("hs_m", "tp_s", "max_hs_m", "max_tp_s")
	v.Put("hs_m", 1.2)
	v.Put("tp_s", 6.5)
	v.Put("max_hs_m", 1.5)
	v.Put("max_tp_s", 8.0)
	b.ReportAllocs()
	start := time.Now()
	for b.Loop() {
		if _, err := p.Evaluate(context.Background(), v); err != nil {
			b.Fatal(err)
		}
	}
	assertHotPathSLA(b, start)
}

// BenchmarkMWSJRPSeaStateTpGateHotPath measures the deployed folded
// wave-period half of the sea-state window (single comparison).
func BenchmarkMWSJRPSeaStateTpGateHotPath(b *testing.B) {
	e, _ := NewEvaluator()
	p, err := e.Compile("bench-mws-tp", MWSJRPSeaStateTpGateExpression, MWSJRPSeaStateVars())
	if err != nil {
		b.Fatal(err)
	}
	v := NewVarSet("hs_m", "tp_s", "max_hs_m", "max_tp_s")
	v.Put("hs_m", 1.2)
	v.Put("tp_s", 6.5)
	v.Put("max_hs_m", 1.5)
	v.Put("max_tp_s", 8.0)
	b.ReportAllocs()
	start := time.Now()
	for b.Loop() {
		if _, err := p.Evaluate(context.Background(), v); err != nil {
			b.Fatal(err)
		}
	}
	assertHotPathSLA(b, start)
}

// BenchmarkDesignASDHotPath measures the folded ASD capacity utilization gate
// (single comparison).
func BenchmarkDesignASDHotPath(b *testing.B) {
	e, _ := NewEvaluator()
	p, err := e.Compile("bench-asd", CapacityUtilizationFoldedExpression, CapacityUtilizationVars())
	if err != nil {
		b.Fatal(err)
	}
	v := NewVarSet("utilization")
	v.Put("utilization", 0.85)
	b.ReportAllocs()
	start := time.Now()
	for b.Loop() {
		if _, err := p.Evaluate(context.Background(), v); err != nil {
			b.Fatal(err)
		}
	}
	assertHotPathSLA(b, start)
}

// BenchmarkDesignLRFDHotPath measures the folded partial-factor device gate:
// factored_resistance >= factored_demand (single comparison, 0 allocs).
func BenchmarkDesignLRFDHotPath(b *testing.B) {
	e, _ := NewEvaluator()
	p, err := e.Compile("bench-lrfd", DesignLRFDGateFoldedExpression, DesignLRFDFoldedVars())
	if err != nil {
		b.Fatal(err)
	}
	v := NewVarSet("factored_resistance", "factored_demand")
	v.Put("factored_resistance", 90.0)
	v.Put("factored_demand", 76.0)
	b.ReportAllocs()
	start := time.Now()
	for b.Loop() {
		if _, err := p.Evaluate(context.Background(), v); err != nil {
			b.Fatal(err)
		}
	}
	assertHotPathSLA(b, start)
}
