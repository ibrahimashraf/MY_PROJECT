package bio

import "math"

// LotkaVolterraState represents predator-prey population pair.
type LotkaVolterraState struct {
	Prey     float64
	Predator float64
}

// LotkaVolterraParams defines ecosystem interaction coefficients.
type LotkaVolterraParams struct {
	Alpha float64 // Prey birth rate
	Beta  float64 // Predation rate
	Delta float64 // Predator reproduction per prey eaten
	Gamma float64 // Predator death rate
}

// Step advances Lotka-Volterra system by dt using RK4.
func (s LotkaVolterraState) Step(p LotkaVolterraParams, dt float64) LotkaVolterraState {
	dxdt := func(x, y float64) float64 { return p.Alpha*x - p.Beta*x*y }
	dydt := func(x, y float64) float64 { return p.Delta*x*y - p.Gamma*y }

	k1x := dxdt(s.Prey, s.Predator)
	k1y := dydt(s.Prey, s.Predator)
	k2x := dxdt(s.Prey+0.5*dt*k1x, s.Predator+0.5*dt*k1y)
	k2y := dydt(s.Prey+0.5*dt*k1x, s.Predator+0.5*dt*k1y)
	k3x := dxdt(s.Prey+0.5*dt*k2x, s.Predator+0.5*dt*k2y)
	k3y := dydt(s.Prey+0.5*dt*k2x, s.Predator+0.5*dt*k2y)
	k4x := dxdt(s.Prey+dt*k3x, s.Predator+dt*k3y)
	k4y := dydt(s.Prey+dt*k3x, s.Predator+dt*k3y)

	return LotkaVolterraState{
		Prey:     math.Max(0, s.Prey+(dt/6.0)*(k1x+2*k2x+2*k3x+k4x)),
		Predator: math.Max(0, s.Predator+(dt/6.0)*(k1y+2*k2y+2*k3y+k4y)),
	}
}

// WildfireCell represents a 2D terrain cell fire state.
type WildfireCell int

const (
	CellGreen WildfireCell = iota
	CellBurning
	CellBurned
)

// WildfireGrid simulates fire spread via cellular automata + wind coupling.
type WildfireGrid struct {
	Width, Height int
	Cells         []WildfireCell
	WindX, WindY  float64 // Wind direction bias
}

// NewWildfireGrid initializes grid and ignites one seed cell.
func NewWildfireGrid(w, h int, windX, windY float64, seedX, seedY int) *WildfireGrid {
	g := &WildfireGrid{
		Width: w, Height: h,
		Cells: make([]WildfireCell, w*h),
		WindX: windX, WindY: windY,
	}
	g.Cells[seedY*w+seedX] = CellBurning
	return g
}

func (g *WildfireGrid) get(x, y int) WildfireCell {
	if x < 0 || x >= g.Width || y < 0 || y >= g.Height {
		return CellBurned
	}
	return g.Cells[y*g.Width+x]
}

// Step advances fire simulation by one generation.
func (g *WildfireGrid) Step(baseSpreadProb float64) {
	next := make([]WildfireCell, len(g.Cells))
	copy(next, g.Cells)

	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	windBias := [][2]float64{{g.WindX, 0}, {-g.WindX, 0}, {0, g.WindY}, {0, -g.WindY}}

	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			if g.get(x, y) == CellBurning {
				next[y*g.Width+x] = CellBurned
				for i, d := range dirs {
					nx, ny := x+d[0], y+d[1]
					if g.get(nx, ny) == CellGreen {
						prob := math.Min(1.0, baseSpreadProb+windBias[i][0]+windBias[i][1])
						if prob > 0.5 { // deterministic threshold for pure Go reproducibility
							next[ny*g.Width+nx] = CellBurning
						}
					}
				}
			}
		}
	}
	g.Cells = next
}

// BurnedCount returns number of burned or burning cells.
func (g *WildfireGrid) BurnedCount() int {
	count := 0
	for _, c := range g.Cells {
		if c != CellGreen {
			count++
		}
	}
	return count
}
