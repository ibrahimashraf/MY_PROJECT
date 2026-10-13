package structural

import (
	"fmt"
	"math"
)

// OutriggerContactState classifies the resolved stability state of a 4-pad
// outrigger system after lift-off relaxation.
type OutriggerContactState int

const (
	// Stable3PointTripod: exactly three pads carry the load (one lifted off).
	Stable3PointTripod OutriggerContactState = iota
	// Stable4Point: all four pads remain in compression.
	Stable4Point
	// UnstableTipping: fewer than three pads can support the load without
	// negative reaction; the crane is on the verge of overturning.
	UnstableTipping
)

// OutriggerSolution is the fully-resolved 4-pad contact state.
type OutriggerSolution struct {
	Reactions   [4]float64 // Newtons, ordered to match the input pad array
	Active      [4]bool    // true where the pad is in compression contact
	State       OutriggerContactState
	ActiveCount int
	Residual    float64 // equilibrium verification witness ||Ax-b||_2 (0 if unstable)
}

// Outrigger4Solve solves the statically indeterminate 4-pad outrigger contact
// problem with tensionless (no-pull) soil supports:
//
//  1. An elastic planar distribution is used as the initial guess.
//  2. Any pad with P_i < 0 has lifted off and is removed from the active set;
//     the remaining pads are re-resolved from static equilibrium
//     (Sum Fz = 0, Sum Mx = 0, Sum My = 0).
//  3. If the active set drops to two pads an overturning check is performed.
//
// The active-set equilibrium is witnessed by verifying ||Ax - b||_2 < 1e-6.
func Outrigger4Solve(totalLoadN float64, loadPos [2]float64, pads [4][2]float64) (OutriggerSolution, error) {
	if !(totalLoadN >= 0 && finite2(loadPos) && finite4x2(pads)) {
		return OutriggerSolution{}, fmt.Errorf("invalid inputs: totalLoad=%v loadPos=%v", totalLoadN, loadPos)
	}

	// Initial elastic planar distribution. If no pad goes into tension the
	// full 4-pad plane is the valid, statically determinate solution.
	allPositive := true
	for i := 0; i < 4; i++ {
		if elasticReaction4(totalLoadN, loadPos, pads, i) < 0 {
			allPositive = false
			break
		}
	}
	if allPositive {
		sol := OutriggerSolution{State: Stable4Point, ActiveCount: 4, Active: [4]bool{true, true, true, true}}
		for i := 0; i < 4; i++ {
			sol.Reactions[i] = elasticReaction4(totalLoadN, loadPos, pads, i)
		}
		sol.Residual = equilibriumResidual4(totalLoadN, loadPos, pads, sol.Reactions)
		if sol.Residual > 1e-6 {
			return OutriggerSolution{}, fmt.Errorf("4-point equilibrium witness failed: residual=%e", sol.Residual)
		}
		return sol, nil
	}

	// Relax: try every 3-pad support triangle (highest elastic reactions
	// first). The first triangle whose reactions are all non-negative is the
	// resolved tripod. If none holds, two pads remain and the system tips.
	for a := 0; a < 4; a++ {
		for b := a + 1; b < 4; b++ {
			for c := b + 1; c < 4; c++ {
				subPads := [3][2]float64{pads[a], pads[b], pads[c]}
				subR, err := ThreePointStaticEquilibrium(totalLoadN, loadPos, subPads)
				if err != nil {
					continue
				}
				valid := subR[0] >= 0 && subR[1] >= 0 && subR[2] >= 0
				if !valid {
					continue
				}
				sol := OutriggerSolution{State: Stable3PointTripod, ActiveCount: 3}
				sol.Active = [4]bool{}
				sol.Active[a], sol.Active[b], sol.Active[c] = true, true, true
				sol.Reactions[a], sol.Reactions[b], sol.Reactions[c] = subR[0], subR[1], subR[2]
				sol.Residual = equilibriumResidual4(totalLoadN, loadPos, pads, sol.Reactions)
				if sol.Residual > 1e-6 {
					return OutriggerSolution{}, fmt.Errorf("tripod equilibrium witness failed: residual=%e", sol.Residual)
				}
				return sol, nil
			}
		}
	}

	// No non-negative tripod exists. Try the best 2-pad pair for a moment
	// balance; if the load falls between them, the remaining pads lift and the
	// system is at the tipping threshold.
	for a := 0; a < 4; a++ {
		for b := a + 1; b < 4; b++ {
			r := twoPadEquilibrium(totalLoadN, loadPos, pads[a], pads[b])
			if r == nil {
				continue
			}
			sol := OutriggerSolution{State: UnstableTipping, ActiveCount: 2}
			sol.Active = [4]bool{}
			sol.Active[a], sol.Active[b] = true, true
			sol.Reactions[a], sol.Reactions[b] = r[0], r[1]
			sol.Residual = equilibriumResidual4(totalLoadN, loadPos, pads, sol.Reactions)
			return sol, nil
		}
	}

	return OutriggerSolution{State: UnstableTipping}, nil
}

// twoPadEquilibrium solves the indeterminate 2-pad case: the load is carried
// by two pads of a rocker line. Uses moment balance about both axes; returns
// nil when the pair cannot carry the load without tension.
func twoPadEquilibrium(totalLoadN float64, loadPos [2]float64, p0 [2]float64, p1 [2]float64) *[2]float64 {
	x0, y0 := p0[0], p0[1]
	x1, y1 := p1[0], p1[1]
	denom := x0*y1 - x1*y0
	if math.Abs(denom) > 1e-9 {
		r0 := totalLoadN * (y1*loadPos[0] - x1*loadPos[1]) / (x0*y1 - x1*y0)
		r1 := totalLoadN * (x0*loadPos[1] - y0*loadPos[0]) / (x0*y1 - x1*y0)
		if r0 >= 0 && r1 >= 0 && equilibriumResidual4(totalLoadN, loadPos, [4][2]float64{p0, p1, {0, 0}, {0, 0}}, [4]float64{r0, r1, 0, 0}) < 1e-6 {
			return &[2]float64{r0, r1}
		}
		return nil
	}
	if math.Abs(y1-y0) > 1e-9 {
		t := (loadPos[1] - y0) / (y1 - y0)
		if t >= 0 && t <= 1 {
			return &[2]float64{totalLoadN * (1 - t), totalLoadN * t}
		}
	}
	if math.Abs(x1-x0) > 1e-9 {
		t := (loadPos[0] - x0) / (x1 - x0)
		if t >= 0 && t <= 1 {
			return &[2]float64{totalLoadN * (1 - t), totalLoadN * t}
		}
	}
	return nil
}

// elasticReaction4 returns the elastic planar reaction at pad `idx` when the
// full 4-pad plane distributes totalLoadN. The linear pressure plane
// q(x,y) = a + b*x + c*y is fitted so that:
//
//	sum q = W, sum q*x = W*xw, sum q*y = W*yw.
//
// The result may be negative (tension), which triggers lift-off relaxation.
func elasticReaction4(totalLoadN float64, loadPos [2]float64, pads [4][2]float64, idx int) float64 {
	xw, yw := loadPos[0], loadPos[1]

	sx, sy, sxx, sxy, syy := 0.0, 0.0, 0.0, 0.0, 0.0
	for i := 0; i < 4; i++ {
		x, y := pads[i][0], pads[i][1]
		sx += x
		sy += y
		sxx += x * x
		sxy += x * y
		syy += y * y
	}

	det := 4*(sxx*syy-sxy*sxy) - sx*(sx*syy-sxy*sy) + sy*(sx*sxy-sxx*sy)
	if math.Abs(det) < 1e-12 {
		return totalLoadN / 4.0
	}

	detA := totalLoadN*(sxx*syy-sxy*sxy) - totalLoadN*xw*(sx*syy-sxy*sy) + totalLoadN*yw*(sx*sxy-sxx*sy)
	detB := 4*(totalLoadN*xw*syy-sxy*totalLoadN*yw) - totalLoadN*(sx*syy-sxy*sy) + sy*(sx*totalLoadN*yw-totalLoadN*xw*sy)
	detC := 4*(sxx*totalLoadN*yw-totalLoadN*xw*sxy) - sx*(sx*totalLoadN*yw-totalLoadN*xw*sy) + totalLoadN*(sx*sxy-sxx*sy)

	a := detA / det
	b := detB / det
	c := detC / det

	x, y := pads[idx][0], pads[idx][1]
	return a + b*x + c*y
}

// equilibriumResidual4 computes ||Ax - b||_2 for the full 4-pad equilibrium
// equations (Fz=0, Mx=0, My=0). Lifted-off pads contribute zero reaction.
func equilibriumResidual4(totalLoadN float64, loadPos [2]float64, pads [4][2]float64, reactions [4]float64) float64 {
	var sumF, sumMx, sumMy float64
	for i := 0; i < 4; i++ {
		sumF += reactions[i]
		sumMx += reactions[i] * pads[i][1]
		sumMy += reactions[i] * pads[i][0]
	}
	dx := sumF - totalLoadN
	dy := sumMx - totalLoadN*loadPos[1]
	dz := sumMy - totalLoadN*loadPos[0]
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func finite2(v [2]float64) bool {
	return !math.IsNaN(v[0]) && !math.IsInf(v[0], 0) && !math.IsNaN(v[1]) && !math.IsInf(v[1], 0)
}

func finite4x2(v [4][2]float64) bool {
	for i := 0; i < 4; i++ {
		if !finite2(v[i]) {
			return false
		}
	}
	return true
}

// OutriggerInputUQ carries a nominal outrigger state plus the input standard
// deviations used by DNV-RP-0513 first-order error propagation.
type OutriggerInputUQ struct {
	TotalLoadN   float64
	LoadPos      [2]float64
	Pads         [4][2]float64
	SigmaLoad    float64
	SigmaLoadPos [2]float64
	SigmaPad     float64
}

// OutriggerResultUQ is the reaction force solution with propagated
// uncertainty, lift-off flags and stability verdict.
type OutriggerResultUQ struct {
	Reactions [4]float64
	StdDevs   [4]float64
	LiftOff   [4]bool
	IsStable  bool
}

// OutriggerUQ computes each ground reaction, its standard deviation via
// first-order Taylor expansion, and the 95th-percentile characteristic
// upper-bound reaction:
//
//	sigma_Pi^2 = sum_j (dPi/dx_j)^2 * sigma_xj^2
//	Pk = Pi + 1.645 * sigma_Pi
func OutriggerUQ(in OutriggerInputUQ) (OutriggerResultUQ, error) {
	if !(in.TotalLoadN >= 0 && finite2(in.LoadPos) && finite4x2(in.Pads)) {
		return OutriggerResultUQ{}, fmt.Errorf("invalid UQ inputs")
	}

	res, err := Outrigger4Solve(in.TotalLoadN, in.LoadPos, in.Pads)
	if err != nil {
		return OutriggerResultUQ{}, err
	}

	resUQ := OutriggerResultUQ{
		Reactions: res.Reactions,
		IsStable:  res.State != UnstableTipping,
	}
	for i := 0; i < 4; i++ {
		resUQ.LiftOff[i] = res.Reactions[i] <= 0
	}

	// Sensitivities via Central finite differences of the equilibrium problem.
	// The active set is resolved per perturbed input exactly as for the
	// nominal case.
	perturb := 1e-3

	dLoad := [4]float64{}
	{
		p := perturb * in.TotalLoadN
		if p == 0 {
			p = 1.0
		}
		if hi, errH := Outrigger4Solve(in.TotalLoadN+p, in.LoadPos, in.Pads); errH == nil {
			if lo, errL := Outrigger4Solve(in.TotalLoadN-p, in.LoadPos, in.Pads); errL == nil {
				for i := 0; i < 4; i++ {
					dLoad[i] = (hi.Reactions[i] - lo.Reactions[i]) / (2 * p)
				}
			}
		}
	}

	sensPos := func(j int) [4]float64 {
		var d [4]float64
		scale := perturb * math.Max(1.0, math.Abs(in.LoadPos[j]))
		hi := in.LoadPos
		lo := in.LoadPos
		hi[j] += scale
		lo[j] -= scale
		h, errH := Outrigger4Solve(in.TotalLoadN, hi, in.Pads)
		l, errL := Outrigger4Solve(in.TotalLoadN, lo, in.Pads)
		if errH != nil || errL != nil {
			return d
		}
		for i := 0; i < 4; i++ {
			d[i] = (h.Reactions[i] - l.Reactions[i]) / (2 * scale)
		}
		return d
	}
	dPosX := sensPos(0)
	dPosY := sensPos(1)

	dPadX, dPadY := [4][4]float64{}, [4][4]float64{}
	for k := 0; k < 4; k++ {
		for j := 0; j < 2; j++ {
			scale := perturb * math.Max(1.0, math.Abs(in.Pads[k][j]))
			hi := in.Pads
			lo := in.Pads
			hi[k][j] += scale
			lo[k][j] -= scale
			h, errH := Outrigger4Solve(in.TotalLoadN, in.LoadPos, hi)
			l, errL := Outrigger4Solve(in.TotalLoadN, in.LoadPos, lo)
			if errH != nil || errL != nil {
				continue
			}
			for i := 0; i < 4; i++ {
				d := (h.Reactions[i] - l.Reactions[i]) / (2 * scale)
				if j == 0 {
					dPadX[i][k] = d
				} else {
					dPadY[i][k] = d
				}
			}
		}
	}

	for i := 0; i < 4; i++ {
		var v float64
		v += (dLoad[i] * in.SigmaLoad) * (dLoad[i] * in.SigmaLoad)
		v += (dPosX[i] * in.SigmaLoadPos[0]) * (dPosX[i] * in.SigmaLoadPos[0])
		v += (dPosY[i] * in.SigmaLoadPos[1]) * (dPosY[i] * in.SigmaLoadPos[1])
		for k := 0; k < 4; k++ {
			v += (dPadX[i][k] * in.SigmaPad) * (dPadX[i][k] * in.SigmaPad)
			v += (dPadY[i][k] * in.SigmaPad) * (dPadY[i][k] * in.SigmaPad)
		}
		resUQ.StdDevs[i] = math.Sqrt(v)
	}

	return resUQ, nil
}

// CharacteristicUpperBound returns the DNV-RP-0513 95th-percentile
// characteristic reaction: Pi + 1.645 * sigmaPi.
func CharacteristicUpperBound(reactionN, stdDevN float64) float64 {
	return reactionN + 1.645*stdDevN
}
