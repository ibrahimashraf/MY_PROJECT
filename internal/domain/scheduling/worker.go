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
	GetCompetencies(ctx context.Context, agentID string) ([]TechnicianCompetency, error)
}

// InMemoryAgentRegistry is a simple registry for pilot testing.
type InMemoryAgentRegistry struct {
	Agents       []*cognitive.BDIAgent
	Competencies map[string][]TechnicianCompetency
}

func (r *InMemoryAgentRegistry) GetAvailableAgents(ctx context.Context) ([]*cognitive.BDIAgent, error) {
	return r.Agents, nil
}

func (r *InMemoryAgentRegistry) GetCompetencies(ctx context.Context, agentID string) ([]TechnicianCompetency, error) {
	return r.Competencies[agentID], nil
}

// DefaultCorrectiveSkill is the fallback skill when the job carries no hazard kind.
const DefaultCorrectiveSkill = "structural_remediation"

func resolveSkill(args CorrectiveWorkOrderArgs) string {
	switch args.HazardKind {
	case "", "MOMENT_UTILIZATION_EXCEEDED":
		return DefaultCorrectiveSkill
	default:
		return args.HazardKind
	}
}
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
	reqSkill := resolveSkill(job.Args)

	for _, a := range agents {
		comps, err := w.Registry.GetCompetencies(ctx, a.ID)
		if err != nil {
			continue
		}

		var validComp *TechnicianCompetency
		for _, c := range comps {
			if c.EquipmentTypeID == reqSkill && c.IsCurrent(time.Now()) {
				validComp = &c
				break
			}
		}
		if validComp == nil {
			continue // Lacks current competency for this specific skill
		}

		// Calculate Euclidean distance in the local ENU frame
		dx := a.Position[0] - job.Args.Location[0]
		dy := a.Position[1] - job.Args.Location[1]
		dz := a.Position[2] - job.Args.Location[2]
		dst := math.Sqrt(dx*dx + dy*dy + dz*dz)

		if dst < minDst {
			minDst = dst
			nearest = a
			bestComp = *validComp
		}
	}

	if nearest == nil {
		return errors.New("no qualified technician available for routing")
	}

	order := WorkOrderCandidate{
		ID:         "remedy-" + job.Args.ComponentID,
		Skill:      reqSkill,
		Priority:   0.95, // Statutorily overrides routine work
		Target:     job.Args.Location,
		Competency: bestComp,
	}

	_, err = DispatchWorkOrder(nearest, order)
	return err
}
