package economics

import "math"

// NashEquilibriumSolver finds pure-strategy Nash equilibria in a 2-player matrix game.
type BimatrixGame struct {
	RowPayoffs [][]float64 // Payoff matrix for Row player
	ColPayoffs [][]float64 // Payoff matrix for Col player
}

// FindPureNashEquilibria returns (row, col) index pairs that are Nash equilibria.
func (g BimatrixGame) FindPureNashEquilibria() [][2]int {
	m := len(g.RowPayoffs)
	if m == 0 {
		return nil
	}
	n := len(g.RowPayoffs[0])
	var results [][2]int

	for r := 0; r < m; r++ {
		for c := 0; c < n; c++ {
			// Check if row player cannot improve (best response given col=c)
			rowBR := true
			for r2 := 0; r2 < m; r2++ {
				if g.RowPayoffs[r2][c] > g.RowPayoffs[r][c] {
					rowBR = false
					break
				}
			}
			// Check if col player cannot improve (best response given row=r)
			colBR := true
			for c2 := 0; c2 < n; c2++ {
				if g.ColPayoffs[r][c2] > g.ColPayoffs[r][c] {
					colBR = false
					break
				}
			}
			if rowBR && colBR {
				results = append(results, [2]int{r, c})
			}
		}
	}
	return results
}

// DijkstraShortestPath computes minimum-cost geodesic path over weighted terrain graph.
func DijkstraShortestPath(graph [][]float64, src int) []float64 {
	n := len(graph)
	dist := make([]float64, n)
	visited := make([]bool, n)
	for i := range dist {
		dist[i] = math.Inf(1)
	}
	dist[src] = 0

	for step := 0; step < n; step++ {
		// Find unvisited node with minimum distance
		u := -1
		for v := 0; v < n; v++ {
			if !visited[v] && (u == -1 || dist[v] < dist[u]) {
				u = v
			}
		}
		if u == -1 || math.IsInf(dist[u], 1) {
			break
		}
		visited[u] = true

		for v := 0; v < n; v++ {
			if graph[u][v] > 0 {
				newDist := dist[u] + graph[u][v]
				if newDist < dist[v] {
					dist[v] = newDist
				}
			}
		}
	}
	return dist
}

// ExponentialDemandElasticity models price elasticity of demand: Q = Q0 * (P / P0)^(-eta)
func ExponentialDemandElasticity(baseQuantity, basePrice, targetPrice, elasticity float64) float64 {
	return baseQuantity * math.Pow(targetPrice/basePrice, -elasticity)
}
