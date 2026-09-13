package equipment

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

var (
	ErrEmptySNCurve            = errors.New("S-N curve must contain at least 2 points")
	ErrInvalidSNCurve          = errors.New("S-N curve points must be strictly increasing load factors with positive cycles-to-failure")
	ErrInvalidSNCurveParams    = errors.New("S-N curve parameters are invalid")
	ErrLoadFactorOutOfRange    = errors.New("load factor must be within (0, 1]")
	ErrInvalidLoadCycle        = errors.New("load cycle block is invalid")
	ErrFatigueLockedOut        = errors.New("component is locked out: cumulative fatigue damage reached the safe threshold")
	ErrInvalidFatigueThreshold = errors.New("fatigue lockout threshold must be positive")
)

// SNPoint is one sample on an S-N (Wöhler) curve: the cycles-to-failure at a
// given load factor (applied cyclic load / rated load). Load factors are
// strictly increasing.
type SNPoint struct {
	LoadFactor      float64 `json:"load_factor"`
	CyclesToFailure float64 `json:"cycles_to_failure"`
}

// SNCurve is a sampled Wöhler curve for a wire rope or structural cyclic
// member, e.g. derived from DIN 15020 load spectrum classes or ISO 4309
// cyclic life data.
type SNCurve struct {
	Material string    `json:"material"`
	Standard string    `json:"standard"`
	Points   []SNPoint `json:"points"`
}

// NewSNCurve validates and stores a sampled S-N curve.
func NewSNCurve(material, standard string, points []SNPoint) (SNCurve, error) {
	if len(points) < 2 {
		return SNCurve{}, ErrEmptySNCurve
	}
	for i, p := range points {
		if p.LoadFactor <= 0 || p.LoadFactor > 1 {
			return SNCurve{}, fmt.Errorf("%w: point %d load factor %v", ErrInvalidSNCurve, i, p.LoadFactor)
		}
		if i > 0 && p.LoadFactor <= points[i-1].LoadFactor {
			return SNCurve{}, fmt.Errorf("%w: load factors must be strictly increasing", ErrInvalidSNCurve)
		}
		if p.CyclesToFailure <= 0 || math.IsNaN(p.CyclesToFailure) || math.IsInf(p.CyclesToFailure, 0) {
			return SNCurve{}, fmt.Errorf("%w: point %d cycles-to-failure must be a positive finite number", ErrInvalidSNCurve, i)
		}
	}
	return SNCurve{Material: material, Standard: standard, Points: append([]SNPoint(nil), points...)}, nil
}

const basquinSamples = 25

// NewBasquinSNCurve builds a wire-rope / structural S-N curve from the Basquin
// power law N = N_ref * (S_ref/S)^exponent, sampled across load factors from
// 0.01 to 1.0. Typical wire-rope exponents sit near m = 3; structural steel
// S-N slopes near m = 4 (per DIN 15020 / ISO 4309 practice).
func NewBasquinSNCurve(exponent, cyclesAtRatedLoad float64) (SNCurve, error) {
	if exponent <= 1 || math.IsNaN(exponent) || math.IsInf(exponent, 0) {
		return SNCurve{}, fmt.Errorf("%w: exponent must be > 1", ErrInvalidSNCurveParams)
	}
	if cyclesAtRatedLoad <= 0 || math.IsNaN(cyclesAtRatedLoad) || math.IsInf(cyclesAtRatedLoad, 0) {
		return SNCurve{}, fmt.Errorf("%w: cycles at rated load must be a positive finite number", ErrInvalidSNCurveParams)
	}
	const minFactor, maxFactor = 0.01, 1.0
	points := make([]SNPoint, basquinSamples)
	for k := 0; k < basquinSamples; k++ {
		f := math.Exp(math.Log(minFactor) + (math.Log(maxFactor)-math.Log(minFactor))*float64(k)/float64(basquinSamples-1))
		points[k] = SNPoint{
			LoadFactor:      f,
			CyclesToFailure: cyclesAtRatedLoad * math.Pow(1.0/f, exponent),
		}
	}
	return NewSNCurve("wire_rope_generic", "DIN 15020 / ISO 4309", points)
}

// CycleLimit returns the cycles-to-failure N(S) at the given load factor via
// log-log linear interpolation on the sampled S-N curve.
func (c SNCurve) CycleLimit(loadFactor float64) (float64, error) {
	if len(c.Points) < 2 {
		return 0, ErrEmptySNCurve
	}
	if loadFactor <= 0 || loadFactor > 1 {
		return 0, ErrLoadFactorOutOfRange
	}
	pts := c.Points
	if loadFactor <= pts[0].LoadFactor {
		return pts[0].CyclesToFailure, nil
	}
	if loadFactor >= pts[len(pts)-1].LoadFactor {
		return pts[len(pts)-1].CyclesToFailure, nil
	}
	hi := sort.Search(len(pts), func(i int) bool { return pts[i].LoadFactor >= loadFactor })
	lo := pts[hi-1]
	upper := pts[hi]
	logN := math.Log(lo.CyclesToFailure) +
		(math.Log(upper.CyclesToFailure)-math.Log(lo.CyclesToFailure))*
			(math.Log(loadFactor)-math.Log(lo.LoadFactor))/(math.Log(upper.LoadFactor)-math.Log(lo.LoadFactor))
	return math.Exp(logN), nil
}

// LoadCycleBlock is one operating block: cycleCount cycles applied at
// loadFactor (fraction of rated load).
type LoadCycleBlock struct {
	LoadFactor  float64 `json:"load_factor"`
	CycleCount  uint64  `json:"cycle_count"`
	Description string  `json:"description,omitempty"`
}

// FatigueState is the ISO 13374 Block 5 prognostics accumulator implementing
// the Palmgren-Miner rule D = sum(n_i / N_i). It locks out the component once
// the accumulated damage reaches the safe threshold (default D = 1.0).
type FatigueState struct {
	Curve     SNCurve          `json:"curve"`
	Damage    float64          `json:"damage"`
	Threshold float64          `json:"threshold"`
	LockedOut bool             `json:"locked_out"`
	Blocks    []LoadCycleBlock `json:"blocks,omitempty"`
}

// NewFatigueState initializes a fatigue accumulator over the given S-N curve
// with the default safe lockout threshold D = 1.0.
func NewFatigueState(curve SNCurve) (*FatigueState, error) {
	if len(curve.Points) < 2 {
		return nil, ErrEmptySNCurve
	}
	return &FatigueState{Curve: curve, Threshold: 1.0}, nil
}

// SetLockoutThreshold overrides the safe lockout threshold (e.g. 0.9 for an
// extra-conservative safe limit under ISO 13374).
func (f *FatigueState) SetLockoutThreshold(threshold float64) error {
	if threshold <= 0 || math.IsNaN(threshold) || math.IsInf(threshold, 0) {
		return ErrInvalidFatigueThreshold
	}
	f.Threshold = threshold
	return nil
}

// Accumulate applies one load block to the Palmgren-Miner sum. Once the safe
// threshold has been crossed the accumulator is locked out and further
// accumulation is rejected.
func (f *FatigueState) Accumulate(block LoadCycleBlock) error {
	if f.LockedOut {
		return ErrFatigueLockedOut
	}
	if block.CycleCount == 0 {
		return nil
	}
	if block.LoadFactor <= 0 || block.LoadFactor > 1 {
		return ErrLoadFactorOutOfRange
	}
	limit, err := f.Curve.CycleLimit(block.LoadFactor)
	if err != nil {
		return err
	}
	if limit <= 0 || math.IsNaN(limit) || math.IsInf(limit, 0) {
		return ErrInvalidLoadCycle
	}
	previous := f.Damage
	f.Damage += float64(block.CycleCount) / limit
	if math.IsNaN(f.Damage) || math.IsInf(f.Damage, 0) {
		f.Damage = previous
		return ErrInvalidLoadCycle
	}
	f.Blocks = append(f.Blocks, block)
	if f.Damage >= f.Threshold {
		f.LockedOut = true
	}
	return nil
}

// RemainingDamage returns the damage headroom left before the safe lockout
// threshold (0 when locked out).
func (f *FatigueState) RemainingDamage() float64 {
	remaining := f.Threshold - f.Damage
	if remaining < 0 {
		return 0
	}
	return remaining
}

// RUL estimates the remaining useful life in cycles at the projected load
// block before the safe threshold is reached. It returns 0 when the component
// is already locked out.
func (f *FatigueState) RUL(projected LoadCycleBlock) (float64, error) {
	if f.LockedOut {
		return 0, nil
	}
	if projected.LoadFactor <= 0 || projected.LoadFactor > 1 {
		return 0, ErrLoadFactorOutOfRange
	}
	limit, err := f.Curve.CycleLimit(projected.LoadFactor)
	if err != nil {
		return 0, err
	}
	remaining := f.RemainingDamage()
	if remaining <= 0 {
		return 0, nil
	}
	return remaining * limit, nil
}
