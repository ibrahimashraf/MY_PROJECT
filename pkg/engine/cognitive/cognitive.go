package cognitive

import "math"

// Belief represents a logical proposition with associated confidence.
type Belief struct {
	Proposition string
	Confidence  float64 // [0, 1]
}

// Desire represents a goal state with priority weight.
type Desire struct {
	Goal     string
	Priority float64
}

// Intention is the selected action plan derived from beliefs and desires.
type Intention struct {
	Action  string
	Target  [3]float64 // World position target
	Urgency float64
}

// BDIAgent implements a Belief-Desire-Intention cognitive agent.
type BDIAgent struct {
	ID        string
	Beliefs   []Belief
	Desires   []Desire
	Intention *Intention
	Position  [3]float64
}

// Deliberate selects the highest-priority feasible intention from the desire set
// given the current belief confidence state.
func (a *BDIAgent) Deliberate() {
	var best *Desire
	for i := range a.Desires {
		if best == nil || a.Desires[i].Priority > best.Priority {
			best = &a.Desires[i]
		}
	}
	if best == nil {
		a.Intention = nil
		return
	}
	a.Intention = &Intention{
		Action:  best.Goal,
		Target:  [3]float64{0, 0, 0},
		Urgency: best.Priority,
	}
}

// UpdateBelief adds or refreshes a belief with new confidence.
func (a *BDIAgent) UpdateBelief(proposition string, confidence float64) {
	for i, b := range a.Beliefs {
		if b.Proposition == proposition {
			a.Beliefs[i].Confidence = confidence
			return
		}
	}
	a.Beliefs = append(a.Beliefs, Belief{Proposition: proposition, Confidence: confidence})
}

// QLearningAgent implements tabular Q-learning for terrain navigation.
type QLearningAgent struct {
	QTable  map[[2]int][]float64 // State -> Action Q-values
	Alpha   float64              // Learning rate
	Gamma   float64              // Discount factor
	Epsilon float64              // Exploration rate
	Actions int
}

// NewQLearningAgent initializes Q-learning agent.
func NewQLearningAgent(actions int, alpha, gamma, epsilon float64) *QLearningAgent {
	return &QLearningAgent{
		QTable:  make(map[[2]int][]float64),
		Alpha:   alpha,
		Gamma:   gamma,
		Epsilon: epsilon,
		Actions: actions,
	}
}

func (q *QLearningAgent) getQ(state [2]int) []float64 {
	if _, ok := q.QTable[state]; !ok {
		q.QTable[state] = make([]float64, q.Actions)
	}
	return q.QTable[state]
}

// SelectAction returns action index using epsilon-greedy policy.
func (q *QLearningAgent) SelectAction(state [2]int, randFloat float64) int {
	if randFloat < q.Epsilon {
		return int(randFloat * float64(q.Actions)) % q.Actions
	}
	qs := q.getQ(state)
	best := 0
	for i, v := range qs {
		if v > qs[best] {
			best = i
		}
	}
	return best
}

// Update performs Q-value Bellman backup: Q(s,a) += alpha * (r + gamma * max_a' Q(s',a') - Q(s,a))
func (q *QLearningAgent) Update(state [2]int, action int, reward float64, nextState [2]int) {
	qs := q.getQ(state)
	nextQs := q.getQ(nextState)

	maxNext := nextQs[0]
	for _, v := range nextQs[1:] {
		if v > maxNext {
			maxNext = v
		}
	}

	qs[action] += q.Alpha * (reward + q.Gamma*maxNext - qs[action])
}

// RewardGradient computes reward for navigating toward a goal on 2D terrain.
func RewardGradient(pos, goal [2]int) float64 {
	dx := float64(goal[0] - pos[0])
	dy := float64(goal[1] - pos[1])
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 1.0 {
		return 100.0 // Goal reached reward
	}
	return -dist * 0.1 // Negative step cost proportional to remaining distance
}
