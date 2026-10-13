package equipment

import (
	"errors"
	"math"
	"testing"
)

// twoPointCurve builds a hand-curve with exactly known cycle limits so the
// Palmgren-Miner math in the tests is exact instead of interpolated.
func twoPointCurve() SNCurve {
	c, err := NewSNCurve("test_wire", "ISO 4309", []SNPoint{
		{LoadFactor: 0.5, CyclesToFailure: 2_000_000},
		{LoadFactor: 1.0, CyclesToFailure: 20_000},
	})
	if err != nil {
		panic(err)
	}
	return c
}

func TestNewSNCurveValidation(t *testing.T) {
	if _, err := NewSNCurve("m", "s", nil); !errors.Is(err, ErrEmptySNCurve) {
		t.Fatalf("err = %v, want %v", err, ErrEmptySNCurve)
	}
	if _, err := NewSNCurve("m", "s", []SNPoint{{LoadFactor: 1, CyclesToFailure: 10}}); !errors.Is(err, ErrEmptySNCurve) {
		t.Fatalf("err = %v, want %v", err, ErrEmptySNCurve)
	}
	if _, err := NewSNCurve("m", "s", []SNPoint{
		{LoadFactor: 0.5, CyclesToFailure: 10},
		{LoadFactor: 0.5, CyclesToFailure: 20}, // not strictly increasing
	}); !errors.Is(err, ErrInvalidSNCurve) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidSNCurve)
	}
	if _, err := NewSNCurve("m", "s", []SNPoint{
		{LoadFactor: 0, CyclesToFailure: 10},
		{LoadFactor: 1, CyclesToFailure: 20},
	}); !errors.Is(err, ErrInvalidSNCurve) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidSNCurve)
	}
	if _, err := NewSNCurve("m", "s", []SNPoint{
		{LoadFactor: 0.5, CyclesToFailure: 0},
		{LoadFactor: 1, CyclesToFailure: 20},
	}); !errors.Is(err, ErrInvalidSNCurve) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidSNCurve)
	}
}

func TestNewBasquinSNCurveValidation(t *testing.T) {
	if _, err := NewBasquinSNCurve(1.0, 20_000); !errors.Is(err, ErrInvalidSNCurveParams) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidSNCurveParams)
	}
	if _, err := NewBasquinSNCurve(3.0, 0); !errors.Is(err, ErrInvalidSNCurveParams) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidSNCurveParams)
	}
	if _, err := NewBasquinSNCurve(math.NaN(), 20_000); !errors.Is(err, ErrInvalidSNCurveParams) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidSNCurveParams)
	}
}

func TestCycleLimitBasquinPowerLaw(t *testing.T) {
	curve, err := NewBasquinSNCurve(3.0, 20_000)
	if err != nil {
		t.Fatalf("curve: %v", err)
	}
	if got, err := curve.CycleLimit(1.0); err != nil || got != 20_000 {
		t.Fatalf("CycleLimit(1.0) = %v, %v; want 20000", got, err)
	}
	// Basquin m=3: N(f) = 20000 * f^-3.
	for _, sf := range []float64{0.9, 0.7, 0.5, 0.3, 0.1, 0.02} {
		got, err := curve.CycleLimit(sf)
		if err != nil {
			t.Fatalf("CycleLimit(%v): %v", sf, err)
		}
		want := 20_000 * math.Pow(1.0/sf, 3.0)
		rel := math.Abs(got-want) / want
		if rel > 1e-9 {
			t.Fatalf("CycleLimit(%v) = %v, want %v (rel err %v)", sf, got, want, rel)
		}
	}
	if _, err := curve.CycleLimit(0); !errors.Is(err, ErrLoadFactorOutOfRange) {
		t.Fatalf("err = %v, want %v", err, ErrLoadFactorOutOfRange)
	}
	if _, err := curve.CycleLimit(1.5); !errors.Is(err, ErrLoadFactorOutOfRange) {
		t.Fatalf("err = %v, want %v", err, ErrLoadFactorOutOfRange)
	}
}

func TestCycleLimitMonotonicity(t *testing.T) {
	curve, err := NewBasquinSNCurve(3.0, 20_000)
	if err != nil {
		t.Fatalf("curve: %v", err)
	}
	previous := math.Inf(1)
	for f := 0.05; f < 1.0; f += 0.05 {
		n, err := curve.CycleLimit(f)
		if err != nil {
			t.Fatalf("CycleLimit(%v): %v", f, err)
		}
		if n > previous {
			t.Fatalf("cycle limit non-monotonic at %v: %v > previous %v", f, n, previous)
		}
		previous = n
	}
}

func TestPalmgrenMinerAccumulation(t *testing.T) {
	curve := twoPointCurve()
	fs, err := NewFatigueState(curve)
	if err != nil {
		t.Fatalf("NewFatigueState: %v", err)
	}
	if fs.Damage != 0 || fs.LockedOut {
		t.Fatalf("fresh state damage=%v locked=%v", fs.Damage, fs.LockedOut)
	}
	// Block 1: 1,000,000 cycles at 0.5 -> D += 1e6/2e6 = 0.5
	if err := fs.Accumulate(LoadCycleBlock{LoadFactor: 0.5, CycleCount: 1_000_000, Description: "hoist cycles"}); err != nil {
		t.Fatalf("accumulate 1: %v", err)
	}
	if math.Abs(fs.Damage-0.5) > 1e-12 {
		t.Fatalf("damage after block 1 = %v, want 0.5", fs.Damage)
	}
	if fs.LockedOut {
		t.Fatal("must not lock out at D = 0.5")
	}
	// Block 2: 10,000 cycles at 1.0 -> D += 10_000/20_000 = 0.5 -> total 1.0
	if err := fs.Accumulate(LoadCycleBlock{LoadFactor: 1.0, CycleCount: 10_000}); err != nil {
		t.Fatalf("accumulate 2: %v", err)
	}
	if !fs.LockedOut {
		t.Fatal("must lock out once D >= 1.0")
	}
	if len(fs.Blocks) != 2 {
		t.Fatalf("blocks = %d, want 2", len(fs.Blocks))
	}
}

func TestFatigueLockoutRejectsFurtherAccumulation(t *testing.T) {
	fs, err := NewFatigueState(twoPointCurve())
	if err != nil {
		t.Fatalf("NewFatigueState: %v", err)
	}
	if err := fs.Accumulate(LoadCycleBlock{LoadFactor: 0.5, CycleCount: 2_000_000}); err != nil {
		t.Fatalf("accumulate to D=1: %v", err)
	}
	err = fs.Accumulate(LoadCycleBlock{LoadFactor: 0.5, CycleCount: 100})
	if !errors.Is(err, ErrFatigueLockedOut) {
		t.Fatalf("err = %v, want %v", err, ErrFatigueLockedOut)
	}
	if fs.RemainingDamage() != 0 {
		t.Fatalf("remaining damage = %v, want 0", fs.RemainingDamage())
	}
	rul, err := fs.RUL(LoadCycleBlock{LoadFactor: 0.5})
	if err != nil {
		t.Fatalf("RUL: %v", err)
	}
	if rul != 0 {
		t.Fatalf("RUL on locked-out state = %v, want 0", rul)
	}
}

func TestRULCountsDown(t *testing.T) {
	fs, err := NewFatigueState(twoPointCurve())
	if err != nil {
		t.Fatalf("NewFatigueState: %v", err)
	}
	if err := fs.Accumulate(LoadCycleBlock{LoadFactor: 0.5, CycleCount: 500_000}); err != nil {
		t.Fatalf("accumulate: %v", err)
	}
	// D = 0.25, projected 0.5 -> N = 2e6 -> RUL = 0.75 * 2e6 = 1.5e6
	rul, err := fs.RUL(LoadCycleBlock{LoadFactor: 0.5})
	if err != nil {
		t.Fatalf("RUL: %v", err)
	}
	if math.Abs(rul-1_500_000) > 1e-6 {
		t.Fatalf("RUL = %v, want 1500000", rul)
	}
}

func TestSafeThresholdLockoutBeforeUnity(t *testing.T) {
	fs, err := NewFatigueState(twoPointCurve())
	if err != nil {
		t.Fatalf("NewFatigueState: %v", err)
	}
	if err := fs.SetLockoutThreshold(0.9); err != nil {
		t.Fatalf("SetLockoutThreshold: %v", err)
	}
	// D = 1,800,000 / 2,000,000 = 0.9 -> >= 0.9 threshold, locked despite D < 1.0
	if err := fs.Accumulate(LoadCycleBlock{LoadFactor: 0.5, CycleCount: 1_800_000}); err != nil {
		t.Fatalf("accumulate: %v", err)
	}
	if !fs.LockedOut {
		t.Fatal("must lock out at D = 0.9 with threshold 0.9")
	}
	if err := fs.Accumulate(LoadCycleBlock{LoadFactor: 0.5, CycleCount: 1}); !errors.Is(err, ErrFatigueLockedOut) {
		t.Fatalf("err = %v, want %v", err, ErrFatigueLockedOut)
	}
}

func TestSetLockoutThresholdValidation(t *testing.T) {
	fs, err := NewFatigueState(twoPointCurve())
	if err != nil {
		t.Fatalf("NewFatigueState: %v", err)
	}
	if err := fs.SetLockoutThreshold(0); !errors.Is(err, ErrInvalidFatigueThreshold) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidFatigueThreshold)
	}
}

func TestAccumulateLoadFactorGuards(t *testing.T) {
	fs, err := NewFatigueState(twoPointCurve())
	if err != nil {
		t.Fatalf("NewFatigueState: %v", err)
	}
	if err := fs.Accumulate(LoadCycleBlock{LoadFactor: 0, CycleCount: 10}); !errors.Is(err, ErrLoadFactorOutOfRange) {
		t.Fatalf("err = %v, want %v", err, ErrLoadFactorOutOfRange)
	}
	if err := fs.Accumulate(LoadCycleBlock{LoadFactor: 1.2, CycleCount: 10}); !errors.Is(err, ErrLoadFactorOutOfRange) {
		t.Fatalf("err = %v, want %v", err, ErrLoadFactorOutOfRange)
	}
	if err := fs.Accumulate(LoadCycleBlock{LoadFactor: 0.5, CycleCount: 0}); err != nil {
		t.Fatalf("zero-count block should no-op, got %v", err)
	}
	if fs.Damage != 0 {
		t.Fatalf("damage = %v, want 0 after no-op block", fs.Damage)
	}
}

func TestFatigueLockoutFlowsIntoComponentDiscard(t *testing.T) {
	comp, err := NewComponentItem("cp-rope", "ROPE-SER-77", ComponentWireRope)
	if err != nil {
		t.Fatalf("NewComponentItem: %v", err)
	}
	curve, err := NewBasquinSNCurve(3.0, 20_000)
	if err != nil {
		t.Fatalf("curve: %v", err)
	}
	fs, err := NewFatigueState(curve)
	if err != nil {
		t.Fatalf("NewFatigueState: %v", err)
	}
	comp.Fatigue = fs

	if comp.SyncFatigueLockout() {
		t.Fatal("should not lock out before fatigue threshold is crossed")
	}
	if comp.IsDiscarded() {
		t.Fatal("component must not be discarded before fatigue lockout")
	}

	// Drive D to 1.0: 10k + 5k + 5k cycles at rated load = 20000/20000.
	for _, n := range []uint64{10_000, 5_000, 5_000} {
		if err := fs.Accumulate(LoadCycleBlock{LoadFactor: 1.0, CycleCount: n}); err != nil {
			// err may be ErrFatigueLockedOut once threshold is crossed mid-drive: expected.
			if !errors.Is(err, ErrFatigueLockedOut) {
				t.Fatalf("accumulate(%d): %v", n, err)
			}
		}
	}
	if !comp.SyncFatigueLockout() {
		t.Fatal("fatigue lockout should mark the component pending discard")
	}
	if comp.DiscardStatus != DiscardPending {
		t.Fatalf("discard status = %q, want PENDING_DISCARD", comp.DiscardStatus)
	}
	if fs.Damage < 1.0 {
		t.Fatalf("damage = %v, want >= 1.0", fs.Damage)
	}
}
