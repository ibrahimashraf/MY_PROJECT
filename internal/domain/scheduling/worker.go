package scheduling

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/riverqueue/river"
	"integin/pkg/engine/cognitive"
)

type AgentRegistry interface {
	GetAvailableAgents(ctx context.Context) ([]*cognitive.BDIAgent, error)
	GetCompetency(ctx context.Context, agentID string) (TechnicianCompetency, error)
}

// InMemoryAgentRegistry is a simple registry for pilot testing.
type InMemoryAgentRegistry struct {
	Agents       []*cognitive.BDIAgent
	Competencies map[string]TechnicianCompetency
}

func (r *InMemoryAgentRegistry) GetAvailableAgents(ctx context.Context) ([]*cognitive.BDIAgent, error) {
	return r.Agents, nil
}

func (r *InMemoryAgentRegistry) GetCompetency(ctx context.Context, agentID string) (TechnicianCompetency, error) {
	if c, ok := r.Competencies[agentID]; ok {
		return c, nil
	}
	// Default to expired/missing if not found
	return TechnicianCompetency{}, errors.New("competency not found")
}

// CorrectiveWorkOrderWorker dequeues structural overload jobs and routes the
// nearest qualified technician via WGS84 local ENU frames.
type CorrectiveWorkOrderWorker struct {
	river.WorkerDefaults[CorrectiveWorkOrderArgs]
	Registry AgentRegistry
}

func (w *CorrectiveWorkOrderWorker) Work(ctx context.Context, job *river.Job[CorrectiveWorkOrderArgs]) error {
	agents, err := w.Registry.GetAvailableAgents(ctx)
	if err != nil {
		return err
	}

	var nearest *cognitive.BDIAgent
	var minDst float64 = math.MaxFloat64
	var bestComp TechnicianCompetency

	for _, a := range agents {
		comp, err := w.Registry.GetCompetency(ctx, a.ID)
		if err != nil || !comp.IsCurrent(time.Now()) {
			continue
		}

		// Calculate Euclidean distance in the local ENU frame
		dx := a.Position[0] - job.Args.Location[0]
		dy := a.Position[1] - job.Args.Location[1]
		dz := a.Position[2] - job.Args.Location[2]
		dst := math.Sqrt(dx*dx + dy*dy + dz*dz)

		if dst < minDst {
			minDst = dst
			nearest = a
			bestComp = comp
		}
	}

	if nearest == nil {
		return errors.New("no qualified technician available for routing")
	}

	order := WorkOrderCandidate{
		ID:         "remedy-" + job.Args.ComponentID,
		Skill:      "structural_remediation",
		Priority:   0.95, // Statutorily overrides routine work
		Target:     job.Args.Location,
		Competency: bestComp,
	}

	_, err = DispatchWorkOrder(nearest, order)
	return err
}
