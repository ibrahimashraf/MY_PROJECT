package engine

import (
	"errors"
	"math"

	"integin/pkg/cad/dxf"
)

// CraneKinematics defines the geometric and loading parameters for a single crane.
type CraneKinematics struct {
	BasePosition         dxf.Point3D `json:"base_position"`
	BoomLengthMeters     float64     `json:"boom_length_meters"`
	BoomAngleDeg         float64     `json:"boom_angle_deg"`
	SlewAngleDeg         float64     `json:"slew_angle_deg"`
	CounterweightTonne   float64     `json:"counterweight_tonne"`
	OutriggerSpreadXM    float64     `json:"outrigger_spread_x_m"` // e.g. 8.5m
	OutriggerSpreadZM    float64     `json:"outrigger_spread_z_m"` // e.g. 8.5m
	ChassisWeightTonne   float64     `json:"chassis_weight_tonne"` // e.g. 60t
	HookBlockWeightTonne float64     `json:"hook_block_weight_tonne,omitempty"`
}

// HookState contains the computed 3D hook location and working radius.
type HookState struct {
	HookTip       dxf.Point3D `json:"hook_tip"`
	WorkingRadius float64     `json:"working_radius_m"`
}

// ComputeHookPosition calculates the boom tip and working radius from angles.
func (c *CraneKinematics) ComputeHookPosition() HookState {
	boomRad := c.BoomAngleDeg * math.Pi / 180.0
	slewRad := c.SlewAngleDeg * math.Pi / 180.0

	radius := c.BoomLengthMeters * math.Cos(boomRad)
	height := c.BoomLengthMeters * math.Sin(boomRad)

	dx := radius * math.Sin(slewRad)
	dz := radius * math.Cos(slewRad)

	tip := dxf.Point3D{
		X: c.BasePosition.X + dx,
		Y: c.BasePosition.Y + height,
		Z: c.BasePosition.Z + dz,
	}

	return HookState{
		HookTip:       tip,
		WorkingRadius: radius,
	}
}

// OutriggerPressures stores vertical reaction forces (tonnes) on 4 outriggers.
type OutriggerPressures struct {
	FrontLeft  float64 `json:"front_left_tonne"`
	FrontRight float64 `json:"front_right_tonne"`
	RearLeft   float64 `json:"rear_left_tonne"`
	RearRight  float64 `json:"rear_right_tonne"`
	MaxLoad    float64 `json:"max_load_tonne"`
}

// ComputeOutriggerPressures calculates ground reaction under 4 outrigger pads.
func (c *CraneKinematics) ComputeOutriggerPressures(hookLoadTonne float64) OutriggerPressures {
	hs := c.ComputeHookPosition()

	// Total vertical load
	totalLoad := c.ChassisWeightTonne + c.CounterweightTonne + hookLoadTonne
	baseP := totalLoad / 4.0

	// Moment calculations relative to crane rotation center
	slewRad := c.SlewAngleDeg * math.Pi / 180.0
	loadDx := hs.WorkingRadius * math.Sin(slewRad)
	loadDz := hs.WorkingRadius * math.Cos(slewRad)

	// Counterweight opposes the load (180 degrees away)
	cwRadius := 4.5 // meters typical counterweight tail swing
	cwDx := -cwRadius * math.Sin(slewRad)
	cwDz := -cwRadius * math.Cos(slewRad)

	// Net moments
	mx := (hookLoadTonne * loadDz) + (c.CounterweightTonne * cwDz) // Pitch moment
	mz := (hookLoadTonne * loadDx) + (c.CounterweightTonne * cwDx) // Roll moment

	// Guard against floating point division by zero (ADV-04 / Hazard 31)
	if c.OutriggerSpreadXM <= 0.1 || c.OutriggerSpreadZM <= 0.1 {
		return OutriggerPressures{
			FrontLeft:  baseP,
			FrontRight: baseP,
			RearLeft:   baseP,
			RearRight:  baseP,
			MaxLoad:    baseP,
		}
	}

	// Delta loads from moments
	deltaZ := mx / (2.0 * c.OutriggerSpreadZM)
	deltaX := mz / (2.0 * c.OutriggerSpreadXM)

	fl := math.Max(0, baseP+deltaZ-deltaX)
	fr := math.Max(0, baseP+deltaZ+deltaX)
	rl := math.Max(0, baseP-deltaZ-deltaX)
	rr := math.Max(0, baseP-deltaZ+deltaX)

	maxL := math.Max(math.Max(fl, fr), math.Max(rl, rr))

	return OutriggerPressures{
		FrontLeft:  fl,
		FrontRight: fr,
		RearLeft:   rl,
		RearRight:  rr,
		MaxLoad:    maxL,
	}
}

// ComputeRelaxedOutriggerPressures implements iterative contact relaxation (lift-off detection):
// If calculated reaction P_i <= 0, that pad loses ground contact. The solver drops that constraint
// and re-equilibrates forces diagonally across active contact points, detecting impending tipping.
func (c *CraneKinematics) ComputeRelaxedOutriggerPressures(hookLoadTonne float64, additionalMomentTonneM float64) (OutriggerPressures, bool) {
	hs := c.ComputeHookPosition()
	totalLoad := c.ChassisWeightTonne + c.CounterweightTonne + hookLoadTonne

	if c.OutriggerSpreadXM <= 0.1 || c.OutriggerSpreadZM <= 0.1 {
		p := totalLoad / 4.0
		return OutriggerPressures{FrontLeft: p, FrontRight: p, RearLeft: p, RearRight: p, MaxLoad: p}, false
	}

	slewRad := c.SlewAngleDeg * math.Pi / 180.0
	loadDx := hs.WorkingRadius * math.Sin(slewRad)
	loadDz := hs.WorkingRadius * math.Cos(slewRad)

	cwRadius := 4.5
	cwDx := -cwRadius * math.Sin(slewRad)
	cwDz := -cwRadius * math.Cos(slewRad)

	mx := (hookLoadTonne * loadDz) + (c.CounterweightTonne * cwDz)
	mz := (hookLoadTonne * loadDx) + (c.CounterweightTonne * cwDx) + additionalMomentTonneM

	baseP := totalLoad / 4.0
	deltaZ := mx / (2.0 * c.OutriggerSpreadZM)
	deltaX := mz / (2.0 * c.OutriggerSpreadXM)

	rawFL := baseP + deltaZ - deltaX
	rawFR := baseP + deltaZ + deltaX
	rawRL := baseP - deltaZ - deltaX
	rawRR := baseP - deltaZ + deltaX

	// Check for lift-off
	if rawFL > 0 && rawFR > 0 && rawRL > 0 && rawRR > 0 {
		maxL := math.Max(math.Max(rawFL, rawFR), math.Max(rawRL, rawRR))
		return OutriggerPressures{
			FrontLeft:  rawFL,
			FrontRight: rawFR,
			RearLeft:   rawRL,
			RearRight:  rawRR,
			MaxLoad:    maxL,
		}, false
	}

	// Lift-off detected: re-equilibrate across active triangle
	w := totalLoad
	spreadX := c.OutriggerSpreadXM
	spreadZ := c.OutriggerSpreadZM

	var fl, fr, rl, rr float64

	switch {
	case rawRL <= 0:
		// RL lifted off -> active points FL, FR, RR
		rr = math.Max(0, 0.5*w - mx/spreadZ)
		sumFront := w - rr
		diffFront := (2.0*mz/spreadX) - rr
		fr = math.Max(0, 0.5*(sumFront+diffFront))
		fl = math.Max(0, sumFront-fr)
		rl = 0.0

	case rawRR <= 0:
		// RR lifted off -> active points FL, FR, RL
		rl = math.Max(0, 0.5*w - mx/spreadZ)
		sumFront := w - rl
		diffFront := (2.0*mz/spreadX) + rl
		fr = math.Max(0, 0.5*(sumFront+diffFront))
		fl = math.Max(0, sumFront-fr)
		rr = 0.0

	case rawFL <= 0:
		// FL lifted off -> active points FR, RL, RR
		fr = math.Max(0, 0.5*w + mx/spreadZ)
		sumRear := w - fr
		diffRear := (2.0*mz/spreadX) - fr
		rr = math.Max(0, 0.5*(sumRear+diffRear))
		rl = math.Max(0, sumRear-rr)
		fl = 0.0

	default: // rawFR <= 0
		// FR lifted off -> active points FL, RL, RR
		fl = math.Max(0, 0.5*w + mx/spreadZ)
		sumRear := w - fl
		diffRear := (2.0*mz/spreadX) + fl
		rr = math.Max(0, 0.5*(sumRear+diffRear))
		rl = math.Max(0, sumRear-rr)
		fr = 0.0
	}

	// Re-normalize sum to strictly preserve total vertical load equilibrium
	activeSum := fl + fr + rl + rr
	if activeSum > 0 {
		scale := w / activeSum
		fl *= scale
		fr *= scale
		rl *= scale
		rr *= scale
	}

	maxL := math.Max(math.Max(fl, fr), math.Max(rl, rr))
	return OutriggerPressures{
		FrontLeft:  fl,
		FrontRight: fr,
		RearLeft:   rl,
		RearRight:  rr,
		MaxLoad:    maxL,
	}, true
}

// TandemLiftResult contains load distribution and spatial clearances.
type TandemLiftResult struct {
	Hook1State        HookState          `json:"crane1_hook"`
	Hook2State        HookState          `json:"crane2_hook"`
	LoadShareCrane1   float64            `json:"load_share_crane1_pct"`
	LoadShareCrane2   float64            `json:"load_share_crane2_pct"`
	Crane1LoadTonnes  float64            `json:"crane1_load_tonnes"`
	Crane2LoadTonnes  float64            `json:"crane2_load_tonnes"`
	Crane1Outriggers  OutriggerPressures `json:"crane1_outriggers"`
	Crane2Outriggers  OutriggerPressures `json:"crane2_outriggers"`
	HookSpanMeters    float64            `json:"hook_span_meters"`
	MinBoomClearanceM float64            `json:"min_boom_clearance_meters"`
}

// SolveTandemLift computes the static equilibrium of a tandem lift.
func SolveTandemLift(
	crane1, crane2 CraneKinematics,
	totalLoadTonnes float64,
	cogOffsetFromCrane1M float64,
) (*TandemLiftResult, error) {
	if totalLoadTonnes <= 0 {
		return nil, errors.New("total load must be positive")
	}

	h1 := crane1.ComputeHookPosition()
	h2 := crane2.ComputeHookPosition()

	// Hook-to-hook distance (Rigging span)
	span := h1.HookTip.DistanceTo(h2.HookTip)

	if span < 1.0 {
		return nil, errors.New("crane hooks are too close together (<1.0m)")
	}

	if cogOffsetFromCrane1M < 0 || cogOffsetFromCrane1M > span {
		cogOffsetFromCrane1M = span * 0.5 // Default to geometric center
	}

	// Lever rule for load distribution
	share2 := cogOffsetFromCrane1M / span
	share1 := 1.0 - share2

	load1 := totalLoadTonnes * share1
	load2 := totalLoadTonnes * share2

	p1 := crane1.ComputeOutriggerPressures(load1)
	p2 := crane2.ComputeOutriggerPressures(load2)

	// Spatial clearance between boom tips
	boomClearance := span

	return &TandemLiftResult{
		Hook1State:        h1,
		Hook2State:        h2,
		LoadShareCrane1:   share1 * 100.0,
		LoadShareCrane2:   share2 * 100.0,
		Crane1LoadTonnes:  load1,
		Crane2LoadTonnes:  load2,
		Crane1Outriggers:  p1,
		Crane2Outriggers:  p2,
		HookSpanMeters:    span,
		MinBoomClearanceM: boomClearance,
	}, nil
}
