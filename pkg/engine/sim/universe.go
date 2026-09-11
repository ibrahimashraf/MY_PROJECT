package sim

import (
	"context"
	"math"
	"sync"
	"time"
)

// SimVec3 represents 3D physics coordinate.
type SimVec3 struct {
	X, Y, Z float64
}

func (v SimVec3) Add(o SimVec3) SimVec3 { return SimVec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }
func (v SimVec3) Sub(o SimVec3) SimVec3 { return SimVec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }
func (v SimVec3) Scale(s float64) SimVec3 { return SimVec3{v.X * s, v.Y * s, v.Z * s} }
func (v SimVec3) Length() float64 { return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z) }

// AgentRole defines autonomous agent task identity in the procedural universe.
type AgentRole string

const (
	RoleSurveyor  AgentRole = "SURVEYOR"
	RoleExcavator AgentRole = "EXCAVATOR"
	RoleBuilder   AgentRole = "BUILDER"
	RoleInspector AgentRole = "INSPECTOR"
)

// UniverseAgent represents an autonomous actor operating on the planetary terrain.
type UniverseAgent struct {
	ID        string
	Role      AgentRole
	Position  SimVec3
	Velocity  SimVec3
	TargetPos SimVec3
	Energy    float64
	Status    string
}

// Update evaluates agent movement and goal progress.
func (a *UniverseAgent) Update(dt float64) {
	delta := a.TargetPos.Sub(a.Position)
	dist := delta.Length()

	if dist < 0.1 {
		a.Status = "GOAL_REACHED"
		a.Velocity = SimVec3{}
		return
	}

	dir := delta.Scale(1.0 / dist)
	speed := 5.0 // 5 m/s
	a.Velocity = dir.Scale(speed)
	a.Position = a.Position.Add(a.Velocity.Scale(dt))
	a.Energy -= 0.01 * dt
	a.Status = "MOVING"
}

// UniverseWorld orchestrates multi-agent planetary simulation with 60Hz tick scheduler.
type UniverseWorld struct {
	mu     sync.RWMutex
	Agents map[string]*UniverseAgent
	Tick   uint64
	Time   float64
}

// NewUniverseWorld initializes simulation world.
func NewUniverseWorld() *UniverseWorld {
	return &UniverseWorld{
		Agents: make(map[string]*UniverseAgent),
	}
}

// SpawnAgent adds an autonomous agent to the universe.
func (w *UniverseWorld) SpawnAgent(id string, role AgentRole, pos, target SimVec3) *UniverseAgent {
	w.mu.Lock()
	defer w.mu.Unlock()

	agent := &UniverseAgent{
		ID:        id,
		Role:      role,
		Position:  pos,
		TargetPos: target,
		Energy:    100.0,
		Status:    "IDLE",
	}
	w.Agents[id] = agent
	return agent
}

// Step advances world physics and agent logic by dt seconds.
func (w *UniverseWorld) Step(dt float64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Tick++
	w.Time += dt

	for _, a := range w.Agents {
		a.Update(dt)
	}
}

// ContinuousCollisionCheck calculates whether a swept moving sphere collides with terrain obstacle.
func ContinuousCollisionCheck(startPos, endPos SimVec3, radius float64, obstacleCenter SimVec3, obstacleRadius float64) (bool, float64) {
	d := endPos.Sub(startPos)
	f := startPos.Sub(obstacleCenter)

	rSum := radius + obstacleRadius

	a := d.X*d.X + d.Y*d.Y + d.Z*d.Z
	b := 2.0 * (f.X*d.X + f.Y*d.Y + f.Z*d.Z)
	c := (f.X*f.X + f.Y*f.Y + f.Z*f.Z) - rSum*rSum

	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return false, 0
	}

	discriminant = math.Sqrt(discriminant)
	t1 := (-b - discriminant) / (2 * a)
	if t1 >= 0 && t1 <= 1.0 {
		return true, t1
	}

	t2 := (-b + discriminant) / (2 * a)
	if t2 >= 0 && t2 <= 1.0 {
		return true, t2
	}

	return false, 0
}

// RunUniverseLoop runs background physics loop with context cancellation.
func (w *UniverseWorld) RunUniverseLoop(ctx context.Context, hz int) error {
	if hz <= 0 {
		hz = 60
	}
	interval := time.Second / time.Duration(hz)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	dt := 1.0 / float64(hz)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			w.Step(dt)
		}
	}
}

// GetAgentSnapshot returns thread-safe copy of agent states.
func (w *UniverseWorld) GetAgentSnapshot() []UniverseAgent {
	w.mu.RLock()
	defer w.mu.RUnlock()

	snapshot := make([]UniverseAgent, 0, len(w.Agents))
	for _, a := range w.Agents {
		snapshot = append(snapshot, *a)
	}
	return snapshot
}
